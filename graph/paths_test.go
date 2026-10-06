package graph_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nguyenvanduocit/go-xstate/graph"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

func pathsMultiPathMachine() *xs.StateMachine[any] {
	return xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{"EVENT": {{Target: "b"}}}},
			{Key: "b", On: map[string]xs.Transitions{"EVENT": {{Target: "c"}}}},
			{Key: "c", On: map[string]xs.Transitions{
				"EVENT":   {{Target: "d"}},
				"EVENT_2": {{Target: "e"}},
			}},
			{Key: "d"},
			{Key: "e"},
		},
	})
}

// JS: testModel.testPaths(...) > custom path generators can be provided
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/paths.test.ts#L35
func TestPaths_TestPaths_CustomPathGeneratorsCanBeProvided(t *testing.T) {
	testModel := graph.CreateTestModel(xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{"EVENT": {{Target: "b"}}}},
			{Key: "b"},
		},
	}))

	paths := testModel.GetPaths(func(logic xs.TypedActorLogic[graphAnySnap], options graph.TraversalOptions[graphAnySnap]) []graph.StatePath[graphAnySnap] {
		initialState := xs.GetInitialSnapshot(logic)
		events := options.Events
		if options.EventsFn != nil {
			events = options.EventsFn(initialState)
		}

		nextState := xs.GetNextSnapshot(logic, initialState, events[0])
		return []graph.StatePath[graphAnySnap]{
			{
				State: nextState,
				Steps: []graph.Step[graphAnySnap]{
					{State: initialState, Event: events[0]},
				},
				Weight: 1,
			},
		}
	})

	graphTestPaths(t, paths, graph.TestParam[graphAnySnap]{})
}

// JS: testModel.testPaths(...) > When the machine only has one path > Should only follow that path
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/paths.test.ts#L76
func TestPaths_TestPaths_WhenTheMachineOnlyHasOnePath_ShouldOnlyFollowThatPath(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{"EVENT": {{Target: "b"}}}},
			{Key: "b", On: map[string]xs.Transitions{"EVENT": {{Target: "c"}}}},
			{Key: "c"},
		},
	})

	model := graph.CreateTestModel(machine)

	paths := model.GetShortestPaths()

	assert.Len(t, paths, 1)
}

// JS: testModel.testPaths(...) > getSimplePaths > Should dedup simple path paths
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/paths.test.ts#L103
func TestPaths_TestPaths_GetSimplePaths_ShouldDedupSimplePathPaths(t *testing.T) {
	model := graph.CreateTestModel(pathsMultiPathMachine())

	paths := model.GetSimplePaths()

	assert.Len(t, paths, 2)
}

// JS: testModel.testPaths(...) > getSimplePaths > Should not dedup simple path paths if deduplicate: false
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/paths.test.ts#L111
func TestPaths_TestPaths_GetSimplePaths_ShouldNotDedupSimplePathPathsIfDeduplicateFalse(t *testing.T) {
	model := graph.CreateTestModel(pathsMultiPathMachine())

	paths := model.GetSimplePaths(graph.GetPathOptions[graphAnySnap]{
		AllowDuplicatePaths: true,
	})

	assert.Len(t, paths, 5)
}

// JS: testModel.testPaths(...) > getSimplePaths > should support filtering disabled events
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/paths.test.ts#L121
func TestPaths_TestPaths_GetSimplePaths_ShouldSupportFilteringDisabledEvents(t *testing.T) {
	type ctx struct {
		Allowed bool `json:"allowed"`
	}
	type snap = *xs.MachineSnapshot[ctx]
	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		ID:      "guarded-test-model",
		Initial: "start",
		Context: ctx{Allowed: false},
		States: xs.States{
			{Key: "start", On: map[string]xs.Transitions{"NEXT": {{Target: "idle"}}}},
			{Key: "idle", On: map[string]xs.Transitions{
				"PROCEED": {{
					Target: "done",
					Guard:  xs.GuardFunc(func(a xs.GuardArgs[ctx]) bool { return a.Context.Allowed }),
				}},
				"ALLOW": {{Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
					return ctx{Allowed: true}
				})}}},
			}},
			{Key: "done", Type: xs.Final},
		},
	})

	model := graph.CreateTestModel(machine)

	paths := model.GetSimplePaths(graph.GetPathOptions[snap]{
		TraversalOptions: graph.TraversalOptions[snap]{
			FilterEvents: func(state snap, event xs.Event) bool { return state.Can(event) },
			ToState:      func(state snap) bool { return state.Status == xs.StatusDone },
		},
	})

	assert.Equal(t, []string{
		`Reaches state "done"({"allowed":true}): xstate.init → NEXT → ALLOW → PROCEED`,
	}, graphDescriptions(paths))
}

