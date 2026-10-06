package xstate_test

import (
	"sync/atomic"
	"testing"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/assert"
)

// state1ExampleMachine mirrors the top-level `exampleMachine` (state.test.ts L22-107).
// It is a constructor because a package-level var would call CreateMachine at init.
func state1ExampleMachine() *xs.StateMachine[any] {
	return xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "one",
		States: xs.States{
			{
				Key:   "one",
				Entry: xs.Actions{xs.ActionRef{Type: "enter"}},
				On: map[string]xs.Transitions{
					"EXTERNAL": {{Target: "one", Reenter: true}},
					"INERT":    {{}},
					"INTERNAL": {{Actions: xs.Actions{xs.ActionRef{Type: "doSomething"}}}},
					"TO_TWO":   {{Target: "two"}},
					"TO_TWO_MAYBE": {{
						Target: "two",
						Guard:  xs.GuardFunc(func(a xs.GuardArgs[any]) bool { return true }),
					}},
					"TO_THREE":        {{Target: "three"}},
					"FORBIDDEN_EVENT": nil,
					"TO_FINAL":        {{Target: "success"}},
				},
			},
			{
				Key:     "two",
				Initial: "deep",
				States: xs.States{
					{
						Key:     "deep",
						Initial: "foo",
						States: xs.States{
							{Key: "foo", On: map[string]xs.Transitions{
								"FOO_EVENT":       {{Target: "bar"}},
								"FORBIDDEN_EVENT": nil,
							}},
							{Key: "bar", On: map[string]xs.Transitions{
								"BAR_EVENT": {{Target: "foo"}},
							}},
						},
					},
				},
				On: map[string]xs.Transitions{
					"DEEP_EVENT": {{Target: "."}},
				},
			},
			{
				Key:  "three",
				Type: xs.Parallel,
				States: xs.States{
					{
						Key:     "first",
						Initial: "p31",
						States: xs.States{
							{Key: "p31", On: map[string]xs.Transitions{"P31": {{Target: "."}}}},
						},
					},
					{
						Key:     "guarded",
						Initial: "p32",
						States: xs.States{
							{Key: "p32", On: map[string]xs.Transitions{"P32": {{Target: "."}}}},
						},
					},
				},
				On: map[string]xs.Transitions{
					"THREE_EVENT": {{Target: "."}},
				},
			},
			{Key: "success", Type: xs.Final},
		},
		On: map[string]xs.Transitions{
			"MACHINE_EVENT": {{Target: ".two"}},
		},
	})
}

// JS: State > status > should show that a machine has not reached its final state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/state.test.ts#L110
func TestState_Status_ShouldShowMachineHasNotReachedFinalState(t *testing.T) {
	assert.NotEqual(t, xs.StatusDone, xs.CreateActor(state1ExampleMachine()).GetSnapshot().Status)
}

// JS: State > status > should show that a machine has reached its final state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/state.test.ts#L114
func TestState_Status_ShouldShowMachineHasReachedFinalState(t *testing.T) {
	actorRef := xs.CreateActor(state1ExampleMachine()).Start()
	actorRef.Send(xs.Ev("TO_FINAL"))
	assert.Equal(t, xs.StatusDone, actorRef.GetSnapshot().Status)
}

// JS: State > .can > should return true for a simple event that results in a transition to a different state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/state.test.ts#L122
func TestState_Can_ShouldReturnTrueForSimpleEventTransitioningToDifferentState(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{"NEXT": {{Target: "b"}}}},
			{Key: "b"},
		},
	})

	assert.Equal(t, true, xs.CreateActor(machine).GetSnapshot().Can(xs.Ev("NEXT")))
}

// JS: State > .can > should return true for an event object that results in a transition to a different state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/state.test.ts#L140
func TestState_Can_ShouldReturnTrueForEventObjectTransitioningToDifferentState(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{"NEXT": {{Target: "b"}}}},
			{Key: "b"},
		},
	})

	assert.Equal(t, true, xs.CreateActor(machine).GetSnapshot().Can(xs.Ev("NEXT")))
}

// JS: State > .can > should return true for an event object that results in a new action
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/state.test.ts#L158
func TestState_Can_ShouldReturnTrueForEventObjectResultingInNewAction(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{
				"NEXT": {{Actions: xs.Actions{xs.ActionRef{Type: "newAction"}}}},
			}},
		},
	})

	assert.Equal(t, true, xs.CreateActor(machine).GetSnapshot().Can(xs.Ev("NEXT")))
}

