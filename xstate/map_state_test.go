package xstate_test

import (
	"testing"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mapState1Results mirrors `results.map((r) => r.result)`.
func mapState1Results[R any](results []xs.StateMapResult[R]) []R {
	out := make([]R, 0, len(results))
	for _, r := range results {
		out = append(out, r.Result)
	}
	return out
}

// JS: mapState > should map context from root state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/mapState.test.ts#L4
func TestMapState_ShouldMapContextFromRootState(t *testing.T) {
	type ctx struct{ Count int }
	machine := xs.NewSetup[ctx](xs.Implementations{}).CreateMachine(xs.MachineConfig[ctx]{
		Context: ctx{Count: 42},
		Initial: "a",
		States:  xs.States{{Key: "a"}},
	})

	snapshot := xs.CreateActor(machine).GetSnapshot()

	results := xs.MapSnapshot(snapshot, xs.StateMapper[ctx, int]{
		Map: func(s *xs.MachineSnapshot[ctx]) int { return s.Context.Count },
	})

	assert.Contains(t, mapState1Results(results), 42)
}

// JS: mapState > should map context from nested states
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/mapState.test.ts#L26
func TestMapState_ShouldMapContextFromNestedStates(t *testing.T) {
	type ctx struct{ Value string }
	machine := xs.NewSetup[ctx](xs.Implementations{}).CreateMachine(xs.MachineConfig[ctx]{
		Context: ctx{Value: "test"},
		Initial: "a",
		States: xs.States{
			{Key: "a", Initial: "one", States: xs.States{{Key: "one"}, {Key: "two"}}},
		},
	})

	snapshot := xs.CreateActor(machine).GetSnapshot()

	results := xs.MapSnapshot(snapshot, xs.StateMapper[ctx, string]{
		Map: func(s *xs.MachineSnapshot[ctx]) string { return "root:" + s.Context.Value },
		States: map[string]xs.StateMapper[ctx, string]{
			"a": {
				Map: func(s *xs.MachineSnapshot[ctx]) string { return "a:" + s.Context.Value },
				States: map[string]xs.StateMapper[ctx, string]{
					"one": {Map: func(s *xs.MachineSnapshot[ctx]) string { return "one:" + s.Context.Value }},
				},
			},
		},
	})

	// results.find((r) => r.stateNode.key === key)?.result
	find := func(key string) (string, bool) {
		for _, r := range results {
			if r.StateNode.Key == key {
				return r.Result, true
			}
		}
		return "", false
	}

	mapped := mapState1Results(results)
	assert.Contains(t, mapped, "root:test")
	rootResult, ok := find("(machine)")
	assert.True(t, ok)
	assert.Equal(t, "root:test", rootResult)
	aResult, ok := find("a")
	assert.True(t, ok)
	assert.Equal(t, "a:test", aResult)
	oneResult, ok := find("one")
	assert.True(t, ok)
	assert.Equal(t, "one:test", oneResult)
}

// JS: mapState > should only call mappers for active states
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/mapState.test.ts#L72
func TestMapState_ShouldOnlyCallMappersForActiveStates(t *testing.T) {
	type ctx struct{ X int }
	machine := xs.NewSetup[ctx](xs.Implementations{}).CreateMachine(xs.MachineConfig[ctx]{
		Context: ctx{X: 1},
		Initial: "a",
		States:  xs.States{{Key: "a"}, {Key: "b"}},
	})

	snapshot := xs.CreateActor(machine).GetSnapshot()

	results := xs.MapSnapshot(snapshot, xs.StateMapper[ctx, string]{
		Map: func(*xs.MachineSnapshot[ctx]) string { return "root" },
		States: map[string]xs.StateMapper[ctx, string]{
			"a": {Map: func(*xs.MachineSnapshot[ctx]) string { return "a" }},
			"b": {Map: func(*xs.MachineSnapshot[ctx]) string { return "b" }},
		},
	})

	mapped := mapState1Results(results)
	assert.Contains(t, mapped, "root")
	assert.Contains(t, mapped, "a")
	assert.NotContains(t, mapped, "b")
}

// JS: mapState > should work with parallel states
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/mapState.test.ts#L106
func TestMapState_ShouldWorkWithParallelStates(t *testing.T) {
	type ctx struct{ Val int }
	machine := xs.NewSetup[ctx](xs.Implementations{}).CreateMachine(xs.MachineConfig[ctx]{
		Context: ctx{Val: 100},
		Type:    xs.Parallel,
		States: xs.States{
			{Key: "region1", Initial: "x", States: xs.States{{Key: "x"}, {Key: "y"}}},
			{Key: "region2", Initial: "p", States: xs.States{{Key: "p"}, {Key: "q"}}},
		},
	})

	snapshot := xs.CreateActor(machine).GetSnapshot()

	results := xs.MapSnapshot(snapshot, xs.StateMapper[ctx, string]{
		Map: func(*xs.MachineSnapshot[ctx]) string { return "root" },
		States: map[string]xs.StateMapper[ctx, string]{
			"region1": {
				Map: func(*xs.MachineSnapshot[ctx]) string { return "region1" },
				States: map[string]xs.StateMapper[ctx, string]{
					"x": {Map: func(*xs.MachineSnapshot[ctx]) string { return "x" }},
				},
			},
			"region2": {
				Map: func(*xs.MachineSnapshot[ctx]) string { return "region2" },
				States: map[string]xs.StateMapper[ctx, string]{
					"p": {Map: func(*xs.MachineSnapshot[ctx]) string { return "p" }},
				},
			},
		},
	})

	mapped := mapState1Results(results)
	assert.Contains(t, mapped, "root")
	assert.Contains(t, mapped, "region1")
	assert.Contains(t, mapped, "x")
	assert.Contains(t, mapped, "region2")
	assert.Contains(t, mapped, "p")
	assert.Len(t, results, 5)
}

// JS: mapState > should handle states without mappers
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/mapState.test.ts#L165
func TestMapState_ShouldHandleStatesWithoutMappers(t *testing.T) {
	type ctx struct{ N int }
	machine := xs.NewSetup[ctx](xs.Implementations{}).CreateMachine(xs.MachineConfig[ctx]{
		Context: ctx{N: 5},
		Initial: "a",
		States: xs.States{
			{Key: "a", Initial: "one", States: xs.States{{Key: "one"}}},
		},
	})

	snapshot := xs.CreateActor(machine).GetSnapshot()

	results := xs.MapSnapshot(snapshot, xs.StateMapper[ctx, string]{
		Map: func(*xs.MachineSnapshot[ctx]) string { return "root" },
		States: map[string]xs.StateMapper[ctx, string]{
			"a": {
				Map: func(*xs.MachineSnapshot[ctx]) string { return "a" },
				States: map[string]xs.StateMapper[ctx, string]{
					"one": {Map: func(*xs.MachineSnapshot[ctx]) string { return "one" }},
				},
			},
		},
	})

	mapped := mapState1Results(results)
	assert.Contains(t, mapped, "root")
	assert.Contains(t, mapped, "a")
	assert.Contains(t, mapped, "one")
	assert.Len(t, results, 3)
}

// JS: mapState > should work with final states
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/mapState.test.ts#L206
func TestMapState_ShouldWorkWithFinalStates(t *testing.T) {
	machine := xs.NewSetup[any](xs.Implementations{}).CreateMachine(xs.MachineConfig[any]{
		Initial: "active",
		States: xs.States{
			{Key: "active", On: map[string]xs.Transitions{"DONE": {{Target: "finished"}}}},
			{Key: "finished", Type: xs.Final},
		},
	})

	actor := xs.CreateActor(machine)
	actor.Start()
	actor.Send(xs.Ev("DONE"))
	snapshot := actor.GetSnapshot()

	results := xs.MapSnapshot(snapshot, xs.StateMapper[any, string]{
		Map: func(*xs.MachineSnapshot[any]) string { return "root" },
		States: map[string]xs.StateMapper[any, string]{
			"finished": {Map: func(*xs.MachineSnapshot[any]) string { return "finished" }},
		},
	})

	mapped := mapState1Results(results)
	assert.Contains(t, mapped, "root")
	assert.Contains(t, mapped, "finished")
}

// JS: mapState > should include stateNode in results
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/mapState.test.ts#L238
func TestMapState_ShouldIncludeStateNodeInResults(t *testing.T) {
	machine := xs.NewSetup[any](xs.Implementations{}).CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", Initial: "one", States: xs.States{{Key: "one"}}},
		},
	})

	snapshot := xs.CreateActor(machine).GetSnapshot()

	results := xs.MapSnapshot(snapshot, xs.StateMapper[any, string]{
		Map: func(*xs.MachineSnapshot[any]) string { return "root" },
		States: map[string]xs.StateMapper[any, string]{
			"a": {
				Map: func(*xs.MachineSnapshot[any]) string { return "a" },
				States: map[string]xs.StateMapper[any, string]{
					"one": {Map: func(*xs.MachineSnapshot[any]) string { return "one" }},
				},
			},
		},
	})

	// JS indexes results[0..2]; an out-of-range index would throw there too.
	require.GreaterOrEqual(t, len(results), 3)
	assert.Equal(t, "one", results[0].StateNode.Key)
	assert.Equal(t, "one", results[0].Result)
	assert.Equal(t, "a", results[1].StateNode.Key)
	assert.Equal(t, "a", results[1].Result)
	// toEqual([]): the root path has no segments (nil or empty slice).
	assert.Empty(t, results[2].StateNode.Path)
	assert.Equal(t, "root", results[2].Result)
}

