package xstate_test

import (
	"strings"
	"sync"
	"testing"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// final1TrackEntries mirrors trackEntries from test/utils.ts: it prepends
// entry/exit tracking actions to every state node of the machine and returns a
// flush function that yields (and clears) the recorded log.
func final1TrackEntries[C any](machine *xs.StateMachine[C]) func() []string {
	var mu sync.Mutex
	logs := []string{}

	addTrackingActions := func(state *xs.StateNode, stateDescription string) {
		state.Entry = append(xs.Actions{xs.ActionFunc(func(xs.ActionArgs[C]) {
			mu.Lock()
			defer mu.Unlock()
			logs = append(logs, "enter: "+stateDescription)
		})}, state.Entry...)
		state.Exit = append(xs.Actions{xs.ActionFunc(func(xs.ActionArgs[C]) {
			mu.Lock()
			defer mu.Unlock()
			logs = append(logs, "exit: "+stateDescription)
		})}, state.Exit...)
	}

	var addTrackingActionsRecursively func(state *xs.StateNode)
	addTrackingActionsRecursively = func(state *xs.StateNode) {
		for _, child := range state.ChildStates() {
			addTrackingActions(child, strings.Join(child.Path, "."))
			addTrackingActionsRecursively(child)
		}
	}

	addTrackingActions(machine.RootNode(), "__root__")
	addTrackingActionsRecursively(machine.RootNode())

	return func() []string {
		mu.Lock()
		defer mu.Unlock()
		flushed := logs
		logs = []string{}
		return flushed
	}
}

// JS: final states > status of a machine with a root state being final should be done
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/final.test.ts#L11
func TestFinal_StatusOfMachineWithRootStateBeingFinalShouldBeDone(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{Type: xs.Final})
	actorRef := xs.CreateActor(machine).Start()

	assert.Equal(t, xs.StatusDone, actorRef.GetSnapshot().Status)
}

// JS: final states > output of a machine with a root state being final should be called with a "xstate.done.state.ROOT_ID" event
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/final.test.ts#L17
func TestFinal_OutputOfMachineWithRootStateBeingFinalCalledWithDoneStateRootEvent(t *testing.T) {
	spy := newSpy()
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Type: xs.Final,
		Output: xs.NewExpr(func(a xs.ExprArgs[any]) any {
			spy.Call(a.Event)
			return nil
		}),
	})
	xs.CreateActor(machine, xs.WithInput(42)).Start()

	assert.Equal(t, [][]any{
		{xs.DoneStateEvent{StateID: "(machine)", Output: nil}},
	}, spy.Calls())
}

// JS: final states > should emit the "xstate.done.state.*" event when all nested states are in their final states
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/final.test.ts#L38
func TestFinal_ShouldEmitDoneStateEventWhenAllNestedStatesAreInFinalStates(t *testing.T) {
	onDoneSpy := newSpy()

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "m",
		Initial: "foo",
		States: xs.States{
			{
				Key:  "foo",
				Type: xs.Parallel,
				States: xs.States{
					{
						Key:     "first",
						Initial: "a",
						States: xs.States{
							{Key: "a", On: map[string]xs.Transitions{"NEXT_1": {{Target: "b"}}}},
							{Key: "b", Type: xs.Final},
						},
					},
					{
						Key:     "second",
						Initial: "a",
						States: xs.States{
							{Key: "a", On: map[string]xs.Transitions{"NEXT_2": {{Target: "b"}}}},
							{Key: "b", Type: xs.Final},
						},
					},
				},
				OnDone: xs.Transitions{{
					Target: "bar",
					Actions: xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[any]) {
						onDoneSpy.Call(a.Event.EventType())
					})},
				}},
			},
			{Key: "bar"},
		},
	})

	actor := xs.CreateActor(machine).Start()

	actor.Send(xs.Ev("NEXT_1"))
	actor.Send(xs.Ev("NEXT_2"))

	assert.Equal(t, "bar", actor.GetSnapshot().Value)
	assert.Contains(t, onDoneSpy.Calls(), []any{"xstate.done.state.m.foo"})
}

