package xstate_test

import (
	"context"
	"testing"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/assert"
)

// JS: spawnChild action > can spawn
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/spawnChild.test.ts#L13
func TestSpawnChild_CanSpawn(t *testing.T) {
	actor := xs.CreateActor(
		xs.CreateMachine(xs.MachineConfig[any]{
			Entry: xs.Actions{xs.SpawnChild(
				xs.FromPromise(func(_ context.Context, _ xs.PromiseArgs) (int, error) { return 42, nil }),
				xs.SpawnOptions{ID: "child"},
			)},
		}),
	)

	actor.Start()

	assert.NotNil(t, actor.GetSnapshot().Children["child"])
}

// JS: spawnChild action > can spawn from named actor
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/spawnChild.test.ts#L28
func TestSpawnChild_CanSpawnFromNamedActor(t *testing.T) {
	fetchNum := xs.FromPromise(func(_ context.Context, a xs.PromiseArgs) (int, error) {
		return a.Input.(int) * 2, nil
	})
	actor := xs.CreateActor(
		xs.CreateMachine(xs.MachineConfig[any]{
			Entry: xs.Actions{xs.SpawnChild("fetchNum", xs.SpawnOptions{ID: "child", Input: 21})},
		}).Provide(xs.Implementations{
			Actors: map[string]xs.ActorLogic{"fetchNum": fetchNum},
		}),
	)

	actor.Start()

	assert.NotNil(t, actor.GetSnapshot().Children["child"])
}

// JS: spawnChild action > should accept `syncSnapshot` option
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/spawnChild.test.ts#L51
func TestSpawnChild_ShouldAcceptSyncSnapshotOption(t *testing.T) {
	type ctx struct{ ObservableRef xs.ActorRef }

	sig := newSignal()
	observableLogic := xs.FromObservable(func(_ xs.ObservableArgs) xs.Subscribable[int] { return rxInterval(10) })
	observableMachine := xs.CreateMachine(xs.MachineConfig[ctx]{
		ID:      "observable",
		Initial: "idle",
		Context: ctx{ObservableRef: nil},
		States: xs.States{
			{
				Key: "idle",
				Entry: xs.Actions{xs.SpawnChild(observableLogic, xs.SpawnOptions{
					ID:           "int",
					SyncSnapshot: true,
				})},
				On: map[string]xs.Transitions{
					"xstate.snapshot.int": {{
						Target: "success",
						Guard: xs.GuardFunc(func(a xs.GuardArgs[ctx]) bool {
							ev, ok := a.Event.(xs.SnapshotEvent)
							if !ok {
								return false
							}
							snap, ok := ev.Snapshot.(*xs.ObservableSnapshot[int])
							return ok && snap.Context == 5
						}),
					}},
				},
			},
			{Key: "success", Type: xs.Final},
		},
	})

	observableService := xs.CreateActor(observableMachine)
	observableService.Subscribe(xs.Observer[*xs.MachineSnapshot[ctx]]{
		Complete: func() { sig.Resolve() },
	})

	observableService.Start()

	sig.Wait(t)
}

// JS: spawnChild action > should handle a dynamic id
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/spawnChild.test.ts#L91
func TestSpawnChild_ShouldHandleADynamicID(t *testing.T) {
	type ctx struct{ ChildID string }

	spy := newSpy()

	child := xs.CreateMachine(xs.MachineConfig[any]{
		On: map[string]xs.Transitions{
			"FOO": {{Actions: xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[any]) { spy.Call(a) })}}},
		},
	})

	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Context: ctx{ChildID: "myChild"},
		Entry: xs.Actions{
			xs.SpawnChild(child, xs.SpawnOptions{ID: xs.NewExpr(func(a xs.ExprArgs[ctx]) any { return a.Context.ChildID })}),
			xs.SendTo("myChild", xs.Ev("FOO")),
		},
	})

	xs.CreateActor(machine).Start()

	assert.Equal(t, 1, spy.Count())
}
