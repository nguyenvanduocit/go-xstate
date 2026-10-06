// Package persistedstate ports references/xstate/examples/mongodb-persisted-state (donutMachine.ts,
// TaskQueue.ts and the persist/restore loop of main.ts).
package persistedstate

import xs "github.com/nguyenvanduocit/go-xstate/xstate"

// Context is the donut machine's extended state. The JS machine has no context,
// so its snapshots carry the default empty object `{}`.
type Context struct{}

// Snapshot is the donut machine's snapshot type.
type Snapshot = *xs.MachineSnapshot[Context]

// mixRegion returns the region config shared by mixDry and mixWet.
func mixRegion(key, event string) xs.StateConfig {
	return xs.StateConfig{
		Key:     key,
		Initial: "mixing",
		States: xs.States{
			{Key: "mixing", On: map[string]xs.Transitions{event: {{Target: "mixed"}}}},
			{Key: "mixed", Type: xs.Final},
		},
	}
}

// DonutMachine mirrors donutMachine.
func DonutMachine() *xs.StateMachine[Context] {
	next := func(target string) map[string]xs.Transitions {
		return map[string]xs.Transitions{"NEXT": {{Target: target}}}
	}
	return xs.CreateMachine(xs.MachineConfig[Context]{
		ID:      "donut",
		Initial: "ingredients",
		States: xs.States{
			{Key: "ingredients", On: next("directions")},
			{
				Key:     "directions",
				Initial: "makeDough",
				OnDone:  xs.Transitions{{Target: "fry"}},
				States: xs.States{
					{Key: "makeDough", On: next("mix")},
					{
						Key:    "mix",
						Type:   xs.Parallel,
						OnDone: xs.Transitions{{Target: "allMixed"}},
						States: xs.States{
							mixRegion("mixDry", "MIXED_DRY"),
							mixRegion("mixWet", "MIXED_WET"),
						},
					},
					{Key: "allMixed", Type: xs.Final},
				},
			},
			{Key: "fry", On: next("flip")},
			{Key: "flip", On: next("dry")},
			{Key: "dry", On: next("glaze")},
			{Key: "glaze", On: next("serve")},
			{Key: "serve", On: map[string]xs.Transitions{"ANOTHER_DONUT": {{Target: "ingredients"}}}},
		},
	})
}