// JS: final states > should execute final child state actions first
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/final.test.ts#L95
func TestFinal_ShouldExecuteFinalChildStateActionsFirst(t *testing.T) {
	var mu sync.Mutex
	actual := []string{}
	push := func(s string) {
		mu.Lock()
		defer mu.Unlock()
		actual = append(actual, s)
	}

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "foo",
		States: xs.States{
			{
				Key:     "foo",
				Initial: "bar",
				OnDone: xs.Transitions{{Actions: xs.Actions{xs.ActionFunc(func(xs.ActionArgs[any]) {
					push("fooAction")
				})}}},
				States: xs.States{
					{
						Key:     "bar",
						Initial: "baz",
						OnDone:  xs.Transitions{{Target: "barFinal"}},
						States: xs.States{
							{
								Key:  "baz",
								Type: xs.Final,
								Entry: xs.Actions{xs.ActionFunc(func(xs.ActionArgs[any]) {
									push("bazAction")
								})},
							},
						},
					},
					{
						Key:  "barFinal",
						Type: xs.Final,
						Entry: xs.Actions{xs.ActionFunc(func(xs.ActionArgs[any]) {
							push("barAction")
						})},
					},
				},
			},
		},
	})

	xs.CreateActor(machine).Start()

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, []string{"bazAction", "barAction", "fooAction"}, actual)
}

// JS: final states > should call output expressions on nested final nodes
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/final.test.ts#L128
func TestFinal_ShouldCallOutputExpressionsOnNestedFinalNodes(t *testing.T) {
	sig := newSignal()

	type ctx struct{ RevealedSecret string }

	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Initial: "secret",
		Context: ctx{RevealedSecret: ""}, // JS: revealedSecret: undefined
		States: xs.States{
			{
				Key:     "secret",
				Initial: "wait",
				States: xs.States{
					{Key: "wait", On: map[string]xs.Transitions{"REQUEST_SECRET": {{Target: "reveal"}}}},
					{
						Key:  "reveal",
						Type: xs.Final,
						Output: xs.NewExpr(func(xs.ExprArgs[ctx]) any {
							return map[string]any{"secret": "the secret"}
						}),
					},
				},
				OnDone: xs.Transitions{{
					Target: "success",
					Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
						c := a.Context
						c.RevealedSecret = a.Event.(xs.DoneStateEvent).Output.(map[string]any)["secret"].(string)
						return c
					})},
				}},
			},
			{Key: "success", Type: xs.Final},
		},
	})

	service := xs.CreateActor(machine)
	service.Subscribe(xs.Observer[*xs.MachineSnapshot[ctx]]{
		Complete: func() {
			assert.Equal(t, ctx{RevealedSecret: "the secret"}, service.GetSnapshot().Context)
			sig.Resolve()
		},
	})
	service.Start()

	service.Send(xs.Ev("REQUEST_SECRET"))

	sig.Wait(t)
}

// JS: final states > should only call data expression once when entering root's final state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/final.test.ts#L188
func TestFinal_ShouldOnlyCallDataExpressionOnceWhenEnteringRootsFinalState(t *testing.T) {
	spy := newSpy()
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "start",
		States: xs.States{
			{Key: "start", On: map[string]xs.Transitions{"FINISH": {{Target: "end"}}}},
			{Key: "end", Type: xs.Final},
		},
		Output: xs.NewExpr(func(a xs.ExprArgs[any]) any {
			spy.Call(a)
			return nil
		}),
	})

	service := xs.CreateActor(machine).Start()
	service.Send(xs.E{"type": "FINISH", "value": 1})
	assert.Equal(t, 1, spy.Count())
}

// JS: final states > output mapper should receive self
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/final.test.ts#L210
func TestFinal_OutputMapperShouldReceiveSelf(t *testing.T) {
	type output struct{ SelfRef xs.ActorRef }

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "done",
		States: xs.States{
			{Key: "done", Type: xs.Final},
		},
		Output: xs.NewExpr(func(a xs.ExprArgs[any]) any {
			return output{SelfRef: a.Self}
		}),
	})

	actor := xs.CreateActor(machine).Start()
	out, ok := actor.GetSnapshot().Output.(output)
	require.True(t, ok)
	// JS: expect(output.selfRef.send).toBeDefined()
	assert.NotNil(t, out.SelfRef)
}

