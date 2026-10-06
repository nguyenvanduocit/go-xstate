package graph_test

import (
	"encoding/json"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nguyenvanduocit/go-xstate/graph"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

func graphLightMachine() *xs.StateMachine[any] {
	return xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "light",
		Initial: "green",
		States: xs.States{
			{Key: "green", On: map[string]xs.Transitions{
				"TIMER":        {{Target: "yellow"}},
				"POWER_OUTAGE": {{Target: "red.flashing"}},
				// pushing the walk button never does anything
				"PUSH_BUTTON": {{Actions: xs.Actions{xs.ActionRef{Type: "doNothing"}}}},
			}},
			{Key: "yellow", On: map[string]xs.Transitions{
				"TIMER":        {{Target: "red"}},
				"POWER_OUTAGE": {{Target: "red.flashing"}},
			}},
			{
				Key: "red",
				On: map[string]xs.Transitions{
					"TIMER":        {{Target: "green"}},
					"POWER_OUTAGE": {{Target: "red.flashing"}},
				},
				// ...pedestrianStates
				Initial: "walk",
				States: xs.States{
					{Key: "walk", On: map[string]xs.Transitions{
						"PED_COUNTDOWN": {{Target: "wait", Actions: xs.Actions{xs.ActionRef{Type: "startCountdown"}}}},
					}},
					{Key: "wait", On: map[string]xs.Transitions{
						"PED_COUNTDOWN": {{Target: "stop"}},
					}},
					{Key: "stop"},
					{Key: "flashing"},
				},
			},
		},
	})
}

func graphParallelMachine() *xs.StateMachine[any] {
	return xs.CreateMachine(xs.MachineConfig[any]{
		Type: xs.Parallel,
		ID:   "p",
		States: xs.States{
			{Key: "a", Initial: "a1", States: xs.States{
				{Key: "a1", On: map[string]xs.Transitions{"2": {{Target: "a2"}}, "3": {{Target: "a3"}}}},
				{Key: "a2", On: map[string]xs.Transitions{"3": {{Target: "a3"}}, "1": {{Target: "a1"}}}},
				{Key: "a3"},
			}},
			{Key: "b", Initial: "b1", States: xs.States{
				{Key: "b1", On: map[string]xs.Transitions{"2": {{Target: "b2"}}, "3": {{Target: "b3"}}}},
				{Key: "b2", On: map[string]xs.Transitions{"3": {{Target: "b3"}}, "1": {{Target: "b1"}}}},
				{Key: "b3"},
			}},
		},
	})
}

func graphEquivMachine() *xs.StateMachine[any] {
	return xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{"FOO": {{Target: "b"}}, "BAR": {{Target: "b"}}}},
			{Key: "b", On: map[string]xs.Transitions{"FOO": {{Target: "a"}}, "BAR": {{Target: "a"}}}},
		},
	})
}

func graphABCMachine() *xs.StateMachine[any] {
	return xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{"toB": {{Target: "b"}}}},
			{Key: "b", On: map[string]xs.Transitions{"toC": {{Target: "c"}}}},
			{Key: "c", On: map[string]xs.Transitions{"toA": {{Target: "a"}}}},
		},
	})
}

func graphJoinMachine() *xs.StateMachine[any] {
	return xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{"NEXT": {{Target: "b"}}}},
			{Key: "b", On: map[string]xs.Transitions{"TO_C": {{Target: "c"}}}},
			{Key: "c"},
		},
	})
}

// graphFindPath mirrors paths.find(pred).
func graphFindPath[S xs.Snapshot](paths []graph.StatePath[S], pred func(graph.StatePath[S]) bool) (graph.StatePath[S], bool) {
	for _, p := range paths {
		if pred(p) {
			return p, true
		}
	}
	return graph.StatePath[S]{}, false
}

func graphStateIDs(nodes []*xs.StateNode) []string {
	ids := make([]string, 0, len(nodes))
	for _, n := range nodes {
		ids = append(ids, n.ID)
	}
	sort.Strings(ids)
	return ids
}

func graphMachineValues[C any](paths []graph.StatePath[*xs.MachineSnapshot[C]]) []any {
	out := make([]any, 0, len(paths))
	for _, p := range paths {
		out = append(out, p.State.Value)
	}
	return out
}

// JS: @xstate/graph > getStateNodes() > should return an array of all nodes
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/graph.test.ts#L168
func TestGraph_GetStateNodes_ShouldReturnAnArrayOfAllNodes(t *testing.T) {
	nodes := graph.GetStateNodes(graphLightMachine().Root)
	for _, node := range nodes {
		assert.IsType(t, &xs.StateNode{}, node)
		assert.NotNil(t, node)
	}
	assert.Equal(t, []string{
		"light.green",
		"light.red",
		"light.red.flashing",
		"light.red.stop",
		"light.red.wait",
		"light.red.walk",
		"light.yellow",
	}, graphStateIDs(nodes))
}

