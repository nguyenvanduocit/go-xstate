package xstate_test

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// actions1TrackEntries mirrors trackEntries from test/utils.ts: it prepends
// entry/exit tracking actions to every state node of the machine and returns a
// flush function that yields (and clears) the recorded log.
func actions1TrackEntries[C any](machine *xs.StateMachine[C]) func() []string {
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

// ---- entry/exit actions > State.actions ----

// JS: entry/exit actions > State.actions > should return the entry actions of an initial state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L36
func TestActions_StateActions_ShouldReturnEntryActionsOfInitialState(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "green",
		States: xs.States{
			{Key: "green"},
		},
	})
	flushTracked := actions1TrackEntries(machine)
	xs.CreateActor(machine).Start()

	assert.Equal(t, []string{"enter: __root__", "enter: green"}, flushTracked())
}

// JS: entry/exit actions > State.actions > should return the entry actions of an initial state (deep)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L49
func TestActions_StateActions_ShouldReturnEntryActionsOfInitialStateDeep(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{
				Key:     "a",
				Initial: "a1",
				States: xs.States{
					{Key: "a1", On: map[string]xs.Transitions{
						"NEXT": {{Target: "a2"}},
					}},
					{Key: "a2"},
				},
				On: map[string]xs.Transitions{"CHANGE": {{Target: "b"}}},
			},
			{Key: "b"},
		},
	})

	flushTracked := actions1TrackEntries(machine)
	xs.CreateActor(machine).Start()

	assert.Equal(t, []string{
		"enter: __root__",
		"enter: a",
		"enter: a.a1",
	}, flushTracked())
}

// JS: entry/exit actions > State.actions > should return the entry actions of an initial state (parallel)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L79
func TestActions_StateActions_ShouldReturnEntryActionsOfInitialStateParallel(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Type: xs.Parallel,
		States: xs.States{
			{
				Key:     "a",
				Initial: "a1",
				States:  xs.States{{Key: "a1"}},
			},
			{
				Key:     "b",
				Initial: "b1",
				States:  xs.States{{Key: "b1"}},
			},
		},
	})

	flushTracked := actions1TrackEntries(machine)
	xs.CreateActor(machine).Start()

	assert.Equal(t, []string{
		"enter: __root__",
		"enter: a",
		"enter: a.a1",
		"enter: b",
		"enter: b.b1",
	}, flushTracked())
}

// JS: entry/exit actions > State.actions > should return the entry and exit actions of a transition
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L110
func TestActions_StateActions_ShouldReturnEntryAndExitActionsOfTransition(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "green",
		States: xs.States{
			{Key: "green", On: map[string]xs.Transitions{
				"TIMER": {{Target: "yellow"}},
			}},
			{Key: "yellow"},
		},
	})

	flushTracked := actions1TrackEntries(machine)

	actor := xs.CreateActor(machine).Start()
	flushTracked()

	actor.Send(xs.Ev("TIMER"))

	assert.Equal(t, []string{"exit: green", "enter: yellow"}, flushTracked())
}

// JS: entry/exit actions > State.actions > should return the entry and exit actions of a deep transition
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L133
func TestActions_StateActions_ShouldReturnEntryAndExitActionsOfDeepTransition(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "green",
		States: xs.States{
			{Key: "green", On: map[string]xs.Transitions{
				"TIMER": {{Target: "yellow"}},
			}},
			{
				Key:     "yellow",
				Initial: "speed_up",
				States:  xs.States{{Key: "speed_up"}},
			},
		},
	})

	flushTracked := actions1TrackEntries(machine)

	actor := xs.CreateActor(machine).Start()
	flushTracked()

	actor.Send(xs.Ev("TIMER"))

	assert.Equal(t, []string{
		"exit: green",
		"enter: yellow",
		"enter: yellow.speed_up",
	}, flushTracked())
}

// JS: entry/exit actions > State.actions > should return the entry and exit actions of a nested transition
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L165
func TestActions_StateActions_ShouldReturnEntryAndExitActionsOfNestedTransition(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "green",
		States: xs.States{
			{
				Key:     "green",
				Initial: "walk",
				States: xs.States{
					{Key: "walk", On: map[string]xs.Transitions{
						"PED_COUNTDOWN": {{Target: "wait"}},
					}},
					{Key: "wait"},
				},
			},
		},
	})

	flushTracked := actions1TrackEntries(machine)

	actor := xs.CreateActor(machine).Start()
	flushTracked()

	actor.Send(xs.Ev("PED_COUNTDOWN"))

	assert.Equal(t, []string{"exit: green.walk", "enter: green.wait"}, flushTracked())
}

// JS: entry/exit actions > State.actions > should not have actions for unhandled events (shallow)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L193
func TestActions_StateActions_ShouldNotHaveActionsForUnhandledEventsShallow(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "green",
		States: xs.States{
			{Key: "green"},
		},
	})
	flushTracked := actions1TrackEntries(machine)

	actor := xs.CreateActor(machine).Start()
	flushTracked()

	actor.Send(xs.Ev("FAKE"))

	assert.Equal(t, []string{}, flushTracked())
}

// JS: entry/exit actions > State.actions > should not have actions for unhandled events (deep)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L210
func TestActions_StateActions_ShouldNotHaveActionsForUnhandledEventsDeep(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "green",
		States: xs.States{
			{
				Key:     "green",
				Initial: "walk",
				States: xs.States{
					{Key: "walk"},
					{Key: "wait"},
					{Key: "stop"},
				},
			},
		},
	})

	flushTracked := actions1TrackEntries(machine)

	actor := xs.CreateActor(machine).Start()
	flushTracked()

	actor.Send(xs.Ev("FAKE"))

	assert.Equal(t, []string{}, flushTracked())
}

// JS: entry/exit actions > State.actions > should exit and enter the state for reentering self-transitions (shallow)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L235
func TestActions_StateActions_ShouldExitAndEnterStateForReenteringSelfTransitionsShallow(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "green",
		States: xs.States{
			{Key: "green", On: map[string]xs.Transitions{
				"RESTART": {{Target: "green", Reenter: true}},
			}},
		},
	})

	flushTracked := actions1TrackEntries(machine)

	actor := xs.CreateActor(machine).Start()
	flushTracked()

	actor.Send(xs.Ev("RESTART"))

	assert.Equal(t, []string{"exit: green", "enter: green"}, flushTracked())
}

// JS: entry/exit actions > State.actions > should exit and enter the state for reentering self-transitions (deep)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L260
func TestActions_StateActions_ShouldExitAndEnterStateForReenteringSelfTransitionsDeep(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "green",
		States: xs.States{
			{
				Key: "green",
				On: map[string]xs.Transitions{
					"RESTART": {{Target: "green", Reenter: true}},
				},
				Initial: "walk",
				States: xs.States{
					{Key: "walk"},
					{Key: "wait"},
					{Key: "stop"},
				},
			},
		},
	})
	flushTracked := actions1TrackEntries(machine)

	actor := xs.CreateActor(machine).Start()

	flushTracked()
	actor.Send(xs.Ev("RESTART"))

	assert.Equal(t, []string{
		"exit: green.walk",
		"exit: green",
		"enter: green",
		"enter: green.walk",
	}, flushTracked())
}

// JS: entry/exit actions > State.actions > should return actions for parallel machines
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L295
func TestActions_StateActions_ShouldReturnActionsForParallelMachines(t *testing.T) {
	actual := []string{}
	push := func(s string) xs.Action {
		return xs.ActionFunc(func(xs.ActionArgs[any]) { actual = append(actual, s) })
	}
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Type: xs.Parallel,
		States: xs.States{
			{
				Key:     "a",
				Initial: "a1",
				States: xs.States{
					{
						Key: "a1",
						On: map[string]xs.Transitions{
							"CHANGE": {{
								Target:  "a2",
								Actions: xs.Actions{push("do_a2"), push("another_do_a2")},
							}},
						},
						Entry: xs.Actions{push("enter_a1")},
						Exit:  xs.Actions{push("exit_a1")},
					},
					{
						Key:   "a2",
						Entry: xs.Actions{push("enter_a2")},
						Exit:  xs.Actions{push("exit_a2")},
					},
				},
				Entry: xs.Actions{push("enter_a")},
				Exit:  xs.Actions{push("exit_a")},
			},
			{
				Key:     "b",
				Initial: "b1",
				States: xs.States{
					{
						Key: "b1",
						On: map[string]xs.Transitions{
							"CHANGE": {{Target: "b2", Actions: xs.Actions{push("do_b2")}}},
						},
						Entry: xs.Actions{push("enter_b1")},
						Exit:  xs.Actions{push("exit_b1")},
					},
					{
						Key:   "b2",
						Entry: xs.Actions{push("enter_b2")},
						Exit:  xs.Actions{push("exit_b2")},
					},
				},
				Entry: xs.Actions{push("enter_b")},
				Exit:  xs.Actions{push("exit_b")},
			},
		},
	})

	actor := xs.CreateActor(machine).Start()
	actual = actual[:0]

	actor.Send(xs.Ev("CHANGE"))

	assert.Equal(t, []string{
		"exit_b1", // reverse document order
		"exit_a1",
		"do_a2",
		"another_do_a2",
		"do_b2",
		"enter_a2",
		"enter_b2",
	}, actual)
}

// JS: entry/exit actions > State.actions > should return nested actions in the correct (child to parent) order
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L361
func TestActions_StateActions_ShouldReturnNestedActionsInChildToParentOrder(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{
				Key:     "a",
				Initial: "a1",
				States:  xs.States{{Key: "a1"}},
				On:      map[string]xs.Transitions{"CHANGE": {{Target: "b"}}},
			},
			{
				Key:     "b",
				Initial: "b1",
				States:  xs.States{{Key: "b1"}},
			},
		},
	})

	flushTracked := actions1TrackEntries(machine)

	actor := xs.CreateActor(machine).Start()

	flushTracked()
	actor.Send(xs.Ev("CHANGE"))

	assert.Equal(t, []string{
		"exit: a.a1",
		"exit: a",
		"enter: b",
		"enter: b.b1",
	}, flushTracked())
}

// JS: entry/exit actions > State.actions > should ignore parent state actions for same-parent substates
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L396
func TestActions_StateActions_ShouldIgnoreParentStateActionsForSameParentSubstates(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{
				Key:     "a",
				Initial: "a1",
				States: xs.States{
					{Key: "a1", On: map[string]xs.Transitions{
						"NEXT": {{Target: "a2"}},
					}},
					{Key: "a2"},
				},
			},
		},
	})

	flushTracked := actions1TrackEntries(machine)

	actor := xs.CreateActor(machine).Start()

	flushTracked()
	actor.Send(xs.Ev("NEXT"))

	assert.Equal(t, []string{"exit: a.a1", "enter: a.a2"}, flushTracked())
}

// JS: entry/exit actions > State.actions > should work with function actions
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L424
func TestActions_StateActions_ShouldWorkWithFunctionActions(t *testing.T) {
	entrySpy := newSpy()
	exitSpy := newSpy()
	transitionSpy := newSpy()
	spyAction := func(s *spy) xs.Action {
		return xs.ActionFunc(func(a xs.ActionArgs[any]) { s.Call(a) })
	}

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{
				Key:     "a",
				Initial: "a1",
				States: xs.States{
					{Key: "a1", On: map[string]xs.Transitions{
						"NEXT_FN": {{Target: "a3"}},
					}},
					{Key: "a2"},
					{
						Key: "a3",
						On: map[string]xs.Transitions{
							"NEXT": {{
								Target:  "a2",
								Actions: xs.Actions{spyAction(transitionSpy)},
							}},
						},
						Entry: xs.Actions{spyAction(entrySpy)},
						Exit:  xs.Actions{spyAction(exitSpy)},
					},
				},
			},
		},
	})

	flushTracked := actions1TrackEntries(machine)

	actor := xs.CreateActor(machine).Start()
	flushTracked()

	actor.Send(xs.Ev("NEXT_FN"))

	assert.Equal(t, []string{"exit: a.a1", "enter: a.a3"}, flushTracked())
	assert.Positive(t, entrySpy.Count())

	actor.Send(xs.Ev("NEXT"))

	assert.Equal(t, []string{"exit: a.a3", "enter: a.a2"}, flushTracked())
	assert.Positive(t, exitSpy.Count())
	assert.Positive(t, transitionSpy.Count())
}

// JS: entry/exit actions > State.actions > should exit children of parallel state nodes
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L473
func TestActions_StateActions_ShouldExitChildrenOfParallelStateNodes(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "B",
		States: xs.States{
			{Key: "A", On: map[string]xs.Transitions{
				"to-B": {{Target: "B"}},
			}},
			{
				Key:  "B",
				Type: xs.Parallel,
				On: map[string]xs.Transitions{
					"to-A": {{Target: "A"}},
				},
				States: xs.States{
					{Key: "C", Initial: "C1", States: xs.States{{Key: "C1"}}},
					{Key: "D", Initial: "D1", States: xs.States{{Key: "D1"}}},
				},
			},
		},
	})

	flushTracked := actions1TrackEntries(machine)

	actor := xs.CreateActor(machine).Start()

	flushTracked()
	actor.Send(xs.Ev("to-A"))

	assert.Equal(t, []string{
		"exit: B.D.D1",
		"exit: B.D",
		"exit: B.C.C1",
		"exit: B.C",
		"exit: B",
		"enter: A",
	}, flushTracked())
}

// JS: entry/exit actions > State.actions > should reenter targeted ancestor (as it's a descendant of the transition domain)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L522
func TestActions_StateActions_ShouldReenterTargetedAncestor(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "loaded",
		States: xs.States{
			{
				Key:     "loaded",
				ID:      "loaded",
				Initial: "idle",
				States: xs.States{
					{Key: "idle", On: map[string]xs.Transitions{
						"UPDATE": {{Target: "#loaded"}},
					}},
				},
			},
		},
	})

	flushTracked := actions1TrackEntries(machine)

	actor := xs.CreateActor(machine).Start()

	flushTracked()
	actor.Send(xs.Ev("UPDATE"))

	assert.Equal(t, []string{
		"exit: loaded.idle",
		"exit: loaded",
		"enter: loaded",
		"enter: loaded.idle",
	}, flushTracked())
}

// JS: entry/exit actions > State.actions > shouldn't use a referenced custom action over a builtin one when there is a naming conflict
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L555
func TestActions_StateActions_ShouldNotUseReferencedCustomActionOverBuiltinOne(t *testing.T) {
	type ctx struct{ Assigned bool }
	s := newSpy()
	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Context: ctx{Assigned: false},
		On: map[string]xs.Transitions{
			"EV": {{Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
				c := a.Context
				c.Assigned = true
				return c
			})}}},
		},
	}, xs.Implementations{
		Actions: map[string]xs.Action{
			"xstate.assign": xs.ActionFunc(func(a xs.ActionArgs[ctx]) { s.Call(a) }),
		},
	})

	actor := xs.CreateActor(machine).Start()
	actor.Send(xs.Ev("EV"))

	assert.Equal(t, 0, s.Count())
	assert.Equal(t, true, actor.GetSnapshot().Context.Assigned)
}

// JS: entry/exit actions > State.actions > shouldn't use a referenced custom action over an inline one when there is a naming conflict
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L582
func TestActions_StateActions_ShouldNotUseReferencedCustomActionOverInlineOne(t *testing.T) {
	s := newSpy()
	called := false

	// JS uses a named function `myFn`; Go funcs carry no name, so the inline
	// action is bound to a variable of the same name.
	myFn := xs.ActionFunc(func(xs.ActionArgs[any]) {
		called = true
	})

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		On: map[string]xs.Transitions{
			"EV": {{Actions: xs.Actions{myFn}}},
		},
	}, xs.Implementations{
		Actions: map[string]xs.Action{
			"myFn": xs.ActionFunc(func(a xs.ActionArgs[any]) { s.Call(a) }),
		},
	})

	actor := xs.CreateActor(machine).Start()
	actor.Send(xs.Ev("EV"))

	assert.Equal(t, 0, s.Count())
	assert.Equal(t, true, called)
}