// JS: final states > state output should be able to use context updated by the entry action of the reached final state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/final.test.ts#L230
func TestFinal_StateOutputShouldUseContextUpdatedByEntryActionOfReachedFinalState(t *testing.T) {
	spy := newSpy()
	type ctx struct{ Count int }

	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Context: ctx{Count: 0},
		Initial: "a",
		States: xs.States{
			{
				Key:     "a",
				Initial: "a1",
				States: xs.States{
					{Key: "a1", On: map[string]xs.Transitions{"NEXT": {{Target: "a2"}}}},
					{
						Key:  "a2",
						Type: xs.Final,
						Entry: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
							c := a.Context
							c.Count = 1
							return c
						})},
						Output: xs.NewExpr(func(a xs.ExprArgs[ctx]) any {
							return a.Context.Count
						}),
					},
				},
				OnDone: xs.Transitions{{Actions: xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[ctx]) {
					spy.Call(a.Event.(xs.DoneStateEvent).Output)
				})}}},
			},
		},
	})
	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.Ev("NEXT"))

	assert.Contains(t, spy.Calls(), []any{1})
}

// final1StartFinishRegion mirrors a JS region
// `{ initial: 'start', states: { start: { on: { [ev]: 'finish' } }, finish: { type: 'final' } } }`.
func final1StartFinishRegion(key, ev string) xs.StateConfig {
	return xs.StateConfig{
		Key:     key,
		Initial: "start",
		States: xs.States{
			{Key: "start", On: map[string]xs.Transitions{ev: {{Target: "finish"}}}},
			{Key: "finish", Type: xs.Final},
		},
	}
}

// final1AlphaParallel mirrors the JS `alpha` parallel state shared by three tests.
func final1AlphaParallel() xs.StateConfig {
	return xs.StateConfig{
		Key:  "alpha",
		Type: xs.Parallel,
		States: xs.States{
			final1StartFinishRegion("one", "finish_one_alpha"),
			final1StartFinishRegion("two", "finish_two_alpha"),
		},
	}
}

// final1CompoundBeta mirrors the JS compound `beta` state
// `{ initial: 'three', states: { three: { on: { finish_beta: 'finish' } }, finish: { type: 'final' } } }`.
func final1CompoundBeta() xs.StateConfig {
	return xs.StateConfig{
		Key:     "beta",
		Initial: "three",
		States: xs.States{
			{Key: "three", On: map[string]xs.Transitions{"finish_beta": {{Target: "finish"}}}},
			{Key: "finish", Type: xs.Final},
		},
	}
}

// JS: final states > should emit a done state event for a parallel state when its parallel children reach their final states
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/final.test.ts#L268
func TestFinal_ShouldEmitDoneStateEventForParallelWhenParallelChildrenReachFinal(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "first",
		States: xs.States{
			{
				Key:  "first",
				Type: xs.Parallel,
				States: xs.States{
					final1AlphaParallel(),
					{
						Key:  "beta",
						Type: xs.Parallel,
						States: xs.States{
							final1StartFinishRegion("third", "finish_three_beta"),
							final1StartFinishRegion("fourth", "finish_four_beta"),
						},
					},
				},
				OnDone: xs.Transitions{{Target: "done"}},
			},
			{Key: "done", Type: xs.Final},
		},
	})

	actorRef := xs.CreateActor(machine).Start()

	actorRef.Send(xs.Ev("finish_one_alpha"))
	actorRef.Send(xs.Ev("finish_two_alpha"))
	actorRef.Send(xs.Ev("finish_three_beta"))
	actorRef.Send(xs.Ev("finish_four_beta"))

	assert.Equal(t, xs.StatusDone, actorRef.GetSnapshot().Status)
}

// JS: final states > should emit a done state event for a parallel state when its compound child reaches its final state when the other parallel child region is already in its final state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/final.test.ts#L364
func TestFinal_DoneStateForParallelWhenCompoundChildFinalAfterParallelRegionFinal(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "first",
		States: xs.States{
			{
				Key:  "first",
				Type: xs.Parallel,
				States: xs.States{
					final1AlphaParallel(),
					final1CompoundBeta(),
				},
				OnDone: xs.Transitions{{Target: "done"}},
			},
			{Key: "done", Type: xs.Final},
		},
	})

	actorRef := xs.CreateActor(machine).Start()

	// reach final state of a parallel state
	actorRef.Send(xs.Ev("finish_one_alpha"))
	actorRef.Send(xs.Ev("finish_two_alpha"))

	// reach final state of a compound state
	actorRef.Send(xs.Ev("finish_beta"))

	assert.Equal(t, xs.StatusDone, actorRef.GetSnapshot().Status)
}