// JS: @xstate/graph > getStateNodes() > should return an array of all nodes (parallel)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/graph.test.ts#L182
func TestGraph_GetStateNodes_ShouldReturnAnArrayOfAllNodesParallel(t *testing.T) {
	nodes := graph.GetStateNodes(graphParallelMachine().Root)
	for _, node := range nodes {
		assert.IsType(t, &xs.StateNode{}, node)
		assert.NotNil(t, node)
	}
	assert.Equal(t, []string{
		"p.a",
		"p.a.a1",
		"p.a.a2",
		"p.a.a3",
		"p.b",
		"p.b.b1",
		"p.b.b2",
		"p.b.b3",
	}, graphStateIDs(nodes))
}

// JS: @xstate/graph > getShortestPaths() > should return a mapping of shortest paths to all states
//
// Snapshot "shortest paths 1" inlined; path order re-derived with sorted own
// events (JS iterates `on` keys in insertion order).
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/graph.test.ts#L199
func TestGraph_GetShortestPaths_ShouldReturnAMappingOfShortestPathsToAllStates(t *testing.T) {
	paths := graph.GetShortestPaths(graphLightMachine())

	assert.Equal(t, []graphPathSnap{
		{State: "green", Steps: []graphStepSnap{
			{State: "green", EventType: "xstate.init"},
		}},
		{State: map[string]any{"red": "flashing"}, Steps: []graphStepSnap{
			{State: "green", EventType: "xstate.init"},
			{State: map[string]any{"red": "flashing"}, EventType: "POWER_OUTAGE"},
		}},
		{State: "yellow", Steps: []graphStepSnap{
			{State: "green", EventType: "xstate.init"},
			{State: "yellow", EventType: "TIMER"},
		}},
		{State: map[string]any{"red": "walk"}, Steps: []graphStepSnap{
			{State: "green", EventType: "xstate.init"},
			{State: "yellow", EventType: "TIMER"},
			{State: map[string]any{"red": "walk"}, EventType: "TIMER"},
		}},
		{State: map[string]any{"red": "wait"}, Steps: []graphStepSnap{
			{State: "green", EventType: "xstate.init"},
			{State: "yellow", EventType: "TIMER"},
			{State: map[string]any{"red": "walk"}, EventType: "TIMER"},
			{State: map[string]any{"red": "wait"}, EventType: "PED_COUNTDOWN"},
		}},
		{State: map[string]any{"red": "stop"}, Steps: []graphStepSnap{
			{State: "green", EventType: "xstate.init"},
			{State: "yellow", EventType: "TIMER"},
			{State: map[string]any{"red": "walk"}, EventType: "TIMER"},
			{State: map[string]any{"red": "wait"}, EventType: "PED_COUNTDOWN"},
			{State: map[string]any{"red": "stop"}, EventType: "PED_COUNTDOWN"},
		}},
	}, graphPathsSnapshot(paths, graphMachineValue[any]))
}

// JS: @xstate/graph > getShortestPaths() > should return a mapping of shortest paths to all states (parallel)
//
// Snapshot "shortest paths parallel 1" inlined.
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/graph.test.ts#L205
func TestGraph_GetShortestPaths_ShouldReturnAMappingOfShortestPathsToAllStatesParallel(t *testing.T) {
	paths := graph.GetShortestPaths(graphParallelMachine())

	assert.Equal(t, []graphPathSnap{
		{State: map[string]any{"a": "a1", "b": "b1"}, Steps: []graphStepSnap{
			{State: map[string]any{"a": "a1", "b": "b1"}, EventType: "xstate.init"},
		}},
		{State: map[string]any{"a": "a2", "b": "b2"}, Steps: []graphStepSnap{
			{State: map[string]any{"a": "a1", "b": "b1"}, EventType: "xstate.init"},
			{State: map[string]any{"a": "a2", "b": "b2"}, EventType: "2"},
		}},
		{State: map[string]any{"a": "a3", "b": "b3"}, Steps: []graphStepSnap{
			{State: map[string]any{"a": "a1", "b": "b1"}, EventType: "xstate.init"},
			{State: map[string]any{"a": "a3", "b": "b3"}, EventType: "3"},
		}},
	}, graphPathsSnapshot(paths, graphMachineValue[any]))
}

// JS: @xstate/graph > getShortestPaths() > the initial state should have a single-length path
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/graph.test.ts#L212
func TestGraph_GetShortestPaths_TheInitialStateShouldHaveASingleLengthPath(t *testing.T) {
	lightMachine := graphLightMachine()
	shortestPaths := graph.GetShortestPaths(lightMachine)

	initialValue := xs.GetInitialSnapshot(lightMachine).Value
	path, ok := graphFindPath(shortestPaths, func(p graph.StatePath[graphAnySnap]) bool {
		return p.State.Matches(initialValue)
	})
	require.True(t, ok)
	assert.Len(t, path.Steps, 1)
}

// JS: @xstate/graph > getShortestPaths() > should not throw when a condition is present
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/graph.test.ts#L224
func TestGraph_GetShortestPaths_ShouldNotThrowWhenAConditionIsPresent(t *testing.T) {
	t.Skip("skipped in JS")
}

