// Package workflow ports references/xstate/examples/express-workflow (machine.ts and index.ts).
package workflow

import xs "github.com/nguyenvanduocit/go-xstate/xstate"

// Context is the traffic-light machine's extended state.
type Context struct {
	Cycles int `json:"cycles"`
}

// Machine mirrors machine.ts. The JS id is 'counter' although the machine is a
// traffic light; it is kept as is.
func Machine() *xs.StateMachine[Context] {
	return xs.CreateMachine(xs.MachineConfig[Context]{
		ID:      "counter",
		Initial: "green",
		Context: Context{Cycles: 0},
		States: xs.States{
			{Key: "green", On: map[string]xs.Transitions{
				"TIMER": {{Target: "yellow"}},
			}},
			{Key: "yellow", On: map[string]xs.Transitions{
				"TIMER": {{Target: "red"}},
			}},
			{Key: "red", On: map[string]xs.Transitions{
				"TIMER": {{
					Target: "green",
					Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[Context]) Context {
						return Context{Cycles: a.Context.Cycles + 1}
					})},
				}},
			}},
		},
	})
}