// JS: final states > should emit a done state event for a parallel state when its parallel child reaches its final state when the other compound child region is already in its final state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/final.test.ts#L442
func TestFinal_DoneStateForParallelWhenParallelChildFinalAfterCompoundRegionFinal(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "first",
		States: xs.States{
			{
				Key:  "first",
				Type: xs.Parallel,
				States: xs.States{
					final1AlphaParallel(),
					final1CompoundBeta(),
				},
				OnDone: xs.Transitions{{Target: "done"}},
			},
			{Key: "done", Type: xs.Final},
		},
	})

	actorRef := xs.CreateActor(machine).Start()

	// reach final state of a compound state
	actorRef.Send(xs.Ev("finish_beta"))

	// reach final state of a parallel state
	actorRef.Send(xs.Ev("finish_one_alpha"))
	actorRef.Send(xs.Ev("finish_two_alpha"))

	assert.Equal(t, xs.StatusDone, actorRef.GetSnapshot().Status)
}

// final1ParallelReachesFinalMachine mirrors the machine shared (verbatim
// duplicated in JS) by the two "should reach a final state when a parallel
// state ... reaches its final state" tests.
func final1ParallelReachesFinalMachine() *xs.StateMachine[any] {
	return xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{
				Key:    "a",
				Type:   xs.Parallel,
				OnDone: xs.Transitions{{Target: "b"}},
				States: xs.States{
					{
						Key:  "a1",
						Type: xs.Parallel,
						States: xs.States{
							{Key: "a1a", Type: xs.Final},
							{Key: "a1b", Type: xs.Final},
						},
					},
					{
						Key:     "a2",
						Initial: "a2a",
						States:  xs.States{{Key: "a2a", Type: xs.Final}},
					},
				},
			},
			{Key: "b", Type: xs.Final},
		},
	})
}

// JS: final states > should reach a final state when a parallel state reaches its final state and transitions to a top-level final state in response to that
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/final.test.ts#L520
func TestFinal_ShouldReachFinalWhenParallelReachesFinalAndTransitionsToTopLevelFinal(t *testing.T) {
	machine := final1ParallelReachesFinalMachine()

	actorRef := xs.CreateActor(machine).Start()

	assert.Equal(t, xs.StatusDone, actorRef.GetSnapshot().Status)
}

// JS: final states > should reach a final state when a parallel state nested in a parallel state reaches its final state and transitions to a top-level final state in response to that
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/final.test.ts#L552
func TestFinal_ShouldReachFinalWhenNestedParallelReachesFinalAndTransitionsToTopLevelFinal(t *testing.T) {
	machine := final1ParallelReachesFinalMachine()

	actorRef := xs.CreateActor(machine).Start()

	assert.Equal(t, xs.StatusDone, actorRef.GetSnapshot().Status)
}

// JS: final states > root output should be called with a "xstate.done.state.*" event of the parallel root when a direct final child of that parallel root is reached
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/final.test.ts#L583
func TestFinal_RootOutputCalledWithParallelRootDoneEventWhenDirectFinalChildReached(t *testing.T) {
	spy := newSpy()
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Type: xs.Parallel,
		States: xs.States{
			{Key: "a", Type: xs.Final},
		},
		Output: xs.NewExpr(func(a xs.ExprArgs[any]) any {
			spy.Call(a.Event)
			return nil
		}),
	})

	xs.CreateActor(machine).Start()

	assert.Equal(t, [][]any{
		{xs.DoneStateEvent{StateID: "(machine)", Output: nil}},
	}, spy.Calls())
}

// JS: final states > root output should be called with a "xstate.done.state.*" event of the parallel root when a final child of its compound child is reached
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/final.test.ts#L611
func TestFinal_RootOutputCalledWithParallelRootDoneEventWhenCompoundChildFinalReached(t *testing.T) {
	spy := newSpy()
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Type: xs.Parallel,
		States: xs.States{
			{
				Key:     "a",
				Initial: "b",
				States: xs.States{
					{Key: "b", Type: xs.Final},
				},
			},
		},
		Output: xs.NewExpr(func(a xs.ExprArgs[any]) any {
			spy.Call(a.Event)
			return nil
		}),
	})

	xs.CreateActor(machine).Start()

	assert.Equal(t, [][]any{
		{xs.DoneStateEvent{StateID: "(machine)", Output: nil}},
	}, spy.Calls())
}

