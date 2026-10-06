package graph_test

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nguyenvanduocit/go-xstate/graph"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// JS: events > should allow for representing many cases
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/index.test.ts#L6
func TestIndex_Events_ShouldAllowForRepresentingManyCases(t *testing.T) {
	feedbackMachine := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "feedback",
		Initial: "question",
		States: xs.States{
			{Key: "question", On: map[string]xs.Transitions{
				"CLICK_GOOD": {{Target: "thanks"}},
				"CLICK_BAD":  {{Target: "form"}},
				"CLOSE":      {{Target: "closed"}},
				"ESC":        {{Target: "closed"}},
			}},
			{
				Key: "form",
				On: map[string]xs.Transitions{
					"SUBMIT": {
						{
							Target: "thanks",
							Guard: xs.GuardFunc(func(a xs.GuardArgs[any]) bool {
								value, _ := a.Event.(xs.E)["value"].(string)
								return len(value) > 0
							}),
						},
						{Target: ".invalid"},
					},
					"CLOSE": {{Target: "closed"}},
					"ESC":   {{Target: "closed"}},
				},
				Initial: "valid",
				States: xs.States{
					{Key: "valid"},
					{Key: "invalid"},
				},
			},
			{Key: "thanks", On: map[string]xs.Transitions{
				"CLOSE": {{Target: "closed"}},
				"ESC":   {{Target: "closed"}},
			}},
			{Key: "closed", Type: xs.Final},
		},
	})

	testModel := graph.CreateTestModel(feedbackMachine, graph.TestModelOptions[graphAnySnap]{
		TraversalOptions: graph.TraversalOptions[graphAnySnap]{
			Events: []xs.Event{
				xs.E{"type": "SUBMIT", "value": "something"},
				xs.E{"type": "SUBMIT", "value": ""},
			},
		},
	})

	graphTestModel(t, testModel, graph.TestParam[graphAnySnap]{})
}

// JS: events > should not throw an error for unimplemented events
//
// JS wraps an async function in expect(...).not.toThrow(), which only checks
// that nothing throws synchronously; Go runs the paths synchronously.
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/index.test.ts#L70
func TestIndex_Events_ShouldNotThrowAnErrorForUnimplementedEvents(t *testing.T) {
	testMachine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "idle",
		States: xs.States{
			{Key: "idle", On: map[string]xs.Transitions{"ACTIVATE": {{Target: "active"}}}},
			{Key: "active"},
		},
	})

	testModel := graph.CreateTestModel(testMachine)

	assert.NotPanics(t, func() {
		graphTestModel(t, testModel, graph.TestParam[graphAnySnap]{})
	})
}

// JS: events > should allow for dynamic generation of cases based on state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/index.test.ts#L88
func TestIndex_Events_ShouldAllowForDynamicGenerationOfCasesBasedOnState(t *testing.T) {
	type ctx struct {
		Values []int `json:"values"`
	}
	type snap = *xs.MachineSnapshot[ctx]
	values := []int{1, 2, 3}
	eventValueIs := func(n int) xs.Guard {
		return xs.GuardFunc(func(a xs.GuardArgs[ctx]) bool { return a.Event.(xs.E)["value"] == n })
	}
	testMachine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Initial: "a",
		Context: ctx{
			Values: values, // to be read by generator
		},
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{
				"EVENT": {
					{Guard: eventValueIs(1), Target: "b"},
					{Guard: eventValueIs(2), Target: "c"},
					{Guard: eventValueIs(3), Target: "d"},
				},
			}},
			{Key: "b"},
			{Key: "c"},
			{Key: "d"},
		},
	})

	var testedEvents []xs.Event

	testModel := graph.CreateTestModel(testMachine, graph.TestModelOptions[snap]{
		TraversalOptions: graph.TraversalOptions[snap]{
			EventsFn: func(state snap) []xs.Event {
				var events []xs.Event
				for _, value := range state.Context.Values {
					events = append(events, xs.E{"type": "EVENT", "value": value})
				}
				return events
			},
		},
	})

	paths := testModel.GetShortestPaths()

	assert.Equal(t, 3, len(paths))

	graphTestPaths(t, paths, graph.TestParam[snap]{
		Events: map[string]graph.EventExecutor[snap]{
			"EVENT": func(step graph.Step[snap]) error {
				testedEvents = append(testedEvents, step.Event)
				return nil
			},
		},
	})

	assert.Equal(t, []xs.Event{
		xs.E{"type": "EVENT", "value": 1},
		xs.E{"type": "EVENT", "value": 2},
		xs.E{"type": "EVENT", "value": 3},
	}, testedEvents)
}

// JS: state limiting > should limit states with filter option
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/index.test.ts#L154
func TestIndex_StateLimiting_ShouldLimitStatesWithFilterOption(t *testing.T) {
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

	testModel := graph.CreateTestModel(machine)

	testPaths := testModel.GetShortestPaths(graph.GetPathOptions[snap]{
		TraversalOptions: graph.TraversalOptions[snap]{
			StopWhen: func(state snap) bool { return state.Context.Count >= 5 },
		},
	})

	assert.Len(t, testPaths, 1)
}