// JS: State > .can > should return true for an event object that results in a context change
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/state.test.ts#L177
func TestState_Can_ShouldReturnTrueForEventObjectResultingInContextChange(t *testing.T) {
	type ctx struct{ Count int }

	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Initial: "a",
		Context: ctx{Count: 0},
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{
				"NEXT": {{Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
					c := a.Context
					c.Count = 1
					return c
				})}}},
			}},
		},
	})

	assert.Equal(t, true, xs.CreateActor(machine).GetSnapshot().Can(xs.Ev("NEXT")))
}

// JS: State > .can > should return true for a reentering self-transition without actions
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/state.test.ts#L197
func TestState_Can_ShouldReturnTrueForReenteringSelfTransitionWithoutActions(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{"EV": {{Target: "a"}}}},
		},
	})

	assert.Equal(t, true, xs.CreateActor(machine).GetSnapshot().Can(xs.Ev("EV")))
}

// JS: State > .can > should return true for a reentering self-transition with reentry action
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/state.test.ts#L212
func TestState_Can_ShouldReturnTrueForReenteringSelfTransitionWithReentryAction(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{
				Key:   "a",
				Entry: xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[any]) {})},
				On:    map[string]xs.Transitions{"EV": {{Target: "a"}}},
			},
		},
	})

	assert.Equal(t, true, xs.CreateActor(machine).GetSnapshot().Can(xs.Ev("EV")))
}

// JS: State > .can > should return true for a reentering self-transition with transition action
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/state.test.ts#L228
func TestState_Can_ShouldReturnTrueForReenteringSelfTransitionWithTransitionAction(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{
				"EV": {{
					Target:  "a",
					Actions: xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[any]) {})},
				}},
			}},
		},
	})

	assert.Equal(t, true, xs.CreateActor(machine).GetSnapshot().Can(xs.Ev("EV")))
}

// JS: State > .can > should return true for a targetless transition with actions
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/state.test.ts#L246
func TestState_Can_ShouldReturnTrueForTargetlessTransitionWithActions(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{
				"EV": {{Actions: xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[any]) {})}}},
			}},
		},
	})

	assert.Equal(t, true, xs.CreateActor(machine).GetSnapshot().Can(xs.Ev("EV")))
}

// JS: State > .can > should return false for a forbidden transition
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/state.test.ts#L263
func TestState_Can_ShouldReturnFalseForForbiddenTransition(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{"EV": nil}},
		},
	})

	assert.Equal(t, false, xs.CreateActor(machine).GetSnapshot().Can(xs.Ev("EV")))
}

// JS: State > .can > should return false for an unknown event
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/state.test.ts#L280
func TestState_Can_ShouldReturnFalseForUnknownEvent(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{"NEXT": {{Target: "b"}}}},
			{Key: "b"},
		},
	})

	assert.Equal(t, false, xs.CreateActor(machine).GetSnapshot().Can(xs.Ev("UNKNOWN")))
}

// JS: State > .can > should return true when a guarded transition allows the transition
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/state.test.ts#L298
func TestState_Can_ShouldReturnTrueWhenGuardedTransitionAllowsTransition(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{
				"CHECK": {{
					Target: "b",
					Guard:  xs.GuardFunc(func(a xs.GuardArgs[any]) bool { return true }),
				}},
			}},
			{Key: "b"},
		},
	})

	assert.Equal(t, true, xs.CreateActor(machine).GetSnapshot().Can(xs.Ev("CHECK")))
}

// JS: State > .can > should return false when a guarded transition disallows the transition
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/state.test.ts#L321
func TestState_Can_ShouldReturnFalseWhenGuardedTransitionDisallowsTransition(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{
				"CHECK": {{
					Target: "b",
					Guard:  xs.GuardFunc(func(a xs.GuardArgs[any]) bool { return false }),
				}},
			}},
			{Key: "b"},
		},
	})

	assert.Equal(t, false, xs.CreateActor(machine).GetSnapshot().Can(xs.Ev("CHECK")))
}

