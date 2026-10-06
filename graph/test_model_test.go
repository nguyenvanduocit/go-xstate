package graph_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nguyenvanduocit/go-xstate/graph"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

type testModelSnap = *xs.TransitionSnapshot[int]

func testModelCollatz() *xs.TransitionLogic[int] {
	return xs.FromTransition(func(value int, event xs.Event, _ *xs.ActorScope) int {
		if event.EventType() == "even" {
			return value / 2
		}
		return value*3 + 1
	}, func(xs.TransitionInitArgs) int { return 15 })
}

func testModelParityEvents(state testModelSnap) []xs.Event {
	if state.Context%2 == 0 {
		return []xs.Event{xs.Ev("even")}
	}
	return []xs.Event{xs.Ev("odd")}
}

// JS: custom test models > tests any logic
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/testModel.test.ts#L6
func TestTestModel_TestsAnyLogic(t *testing.T) {
	model := graph.NewTestModel(testModelCollatz(), graph.TestModelOptions[testModelSnap]{
		TraversalOptions: graph.TraversalOptions[testModelSnap]{EventsFn: testModelParityEvents},
	})

	paths := model.GetShortestPaths(graph.GetPathOptions[testModelSnap]{
		TraversalOptions: graph.TraversalOptions[testModelSnap]{
			ToState: func(state testModelSnap) bool { return state.Context == 1 },
		},
	})

	assert.Greater(t, len(paths), 0)
}

// JS: custom test models > tests states for any logic
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/testModel.test.ts#L31
func TestTestModel_TestsStatesForAnyLogic(t *testing.T) {
	var testedStateKeys []string

	model := graph.NewTestModel(testModelCollatz(), graph.TestModelOptions[testModelSnap]{
		TraversalOptions: graph.TraversalOptions[testModelSnap]{EventsFn: testModelParityEvents},
		StateMatcher: func(state testModelSnap, key string) bool {
			if key == "even" {
				return state.Context%2 == 0
			}
			if key == "odd" {
				return state.Context%2 == 1
			}
			return false
		},
	})

	paths := model.GetShortestPaths(graph.GetPathOptions[testModelSnap]{
		TraversalOptions: graph.TraversalOptions[testModelSnap]{
			ToState: func(state testModelSnap) bool { return state.Context == 1 },
		},
	})

	graphTestPaths(t, paths, graph.TestParam[testModelSnap]{
		States: map[string]func(testModelSnap) error{
			"even": func(state testModelSnap) error {
				testedStateKeys = append(testedStateKeys, "even")
				assert.Equal(t, 0, state.Context%2)
				return nil
			},
			"odd": func(state testModelSnap) error {
				testedStateKeys = append(testedStateKeys, "odd")
				assert.Equal(t, 1, state.Context%2)
				return nil
			},
		},
	})

	assert.Contains(t, testedStateKeys, "even")
	assert.Contains(t, testedStateKeys, "odd")
}
