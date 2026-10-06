package xstate_test

import (
	"context"
	"testing"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/assert"
)

// Regression tests for: Bug #5454
// `initialTransition` fails when invoke has `systemId`.

// JS: initialTransition / transition with invoke systemId (issue #5454) > does not throw when the initial state has an invoke with systemId
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/issue5454.test.ts#L20
func TestIssue5454_DoesNotThrowWhenInitialStateHasInvokeWithSystemID(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "idle",
		States: xs.States{
			{Key: "idle", Invoke: []xs.InvokeConfig{{
				Logic:    xs.FromPromise(func(_ context.Context, _ xs.PromiseArgs) (int, error) { return 42, nil }),
				SystemID: "myActor",
			}}},
		},
	})

	assert.NotPanics(t, func() { xs.InitialTransition(machine) })
}

// JS: initialTransition / transition with invoke systemId (issue #5454) > returns the correct initial snapshot when invoke has systemId
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/issue5454.test.ts#L36
func TestIssue5454_ReturnsCorrectInitialSnapshotWhenInvokeHasSystemID(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "idle",
		States: xs.States{
			{Key: "idle", Invoke: []xs.InvokeConfig{{
				Logic:    xs.FromPromise(func(_ context.Context, _ xs.PromiseArgs) (int, error) { return 42, nil }),
				SystemID: "myActor",
			}}},
		},
	})

	snapshot, actions := xs.InitialTransition(machine)
	assert.Equal(t, "idle", snapshot.Value)
	assert.Len(t, actions, 1) // the spawnChild action for the invoke
}

// JS: initialTransition / transition with invoke systemId (issue #5454) > is idempotent: repeated calls do not throw
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/issue5454.test.ts#L54
func TestIssue5454_IsIdempotentRepeatedCallsDoNotThrow(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "idle",
		States: xs.States{
			{Key: "idle", Invoke: []xs.InvokeConfig{{
				Logic:    xs.FromPromise(func(_ context.Context, _ xs.PromiseArgs) (int, error) { return 42, nil }),
				SystemID: "myActor",
			}}},
		},
	})

	assert.NotPanics(t, func() {
		xs.InitialTransition(machine)
		xs.InitialTransition(machine)
		xs.InitialTransition(machine)
	})
}

// JS: initialTransition / transition with invoke systemId (issue #5454) > transition() does not throw when the target state has an invoke with systemId
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/issue5454.test.ts#L74
func TestIssue5454_TransitionDoesNotThrowWhenTargetStateHasInvokeWithSystemID(t *testing.T) {
	countMachine := xs.FromTransition(
		func(s int, e xs.Event, _ *xs.ActorScope) int {
			if e.EventType() == "INC" {
				return s + 1
			}
			return s
		},
		func(_ xs.TransitionInitArgs) int { return 0 },
	)

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "idle",
		States: xs.States{
			{Key: "idle", On: map[string]xs.Transitions{"START": {{Target: "running"}}}},
			{Key: "running", Invoke: []xs.InvokeConfig{{
				Logic:    countMachine,
				SystemID: "counter",
			}}},
		},
	})

	initial, _ := xs.InitialTransition(machine)

	assert.NotPanics(t, func() { xs.Transition(machine, initial, xs.Ev("START")) })
}

// JS: initialTransition / transition with invoke systemId (issue #5454) > works with multiple invokes each having a distinct systemId
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/issue5454.test.ts#L100
func TestIssue5454_WorksWithMultipleInvokesEachHavingDistinctSystemID(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "idle",
		States: xs.States{
			{Key: "idle", Invoke: []xs.InvokeConfig{
				{
					Logic:    xs.FromPromise(func(_ context.Context, _ xs.PromiseArgs) (int, error) { return 1, nil }),
					SystemID: "actorOne",
				},
				{
					Logic:    xs.FromPromise(func(_ context.Context, _ xs.PromiseArgs) (int, error) { return 2, nil }),
					SystemID: "actorTwo",
				},
			}},
		},
	})

	assert.NotPanics(t, func() { xs.InitialTransition(machine) })
	snapshot, _ := xs.InitialTransition(machine)
	assert.Equal(t, "idle", snapshot.Value)
}
