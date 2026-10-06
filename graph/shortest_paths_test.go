package graph_test

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nguyenvanduocit/go-xstate/graph"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

type shortestPathsCtx struct {
	Count int `json:"count"`
}

type shortestPathsSnap = *xs.MachineSnapshot[shortestPathsCtx]

// JS: getShortestPaths > finds the shortest paths to a state without continuing traversal from that state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/shortestPaths.test.ts#L6
func TestShortestPaths_FindsTheShortestPathsToAStateWithoutContinuingTraversalFromThatState(t *testing.T) {
	m := xs.CreateMachine(xs.MachineConfig[shortestPathsCtx]{
		Initial: "a",
		Context: shortestPathsCtx{Count: 0},
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{"NEXT": {{Target: "b"}}}},
			{Key: "b", On: map[string]xs.Transitions{"NEXT": {{Target: "c"}}}},
			{Key: "c", On: map[string]xs.Transitions{"NEXT": {{Target: "d"}}}},
			{
				Key: "d",
				// If we reach this state, this will cause an infinite loop
				// if the stop condition does not stop the algorithm
				On: map[string]xs.Transitions{
					"NEXT": {{
						Target: "d",
						Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[shortestPathsCtx]) shortestPathsCtx {
							return shortestPathsCtx{Count: a.Context.Count + 1}
						})},
					}},
				},
			},
		},
	})

	p := graph.GetShortestPaths(m, graph.TraversalOptions[shortestPathsSnap]{
		ToState: func(state shortestPathsSnap) bool { return state.Matches("c") },
	})

	require.Len(t, p, 1)
	assert.True(t, p[0].State.Matches("c"))
}

// JS: getShortestPaths > finds the shortest paths from a state to another state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/shortestPaths.test.ts#L50
func TestShortestPaths_FindsTheShortestPathsFromAStateToAnotherState(t *testing.T) {
	m := xs.CreateMachine(xs.MachineConfig[shortestPathsCtx]{
		Initial: "a",
		Context: shortestPathsCtx{Count: 0},
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{
				"TO_Y": {{Target: "y"}},
				"TO_B": {{Target: "b"}},
			}},
			{Key: "b", On: map[string]xs.Transitions{"NEXT_B_TO_X": {{Target: "x"}}}},
			{Key: "x", On: map[string]xs.Transitions{"NEXT_X_TO_Y": {{Target: "y"}}}},
			{Key: "y"},
		},
	})

	pathsToB := graph.GetShortestPaths(m, graph.TraversalOptions[shortestPathsSnap]{
		ToState: func(state shortestPathsSnap) bool { return state.Matches("b") },
	})
	var paths []graph.StatePath[shortestPathsSnap]
	for _, path := range pathsToB {
		pathsToY := graph.GetShortestPaths(m, graph.TraversalOptions[shortestPathsSnap]{
			FromState: path.State,
			ToState:   func(state shortestPathsSnap) bool { return state.Matches("y") },
		})
		for _, pathToY := range pathsToY {
			paths = append(paths, graph.JoinPaths(path, pathToY))
		}
	}

	require.Len(t, paths, 1)
	assert.Equal(t, []string{
		"xstate.init",
		"TO_B",
		"NEXT_B_TO_X",
		"NEXT_X_TO_Y",
	}, graphEventTypes(paths[0].Steps))
}

// JS: getShortestPaths > handles event cases
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/shortestPaths.test.ts#L103
func TestShortestPaths_HandlesEventCases(t *testing.T) {
	type ctx struct {
		Todos []string `json:"todos"`
	}
	type snap = *xs.MachineSnapshot[ctx]
	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Context: ctx{Todos: []string{}},
		On: map[string]xs.Transitions{
			"todo.add": {{Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
				todo, _ := a.Event.(xs.E)["todo"].(string)
				// context.todos.concat(event.todo): never mutate the previous slice
				return ctx{Todos: append(slices.Clone(a.Context.Todos), todo)}
			})}}},
		},
	})

	shortestPaths := graph.GetShortestPaths(machine, graph.TraversalOptions[snap]{
		Events: []xs.Event{
			xs.E{"type": "todo.add", "todo": "one"},
			xs.E{"type": "todo.add", "todo": "two"},
		},
		StopWhen: func(state snap) bool { return len(state.Context.Todos) >= 3 },
	})

	var pathWithTwoTodos []graph.StatePath[snap]
	for _, path := range shortestPaths {
		if slices.Contains(path.State.Context.Todos, "one") && slices.Contains(path.State.Context.Todos, "two") {
			pathWithTwoTodos = append(pathWithTwoTodos, path)
		}
	}

	// expect(array.filter(...)).toBeDefined(): JS always passes (a filtered
	// array is defined); the reference run yields 8 matching paths, so a nil
	// (empty) result here means the traversal diverged.
	assert.NotNil(t, pathWithTwoTodos)
}

// JS: getShortestPaths > should work for machines with delays
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/shortestPaths.test.ts#L146
func TestShortestPaths_ShouldWorkForMachinesWithDelays(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", After: map[string]xs.Transitions{"1000": {{Target: "b"}}}},
			{Key: "b"},
		},
	})

	shortestPaths := graph.GetShortestPaths(machine)

	var got [][]string
	for _, p := range shortestPaths {
		got = append(got, graphEventTypes(p.Steps))
	}
	assert.Equal(t, [][]string{
		{"xstate.init"},
		{"xstate.init", "xstate.after.1000.(machine).a"},
	}, got)
}