// JS: path.description > Should write a readable description including the target state and the path
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/paths.test.ts#L164
func TestPaths_PathDescription_ShouldWriteAReadableDescriptionIncludingTheTargetStateAndThePath(t *testing.T) {
	model := graph.CreateTestModel(pathsMultiPathMachine())

	paths := model.GetShortestPaths()

	assert.Equal(t, []string{
		`Reaches state "d": xstate.init → EVENT → EVENT → EVENT`,
		`Reaches state "e": xstate.init → EVENT → EVENT → EVENT_2`,
	}, graphDescriptions(paths))
}

// JS: transition coverage > path generation should cover all transitions by default
//
// JS explores a's events as NEXT, END (`on` insertion order) and yields
// "NEXT → PREV", "NEXT → RESTART", "END". The Go API returns own events
// sorted (END, NEXT); expectation re-derived accordingly (see manifest).
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/paths.test.ts#L177
func TestPaths_TransitionCoverage_PathGenerationShouldCoverAllTransitionsByDefault(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{
				"NEXT": {{Target: "b"}},
				"END":  {{Target: "b"}},
			}},
			{Key: "b", On: map[string]xs.Transitions{
				"PREV":    {{Target: "a"}},
				"RESTART": {{Target: "a"}},
			}},
		},
	})

	model := graph.CreateTestModel(machine)

	paths := model.GetShortestPaths()

	assert.Equal(t, []string{
		`Reaches state "a": xstate.init → END → PREV`,
		`Reaches state "a": xstate.init → END → RESTART`,
		`Reaches state "b": xstate.init → NEXT`,
	}, graphDescriptions(paths))
}

// JS: transition coverage > transition coverage should consider guarded transitions
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/paths.test.ts#L209
func TestPaths_TransitionCoverage_TransitionCoverageShouldConsiderGuardedTransitions(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{
				"NEXT": {{Guard: xs.GuardRef{Type: "valid"}, Target: "b"}, {Target: "b"}},
			}},
			{Key: "b"},
		},
	}, xs.Implementations{
		Guards: map[string]xs.Guard{
			"valid": xs.GuardFunc(func(a xs.GuardArgs[any]) bool {
				value, _ := a.Event.(xs.E)["value"].(int)
				return value > 10
			}),
		},
	})

	model := graph.CreateTestModel(machine)

	paths := model.GetShortestPaths(graph.GetPathOptions[graphAnySnap]{
		TraversalOptions: graph.TraversalOptions[graphAnySnap]{
			Events: []xs.Event{
				xs.E{"type": "NEXT", "value": 0},
				xs.E{"type": "NEXT", "value": 100},
				xs.E{"type": "NEXT", "value": 1000},
			},
		},
	})

	// { value: 1000 } already covered by first guarded transition
	assert.Equal(t, []string{
		`Reaches state "b": xstate.init → NEXT ({"value":0}) → NEXT ({"value":0})`,
		`Reaches state "b": xstate.init → NEXT ({"value":100})`,
		`Reaches state "b": xstate.init → NEXT ({"value":1000})`,
	}, graphDescriptions(paths))
}

// JS: transition coverage > transition coverage should consider multiple transitions with the same target
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/paths.test.ts#L251
func TestPaths_TransitionCoverage_TransitionCoverageShouldConsiderMultipleTransitionsWithTheSameTarget(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{
				"GO_TO_B": {{Target: "b"}},
				"GO_TO_C": {{Target: "c"}},
			}},
			{Key: "b", On: map[string]xs.Transitions{"GO_TO_A": {{Target: "a"}}}},
			{Key: "c", On: map[string]xs.Transitions{"GO_TO_A": {{Target: "a"}}}},
		},
	})

	model := graph.CreateTestModel(machine)

	paths := model.GetShortestPaths()

	assert.Equal(t, []string{
		`Reaches state "a": xstate.init → GO_TO_B → GO_TO_A`,
		`Reaches state "a": xstate.init → GO_TO_C → GO_TO_A`,
	}, graphDescriptions(paths))
}