// JS: prevents infinite recursion based on a provided limit
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/index.test.ts#L185
func TestIndex_PreventsInfiniteRecursionBasedOnAProvidedLimit(t *testing.T) {
	type ctx struct {
		Count int `json:"count"`
	}
	type snap = *xs.MachineSnapshot[ctx]
	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		ID:      "machine",
		Context: ctx{Count: 0},
		On: map[string]xs.Transitions{
			"TOGGLE": {{Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
				return ctx{Count: a.Context.Count + 1}
			})}}},
		},
	})

	model := graph.CreateTestModel(machine)

	assert.PanicsWithError(t, "Traversal limit exceeded", func() {
		model.GetShortestPaths(graph.GetPathOptions[snap]{
			TraversalOptions: graph.TraversalOptions[snap]{Limit: 100},
		})
	})
}

// JS: test model options > options.testState(...) should test state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/index.test.ts#L207
func TestIndex_TestModelOptions_OptionsTestStateShouldTestState(t *testing.T) {
	var testedStates []xs.StateValue

	model := graph.CreateTestModel(xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "inactive",
		States: xs.States{
			{Key: "inactive", On: map[string]xs.Transitions{"NEXT": {{Target: "active"}}}},
			{Key: "active"},
		},
	}))

	graphTestModel(t, model, graph.TestParam[graphAnySnap]{
		States: map[string]func(graphAnySnap) error{
			"*": func(state graphAnySnap) error {
				testedStates = append(testedStates, state.Value)
				return nil
			},
		},
	})

	assert.Equal(t, []xs.StateValue{"inactive", "active"}, testedStates)
}

func indexFirstSecondMachine() *xs.StateMachine[any] {
	return xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "first",
		States: xs.States{
			{Key: "first", On: map[string]xs.Transitions{"NEXT": {{Target: "second"}}}},
			{Key: "second"},
		},
	})
}

// JS: tests transitions
//
// expect.assertions(2) → the executor (2 assertions) must run exactly once.
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/index.test.ts#L237
func TestIndex_TestsTransitions(t *testing.T) {
	machine := indexFirstSecondMachine()

	model := graph.CreateTestModel(machine)

	paths := model.GetShortestPaths(graph.GetPathOptions[graphAnySnap]{
		TraversalOptions: graph.TraversalOptions[graphAnySnap]{
			ToState: func(state graphAnySnap) bool { return state.Matches("second") },
		},
	})
	require.NotEmpty(t, paths)

	calls := 0
	_, err := paths[0].Test(graph.TestParam[graphAnySnap]{
		Events: map[string]graph.EventExecutor[graphAnySnap]{
			"NEXT": func(step graph.Step[graphAnySnap]) error {
				calls++
				assert.NotNil(t, step.Event)
				assert.NotNil(t, step.State)
				return nil
			},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, 1, calls)
}

// JS: Event in event executor should contain payload from case
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/index.test.ts#L266
func TestIndex_EventInEventExecutorShouldContainPayloadFromCase(t *testing.T) {
	machine := indexFirstSecondMachine()

	nonSerializableData := func() int { return 42 }

	model := graph.CreateTestModel(machine, graph.TestModelOptions[graphAnySnap]{
		TraversalOptions: graph.TraversalOptions[graphAnySnap]{
			Events: []xs.Event{xs.E{"type": "NEXT", "payload": 10, "fn": nonSerializableData}},
		},
	})

	paths := model.GetShortestPaths(graph.GetPathOptions[graphAnySnap]{
		TraversalOptions: graph.TraversalOptions[graphAnySnap]{
			ToState: func(state graphAnySnap) bool { return state.Matches("second") },
		},
	})
	require.NotEmpty(t, paths)

	_, err := model.TestPath(paths[0].StatePath, graph.TestParam[graphAnySnap]{
		Events: map[string]graph.EventExecutor[graphAnySnap]{
			"NEXT": func(step graph.Step[graphAnySnap]) error {
				// toEqual({type, payload, fn}); funcs compare by identity, which
				// reflect.DeepEqual cannot express, so compare field by field.
				ev, ok := step.Event.(xs.E)
				require.True(t, ok)
				assert.Len(t, ev, 3)
				assert.Equal(t, "NEXT", ev["type"])
				assert.Equal(t, 10, ev["payload"])
				fn, ok := ev["fn"].(func() int)
				require.True(t, ok)
				assert.Equal(t, reflect.ValueOf(nonSerializableData).Pointer(), reflect.ValueOf(fn).Pointer())
				return nil
			},
		},
	}, graph.TestModelOptions[graphAnySnap]{})
	require.NoError(t, err)
}

// JS: state tests > should test states
//
// expect.assertions(2) → exactly 2 state assertions run.
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/index.test.ts#L307
func TestIndex_StateTests_ShouldTestStates(t *testing.T) {
	// a (1)
	// a -> b (2)
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{"NEXT": {{Target: "b"}}}},
			{Key: "b"},
		},
	})

	model := graph.CreateTestModel(machine)

	assertions := 0
	graphTestModel(t, model, graph.TestParam[graphAnySnap]{
		States: map[string]func(graphAnySnap) error{
			"a": func(state graphAnySnap) error {
				assertions++
				assert.Equal(t, "a", state.Value)
				return nil
			},
			"b": func(state graphAnySnap) error {
				assertions++
				assert.Equal(t, "b", state.Value)
				return nil
			},
		},
	})
	assert.Equal(t, 2, assertions)
}

