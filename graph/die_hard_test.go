package graph_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nguyenvanduocit/go-xstate/graph"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

type dieHardCtx struct {
	Three int `json:"three"`
	Five  int `json:"five"`
}

type dieHardSnap = *xs.MachineSnapshot[dieHardCtx]

// dieHardJugs mirrors the Jugs class (the system under test).
type dieHardJugs struct {
	three int
	five  int
}

func (j *dieHardJugs) fillThree()  { j.three = 3 }
func (j *dieHardJugs) fillFive()   { j.five = 5 }
func (j *dieHardJugs) emptyThree() { j.three = 0 }
func (j *dieHardJugs) emptyFive()  { j.five = 0 }
func (j *dieHardJugs) transferThree() {
	poured := min(5-j.five, j.three)

	j.three = j.three - poured
	j.five = j.five + poured
}
func (j *dieHardJugs) transferFive() {
	poured := min(3-j.three, j.five)

	j.three = j.three + poured
	j.five = j.five - poured
}

// dieHardModel mirrors createDieHardModel().model.
func dieHardModel() *graph.TestModel[dieHardSnap] {
	setContext := func(fn func(c dieHardCtx) dieHardCtx) xs.Action {
		return xs.Assign(func(a xs.AssignArgs[dieHardCtx]) dieHardCtx { return fn(a.Context) })
	}
	dieHardMachine := xs.CreateMachine(xs.MachineConfig[dieHardCtx]{
		ID:      "dieHard",
		Initial: "pending",
		Context: dieHardCtx{Three: 0, Five: 0},
		States: xs.States{
			{
				Key: "pending",
				Always: xs.Transitions{{
					Target: "success",
					Guard:  xs.GuardRef{Type: "weHave4Gallons"},
				}},
				On: map[string]xs.Transitions{
					"POUR_3_TO_5": {{Actions: xs.Actions{setContext(func(c dieHardCtx) dieHardCtx {
						poured := min(5-c.Five, c.Three)

						return dieHardCtx{Three: c.Three - poured, Five: c.Five + poured}
					})}}},
					"POUR_5_TO_3": {{Actions: xs.Actions{setContext(func(c dieHardCtx) dieHardCtx {
						poured := min(3-c.Three, c.Five)

						res := dieHardCtx{Three: c.Three + poured, Five: c.Five - poured}

						return res
					})}}},
					"FILL_3": {{Actions: xs.Actions{setContext(func(c dieHardCtx) dieHardCtx {
						c.Three = 3
						return c
					})}}},
					"FILL_5": {{Actions: xs.Actions{setContext(func(c dieHardCtx) dieHardCtx {
						c.Five = 5
						return c
					})}}},
					"EMPTY_3": {{Actions: xs.Actions{setContext(func(c dieHardCtx) dieHardCtx {
						c.Three = 0
						return c
					})}}},
					"EMPTY_5": {{Actions: xs.Actions{setContext(func(c dieHardCtx) dieHardCtx {
						c.Five = 0
						return c
					})}}},
				},
			},
			{Key: "success", Type: xs.Final},
		},
	}, xs.Implementations{
		Guards: map[string]xs.Guard{
			"weHave4Gallons": xs.GuardFunc(func(a xs.GuardArgs[dieHardCtx]) bool { return a.Context.Five == 4 }),
		},
	})

	return graph.CreateTestModel(dieHardMachine)
}

// dieHardOptions mirrors createDieHardModel().options, bound to jugs.
func dieHardOptions(t *testing.T, jugs *dieHardJugs) graph.TestParam[dieHardSnap] {
	exec := func(fn func()) graph.EventExecutor[dieHardSnap] {
		return func(graph.Step[dieHardSnap]) error {
			fn()
			return nil
		}
	}
	return graph.TestParam[dieHardSnap]{
		States: map[string]func(dieHardSnap) error{
			"pending": func(state dieHardSnap) error {
				assert.NotEqual(t, 4, jugs.five)
				assert.Equal(t, state.Context.Three, jugs.three)
				assert.Equal(t, state.Context.Five, jugs.five)
				return nil
			},
			"success": func(dieHardSnap) error {
				assert.Equal(t, 4, jugs.five)
				return nil
			},
		},
		Events: map[string]graph.EventExecutor[dieHardSnap]{
			"POUR_3_TO_5": exec(jugs.transferThree),
			"POUR_5_TO_3": exec(jugs.transferFive),
			"EMPTY_3":     exec(jugs.emptyThree),
			"EMPTY_5":     exec(jugs.emptyFive),
			"FILL_3":      exec(jugs.fillThree),
			"FILL_5":      exec(jugs.fillFive),
		},
	}
}

