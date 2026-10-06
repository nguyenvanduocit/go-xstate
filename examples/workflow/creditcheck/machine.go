// Package creditcheck ports references/xstate/examples/workflow-credit-check/main.ts
// (serverless workflow "perform customer credit check"): a workflow that calls
// a credit check, branches on the decision, then starts an application or
// sends a rejection email, with a 15 minute timeout on the credit check.
package creditcheck

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// Customer mirrors the `Customer` interface of main.ts.
type Customer struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	SSN          int    `json:"SSN"`
	YearlyIncome int    `json:"yearlyIncome"`
	Address      string `json:"address"`
	Employer     string `json:"employer"`
}

// Input is the machine input: `{ customer }`. It is also the input of
// callCreditCheckMicroservice and startApplicationWorkflowId.
type Input struct {
	Customer Customer `json:"customer"`
}

// RejectionInput is the input of sendRejectionEmailFunction: `{ applicant }`.
type RejectionInput struct {
	Applicant Customer `json:"applicant"`
}

// CreditCheck is the output of callCreditCheckMicroservice, assigned to
// `context.creditCheck` as is (main.ts types it as `{ decision }`, but the
// whole resolved object is stored).
type CreditCheck struct {
	ID       string `json:"id"`
	Score    int    `json:"score"`
	Decision string `json:"decision"`
	Reason   string `json:"reason"`
}

// Application is the `application` of the startApplicationWorkflowId output.
type Application struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

// ApplicationOutput is the output of startApplicationWorkflowId.
type ApplicationOutput struct {
	Application Application `json:"application"`
}

// Email is the `email` of the sendRejectionEmailFunction output.
type Email struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

// EmailOutput is the output of sendRejectionEmailFunction.
type EmailOutput struct {
	Email Email `json:"email"`
}

// Context is the machine's extended state. CreditCheck is `null` (nil) until
// the credit check resolves.
type Context struct {
	Customer    Customer     `json:"customer"`
	CreditCheck *CreditCheck `json:"creditCheck"`
}

// PT15M is the `PT15M` delay of the machine.
const PT15M = 15 * 60 * time.Second

// Actors are the three promise actors of main.ts.
type Actors struct {
	CallCreditCheckMicroservice xs.ActorLogic
	StartApplicationWorkflowID  xs.ActorLogic
	SendRejectionEmailFunction  xs.ActorLogic
}

// sleep waits d or until ctx is cancelled. Cancelling ctx stands in for
// abandoning the promise when the invoking state exits.
func sleep(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return context.Cause(ctx)
	case <-timer.C:
		return nil
	}
}

// DefaultActors mirrors the actors of main.ts, logging to w. delay is the
// "fake 1s" of startApplicationWorkflowId and sendRejectionEmailFunction.
func DefaultActors(w io.Writer, delay time.Duration) Actors {
	return Actors{
		CallCreditCheckMicroservice: xs.FromPromise(func(_ context.Context, a xs.PromiseArgs) (CreditCheck, error) {
			logInspect(w, "calling credit check microservice", a.Input)
			return CreditCheck{ID: "customer123", Score: 700, Decision: "Approved", Reason: "Good credit score"}, nil
		}),
		StartApplicationWorkflowID: xs.FromPromise(func(ctx context.Context, a xs.PromiseArgs) (ApplicationOutput, error) {
			logInspect(w, "starting application workflow", a.Input)
			if err := sleep(ctx, delay); err != nil {
				return ApplicationOutput{}, err
			}
			return ApplicationOutput{Application: Application{ID: "application123", Status: "Approved"}}, nil
		}),
		SendRejectionEmailFunction: xs.FromPromise(func(ctx context.Context, a xs.PromiseArgs) (EmailOutput, error) {
			logInspect(w, "sending rejection email", a.Input)
			if err := sleep(ctx, delay); err != nil {
				return EmailOutput{}, err
			}
			return EmailOutput{Email: Email{ID: "email123", Status: "Sent"}}, nil
		}),
	}
}

func decisionIs(want string) xs.Guard {
	return xs.GuardFunc(func(a xs.GuardArgs[Context]) bool {
		return a.Context.CreditCheck != nil && a.Context.CreditCheck.Decision == want
	})
}

