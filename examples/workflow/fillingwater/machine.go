// Package fillingwater ports references/xstate/examples/workflow-filling-water/main.ts
// (serverless workflow "filling a glass of water").
package fillingwater

import (
	"fmt"
	"io"
	"strconv"
	"time"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// Counts is `counts` of the context: the current and maximum water level.
type Counts struct {
	Current int `json:"current"`
	Max     int `json:"max"`
}

// Input is the machine input: `{ current, max }`.
type Input struct {
	Current int `json:"current"`
	Max     int `json:"max"`
}

// Context is the machine's extended state.
type Context struct {
	Counts Counts `json:"counts"`
}

// AddWaterDelay is the `after` delay of AddWater in main.ts.
const AddWaterDelay = 500 * time.Millisecond

// Machine mirrors `workflow` with main.ts's 500 ms delay.
func Machine() *xs.StateMachine[Context] { return NewMachine(AddWaterDelay) }

// NewMachine mirrors `workflow` with the AddWater delay given (whole milliseconds, as the
// numeric key of JS `after`).
func NewMachine(delay time.Duration) *xs.StateMachine[Context] {
	return xs.CreateMachine(xs.MachineConfig[Context]{
		ID:      "fillglassofwater",
		Initial: "CheckIfFull",
		ContextFn: func(a xs.ContextArgs) Context {
			in := a.Input.(Input)
			return Context{Counts: Counts{Current: in.Current, Max: in.Max}}
		},
		States: xs.States{
			{
				Key: "CheckIfFull",
				Always: xs.Transitions{
					{
						Target: "AddWater",
						Guard: xs.GuardFunc(func(a xs.GuardArgs[Context]) bool {
							return a.Context.Counts.Current < a.Context.Counts.Max
						}),
					},
					{Target: "GlassFull"},
				},
			},
			{
				Key: "AddWater",
				After: map[string]xs.Transitions{
					strconv.FormatInt(delay.Milliseconds(), 10): {{
						Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[Context]) Context {
							c := a.Context
							c.Counts.Current++
							return c
						})},
						Target: "CheckIfFull",
					}},
				},
			},
			{Key: "GlassFull", Type: xs.Final},
		},
	})
}

// Run mirrors the entry of main.ts with the real 500 ms delay (about 5 s), printing to w.
func Run(w io.Writer) { RunWith(w, AddWaterDelay) }

// RunWith is Run with the AddWater delay injected. It returns once the glass is full and
// "workflow completed" has been printed.
func RunWith(w io.Writer, delay time.Duration) {
	done := make(chan struct{})
	actor := xs.CreateActor(NewMachine(delay), xs.WithInput(Input{Current: 0, Max: 10}))
	actor.Subscribe(xs.Observer[*xs.MachineSnapshot[Context]]{
		Next: func(s *xs.MachineSnapshot[Context]) {
			fmt.Fprintf(w, "workflow state %v\n", s.Value)
			fmt.Fprintf(w, "workflow context {\n  counts: {\n    current: %d,\n    max: %d,\n  },\n}\n",
				s.Context.Counts.Current, s.Context.Counts.Max)
		},
		Complete: func() {
			fmt.Fprintln(w, "workflow completed", formatOutput(actor.GetSnapshot().Output))
			close(done)
		},
	})
	actor.Start()
	<-done
}

// formatOutput renders the machine output the way console.log prints it in the JS runtime:
// this machine has no output, so `undefined`.
func formatOutput(out any) string {
	if out == nil {
		return "undefined"
	}
	return fmt.Sprint(out)
}
