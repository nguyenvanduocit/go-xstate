package xstate_test

import (
	"testing"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/assert"
)

// JS: matchesState() > should return true if two states are equivalent
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/match.test.ts#L4
func TestMatch_MatchesState_ShouldReturnTrueIfTwoStatesAreEquivalent(t *testing.T) {
	assert.True(t, xs.MatchesState("a", "a"))

	assert.True(t, xs.MatchesState("b.b1", "b.b1"))

	assert.False(t, xs.MatchesState("B.bar", map[string]any{"A": "foo"}))
}

// JS: matchesState() > should return true if two state values are equivalent
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/match.test.ts#L12
func TestMatch_MatchesState_ShouldReturnTrueIfTwoStateValuesAreEquivalent(t *testing.T) {
	assert.True(t, xs.MatchesState(map[string]any{"a": "b"}, map[string]any{"a": "b"}))
	assert.True(t, xs.MatchesState(
		map[string]any{"a": map[string]any{"b": "c"}},
		map[string]any{"a": map[string]any{"b": "c"}},
	))
}

// JS: matchesState() > should return true if two parallel states are equivalent
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/match.test.ts#L17
func TestMatch_MatchesState_ShouldReturnTrueIfTwoParallelStatesAreEquivalent(t *testing.T) {
	assert.True(t, xs.MatchesState(
		map[string]any{"a": map[string]any{"b1": "foo", "b2": "bar"}},
		map[string]any{"a": map[string]any{"b1": "foo", "b2": "bar"}},
	))

	assert.True(t, xs.MatchesState(
		map[string]any{
			"a": map[string]any{"b1": "foo", "b2": "bar"},
			"b": map[string]any{"b3": "baz", "b4": "quo"},
		},
		map[string]any{
			"a": map[string]any{"b1": "foo", "b2": "bar"},
			"b": map[string]any{"b3": "baz", "b4": "quo"},
		},
	))

	assert.True(t, xs.MatchesState(
		map[string]any{"a": "foo", "b": "bar"},
		map[string]any{"a": "foo", "b": "bar"},
	))
}

// JS: matchesState() > should return true if a state is a substate of a superstate
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/match.test.ts#L37
func TestMatch_MatchesState_ShouldReturnTrueIfAStateIsASubstateOfASuperstate(t *testing.T) {
	assert.True(t, xs.MatchesState("b", "b.b1"))

	assert.True(t, xs.MatchesState("foo.bar", "foo.bar.baz.quo"))
}

// JS: matchesState() > should return true if a state value is a substate of a superstate value
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/match.test.ts#L43
func TestMatch_MatchesState_ShouldReturnTrueIfAStateValueIsASubstateOfASuperstateValue(t *testing.T) {
	assert.True(t, xs.MatchesState("b", map[string]any{"b": "b1"}))

	assert.True(t, xs.MatchesState(
		map[string]any{"foo": "bar"},
		map[string]any{"foo": map[string]any{"bar": map[string]any{"baz": "quo"}}},
	))
}

// JS: matchesState() > should return true if a parallel state value is a substate of a superstate value
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/match.test.ts#L51
func TestMatch_MatchesState_ShouldReturnTrueIfAParallelStateValueIsASubstateOfASuperstateValue(t *testing.T) {
	assert.True(t, xs.MatchesState("b", map[string]any{"b": "b1", "c": "c1"}))

	assert.True(t, xs.MatchesState(
		map[string]any{"foo": "bar", "fooAgain": "barAgain"},
		map[string]any{
			"foo":      map[string]any{"bar": map[string]any{"baz": "quo"}},
			"fooAgain": map[string]any{"barAgain": "baz"},
		},
	))
}

// JS: matchesState() > should return false if two states are not equivalent
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/match.test.ts#L62
func TestMatch_MatchesState_ShouldReturnFalseIfTwoStatesAreNotEquivalent(t *testing.T) {
	assert.True(t, !xs.MatchesState("a", "b"))

	assert.True(t, !xs.MatchesState("a.a1", "b.b1"))
}

// JS: matchesState() > should return false if parent state is more specific than child state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/match.test.ts#L68
func TestMatch_MatchesState_ShouldReturnFalseIfParentStateIsMoreSpecificThanChildState(t *testing.T) {
	assert.True(t, !xs.MatchesState("a.b.c", "a.b"))

	assert.True(t, !xs.MatchesState(
		map[string]any{"a": map[string]any{"b": map[string]any{"c": "d"}}},
		map[string]any{"a": "b"},
	))
}

// JS: matchesState() > should return false if two state values are not equivalent
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/match.test.ts#L74
func TestMatch_MatchesState_ShouldReturnFalseIfTwoStateValuesAreNotEquivalent(t *testing.T) {
	assert.True(t, !xs.MatchesState(map[string]any{"a": "a1"}, map[string]any{"b": "b1"}))
}

// JS: matchesState() > should return false if a state is not a substate of a superstate
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/match.test.ts#L78
func TestMatch_MatchesState_ShouldReturnFalseIfAStateIsNotASubstateOfASuperstate(t *testing.T) {
	assert.True(t, !xs.MatchesState("a", "b.b1"))

	assert.True(t, !xs.MatchesState("foo.false.baz", "foo.bar.baz.quo"))
}

// JS: matchesState() > should return false if a state value is not a substate of a superstate value
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/match.test.ts#L84
func TestMatch_MatchesState_ShouldReturnFalseIfAStateValueIsNotASubstateOfASuperstateValue(t *testing.T) {
	assert.True(t, !xs.MatchesState("a", map[string]any{"b": "b1"}))

	assert.True(t, !xs.MatchesState(
		map[string]any{"foo": map[string]any{"false": "baz"}},
		map[string]any{"foo": map[string]any{"bar": map[string]any{"baz": "quo"}}},
	))
}

// JS: matchesState() > should mix/match string state values and object state values
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/match.test.ts#L92
func TestMatch_MatchesState_ShouldMixMatchStringStateValuesAndObjectStateValues(t *testing.T) {
	assert.True(t, xs.MatchesState("a.b.c", map[string]any{"a": map[string]any{"b": "c"}}))
}

// JS: matches() method > should execute matchesState on a State given the parent state value
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/match.test.ts#L98
func TestMatch_MatchesMethod_ShouldExecuteMatchesStateOnAStateGivenTheParentStateValue(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "foo",
		States: xs.States{
			{
				Key:     "foo",
				Initial: "bar",
				States: xs.States{
					{
						Key:     "bar",
						Initial: "baz",
						States: xs.States{
							{Key: "baz"},
						},
					},
				},
			},
		},
	})

	initialState := xs.CreateActor(machine).GetSnapshot()

	assert.True(t, initialState.Matches("foo"))
	assert.True(t, initialState.Matches(map[string]any{"foo": "bar"}))
	assert.False(t, initialState.Matches("fake"))
}