// JS: mapState > type safety > should accept valid state keys
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/mapState.test.ts#L276
func TestMapState_TypeSafety_ShouldAcceptValidStateKeys(t *testing.T) {
	type ctx struct{ Foo string }
	machine := xs.NewSetup[ctx](xs.Implementations{}).CreateMachine(xs.MachineConfig[ctx]{
		Context: ctx{Foo: "bar"},
		Initial: "idle",
		States:  xs.States{{Key: "idle"}, {Key: "loading"}, {Key: "success"}},
	})

	snapshot := xs.CreateActor(machine).GetSnapshot()

	foo := func(s *xs.MachineSnapshot[ctx]) string { return s.Context.Foo }
	// This should compile (and run) without errors.
	assert.NotPanics(t, func() {
		xs.MapSnapshot(snapshot, xs.StateMapper[ctx, string]{
			Map: foo,
			States: map[string]xs.StateMapper[ctx, string]{
				"idle":    {Map: foo},
				"loading": {Map: foo},
				"success": {Map: foo},
			},
		})
	})
}

// JS: mapState > type safety > should error on invalid state keys
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/mapState.test.ts#L310
func TestMapState_TypeSafety_ShouldErrorOnInvalidStateKeys(t *testing.T) {
	// The JS `@ts-expect-error` on the 'nonexistent' key has no Go equivalent
	// (StateMapper.States is map[string]); the runtime call is ported.
	type ctx struct{ Foo string }
	machine := xs.NewSetup[ctx](xs.Implementations{}).CreateMachine(xs.MachineConfig[ctx]{
		Context: ctx{Foo: "bar"},
		Initial: "idle",
		States:  xs.States{{Key: "idle"}, {Key: "loading"}},
	})

	snapshot := xs.CreateActor(machine).GetSnapshot()

	foo := func(s *xs.MachineSnapshot[ctx]) string { return s.Context.Foo }
	assert.NotPanics(t, func() {
		xs.MapSnapshot(snapshot, xs.StateMapper[ctx, string]{
			Map: foo,
			States: map[string]xs.StateMapper[ctx, string]{
				"idle":        {Map: foo},
				"nonexistent": {Map: foo},
			},
		})
	})
}