// JS: final states > root output should be called with a "xstate.done.state.*" event of the parallel root when a final descendant is reached 2 parallel levels deep
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/final.test.ts#L644
func TestFinal_RootOutputCalledWithParallelRootDoneEventWhenFinalDescendant2LevelsDeep(t *testing.T) {
	spy := newSpy()
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Type: xs.Parallel,
		States: xs.States{
			{
				Key:  "a",
				Type: xs.Parallel,
				States: xs.States{
					{
						Key:     "b",
						Initial: "c",
						States: xs.States{
							{Key: "c", Type: xs.Final},
						},
					},
				},
			},
		},
		Output: xs.NewExpr(func(a xs.ExprArgs[any]) any {
			spy.Call(a.Event)
			return nil
		}),
	})

	xs.CreateActor(machine).Start()

	assert.Equal(t, [][]any{
		{xs.DoneStateEvent{StateID: "(machine)", Output: nil}},
	}, spy.Calls())
}

// JS: final states > onDone of an outer parallel state should be called with its own "xstate.done.state.*" event when its direct parallel child completes
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/final.test.ts#L682
func TestFinal_OnDoneOfOuterParallelCalledWithOwnDoneEventWhenDirectParallelChildCompletes(t *testing.T) {
	spy := newSpy()
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{
				Key:  "a",
				Type: xs.Parallel,
				States: xs.States{
					{
						Key:  "b",
						Type: xs.Parallel,
						States: xs.States{
							{
								Key:     "c",
								Initial: "d",
								States: xs.States{
									{Key: "d", Type: xs.Final},
								},
							},
						},
					},
				},
				OnDone: xs.Transitions{{Actions: xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[any]) {
					spy.Call(a.Event)
				})}}},
			},
		},
	})
	xs.CreateActor(machine).Start()

	assert.Equal(t, [][]any{
		{xs.DoneStateEvent{StateID: "(machine).a", Output: nil}},
	}, spy.Calls())
}

// JS: final states > onDone should not be called when the machine reaches its final state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/final.test.ts#L726
func TestFinal_OnDoneShouldNotBeCalledWhenMachineReachesItsFinalState(t *testing.T) {
	spy := newSpy()
	spyAction := xs.ActionFunc(func(a xs.ActionArgs[any]) { spy.Call(a) })

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Type: xs.Parallel,
		States: xs.States{
			{
				Key:  "a",
				Type: xs.Parallel,
				States: xs.States{
					{
						Key:     "b",
						Initial: "c",
						States: xs.States{
							{Key: "c", Type: xs.Final},
						},
						OnDone: xs.Transitions{{Actions: xs.Actions{spyAction}}},
					},
				},
				OnDone: xs.Transitions{{Actions: xs.Actions{spyAction}}},
			},
		},
		OnDone: xs.Transitions{{Actions: xs.Actions{spyAction}}},
	})
	xs.CreateActor(machine).Start()

	assert.Equal(t, 0, spy.Count())
}

// JS: final states > machine should not complete when a parallel child of a compound state completes
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/final.test.ts#L760
func TestFinal_MachineShouldNotCompleteWhenParallelChildOfCompoundStateCompletes(t *testing.T) {
	// JS declares an unused `const spy = vi.fn()` here; nothing asserts on it.
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{
				Key:  "a",
				Type: xs.Parallel,
				States: xs.States{
					{
						Key:     "b",
						Initial: "c",
						States: xs.States{
							{Key: "c", Type: xs.Final},
						},
					},
				},
			},
		},
	})

	actorRef := xs.CreateActor(machine).Start()

	assert.Equal(t, xs.StatusActive, actorRef.GetSnapshot().Status)
}

// JS: final states > root output should only be called once when multiple parallel regions complete at once
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/final.test.ts#L786
func TestFinal_RootOutputOnlyCalledOnceWhenMultipleParallelRegionsCompleteAtOnce(t *testing.T) {
	spy := newSpy()

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Type: xs.Parallel,
		States: xs.States{
			{Key: "a", Type: xs.Final},
			{Key: "b", Type: xs.Final},
		},
		Output: xs.NewExpr(func(a xs.ExprArgs[any]) any {
			spy.Call(a)
			return nil
		}),
	})

	xs.CreateActor(machine).Start()

	assert.Equal(t, 1, spy.Count())
}

