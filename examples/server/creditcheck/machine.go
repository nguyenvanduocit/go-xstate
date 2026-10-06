package creditcheck

import (
	"context"
	"fmt"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// saveReportParams are the params of the saveReport action.
type saveReportParams struct{ BureauName string }

// Machine mirrors creditCheckMachine (machine.ts); svc supplies the actors' work and the
// saving and logging of its actions and guards.
func Machine(svc Services) *xs.StateMachine[CreditProfile] {
	return xs.NewSetup[CreditProfile](implementations(svc)).CreateMachine(xs.MachineConfig[CreditProfile]{
		ID: "multipleCreditCheck",
		Context: CreditProfile{
			InterestRateOptions: []float64{},
		},
		Initial: "creditCheck",
		States: xs.States{{
			Key:     "creditCheck",
			Initial: "Entering Information",
			States: xs.States{
				{
					Key: "Entering Information",
					On: map[string]xs.Transitions{
						"Submit": {{Target: "Verifying Credentials", Reenter: true}},
					},
				},
				verifyingCredentials(),
				checkingCreditScores(),
				determiningInterestRateOptions(),
			},
		}},
	})
}

func implementations(svc Services) xs.Implementations {
	return xs.Implementations{
		Actors: map[string]xs.ActorLogic{
			"checkBureau": xs.FromPromise(func(ctx context.Context, a xs.PromiseArgs) (int, error) {
				return svc.CheckBureau(ctx, a.Input.(BureauQuery))
			}),
			"checkReportsTable": xs.FromPromise(func(ctx context.Context, a xs.PromiseArgs) (*CreditReport, error) {
				return svc.CheckReportsTable(ctx, a.Input.(BureauQuery))
			}),
			"verifyCredentials": xs.FromPromise(func(ctx context.Context, a xs.PromiseArgs) (UserCredential, error) {
				return svc.VerifyCredentials(ctx, a.Input.(map[string]any))
			}),
			"determineMiddleScore": xs.FromPromise(func(ctx context.Context, a xs.PromiseArgs) (int, error) {
				return svc.DetermineMiddleScore(ctx, a.Input.([]int))
			}),
			"generateInterestRates": xs.FromPromise(func(ctx context.Context, a xs.PromiseArgs) (float64, error) {
				return svc.GenerateInterestRate(ctx, a.Input.(int))
			}),
		},
		Actions: map[string]xs.Action{
			"saveReport": xs.ActionFunc(func(a xs.ActionArgs[CreditProfile]) {
				svc.log("saving report to the database...")
				// machine.ts stores context.EquiGavinScore for every bureau. JS does not await
				// the save; a failure is only logged by the service.
				_ = svc.SaveCreditReport(CreditReport{
					SSN:         a.Context.SSN,
					BureauName:  a.Params.(saveReportParams).BureauName,
					CreditScore: a.Context.EquiGavinScore,
				})
			}),
			"emailUser": xs.ActionFunc(func(a xs.ActionArgs[CreditProfile]) {
				svc.log("emailing user with their interest rate options: ", a.Context.InterestRateOptions)
			}),
			"saveCreditProfile": xs.ActionFunc(func(a xs.ActionArgs[CreditProfile]) {
				svc.log("saving results to the database...")
				_ = svc.SaveCreditProfile(a.Context)
			}),
			"emailSalesTeam": xs.ActionFunc(func(a xs.ActionArgs[CreditProfile]) {
				c := a.Context
				svc.log("emailing sales team with the user\"s information: ", c.FirstName, c.LastName, c.InterestRateOptions, c.MiddleScore)
			}),
		},
		Guards: map[string]xs.Guard{
			"allSucceeded": xs.GuardFunc(func(a xs.GuardArgs[CreditProfile]) bool {
				svc.log("allSucceeded guard called")
				c := a.Context
				return c.EquiGavinScore > 0 && c.GavUnionScore > 0 && c.GavperianScore > 0
			}),
			"gavUnionReportFound": xs.GuardFunc(func(a xs.GuardArgs[CreditProfile]) bool {
				return a.Context.GavUnionScore > 0
			}),
			"equiGavinReportFound": xs.GuardFunc(func(a xs.GuardArgs[CreditProfile]) bool {
				return a.Context.EquiGavinScore > 0
			}),
			"gavperianReportFound": xs.GuardFunc(func(a xs.GuardArgs[CreditProfile]) bool {
				return a.Context.GavperianScore > 0
			}),
		},
	}
}

// jsString is how JS string concatenation renders a caught value.
func jsString(v any) string {
	if err, ok := v.(error); ok {
		return "Error: " + err.Error()
	}
	return fmt.Sprint(v)
}

func doneOutput(e xs.Event) any { return e.(xs.DoneActorEvent).Output }

func errorMessage(msg string) xs.Action {
	return xs.Assign(func(a xs.AssignArgs[CreditProfile]) CreditProfile {
		c := a.Context
		c.ErrorMessage = msg
		return c
	})
}

func verifyingCredentials() xs.StateConfig {
	return xs.StateConfig{
		Key: "Verifying Credentials",
		Invoke: []xs.InvokeConfig{{
			Src:   "verifyCredentials",
			Input: xs.NewExpr(func(a xs.ExprArgs[CreditProfile]) any { return map[string]any(a.Event.(xs.E)) }),
			OnDone: xs.Transitions{{
				Target: "CheckingCreditScores",
				Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[CreditProfile]) CreditProfile {
					out := doneOutput(a.Event).(UserCredential)
					c := a.Context
					c.SSN, c.FirstName, c.LastName = out.SSN, out.FirstName, out.LastName
					return c
				})},
			}},
			OnError: xs.Transitions{{
				Target: "Entering Information",
				Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[CreditProfile]) CreditProfile {
					c := a.Context
					c.ErrorMessage = "Failed to verify credentials. Details: " + jsString(a.Event.(xs.ErrorActorEvent).Error)
					return c
				})},
			}},
		}},
	}
}

