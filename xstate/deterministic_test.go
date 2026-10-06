package xstate_test

import (
	"testing"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/assert"
)

// deterministic1LightMachine mirrors the describe-level `lightMachine` of
// deterministic.test.ts (JS L10-48).
func deterministic1LightMachine() *xs.StateMachine[any] {
	return xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "green",
		States: xs.States{
			{Key: "green", On: map[string]xs.Transitions{
				"TIMER":        {{Target: "yellow"}},
				"POWER_OUTAGE": {{Target: "red"}},
			}},
			{Key: "yellow", On: map[string]xs.Transitions{
				"TIMER":        {{Target: "red"}},
				"POWER_OUTAGE": {{Target: "red"}},
			}},
			{
				Key: "red",
				On: map[string]xs.Transitions{
					"TIMER":        {{Target: "green"}},
					"POWER_OUTAGE": {{Target: "red"}},
				},
				Initial: "walk",
				States: xs.States{
					{Key: "walk", On: map[string]xs.Transitions{
						"PED_COUNTDOWN": {{Target: "wait"}},
						"TIMER":         nil, // forbidden event
					}},
					{Key: "wait", On: map[string]xs.Transitions{
						"PED_COUNTDOWN": {{Target: "stop"}},
						"TIMER":         nil, // forbidden event
					}},
					{Key: "stop"},
				},
			},
		},
	})
}

// deterministic1TestMachine mirrors the describe-level `testMachine` of
// deterministic.test.ts (JS L50-67).
func deterministic1TestMachine() *xs.StateMachine[any] {
	return xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{
				"T": {{Target: "b.b1"}},
				"F": {{Target: "c"}},
			}},
			{Key: "b", Initial: "b1", States: xs.States{{Key: "b1"}}},
			{Key: "c"},
		},
	})
}

// JS: deterministic machine > machine transitions > should properly transition states based on event-like object
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/deterministic.test.ts#L70
func TestDeterministic_MachineTransitions_ShouldProperlyTransitionStatesBasedOnEventLikeObject(t *testing.T) {
	lightMachine := deterministic1LightMachine()

	next, _ := xs.Transition(
		lightMachine,
		lightMachine.ResolveState(xs.ResolveStateConfig[any]{Value: "green"}),
		xs.Ev("TIMER"),
	)
	assert.Equal(t, "yellow", next.Value)
}

// JS: deterministic machine > machine transitions > should not transition states for illegal transitions
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/deterministic.test.ts#L82
func TestDeterministic_MachineTransitions_ShouldNotTransitionStatesForIllegalTransitions(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{"NEXT": {{Target: "b"}}}},
			{Key: "b"},
		},
	})

	actor := xs.CreateActor(machine).Start()

	previousSnapshot := actor.GetSnapshot()

	actor.Send(xs.Ev("FAKE"))

	assert.Equal(t, "a", actor.GetSnapshot().Value)
	assert.Same(t, previousSnapshot, actor.GetSnapshot())
}

// JS: deterministic machine > machine transitions > should throw an error if not given an event
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/deterministic.test.ts#L105
func TestDeterministic_MachineTransitions_ShouldThrowAnErrorIfNotGivenAnEvent(t *testing.T) {
	lightMachine := deterministic1LightMachine()
	testMachine := deterministic1TestMachine()

	assert.Panics(t, func() {
		xs.Transition(
			lightMachine,
			testMachine.ResolveState(xs.ResolveStateConfig[any]{Value: "red"}),
			nil,
		)
	})
}

// JS: deterministic machine > machine transitions > should transition to nested states as target
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/deterministic.test.ts#L115
func TestDeterministic_MachineTransitions_ShouldTransitionToNestedStatesAsTarget(t *testing.T) {
	testMachine := deterministic1TestMachine()

	next, _ := xs.Transition(
		testMachine,
		testMachine.ResolveState(xs.ResolveStateConfig[any]{Value: "a"}),
		xs.Ev("T"),
	)
	assert.Equal(t, map[string]any{"b": "b1"}, next.Value)
}