// dieHardTestPaths runs model.testPath(path, options) for every path in a
// subtest with fresh jugs (JS: one generated `it` per path + beforeEach).
func dieHardTestPaths(t *testing.T, model *graph.TestModel[dieHardSnap], paths []graph.TestPath[dieHardSnap]) {
	for _, path := range paths {
		t.Run(path.Description, func(t *testing.T) {
			jugs := &dieHardJugs{}
			_, err := model.TestPath(path.StatePath, dieHardOptions(t, jugs))
			require.NoError(t, err)
		})
	}
}

func dieHardMatchesSuccess(state dieHardSnap) bool { return state.Matches("success") }

// JS: die hard example > testing a model (shortestPathsTo) > should generate the right number of paths
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/dieHard.test.ts#L156
func TestDieHard_ShortestPathsTo_ShouldGenerateTheRightNumberOfPaths(t *testing.T) {
	paths := dieHardModel().GetShortestPaths(graph.GetPathOptions[dieHardSnap]{
		TraversalOptions: graph.TraversalOptions[dieHardSnap]{ToState: dieHardMatchesSuccess},
	})

	assert.Equal(t, 2, len(paths))
}

// JS: die hard example > testing a model (shortestPathsTo) > path ${getDescription(path.state)} > path ${getDescription(path.state)}
//
// One JS `it` generated per path; one subtest per path here.
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/dieHard.test.ts#L162
func TestDieHard_ShortestPathsTo_Path(t *testing.T) {
	model := dieHardModel()
	paths := model.GetShortestPaths(graph.GetPathOptions[dieHardSnap]{
		TraversalOptions: graph.TraversalOptions[dieHardSnap]{ToState: dieHardMatchesSuccess},
	})

	dieHardTestPaths(t, model, paths)
}

// JS: die hard example > testing a model (simplePathsTo) > should generate the right number of paths
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/dieHard.test.ts#L175
func TestDieHard_SimplePathsTo_ShouldGenerateTheRightNumberOfPaths(t *testing.T) {
	paths := dieHardModel().GetSimplePaths(graph.GetPathOptions[dieHardSnap]{
		TraversalOptions: graph.TraversalOptions[dieHardSnap]{ToState: dieHardMatchesSuccess},
	})

	assert.Equal(t, 14, len(paths))
}

// JS: die hard example > testing a model (simplePathsTo) > reaches state ${value} (${context}) > path ${getDescription(path.state)}
//
// One JS `it` generated per path; one subtest per path here.
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/dieHard.test.ts#L183
func TestDieHard_SimplePathsTo_ReachesStatePath(t *testing.T) {
	model := dieHardModel()
	paths := model.GetSimplePaths(graph.GetPathOptions[dieHardSnap]{
		TraversalOptions: graph.TraversalOptions[dieHardSnap]{ToState: dieHardMatchesSuccess},
	})

	dieHardTestPaths(t, model, paths)
}

// JS: die hard example > testing a model (getPathFromEvents) > reaches state ${value} (${context}) > path ${getDescription(path.state)}
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/dieHard.test.ts#L208
func TestDieHard_GetPathFromEvents_ReachesStatePath(t *testing.T) {
	model := dieHardModel()

	paths := model.GetPathsFromEvents(
		[]xs.Event{
			xs.Ev("FILL_5"),
			xs.Ev("POUR_5_TO_3"),
			xs.Ev("EMPTY_3"),
			xs.Ev("POUR_5_TO_3"),
			xs.Ev("FILL_5"),
			xs.Ev("POUR_5_TO_3"),
		},
		graph.GetPathOptions[dieHardSnap]{
			TraversalOptions: graph.TraversalOptions[dieHardSnap]{ToState: dieHardMatchesSuccess},
		},
	)
	require.NotEmpty(t, paths)

	dieHardTestPaths(t, model, paths[:1])
}