// JS: final states > onDone of a parallel state should only be called once when multiple parallel regions complete at once
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/final.test.ts#L807
func TestFinal_OnDoneOfParallelOnlyCalledOnceWhenMultipleRegionsCompleteAtOnce(t *testing.T) {
	spy := newSpy()

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{
				Key:  "a",
				Type: xs.Parallel,
				States: xs.States{
					{Key: "b", Type: xs.Final},
					{Key: "c", Type: xs.Final},
				},
				OnDone: xs.Transitions{{Actions: xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[any]) {
					spy.Call(a)
				})}}},
			},
		},
	})

	xs.CreateActor(machine).Start()

	assert.Equal(t, 1, spy.Count())
}

// JS: final states > should call exit actions in reversed document order when the machines reaches its final state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/final.test.ts#L835
func TestFinal_ShouldCallExitActionsInReversedDocOrderWhenMachineReachesFinalState(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{"EV": {{Target: "b"}}}},
			{Key: "b", Type: xs.Final},
		},
	})

	flushTracked := final1TrackEntries(machine)

	actorRef := xs.CreateActor(machine).Start()
	flushTracked()

	// it's important to send an event here that results in a transition that computes new `state._nodes`
	// and that could impact the order in which exit actions are called
	actorRef.Send(xs.Ev("EV"))

	assert.Equal(t, []string{
		// result of the transition
		"exit: a",
		"enter: b",
		// result of reaching final states
		"exit: b",
		"exit: __root__",
	}, flushTracked())
}

// final1TwoRegionMachine mirrors the parallel root machine with regions a/b
// shared by the "exit actions of parallel states" tests; evA/evB are the
// events moving region a/b to its final child.
func final1TwoRegionMachine(evA, evB string) *xs.StateMachine[any] {
	return xs.CreateMachine(xs.MachineConfig[any]{
		Type: xs.Parallel,
		States: xs.States{
			{
				Key:     "a",
				Initial: "child_a1",
				States: xs.States{
					{Key: "child_a1", On: map[string]xs.Transitions{evA: {{Target: "child_a2"}}}},
					{Key: "child_a2", Type: xs.Final},
				},
			},
			{
				Key:     "b",
				Initial: "child_b1",
				States: xs.States{
					{Key: "child_b1", On: map[string]xs.Transitions{evB: {{Target: "child_b2"}}}},
					{Key: "child_b2", Type: xs.Final},
				},
			},
		},
	})
}

// JS: final states > should call exit actions of parallel states in reversed document order when the machines reaches its final state after earlier region transition
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/final.test.ts#L869
func TestFinal_ExitActionsOfParallelInReversedDocOrderAfterEarlierRegionTransition(t *testing.T) {
	machine := final1TwoRegionMachine("EV2", "EV1")

	flushTracked := final1TrackEntries(machine)

	actorRef := xs.CreateActor(machine).Start()

	// it's important to send an event here that results in a transition as that computes new `state._nodes`
	// and that could impact the order in which exit actions are called
	actorRef.Send(xs.Ev("EV1"))
	flushTracked()
	actorRef.Send(xs.Ev("EV2"))

	assert.Equal(t, []string{
		// result of the transition
		"exit: a.child_a1",
		"enter: a.child_a2",
		// result of reaching final states
		"exit: b.child_b2",
		"exit: b",
		"exit: a.child_a2",
		"exit: a",
		"exit: __root__",
	}, flushTracked())
}

// JS: final states > should call exit actions of parallel states in reversed document order when the machines reaches its final state after later region transition
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/final.test.ts#L925
func TestFinal_ExitActionsOfParallelInReversedDocOrderAfterLaterRegionTransition(t *testing.T) {
	machine := final1TwoRegionMachine("EV2", "EV1")

	flushTracked := final1TrackEntries(machine)

	actorRef := xs.CreateActor(machine).Start()
	// it's important to send an event here that results in a transition as that computes new `state._nodes`
	// and that could impact the order in which exit actions are called
	actorRef.Send(xs.Ev("EV1"))
	flushTracked()
	actorRef.Send(xs.Ev("EV2"))

	assert.Equal(t, []string{
		// result of the transition
		"exit: a.child_a1",
		"enter: a.child_a2",
		// result of reaching final states
		"exit: b.child_b2",
		"exit: b",
		"exit: a.child_a2",
		"exit: a",
		"exit: __root__",
	}, flushTracked())
}