func checkingCreditScores() xs.StateConfig {
	return xs.StateConfig{
		Key:         "CheckingCreditScores",
		Type:        xs.Parallel,
		Description: "Kick off a series of requests to the 3 American Credit Bureaus and await their results",
		States: xs.States{
			bureauRegion(bureauSpec{
				region: "CheckingEquiGavin", bureau: EquiGavin, guard: "equiGavinReportFound",
				dbID: "equiGavinDBActor", fetchID: "equiGavinFetchActor",
				set: func(c CreditProfile, v int) CreditProfile { c.EquiGavinScore = v; return c },
			}),
			bureauRegion(bureauSpec{
				region: "CheckingGavUnion", bureau: GavUnion, guard: "gavUnionReportFound",
				dbID: "gavUnionDBActor", fetchID: "gavUnionFetchActor",
				set: func(c CreditProfile, v int) CreditProfile { c.GavUnionScore = v; return c },
			}),
			bureauRegion(bureauSpec{
				region: "CheckingGavperian", bureau: Gavperian, guard: "gavperianReportFound",
				dbID: "gavperianCheckActor", fetchID: "checkGavPerianActor",
				set: func(c CreditProfile, v int) CreditProfile { c.GavperianScore = v; return c },
			}),
		},
		OnDone: xs.Transitions{
			{
				Target:  "DeterminingInterestRateOptions",
				Guard:   xs.GuardRef{Type: "allSucceeded"},
				Reenter: true,
			},
			{
				Target:  "Entering Information",
				Actions: xs.Actions{errorMessage("Failed to retrieve credit scores.")},
			},
		},
	}
}

// bureauSpec is what differs between the three bureau regions of CheckingCreditScores.
type bureauSpec struct {
	region, bureau, guard, dbID, fetchID string
	set                                  func(CreditProfile, int) CreditProfile
}

