// Package timer ports references/xstate/examples/timer/src/timerMachine.ts.
package timer

import (
	"time"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// TickInterval is the period of the production ticks actor (setInterval(..., 1000) in JS).
const TickInterval = time.Second

// Context is the timer's extended state.
type Context struct {
	Seconds int `json:"seconds"`
}

// Ticks mirrors the `ticks` fromCallback actor: it sends TICK every interval
// until the invoking state exits (the returned cleanup stops the ticker).
func Ticks(interval time.Duration) xs.ActorLogic {
	return xs.FromCallback(func(a xs.CallbackArgs) func() {
		ticker := time.NewTicker(interval)
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
		}
	})
}

func addSeconds(delta int) xs.Action {
	return xs.Assign(func(a xs.AssignArgs[Context]) Context {
		return Context{Seconds: a.Context.Seconds + delta}
	})
}

// NewMachine mirrors timerMachine with the `ticks` actor supplied by the caller,
// so tests can swap the wall-clock ticker for a stub.
func NewMachine(ticks xs.ActorLogic) *xs.StateMachine[Context] {
	return xs.NewSetup[Context](xs.Implementations{
		Actors: map[string]xs.ActorLogic{"ticks": ticks},
	}).CreateMachine(xs.MachineConfig[Context]{
		Initial: "stopped",
		Context: Context{Seconds: 0},
		States: xs.States{
			{
				Key: "stopped",
				On: map[string]xs.Transitions{
					"start": {{
						Guard:  xs.GuardFunc(func(a xs.GuardArgs[Context]) bool { return a.Context.Seconds > 0 }),
						Target: "running",
					}},
					"minute": {{Actions: xs.Actions{addSeconds(60)}}},
					"second": {{Actions: xs.Actions{addSeconds(1)}}},
				},
			},
			{
				Key:    "running",
				Invoke: []xs.InvokeConfig{{Src: "ticks"}},
				On: map[string]xs.Transitions{
					"stop": {{Target: "stopped"}},
					"TICK": {{Actions: xs.Actions{addSeconds(-1)}}},
				},
				Always: xs.Transitions{{
					Guard:  xs.GuardFunc(func(a xs.GuardArgs[Context]) bool { return a.Context.Seconds == 0 }),
					Target: "stopped",
				}},
			},
		},
		On: map[string]xs.Transitions{
			"reset": {{
				Guard: xs.GuardFunc(func(a xs.GuardArgs[Context]) bool { return a.Context.Seconds > 0 }),
				Actions: xs.Actions{xs.Assign(func(xs.AssignArgs[Context]) Context {
					return Context{Seconds: 0}
				})},
			}},
		},
	})
}

// Machine is timerMachine with the real 1 s ticker.
func Machine() *xs.StateMachine[Context] { return NewMachine(Ticks(TickInterval)) }