// JS: final states > should call exit actions of parallel states in reversed document order when the machines reaches its final state after multiple regions transition
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/final.test.ts#L980
func TestFinal_ExitActionsOfParallelInReversedDocOrderAfterMultipleRegionsTransition(t *testing.T) {
	machine := final1TwoRegionMachine("EV", "EV")

	flushTracked := final1TrackEntries(machine)

	actorRef := xs.CreateActor(machine).Start()
	flushTracked()
	// it's important to send an event here that results in a transition as that computes new `state._nodes`
	// and that could impact the order in which exit actions are called
	actorRef.Send(xs.Ev("EV"))

	assert.Equal(t, []string{
		// result of the transition
		"exit: b.child_b1",
		"exit: a.child_a1",
		"enter: a.child_a2",
		"enter: b.child_b2",
		// result of reaching final states
		"exit: b.child_b2",
		"exit: b",
		"exit: a.child_a2",
		"exit: a",
		"exit: __root__",
	}, flushTracked())
}

// JS: final states > should not complete a parallel root immediately when only some of its regions are in their final states (final state reached in a compound region)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/final.test.ts#L1036
func TestFinal_ShouldNotCompleteParallelRootWhenOnlySomeRegionsFinalCompoundRegion(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Type: xs.Parallel,
		States: xs.States{
			{
				Key:     "A",
				Initial: "A1",
				States: xs.States{
					{Key: "A1", Type: xs.Final},
				},
			},
			{
				Key:     "B",
				Initial: "B1",
				States: xs.States{
					{Key: "B1"},
					{Key: "B2", Type: xs.Final},
				},
			},
		},
	})

	actorRef := xs.CreateActor(machine).Start()

	assert.Equal(t, xs.StatusActive, actorRef.GetSnapshot().Status)
}

// JS: final states > should not complete a parallel root immediately when only some of its regions are in their final states (a direct final child state reached)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/final.test.ts#L1065
func TestFinal_ShouldNotCompleteParallelRootWhenOnlySomeRegionsFinalDirectFinalChild(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Type: xs.Parallel,
		States: xs.States{
			{Key: "A", Type: xs.Final},
			{
				Key:     "B",
				Initial: "B1",
				States: xs.States{
					{Key: "B1"},
					{Key: "B2", Type: xs.Final},
				},
			},
		},
	})

	actorRef := xs.CreateActor(machine).Start()

	assert.Equal(t, xs.StatusActive, actorRef.GetSnapshot().Status)
}

// JS: final states > should not resolve output of a final state if its parent is a parallel state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/final.test.ts#L1089
func TestFinal_ShouldNotResolveOutputOfFinalStateIfParentIsParallel(t *testing.T) {
	spy := newSpy()

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "A",
		States: xs.States{
			{
				Key:  "A",
				Type: xs.Parallel,
				States: xs.States{
					{
						Key:  "B",
						Type: xs.Final,
						Output: xs.NewExpr(func(a xs.ExprArgs[any]) any {
							spy.Call(a)
							return nil
						}),
					},
					{
						Key:     "C",
						Initial: "C1",
						States: xs.States{
							{Key: "C1"},
						},
					},
				},
			},
		},
	})

	xs.CreateActor(machine).Start()

	assert.Equal(t, 0, spy.Count())
}

// JS: final states > should only call exit actions once when a child machine reaches its final state and sends an event to its parent that ends up stopping that child
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/final.test.ts#L1118
func TestFinal_OnlyCallExitActionsOnceWhenChildReachesFinalAndSendsEventStoppingIt(t *testing.T) {
	spy := newSpy()

	child := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "start",
		Exit:    xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[any]) { spy.Call(a) })},
		States: xs.States{
			{Key: "start", On: map[string]xs.Transitions{"CANCEL": {{Target: "canceled"}}}},
			{
				Key:   "canceled",
				Type:  xs.Final,
				Entry: xs.Actions{xs.SendParent(xs.Ev("CHILD_CANCELED"))},
			},
		},
	})
	parent := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "start",
		States: xs.States{
			{
				Key: "start",
				Invoke: []xs.InvokeConfig{{
					ID:     "child",
					Logic:  child,
					OnDone: xs.Transitions{{Target: "completed"}},
				}},
				On: map[string]xs.Transitions{"CHILD_CANCELED": {{Target: "canceled"}}},
			},
			{Key: "canceled"},
			{Key: "completed"},
		},
	})

	actorRef := xs.CreateActor(parent).Start()

	actorRef.GetSnapshot().Children["child"].Send(xs.Ev("CANCEL"))

	assert.Equal(t, 1, spy.Count())
}

