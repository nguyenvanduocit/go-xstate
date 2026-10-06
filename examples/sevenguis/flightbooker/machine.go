// Package flightbooker ports references/xstate/examples/7guis-flight-booker-react/src/machines/flightMachine.ts
// and the constants/helpers of src/utils/index.ts. The directory name starts with a digit, which is not a
// valid Go identifier, so the package name spells the number out.
package flightbooker

import (
	"context"
	"time"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

const dateLayout = "2006-01-02"

// Today mirrors utils' TODAY: the current UTC date as YYYY-MM-DD.
func Today() string { return time.Now().UTC().Format(dateLayout) }

// Tomorrow mirrors utils' TOMORROW: the UTC date 24 hours from now as YYYY-MM-DD.
func Tomorrow() string { return time.Now().UTC().Add(24 * time.Hour).Format(dateLayout) }

// Context mirrors the FlightData context: ISO dates (YYYY-MM-DD) compared as strings.
type Context struct {
	DepartDate string `json:"departDate"`
	ReturnDate string `json:"returnDate"`
}

// Booker mirrors the Booker actor: fromPromise(() => sleep(2000)).
func Booker() xs.ActorLogic {
	return xs.FromPromise(func(ctx context.Context, _ xs.PromiseArgs) (any, error) {
		select {
		case <-time.After(2 * time.Second):
			return nil, nil
		case <-ctx.Done():
			return nil, context.Cause(ctx)
		}
	})
}

func setDate(set func(*Context, string)) xs.Action {
	return xs.Assign(func(a xs.AssignArgs[Context]) Context {
		c := a.Context
		set(&c, a.Event.(xs.E)["value"].(string))
		return c
	})
}

// Machine mirrors flightBookerMachine. today and tomorrow are the initial depart/return dates and the
// reference date of the guards (JS: the module-level TODAY and TOMORROW).
func Machine(today, tomorrow string) *xs.StateMachine[Context] {
	return xs.NewSetup[Context](xs.Implementations{
		Actions: map[string]xs.Action{
			"setDepartDate": setDate(func(c *Context, v string) { c.DepartDate = v }),
			"setReturnDate": setDate(func(c *Context, v string) { c.ReturnDate = v }),
		},
		Actors: map[string]xs.ActorLogic{
			"Booker": Booker(),
		},
		Guards: map[string]xs.Guard{
			"isValidDepartDate?": xs.GuardFunc(func(a xs.GuardArgs[Context]) bool {
				return a.Context.DepartDate >= today
			}),
			"isValidReturnDate?": xs.GuardFunc(func(a xs.GuardArgs[Context]) bool {
				return a.Context.DepartDate >= today && a.Context.ReturnDate > a.Context.DepartDate
			}),
		},
	}).CreateMachine(xs.MachineConfig[Context]{
		ID:      "flightBookerMachine",
		Context: Context{DepartDate: today, ReturnDate: tomorrow},
		Initial: "scheduling",
		States: xs.States{
			{
				Key:     "scheduling",
				Initial: "oneWay",
				On: map[string]xs.Transitions{
					"CHANGE_DEPART_DATE": {{Actions: xs.Actions{xs.ActionRef{Type: "setDepartDate"}}}},
				},
				States: xs.States{
					{
						Key: "oneWay",
						On: map[string]xs.Transitions{
							"CHANGE_TRIP_TYPE": {{Target: "roundTrip"}},
							"BOOK_DEPART": {{
								Target: "#flightBookerMachine.booking",
								Guard:  xs.GuardRef{Type: "isValidDepartDate?"},
							}},
						},
					},
					{
						Key: "roundTrip",
						On: map[string]xs.Transitions{
							"CHANGE_TRIP_TYPE":   {{Target: "oneWay"}},
							"CHANGE_RETURN_DATE": {{Actions: xs.Actions{xs.ActionRef{Type: "setReturnDate"}}}},
							"BOOK_RETURN": {{
								Target: "#flightBookerMachine.booking",
								Guard:  xs.GuardRef{Type: "isValidReturnDate?"},
							}},
						},
					},
				},
			},
			{
				Key: "booking",
				Invoke: []xs.InvokeConfig{{
					Src:     "Booker",
					OnDone:  xs.Transitions{{Target: "booked"}},
					OnError: xs.Transitions{{Target: "scheduling"}},
				}},
			},
			{Key: "booked", Type: xs.Final},
		},
	})
}
