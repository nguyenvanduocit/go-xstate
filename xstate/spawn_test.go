package xstate_test

import (
	"testing"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/assert"
)

// JS: spawn inside machine > input is required when defined in actor
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/spawn.test.ts#L4
func TestSpawn_SpawnInsideMachine_InputIsRequiredWhenDefinedInActor(t *testing.T) {
	type childInput struct{ Value int }
	type ctx struct{ Ref xs.ActorRef }

	childMachine := xs.CreateMachine(xs.MachineConfig[any]{})
	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		ContextFn: func(a xs.ContextArgs) ctx {
			return ctx{
				Ref: a.Spawn(childMachine, xs.SpawnOptions{Input: childInput{Value: 42}, SystemID: "test"}),
			}
		},
	})

	actor := xs.CreateActor(machine).Start()
	assert.NotNil(t, actor.System().Get("test"))
}