func bureauRegion(b bureauSpec) xs.StateConfig {
	input := xs.NewExpr(func(a xs.ExprArgs[CreditProfile]) any {
		return BureauQuery{BureauName: b.bureau, SSN: a.Context.SSN}
	})
	assignScore := func(score func(xs.Event) int) xs.Action {
		return xs.Assign(func(a xs.AssignArgs[CreditProfile]) CreditProfile {
			return b.set(a.Context, score(a.Event))
		})
	}
	return xs.StateConfig{
		Key:     b.region,
		Initial: "CheckingForExistingReport",
		States: xs.States{
			{
				Key: "CheckingForExistingReport",
				Invoke: []xs.InvokeConfig{{
					Src:   "checkReportsTable",
					ID:    b.dbID,
					Input: input,
					OnDone: xs.Transitions{
						{
							Target: "FetchingComplete",
							Guard:  xs.GuardRef{Type: b.guard},
							// event.output?.creditScore ?? 0
							Actions: xs.Actions{assignScore(func(e xs.Event) int {
								if report, _ := doneOutput(e).(*CreditReport); report != nil {
									return report.CreditScore
								}
								return 0
							})},
						},
						{Target: "FetchingReport"},
					},
					OnError: xs.Transitions{{Target: "FetchingFailed"}},
				}},
			},
			{
				Key:  "FetchingComplete",
				Type: xs.Final,
				Entry: xs.Actions{
					xs.ActionRef{Type: "saveReport", Params: saveReportParams{BureauName: b.bureau}},
				},
			},
			{
				Key: "FetchingReport",
				Invoke: []xs.InvokeConfig{{
					Src:   "checkBureau",
					ID:    b.fetchID,
					Input: input,
					OnDone: xs.Transitions{{
						Target:  "FetchingComplete",
						Actions: xs.Actions{assignScore(func(e xs.Event) int { return doneOutput(e).(int) })},
					}},
					OnError: xs.Transitions{{Target: "FetchingFailed"}},
				}},
			},
			{Key: "FetchingFailed", Type: xs.Final},
		},
	}
}

func determiningInterestRateOptions() xs.StateConfig {
	return xs.StateConfig{
		Key:         "DeterminingInterestRateOptions",
		Initial:     "DeterminingMiddleScore",
		Description: "After retrieving results, determine the middle score to be used in home loan interest rate decision",
		States: xs.States{
			{
				Key: "DeterminingMiddleScore",
				Invoke: []xs.InvokeConfig{{
					Src: "determineMiddleScore",
					ID:  "scoreDeterminationActor",
					Input: xs.NewExpr(func(a xs.ExprArgs[CreditProfile]) any {
						c := a.Context
						return []int{c.EquiGavinScore, c.GavUnionScore, c.GavperianScore}
					}),
					OnDone: xs.Transitions{{
						Target: "FetchingRates",
						Actions: xs.Actions{
							xs.Assign(func(a xs.AssignArgs[CreditProfile]) CreditProfile {
								c := a.Context
								c.MiddleScore = doneOutput(a.Event).(int)
								return c
							}),
							xs.ActionRef{Type: "saveCreditProfile"},
						},
					}},
				}},
			},
			{
				Key: "FetchingRates",
				Invoke: []xs.InvokeConfig{{
					Src:   "generateInterestRates",
					Input: xs.NewExpr(func(a xs.ExprArgs[CreditProfile]) any { return a.Context.MiddleScore }),
					OnDone: xs.Transitions{{
						Target: "RatesProvided",
						Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[CreditProfile]) CreditProfile {
							c := a.Context
							c.InterestRateOptions = []float64{doneOutput(a.Event).(float64)}
							return c
						})},
					}},
				}},
			},
			{
				Key:  "RatesProvided",
				Type: xs.Final,
				Entry: xs.Actions{
					xs.ActionRef{Type: "emailUser"},
					xs.ActionRef{Type: "emailSalesTeam"},
				},
			},
		},
	}
}