// JS: mapState > type safety > should error on invalid nested state keys
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/mapState.test.ts#L340
func TestMapState_TypeSafety_ShouldErrorOnInvalidNestedStateKeys(t *testing.T) {
	// The JS `@ts-expect-error` on the 'invalidChild' key has no Go equivalent
	// (StateMapper.States is map[string]); the runtime call is ported.
	type ctx struct{ Val int }
	machine := xs.NewSetup[ctx](xs.Implementations{}).CreateMachine(xs.MachineConfig[ctx]{
		Context: ctx{Val: 0},
		Initial: "parent",
		States: xs.States{
			{Key: "parent", Initial: "child1", States: xs.States{{Key: "child1"}, {Key: "child2"}}},
		},
	})

	snapshot := xs.CreateActor(machine).GetSnapshot()

	val := func(s *xs.MachineSnapshot[ctx]) int { return s.Context.Val }
	assert.NotPanics(t, func() {
		xs.MapSnapshot(snapshot, xs.StateMapper[ctx, int]{
			Map: val,
			States: map[string]xs.StateMapper[ctx, int]{
				"parent": {
					Map: val,
					States: map[string]xs.StateMapper[ctx, int]{
						"child1":       {Map: val},
						"invalidChild": {Map: val},
					},
				},
			},
		})
	})
}

