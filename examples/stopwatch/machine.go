// Package stopwatch ports references/xstate/examples/stopwatch/src/stopwatchMachine.ts.
package stopwatch

import (
	"time"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// TickInterval is the period of the production ticks actor (setInterval(..., 10) in JS).
const TickInterval = 10 * time.Millisecond

// Context is the stopwatch's extended state.
type Context struct {
	Elapsed int `json:"elapsed"`
}

// Ticks mirrors the `ticks` fromCallback actor: it sends TICK every TickInterval
// until the invoking state exits (the returned cleanup stops the ticker).
func Ticks() xs.ActorLogic {
	return xs.FromCallback(func(a xs.CallbackArgs) func() {
		ticker := time.NewTicker(TickInterval)
		stop := make(chan struct{})
		go func() {
			for {
				select {
				case <-ticker.C:
					a.SendBack(xs.Ev("TICK"))
				case <-stop:
					return
				}
			}
		}()
		return func() {
			ticker.Stop()
			close(stop)
			// SendBack may be waiting for the system lock held by cleanup.
			// The callback actor discards that send after it is stopped.
		}
	})
}

// NewMachine mirrors stopwatchMachine with the `ticks` actor supplied by the caller,
// so tests can swap the wall-clock ticker for a stub.
func NewMachine(ticks xs.ActorLogic) *xs.StateMachine[Context] {
	return xs.NewSetup[Context](xs.Implementations{
		Actors: map[string]xs.ActorLogic{"ticks": ticks},
	}).CreateMachine(xs.MachineConfig[Context]{
		ID:      "stopwatch",
		Initial: "stopped",
		Context: Context{Elapsed: 0},
		States: xs.States{
			{
				Key: "stopped",
				On:  map[string]xs.Transitions{"start": {{Target: "running"}}},
			},
			{
				Key:    "running",
				Invoke: []xs.InvokeConfig{{Src: "ticks"}},
				On: map[string]xs.Transitions{
					"TICK": {{Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[Context]) Context {
						return Context{Elapsed: a.Context.Elapsed + 1}
					})}}},
					"stop": {{Target: "stopped"}},
				},
			},
		},
		On: map[string]xs.Transitions{
			"reset": {{
				Target: ".stopped",
				Actions: xs.Actions{xs.Assign(func(xs.AssignArgs[Context]) Context {
					return Context{Elapsed: 0}
				})},
			}},
		},
	})
}

// Machine is stopwatchMachine with the real 10 ms ticker.
func Machine() *xs.StateMachine[Context] { return NewMachine(Ticks()) }
