package graph_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nguyenvanduocit/go-xstate/graph"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

func eventsABMachine() *xs.StateMachine[any] {
	return xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{"EVENT": {{Target: "b"}}}},
			{Key: "b"},
		},
	})
}

// JS: events > should execute events (`exec` property)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/events.test.ts#L6
func TestEvents_ShouldExecuteEventsExecProperty(t *testing.T) {
	executed := false

	testModel := graph.CreateTestModel(eventsABMachine())

	graphTestModel(t, testModel, graph.TestParam[graphAnySnap]{
		Events: map[string]graph.EventExecutor[graphAnySnap]{
			"EVENT": func(graph.Step[graphAnySnap]) error {
				executed = true
				return nil
			},
		},
	})

	assert.True(t, executed)
}

// JS: events > should execute events (function)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/events.test.ts#L34
func TestEvents_ShouldExecuteEventsFunction(t *testing.T) {
	executed := false

	testModel := graph.CreateTestModel(eventsABMachine())

	graphTestModel(t, testModel, graph.TestParam[graphAnySnap]{
		Events: map[string]graph.EventExecutor[graphAnySnap]{
			"EVENT": func(graph.Step[graphAnySnap]) error {
				executed = true
				return nil
			},
		},
	})

	assert.True(t, executed)
}