// JS: entry/exit actions > State.actions > root entry/exit actions should be called on root reentering transitions
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L611
func TestActions_StateActions_RootEntryExitActionsCalledOnRootReenteringTransitions(t *testing.T) {
	entrySpy := newSpy()
	exitSpy := newSpy()

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		ID:    "root",
		Entry: xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[any]) { entrySpy.Call(a) })},
		Exit:  xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[any]) { exitSpy.Call(a) })},
		On: map[string]xs.Transitions{
			"EVENT": {{Target: "#two", Reenter: true}},
		},
		Initial: "one",
		States: xs.States{
			{Key: "one"},
			{Key: "two", ID: "two"},
		},
	})

	service := xs.CreateActor(machine).Start()

	// mockClear(): compare against the call counts at this point.
	entryBefore := entrySpy.Count()
	exitBefore := exitSpy.Count()

	service.Send(xs.Ev("EVENT"))

	assert.Greater(t, entrySpy.Count(), entryBefore)
	assert.Greater(t, exitSpy.Count(), exitBefore)
}

// JS: entry/exit actions > State.actions > should ignore same-parent state actions (sparse) > with a relative transition
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L646
func TestActions_StateActions_IgnoreSameParentSparse_WithRelativeTransition(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "ping",
		States: xs.States{
			{
				Key:     "ping",
				Initial: "foo",
				States: xs.States{
					{Key: "foo", On: map[string]xs.Transitions{
						"TACK": {{Target: "bar"}},
					}},
					{Key: "bar"},
				},
			},
		},
	})

	flushTracked := actions1TrackEntries(machine)

	actor := xs.CreateActor(machine).Start()
	flushTracked()

	actor.Send(xs.Ev("TACK"))

	assert.Equal(t, []string{"exit: ping.foo", "enter: ping.bar"}, flushTracked())
}

// JS: entry/exit actions > State.actions > should ignore same-parent state actions (sparse) > with an absolute transition
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L674
func TestActions_StateActions_IgnoreSameParentSparse_WithAbsoluteTransition(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "root",
		Initial: "ping",
		States: xs.States{
			{
				Key:     "ping",
				Initial: "foo",
				States: xs.States{
					{Key: "foo", On: map[string]xs.Transitions{
						"ABSOLUTE_TACK": {{Target: "#root.ping.bar"}},
					}},
					{Key: "bar"},
				},
			},
			{Key: "pong"},
		},
	})

	flushTracked := actions1TrackEntries(machine)

	actor := xs.CreateActor(machine).Start()
	flushTracked()

	actor.Send(xs.Ev("ABSOLUTE_TACK"))

	assert.Equal(t, []string{"exit: ping.foo", "enter: ping.bar"}, flushTracked())
}

// ---- entry/exit actions > entry/exit actions ----

// JS: entry/exit actions > entry/exit actions > should return the entry actions of an initial state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L707
func TestActions_EntryExitActions_ShouldReturnEntryActionsOfInitialState(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "green",
		States: xs.States{
			{Key: "green"},
		},
	})
	flushTracked := actions1TrackEntries(machine)
	xs.CreateActor(machine).Start()

	assert.Equal(t, []string{"enter: __root__", "enter: green"}, flushTracked())
}

// JS: entry/exit actions > entry/exit actions > should return the entry and exit actions of a transition
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L720
func TestActions_EntryExitActions_ShouldReturnEntryAndExitActionsOfTransition(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "green",
		States: xs.States{
			{Key: "green", On: map[string]xs.Transitions{
				"TIMER": {{Target: "yellow"}},
			}},
			{Key: "yellow"},
		},
	})

	flushTracked := actions1TrackEntries(machine)

	actor := xs.CreateActor(machine).Start()
	flushTracked()

	actor.Send(xs.Ev("TIMER"))

	assert.Equal(t, []string{"exit: green", "enter: yellow"}, flushTracked())
}

// JS: entry/exit actions > entry/exit actions > should return the entry and exit actions of a deep transition
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L743
func TestActions_EntryExitActions_ShouldReturnEntryAndExitActionsOfDeepTransition(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "green",
		States: xs.States{
			{Key: "green", On: map[string]xs.Transitions{
				"TIMER": {{Target: "yellow"}},
			}},
			{
				Key:     "yellow",
				Initial: "speed_up",
				States:  xs.States{{Key: "speed_up"}},
			},
		},
	})
	flushTracked := actions1TrackEntries(machine)

	actor := xs.CreateActor(machine).Start()
	flushTracked()

	actor.Send(xs.Ev("TIMER"))

	assert.Equal(t, []string{
		"exit: green",
		"enter: yellow",
		"enter: yellow.speed_up",
	}, flushTracked())
}

// JS: entry/exit actions > entry/exit actions > should return the entry and exit actions of a nested transition
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L774
func TestActions_EntryExitActions_ShouldReturnEntryAndExitActionsOfNestedTransition(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "green",
		States: xs.States{
			{
				Key:     "green",
				Initial: "walk",
				States: xs.States{
					{Key: "walk", On: map[string]xs.Transitions{
						"PED_COUNTDOWN": {{Target: "wait"}},
					}},
					{Key: "wait"},
				},
			},
		},
	})
	flushTracked := actions1TrackEntries(machine)

	actor := xs.CreateActor(machine).Start()
	flushTracked()

	actor.Send(xs.Ev("PED_COUNTDOWN"))

	assert.Equal(t, []string{"exit: green.walk", "enter: green.wait"}, flushTracked())
}

// JS: entry/exit actions > entry/exit actions > should keep the same state for unhandled events (shallow)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L801
func TestActions_EntryExitActions_ShouldKeepSameStateForUnhandledEventsShallow(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "green",
		States: xs.States{
			{Key: "green"},
		},
	})
	flushTracked := actions1TrackEntries(machine)

	actor := xs.CreateActor(machine).Start()
	flushTracked()

	actor.Send(xs.Ev("FAKE"))

	assert.Equal(t, []string{}, flushTracked())
}

// JS: entry/exit actions > entry/exit actions > should keep the same state for unhandled events (deep)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L818
func TestActions_EntryExitActions_ShouldKeepSameStateForUnhandledEventsDeep(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "green",
		States: xs.States{
			{
				Key:     "green",
				Initial: "walk",
				States:  xs.States{{Key: "walk"}},
			},
		},
	})
	flushTracked := actions1TrackEntries(machine)

	actor := xs.CreateActor(machine).Start()
	flushTracked()

	actor.Send(xs.Ev("FAKE"))

	assert.Equal(t, []string{}, flushTracked())
}

// JS: entry/exit actions > entry/exit actions > should exit and enter the state for reentering self-transitions (shallow)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L840
func TestActions_EntryExitActions_ShouldExitAndEnterStateForReenteringSelfTransitionsShallow(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "green",
		States: xs.States{
			{Key: "green", On: map[string]xs.Transitions{
				"RESTART": {{Target: "green", Reenter: true}},
			}},
		},
	})

	flushTracked := actions1TrackEntries(machine)

	actor := xs.CreateActor(machine).Start()
	flushTracked()

	actor.Send(xs.Ev("RESTART"))

	assert.Equal(t, []string{"exit: green", "enter: green"}, flushTracked())
}

// JS: entry/exit actions > entry/exit actions > should exit and enter the state for reentering self-transitions (deep)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L865
func TestActions_EntryExitActions_ShouldExitAndEnterStateForReenteringSelfTransitionsDeep(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "green",
		States: xs.States{
			{
				Key: "green",
				On: map[string]xs.Transitions{
					"RESTART": {{Target: "green", Reenter: true}},
				},
				Initial: "walk",
				States:  xs.States{{Key: "walk"}},
			},
		},
	})

	flushTracked := actions1TrackEntries(machine)

	actor := xs.CreateActor(machine).Start()
	flushTracked()

	actor.Send(xs.Ev("RESTART"))
	assert.Equal(t, []string{
		"exit: green.walk",
		"exit: green",
		"enter: green",
		"enter: green.walk",
	}, flushTracked())
}

// JS: entry/exit actions > entry/exit actions > should exit current node and enter target node when target is not a descendent or ancestor of current
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L898
func TestActions_EntryExitActions_ExitCurrentEnterTargetWhenTargetNotDescendantOrAncestor(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "A",
		States: xs.States{
			{
				Key:     "A",
				Initial: "A1",
				States: xs.States{
					{Key: "A1", On: map[string]xs.Transitions{
						"NEXT": {{Target: "#sibling_descendant"}},
					}},
					{
						Key:     "A2",
						Initial: "A2_child",
						States: xs.States{
							{Key: "A2_child", ID: "sibling_descendant"},
						},
					},
				},
			},
		},
	})

	flushTracked := actions1TrackEntries(machine)

	service := xs.CreateActor(machine).Start()
	flushTracked()
	service.Send(xs.Ev("NEXT"))

	assert.Equal(t, []string{
		"exit: A.A1",
		"enter: A.A2",
		"enter: A.A2.A2_child",
	}, flushTracked())
}

// JS: entry/exit actions > entry/exit actions > should exit current node and reenter target node when target is ancestor of current
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L936
func TestActions_EntryExitActions_ExitCurrentReenterTargetWhenTargetIsAncestor(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "A",
		States: xs.States{
			{
				Key:     "A",
				ID:      "ancestor",
				Initial: "A1",
				States: xs.States{
					{Key: "A1", On: map[string]xs.Transitions{
						"NEXT": {{Target: "A2"}},
					}},
					{
						Key:     "A2",
						Initial: "A2_child",
						States: xs.States{
							{Key: "A2_child", On: map[string]xs.Transitions{
								"NEXT": {{Target: "#ancestor"}},
							}},
						},
					},
				},
			},
		},
	})

	flushTracked := actions1TrackEntries(machine)

	service := xs.CreateActor(machine).Start()
	service.Send(xs.Ev("NEXT"))

	flushTracked()
	service.Send(xs.Ev("NEXT"))

	assert.Equal(t, []string{
		"exit: A.A2.A2_child",
		"exit: A.A2",
		"exit: A",
		"enter: A",
		"enter: A.A1",
	}, flushTracked())
}

// JS: entry/exit actions > entry/exit actions > should enter all descendents when target is a descendent of the source when using an reentering transition
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L981
func TestActions_EntryExitActions_EnterAllDescendantsWhenTargetIsDescendantWithReenter(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "A",
		States: xs.States{
			{
				Key:     "A",
				Initial: "A1",
				On: map[string]xs.Transitions{
					"NEXT": {{Reenter: true, Target: ".A2"}},
				},
				States: xs.States{
					{Key: "A1"},
					{
						Key:     "A2",
						Initial: "A2a",
						States:  xs.States{{Key: "A2a"}},
					},
				},
			},
		},
	})

	flushTracked := actions1TrackEntries(machine)

	service := xs.CreateActor(machine).Start()
	flushTracked()
	service.Send(xs.Ev("NEXT"))

	assert.Equal(t, []string{
		"exit: A.A1",
		"exit: A",
		"enter: A",
		"enter: A.A2",
		"enter: A.A2.A2a",
	}, flushTracked())
}

// JS: entry/exit actions > entry/exit actions > should exit deep descendant during a default self-transition
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L1021
func TestActions_EntryExitActions_ShouldExitDeepDescendantDuringDefaultSelfTransition(t *testing.T) {
	m := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{
				Key: "a",
				On: map[string]xs.Transitions{
					"EV": {{Target: "a"}},
				},
				Initial: "a1",
				States: xs.States{
					{
						Key:     "a1",
						Initial: "a11",
						States:  xs.States{{Key: "a11"}},
					},
				},
			},
		},
	})

	flushTracked := actions1TrackEntries(m)

	service := xs.CreateActor(m).Start()

	flushTracked()
	service.Send(xs.Ev("EV"))

	assert.Equal(t, []string{
		"exit: a.a1.a11",
		"exit: a.a1",
		"enter: a.a1",
		"enter: a.a1.a11",
	}, flushTracked())
}

// JS: entry/exit actions > entry/exit actions > should exit deep descendant during a reentering self-transition
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L1057
func TestActions_EntryExitActions_ShouldExitDeepDescendantDuringReenteringSelfTransition(t *testing.T) {
	m := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{
				Key: "a",
				On: map[string]xs.Transitions{
					"EV": {{Target: "a", Reenter: true}},
				},
				Initial: "a1",
				States: xs.States{
					{
						Key:     "a1",
						Initial: "a11",
						States:  xs.States{{Key: "a11"}},
					},
				},
			},
		},
	})

	flushTracked := actions1TrackEntries(m)

	service := xs.CreateActor(m).Start()

	flushTracked()
	service.Send(xs.Ev("EV"))

	assert.Equal(t, []string{
		"exit: a.a1.a11",
		"exit: a.a1",
		"exit: a",
		"enter: a",
		"enter: a.a1",
		"enter: a.a1.a11",
	}, flushTracked())
}

// JS: entry/exit actions > entry/exit actions > should not reenter leaf state during its default self-transition
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L1098
func TestActions_EntryExitActions_ShouldNotReenterLeafStateDuringDefaultSelfTransition(t *testing.T) {
	m := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{
				Key:     "a",
				Initial: "a1",
				States: xs.States{
					{Key: "a1", On: map[string]xs.Transitions{
						"EV": {{Target: "a1"}},
					}},
				},
			},
		},
	})

	flushTracked := actions1TrackEntries(m)

	service := xs.CreateActor(m).Start()

	flushTracked()
	service.Send(xs.Ev("EV"))

	assert.Equal(t, []string{}, flushTracked())
}

// JS: entry/exit actions > entry/exit actions > should reenter leaf state during its reentering self-transition
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L1125
func TestActions_EntryExitActions_ShouldReenterLeafStateDuringReenteringSelfTransition(t *testing.T) {
	m := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{
				Key:     "a",
				Initial: "a1",
				States: xs.States{
					{Key: "a1", On: map[string]xs.Transitions{
						"EV": {{Target: "a1", Reenter: true}},
					}},
				},
			},
		},
	})

	flushTracked := actions1TrackEntries(m)

	service := xs.CreateActor(m).Start()

	flushTracked()
	service.Send(xs.Ev("EV"))

	assert.Equal(t, []string{"exit: a.a1", "enter: a.a1"}, flushTracked())
}

// JS: entry/exit actions > entry/exit actions > should not enter exited state when targeting its ancestor and when its former descendant gets selected through initial state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L1155
func TestActions_EntryExitActions_NotEnterExitedStateTargetingAncestorFormerDescendant(t *testing.T) {
	m := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{
				Key:     "a",
				ID:      "parent",
				Initial: "a1",
				States: xs.States{
					{Key: "a1", On: map[string]xs.Transitions{
						"EV": {{Target: "a2"}},
					}},
					{Key: "a2", On: map[string]xs.Transitions{
						"EV": {{Target: "#parent"}},
					}},
				},
			},
		},
	})

	flushTracked := actions1TrackEntries(m)

	service := xs.CreateActor(m).Start()
	service.Send(xs.Ev("EV"))

	flushTracked()
	service.Send(xs.Ev("EV"))

	assert.Equal(t, []string{
		"exit: a.a2",
		"exit: a",
		"enter: a",
		"enter: a.a1",
	}, flushTracked())
}

// JS: entry/exit actions > entry/exit actions > should not enter exited state when targeting its ancestor and when its latter descendant gets selected through initial state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L1194
func TestActions_EntryExitActions_NotEnterExitedStateTargetingAncestorLatterDescendant(t *testing.T) {
	m := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{
				Key:     "a",
				ID:      "parent",
				Initial: "a2",
				States: xs.States{
					{Key: "a1", On: map[string]xs.Transitions{
						"EV": {{Target: "#parent"}},
					}},
					{Key: "a2", On: map[string]xs.Transitions{
						"EV": {{Target: "a1"}},
					}},
				},
			},
		},
	})

	flushTracked := actions1TrackEntries(m)

	service := xs.CreateActor(m).Start()
	service.Send(xs.Ev("EV"))

	flushTracked()
	service.Send(xs.Ev("EV"))

	assert.Equal(t, []string{
		"exit: a.a1",
		"exit: a",
		"enter: a",
		"enter: a.a2",
	}, flushTracked())
}

