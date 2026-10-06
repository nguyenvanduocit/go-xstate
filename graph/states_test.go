package graph_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nguyenvanduocit/go-xstate/graph"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// JS: states > should test states by key
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/states.test.ts#L6
func TestStates_ShouldTestStatesByKey(t *testing.T) {
	var testedStateValues []xs.StateValue
	push := func(state graphAnySnap) error {
		testedStateValues = append(testedStateValues, state.Value)
		return nil
	}

	testModel := graph.CreateTestModel(xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{"EVENT": {{Target: "b"}}}},
			{Key: "b", Initial: "b1", States: xs.States{
				{Key: "b1", On: map[string]xs.Transitions{"NEXT": {{Target: "b2"}}}},
				{Key: "b2"},
			}},
		},
	}))

	graphTestModel(t, testModel, graph.TestParam[graphAnySnap]{
		States: map[string]func(graphAnySnap) error{
			"a":    push,
			"b":    push,
			"b.b1": push,
			"b.b2": push,
		},
	})

	assert.Equal(t, []xs.StateValue{
		"a",
		map[string]any{"b": "b1"},
		map[string]any{"b": "b1"},
		map[string]any{"b": "b2"},
		map[string]any{"b": "b2"},
	}, testedStateValues)
}

// JS: states > should test states by ID
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/states.test.ts#L63
func TestStates_ShouldTestStatesByID(t *testing.T) {
	var testedStateValues []xs.StateValue
	push := func(state graphAnySnap) error {
		testedStateValues = append(testedStateValues, state.Value)
		return nil
	}

	testModel := graph.CreateTestModel(xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", ID: "state_a", On: map[string]xs.Transitions{"EVENT": {{Target: "b"}}}},
			{Key: "b", ID: "state_b", Initial: "b1", States: xs.States{
				{Key: "b1", ID: "state_b1", On: map[string]xs.Transitions{"NEXT": {{Target: "b2"}}}},
				{Key: "b2", ID: "state_b2"},
			}},
		},
	}))

	graphTestModel(t, testModel, graph.TestParam[graphAnySnap]{
		States: map[string]func(graphAnySnap) error{
			"#state_a":  push,
			"#state_b":  push,
			"#state_b1": push,
			"#state_b2": push,
		},
	})

	assert.Equal(t, []xs.StateValue{
		"a",
		map[string]any{"b": "b1"},
		map[string]any{"b": "b1"},
		map[string]any{"b": "b2"},
		map[string]any{"b": "b2"},
	}, testedStateValues)
}