// JS: deterministic machine > machine transitions > should throw an error for transitions from invalid states
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/deterministic.test.ts#L125
func TestDeterministic_MachineTransitions_ShouldThrowAnErrorForTransitionsFromInvalidStates(t *testing.T) {
	testMachine := deterministic1TestMachine()

	assert.Panics(t, func() {
		xs.Transition(
			testMachine,
			testMachine.ResolveState(xs.ResolveStateConfig[any]{Value: "fake"}),
			xs.Ev("T"),
		)
	})
}

// JS: deterministic machine > machine transitions > should throw an error for transitions from invalid substates
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/deterministic.test.ts#L133
func TestDeterministic_MachineTransitions_ShouldThrowAnErrorForTransitionsFromInvalidSubstates(t *testing.T) {
	testMachine := deterministic1TestMachine()

	assert.Panics(t, func() {
		xs.Transition(
			testMachine,
			testMachine.ResolveState(xs.ResolveStateConfig[any]{Value: "a.fake"}),
			xs.Ev("T"),
		)
	})
}

// JS: deterministic machine > machine transitions > should use the machine.initialState when an undefined state is given
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/deterministic.test.ts#L141
func TestDeterministic_MachineTransitions_ShouldUseInitialStateWhenUndefinedStateIsGiven(t *testing.T) {
	lightMachine := deterministic1LightMachine()

	init := xs.GetInitialSnapshot(lightMachine)
	next, _ := xs.Transition(lightMachine, init, xs.Ev("TIMER"))
	assert.Equal(t, "yellow", next.Value)
}

// JS: deterministic machine > machine transitions > should use the machine.initialState when an undefined state is given (unhandled event)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/deterministic.test.ts#L148
func TestDeterministic_MachineTransitions_ShouldUseInitialStateWhenUndefinedStateIsGivenUnhandledEvent(t *testing.T) {
	lightMachine := deterministic1LightMachine()

	init := xs.GetInitialSnapshot(lightMachine)
	next, _ := xs.Transition(lightMachine, init, xs.Ev("TIMER"))
	assert.Equal(t, "yellow", next.Value)
}

// JS: deterministic machine > machine transition with nested states > should properly transition a nested state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/deterministic.test.ts#L157
func TestDeterministic_NestedStates_ShouldProperlyTransitionANestedState(t *testing.T) {
	lightMachine := deterministic1LightMachine()

	next, _ := xs.Transition(
		lightMachine,
		lightMachine.ResolveState(xs.ResolveStateConfig[any]{Value: map[string]any{"red": "walk"}}),
		xs.Ev("PED_COUNTDOWN"),
	)
	assert.Equal(t, map[string]any{"red": "wait"}, next.Value)
}

// JS: deterministic machine > machine transition with nested states > should transition from initial nested states
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/deterministic.test.ts#L167
func TestDeterministic_NestedStates_ShouldTransitionFromInitialNestedStates(t *testing.T) {
	lightMachine := deterministic1LightMachine()

	next, _ := xs.Transition(
		lightMachine,
		lightMachine.ResolveState(xs.ResolveStateConfig[any]{Value: "red"}),
		xs.Ev("PED_COUNTDOWN"),
	)
	assert.Equal(t, map[string]any{"red": "wait"}, next.Value)
}

// JS: deterministic machine > machine transition with nested states > should transition from deep initial nested states
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/deterministic.test.ts#L177
func TestDeterministic_NestedStates_ShouldTransitionFromDeepInitialNestedStates(t *testing.T) {
	lightMachine := deterministic1LightMachine()

	next, _ := xs.Transition(
		lightMachine,
		lightMachine.ResolveState(xs.ResolveStateConfig[any]{Value: "red"}),
		xs.Ev("PED_COUNTDOWN"),
	)
	assert.Equal(t, map[string]any{"red": "wait"}, next.Value)
}