// actions2TrackEntries mirrors trackEntries() from core/test/utils.ts: it
// prepends entry/exit tracker actions to every state node of the machine and
// returns a flush function that returns (and resets) the collected logs.
func actions2TrackEntries[C any](machine *xs.StateMachine[C]) func() []string {
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
		for _, child := range state.States {
			addTrackingActions(child, strings.Join(child.Path, "."))
			addTrackingActionsRecursively(child)
		}
	}

	addTrackingActions(machine.Root, "__root__")
	addTrackingActionsRecursively(machine.Root)

	return func() []string {
		mu.Lock()
		defer mu.Unlock()
		flushed := logs
		logs = []string{}
		return flushed
	}
}

// ---- entry/exit actions > parallel states ----

// JS: entry/exit actions > parallel states > should return entry action defined on parallel state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L1235
func TestActions_ParallelStates_ShouldReturnEntryActionDefinedOnParallelState(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "start",
		States: xs.States{
			{Key: "start", On: map[string]xs.Transitions{
				"ENTER_PARALLEL": {{Target: "p1"}},
			}},
			{Key: "p1", Type: xs.Parallel, States: xs.States{
				{Key: "nested", Initial: "inner", States: xs.States{
					{Key: "inner"},
				}},
			}},
		},
	})

	flushTracked := actions2TrackEntries(machine)

	actor := xs.CreateActor(machine).Start()

	flushTracked()
	actor.Send(xs.Ev("ENTER_PARALLEL"))

	assert.Equal(t, []string{
		"exit: start",
		"enter: p1",
		"enter: p1.nested",
		"enter: p1.nested.inner",
	}, flushTracked())
}

func actions2CameraMachine() *xs.StateMachine[any] {
	return xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "ready",
		States: xs.States{
			{
				Key:  "ready",
				Type: xs.Parallel,
				On: map[string]xs.Transitions{
					"FOO": {{Target: "#cameraOff", Reenter: true}},
				},
				States: xs.States{
					{Key: "devicesInfo"},
					{Key: "camera", Initial: "on", States: xs.States{
						{Key: "on"},
						{Key: "off", ID: "cameraOff"},
					}},
				},
			},
		},
	})
}

// JS: entry/exit actions > parallel states > should reenter parallel region when a parallel state gets reentered while targeting another region
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L1271
func TestActions_ParallelStates_ShouldReenterRegionWhenParallelStateGetsReenteredTargetingAnotherRegion(t *testing.T) {
	machine := actions2CameraMachine()

	flushTracked := actions2TrackEntries(machine)

	service := xs.CreateActor(machine).Start()

	flushTracked()
	service.Send(xs.Ev("FOO"))

	assert.Equal(t, []string{
		"exit: ready.camera.on",
		"exit: ready.camera",
		"exit: ready.devicesInfo",
		"exit: ready",
		"enter: ready",
		"enter: ready.devicesInfo",
		"enter: ready.camera",
		"enter: ready.camera.off",
	}, flushTracked())
}

// JS: entry/exit actions > parallel states > should reenter parallel region when a parallel state is reentered while targeting another region
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L1318
func TestActions_ParallelStates_ShouldReenterRegionWhenParallelStateIsReenteredTargetingAnotherRegion(t *testing.T) {
	machine := actions2CameraMachine()

	flushTracked := actions2TrackEntries(machine)

	service := xs.CreateActor(machine).Start()

	flushTracked()
	service.Send(xs.Ev("FOO"))

	assert.Equal(t, []string{
		"exit: ready.camera.on",
		"exit: ready.camera",
		"exit: ready.devicesInfo",
		"exit: ready",
		"enter: ready",
		"enter: ready.devicesInfo",
		"enter: ready.camera",
		"enter: ready.camera.off",
	}, flushTracked())
}

// ---- entry/exit actions > targetless transitions ----

// JS: entry/exit actions > targetless transitions > shouldn't exit a state on a parent's targetless transition
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L1367
func TestActions_TargetlessTransitions_ShouldNotExitStateOnParentsTargetlessTransition(t *testing.T) {
	parent := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "one",
		On: map[string]xs.Transitions{
			"WHATEVER": {{Actions: xs.Actions{xs.ActionFunc(func(xs.ActionArgs[any]) {})}}},
		},
		States: xs.States{
			{Key: "one"},
		},
	})

	flushTracked := actions2TrackEntries(parent)

	service := xs.CreateActor(parent).Start()

	flushTracked()
	service.Send(xs.Ev("WHATEVER"))

	assert.Equal(t, []string{}, flushTracked())
}

// JS: entry/exit actions > targetless transitions > shouldn't exit (and reenter) state on targetless delayed transition
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L1390
func TestActions_TargetlessTransitions_ShouldNotExitAndReenterStateOnTargetlessDelayedTransition(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "one",
		States: xs.States{
			{Key: "one", After: map[string]xs.Transitions{
				"10": {{Actions: xs.Actions{xs.ActionFunc(func(xs.ActionArgs[any]) {
					// do smth
				})}}},
			}},
		},
	})

	flushTracked := actions2TrackEntries(machine)

	xs.CreateActor(machine).Start()
	flushTracked()

	sleep(50)

	assert.Equal(t, []string{}, flushTracked())
}

// ---- entry/exit actions > when reaching a final state ----

// JS: entry/exit actions > when reaching a final state > exit actions should be called when invoked machine reaches its final state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L1419
func TestActions_WhenReachingFinalState_ExitActionsShouldBeCalledWhenInvokedMachineReachesFinalState(t *testing.T) {
	sig := newSignal()
	var exitCalled atomic.Bool
	var childExitCalled atomic.Bool
	childMachine := xs.CreateMachine(xs.MachineConfig[any]{
		Exit: xs.Actions{xs.ActionFunc(func(xs.ActionArgs[any]) {
			exitCalled.Store(true)
		})},
		Initial: "a",
		States: xs.States{
			{Key: "a", Type: xs.Final, Exit: xs.Actions{xs.ActionFunc(func(xs.ActionArgs[any]) {
				childExitCalled.Store(true)
			})}},
		},
	})

	parentMachine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "active",
		States: xs.States{
			{Key: "active", Invoke: []xs.InvokeConfig{{
				Logic:  childMachine,
				OnDone: xs.Transitions{{Target: "finished"}},
			}}},
			{Key: "finished", Type: xs.Final},
		},
	})

	actor := xs.CreateActor(parentMachine)
	actor.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{
		Complete: func() {
			assert.True(t, exitCalled.Load())
			assert.True(t, childExitCalled.Load())
			sig.Resolve()
		},
	})
	actor.Start()
	sig.Wait(t)
}

// ---- entry/exit actions > when stopped ----

// JS: entry/exit actions > when stopped > exit actions should not be called when stopping a machine
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L1467
func TestActions_WhenStopped_ExitActionsShouldNotBeCalledWhenStoppingMachine(t *testing.T) {
	rootSpy := newSpy()
	childSpy := newSpy()

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Exit:    xs.Actions{xs.ActionFunc(func(xs.ActionArgs[any]) { rootSpy.Call() })},
		Initial: "a",
		States: xs.States{
			{Key: "a", Exit: xs.Actions{xs.ActionFunc(func(xs.ActionArgs[any]) { childSpy.Call() })}},
		},
	})

	service := xs.CreateActor(machine).Start()
	service.Stop()

	assert.Equal(t, 0, rootSpy.Count())
	assert.Equal(t, 0, childSpy.Count())
}

// JS: entry/exit actions > when stopped > an exit action executed when an interpreter reaches its final state should be called with the last received event
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L1488
func TestActions_WhenStopped_ExitActionOnFinalStateShouldBeCalledWithLastReceivedEvent(t *testing.T) {
	var mu sync.Mutex
	var receivedEvent xs.Event
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{"NEXT": {{Target: "b"}}}},
			{Key: "b", Type: xs.Final},
		},
		Exit: xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[any]) {
			mu.Lock()
			defer mu.Unlock()
			receivedEvent = a.Event
		})},
	})

	service := xs.CreateActor(machine).Start()
	service.Send(xs.Ev("NEXT"))

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, xs.Ev("NEXT"), receivedEvent)
}

// https://github.com/statelyai/xstate/issues/2880
// JS: entry/exit actions > when stopped > stopping an interpreter that receives events from its children exit handlers should not throw
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L1514
func TestActions_WhenStopped_StoppingInterpreterReceivingEventsFromChildExitHandlersShouldNotThrow(t *testing.T) {
	child := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "child",
		Initial: "idle",
		States: xs.States{
			{Key: "idle", Exit: xs.Actions{xs.SendParent(xs.Ev("EXIT"))}},
		},
	})

	parent := xs.CreateMachine(xs.MachineConfig[any]{
		ID:     "parent",
		Invoke: []xs.InvokeConfig{{Logic: child}},
	})

	interpreter := xs.CreateActor(parent)
	interpreter.Start()

	assert.NotPanics(t, func() { interpreter.Stop() })
}

// JS: entry/exit actions > when stopped > sent events from exit handlers of a stopped child should not be received by the parent
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L1541
func TestActions_WhenStopped_SentEventsFromExitHandlersOfStoppedChildShouldNotBeReceivedByParent(t *testing.T) {
	t.Skip("skipped in JS")

	type ctx struct{ Child xs.ActorRef }

	child := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "child",
		Initial: "idle",
		States: xs.States{
			{Key: "idle", Exit: xs.Actions{xs.SendParent(xs.Ev("EXIT"))}},
		},
	})

	parent := xs.CreateMachine(xs.MachineConfig[ctx]{
		ID: "parent",
		ContextFn: func(a xs.ContextArgs) ctx {
			return ctx{Child: a.Spawn(child)}
		},
		On: map[string]xs.Transitions{
			"STOP_CHILD": {{Actions: xs.Actions{xs.StopChild(xs.NewExpr(func(a xs.ExprArgs[ctx]) any {
				return a.Context.Child
			}))}}},
			"EXIT": {{Actions: xs.Actions{xs.ActionFunc(func(xs.ActionArgs[ctx]) {
				panic(errors.New("This should not be called."))
			})}}},
		},
	})

	interpreter := xs.CreateActor(parent).Start()
	interpreter.Send(xs.Ev("STOP_CHILD"))
}

// JS: entry/exit actions > when stopped > sent events from exit handlers of a done child should be received by the parent
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L1578
func TestActions_WhenStopped_SentEventsFromExitHandlersOfDoneChildShouldBeReceivedByParent(t *testing.T) {
	type ctx struct{ Child xs.ActorRef }
	var eventReceived atomic.Bool

	child := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "child",
		Initial: "active",
		States: xs.States{
			{Key: "active", On: map[string]xs.Transitions{"FINISH": {{Target: "done"}}}},
			{Key: "done", Type: xs.Final},
		},
		Exit: xs.Actions{xs.SendParent(xs.Ev("CHILD_DONE"))},
	})

	parent := xs.CreateMachine(xs.MachineConfig[ctx]{
		ID: "parent",
		ContextFn: func(a xs.ContextArgs) ctx {
			return ctx{Child: a.Spawn(child)}
		},
		On: map[string]xs.Transitions{
			"FINISH_CHILD": {{Actions: xs.Actions{xs.SendTo(xs.NewExpr(func(a xs.ExprArgs[ctx]) any {
				return a.Context.Child
			}), xs.Ev("FINISH"))}}},
			"CHILD_DONE": {{Actions: xs.Actions{xs.ActionFunc(func(xs.ActionArgs[ctx]) {
				eventReceived.Store(true)
			})}}},
		},
	})

	interpreter := xs.CreateActor(parent).Start()
	interpreter.Send(xs.Ev("FINISH_CHILD"))

	assert.Equal(t, true, eventReceived.Load())
}

// JS: entry/exit actions > when stopped > sent events from exit handlers of a stopped child should not be received by its children
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L1625
func TestActions_WhenStopped_SentEventsFromExitHandlersOfStoppedChildShouldNotBeReceivedByItsChildren(t *testing.T) {
	s := newSpy()

	grandchild := xs.CreateMachine(xs.MachineConfig[any]{
		ID: "grandchild",
		On: map[string]xs.Transitions{
			"STOPPED": {{Actions: xs.Actions{xs.ActionFunc(func(xs.ActionArgs[any]) { s.Call() })}}},
		},
	})

	child := xs.CreateMachine(xs.MachineConfig[any]{
		ID:     "child",
		Invoke: []xs.InvokeConfig{{ID: "myChild", Logic: grandchild}},
		Exit:   xs.Actions{xs.SendTo("myChild", xs.Ev("STOPPED"))},
	})

	parent := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "parent",
		Initial: "a",
		States: xs.States{
			{
				Key:    "a",
				Invoke: []xs.InvokeConfig{{Logic: child}},
				On:     map[string]xs.Transitions{"NEXT": {{Target: "b"}}},
			},
			{Key: "b"},
		},
	})

	interpreter := xs.CreateActor(parent).Start()
	interpreter.Send(xs.Ev("NEXT"))

	assert.Equal(t, 0, s.Count())
}

// JS: entry/exit actions > when stopped > sent events from exit handlers of a done child should be received by its children
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L1668
func TestActions_WhenStopped_SentEventsFromExitHandlersOfDoneChildShouldBeReceivedByItsChildren(t *testing.T) {
	s := newSpy()

	grandchild := xs.CreateMachine(xs.MachineConfig[any]{
		ID: "grandchild",
		On: map[string]xs.Transitions{
			"STOPPED": {{Actions: xs.Actions{xs.ActionFunc(func(xs.ActionArgs[any]) { s.Call() })}}},
		},
	})

	child := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "child",
		Initial: "a",
		Invoke:  []xs.InvokeConfig{{ID: "myChild", Logic: grandchild}},
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{"FINISH": {{Target: "b"}}}},
			{Key: "b", Type: xs.Final},
		},
		Exit: xs.Actions{xs.SendTo("myChild", xs.Ev("STOPPED"))},
	})

	parent := xs.CreateMachine(xs.MachineConfig[any]{
		ID:     "parent",
		Invoke: []xs.InvokeConfig{{ID: "myChild", Logic: child}},
		On: map[string]xs.Transitions{
			"NEXT": {{Actions: xs.Actions{xs.SendTo("myChild", xs.Ev("FINISH"))}}},
		},
	})

	interpreter := xs.CreateActor(parent).Start()
	interpreter.Send(xs.Ev("NEXT"))

	assert.Equal(t, 1, s.Count())
}

// JS: entry/exit actions > when stopped > actors spawned in exit handlers of a stopped child should not be started
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L1719
func TestActions_WhenStopped_ActorsSpawnedInExitHandlersOfStoppedChildShouldNotBeStarted(t *testing.T) {
	type ctx struct{ ActorRef xs.ActorRef }

	grandchild := xs.CreateMachine(xs.MachineConfig[any]{
		ID: "grandchild",
		Entry: xs.Actions{xs.ActionFunc(func(xs.ActionArgs[any]) {
			panic(errors.New("This should not be called."))
		})},
	})

	parent := xs.CreateMachine(xs.MachineConfig[ctx]{
		ID:      "parent",
		Context: ctx{},
		Exit: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
			c := a.Context
			c.ActorRef = a.Spawn(grandchild)
			return c
		})},
	})

	// JS has no explicit expectation: the test fails if the grandchild's entry
	// throws (surfaced as an unhandled error).
	interpreter := xs.CreateActor(parent, xs.WithUnhandledErrorHandler(func(err any) {
		t.Errorf("unexpected unhandled error: %v", err)
	})).Start()
	interpreter.Stop()
}