// JS: @xstate/graph > getShortestPaths() > should represent conditional paths based on context
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/graph.test.ts#L228
func TestGraph_GetShortestPaths_ShouldRepresentConditionalPathsBasedOnContext(t *testing.T) {
	t.Skip("skipped in JS")
}

// JS: @xstate/graph > getSimplePaths() > should return a mapping of arrays of simple paths to all states
//
// Inline snapshot + snapshot "should return a mapping of arrays of simple
// paths to all states 2" inlined; path order re-derived with sorted own
// events (JS iterates `on` keys in insertion order).
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/graph.test.ts#L278
func TestGraph_GetSimplePaths_ShouldReturnAMappingOfArraysOfSimplePathsToAllStates(t *testing.T) {
	paths := graph.GetSimplePaths(graphLightMachine())

	// Multiple different ways to get to flashing (from any other state)
	assert.Equal(t, []any{"green", map[string]any{"red": "flashing"}, map[string]any{"red": "flashing"}, map[string]any{"red": "flashing"}, map[string]any{"red": "flashing"}, map[string]any{"red": "flashing"}, "yellow", map[string]any{"red": "walk"}, map[string]any{"red": "wait"}, map[string]any{"red": "stop"}}, graphMachineValues(paths))

	assert.Equal(t, []graphPathSnap{
		{State: "green", Steps: []graphStepSnap{
			{State: "green", EventType: "xstate.init"},
		}},
		{State: map[string]any{"red": "flashing"}, Steps: []graphStepSnap{
			{State: "green", EventType: "xstate.init"},
			{State: map[string]any{"red": "flashing"}, EventType: "POWER_OUTAGE"},
		}},
		{State: map[string]any{"red": "flashing"}, Steps: []graphStepSnap{
			{State: "green", EventType: "xstate.init"},
			{State: "yellow", EventType: "TIMER"},
			{State: map[string]any{"red": "flashing"}, EventType: "POWER_OUTAGE"},
		}},
		{State: map[string]any{"red": "flashing"}, Steps: []graphStepSnap{
			{State: "green", EventType: "xstate.init"},
			{State: "yellow", EventType: "TIMER"},
			{State: map[string]any{"red": "walk"}, EventType: "TIMER"},
			{State: map[string]any{"red": "flashing"}, EventType: "POWER_OUTAGE"},
		}},
		{State: map[string]any{"red": "flashing"}, Steps: []graphStepSnap{
			{State: "green", EventType: "xstate.init"},
			{State: "yellow", EventType: "TIMER"},
			{State: map[string]any{"red": "walk"}, EventType: "TIMER"},
			{State: map[string]any{"red": "wait"}, EventType: "PED_COUNTDOWN"},
			{State: map[string]any{"red": "flashing"}, EventType: "POWER_OUTAGE"},
		}},
		{State: map[string]any{"red": "flashing"}, Steps: []graphStepSnap{
			{State: "green", EventType: "xstate.init"},
			{State: "yellow", EventType: "TIMER"},
			{State: map[string]any{"red": "walk"}, EventType: "TIMER"},
			{State: map[string]any{"red": "wait"}, EventType: "PED_COUNTDOWN"},
			{State: map[string]any{"red": "stop"}, EventType: "PED_COUNTDOWN"},
			{State: map[string]any{"red": "flashing"}, EventType: "POWER_OUTAGE"},
		}},
		{State: "yellow", Steps: []graphStepSnap{
			{State: "green", EventType: "xstate.init"},
			{State: "yellow", EventType: "TIMER"},
		}},
		{State: map[string]any{"red": "walk"}, Steps: []graphStepSnap{
			{State: "green", EventType: "xstate.init"},
			{State: "yellow", EventType: "TIMER"},
			{State: map[string]any{"red": "walk"}, EventType: "TIMER"},
		}},
		{State: map[string]any{"red": "wait"}, Steps: []graphStepSnap{
			{State: "green", EventType: "xstate.init"},
			{State: "yellow", EventType: "TIMER"},
			{State: map[string]any{"red": "walk"}, EventType: "TIMER"},
			{State: map[string]any{"red": "wait"}, EventType: "PED_COUNTDOWN"},
		}},
		{State: map[string]any{"red": "stop"}, Steps: []graphStepSnap{
			{State: "green", EventType: "xstate.init"},
			{State: "yellow", EventType: "TIMER"},
			{State: map[string]any{"red": "walk"}, EventType: "TIMER"},
			{State: map[string]any{"red": "wait"}, EventType: "PED_COUNTDOWN"},
			{State: map[string]any{"red": "stop"}, EventType: "PED_COUNTDOWN"},
		}},
	}, graphPathsSnapshot(paths, graphMachineValue[any]))
}

