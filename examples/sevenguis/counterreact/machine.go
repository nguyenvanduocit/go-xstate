// Package counterreact ports references/xstate/examples/7guis-counter-react/src/counterMachine.ts.
// The directory name starts with a digit, which is not a valid Go identifier, so the
// package name spells the number out.
package counterreact

import xs "github.com/nguyenvanduocit/go-xstate/xstate"

// Context is the counter's extended state.
type Context struct {
	Count int `json:"count"`
}

// Machine mirrors counterMachine.
func Machine() *xs.StateMachine[Context] {
	return xs.CreateMachine(xs.MachineConfig[Context]{
		ID:      "counter",
		Context: Context{Count: 0},
		On: map[string]xs.Transitions{
			"INCREMENT": {{Actions: xs.Actions{
				xs.Assign(func(a xs.AssignArgs[Context]) Context {
					return Context{Count: a.Context.Count + 1}
				}),
			}}},
		},
	})
}
