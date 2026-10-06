package graph_test

import (
	"testing"

	"github.com/nguyenvanduocit/go-xstate/graph"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/require"
)

// graphAnySnap is the snapshot type of a machine without context.
type graphAnySnap = *xs.MachineSnapshot[any]

// graphTestModel mirrors testUtils.testModel: tests every shortest path.
func graphTestModel[S xs.Snapshot](t *testing.T, model *graph.TestModel[S], params graph.TestParam[S]) {
	t.Helper()
	for _, path := range model.GetShortestPaths() {
		_, err := path.Test(params)
		require.NoError(t, err)
	}
}

// graphTestPaths mirrors testUtils.testPaths.
func graphTestPaths[S xs.Snapshot](t *testing.T, paths []graph.TestPath[S], params graph.TestParam[S]) {
	t.Helper()
	for _, path := range paths {
		_, err := path.Test(params)
		require.NoError(t, err)
	}
}

// graphStepSnap / graphPathSnap mirror the result of getPathSnapshot() in
// graph.test.ts: machine snapshots are reduced to their value, other
// snapshots to their context.
type graphStepSnap struct {
	State     any
	EventType string
}

type graphPathSnap struct {
	State any
	Steps []graphStepSnap
}

func graphPathSnapshot[S xs.Snapshot](path graph.StatePath[S], stateOf func(S) any) graphPathSnap {
	steps := make([]graphStepSnap, 0, len(path.Steps))
	for _, step := range path.Steps {
		steps = append(steps, graphStepSnap{State: stateOf(step.State), EventType: step.Event.EventType()})
	}
	return graphPathSnap{State: stateOf(path.State), Steps: steps}
}

func graphPathsSnapshot[S xs.Snapshot](paths []graph.StatePath[S], stateOf func(S) any) []graphPathSnap {
	out := make([]graphPathSnap, 0, len(paths))
	for _, p := range paths {
		out = append(out, graphPathSnapshot(p, stateOf))
	}
	return out
}

// graphMachineValue reduces a machine snapshot to its value.
func graphMachineValue[C any](s *xs.MachineSnapshot[C]) any { return s.Value }

// graphTransitionContext reduces a transition snapshot to its context.
func graphTransitionContext[T any](s *xs.TransitionSnapshot[T]) any { return s.Context }

// graphEventTypes returns the event types of the steps of a path.
func graphEventTypes[S xs.Snapshot](steps []graph.Step[S]) []string {
	out := make([]string, 0, len(steps))
	for _, s := range steps {
		out = append(out, s.Event.EventType())
	}
	return out
}

// graphDescriptions returns the descriptions of test paths.
func graphDescriptions[S xs.Snapshot](paths []graph.TestPath[S]) []string {
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		out = append(out, p.Description)
	}
	return out
}