// JS: @xstate/graph > getSimplePaths() > should return a mapping of simple paths to all states (parallel)
//
// Snapshot "simple paths parallel 1" inlined.
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/graph.test.ts#L324
func TestGraph_GetSimplePaths_ShouldReturnAMappingOfSimplePathsToAllStatesParallel(t *testing.T) {
	paths := graph.GetSimplePaths(graphParallelMachine())

	assert.Equal(t, []any{map[string]any{"a": "a1", "b": "b1"}, map[string]any{"a": "a2", "b": "b2"}, map[string]any{"a": "a3", "b": "b3"}, map[string]any{"a": "a3", "b": "b3"}}, graphMachineValues(paths))
	assert.Equal(t, []graphPathSnap{
		{State: map[string]any{"a": "a1", "b": "b1"}, Steps: []graphStepSnap{
			{State: map[string]any{"a": "a1", "b": "b1"}, EventType: "xstate.init"},
		}},
		{State: map[string]any{"a": "a2", "b": "b2"}, Steps: []graphStepSnap{
			{State: map[string]any{"a": "a1", "b": "b1"}, EventType: "xstate.init"},
			{State: map[string]any{"a": "a2", "b": "b2"}, EventType: "2"},
		}},
		{State: map[string]any{"a": "a3", "b": "b3"}, Steps: []graphStepSnap{
			{State: map[string]any{"a": "a1", "b": "b1"}, EventType: "xstate.init"},
			{State: map[string]any{"a": "a2", "b": "b2"}, EventType: "2"},
			{State: map[string]any{"a": "a3", "b": "b3"}, EventType: "3"},
		}},
		{State: map[string]any{"a": "a3", "b": "b3"}, Steps: []graphStepSnap{
			{State: map[string]any{"a": "a1", "b": "b1"}, EventType: "xstate.init"},
			{State: map[string]any{"a": "a3", "b": "b3"}, EventType: "3"},
		}},
	}, graphPathsSnapshot(paths, graphMachineValue[any]))
}

// JS: @xstate/graph > getSimplePaths() > should return multiple paths for equivalent transitions
//
// Snapshot "simple paths equal transitions 1" inlined; BAR precedes FOO
// because own events are sorted (JS: FOO first, by `on` insertion order).
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/graph.test.ts#L350
func TestGraph_GetSimplePaths_ShouldReturnMultiplePathsForEquivalentTransitions(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{"FOO": {{Target: "b"}}, "BAR": {{Target: "b"}}}},
			{Key: "b", On: map[string]xs.Transitions{"FOO": {{Target: "a"}}, "BAR": {{Target: "a"}}}},
		},
	})

	paths := graph.GetSimplePaths(machine)

	assert.Equal(t, []any{"a", "b", "b"}, graphMachineValues(paths))
	assert.Equal(t, []graphPathSnap{
		{State: "a", Steps: []graphStepSnap{
			{State: "a", EventType: "xstate.init"},
		}},
		{State: "b", Steps: []graphStepSnap{
			{State: "a", EventType: "xstate.init"},
			{State: "b", EventType: "BAR"},
		}},
		{State: "b", Steps: []graphStepSnap{
			{State: "a", EventType: "xstate.init"},
			{State: "b", EventType: "FOO"},
		}},
	}, graphPathsSnapshot(paths, graphMachineValue[any]))
}

// JS: @xstate/graph > getSimplePaths() > should return a single-length path for the initial state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/graph.test.ts#L373
func TestGraph_GetSimplePaths_ShouldReturnASingleLengthPathForTheInitialState(t *testing.T) {
	lightMachine := graphLightMachine()
	equivMachine := graphEquivMachine()
	matchesInitial := func(m *xs.StateMachine[any]) func(graph.StatePath[graphAnySnap]) bool {
		initialValue := xs.GetInitialSnapshot(m).Value
		return func(p graph.StatePath[graphAnySnap]) bool { return p.State.Matches(initialValue) }
	}

	_, ok := graphFindPath(graph.GetSimplePaths(lightMachine), matchesInitial(lightMachine))
	assert.True(t, ok)
	path, ok := graphFindPath(graph.GetSimplePaths(lightMachine), matchesInitial(lightMachine))
	require.True(t, ok)
	assert.Len(t, path.Steps, 1)
	_, ok = graphFindPath(graph.GetSimplePaths(equivMachine), matchesInitial(equivMachine))
	assert.True(t, ok)
	path, ok = graphFindPath(graph.GetSimplePaths(equivMachine), matchesInitial(equivMachine))
	require.True(t, ok)
	assert.Len(t, path.Steps, 1)
}