// JS: deterministic machine > machine transition with nested states > should bubble up events that nested states cannot handle
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/deterministic.test.ts#L187
func TestDeterministic_NestedStates_ShouldBubbleUpEventsThatNestedStatesCannotHandle(t *testing.T) {
	lightMachine := deterministic1LightMachine()

	next, _ := xs.Transition(
		lightMachine,
		lightMachine.ResolveState(xs.ResolveStateConfig[any]{Value: map[string]any{"red": "stop"}}),
		xs.Ev("TIMER"),
	)
	assert.Equal(t, "green", next.Value)
}

// JS: deterministic machine > machine transition with nested states > should not transition from illegal events
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/deterministic.test.ts#L197
func TestDeterministic_NestedStates_ShouldNotTransitionFromIllegalEvents(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{
				Key:     "a",
				Initial: "b",
				States: xs.States{
					{Key: "b", On: map[string]xs.Transitions{"NEXT": {{Target: "c"}}}},
					{Key: "c"},
				},
			},
		},
	})

	actor := xs.CreateActor(machine).Start()

	previousSnapshot := actor.GetSnapshot()

	actor.Send(xs.Ev("FAKE"))

	assert.Equal(t, map[string]any{"a": "b"}, actor.GetSnapshot().Value)
	assert.Same(t, previousSnapshot, actor.GetSnapshot())
}

// JS: deterministic machine > machine transition with nested states > should transition to the deepest initial state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/deterministic.test.ts#L225
func TestDeterministic_NestedStates_ShouldTransitionToTheDeepestInitialState(t *testing.T) {
	lightMachine := deterministic1LightMachine()

	next, _ := xs.Transition(
		lightMachine,
		lightMachine.ResolveState(xs.ResolveStateConfig[any]{Value: "yellow"}),
		xs.Ev("TIMER"),
	)
	assert.Equal(t, map[string]any{"red": "walk"}, next.Value)
}

// JS: deterministic machine > machine transition with nested states > should return the same state if no transition occurs
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/deterministic.test.ts#L239
func TestDeterministic_NestedStates_ShouldReturnTheSameStateIfNoTransitionOccurs(t *testing.T) {
	lightMachine := deterministic1LightMachine()

	init := xs.GetInitialSnapshot(lightMachine)
	initialState, _ := xs.Transition(lightMachine, init, xs.Ev("NOTHING"))
	nextState, _ := xs.Transition(lightMachine, initialState, xs.Ev("NOTHING"))

	assert.Equal(t, initialState.Value, nextState.Value)
	assert.Same(t, initialState, nextState)
}

// JS: deterministic machine > state key names > should work with substate nodes that have the same key
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/deterministic.test.ts#L275
func TestDeterministic_StateKeyNames_ShouldWorkWithSubstateNodesThatHaveTheSameKey(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "test",
		States: xs.States{
			{
				Key:    "test",
				Invoke: []xs.InvokeConfig{{Src: "activity"}},
				Entry:  xs.Actions{xs.ActionRef{Type: "onEntry"}},
				On:     map[string]xs.Transitions{"NEXT": {{Target: "test"}}},
				Exit:   xs.Actions{xs.ActionRef{Type: "onExit"}},
			},
		},
	}, xs.Implementations{
		Actors: map[string]xs.ActorLogic{
			"activity": xs.FromCallback(func(xs.CallbackArgs) func() { return func() {} }),
		},
	})

	init := xs.GetInitialSnapshot(machine)
	next, _ := xs.Transition(machine, init, xs.Ev("NEXT"))
	assert.Equal(t, "test", next.Value)
}

// JS: deterministic machine > forbidden events > undefined transitions should forbid events
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/deterministic.test.ts#L284
func TestDeterministic_ForbiddenEvents_UndefinedTransitionsShouldForbidEvents(t *testing.T) {
	lightMachine := deterministic1LightMachine()

	walkState, _ := xs.Transition(
		lightMachine,
		lightMachine.ResolveState(xs.ResolveStateConfig[any]{Value: map[string]any{"red": "walk"}}),
		xs.Ev("TIMER"),
	)

	assert.Equal(t, map[string]any{"red": "walk"}, walkState.Value)
}
