// Package counter ports references/xstate/examples/counter/src/counterMachine.ts.
package counter

import xs "github.com/nguyenvanduocit/go-xstate/xstate"

// Context is the counter's extended state.
type Context struct {
	Count int `json:"count"`
}

func inc(delta int) xs.Action {
	return xs.Assign(func(a xs.AssignArgs[Context]) Context {
		return Context{Count: a.Context.Count + delta}
	})
}

// Machine mirrors counterMachine.
func Machine() *xs.StateMachine[Context] {
	return xs.CreateMachine(xs.MachineConfig[Context]{
		ID:      "counter",
		Context: Context{Count: 0},
		On: map[string]xs.Transitions{
			"increment": {{Actions: xs.Actions{inc(1)}}},
			"decrement": {{Actions: xs.Actions{inc(-1)}}},
		},
	})
}
