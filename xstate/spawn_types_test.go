package xstate_test

import (
	"testing"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/assert"
)

// JS: spawn inside machine > input is required when defined in actor
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/spawn.types.test.ts#L4
func TestSpawnTypes_SpawnInsideMachine_InputIsRequiredWhenDefinedInActor(t *testing.T) {
	type ctx struct{ Ref xs.ActorRef }

	// The JS test has no `expect`; the runtime part is that building the machines does not throw.
	assert.NotPanics(t, func() {
		// types: { input: {} as { value: number } }
		childMachine := xs.CreateMachine(xs.MachineConfig[any]{})

		xs.CreateMachine(xs.MachineConfig[ctx]{
			ContextFn: func(a xs.ContextArgs) ctx {
				return ctx{Ref: a.Spawn(childMachine, xs.SpawnOptions{Input: map[string]any{"value": 42}})}
			},
			Initial: "idle",
			States: xs.States{
				{Key: "Idle", On: map[string]xs.Transitions{
					"event": {{Actions: xs.Actions{
						xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
							return ctx{Ref: a.Spawn(childMachine, xs.SpawnOptions{Input: map[string]any{"value": 42}})}
						}),
					}}},
				}},
			},
		})
	})
}

// JS: spawn inside machine > input is not required when not defined in actor
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/spawn.types.test.ts#L28
func TestSpawnTypes_SpawnInsideMachine_InputIsNotRequiredWhenNotDefinedInActor(t *testing.T) {
	type ctx struct{ Ref xs.ActorRef }

	// The JS test has no `expect`; the runtime part is that building the machines does not throw.
	assert.NotPanics(t, func() {
		childMachine := xs.CreateMachine(xs.MachineConfig[any]{})

		xs.CreateMachine(xs.MachineConfig[ctx]{
			ContextFn: func(a xs.ContextArgs) ctx {
				return ctx{Ref: a.Spawn(childMachine)}
			},
			Initial: "idle",
			States: xs.States{
				{Key: "Idle", On: map[string]xs.Transitions{
					"some": {{Actions: xs.Actions{
						xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
							return ctx{Ref: a.Spawn(childMachine)}
						}),
					}}},
				}},
			},
		})
	})
}