// JS: @xstate/graph > getSimplePaths() > should return value-based paths
//
// Snapshot "simple paths context 1" inlined.
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/graph.test.ts#L404
func TestGraph_GetSimplePaths_ShouldReturnValueBasedPaths(t *testing.T) {
	type ctx struct {
		Count int `json:"count"`
	}
	countMachine := xs.CreateMachine(xs.MachineConfig[ctx]{
		ID:      "count",
		Initial: "start",
		Context: ctx{Count: 0},
		States: xs.States{
			{
				Key: "start",
				Always: xs.Transitions{{
					Target: "finish",
					Guard:  xs.GuardFunc(func(a xs.GuardArgs[ctx]) bool { return a.Context.Count == 3 }),
				}},
				On: map[string]xs.Transitions{
					"INC": {{Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
						return ctx{Count: a.Context.Count + 1}
					})}}},
				},
			},
			{Key: "finish"},
		},
	})

	paths := graph.GetSimplePaths(countMachine, graph.TraversalOptions[*xs.MachineSnapshot[ctx]]{
		Events: []xs.Event{xs.E{"type": "INC", "value": 1}},
	})

	assert.Equal(t, []any{"start", "start", "start", "finish"}, graphMachineValues(paths))
	assert.Equal(t, []graphPathSnap{
		{State: "start", Steps: []graphStepSnap{
			{State: "start", EventType: "xstate.init"},
		}},
		{State: "start", Steps: []graphStepSnap{
			{State: "start", EventType: "xstate.init"},
			{State: "start", EventType: "INC"},
		}},
		{State: "start", Steps: []graphStepSnap{
			{State: "start", EventType: "xstate.init"},
			{State: "start", EventType: "INC"},
			{State: "start", EventType: "INC"},
		}},
		{State: "finish", Steps: []graphStepSnap{
			{State: "start", EventType: "xstate.init"},
			{State: "start", EventType: "INC"},
			{State: "start", EventType: "INC"},
			{State: "finish", EventType: "INC"},
		}},
	}, graphPathsSnapshot(paths, graphMachineValue[ctx]))
}

// JS: @xstate/graph > getSimplePaths() > should support filtering disabled events
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/graph.test.ts#L452
func TestGraph_GetSimplePaths_ShouldSupportFilteringDisabledEvents(t *testing.T) {
	type ctx struct {
		Allowed bool `json:"allowed"`
	}
	type snap = *xs.MachineSnapshot[ctx]
	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		ID:      "guarded-default-events",
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

	paths := graph.GetSimplePaths(machine, graph.TraversalOptions[snap]{
		FilterEvents: func(state snap, event xs.Event) bool {
			return !xs.IsMachineSnapshot(state) || state.Can(event)
		},
		ToState: func(state snap) bool { return state.Status == xs.StatusDone },
	})

	var got [][]string
	for _, p := range paths {
		got = append(got, graphEventTypes(p.Steps))
	}
	assert.Equal(t, [][]string{
		{"xstate.init", "NEXT", "ALLOW", "PROCEED"},
	}, got)
}

// JS: @xstate/graph > getPathFromEvents() > should return a path to the last entered state by the event sequence
//
// Snapshot "path from events 1" inlined.
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/graph.test.ts#L501
func TestGraph_GetPathFromEvents_ShouldReturnAPathToTheLastEnteredStateByTheEventSequence(t *testing.T) {
	paths := graph.GetPathsFromEvents(graphLightMachine(), []xs.Event{
		xs.Ev("TIMER"),
		xs.Ev("TIMER"),
		xs.Ev("TIMER"),
		xs.Ev("POWER_OUTAGE"),
	})

	require.Equal(t, 1, len(paths))

	assert.Equal(t, graphPathSnap{State: map[string]any{"red": "flashing"}, Steps: []graphStepSnap{
		{State: "green", EventType: "xstate.init"},
		{State: "yellow", EventType: "TIMER"},
		{State: map[string]any{"red": "walk"}, EventType: "TIMER"},
		{State: "green", EventType: "TIMER"},
		{State: map[string]any{"red": "flashing"}, EventType: "POWER_OUTAGE"},
	}}, graphPathSnapshot(paths[0], graphMachineValue[any]))
}

// JS: @xstate/graph > getPathFromEvents() > should throw when an invalid event sequence is provided
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/graph.test.ts#L514
func TestGraph_GetPathFromEvents_ShouldThrowWhenAnInvalidEventSequenceIsProvided(t *testing.T) {
	t.Skip("skipped in JS")
}

// JS: @xstate/graph > getPathFromEvents() > should return a path from a specified from-state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/graph.test.ts#L523
func TestGraph_GetPathFromEvents_ShouldReturnAPathFromASpecifiedFromState(t *testing.T) {
	lightMachine := graphLightMachine()
	paths := graph.GetPathsFromEvents(lightMachine, []xs.Event{xs.Ev("TIMER")}, graph.TraversalOptions[graphAnySnap]{
		FromState: lightMachine.ResolveState(xs.ResolveStateConfig[any]{Value: "yellow"}),
	})

	require.NotEmpty(t, paths)
	path := paths[0]

	assert.True(t, path.State.Matches("red"))
}

