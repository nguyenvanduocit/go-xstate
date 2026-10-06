// Package countervue ports references/xstate/examples/7guis-1-counter-vue/src/counterMachine.ts.
package countervue

import xs "github.com/nguyenvanduocit/go-xstate/xstate"

// Context is the counter's extended state.
type Context struct {
	Count int `json:"count"`
}

// Machine mirrors counterMachine.
func Machine() *xs.StateMachine[Context] {
	return xs.CreateMachine(xs.MachineConfig[Context]{
		ID:      "Counter",
		Initial: "ready",
		Context: Context{Count: 0},
		States: xs.States{
			{Key: "ready", On: map[string]xs.Transitions{
				"increase": {{
					Target: "ready",
					Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[Context]) Context {
						return Context{Count: a.Context.Count + 1}
					})},
				}},
			}},
		},
	})
}
