package xstate_test

import (
	"testing"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/assert"
)

// JS: getNextSnapshot > should calculate the next snapshot for transition logic
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/getNextSnapshot.test.ts#L9
func TestGetNextSnapshot_ShouldCalculateTheNextSnapshotForTransitionLogic(t *testing.T) {
	type state struct{ Count int }

	logic := xs.FromTransition(
		func(s state, event xs.Event, _ *xs.ActorScope) state {
			if event.EventType() == "next" {
				return state{Count: s.Count + 1}
			}
			return s
		},
		func(_ xs.TransitionInitArgs) state { return state{Count: 0} },
	)

	init := xs.GetInitialSnapshot(logic)
	s1 := xs.GetNextSnapshot(logic, init, xs.Ev("next"))
	assert.Equal(t, 1, s1.Context.Count)
	s2 := xs.GetNextSnapshot(logic, s1, xs.Ev("next"))
	assert.Equal(t, 2, s2.Context.Count)
}

// JS: getNextSnapshot > should calculate the next snapshot for machine logic
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/getNextSnapshot.test.ts#L27
func TestGetNextSnapshot_ShouldCalculateTheNextSnapshotForMachineLogic(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{"NEXT": {{Target: "b"}}}},
			{Key: "b", On: map[string]xs.Transitions{"NEXT": {{Target: "c"}}}},
			{Key: "c"},
		},
	})

	init := xs.GetInitialSnapshot(machine)
	s1 := xs.GetNextSnapshot(machine, init, xs.Ev("NEXT"))

	assert.Equal(t, "b", s1.Value)

	s2 := xs.GetNextSnapshot(machine, s1, xs.Ev("NEXT"))

	assert.Equal(t, "c", s2.Value)
}

// JS: getNextSnapshot > should not execute actions
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/getNextSnapshot.test.ts#L54
func TestGetNextSnapshot_ShouldNotExecuteActions(t *testing.T) {
	fn := newSpy()

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{
				"event": {{
					Target:  "b",
					Actions: xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[any]) { fn.Call(a) })},
				}},
			}},
			{Key: "b"},
		},
	})

	init := xs.GetInitialSnapshot(machine)
	nextSnapshot := xs.GetNextSnapshot(machine, init, xs.Ev("event"))

	assert.Equal(t, 0, fn.Count())
	assert.Equal(t, "b", nextSnapshot.Value)
}