// JS: final states > should deliver final outgoing events (from final entry action) to the parent before delivering the `xstate.done.actor.*` event
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/final.test.ts#L1163
func TestFinal_DeliverFinalOutgoingEventsFromFinalEntryBeforeDoneActorEvent(t *testing.T) {
	child := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "start",
		States: xs.States{
			{Key: "start", On: map[string]xs.Transitions{"CANCEL": {{Target: "canceled"}}}},
			{
				Key:   "canceled",
				Type:  xs.Final,
				Entry: xs.Actions{xs.SendParent(xs.Ev("CHILD_CANCELED"))},
			},
		},
	})
	parent := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "start",
		States: xs.States{
			{
				Key: "start",
				Invoke: []xs.InvokeConfig{{
					ID:     "child",
					Logic:  child,
					OnDone: xs.Transitions{{Target: "completed"}},
				}},
				On: map[string]xs.Transitions{"CHILD_CANCELED": {{Target: "canceled"}}},
			},
			{Key: "canceled"},
			{Key: "completed"},
		},
	})

	actorRef := xs.CreateActor(parent).Start()

	actorRef.GetSnapshot().Children["child"].Send(xs.Ev("CANCEL"))

	// if `xstate.done.actor.*` would be delivered first the value would be `completed`
	assert.Equal(t, "canceled", actorRef.GetSnapshot().Value)
}

// JS: final states > should deliver final outgoing events (from root exit action) to the parent before delivering the `xstate.done.actor.*` event
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/final.test.ts#L1206
func TestFinal_DeliverFinalOutgoingEventsFromRootExitBeforeDoneActorEvent(t *testing.T) {
	child := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "start",
		States: xs.States{
			{Key: "start", On: map[string]xs.Transitions{"CANCEL": {{Target: "canceled"}}}},
			{Key: "canceled", Type: xs.Final},
		},
		Exit: xs.Actions{xs.SendParent(xs.Ev("CHILD_CANCELED"))},
	})
	parent := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "start",
		States: xs.States{
			{
				Key: "start",
				Invoke: []xs.InvokeConfig{{
					ID:     "child",
					Logic:  child,
					OnDone: xs.Transitions{{Target: "completed"}},
				}},
				On: map[string]xs.Transitions{"CHILD_CANCELED": {{Target: "canceled"}}},
			},
			{Key: "canceled"},
			{Key: "completed"},
		},
	})

	actorRef := xs.CreateActor(parent).Start()

	actorRef.GetSnapshot().Children["child"].Send(xs.Ev("CANCEL"))

	// if `xstate.done.actor.*` would be delivered first the value would be `completed`
	assert.Equal(t, "canceled", actorRef.GetSnapshot().Value)
}

// JS: final states > should be possible to complete with a null output (directly on root)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/final.test.ts#L1249
func TestFinal_ShouldBePossibleToCompleteWithNullOutputDirectlyOnRoot(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "start",
		States: xs.States{
			{Key: "start", On: map[string]xs.Transitions{"NEXT": {{Target: "end"}}}},
			{Key: "end", Type: xs.Final},
		},
		Output: nil, // JS: output: null
	})

	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.Ev("NEXT"))

	snap := actorRef.GetSnapshot()
	// The JS test only reaches `toBe(null)` on a completed actor; Go nil also
	// covers "no output", so the completion is asserted explicitly.
	assert.Equal(t, xs.StatusDone, snap.Status)
	assert.Nil(t, snap.Output)
}

// JS: final states > should be possible to complete with a null output (resolving with final state's output)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/final.test.ts#L1271
func TestFinal_ShouldBePossibleToCompleteWithNullOutputResolvingFinalStateOutput(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "start",
		States: xs.States{
			{Key: "start", On: map[string]xs.Transitions{"NEXT": {{Target: "end"}}}},
			{Key: "end", Type: xs.Final, Output: nil}, // JS: output: null
		},
		Output: xs.NewExpr(func(a xs.ExprArgs[any]) any {
			return a.Event.(xs.DoneStateEvent).Output
		}),
	})

	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.Ev("NEXT"))

	snap := actorRef.GetSnapshot()
	// See the note on the "directly on root" variant.
	assert.Equal(t, xs.StatusDone, snap.Status)
	assert.Nil(t, snap.Output)
}