// JS: entry/exit actions > when stopped > should note execute referenced custom actions correctly when stopping an interpreter
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L1739
func TestActions_WhenStopped_ShouldNoteExecuteReferencedCustomActionsWhenStoppingInterpreter(t *testing.T) {
	type ctx struct{}
	s := newSpy()
	parent := xs.CreateMachine(
		xs.MachineConfig[ctx]{
			ID:      "parent",
			Context: ctx{},
			Exit:    xs.Actions{xs.ActionRef{Type: "referencedAction"}},
		},
		xs.Implementations{
			Actions: map[string]xs.Action{
				"referencedAction": xs.ActionFunc(func(xs.ActionArgs[ctx]) { s.Call() }),
			},
		},
	)

	interpreter := xs.CreateActor(parent).Start()
	interpreter.Stop()

	assert.Equal(t, 0, s.Count())
}

// JS: entry/exit actions > when stopped > should not execute builtin actions when stopping an interpreter
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L1760
func TestActions_WhenStopped_ShouldNotExecuteBuiltinActionsWhenStoppingInterpreter(t *testing.T) {
	type ctx struct{ ExecutedAssigns []string }
	machine := xs.CreateMachine(
		xs.MachineConfig[ctx]{
			Context: ctx{ExecutedAssigns: []string{}},
			Exit: xs.Actions{
				xs.ActionRef{Type: "referencedAction"},
				xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
					next := append(append([]string{}, a.Context.ExecutedAssigns...), "inline")
					return ctx{ExecutedAssigns: next}
				}),
			},
		},
		xs.Implementations{
			Actions: map[string]xs.Action{
				"referencedAction": xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
					next := append(append([]string{}, a.Context.ExecutedAssigns...), "referenced")
					return ctx{ExecutedAssigns: next}
				}),
			},
		},
	)

	interpreter := xs.CreateActor(machine).Start()
	interpreter.Stop()

	assert.Equal(t, []string{}, interpreter.GetSnapshot().Context.ExecutedAssigns)
}

// JS: entry/exit actions > when stopped > should clear all scheduled events when the interpreter gets stopped
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L1794
func TestActions_WhenStopped_ShouldClearAllScheduledEventsWhenInterpreterGetsStopped(t *testing.T) {
	var service *xs.Actor[*xs.MachineSnapshot[any]]
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		On: map[string]xs.Transitions{
			"INITIALIZE_SYNC_SEQUENCE": {{Actions: xs.Actions{xs.ActionFunc(func(xs.ActionArgs[any]) {
				// schedule those 2 events
				service.Send(xs.Ev("SOME_EVENT"))
				service.Send(xs.Ev("SOME_EVENT"))
				// but also immediately stop *while* the `INITIALIZE_SYNC_SEQUENCE` is still being processed
				service.Stop()
			})}}},
			"SOME_EVENT": {{Actions: xs.Actions{xs.ActionFunc(func(xs.ActionArgs[any]) {
				panic(errors.New("This should not be called."))
			})}}},
		},
	})

	// JS has no explicit expectation: the test fails if the SOME_EVENT action
	// throws (surfaced as an unhandled error).
	service = xs.CreateActor(machine, xs.WithUnhandledErrorHandler(func(err any) {
		t.Errorf("unexpected unhandled error: %v", err)
	})).Start()

	service.Send(xs.Ev("INITIALIZE_SYNC_SEQUENCE"))
}

// JS: entry/exit actions > when stopped > should execute exit actions of the settled state of the last initiated microstep
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L1819
func TestActions_WhenStopped_ShouldExecuteExitActionsOfSettledStateOfLastInitiatedMicrostep(t *testing.T) {
	var mu sync.Mutex
	exitActions := []string{}
	var service *xs.Actor[*xs.MachineSnapshot[any]]
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "foo",
		States: xs.States{
			{
				Key: "foo",
				Exit: xs.Actions{xs.ActionFunc(func(xs.ActionArgs[any]) {
					mu.Lock()
					defer mu.Unlock()
					exitActions = append(exitActions, "foo action")
				})},
				On: map[string]xs.Transitions{
					"INITIALIZE_SYNC_SEQUENCE": {{
						Target: "bar",
						Actions: xs.Actions{
							xs.ActionFunc(func(xs.ActionArgs[any]) {
								// immediately stop *while* the `INITIALIZE_SYNC_SEQUENCE` is still being processed
								service.Stop()
							}),
							xs.ActionFunc(func(xs.ActionArgs[any]) {}),
						},
					}},
				},
			},
			{
				Key: "bar",
				Exit: xs.Actions{xs.ActionFunc(func(xs.ActionArgs[any]) {
					mu.Lock()
					defer mu.Unlock()
					exitActions = append(exitActions, "bar action")
				})},
			},
		},
	})

	service = xs.CreateActor(machine).Start()

	service.Send(xs.Ev("INITIALIZE_SYNC_SEQUENCE"))

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, []string{"foo action"}, exitActions)
}

// JS: entry/exit actions > when stopped > should not execute exit actions of the settled state of the last initiated microstep after executing all actions from that microstep
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L1856
func TestActions_WhenStopped_ShouldNotExecuteExitActionsOfSettledStateAfterAllMicrostepActions(t *testing.T) {
	var mu sync.Mutex
	executedActions := []string{}
	push := func(s string) {
		mu.Lock()
		defer mu.Unlock()
		executedActions = append(executedActions, s)
	}
	var service *xs.Actor[*xs.MachineSnapshot[any]]
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "foo",
		States: xs.States{
			{
				Key: "foo",
				Exit: xs.Actions{xs.ActionFunc(func(xs.ActionArgs[any]) {
					push("foo exit action")
				})},
				On: map[string]xs.Transitions{
					"INITIALIZE_SYNC_SEQUENCE": {{
						Target: "bar",
						Actions: xs.Actions{
							xs.ActionFunc(func(xs.ActionArgs[any]) {
								// immediately stop *while* the `INITIALIZE_SYNC_SEQUENCE` is still being processed
								service.Stop()
							}),
							xs.ActionFunc(func(xs.ActionArgs[any]) {
								push("foo transition action")
							}),
						},
					}},
				},
			},
			{
				Key: "bar",
				Exit: xs.Actions{xs.ActionFunc(func(xs.ActionArgs[any]) {
					push("bar exit action")
				})},
			},
		},
	})

	service = xs.CreateActor(machine).Start()

	service.Send(xs.Ev("INITIALIZE_SYNC_SEQUENCE"))

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, []string{
		"foo exit action",
		"foo transition action",
	}, executedActions)
}

// ---- initial actions ----

// JS: initial actions > should support initial actions
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L1901
func TestActions_InitialActions_ShouldSupportInitialActions(t *testing.T) {
	var mu sync.Mutex
	actual := []string{}
	push := func(s string) {
		mu.Lock()
		defer mu.Unlock()
		actual = append(actual, s)
	}
	machine := xs.CreateMachine(xs.WithMachineInitialActions(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", Entry: xs.Actions{xs.ActionFunc(func(xs.ActionArgs[any]) { push("entryA") })}},
		},
	}, xs.ActionFunc(func(xs.ActionArgs[any]) { push("initialA") })))
	xs.CreateActor(machine).Start()

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, []string{"initialA", "entryA"}, actual)
}

// JS: initial actions > should support initial actions from transition
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L1918
func TestActions_InitialActions_ShouldSupportInitialActionsFromTransition(t *testing.T) {
	var mu sync.Mutex
	actual := []string{}
	push := func(s string) {
		mu.Lock()
		defer mu.Unlock()
		actual = append(actual, s)
	}
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{"NEXT": {{Target: "b"}}}},
			xs.WithStateInitialActions(xs.StateConfig{
				Key:     "b",
				Entry:   xs.Actions{xs.ActionFunc(func(xs.ActionArgs[any]) { push("entryB") })},
				Initial: "foo",
				States: xs.States{
					{Key: "foo", Entry: xs.Actions{xs.ActionFunc(func(xs.ActionArgs[any]) { push("entryFoo") })}},
				},
			}, xs.ActionFunc(func(xs.ActionArgs[any]) { push("initialFoo") })),
		},
	})

	actor := xs.CreateActor(machine).Start()

	actor.Send(xs.Ev("NEXT"))

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, []string{"entryB", "initialFoo", "entryFoo"}, actual)
}

// JS: initial actions > should execute actions of initial transitions only once when taking an explicit transition
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L1950
func TestActions_InitialActions_ShouldExecuteInitialTransitionActionsOnlyOnceOnExplicitTransition(t *testing.T) {
	s := newSpy()
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{"NEXT": {{Target: "b"}}}},
			xs.WithStateInitialActions(xs.StateConfig{
				Key:     "b",
				Initial: "b_child",
				States: xs.States{
					xs.WithStateInitialActions(xs.StateConfig{
						Key:     "b_child",
						Initial: "b_granchild",
						States: xs.States{
							{Key: "b_granchild"},
						},
					}, xs.ActionFunc(func(xs.ActionArgs[any]) { s.Call("initial in b_child") })),
				},
			}, xs.ActionFunc(func(xs.ActionArgs[any]) { s.Call("initial in b") })),
		},
	})

	actorRef := xs.CreateActor(machine).Start()

	actorRef.Send(xs.Ev("NEXT"))

	assert.Equal(t, [][]any{
		{"initial in b"},
		{"initial in b_child"},
	}, s.Calls())
}

// JS: initial actions > should execute actions of all initial transitions resolving to the initial state value
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L1998
func TestActions_InitialActions_ShouldExecuteActionsOfAllInitialTransitionsResolvingToInitialValue(t *testing.T) {
	s := newSpy()
	machine := xs.CreateMachine(xs.WithMachineInitialActions(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			xs.WithStateInitialActions(xs.StateConfig{
				Key:     "a",
				Initial: "a1",
				States: xs.States{
					{Key: "a1"},
				},
			}, xs.ActionFunc(func(xs.ActionArgs[any]) { s.Call("inner") })),
		},
	}, xs.ActionFunc(func(xs.ActionArgs[any]) { s.Call("root") })))

	xs.CreateActor(machine).Start()

	assert.Equal(t, [][]any{
		{"root"},
		{"inner"},
	}, s.Calls())
}

// JS: initial actions > should execute actions of the initial transition when taking a root reentering self-transition
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L2032
func TestActions_InitialActions_ShouldExecuteInitialTransitionActionsOnRootReenteringSelfTransition(t *testing.T) {
	var mu sync.Mutex
	s := newSpy()
	machine := xs.CreateMachine(xs.WithMachineInitialActions(xs.MachineConfig[any]{
		ID:      "root",
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{"NEXT": {{Target: "b"}}}},
			{Key: "b"},
		},
		On: map[string]xs.Transitions{
			"REENTER": {{Target: "#root", Reenter: true}},
		},
	}, xs.ActionFunc(func(xs.ActionArgs[any]) {
		mu.Lock()
		cur := s
		mu.Unlock()
		cur.Call()
	})))

	actorRef := xs.CreateActor(machine).Start()

	actorRef.Send(xs.Ev("NEXT"))
	// spy.mockClear()
	mu.Lock()
	s = newSpy()
	mu.Unlock()

	actorRef.Send(xs.Ev("REENTER"))

	mu.Lock()
	cur := s
	mu.Unlock()
	assert.Equal(t, 1, cur.Count())
	assert.Equal(t, "a", actorRef.GetSnapshot().Value)
}

// ---- actions on invalid transition ----

// JS: actions on invalid transition > should not recall previous actions
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L2069
func TestActions_ActionsOnInvalidTransition_ShouldNotRecallPreviousActions(t *testing.T) {
	s := newSpy()
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "idle",
		States: xs.States{
			{Key: "idle", On: map[string]xs.Transitions{
				"STOP": {{
					Target:  "stop",
					Actions: xs.Actions{xs.ActionFunc(func(xs.ActionArgs[any]) { s.Call() })},
				}},
			}},
			{Key: "stop"},
		},
	})
	actor := xs.CreateActor(machine).Start()

	actor.Send(xs.Ev("STOP"))
	assert.Equal(t, 1, s.Count())

	actor.Send(xs.Ev("INVALID"))
	assert.Equal(t, 1, s.Count())
}

// ---- actions config ----

// JS: actions config > should reference actions defined in actions parameter of machine options (entry actions)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L2107
func TestActions_ActionsConfig_ShouldReferenceActionsFromMachineOptionsEntryActions(t *testing.T) {
	s := newSpy()
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{"EVENT": {{Target: "b"}}}},
			{Key: "b", Entry: xs.Actions{
				xs.ActionRef{Type: "definedAction"},
				xs.ActionRef{Type: "definedAction"},
				xs.ActionRef{Type: "undefinedAction"},
			}},
		},
		On: map[string]xs.Transitions{
			"E": {{Target: ".a"}},
		},
	}).Provide(xs.Implementations{
		Actions: map[string]xs.Action{
			"definedAction": xs.ActionFunc(func(xs.ActionArgs[any]) { s.Call() }),
		},
	})

	actor := xs.CreateActor(machine).Start()
	actor.Send(xs.Ev("EVENT"))

	assert.Equal(t, 2, s.Count())
}

// JS: actions config > should reference actions defined in actions parameter of machine options (initial state)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L2136
func TestActions_ActionsConfig_ShouldReferenceActionsFromMachineOptionsInitialState(t *testing.T) {
	s := newSpy()
	machine := xs.CreateMachine(
		xs.MachineConfig[any]{
			Entry: xs.Actions{
				xs.ActionRef{Type: "definedAction"},
				xs.ActionRef{Type: "definedAction"},
				xs.ActionRef{Type: "undefinedAction"},
			},
		},
		xs.Implementations{
			Actions: map[string]xs.Action{
				"definedAction": xs.ActionFunc(func(xs.ActionArgs[any]) { s.Call() }),
			},
		},
	)

	xs.CreateActor(machine).Start()

	assert.Equal(t, 2, s.Count())
}

// JS: actions config > should be able to reference action implementations from action objects
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L2154
func TestActions_ActionsConfig_ShouldReferenceActionImplementationsFromActionObjects(t *testing.T) {
	type ctx struct{ Count int }
	definedAction := xs.ActionFunc(func(xs.ActionArgs[ctx]) {})

	machine := xs.CreateMachine(
		xs.MachineConfig[ctx]{
			Initial: "a",
			Context: ctx{Count: 0},
			States: xs.States{
				{
					Key: "a",
					Entry: xs.Actions{
						xs.ActionRef{Type: "definedAction"},
						xs.ActionRef{Type: "definedAction"},
						xs.ActionRef{Type: "undefinedAction"},
					},
					On: map[string]xs.Transitions{
						"EVENT": {{
							Target: "b",
							Actions: xs.Actions{
								xs.ActionRef{Type: "definedAction"},
								xs.ActionRef{Type: "updateContext"},
							},
						}},
					},
				},
				{Key: "b"},
			},
		},
		xs.Implementations{
			Actions: map[string]xs.Action{
				"definedAction": definedAction,
				"updateContext": xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
					return ctx{Count: 10}
				}),
			},
		},
	)
	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.Ev("EVENT"))
	snapshot := actorRef.GetSnapshot()

	assert.Equal(t, ctx{Count: 10}, snapshot.Context)
}

// JS: actions config > should work with anonymous functions (with warning)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L2203
func TestActions_ActionsConfig_ShouldWorkWithAnonymousFunctions(t *testing.T) {
	var entryCalled, actionCalled, exitCalled atomic.Bool

	anonMachine := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "anon",
		Initial: "active",
		States: xs.States{
			{
				Key:   "active",
				Entry: xs.Actions{xs.ActionFunc(func(xs.ActionArgs[any]) { entryCalled.Store(true) })},
				Exit:  xs.Actions{xs.ActionFunc(func(xs.ActionArgs[any]) { exitCalled.Store(true) })},
				On: map[string]xs.Transitions{
					"EVENT": {{
						Target:  "inactive",
						Actions: xs.Actions{xs.ActionFunc(func(xs.ActionArgs[any]) { actionCalled.Store(true) })},
					}},
				},
			},
			{Key: "inactive"},
		},
	})

	actor := xs.CreateActor(anonMachine).Start()

	assert.Equal(t, true, entryCalled.Load())

	actor.Send(xs.Ev("EVENT"))

	assert.Equal(t, true, exitCalled.Load())
	assert.Equal(t, true, actionCalled.Load())
}

// ---- action meta ----