// JS: die hard example > testing a model (getPathFromEvents) > should return no paths if the target does not match the last entered state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/dieHard.test.ts#L213
func TestDieHard_GetPathFromEvents_ShouldReturnNoPathsIfTheTargetDoesNotMatchTheLastEnteredState(t *testing.T) {
	paths := dieHardModel().GetPathsFromEvents(
		[]xs.Event{xs.Ev("FILL_5")},
		graph.GetPathOptions[dieHardSnap]{
			TraversalOptions: graph.TraversalOptions[dieHardSnap]{ToState: dieHardMatchesSuccess},
		},
	)

	assert.Len(t, paths, 0)
}

func dieHardSuccessWithThreeEmpty(state dieHardSnap) bool {
	return state.Matches("success") && state.Context.Three == 0
}

// JS: die hard example > .testPath(path) > should generate the right number of paths
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/dieHard.test.ts#L233
func TestDieHard_TestPath_ShouldGenerateTheRightNumberOfPaths(t *testing.T) {
	paths := dieHardModel().GetSimplePaths(graph.GetPathOptions[dieHardSnap]{
		TraversalOptions: graph.TraversalOptions[dieHardSnap]{ToState: dieHardSuccessWithThreeEmpty},
	})

	assert.Equal(t, 6, len(paths))
}

// JS: die hard example > .testPath(path) > reaches state ${value} (${context}) > path ${getDescription(path.state)} > reaches the target state
//
// One JS `it` generated per path; one subtest per path here.
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/dieHard.test.ts#L242
func TestDieHard_TestPath_ReachesTheTargetState(t *testing.T) {
	model := dieHardModel()
	paths := model.GetSimplePaths(graph.GetPathOptions[dieHardSnap]{
		TraversalOptions: graph.TraversalOptions[dieHardSnap]{ToState: dieHardSuccessWithThreeEmpty},
	})

	dieHardTestPaths(t, model, paths)
}

func dieHardTraceModel() *graph.TestModel[graphAnySnap] {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "first",
		States: xs.States{
			{Key: "first", On: map[string]xs.Transitions{"NEXT_1": {{Target: "second"}}}},
			{Key: "second", On: map[string]xs.Transitions{"NEXT_2": {{Target: "third"}}}},
			{Key: "third"},
		},
	})

	return graph.CreateTestModel(machine)
}

func dieHardToThird() graph.GetPathOptions[graphAnySnap] {
	return graph.GetPathOptions[graphAnySnap]{
		TraversalOptions: graph.TraversalOptions[graphAnySnap]{
			ToState: func(state graphAnySnap) bool { return state.Matches("third") },
		},
	}
}

// JS: error path trace > should return trace for failed state > should generate the right number of paths
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/dieHard.test.ts#L267
func TestDieHard_ErrorPathTrace_ShouldGenerateTheRightNumberOfPaths(t *testing.T) {
	testModel := dieHardTraceModel()

	assert.Equal(t, 1, len(testModel.GetShortestPaths(dieHardToThird())))
}

// JS: error path trace > should return trace for failed state > should show an error path trace
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/dieHard.test.ts#L275
func TestDieHard_ErrorPathTrace_ShouldShowAnErrorPathTrace(t *testing.T) {
	testModel := dieHardTraceModel()

	paths := testModel.GetShortestPaths(dieHardToThird())
	require.NotEmpty(t, paths)
	path := paths[0]

	_, err := testModel.TestPath(path.StatePath, graph.TestParam[graphAnySnap]{
		States: map[string]func(graphAnySnap) error{
			"third": func(graphAnySnap) error {
				return errors.New("test error")
			},
		},
	})
	require.Error(t, err, "Should have failed")

	assert.Contains(t, err.Error(), "test error")
	assert.Equal(t, "test error\n"+
		"Path:\n"+
		"\tState: {\"value\":\"first\"}\n"+
		"\tEvent: {\"type\":\"xstate.init\"}\n"+
		"\n"+
		"\tState: {\"value\":\"second\"} via {\"type\":\"xstate.init\"}\n"+
		"\tEvent: {\"type\":\"NEXT_1\"}\n"+
		"\n"+
		"\tState: {\"value\":\"third\"} via {\"type\":\"NEXT_1\"}\n"+
		"\tEvent: {\"type\":\"NEXT_2\"}\n"+
		"\n"+
		"\tState: {\"value\":\"third\"} via {\"type\":\"NEXT_2\"}", err.Error())
}