// JS: @xstate/graph > toDirectedGraph > should represent a statechart as a directed graph
//
// Snapshot "should represent a statechart as a directed graph 1" inlined
// (pretty-format serializes via toJSON()).
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/graph.test.ts#L535
func TestGraph_ToDirectedGraph_ShouldRepresentAStatechartAsADirectedGraph(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "light",
		Initial: "green",
		States: xs.States{
			{Key: "green", On: map[string]xs.Transitions{"TIMER": {{Target: "yellow"}}}},
			{Key: "yellow", On: map[string]xs.Transitions{"TIMER": {{Target: "red"}}}},
			{
				Key:     "red",
				Initial: "walk",
				States: xs.States{
					{Key: "walk", On: map[string]xs.Transitions{"COUNTDOWN": {{Target: "wait"}}}},
					{Key: "wait", On: map[string]xs.Transitions{"COUNTDOWN": {{Target: "stop"}}}},
					{Key: "stop", On: map[string]xs.Transitions{"COUNTDOWN": {{Target: "finished"}}}},
					{Key: "finished", Type: xs.Final},
				},
				OnDone: xs.Transitions{{Target: "green"}},
			},
		},
	})

	digraph := graph.ToDirectedGraph(machine.Root)

	type node = graph.DirectedGraphNodeJSON
	type edge = graph.DirectedGraphEdgeJSON
	label := func(text string) graph.DirectedGraphLabel { return graph.DirectedGraphLabel{Text: text} }
	leaf := func(id string, edges ...edge) node {
		if edges == nil {
			edges = []edge{}
		}
		return node{ID: id, Children: []node{}, Edges: edges}
	}

	assert.Equal(t, node{
		ID: "light",
		Children: []node{
			leaf("light.green", edge{Label: label("TIMER"), Source: "light.green", Target: "light.yellow"}),
			leaf("light.yellow", edge{Label: label("TIMER"), Source: "light.yellow", Target: "light.red"}),
			{
				ID: "light.red",
				Children: []node{
					leaf("light.red.walk", edge{Label: label("COUNTDOWN"), Source: "light.red.walk", Target: "light.red.wait"}),
					leaf("light.red.wait", edge{Label: label("COUNTDOWN"), Source: "light.red.wait", Target: "light.red.stop"}),
					leaf("light.red.stop", edge{Label: label("COUNTDOWN"), Source: "light.red.stop", Target: "light.red.finished"}),
					leaf("light.red.finished"),
				},
				Edges: []edge{
					{Label: label("xstate.done.state.light.red"), Source: "light.red", Target: "light.green"},
				},
			},
		},
		Edges: []edge{},
	}, digraph.ToJSON())
}

type graphTransitionSnap = *xs.TransitionSnapshot[int]

func graphABResetLogic() *xs.TransitionLogic[int] {
	return xs.FromTransition(func(s int, e xs.Event, _ *xs.ActorScope) int {
		if e.EventType() == "a" {
			return 1
		}
		if e.EventType() == "b" && s == 1 {
			return 2
		}
		if e.EventType() == "reset" {
			return 0
		}
		return s
	}, func(xs.TransitionInitArgs) int { return 0 })
}

func graphABResetOptions() graph.TraversalOptions[graphTransitionSnap] {
	return graph.TraversalOptions[graphTransitionSnap]{
		Events: []xs.Event{xs.Ev("a"), xs.Ev("b"), xs.Ev("reset")},
		// JSON.stringify(v) + ' | ' + JSON.stringify(e)
		SerializeState: func(v graphTransitionSnap, e xs.Event, _ graphTransitionSnap) string {
			vb, err := json.Marshal(v)
			if err != nil {
				panic(err)
			}
			eb, err := json.Marshal(e)
			if err != nil {
				panic(err)
			}
			return string(vb) + " | " + string(eb)
		},
	}
}

// JS: simple paths for transition functions
//
// Snapshot "simple paths for transition functions 1" inlined (the JS test
// calls getShortestPaths despite its name).
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/graph.test.ts#L562
func TestGraph_SimplePathsForTransitionFunctions(t *testing.T) {
	a := graph.GetShortestPaths(graphABResetLogic(), graphABResetOptions())

	assert.Equal(t, []graphPathSnap{
		{State: 0, Steps: []graphStepSnap{
			{State: 0, EventType: "xstate.init"},
		}},
		{State: 1, Steps: []graphStepSnap{
			{State: 0, EventType: "xstate.init"},
			{State: 1, EventType: "a"},
		}},
		{State: 0, Steps: []graphStepSnap{
			{State: 0, EventType: "xstate.init"},
			{State: 0, EventType: "b"},
		}},
		{State: 0, Steps: []graphStepSnap{
			{State: 0, EventType: "xstate.init"},
			{State: 0, EventType: "reset"},
		}},
		{State: 2, Steps: []graphStepSnap{
			{State: 0, EventType: "xstate.init"},
			{State: 1, EventType: "a"},
			{State: 2, EventType: "b"},
		}},
	}, graphPathsSnapshot(a, graphTransitionContext[int]))
}

