// Package toggle ports references/xstate/examples/toggle/src/toggleMachine.ts.
package toggle

import xs "github.com/nguyenvanduocit/go-xstate/xstate"

// Context is empty: the JS machine declares none, and xstate snapshots then
// carry `context: {}` (StateMachine.ts:262), which an empty struct serializes to.
type Context struct{}

// Machine mirrors toggleMachine.
func Machine() *xs.StateMachine[Context] {
	return xs.CreateMachine(xs.MachineConfig[Context]{
		ID:      "toggle",
		Initial: "inactive",
		States: xs.States{
			{Key: "inactive", On: map[string]xs.Transitions{
				"toggle": {{Target: "active"}},
			}},
			{Key: "active", On: map[string]xs.Transitions{
				"toggle": {{Target: "inactive"}},
			}},
		},
	})
}
