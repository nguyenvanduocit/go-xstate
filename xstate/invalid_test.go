package xstate_test

import (
	"regexp"
	"testing"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/assert"
)

// invalid1ParallelMachine mirrors the parallel machine (A: A1/A2, B: B1/B2)
// declared identically inside each test of describe('invalid or resolved states').
func invalid1ParallelMachine() *xs.StateMachine[any] {
	return xs.CreateMachine(xs.MachineConfig[any]{
		Type: xs.Parallel,
		States: xs.States{
			{
				Key:     "A",
				Initial: "A1",
				States: xs.States{
					{Key: "A1"},
					{Key: "A2"},
				},
			},
			{
				Key:     "B",
				Initial: "B1",
				States: xs.States{
					{Key: "B1"},
					{Key: "B2"},
				},
			},
		},
	})
}

// JS: invalid or resolved states > should resolve a String state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invalid.test.ts#L4
func TestInvalid_ResolvedStates_ShouldResolveAStringState(t *testing.T) {
	machine := invalid1ParallelMachine()

	assert.Equal(t,
		map[string]any{"A": "A1", "B": "B1"},
		xs.GetNextSnapshot(machine, machine.ResolveState(xs.ResolveStateConfig[any]{Value: "A"}), xs.Ev("E")).Value,
	)
}

// JS: invalid or resolved states > should resolve transitions from empty states
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invalid.test.ts#L34
func TestInvalid_ResolvedStates_ShouldResolveTransitionsFromEmptyStates(t *testing.T) {
	machine := invalid1ParallelMachine()

	assert.Equal(t,
		map[string]any{"A": "A1", "B": "B1"},
		xs.GetNextSnapshot(
			machine,
			machine.ResolveState(xs.ResolveStateConfig[any]{Value: map[string]any{"A": map[string]any{}, "B": map[string]any{}}}),
			xs.Ev("E"),
		).Value,
	)
}

// JS: invalid or resolved states > should allow transitioning from valid states
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invalid.test.ts#L66
func TestInvalid_ResolvedStates_ShouldAllowTransitioningFromValidStates(t *testing.T) {
	machine := invalid1ParallelMachine()

	// The JS test has no explicit expectation: it passes when nothing throws.
	assert.NotPanics(t, func() {
		xs.GetNextSnapshot(
			machine,
			machine.ResolveState(xs.ResolveStateConfig[any]{Value: map[string]any{"A": "A1", "B": "B1"}}),
			xs.Ev("E"),
		)
	})
}

// JS: invalid or resolved states > should reject transitioning from bad state configs
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invalid.test.ts#L93
func TestInvalid_ResolvedStates_ShouldRejectTransitioningFromBadStateConfigs(t *testing.T) {
	machine := invalid1ParallelMachine()

	assert.Panics(t, func() {
		xs.GetNextSnapshot(
			machine,
			machine.ResolveState(xs.ResolveStateConfig[any]{Value: map[string]any{"A": "A3", "B": "B3"}}),
			xs.Ev("E"),
		)
	})
}

// JS: invalid or resolved states > should resolve transitioning from partially valid states
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invalid.test.ts#L122
func TestInvalid_ResolvedStates_ShouldResolveTransitioningFromPartiallyValidStates(t *testing.T) {
	machine := invalid1ParallelMachine()

	assert.Equal(t,
		map[string]any{"A": "A1", "B": "B1"},
		xs.GetNextSnapshot(
			machine,
			machine.ResolveState(xs.ResolveStateConfig[any]{Value: map[string]any{"A": "A1", "B": map[string]any{}}}),
			xs.Ev("E"),
		).Value,
	)
}

// JS: invalid transition > should throw when attempting to create a machine with a sibling target on the root node
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/invalid.test.ts#L156
func TestInvalid_Transition_ShouldThrowWhenCreatingMachineWithSiblingTargetOnRootNode(t *testing.T) {
	create := func() {
		xs.CreateMachine(xs.MachineConfig[any]{
			ID:      "direction",
			Initial: "left",
			States: xs.States{
				{Key: "left"},
				{Key: "right"},
			},
			On: map[string]xs.Transitions{
				"LEFT_CLICK":  {{Target: "left"}},
				"RIGHT_CLICK": {{Target: "right"}},
			},
		})
	}

	assert.Panics(t, create)
	assert.Regexp(t, regexp.MustCompile(`(?i)invalid target`), panicMessage(create))
}