// JS: action meta > should provide the original params
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L2238
func TestActions_ActionMeta_ShouldProvideOriginalParams(t *testing.T) {
	s := newSpy()

	testMachine := xs.CreateMachine(
		xs.MachineConfig[any]{
			ID:      "test",
			Initial: "foo",
			States: xs.States{
				{Key: "foo", Entry: xs.Actions{xs.ActionRef{
					Type:   "entryAction",
					Params: map[string]any{"value": "something"},
				}}},
			},
		},
		xs.Implementations{
			Actions: map[string]xs.Action{
				"entryAction": xs.ActionFunc(func(a xs.ActionArgs[any]) { s.Call(a.Params) }),
			},
		},
	)

	xs.CreateActor(testMachine).Start()

	assert.Contains(t, s.Calls(), []any{map[string]any{"value": "something"}})
}

// JS: action meta > should provide undefined params when it was configured as string
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L2272
func TestActions_ActionMeta_ShouldProvideUndefinedParamsWhenConfiguredAsString(t *testing.T) {
	s := newSpy()

	testMachine := xs.CreateMachine(
		xs.MachineConfig[any]{
			ID:      "test",
			Initial: "foo",
			States: xs.States{
				{Key: "foo", Entry: xs.Actions{xs.ActionRef{Type: "entryAction"}}},
			},
		},
		xs.Implementations{
			Actions: map[string]xs.Action{
				"entryAction": xs.ActionFunc(func(a xs.ActionArgs[any]) { s.Call(a.Params) }),
			},
		},
	)

	xs.CreateActor(testMachine).Start()

	assert.Contains(t, s.Calls(), []any{nil})
}

// JS: action meta > should provide the action with resolved params when they are dynamic
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L2299
func TestActions_ActionMeta_ShouldProvideResolvedParamsWhenDynamic(t *testing.T) {
	s := newSpy()

	machine := xs.CreateMachine(
		xs.MachineConfig[any]{
			Entry: xs.Actions{xs.ActionRef{
				Type: "entryAction",
				Params: xs.NewExpr(func(xs.ExprArgs[any]) any {
					return map[string]any{"stuff": 100}
				}),
			}},
		},
		xs.Implementations{
			Actions: map[string]xs.Action{
				"entryAction": xs.ActionFunc(func(a xs.ActionArgs[any]) { s.Call(a.Params) }),
			},
		},
	)

	xs.CreateActor(machine).Start()

	assert.Contains(t, s.Calls(), []any{map[string]any{"stuff": 100}})
}

// JS: action meta > should resolve dynamic params using context value
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L2325
func TestActions_ActionMeta_ShouldResolveDynamicParamsUsingContextValue(t *testing.T) {
	type ctx struct{ Secret int }
	s := newSpy()

	machine := xs.CreateMachine(
		xs.MachineConfig[ctx]{
			Context: ctx{Secret: 42},
			Entry: xs.Actions{xs.ActionRef{
				Type: "entryAction",
				Params: xs.NewExpr(func(a xs.ExprArgs[ctx]) any {
					return map[string]any{"secret": a.Context.Secret}
				}),
			}},
		},
		xs.Implementations{
			Actions: map[string]xs.Action{
				"entryAction": xs.ActionFunc(func(a xs.ActionArgs[ctx]) { s.Call(a.Params) }),
			},
		},
	)

	xs.CreateActor(machine).Start()

	assert.Contains(t, s.Calls(), []any{map[string]any{"secret": 42}})
}

// JS: action meta > should resolve dynamic params using event value
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L2354
func TestActions_ActionMeta_ShouldResolveDynamicParamsUsingEventValue(t *testing.T) {
	s := newSpy()

	machine := xs.CreateMachine(
		xs.MachineConfig[any]{
			On: map[string]xs.Transitions{
				"FOO": {{Actions: xs.Actions{xs.ActionRef{
					Type: "myAction",
					Params: xs.NewExpr(func(a xs.ExprArgs[any]) any {
						return map[string]any{"secret": a.Event.(xs.E)["secret"]}
					}),
				}}}},
			},
		},
		xs.Implementations{
			Actions: map[string]xs.Action{
				"myAction": xs.ActionFunc(func(a xs.ActionArgs[any]) { s.Call(a.Params) }),
			},
		},
	)

	actorRef := xs.CreateActor(machine).Start()

	actorRef.Send(xs.E{"type": "FOO", "secret": 77})

	assert.Contains(t, s.Calls(), []any{map[string]any{"secret": 77}})
}

// ---- forwardTo() ----

func actions2ForwardChild() *xs.StateMachine[any] {
	return xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "child",
		Initial: "active",
		States: xs.States{
			{Key: "active", On: map[string]xs.Transitions{
				"EVENT": {{
					Actions: xs.Actions{xs.SendParent(xs.Ev("SUCCESS"))},
					Guard: xs.GuardFunc(func(a xs.GuardArgs[any]) bool {
						return a.Event.(xs.E)["value"] == 42
					}),
				}},
			}},
		},
	})
}

// JS: forwardTo() > should forward an event to a service
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L2388
func TestActions_ForwardTo_ShouldForwardEventToService(t *testing.T) {
	sig := newSignal()
	child := actions2ForwardChild()

	parent := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "parent",
		Initial: "first",
		States: xs.States{
			{
				Key:    "first",
				Invoke: []xs.InvokeConfig{{Logic: child, ID: "myChild"}},
				On: map[string]xs.Transitions{
					"EVENT":   {{Actions: xs.Actions{xs.ForwardTo("myChild")}}},
					"SUCCESS": {{Target: "last"}},
				},
			},
			{Key: "last", Type: xs.Final},
		},
	})

	service := xs.CreateActor(parent)
	service.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{Complete: func() { sig.Resolve() }})
	service.Start()

	service.Send(xs.E{"type": "EVENT", "value": 42})
	sig.Wait(t)
}

// JS: forwardTo() > should forward an event to a service (dynamic)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L2448
func TestActions_ForwardTo_ShouldForwardEventToServiceDynamic(t *testing.T) {
	type ctx struct{ Child xs.ActorRef }
	sig := newSignal()

	child := actions2ForwardChild()

	parent := xs.CreateMachine(xs.MachineConfig[ctx]{
		ID:      "parent",
		Initial: "first",
		Context: ctx{Child: nil},
		States: xs.States{
			{
				Key: "first",
				Entry: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
					c := a.Context
					c.Child = a.Spawn(child, xs.SpawnOptions{ID: "x"})
					return c
				})},
				On: map[string]xs.Transitions{
					"EVENT": {{Actions: xs.Actions{xs.ForwardTo(xs.NewExpr(func(a xs.ExprArgs[ctx]) any {
						return a.Context.Child
					}))}}},
					"SUCCESS": {{Target: "last"}},
				},
			},
			{Key: "last", Type: xs.Final},
		},
	})

	service := xs.CreateActor(parent)
	service.Subscribe(xs.Observer[*xs.MachineSnapshot[ctx]]{Complete: func() { sig.Resolve() }})
	service.Start()

	service.Send(xs.E{"type": "EVENT", "value": 42})
	sig.Wait(t)
}

// JS: forwardTo() > should not cause an infinite loop when forwarding to undefined
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L2508
func TestActions_ForwardTo_ShouldNotCauseInfiniteLoopWhenForwardingToUndefined(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		On: map[string]xs.Transitions{
			"*": {{
				Guard:   xs.GuardFunc(func(xs.GuardArgs[any]) bool { return true }),
				Actions: xs.Actions{xs.ForwardTo(nil)},
			}},
		},
	})

	errorSpy := newSpy()

	actorRef := xs.CreateActor(machine)
	actorRef.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{
		Error: func(err any) { errorSpy.Call(err) },
	})
	actorRef.Start()
	actorRef.Send(xs.Ev("TEST"))

	calls := errorSpy.Calls()
	require.Len(t, calls, 1)
	require.Len(t, calls[0], 1)
	err, ok := calls[0][0].(error)
	require.True(t, ok, "expected an error value, got %T", calls[0][0])
	assert.Equal(t, "Attempted to forward event to undefined actor. This risks an infinite loop in the sender.", err.Error())
}

// actions3ClearSpyClock mirrors the JS clock literal
// `{ setTimeout, clearTimeout: spy }`: real timers, spied clearTimeout.
type actions3ClearSpyClock struct{ clearSpy *spy }

func (c *actions3ClearSpyClock) SetTimeout(fn func(), d time.Duration) xs.TimerID {
	return time.AfterFunc(d, fn)
}

func (c *actions3ClearSpyClock) ClearTimeout(id xs.TimerID) { c.clearSpy.Call(id) }

// JS: log() > should log a string
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L2535
func TestActions_Log_ShouldLogAString(t *testing.T) {
	consoleSpy := newSpy()
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Entry: xs.Actions{xs.Log("some string", "string label")},
	})
	xs.CreateActor(machine, xs.WithLogger(func(args ...any) { consoleSpy.Call(args...) })).Start()

	assert.Equal(t, [][]any{{"string label", "some string"}}, consoleSpy.Calls())
}

// JS: log() > should log an expression
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L2553
func TestActions_Log_ShouldLogAnExpression(t *testing.T) {
	type ctx struct{ Count int }
	consoleSpy := newSpy()
	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Context: ctx{Count: 42},
		Entry: xs.Actions{xs.Log(xs.NewExpr(func(a xs.ExprArgs[ctx]) any {
			return fmt.Sprintf("expr %d", a.Context.Count)
		}), "expr label")},
	})
	xs.CreateActor(machine, xs.WithLogger(func(args ...any) { consoleSpy.Call(args...) })).Start()

	assert.Equal(t, [][]any{{"expr label", "expr 42"}}, consoleSpy.Calls())
}

// JS: enqueueActions > should execute a simple referenced action
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L2576
func TestActions_EnqueueActions_ShouldExecuteASimpleReferencedAction(t *testing.T) {
	actionSpy := newSpy()
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Entry: xs.Actions{xs.EnqueueActions(func(a xs.EnqueueArgs[any]) {
			a.Enqueue(xs.ActionRef{Type: "someAction"})
		})},
	}, xs.Implementations{Actions: map[string]xs.Action{
		"someAction": xs.ActionFunc(func(xs.ActionArgs[any]) { actionSpy.Call() }),
	}})

	xs.CreateActor(machine).Start()

	assert.Equal(t, 1, actionSpy.Count())
}

// JS: enqueueActions > should execute multiple different referenced actions
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L2597
func TestActions_EnqueueActions_ShouldExecuteMultipleDifferentReferencedActions(t *testing.T) {
	spy1 := newSpy()
	spy2 := newSpy()
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Entry: xs.Actions{xs.EnqueueActions(func(a xs.EnqueueArgs[any]) {
			a.Enqueue(xs.ActionRef{Type: "someAction"})
			a.Enqueue(xs.ActionRef{Type: "otherAction"})
		})},
	}, xs.Implementations{Actions: map[string]xs.Action{
		"someAction":  xs.ActionFunc(func(xs.ActionArgs[any]) { spy1.Call() }),
		"otherAction": xs.ActionFunc(func(xs.ActionArgs[any]) { spy2.Call() }),
	}})

	xs.CreateActor(machine).Start()

	assert.Equal(t, 1, spy1.Count())
	assert.Equal(t, 1, spy2.Count())
}

// JS: enqueueActions > should execute multiple same referenced actions
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L2622
func TestActions_EnqueueActions_ShouldExecuteMultipleSameReferencedActions(t *testing.T) {
	actionSpy := newSpy()
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Entry: xs.Actions{xs.EnqueueActions(func(a xs.EnqueueArgs[any]) {
			a.Enqueue(xs.ActionRef{Type: "someAction"})
			a.Enqueue(xs.ActionRef{Type: "someAction"})
		})},
	}, xs.Implementations{Actions: map[string]xs.Action{
		"someAction": xs.ActionFunc(func(xs.ActionArgs[any]) { actionSpy.Call() }),
	}})

	xs.CreateActor(machine).Start()

	assert.Equal(t, 2, actionSpy.Count())
}

// JS: enqueueActions > should execute a parameterized action
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L2644
func TestActions_EnqueueActions_ShouldExecuteAParameterizedAction(t *testing.T) {
	actionSpy := newSpy()
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Entry: xs.Actions{xs.EnqueueActions(func(a xs.EnqueueArgs[any]) {
			a.Enqueue(xs.ActionRef{Type: "someAction", Params: map[string]any{"answer": 42}})
		})},
	}, xs.Implementations{Actions: map[string]xs.Action{
		"someAction": xs.ActionFunc(func(a xs.ActionArgs[any]) { actionSpy.Call(a.Params) }),
	}})

	xs.CreateActor(machine).Start()

	assert.Equal(t, [][]any{{map[string]any{"answer": 42}}}, actionSpy.Calls())
}

// JS: enqueueActions > should execute a function
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L2676
func TestActions_EnqueueActions_ShouldExecuteAFunction(t *testing.T) {
	actionSpy := newSpy()
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Entry: xs.Actions{xs.EnqueueActions(func(a xs.EnqueueArgs[any]) {
			a.Enqueue(xs.ActionFunc(func(xs.ActionArgs[any]) { actionSpy.Call() }))
		})},
	})

	xs.CreateActor(machine).Start()

	assert.Equal(t, 1, actionSpy.Count())
}

// JS: enqueueActions > should execute a builtin action using its own action creator
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L2690
func TestActions_EnqueueActions_ShouldExecuteBuiltinActionUsingItsOwnActionCreator(t *testing.T) {
	actionSpy := newSpy()
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		On: map[string]xs.Transitions{
			"FOO": {{Actions: xs.Actions{xs.EnqueueActions(func(a xs.EnqueueArgs[any]) {
				a.Enqueue(xs.Raise(xs.Ev("RAISED")))
			})}}},
			"RAISED": {{Actions: xs.Actions{xs.ActionFunc(func(xs.ActionArgs[any]) { actionSpy.Call() })}}},
		},
	})

	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.Ev("FOO"))

	assert.Equal(t, 1, actionSpy.Count())
}

// JS: enqueueActions > should execute a builtin action using its bound action creator
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L2717
func TestActions_EnqueueActions_ShouldExecuteBuiltinActionUsingItsBoundActionCreator(t *testing.T) {
	// The Go contract has no bound creators (`enqueue.raise(...)`); the
	// equivalent is enqueuing the built-in action: a.Enqueue(xs.Raise(...)).
	actionSpy := newSpy()
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		On: map[string]xs.Transitions{
			"FOO": {{Actions: xs.Actions{xs.EnqueueActions(func(a xs.EnqueueArgs[any]) {
				a.Enqueue(xs.Raise(xs.Ev("RAISED")))
			})}}},
			"RAISED": {{Actions: xs.Actions{xs.ActionFunc(func(xs.ActionArgs[any]) { actionSpy.Call() })}}},
		},
	})

	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.Ev("FOO"))

	assert.Equal(t, 1, actionSpy.Count())
}

// JS: enqueueActions > should execute assigns when resolving the initial snapshot
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L2742
func TestActions_EnqueueActions_ShouldExecuteAssignsWhenResolvingInitialSnapshot(t *testing.T) {
	type ctx struct{ Count int }
	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Context: ctx{Count: 0},
		Entry: xs.Actions{xs.EnqueueActions(func(a xs.EnqueueArgs[ctx]) {
			// enqueue.assign({ count: 42 })
			a.Enqueue(xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
				c := a.Context
				c.Count = 42
				return c
			}))
		})},
	})

	snapshot := xs.CreateActor(machine).GetSnapshot()

	assert.Equal(t, ctx{Count: 42}, snapshot.Context)
}

// JS: enqueueActions > should be able to check a simple referenced guard
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L2759
func TestActions_EnqueueActions_ShouldBeAbleToCheckASimpleReferencedGuard(t *testing.T) {
	type ctx struct{ Count int }
	guardSpy := newSpy()
	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Context: ctx{Count: 0},
		Entry: xs.Actions{xs.EnqueueActions(func(a xs.EnqueueArgs[ctx]) {
			a.Check(xs.GuardRef{Type: "alwaysTrue"})
		})},
	}, xs.Implementations{Guards: map[string]xs.Guard{
		"alwaysTrue": xs.GuardFunc(func(xs.GuardArgs[ctx]) bool {
			guardSpy.Call()
			return true
		}),
	}})

	xs.CreateActor(machine)

	assert.Equal(t, 1, guardSpy.Count())
}

