package xstate_test

import (
	"testing"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func microstep1Values[C any](states []*xs.MachineSnapshot[C]) []any {
	out := make([]any, len(states))
	for i, s := range states {
		out[i] = s.Value
	}
	return out
}

func microstep1Noop() xs.Action { return xs.ActionFunc(func(xs.ActionArgs[any]) {}) }

// JS: machine.microstep() > should return an array of states from all microsteps
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/microstep.test.ts#L10
func TestMicrostep_MachineMicrostep_ShouldReturnAnArrayOfStatesFromAllMicrosteps(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "start",
		States: xs.States{
			{Key: "start", On: map[string]xs.Transitions{"GO": {{Target: "a"}}}},
			{
				Key:   "a",
				Entry: xs.Actions{xs.Raise(xs.Ev("NEXT"))},
				On:    map[string]xs.Transitions{"NEXT": {{Target: "b"}}},
			},
			{Key: "b", Always: xs.Transitions{{Target: "c"}}},
			{
				Key:   "c",
				Entry: xs.Actions{xs.Raise(xs.Ev("NEXT"))},
				On:    map[string]xs.Transitions{"NEXT": {{Target: "d"}}},
			},
			{Key: "d"},
		},
	})

	actorScope := xs.CreateInertActorScope(machine)
	states := machine.Microstep(
		machine.GetInitialSnapshot(actorScope, nil),
		xs.Ev("GO"),
		actorScope,
	)

	assert.Equal(t, []any{"a", "b", "c", "d"}, microstep1Values(states))
}

// JS: machine.microstep() > should return the states from microstep (transient)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/microstep.test.ts#L48
func TestMicrostep_MachineMicrostep_ShouldReturnTheStatesFromMicrostepTransient(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "first",
		States: xs.States{
			{Key: "first", On: map[string]xs.Transitions{"TRIGGER": {{Target: "second"}}}},
			{Key: "second", Always: xs.Transitions{{Target: "third"}}},
			{Key: "third"},
		},
	})

	actorScope := xs.CreateInertActorScope(machine)
	states := machine.Microstep(
		machine.ResolveState(xs.ResolveStateConfig[any]{Value: "first"}),
		xs.Ev("TRIGGER"),
		actorScope,
	)

	assert.Equal(t, []any{"second", "third"}, microstep1Values(states))
}

// JS: machine.microstep() > should return the states from microstep (raised event)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/microstep.test.ts#L74
func TestMicrostep_MachineMicrostep_ShouldReturnTheStatesFromMicrostepRaisedEvent(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "first",
		States: xs.States{
			{Key: "first", On: map[string]xs.Transitions{"TRIGGER": {{
				Target:  "second",
				Actions: xs.Actions{xs.Raise(xs.Ev("RAISED"))},
			}}}},
			{Key: "second", On: map[string]xs.Transitions{"RAISED": {{Target: "third"}}}},
			{Key: "third"},
		},
	})

	actorScope := xs.CreateInertActorScope(machine)
	states := machine.Microstep(
		machine.ResolveState(xs.ResolveStateConfig[any]{Value: "first"}),
		xs.Ev("TRIGGER"),
		actorScope,
	)

	assert.Equal(t, []any{"second", "third"}, microstep1Values(states))
}

// JS: machine.microstep() > should return a single-item array for normal transitions
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/microstep.test.ts#L105
func TestMicrostep_MachineMicrostep_ShouldReturnASingleItemArrayForNormalTransitions(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "first",
		States: xs.States{
			{Key: "first", On: map[string]xs.Transitions{"TRIGGER": {{Target: "second"}}}},
			{Key: "second"},
		},
	})

	actorScope := xs.CreateInertActorScope(machine)
	states := machine.Microstep(
		machine.GetInitialSnapshot(actorScope, nil),
		xs.Ev("TRIGGER"),
		actorScope,
	)

	assert.Equal(t, []any{"second"}, microstep1Values(states))
}

// JS: machine.microstep() > each state should preserve their internal queue
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/microstep.test.ts#L128
func TestMicrostep_MachineMicrostep_EachStateShouldPreserveTheirInternalQueue(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "first",
		States: xs.States{
			{Key: "first", On: map[string]xs.Transitions{"TRIGGER": {{
				Target:  "second",
				Actions: xs.Actions{xs.Raise(xs.Ev("FOO")), xs.Raise(xs.Ev("BAR"))},
			}}}},
			{Key: "second", On: map[string]xs.Transitions{"FOO": {{Target: "third"}}}},
			{Key: "third", On: map[string]xs.Transitions{"BAR": {{Target: "fourth"}}}},
			{Key: "fourth", Always: xs.Transitions{{Target: "fifth"}}},
			{Key: "fifth"},
		},
	})

	actorScope := xs.CreateInertActorScope(machine)
	states := machine.Microstep(
		machine.GetInitialSnapshot(actorScope, nil),
		xs.Ev("TRIGGER"),
		actorScope,
	)

	assert.Equal(t, []any{"second", "third", "fourth", "fifth"}, microstep1Values(states))
}