// NewMachine mirrors `workflow` with the given actors.
func NewMachine(actors Actors) *xs.StateMachine[Context] {
	return xs.NewSetup[Context](xs.Implementations{
		Actors: map[string]xs.ActorLogic{
			"callCreditCheckMicroservice": actors.CallCreditCheckMicroservice,
			"startApplicationWorkflowId":  actors.StartApplicationWorkflowID,
			"sendRejectionEmailFunction":  actors.SendRejectionEmailFunction,
		},
		Delays: map[string]any{"PT15M": PT15M},
	}).CreateMachine(xs.MachineConfig[Context]{
		ID:      "customercreditcheck",
		Initial: "CheckCredit",
		ContextFn: func(a xs.ContextArgs) Context {
			return Context{Customer: a.Input.(Input).Customer}
		},
		States: xs.States{
			{
				Key: "CheckCredit",
				Invoke: []xs.InvokeConfig{{
					Src: "callCreditCheckMicroservice",
					Input: xs.NewExpr(func(a xs.ExprArgs[Context]) any {
						return Input{Customer: a.Context.Customer}
					}),
					OnDone: xs.Transitions{{
						Target: "EvaluateDecision",
						Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[Context]) Context {
							c := a.Context
							out := a.Event.(xs.DoneActorEvent).Output.(CreditCheck)
							c.CreditCheck = &out
							return c
						})},
					}},
				}},
				After: map[string]xs.Transitions{
					"PT15M": {{Target: "Timeout"}},
				},
			},
			{
				Key: "EvaluateDecision",
				Always: xs.Transitions{
					{Guard: decisionIs("Approved"), Target: "StartApplication"},
					{Guard: decisionIs("Denied"), Target: "RejectApplication"},
					{Target: "RejectApplication"},
				},
			},
			{
				Key: "StartApplication",
				Invoke: []xs.InvokeConfig{{
					Src: "startApplicationWorkflowId",
					Input: xs.NewExpr(func(a xs.ExprArgs[Context]) any {
						return Input{Customer: a.Context.Customer}
					}),
					OnDone: xs.Transitions{{Target: "End"}},
				}},
			},
			{
				Key: "RejectApplication",
				Invoke: []xs.InvokeConfig{{
					Src: "sendRejectionEmailFunction",
					Input: xs.NewExpr(func(a xs.ExprArgs[Context]) any {
						return RejectionInput{Applicant: a.Context.Customer}
					}),
					OnDone: xs.Transitions{{Target: "End"}},
				}},
			},
			{Key: "End", Type: xs.Final},
			{Key: "Timeout"},
		},
	})
}

// DemoInput is the input of the demo actor at the bottom of main.ts.
var DemoInput = Input{Customer: Customer{
	ID:           "customer123",
	Name:         "John Doe",
	SSN:          123456,
	YearlyIncome: 50000,
	Address:      "123 MyLane, MyCity, MyCountry",
	Employer:     "MyCompany",
}}

// Run mirrors the entry of main.ts with the real 1000 ms delays, printing to w.
func Run(w io.Writer) {
	RunWith(w, time.Second)
}

// RunWith is Run with the delay of the two "fake 1s" actors injected.
func RunWith(w io.Writer, delay time.Duration) {
	locked := &lockedWriter{w: w}
	done := make(chan struct{})
	actor := xs.CreateActor(
		NewMachine(DefaultActors(locked, delay)),
		xs.WithInput(DemoInput),
		xs.WithInspect(func(ev xs.InspectionEvent) {
			if ev.Type == xs.InspectEvent {
				logInspect(locked, "Received event", eventView(ev.Event))
			}
		}),
	)
	actor.Subscribe(xs.Observer[*xs.MachineSnapshot[Context]]{
		Complete: func() {
			// The machine has no output: `undefined` in the JS console.
			fmt.Fprintln(locked, "workflow completed undefined")
			close(done)
		},
	})
	actor.Start()
	<-done
	actor.Stop()
}

type lockedWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func (l *lockedWriter) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.w.Write(p)
}