// JS: enqueueActions > should be able to check a parameterized guard
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L2782
func TestActions_EnqueueActions_ShouldBeAbleToCheckAParameterizedGuard(t *testing.T) {
	type ctx struct{ Count int }
	guardSpy := newSpy()
	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Context: ctx{Count: 0},
		Entry: xs.Actions{xs.EnqueueActions(func(a xs.EnqueueArgs[ctx]) {
			a.Check(xs.GuardRef{Type: "alwaysTrue", Params: map[string]any{"max": 100}})
		})},
	}, xs.Implementations{Guards: map[string]xs.Guard{
		"alwaysTrue": xs.GuardFunc(func(a xs.GuardArgs[ctx]) bool {
			guardSpy.Call(a.Params)
			return true
		}),
	}})

	xs.CreateActor(machine)

	assert.Equal(t, [][]any{{map[string]any{"max": 100}}}, guardSpy.Calls())
}

// JS: enqueueActions > should provide self
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L2822
func TestActions_EnqueueActions_ShouldProvideSelf(t *testing.T) {
	// expect.assertions(1): the assertion inside the action must run exactly once.
	var assertions atomic.Int32
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Entry: xs.Actions{xs.EnqueueActions(func(a xs.EnqueueArgs[any]) {
			assertions.Add(1)
			assert.NotNil(t, a.Self)
		})},
	})

	xs.CreateActor(machine).Start()

	assert.Equal(t, int32(1), assertions.Load())
}

// JS: enqueueActions > should be able to communicate with the parent using params
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L2833
func TestActions_EnqueueActions_ShouldBeAbleToCommunicateWithParentUsingParams(t *testing.T) {
	type childInput struct{ Parent xs.ActorRef }
	type childCtx struct{ Parent xs.ActorRef }

	childMachine := xs.NewSetup[childCtx](xs.Implementations{Actions: map[string]xs.Action{
		"mySendParent": xs.EnqueueActions(func(a xs.EnqueueArgs[childCtx]) {
			event := a.Params.(xs.Event)
			if a.Context.Parent == nil {
				// it's here just for illustration purposes
				t.Log("WARN: an attempt to send an event to a non-existent parent")
				return
			}
			a.Enqueue(xs.SendTo(a.Context.Parent, event))
		}),
	}}).CreateMachine(xs.MachineConfig[childCtx]{
		ContextFn: func(a xs.ContextArgs) childCtx {
			return childCtx{Parent: a.Input.(childInput).Parent}
		},
		Entry: xs.Actions{xs.ActionRef{Type: "mySendParent", Params: xs.Ev("FOO")}},
	})

	parentSpy := newSpy()

	parentMachine := xs.NewSetup[any](xs.Implementations{Actors: map[string]xs.ActorLogic{
		"child": childMachine,
	}}).CreateMachine(xs.MachineConfig[any]{
		On: map[string]xs.Transitions{
			"FOO": {{Actions: xs.Actions{xs.ActionFunc(func(xs.ActionArgs[any]) { parentSpy.Call() })}}},
		},
		Invoke: []xs.InvokeConfig{{
			Src: "child",
			Input: xs.NewExpr(func(a xs.ExprArgs[any]) any {
				return childInput{Parent: a.Self}
			}),
		}},
	})

	xs.CreateActor(parentMachine).Start()

	assert.Equal(t, 1, parentSpy.Count())
}

// JS: enqueueActions > should enqueue.sendParent
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L2893
func TestActions_EnqueueActions_ShouldEnqueueSendParent(t *testing.T) {
	childMachine := xs.NewSetup[any](xs.Implementations{Actions: map[string]xs.Action{
		"sendToParent": xs.EnqueueActions(func(a xs.EnqueueArgs[any]) {
			// enqueue.sendParent({ type: 'PARENT_EVENT' })
			a.Enqueue(xs.SendParent(xs.Ev("PARENT_EVENT")))
		}),
	}}).CreateMachine(xs.MachineConfig[any]{
		Entry: xs.Actions{xs.ActionRef{Type: "sendToParent"}},
	})

	parentSpy := newSpy()

	parentMachine := xs.NewSetup[any](xs.Implementations{Actors: map[string]xs.ActorLogic{
		"child": childMachine,
	}}).CreateMachine(xs.MachineConfig[any]{
		On: map[string]xs.Transitions{
			"PARENT_EVENT": {{Actions: xs.Actions{xs.ActionFunc(func(xs.ActionArgs[any]) { parentSpy.Call() })}}},
		},
		Invoke: []xs.InvokeConfig{{Src: "child"}},
	})

	xs.CreateActor(parentMachine).Start()

	assert.Equal(t, 1, parentSpy.Count())
}

// JS: sendParent > TS: should compile for any event
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L2941
func TestActions_SendParent_TSShouldCompileForAnyEvent(t *testing.T) {
	// The TS part (no type error for an event outside the child's events) has
	// no Go meaning; the runtime expectation is that the machine is created.
	child := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "child",
		Initial: "start",
		States: xs.States{
			{Key: "start", Entry: xs.Actions{xs.SendParent(xs.Ev("PARENT"))}},
		},
	})

	assert.NotNil(t, child)
}

// JS: sendTo > should be able to send an event to an actor
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L2965
func TestActions_SendTo_ShouldBeAbleToSendAnEventToAnActor(t *testing.T) {
	sig := newSignal()
	childMachine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "waiting",
		States: xs.States{
			{Key: "waiting", On: map[string]xs.Transitions{
				"EVENT": {{Actions: xs.Actions{xs.ActionFunc(func(xs.ActionArgs[any]) { sig.Resolve() })}}},
			}},
		},
	})

	type ctx struct{ Child xs.ActorRef }
	parentMachine := xs.CreateMachine(xs.MachineConfig[ctx]{
		ContextFn: func(a xs.ContextArgs) ctx {
			return ctx{Child: a.Spawn(childMachine)}
		},
		Entry: xs.Actions{xs.SendTo(
			xs.NewExpr(func(a xs.ExprArgs[ctx]) any { return a.Context.Child }),
			xs.Ev("EVENT"),
		)},
	})

	xs.CreateActor(parentMachine).Start()
	sig.Wait(t)
}

// JS: sendTo > should be able to send an event from expression to an actor
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L2999
func TestActions_SendTo_ShouldBeAbleToSendAnEventFromExpressionToAnActor(t *testing.T) {
	sig := newSignal()
	childMachine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "waiting",
		States: xs.States{
			{Key: "waiting", On: map[string]xs.Transitions{
				"EVENT": {{Actions: xs.Actions{xs.ActionFunc(func(xs.ActionArgs[any]) { sig.Resolve() })}}},
			}},
		},
	})

	type ctx struct {
		Child xs.ActorRef
		Count int
	}
	parentMachine := xs.CreateMachine(xs.MachineConfig[ctx]{
		ContextFn: func(a xs.ContextArgs) ctx {
			return ctx{
				Child: a.Spawn(childMachine, xs.SpawnOptions{ID: "child"}),
				Count: 42,
			}
		},
		Entry: xs.Actions{xs.SendTo(
			xs.NewExpr(func(a xs.ExprArgs[ctx]) any { return a.Context.Child }),
			xs.NewExpr(func(a xs.ExprArgs[ctx]) any {
				return xs.E{"type": "EVENT", "count": a.Context.Count}
			}),
		)},
	})

	xs.CreateActor(parentMachine).Start()
	sig.Wait(t)
}

// JS: sendTo > should report a type error for an invalid event
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L3040
func TestActions_SendTo_ShouldReportATypeErrorForAnInvalidEvent(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error that sendTo rejects an event type the child does not accept")
}

// JS: sendTo > should be able to send an event to a named actor
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L3071
func TestActions_SendTo_ShouldBeAbleToSendAnEventToANamedActor(t *testing.T) {
	sig := newSignal()
	childMachine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "waiting",
		States: xs.States{
			{Key: "waiting", On: map[string]xs.Transitions{
				"EVENT": {{Actions: xs.Actions{xs.ActionFunc(func(xs.ActionArgs[any]) { sig.Resolve() })}}},
			}},
		},
	})

	type ctx struct{ Child xs.ActorRef }
	parentMachine := xs.CreateMachine(xs.MachineConfig[ctx]{
		ContextFn: func(a xs.ContextArgs) ctx {
			return ctx{Child: a.Spawn(childMachine, xs.SpawnOptions{ID: "child"})}
		},
		// No type-safety for the event yet
		Entry: xs.Actions{xs.SendTo("child", xs.Ev("EVENT"))},
	})

	xs.CreateActor(parentMachine).Start()
	sig.Wait(t)
}

// JS: sendTo > should be able to send an event directly to an ActorRef
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L3104
func TestActions_SendTo_ShouldBeAbleToSendAnEventDirectlyToAnActorRef(t *testing.T) {
	sig := newSignal()
	childMachine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "waiting",
		States: xs.States{
			{Key: "waiting", On: map[string]xs.Transitions{
				"EVENT": {{Actions: xs.Actions{xs.ActionFunc(func(xs.ActionArgs[any]) { sig.Resolve() })}}},
			}},
		},
	})

	type ctx struct{ Child xs.ActorRef }
	parentMachine := xs.CreateMachine(xs.MachineConfig[ctx]{
		ContextFn: func(a xs.ContextArgs) ctx {
			return ctx{Child: a.Spawn(childMachine)}
		},
		Entry: xs.Actions{xs.SendTo(
			xs.NewExpr(func(a xs.ExprArgs[ctx]) any { return a.Context.Child }),
			xs.Ev("EVENT"),
		)},
	})

	xs.CreateActor(parentMachine).Start()
	sig.Wait(t)
}

// JS: sendTo > should be able to read from event
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L3136
func TestActions_SendTo_ShouldBeAbleToReadFromEvent(t *testing.T) {
	// expect.assertions(1): the assertion inside receive must run exactly once.
	var assertions atomic.Int32
	type ctx = map[string]xs.ActorRef
	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Initial: "a",
		ContextFn: func(a xs.ContextArgs) ctx {
			return ctx{
				"foo": a.Spawn(xs.FromCallback(func(c xs.CallbackArgs) func() {
					c.Receive(func(event xs.Event) {
						assertions.Add(1)
						assert.Equal(t, xs.Ev("EVENT"), event)
					})
					return nil
				})),
			}
		},
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{
				"EVENT": {{Actions: xs.Actions{xs.SendTo(
					xs.NewExpr(func(a xs.ExprArgs[ctx]) any {
						return a.Context[a.Event.(xs.E)["value"].(string)]
					}),
					xs.Ev("EVENT"),
				)}}},
			}},
		},
	})

	service := xs.CreateActor(machine).Start()

	service.Send(xs.E{"type": "EVENT", "value": "foo"})

	assert.Equal(t, int32(1), assertions.Load())
}

// JS: sendTo > should error if given a string
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L3171
func TestActions_SendTo_ShouldErrorIfGivenAString(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Invoke: []xs.InvokeConfig{{
			ID:    "child",
			Logic: xs.FromCallback(func(xs.CallbackArgs) func() { return nil }),
		}},
		Entry: xs.Actions{xs.SendTo("child", "a string")},
	})

	errorSpy := newSpy()

	actorRef := xs.CreateActor(machine)
	actorRef.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{
		Error: func(err any) { errorSpy.Call(err) },
	})
	actorRef.Start()

	calls := errorSpy.Calls()
	require.Len(t, calls, 1)
	require.Len(t, calls[0], 1)
	err, ok := calls[0][0].(error)
	require.True(t, ok, "expected an error value, got %#v", calls[0][0])
	assert.Equal(t, `Only event objects may be used with sendTo; use sendTo({ type: "a string" }) instead`, err.Error())
}

// JS: sendTo > a self-event "handler" of an event sent using sendTo should be able to read updated snapshot of self
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L3197
func TestActions_SendTo_SelfEventHandlerShouldReadUpdatedSnapshotOfSelf(t *testing.T) {
	type ctx struct{ Counter int }
	handlerSpy := newSpy()
	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Context: ctx{Counter: 0},
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{"NEXT": {{Target: "b"}}}},
			{
				Key: "b",
				Entry: xs.Actions{
					xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
						c := a.Context
						c.Counter = 1
						return c
					}),
					xs.SendTo(xs.NewExpr(func(a xs.ExprArgs[ctx]) any { return a.Self }), xs.Ev("EVENT")),
				},
				On: map[string]xs.Transitions{
					"EVENT": {{
						Actions: xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[ctx]) {
							handlerSpy.Call(machineSnap[ctx](a.Self).Context)
						})},
						Target: "c",
					}},
				},
			},
			{Key: "c"},
		},
	})

	actorRef := xs.CreateActor(machine).Start()

	actorRef.Send(xs.Ev("NEXT"))
	actorRef.Send(xs.Ev("EVENT"))

	assert.Equal(t, [][]any{{ctx{Counter: 1}}}, handlerSpy.Calls())
}

// JS: sendTo > should not attempt to deliver a delayed event to the spawned actor's ID that was stopped since the event was scheduled
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L3240
func TestActions_SendTo_ShouldNotDeliverDelayedEventToStoppedSpawnedActorID(t *testing.T) {
	warnSpy := newSpy()
	spy1 := newSpy()

	child1 := xs.CreateMachine(xs.MachineConfig[any]{
		On: map[string]xs.Transitions{
			"PING": {{Actions: xs.Actions{xs.ActionFunc(func(xs.ActionArgs[any]) { spy1.Call() })}}},
		},
	})

	spy2 := newSpy()

	child2 := xs.CreateMachine(xs.MachineConfig[any]{
		On: map[string]xs.Transitions{
			"PING": {{Actions: xs.Actions{xs.ActionFunc(func(xs.ActionArgs[any]) { spy2.Call() })}}},
		},
	})

	machine := xs.NewSetup[any](xs.Implementations{Actors: map[string]xs.ActorLogic{
		"child1": child1,
		"child2": child2,
	}}).CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{"START": {{Target: "b"}}}},
			{Key: "b", Entry: xs.Actions{
				xs.SpawnChild("child1", xs.SpawnOptions{ID: "myChild"}),
				xs.SendTo("myChild", xs.Ev("PING"), xs.SendOptions{Delay: ms(1)}),
				xs.StopChild("myChild"),
				xs.SpawnChild("child2", xs.SpawnOptions{ID: "myChild"}),
			}},
		},
	})

	// The JS inline snapshot hardcodes the session id of the first "myChild"
	// actor (child1); capture it to build the same message.
	var mu sync.Mutex
	firstChildSessionID := ""
	actorRef := xs.CreateActor(machine,
		xs.WithWarnHandler(func(args ...any) { warnSpy.Call(args...) }),
		xs.WithInspect(func(ev xs.InspectionEvent) {
			if ev.Type == xs.InspectActor && ev.ActorRef != nil && ev.ActorRef.ID() == "myChild" {
				mu.Lock()
				if firstChildSessionID == "" {
					firstChildSessionID = ev.ActorRef.SessionID()
				}
				mu.Unlock()
			}
		}),
	).Start()

	actorRef.Send(xs.Ev("START"))

	sleep(10)

	assert.Equal(t, 0, spy1.Count())
	assert.Equal(t, 0, spy2.Count())

	mu.Lock()
	sid := firstChildSessionID
	mu.Unlock()
	assert.Equal(t, [][]any{{fmt.Sprintf(
		"Event \"PING\" was sent to stopped actor \"myChild (%s)\". This actor has already reached its final state, and will not transition.\nEvent: {\"type\":\"PING\"}",
		sid,
	)}}, warnSpy.Calls())
}