// JS: getMicrosteps > should return microsteps with actions
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/microstep.test.ts#L178
func TestMicrostep_GetMicrosteps_ShouldReturnMicrostepsWithActions(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{"GO": {{
				Target:  "b",
				Actions: xs.Actions{microstep1Noop()},
			}}}},
			{
				Key:   "b",
				Entry: xs.Actions{microstep1Noop()},
				Always: xs.Transitions{{
					Target:  "c",
					Actions: xs.Actions{microstep1Noop()},
				}},
			},
			{Key: "c"},
		},
	})

	actorScope := xs.CreateInertActorScope(machine)
	initialSnapshot := machine.GetInitialSnapshot(actorScope, nil)

	microsteps := xs.GetMicrosteps(machine, initialSnapshot, xs.Ev("GO"))

	require.Len(t, microsteps, 2)

	// First microstep: a -> b
	assert.Equal(t, "b", microsteps[0].Snapshot.Value)
	assert.Len(t, microsteps[0].Actions, 2) // transition action + entry action

	// Second microstep: b -> c (always)
	assert.Equal(t, "c", microsteps[1].Snapshot.Value)
	assert.Len(t, microsteps[1].Actions, 1) // always transition action
}

// JS: getMicrosteps > should capture actions from raised events
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/microstep.test.ts#L217
func TestMicrostep_GetMicrosteps_ShouldCaptureActionsFromRaisedEvents(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{"GO": {{
				Target:  "b",
				Actions: xs.Actions{microstep1Noop(), xs.Raise(xs.Ev("NEXT"))},
			}}}},
			{Key: "b", On: map[string]xs.Transitions{"NEXT": {{
				Target:  "c",
				Actions: xs.Actions{microstep1Noop()},
			}}}},
			{Key: "c"},
		},
	})

	actorScope := xs.CreateInertActorScope(machine)
	initialSnapshot := machine.GetInitialSnapshot(actorScope, nil)

	microsteps := xs.GetMicrosteps(machine, initialSnapshot, xs.Ev("GO"))

	require.Len(t, microsteps, 2)
	assert.Equal(t, "b", microsteps[0].Snapshot.Value)
	assert.Equal(t, "c", microsteps[1].Snapshot.Value)
}

// JS: getInitialMicrosteps > should return initial microsteps with entry actions
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/microstep.test.ts#L253
func TestMicrostep_GetInitialMicrosteps_ShouldReturnInitialMicrostepsWithEntryActions(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", Entry: xs.Actions{microstep1Noop()}},
		},
	})

	microsteps := xs.GetInitialMicrosteps(machine)

	require.Len(t, microsteps, 1)
	assert.Equal(t, "a", microsteps[0].Snapshot.Value)
	assert.Len(t, microsteps[0].Actions, 1) // entry action
}

// JS: getInitialMicrosteps > should capture actions from initial always transitions
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/microstep.test.ts#L270
func TestMicrostep_GetInitialMicrosteps_ShouldCaptureActionsFromInitialAlwaysTransitions(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{
				Key:   "a",
				Entry: xs.Actions{microstep1Noop()},
				Always: xs.Transitions{{
					Target:  "b",
					Actions: xs.Actions{microstep1Noop()},
				}},
			},
			{Key: "b", Entry: xs.Actions{microstep1Noop()}},
		},
	})

	microsteps := xs.GetInitialMicrosteps(machine)

	require.Len(t, microsteps, 2)
	assert.Equal(t, "a", microsteps[0].Snapshot.Value)
	assert.Len(t, microsteps[0].Actions, 1) // entry action for 'a'
	assert.Equal(t, "b", microsteps[1].Snapshot.Value)
	assert.Len(t, microsteps[1].Actions, 2) // always action + entry action for 'b'
}

// JS: getInitialMicrosteps > should work with nested initial states
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/microstep.test.ts#L296
func TestMicrostep_GetInitialMicrosteps_ShouldWorkWithNestedInitialStates(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "parent",
		States: xs.States{
			{
				Key:     "parent",
				Entry:   xs.Actions{microstep1Noop()},
				Initial: "child",
				States: xs.States{
					{Key: "child", Entry: xs.Actions{microstep1Noop()}},
				},
			},
		},
	})

	microsteps := xs.GetInitialMicrosteps(machine)

	require.Len(t, microsteps, 1)
	assert.Equal(t, map[string]any{"parent": "child"}, microsteps[0].Snapshot.Value)
	assert.Len(t, microsteps[0].Actions, 2) // parent entry + child entry
}

// JS: getInitialMicrosteps > should pass input to context function
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/microstep.test.ts#L319
func TestMicrostep_GetInitialMicrosteps_ShouldPassInputToContextFunction(t *testing.T) {
	type input struct{ Value int }
	type ctx struct{ Count int }

	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		ContextFn: func(a xs.ContextArgs) ctx {
			return ctx{Count: a.Input.(input).Value}
		},
		Initial: "a",
		States: xs.States{
			{Key: "a"},
		},
	})

	microsteps := xs.GetInitialMicrosteps(machine, input{Value: 42})

	require.NotEmpty(t, microsteps)
	assert.Equal(t, ctx{Count: 42}, microsteps[0].Snapshot.Context)
}