func pathsOpenClosedMachine() *xs.StateMachine[any] {
	return xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "open",
		States: xs.States{
			{Key: "open", On: map[string]xs.Transitions{"CLOSE": {{Target: "closed"}}}},
			{Key: "closed", On: map[string]xs.Transitions{"OPEN": {{Target: "open"}}}},
		},
	})
}

// JS: getShortestPathsTo > Should find a path to a non-initial target state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/paths.test.ts#L301
func TestPaths_GetShortestPathsTo_ShouldFindAPathToANonInitialTargetState(t *testing.T) {
	closedPaths := graph.CreateTestModel(pathsOpenClosedMachine()).GetShortestPaths(graph.GetPathOptions[graphAnySnap]{
		TraversalOptions: graph.TraversalOptions[graphAnySnap]{
			ToState: func(state graphAnySnap) bool { return state.Matches("closed") },
		},
	})

	assert.Len(t, closedPaths, 1)
}

// JS: getShortestPathsTo > Should find a path to an initial target state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/paths.test.ts#L309
func TestPaths_GetShortestPathsTo_ShouldFindAPathToAnInitialTargetState(t *testing.T) {
	openPaths := graph.CreateTestModel(pathsOpenClosedMachine()).GetShortestPaths(graph.GetPathOptions[graphAnySnap]{
		TraversalOptions: graph.TraversalOptions[graphAnySnap]{
			ToState: func(state graphAnySnap) bool { return state.Matches("open") },
		},
	})

	assert.Len(t, openPaths, 1)
}

func pathsFromMachine() *xs.StateMachine[any] {
	return xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{
				"NEXT":  {{Target: "b"}},
				"OTHER": {{Target: "b"}},
				"TO_C":  {{Target: "c"}},
				"TO_D":  {{Target: "d"}},
				"TO_E":  {{Target: "e"}},
			}},
			{Key: "b", On: map[string]xs.Transitions{
				"TO_C": {{Target: "c"}},
				"TO_D": {{Target: "d"}},
			}},
			{Key: "c"},
			{Key: "d"},
			{Key: "e"},
		},
	})
}

// JS: getShortestPathsFrom > should get shortest paths from array of paths
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/paths.test.ts#L319
func TestPaths_GetShortestPathsFrom_ShouldGetShortestPathsFromArrayOfPaths(t *testing.T) {
	model := graph.CreateTestModel(pathsFromMachine())
	pathsToB := model.GetShortestPaths(graph.GetPathOptions[graphAnySnap]{
		TraversalOptions: graph.TraversalOptions[graphAnySnap]{
			ToState: func(state graphAnySnap) bool { return state.Matches("b") },
		},
	})

	// a (NEXT) -> b
	// a (OTHER) -> b
	require.Len(t, pathsToB, 2)

	shortestPaths := model.GetShortestPathsFrom(pathsToB)

	// a (NEXT) -> b (TO_C) -> c
	// a (OTHER) -> b (TO_C) -> c
	// a (NEXT) -> b (TO_D) -> d
	// a (OTHER) -> b (TO_D) -> d
	assert.Len(t, shortestPaths, 4)

	for _, path := range shortestPaths {
		assert.Len(t, path.Steps, 3)
	}
}

// JS: getShortestPathsFrom > getSimplePathsFrom > should get simple paths from array of paths
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/paths.test.ts#L358
func TestPaths_GetShortestPathsFrom_GetSimplePathsFrom_ShouldGetSimplePathsFromArrayOfPaths(t *testing.T) {
	model := graph.CreateTestModel(pathsFromMachine())
	pathsToB := model.GetSimplePaths(graph.GetPathOptions[graphAnySnap]{
		TraversalOptions: graph.TraversalOptions[graphAnySnap]{
			ToState: func(state graphAnySnap) bool { return state.Matches("b") },
		},
	})

	// a (NEXT) -> b
	// a (OTHER) -> b
	require.Len(t, pathsToB, 2)

	simplePaths := model.GetSimplePathsFrom(pathsToB)

	// a (NEXT) -> b (TO_C) -> c
	// a (OTHER) -> b (TO_C) -> c
	// a (NEXT) -> b (TO_D) -> d
	// a (OTHER) -> b (TO_D) -> d
	assert.Len(t, simplePaths, 4)

	for _, path := range simplePaths {
		assert.Len(t, path.Steps, 3)
	}
}