// JS: sendTo > should not attempt to deliver a delayed event to the invoked actor's ID that was stopped since the event was scheduled
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L3309
func TestActions_SendTo_ShouldNotDeliverDelayedEventToStoppedInvokedActorID(t *testing.T) {
	warnSpy := newSpy()
	spy1 := newSpy()

	child1 := xs.CreateMachine(xs.MachineConfig[any]{
		On: map[string]xs.Transitions{
			"PING": {{Actions: xs.Actions{xs.ActionFunc(func(xs.ActionArgs[any]) { spy1.Call() })}}},
		},
	})

	spy2 := newSpy()

	child2 := xs.CreateMachine(xs.MachineConfig[any]{
		On: map[string]xs.Transitions{
			"PING": {{Actions: xs.Actions{xs.ActionFunc(func(xs.ActionArgs[any]) { spy2.Call() })}}},
		},
	})

	machine := xs.NewSetup[any](xs.Implementations{Actors: map[string]xs.ActorLogic{
		"child1": child1,
		"child2": child2,
	}}).CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{"START": {{Target: "b"}}}},
			{
				Key:    "b",
				Entry:  xs.Actions{xs.SendTo("myChild", xs.Ev("PING"), xs.SendOptions{Delay: ms(1)})},
				Invoke: []xs.InvokeConfig{{Src: "child1", ID: "myChild"}},
				On:     map[string]xs.Transitions{"NEXT": {{Target: "c"}}},
			},
			{
				Key:    "c",
				Invoke: []xs.InvokeConfig{{Src: "child2", ID: "myChild"}},
			},
		},
	})

	// The JS inline snapshot hardcodes the session id of the first "myChild"
	// actor (child1); capture it to build the same message.
	var mu sync.Mutex
	firstChildSessionID := ""
	actorRef := xs.CreateActor(machine,
		xs.WithWarnHandler(func(args ...any) { warnSpy.Call(args...) }),
		xs.WithInspect(func(ev xs.InspectionEvent) {
			if ev.Type == xs.InspectActor && ev.ActorRef != nil && ev.ActorRef.ID() == "myChild" {
				mu.Lock()
				if firstChildSessionID == "" {
					firstChildSessionID = ev.ActorRef.SessionID()
				}
				mu.Unlock()
			}
		}),
	).Start()

	actorRef.Send(xs.Ev("START"))
	actorRef.Send(xs.Ev("NEXT"))

	sleep(10)

	assert.Equal(t, 0, spy1.Count())
	assert.Equal(t, 0, spy2.Count())

	mu.Lock()
	sid := firstChildSessionID
	mu.Unlock()
	assert.Equal(t, [][]any{{fmt.Sprintf(
		"Event \"PING\" was sent to stopped actor \"myChild (%s)\". This actor has already reached its final state, and will not transition.\nEvent: {\"type\":\"PING\"}",
		sid,
	)}}, warnSpy.Calls())
}

// JS: raise > should be able to send a delayed event to itself
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L3385
func TestActions_Raise_ShouldBeAbleToSendADelayedEventToItself(t *testing.T) {
	sig := newSignal()
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{
				Key:   "a",
				Entry: xs.Actions{xs.Raise(xs.Ev("EVENT"), xs.SendOptions{Delay: ms(1)})},
				On:    map[string]xs.Transitions{"TO_B": {{Target: "b"}}},
			},
			{Key: "b", On: map[string]xs.Transitions{"EVENT": {{Target: "c"}}}},
			{Key: "c", Type: xs.Final},
		},
	})

	service := xs.CreateActor(machine).Start()

	service.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{Complete: func() { sig.Resolve() }})

	// Ensures that the delayed self-event is sent when in the `b` state
	service.Send(xs.Ev("TO_B"))
	sig.Wait(t)
}

// JS: raise > should be able to send a delayed event to itself with delay = 0
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L3421
func TestActions_Raise_ShouldBeAbleToSendADelayedEventToItselfWithDelay0(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{
				Key:   "a",
				Entry: xs.Actions{xs.Raise(xs.Ev("EVENT"), xs.SendOptions{Delay: ms(0)})},
				On:    map[string]xs.Transitions{"EVENT": {{Target: "b"}}},
			},
			{Key: "b"},
		},
	})

	service := xs.CreateActor(machine).Start()

	// The state should not be changed yet; `delay: 0` is equivalent to `setTimeout(..., 0)`
	assert.Equal(t, "a", service.GetSnapshot().Value)

	// JS awaits sleep(0), a macrotask queued after the zero-delay timer. Go
	// timers fire on other goroutines, so wait a few ms for the same ordering.
	sleep(10)
	// The state should be changed now
	assert.Equal(t, "b", service.GetSnapshot().Value)
}

// JS: raise > should be able to raise an event and respond to it in the same state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L3450
func TestActions_Raise_ShouldBeAbleToRaiseAnEventAndRespondToItInTheSameState(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{
				Key:   "a",
				Entry: xs.Actions{xs.Raise(xs.Ev("TO_B"))},
				On:    map[string]xs.Transitions{"TO_B": {{Target: "b"}}},
			},
			{Key: "b", Type: xs.Final},
		},
	})

	service := xs.CreateActor(machine).Start()

	assert.Equal(t, "b", service.GetSnapshot().Value)
}

// JS: raise > should be able to raise a delayed event and respond to it in the same state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L3471
func TestActions_Raise_ShouldBeAbleToRaiseADelayedEventAndRespondToItInTheSameState(t *testing.T) {
	sig := newSignal()
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{
				Key:   "a",
				Entry: xs.Actions{xs.Raise(xs.Ev("TO_B"), xs.SendOptions{Delay: ms(100)})},
				On:    map[string]xs.Transitions{"TO_B": {{Target: "b"}}},
			},
			{Key: "b", Type: xs.Final},
		},
	})

	service := xs.CreateActor(machine).Start()

	service.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{Complete: func() { sig.Resolve() }})

	sleep(50)

	// didn't transition yet
	assert.Equal(t, "a", service.GetSnapshot().Value)

	sig.Wait(t)
}

// JS: raise > should accept event expression
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L3505
func TestActions_Raise_ShouldAcceptEventExpression(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{
				"NEXT": {{Actions: xs.Actions{xs.Raise(xs.NewExpr(func(xs.ExprArgs[any]) any {
					return xs.Ev("RAISED")
				}))}}},
				"RAISED": {{Target: "b"}},
			}},
			{Key: "b"},
		},
	})

	actor := xs.CreateActor(machine).Start()

	actor.Send(xs.Ev("NEXT"))

	assert.Equal(t, "b", actor.GetSnapshot().Value)
}

// JS: raise > should be possible to access context in the event expression
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L3528
func TestActions_Raise_ShouldBePossibleToAccessContextInTheEventExpression(t *testing.T) {
	type ctx struct{ EventType string }
	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Initial: "a",
		Context: ctx{EventType: "RAISED"},
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{
				"NEXT": {{Actions: xs.Actions{xs.Raise(xs.NewExpr(func(a xs.ExprArgs[ctx]) any {
					return xs.Ev(a.Context.EventType)
				}))}}},
				"RAISED": {{Target: "b"}},
			}},
			{Key: "b"},
		},
	})

	actor := xs.CreateActor(machine).Start()

	actor.Send(xs.Ev("NEXT"))

	assert.Equal(t, "b", actor.GetSnapshot().Value)
}

// JS: raise > should error if given a string
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L3567
func TestActions_Raise_ShouldErrorIfGivenAString(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Entry: xs.Actions{xs.Raise("a string")},
	})

	errorSpy := newSpy()

	actorRef := xs.CreateActor(machine)
	actorRef.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{
		Error: func(err any) { errorSpy.Call(err) },
	})
	actorRef.Start()

	calls := errorSpy.Calls()
	require.Len(t, calls, 1)
	require.Len(t, calls[0], 1)
	err, ok := calls[0][0].(error)
	require.True(t, ok, "expected an error value, got %#v", calls[0][0])
	assert.Equal(t, `Only event objects may be used with raise; use raise({ type: "a string" }) instead`, err.Error())
}

// JS: cancel > should be possible to cancel a raised delayed event
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L3594
func TestActions_Cancel_ShouldBePossibleToCancelARaisedDelayedEvent(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{
				"NEXT": {{Actions: xs.Actions{
					xs.Raise(xs.Ev("RAISED"), xs.SendOptions{Delay: ms(1), ID: "myId"}),
				}}},
				"RAISED": {{Target: "b"}},
				"CANCEL": {{Actions: xs.Actions{xs.Cancel("myId")}}},
			}},
			{Key: "b"},
		},
	})

	actor := xs.CreateActor(machine).Start()

	// This should raise the 'RAISED' event after 1ms
	actor.Send(xs.Ev("NEXT"))

	// This should cancel the 'RAISED' event
	actor.Send(xs.Ev("CANCEL"))

	sleep(10)
	assert.Equal(t, "a", actor.GetSnapshot().Value)
}

// JS: cancel > should cancel only the delayed event in the machine that scheduled it when canceling the event with the same ID in the machine that sent it first
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L3625
func TestActions_Cancel_ShouldCancelOnlyInSchedulingMachineWhenSameIDSentFirst(t *testing.T) {
	fooSpy := newSpy()
	barSpy := newSpy()

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Invoke: []xs.InvokeConfig{
			{
				ID: "foo",
				Logic: xs.CreateMachine(xs.MachineConfig[any]{
					ID:    "foo",
					Entry: xs.Actions{xs.Raise(xs.Ev("event"), xs.SendOptions{ID: "sameId", Delay: ms(100)})},
					On: map[string]xs.Transitions{
						"event":  {{Actions: xs.Actions{xs.ActionFunc(func(xs.ActionArgs[any]) { fooSpy.Call() })}}},
						"cancel": {{Actions: xs.Actions{xs.Cancel("sameId")}}},
					},
				}),
			},
			{
				ID: "bar",
				Logic: xs.CreateMachine(xs.MachineConfig[any]{
					ID:    "bar",
					Entry: xs.Actions{xs.Raise(xs.Ev("event"), xs.SendOptions{ID: "sameId", Delay: ms(100)})},
					On: map[string]xs.Transitions{
						"event": {{Actions: xs.Actions{xs.ActionFunc(func(xs.ActionArgs[any]) { barSpy.Call() })}}},
					},
				}),
			},
		},
		On: map[string]xs.Transitions{
			"cancelFoo": {{Actions: xs.Actions{xs.SendTo("foo", xs.Ev("cancel"))}}},
		},
	})
	clock := xs.NewSimulatedClock()
	actor := xs.CreateActor(machine, xs.WithClock(clock)).Start()
	t.Cleanup(func() { actor.Stop() })

	clock.Increment(ms(50))

	// This will cause the foo actor to cancel its 'sameId' delayed event
	// This should NOT cancel the 'sameId' delayed event in the other actor
	actor.Send(xs.Ev("cancelFoo"))

	clock.Increment(ms(55))

	assert.Equal(t, 0, fooSpy.Count())
	assert.Equal(t, 1, barSpy.Count())
}

// JS: cancel > should cancel only the delayed event in the machine that scheduled it when canceling the event with the same ID in the machine that sent it second
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L3673
func TestActions_Cancel_ShouldCancelOnlyInSchedulingMachineWhenSameIDSentSecond(t *testing.T) {
	fooSpy := newSpy()
	barSpy := newSpy()

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Invoke: []xs.InvokeConfig{
			{
				ID: "foo",
				Logic: xs.CreateMachine(xs.MachineConfig[any]{
					ID:    "foo",
					Entry: xs.Actions{xs.Raise(xs.Ev("event"), xs.SendOptions{ID: "sameId", Delay: ms(100)})},
					On: map[string]xs.Transitions{
						"event": {{Actions: xs.Actions{xs.ActionFunc(func(xs.ActionArgs[any]) { fooSpy.Call() })}}},
					},
				}),
			},
			{
				ID: "bar",
				Logic: xs.CreateMachine(xs.MachineConfig[any]{
					ID:    "bar",
					Entry: xs.Actions{xs.Raise(xs.Ev("event"), xs.SendOptions{ID: "sameId", Delay: ms(100)})},
					On: map[string]xs.Transitions{
						"event":  {{Actions: xs.Actions{xs.ActionFunc(func(xs.ActionArgs[any]) { barSpy.Call() })}}},
						"cancel": {{Actions: xs.Actions{xs.Cancel("sameId")}}},
					},
				}),
			},
		},
		On: map[string]xs.Transitions{
			"cancelBar": {{Actions: xs.Actions{xs.SendTo("bar", xs.Ev("cancel"))}}},
		},
	})
	clock := xs.NewSimulatedClock()
	actor := xs.CreateActor(machine, xs.WithClock(clock)).Start()
	t.Cleanup(func() { actor.Stop() })

	clock.Increment(ms(50))

	// This will cause the bar actor to cancel its 'sameId' delayed event
	// This should NOT cancel the 'sameId' delayed event in the other actor
	actor.Send(xs.Ev("cancelBar"))

	clock.Increment(ms(55))

	assert.Equal(t, 1, fooSpy.Count())
	assert.Equal(t, 0, barSpy.Count())
}

// JS: cancel > should not try to clear an undefined timeout when canceling an unscheduled timer
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L3721
func TestActions_Cancel_ShouldNotClearUndefinedTimeoutWhenCancelingUnscheduledTimer(t *testing.T) {
	clearSpy := newSpy()

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		On: map[string]xs.Transitions{
			"FOO": {{Actions: xs.Actions{xs.Cancel("foo")}}},
		},
	})

	actorRef := xs.CreateActor(machine, xs.WithClock(&actions3ClearSpyClock{clearSpy: clearSpy})).Start()

	actorRef.Send(xs.Ev("FOO"))

	assert.Equal(t, 0, clearSpy.Count())
}

// JS: cancel > should be able to cancel a just scheduled delayed event to a just invoked child
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L3746
func TestActions_Cancel_ShouldBeAbleToCancelJustScheduledDelayedEventToJustInvokedChild(t *testing.T) {
	pingSpy := newSpy()

	child := xs.CreateMachine(xs.MachineConfig[any]{
		On: map[string]xs.Transitions{
			"PING": {{Actions: xs.Actions{xs.ActionFunc(func(xs.ActionArgs[any]) { pingSpy.Call() })}}},
		},
	})

	machine := xs.NewSetup[any](xs.Implementations{Actors: map[string]xs.ActorLogic{
		"child": child,
	}}).CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{"START": {{Target: "b"}}}},
			{
				Key: "b",
				Entry: xs.Actions{
					xs.SendTo("myChild", xs.Ev("PING"), xs.SendOptions{ID: "myEvent", Delay: ms(0)}),
					xs.Cancel("myEvent"),
				},
				Invoke: []xs.InvokeConfig{{Src: "child", ID: "myChild"}},
			},
		},
	})

	actorRef := xs.CreateActor(machine).Start()

	actorRef.Send(xs.Ev("START"))

	sleep(10)
	assert.Equal(t, 0, pingSpy.Count())
}

// JS: cancel > should not be able to cancel a just scheduled non-delayed event to a just invoked child
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L3792
func TestActions_Cancel_ShouldNotBeAbleToCancelJustScheduledNonDelayedEventToJustInvokedChild(t *testing.T) {
	s := newSpy()

	child := xs.CreateMachine(xs.MachineConfig[any]{
		On: map[string]xs.Transitions{
			"PING": {{Actions: xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[any]) { s.Call(a.Event) })}}},
		},
	})

	machine := xs.NewSetup[any](xs.Implementations{
		Actors: map[string]xs.ActorLogic{"child": child},
	}).CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{"START": {{Target: "b"}}}},
			{
				Key: "b",
				Entry: xs.Actions{
					xs.SendTo("myChild", xs.Ev("PING"), xs.SendOptions{ID: "myEvent"}),
					xs.Cancel("myEvent"),
				},
				Invoke: []xs.InvokeConfig{{Src: "child", ID: "myChild"}},
			},
		},
	})

	actorRef := xs.CreateActor(machine).Start()

	actorRef.Send(xs.Ev("START"))

	assert.Equal(t, 1, s.Count())
}