// JS: shortest paths for transition functions
//
// Snapshot "shortest paths for transition functions 1" inlined (the JS test
// calls getSimplePaths despite its name).
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/graph.test.ts#L583
func TestGraph_ShortestPathsForTransitionFunctions(t *testing.T) {
	a := graph.GetSimplePaths(graphABResetLogic(), graphABResetOptions())

	assert.Equal(t, []graphPathSnap{
		{State: 0, Steps: []graphStepSnap{
			{State: 0, EventType: "xstate.init"},
		}},
		{State: 1, Steps: []graphStepSnap{
			{State: 0, EventType: "xstate.init"},
			{State: 1, EventType: "a"},
		}},
		{State: 1, Steps: []graphStepSnap{
			{State: 0, EventType: "xstate.init"},
			{State: 0, EventType: "b"},
			{State: 1, EventType: "a"},
		}},
		{State: 1, Steps: []graphStepSnap{
			{State: 0, EventType: "xstate.init"},
			{State: 0, EventType: "b"},
			{State: 0, EventType: "reset"},
			{State: 1, EventType: "a"},
		}},
		{State: 1, Steps: []graphStepSnap{
			{State: 0, EventType: "xstate.init"},
			{State: 0, EventType: "reset"},
			{State: 1, EventType: "a"},
		}},
		{State: 1, Steps: []graphStepSnap{
			{State: 0, EventType: "xstate.init"},
			{State: 0, EventType: "reset"},
			{State: 0, EventType: "b"},
			{State: 1, EventType: "a"},
		}},
		{State: 0, Steps: []graphStepSnap{
			{State: 0, EventType: "xstate.init"},
			{State: 1, EventType: "a"},
			{State: 2, EventType: "b"},
			{State: 0, EventType: "reset"},
			{State: 0, EventType: "b"},
		}},
		{State: 0, Steps: []graphStepSnap{
			{State: 0, EventType: "xstate.init"},
			{State: 1, EventType: "a"},
			{State: 0, EventType: "reset"},
			{State: 0, EventType: "b"},
		}},
		{State: 0, Steps: []graphStepSnap{
			{State: 0, EventType: "xstate.init"},
			{State: 0, EventType: "b"},
		}},
		{State: 0, Steps: []graphStepSnap{
			{State: 0, EventType: "xstate.init"},
			{State: 0, EventType: "reset"},
			{State: 0, EventType: "b"},
		}},
		{State: 0, Steps: []graphStepSnap{
			{State: 0, EventType: "xstate.init"},
			{State: 1, EventType: "a"},
			{State: 2, EventType: "b"},
			{State: 0, EventType: "reset"},
		}},
		{State: 0, Steps: []graphStepSnap{
			{State: 0, EventType: "xstate.init"},
			{State: 1, EventType: "a"},
			{State: 0, EventType: "reset"},
		}},
		{State: 0, Steps: []graphStepSnap{
			{State: 0, EventType: "xstate.init"},
			{State: 0, EventType: "b"},
			{State: 1, EventType: "a"},
			{State: 2, EventType: "b"},
			{State: 0, EventType: "reset"},
		}},
		{State: 0, Steps: []graphStepSnap{
			{State: 0, EventType: "xstate.init"},
			{State: 0, EventType: "b"},
			{State: 1, EventType: "a"},
			{State: 0, EventType: "reset"},
		}},
		{State: 0, Steps: []graphStepSnap{
			{State: 0, EventType: "xstate.init"},
			{State: 0, EventType: "b"},
			{State: 0, EventType: "reset"},
		}},
		{State: 0, Steps: []graphStepSnap{
			{State: 0, EventType: "xstate.init"},
			{State: 0, EventType: "reset"},
		}},
		{State: 2, Steps: []graphStepSnap{
			{State: 0, EventType: "xstate.init"},
			{State: 1, EventType: "a"},
			{State: 2, EventType: "b"},
		}},
		{State: 2, Steps: []graphStepSnap{
			{State: 0, EventType: "xstate.init"},
			{State: 0, EventType: "b"},
			{State: 1, EventType: "a"},
			{State: 2, EventType: "b"},
		}},
		{State: 2, Steps: []graphStepSnap{
			{State: 0, EventType: "xstate.init"},
			{State: 0, EventType: "b"},
			{State: 0, EventType: "reset"},
			{State: 1, EventType: "a"},
			{State: 2, EventType: "b"},
		}},
		{State: 2, Steps: []graphStepSnap{
			{State: 0, EventType: "xstate.init"},
			{State: 0, EventType: "reset"},
			{State: 1, EventType: "a"},
			{State: 2, EventType: "b"},
		}},
		{State: 2, Steps: []graphStepSnap{
			{State: 0, EventType: "xstate.init"},
			{State: 0, EventType: "reset"},
			{State: 0, EventType: "b"},
			{State: 1, EventType: "a"},
			{State: 2, EventType: "b"},
		}},
	}, graphPathsSnapshot(a, graphTransitionContext[int]))
}

// JS: filtering > should not traverse past filtered states
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/graph.test.ts#L605
func TestGraph_Filtering_ShouldNotTraversePastFilteredStates(t *testing.T) {
	type ctx struct {
		Count int `json:"count"`
	}
	type snap = *xs.MachineSnapshot[ctx]
	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Initial: "counting",
		Context: ctx{Count: 0},
		States: xs.States{
			{Key: "counting", On: map[string]xs.Transitions{
				"INC": {{Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
					return ctx{Count: a.Context.Count + 1}
				})}}},
			}},
		},
	})

	shortestPaths := graph.GetShortestPaths(machine, graph.TraversalOptions[snap]{
		Events:   []xs.Event{xs.Ev("INC")},
		StopWhen: func(state snap) bool { return state.Context.Count == 5 },
	})

	var contexts []ctx
	for _, p := range shortestPaths {
		contexts = append(contexts, p.State.Context)
	}
	assert.Equal(t, []ctx{{Count: 0}, {Count: 1}, {Count: 2}, {Count: 3}, {Count: 4}, {Count: 5}}, contexts)
}