// JS: state tests > should test wildcard state for non-matching states
//
// expect.assertions(4) → exactly 4 state assertions run.
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/index.test.ts#L336
func TestIndex_StateTests_ShouldTestWildcardStateForNonMatchingStates(t *testing.T) {
	// a (1)
	// a -> b (2)
	// a -> c (2)
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{"NEXT": {{Target: "b"}}, "OTHER": {{Target: "c"}}}},
			{Key: "b"},
			{Key: "c"},
		},
	})

	model := graph.CreateTestModel(machine)

	assertions := 0
	graphTestModel(t, model, graph.TestParam[graphAnySnap]{
		States: map[string]func(graphAnySnap) error{
			"a": func(state graphAnySnap) error {
				assertions++
				assert.Equal(t, "a", state.Value)
				return nil
			},
			"b": func(state graphAnySnap) error {
				assertions++
				assert.Equal(t, "b", state.Value)
				return nil
			},
			"*": func(state graphAnySnap) error {
				assertions++
				assert.Equal(t, "c", state.Value)
				return nil
			},
		},
	})
	assert.Equal(t, 4, assertions)
}

// JS: state tests > should test nested states
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/index.test.ts#L370
func TestIndex_StateTests_ShouldTestNestedStates(t *testing.T) {
	var testedStateValues []string

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{"NEXT": {{Target: "b"}}}},
			{Key: "b", Initial: "b1", States: xs.States{
				{Key: "b1"},
			}},
		},
	})

	model := graph.CreateTestModel(machine)

	graphTestModel(t, model, graph.TestParam[graphAnySnap]{
		States: map[string]func(graphAnySnap) error{
			"a": func(state graphAnySnap) error {
				testedStateValues = append(testedStateValues, "a")
				assert.Equal(t, "a", state.Value)
				return nil
			},
			"b": func(state graphAnySnap) error {
				testedStateValues = append(testedStateValues, "b")
				assert.True(t, state.Matches("b"))
				return nil
			},
			"b.b1": func(state graphAnySnap) error {
				testedStateValues = append(testedStateValues, "b.b1")
				assert.Equal(t, map[string]any{"b": "b1"}, state.Value)
				return nil
			},
		},
	})
	assert.Equal(t, []string{"a", "b", "b.b1"}, testedStateValues)
}

// JS: state tests > should test with input
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/index.test.ts#L415
func TestIndex_StateTests_ShouldTestWithInput(t *testing.T) {
	type input struct{ Name string }
	type ctx struct {
		Name string `json:"name"`
	}
	type snap = *xs.MachineSnapshot[ctx]
	machine := xs.NewSetup[ctx](xs.Implementations{}).CreateMachine(xs.MachineConfig[ctx]{
		ContextFn: func(x xs.ContextArgs) ctx {
			return ctx{Name: x.Input.(input).Name}
		},
		Initial: "checking",
		States: xs.States{
			{Key: "checking", Always: xs.Transitions{
				{Guard: xs.GuardFunc(func(x xs.GuardArgs[ctx]) bool { return len(x.Context.Name) > 3 }), Target: "longName"},
				{Target: "shortName"},
			}},
			{Key: "longName"},
			{Key: "shortName"},
		},
	})

	model := graph.CreateTestModel(machine)

	stepValues := func(path graph.TestPath[snap]) []xs.StateValue {
		var out []xs.StateValue
		for _, s := range path.Steps {
			out = append(out, s.State.Value)
		}
		return out
	}

	path1 := model.GetShortestPaths(graph.GetPathOptions[snap]{
		TraversalOptions: graph.TraversalOptions[snap]{Input: input{Name: "ed"}},
	})
	require.NotEmpty(t, path1)
	assert.Equal(t, []xs.StateValue{"shortName"}, stepValues(path1[0]))

	path2 := model.GetShortestPaths(graph.GetPathOptions[snap]{
		TraversalOptions: graph.TraversalOptions[snap]{Input: input{Name: "edward"}},
	})
	require.NotEmpty(t, path2)
	assert.Equal(t, []xs.StateValue{"longName"}, stepValues(path2[0]))
}