// JS: assign action order > should preserve action order
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L3839
func TestActions_AssignActionOrder_ShouldPreserveActionOrder(t *testing.T) {
	type ctx struct{ Count int }
	captured := newSpy()

	capture := xs.ActionFunc(func(a xs.ActionArgs[ctx]) { captured.Call(a.Context.Count) })
	inc := func() xs.Action {
		return xs.Assign(func(a xs.AssignArgs[ctx]) ctx { return ctx{Count: a.Context.Count + 1} })
	}

	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Context: ctx{Count: 0},
		Entry: xs.Actions{
			capture, // 0
			inc(),
			capture, // 1
			inc(),
			capture, // 2
		},
	})

	actor := xs.CreateActor(machine).Start()

	assert.Equal(t, ctx{Count: 2}, actor.GetSnapshot().Context)

	assert.Equal(t, [][]any{{0}, {1}, {2}}, captured.Calls())
}

// JS: assign action order > should deeply preserve action order
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L3863
func TestActions_AssignActionOrder_ShouldDeeplyPreserveActionOrder(t *testing.T) {
	type countCtx struct{ Count int }
	captured := newSpy()

	machine := xs.CreateMachine(xs.MachineConfig[countCtx]{
		Context: countCtx{Count: 0},
		Entry: xs.Actions{
			xs.ActionFunc(func(a xs.ActionArgs[countCtx]) { captured.Call(a.Context.Count) }), // 0
			xs.EnqueueActions(func(a xs.EnqueueArgs[countCtx]) {
				a.Enqueue(xs.Assign(func(a xs.AssignArgs[countCtx]) countCtx {
					return countCtx{Count: a.Context.Count + 1}
				}))
				a.Enqueue(xs.ActionRef{Type: "capture"})
				a.Enqueue(xs.Assign(func(a xs.AssignArgs[countCtx]) countCtx {
					return countCtx{Count: a.Context.Count + 1}
				}))
			}),
			xs.ActionFunc(func(a xs.ActionArgs[countCtx]) { captured.Call(a.Context.Count) }), // 2
		},
	}, xs.Implementations{
		Actions: map[string]xs.Action{
			"capture": xs.ActionFunc(func(a xs.ActionArgs[countCtx]) { captured.Call(a.Context.Count) }),
		},
	})

	xs.CreateActor(machine).Start()

	assert.Equal(t, [][]any{{0}, {1}, {2}}, captured.Calls())
}

// JS: assign action order > should capture correct context values on subsequent transitions
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L3898
func TestActions_AssignActionOrder_ShouldCaptureCorrectContextValuesOnSubsequentTransitions(t *testing.T) {
	type ctx struct{ Counter int }
	captured := newSpy()

	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Context: ctx{Counter: 0},
		On: map[string]xs.Transitions{
			"EV": {{Actions: xs.Actions{
				xs.Assign(func(a xs.AssignArgs[ctx]) ctx { return ctx{Counter: a.Context.Counter + 1} }),
				xs.ActionFunc(func(a xs.ActionArgs[ctx]) { captured.Call(a.Context.Counter) }),
			}}},
		},
	})

	service := xs.CreateActor(machine).Start()

	service.Send(xs.Ev("EV"))
	service.Send(xs.Ev("EV"))

	assert.Equal(t, [][]any{{1}, {2}}, captured.Calls())
}

// JS: types > assign actions should be inferred correctly
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L3928
func TestActions_Types_AssignActionsShouldBeInferredCorrectly(t *testing.T) {
	t.Skip("N/A: type-level only — checks assign() property/return types against context and event types via @ts-expect-error; no runtime expectations")
}

// JS: action meta > base action objects should have meta.action as the same base action object
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L3977
func TestActions_ActionMeta_BaseActionObjectsShouldHaveMetaActionAsSameBaseActionObject(t *testing.T) {
	t.Skip("skipped in JS")
}

// JS: action meta > should provide self
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L3981
func TestActions_ActionMeta_ShouldProvideSelf(t *testing.T) {
	selves := newSpy()

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Entry: xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[any]) { selves.Call(a.Self) })},
	})

	xs.CreateActor(machine).Start()

	// expect.assertions(1): the entry action (holding the only assertion) runs exactly once.
	calls := selves.Calls()
	require.Len(t, calls, 1)
	// expect(self.send).toBeDefined()
	assert.NotNil(t, calls[0][0])
}

// JS: actions > should call transition actions in document order for same-level parallel regions
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L3995
func TestActions_Actions_ShouldCallTransitionActionsInDocOrderForSameLevelParallelRegions(t *testing.T) {
	actual := newSpy()

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Type: xs.Parallel,
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{
				"FOO": {{Actions: xs.Actions{xs.ActionFunc(func(xs.ActionArgs[any]) { actual.Call("a") })}}},
			}},
			{Key: "b", On: map[string]xs.Transitions{
				"FOO": {{Actions: xs.Actions{xs.ActionFunc(func(xs.ActionArgs[any]) { actual.Call("b") })}}},
			}},
		},
	})
	service := xs.CreateActor(machine).Start()
	service.Send(xs.Ev("FOO"))

	assert.Equal(t, [][]any{{"a"}, {"b"}}, actual.Calls())
}

// JS: actions > should call transition actions in document order for states at different levels of parallel regions
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L4023
func TestActions_Actions_ShouldCallTransitionActionsInDocOrderForDifferentLevelParallelRegions(t *testing.T) {
	actual := newSpy()

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Type: xs.Parallel,
		States: xs.States{
			{
				Key:     "a",
				Initial: "a1",
				States: xs.States{
					{Key: "a1", On: map[string]xs.Transitions{
						"FOO": {{Actions: xs.Actions{xs.ActionFunc(func(xs.ActionArgs[any]) { actual.Call("a1") })}}},
					}},
				},
			},
			{Key: "b", On: map[string]xs.Transitions{
				"FOO": {{Actions: xs.Actions{xs.ActionFunc(func(xs.ActionArgs[any]) { actual.Call("b") })}}},
			}},
		},
	})
	service := xs.CreateActor(machine).Start()
	service.Send(xs.Ev("FOO"))

	assert.Equal(t, [][]any{{"a1"}, {"b"}}, actual.Calls())
}

// JS: actions > should call an inline action responding to an initial raise with the raised event
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L4056
func TestActions_Actions_ShouldCallInlineActionOnInitialRaiseWithRaisedEvent(t *testing.T) {
	s := newSpy()

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Entry: xs.Actions{xs.Raise(xs.Ev("HELLO"))},
		On: map[string]xs.Transitions{
			"HELLO": {{Actions: xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[any]) { s.Call(a.Event) })}}},
		},
	})

	xs.CreateActor(machine).Start()

	assert.Contains(t, s.Calls(), []any{xs.Ev("HELLO")})
}

// JS: actions > should call a referenced action responding to an initial raise with the raised event
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L4075
func TestActions_Actions_ShouldCallReferencedActionOnInitialRaiseWithRaisedEvent(t *testing.T) {
	s := newSpy()

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Entry: xs.Actions{xs.Raise(xs.Ev("HELLO"))},
		On: map[string]xs.Transitions{
			"HELLO": {{Actions: xs.Actions{xs.ActionRef{Type: "foo"}}}},
		},
	}, xs.Implementations{
		Actions: map[string]xs.Action{
			"foo": xs.ActionFunc(func(a xs.ActionArgs[any]) { s.Call(a.Event) }),
		},
	})

	xs.CreateActor(machine).Start()

	assert.Contains(t, s.Calls(), []any{xs.Ev("HELLO")})
}

// JS: actions > should call an inline action responding to an initial raise with updated (non-initial) context
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L4101
func TestActions_Actions_ShouldCallInlineActionOnInitialRaiseWithUpdatedContext(t *testing.T) {
	type ctx struct{ Count int }
	s := newSpy()

	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Context: ctx{Count: 0},
		Entry: xs.Actions{
			xs.Assign(func(xs.AssignArgs[ctx]) ctx { return ctx{Count: 42} }),
			xs.Raise(xs.Ev("HELLO")),
		},
		On: map[string]xs.Transitions{
			"HELLO": {{Actions: xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[ctx]) { s.Call(a.Context) })}}},
		},
	})

	xs.CreateActor(machine).Start()

	assert.Contains(t, s.Calls(), []any{ctx{Count: 42}})
}

// JS: actions > should call a referenced action responding to an initial raise with updated (non-initial) context
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L4121
func TestActions_Actions_ShouldCallReferencedActionOnInitialRaiseWithUpdatedContext(t *testing.T) {
	type ctx struct{ Count int }
	s := newSpy()

	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Context: ctx{Count: 0},
		Entry: xs.Actions{
			xs.Assign(func(xs.AssignArgs[ctx]) ctx { return ctx{Count: 42} }),
			xs.Raise(xs.Ev("HELLO")),
		},
		On: map[string]xs.Transitions{
			"HELLO": {{Actions: xs.Actions{xs.ActionRef{Type: "foo"}}}},
		},
	}, xs.Implementations{
		Actions: map[string]xs.Action{
			"foo": xs.ActionFunc(func(a xs.ActionArgs[ctx]) { s.Call(a.Context) }),
		},
	})

	xs.CreateActor(machine).Start()

	assert.Contains(t, s.Calls(), []any{ctx{Count: 42}})
}

// JS: actions > should call inline entry custom action with undefined parametrized action object
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L4148
func TestActions_Actions_ShouldCallInlineEntryCustomActionWithUndefinedParams(t *testing.T) {
	s := newSpy()
	xs.CreateActor(
		xs.CreateMachine(xs.MachineConfig[any]{
			Entry: xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[any]) { s.Call(a.Params) })},
		}),
	).Start()

	assert.Contains(t, s.Calls(), []any{nil})
}

// JS: actions > should call inline entry builtin action with undefined parametrized action object
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L4161
func TestActions_Actions_ShouldCallInlineEntryBuiltinActionWithUndefinedParams(t *testing.T) {
	s := newSpy()
	xs.CreateActor(
		xs.CreateMachine(xs.MachineConfig[any]{
			Entry: xs.Actions{xs.Assign(func(a xs.AssignArgs[any]) any {
				s.Call(a.Params)
				return a.Context // JS returns {} (no context change)
			})},
		}),
	).Start()

	assert.Contains(t, s.Calls(), []any{nil})
}

// JS: actions > should call inline transition custom action with undefined parametrized action object
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L4175
func TestActions_Actions_ShouldCallInlineTransitionCustomActionWithUndefinedParams(t *testing.T) {
	s := newSpy()

	actorRef := xs.CreateActor(
		xs.CreateMachine(xs.MachineConfig[any]{
			On: map[string]xs.Transitions{
				"FOO": {{Actions: xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[any]) { s.Call(a.Params) })}}},
			},
		}),
	).Start()
	actorRef.Send(xs.Ev("FOO"))

	assert.Contains(t, s.Calls(), []any{nil})
}

// JS: actions > should call inline transition builtin action with undefined parameters
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L4194
func TestActions_Actions_ShouldCallInlineTransitionBuiltinActionWithUndefinedParams(t *testing.T) {
	s := newSpy()

	actorRef := xs.CreateActor(
		xs.CreateMachine(xs.MachineConfig[any]{
			On: map[string]xs.Transitions{
				"FOO": {{Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[any]) any {
					s.Call(a.Params)
					return a.Context // JS returns {} (no context change)
				})}}},
			},
		}),
	).Start()
	actorRef.Send(xs.Ev("FOO"))

	assert.Contains(t, s.Calls(), []any{nil})
}

// JS: actions > should call a referenced custom action with undefined params when it has no params and it is referenced using a string
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L4214
func TestActions_Actions_ShouldCallReferencedCustomActionWithUndefinedParamsWhenStringRef(t *testing.T) {
	s := newSpy()

	xs.CreateActor(
		xs.CreateMachine(xs.MachineConfig[any]{
			Entry: xs.Actions{xs.ActionRef{Type: "myAction"}},
		}, xs.Implementations{
			Actions: map[string]xs.Action{
				"myAction": xs.ActionFunc(func(a xs.ActionArgs[any]) { s.Call(a.Params) }),
			},
		}),
	).Start()

	assert.Contains(t, s.Calls(), []any{nil})
}

// JS: actions > should call a referenced builtin action with undefined params when it has no params and it is referenced using a string
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L4235
func TestActions_Actions_ShouldCallReferencedBuiltinActionWithUndefinedParamsWhenStringRef(t *testing.T) {
	s := newSpy()

	xs.CreateActor(
		xs.CreateMachine(xs.MachineConfig[any]{
			Entry: xs.Actions{xs.ActionRef{Type: "myAction"}},
		}, xs.Implementations{
			Actions: map[string]xs.Action{
				"myAction": xs.Assign(func(a xs.AssignArgs[any]) any {
					s.Call(a.Params)
					return a.Context // JS returns {} (no context change)
				}),
			},
		}),
	).Start()

	assert.Contains(t, s.Calls(), []any{nil})
}

// JS: actions > should call a referenced custom action with the provided parametrized action object
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L4257
func TestActions_Actions_ShouldCallReferencedCustomActionWithProvidedParams(t *testing.T) {
	s := newSpy()

	xs.CreateActor(
		xs.CreateMachine(xs.MachineConfig[any]{
			Entry: xs.Actions{xs.ActionRef{Type: "myAction", Params: map[string]any{"foo": "bar"}}},
		}, xs.Implementations{
			Actions: map[string]xs.Action{
				"myAction": xs.ActionFunc(func(a xs.ActionArgs[any]) { s.Call(a.Params) }),
			},
		}),
	).Start()

	assert.Contains(t, s.Calls(), []any{map[string]any{"foo": "bar"}})
}

// JS: actions > should call a referenced builtin action with the provided parametrized action object
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L4285
func TestActions_Actions_ShouldCallReferencedBuiltinActionWithProvidedParams(t *testing.T) {
	s := newSpy()

	xs.CreateActor(
		xs.CreateMachine(xs.MachineConfig[any]{
			Entry: xs.Actions{xs.ActionRef{Type: "myAction", Params: map[string]any{"foo": "bar"}}},
		}, xs.Implementations{
			Actions: map[string]xs.Action{
				"myAction": xs.Assign(func(a xs.AssignArgs[any]) any {
					s.Call(a.Params)
					return a.Context // JS returns {} (no context change)
				}),
			},
		}),
	).Start()

	assert.Contains(t, s.Calls(), []any{map[string]any{"foo": "bar"}})
}

// JS: actions > should warn if called in custom action
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L4314
func TestActions_Actions_ShouldWarnIfCalledInCustomAction(t *testing.T) {
	warnSpy := newSpy()
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Entry: xs.Actions{xs.ActionFunc(func(xs.ActionArgs[any]) {
			xs.Assign(func(a xs.AssignArgs[any]) any { return a.Context })
			xs.Raise(xs.Ev(""))
			xs.SendTo("", xs.Ev(""))
			xs.Emit(xs.Ev(""))
		})},
	})

	xs.CreateActor(machine, xs.WithWarnHandler(func(args ...any) { warnSpy.Call(args...) })).Start()

	assert.Equal(t, [][]any{
		{"Custom actions should not call `assign()` directly, as it is not imperative. See https://stately.ai/docs/actions#built-in-actions for more details."},
		{"Custom actions should not call `raise()` directly, as it is not imperative. See https://stately.ai/docs/actions#built-in-actions for more details."},
		{"Custom actions should not call `sendTo()` directly, as it is not imperative. See https://stately.ai/docs/actions#built-in-actions for more details."},
		{"Custom actions should not call `emit()` directly, as it is not imperative. See https://stately.ai/docs/actions#built-in-actions for more details."},
	}, warnSpy.Calls())
}

// JS: actions > inline actions should not leak into provided actions object
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actions.test.ts#L4345
func TestActions_Actions_InlineActionsShouldNotLeakIntoProvidedActionsObject(t *testing.T) {
	actions := map[string]xs.Action{}

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Entry: xs.Actions{xs.ActionFunc(func(xs.ActionArgs[any]) {})},
	}, xs.Implementations{Actions: actions})

	xs.CreateActor(machine).Start()

	assert.Equal(t, map[string]xs.Action{}, actions)
}