// JS: should provide previous state for serializeState()
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/graph.test.ts#L653
func TestGraph_ShouldProvidePreviousStateForSerializeState(t *testing.T) {
	machine := graphABCMachine()

	jsonValue := func(v xs.StateValue) string {
		b, err := json.Marshal(v)
		if err != nil {
			panic(err)
		}
		return string(b)
	}

	shortestPaths := graph.GetShortestPaths(machine, graph.TraversalOptions[graphAnySnap]{
		SerializeState: func(state graphAnySnap, event xs.Event, prevState graphAnySnap) string {
			eventType := "undefined" // `${event?.type}`
			if event != nil {
				eventType = event.EventType()
			}
			prev := ""
			if prevState != nil {
				prev = " via " + jsonValue(prevState.Value)
			}
			return jsonValue(state.Value) + " via " + eventType + prev
		},
	})

	// Should be [1, 4]:
	// 1 (a)
	// 4 (a -> b -> c -> a)
	var lengths []int
	for _, p := range shortestPaths {
		if p.State.Matches("a") {
			lengths = append(lengths, len(p.Steps))
		}
	}
	assert.Equal(t, []int{1, 4}, lengths)
}

// JS: from-state can be specified (it.each([getShortestPaths, getSimplePaths]))
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/graph.test.ts#L687
func TestGraph_FromStateCanBeSpecified(t *testing.T) {
	type pathGetter = func(xs.TypedActorLogic[graphAnySnap], ...graph.TraversalOptions[graphAnySnap]) []graph.StatePath[graphAnySnap]
	for _, tc := range []struct {
		name       string
		pathGetter pathGetter
	}{
		{"getShortestPaths", graph.GetShortestPaths[graphAnySnap]},
		{"getSimplePaths", graph.GetSimplePaths[graphAnySnap]},
	} {
		// JS test (shared case definition): https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/graph.test.ts#L687
		t.Run(tc.name, func(t *testing.T) {
			machine := graphABCMachine()

			paths := tc.pathGetter(machine, graph.TraversalOptions[graphAnySnap]{
				FromState: machine.ResolveState(xs.ResolveStateConfig[any]{Value: "b"}),
			})

			// Instead of taking 2 steps to reach state 'b' (A, B),
			// there should exist a path that takes 1 step
			_, ok := graphFindPath(paths, func(p graph.StatePath[graphAnySnap]) bool {
				return p.State.Matches("b") && len(p.Steps) == 1
			})
			assert.True(t, ok)

			// Instead of starting at state 'a', it should take > 0 steps to reach 'a'
			_, ok = graphFindPath(paths, func(p graph.StatePath[graphAnySnap]) bool {
				return p.State.Matches("a") && len(p.Steps) > 0
			})
			assert.True(t, ok)
		})
	}
}

// JS: joinPaths() > should join two paths
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/graph.test.ts#L723
func TestGraph_JoinPaths_ShouldJoinTwoPaths(t *testing.T) {
	machine := graphJoinMachine()

	pathsToB := graph.GetPathsFromEvents(machine, []xs.Event{xs.Ev("NEXT")})
	require.NotEmpty(t, pathsToB)
	pathToB := pathsToB[0]
	pathsToC := graph.GetPathsFromEvents(machine, []xs.Event{xs.Ev("TO_C")}, graph.TraversalOptions[graphAnySnap]{
		FromState: pathToB.State,
	})
	require.NotEmpty(t, pathsToC)
	pathToC := pathsToC[0]

	pathToBAndC := graph.JoinPaths(pathToB, pathToC)

	assert.Equal(t, []string{"xstate.init", "NEXT", "TO_C"}, graphEventTypes(pathToBAndC.Steps))

	assert.True(t, pathToBAndC.State.Matches("c"))
}

// JS: joinPaths() > should not join two paths with mismatched source/target states
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/graph.test.ts#L761
func TestGraph_JoinPaths_ShouldNotJoinTwoPathsWithMismatchedSourceTargetStates(t *testing.T) {
	machine := graphJoinMachine()

	pathsToB := graph.GetPathsFromEvents(machine, []xs.Event{xs.Ev("NEXT")})
	require.NotEmpty(t, pathsToB)
	pathToB := pathsToB[0]
	pathsToCFromA := graph.GetPathsFromEvents(machine, []xs.Event{xs.Ev("TO_C")})
	require.NotEmpty(t, pathsToCFromA)
	pathToCFromA := pathsToCFromA[0]

	assert.PanicsWithError(t, "Paths cannot be joined", func() {
		graph.JoinPaths(pathToB, pathToCFromA)
	})
}