// JS: State > .can > should not spawn actors when determining if an event is accepted
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/state.test.ts#L344
func TestState_Can_ShouldNotSpawnActorsWhenDeterminingIfEventIsAccepted(t *testing.T) {
	type ctx struct{ Ref xs.ActorRef }

	var spawned atomic.Bool
	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Context: ctx{},
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{
				"SPAWN": {{Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
					return ctx{Ref: a.Spawn(xs.FromCallback(func(xs.CallbackArgs) func() {
						spawned.Store(true)
						return nil
					}))}
				})}}},
			}},
			{Key: "b"},
		},
	})

	service := xs.CreateActor(machine).Start()
	service.GetSnapshot().Can(xs.Ev("SPAWN"))
	assert.Equal(t, false, spawned.Load())
}

// JS: State > .can > should not execute assignments when used with non-started actor
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/state.test.ts#L372
func TestState_Can_ShouldNotExecuteAssignmentsWithNonStartedActor(t *testing.T) {
	type ctx struct{}

	executed := false
	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Context: ctx{},
		On: map[string]xs.Transitions{
			"EVENT": {{Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
				// Side-effect just for testing
				executed = true
				return a.Context
			})}}},
		},
	})

	actorRef := xs.CreateActor(machine)

	assert.True(t, actorRef.GetSnapshot().Can(xs.Ev("EVENT")))

	assert.False(t, executed)
}

// JS: State > .can > should not execute assignments when used with started actor
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/state.test.ts#L394
func TestState_Can_ShouldNotExecuteAssignmentsWithStartedActor(t *testing.T) {
	type ctx struct{}

	executed := false
	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Context: ctx{},
		On: map[string]xs.Transitions{
			"EVENT": {{Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
				// Side-effect just for testing
				executed = true
				return a.Context
			})}}},
		},
	})

	actorRef := xs.CreateActor(machine).Start()

	assert.True(t, actorRef.GetSnapshot().Can(xs.Ev("EVENT")))

	assert.False(t, executed)
}

// JS: State > .can > should return true when non-first parallel region changes value
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/state.test.ts#L416
func TestState_Can_ShouldReturnTrueWhenNonFirstParallelRegionChangesValue(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Type: xs.Parallel,
		States: xs.States{
			{
				Key:     "a",
				Initial: "a1",
				States: xs.States{
					{
						Key: "a1",
						ID:  "foo",
						On: map[string]xs.Transitions{
							// first region doesn't change value here
							"EVENT": {{Targets: []string{"#foo", "#bar"}}},
						},
					},
				},
			},
			{
				Key:     "b",
				Initial: "b1",
				States: xs.States{
					{Key: "b1"},
					{Key: "b2", ID: "bar"},
				},
			},
		},
	})

	assert.True(t, xs.CreateActor(machine).GetSnapshot().Can(xs.Ev("EVENT")))
}

// JS: State > .can > should return true when transition targets a state that is already part of the current configuration but the final state value changes
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/state.test.ts#L449
func TestState_Can_ShouldReturnTrueWhenTargetAlreadyActiveButFinalStateValueChanges(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{
				Key:     "a",
				ID:      "foo",
				Initial: "a1",
				States: xs.States{
					{Key: "a1", On: map[string]xs.Transitions{"NEXT": {{Target: "a2"}}}},
					{Key: "a2", On: map[string]xs.Transitions{"NEXT": {{Target: "#foo"}}}},
				},
			},
		},
	})

	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.Ev("NEXT"))

	assert.True(t, actorRef.GetSnapshot().Can(xs.Ev("NEXT")))
}

// JS: State > .hasTag > should be able to check a tag after recreating a persisted state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/state.test.ts#L480
func TestState_HasTag_ShouldCheckTagAfterRecreatingPersistedState(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", Tags: xs.Tags{"foo"}},
		},
	})

	actorRef := xs.CreateActor(machine).Start()
	persistedState := actorRef.GetPersistedSnapshot()
	actorRef.Stop()
	restoredSnapshot := xs.CreateActor(machine, xs.WithSnapshot(persistedState)).GetSnapshot()

	assert.Equal(t, true, restoredSnapshot.HasTag("foo"))
}

// JS: State > .status > should be 'stopped' after a running actor gets stopped
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/state.test.ts#L502
func TestState_Status_ShouldBeStoppedAfterRunningActorGetsStopped(t *testing.T) {
	snapshot := xs.CreateActor(xs.CreateMachine(xs.MachineConfig[any]{})).
		Start().
		Stop().
		GetSnapshot()
	assert.Equal(t, xs.StatusStopped, snapshot.Status)
}