// JS: mapState > type safety > should infer snapshot type in map function
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/mapState.test.ts#L380
func TestMapState_TypeSafety_ShouldInferSnapshotTypeInMapFunction(t *testing.T) {
	type ctx struct {
		Count int
		Name  string
	}
	type result struct {
		N int
		S string
	}
	machine := xs.NewSetup[ctx](xs.Implementations{}).CreateMachine(xs.MachineConfig[ctx]{
		Context: ctx{Count: 0, Name: "test"},
		Initial: "idle",
		States:  xs.States{{Key: "idle"}},
	})

	snapshot := xs.CreateActor(machine).GetSnapshot()

	assert.NotPanics(t, func() {
		xs.MapSnapshot(snapshot, xs.StateMapper[ctx, result]{
			Map: func(s *xs.MachineSnapshot[ctx]) result {
				// These should all be valid (checked statically by Go).
				var n int = s.Context.Count
				var str string = s.Context.Name
				return result{N: n, S: str}
			},
		})
	})
}

// JS: mapState > type safety > should enforce consistent TResult type across all map functions
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/mapState.test.ts#L405
func TestMapState_TypeSafety_ShouldEnforceConsistentTResultTypeAcrossAllMapFunctions(t *testing.T) {
	type ctx struct{ Count int }
	machine := xs.NewSetup[ctx](xs.Implementations{}).CreateMachine(xs.MachineConfig[ctx]{
		Context: ctx{Count: 0},
		Initial: "a",
		States: xs.States{
			{Key: "a", Initial: "one", States: xs.States{{Key: "one"}}},
		},
	})

	snapshot := xs.CreateActor(machine).GetSnapshot()

	// All returning int (mapState<typeof snapshot, number>) - should work.
	assert.NotPanics(t, func() {
		xs.MapSnapshot(snapshot, xs.StateMapper[ctx, int]{
			Map: func(*xs.MachineSnapshot[ctx]) int { return 42 },
			States: map[string]xs.StateMapper[ctx, int]{
				"a": {
					Map: func(*xs.MachineSnapshot[ctx]) int { return 100 },
					States: map[string]xs.StateMapper[ctx, int]{
						"one": {Map: func(*xs.MachineSnapshot[ctx]) int { return 200 }},
					},
				},
			},
		})
	})
}

// JS: mapState > type safety > should error when nested map returns wrong type
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/mapState.test.ts#L441
func TestMapState_TypeSafety_ShouldErrorWhenNestedMapReturnsWrongType(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error that a nested map returning boolean is rejected when TResult is number; Go generics reject it at compile time, so it cannot be written")
}

// JS: mapState > type safety > should error when deeply nested map returns wrong type
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/mapState.test.ts#L467
func TestMapState_TypeSafety_ShouldErrorWhenDeeplyNestedMapReturnsWrongType(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error that a deeply nested map returning number is rejected when TResult is string; Go generics reject it at compile time, so it cannot be written")
}

// JS: mapState > type safety > should infer result type in return value
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/mapState.test.ts#L503
func TestMapState_TypeSafety_ShouldInferResultTypeInReturnValue(t *testing.T) {
	// The `@ts-expect-error results[0].result satisfies string` check has no
	// Go equivalent; `satisfies number` is the static assignment below.
	machine := xs.NewSetup[any](xs.Implementations{}).CreateMachine(xs.MachineConfig[any]{
		Initial: "idle",
		States:  xs.States{{Key: "idle"}},
	})

	snapshot := xs.CreateActor(machine).GetSnapshot()

	results := xs.MapSnapshot(snapshot, xs.StateMapper[any, int]{
		Map: func(*xs.MachineSnapshot[any]) int { return 42 },
	})

	// result should be typed as int, not any.
	require.NotEmpty(t, results)
	var _ int = results[0].Result
}
