package xstate_test

import (
	"strings"
	"sync"
	"testing"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/assert"
)

// internalTransitions1TrackEntries mirrors trackEntries from test/utils.ts: it
// prepends entry/exit tracking actions to every state node of the machine and
// returns a flush function that yields (and clears) the recorded log.
func internalTransitions1TrackEntries[C any](machine *xs.StateMachine[C]) func() []string {
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

// JS: internal transitions > parent state should enter child state without re-entering self
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/internalTransitions.test.ts#L5
func TestInternalTransitions_ParentStateShouldEnterChildStateWithoutReenteringSelf(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "foo",
		States: xs.States{
			{
				Key:     "foo",
				Initial: "a",
				States:  xs.States{{Key: "a"}, {Key: "b"}},
				On: map[string]xs.Transitions{
					"CLICK": {{Target: ".b"}},
				},
			},
		},
	})

	flushTracked := internalTransitions1TrackEntries(machine)
	actor := xs.CreateActor(machine).Start()
	flushTracked()

	actor.Send(xs.Ev("CLICK"))

	assert.Equal(t, map[string]any{"foo": "b"}, actor.GetSnapshot().Value)
	assert.Equal(t, []string{"exit: foo.a", "enter: foo.b"}, flushTracked())
}

// JS: internal transitions > parent state should re-enter self upon transitioning to child state if transition is reentering
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/internalTransitions.test.ts#L34
func TestInternalTransitions_ParentStateShouldReenterSelfUponTransitioningToChildIfReentering(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "foo",
		States: xs.States{
			{
				Key:     "foo",
				Initial: "left",
				States:  xs.States{{Key: "left"}, {Key: "right"}},
				On: map[string]xs.Transitions{
					"NEXT": {{Target: ".right", Reenter: true}},
				},
			},
		},
	})

	flushTracked := internalTransitions1TrackEntries(machine)
	actor := xs.CreateActor(machine).Start()
	flushTracked()

	actor.Send(xs.Ev("NEXT"))

	assert.Equal(t, map[string]any{"foo": "right"}, actor.GetSnapshot().Value)
	assert.Equal(t, []string{
		"exit: foo.left",
		"exit: foo",
		"enter: foo",
		"enter: foo.right",
	}, flushTracked())
}

// JS: internal transitions > parent state should only exit/reenter if there is an explicit self-transition
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/internalTransitions.test.ts#L71
func TestInternalTransitions_ParentStateShouldOnlyExitReenterIfExplicitSelfTransition(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "foo",
		States: xs.States{
			{
				Key:     "foo",
				Initial: "a",
				States: xs.States{
					{Key: "a", On: map[string]xs.Transitions{
						"NEXT": {{Target: "b"}},
					}},
					{Key: "b"},
				},
				On: map[string]xs.Transitions{
					"RESET": {{Target: "foo", Reenter: true}},
				},
			},
		},
	})

	flushTracked := internalTransitions1TrackEntries(machine)
	actor := xs.CreateActor(machine).Start()
	actor.Send(xs.Ev("NEXT"))
	flushTracked()

	actor.Send(xs.Ev("RESET"))

	assert.Equal(t, map[string]any{"foo": "a"}, actor.GetSnapshot().Value)
	assert.Equal(t, []string{
		"exit: foo.b",
		"exit: foo",
		"enter: foo",
		"enter: foo.a",
	}, flushTracked())
}

// JS: internal transitions > parent state should only exit/reenter if there is an explicit self-transition (to child)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/internalTransitions.test.ts#L115
func TestInternalTransitions_ParentStateShouldOnlyExitReenterIfExplicitSelfTransitionToChild(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "foo",
		States: xs.States{
			{
				Key:     "foo",
				Initial: "a",
				States:  xs.States{{Key: "a"}, {Key: "b"}},
				On: map[string]xs.Transitions{
					"RESET_TO_B": {{Target: "foo.b", Reenter: true}},
				},
			},
		},
	})

	flushTracked := internalTransitions1TrackEntries(machine)
	actor := xs.CreateActor(machine).Start()
	flushTracked()

	actor.Send(xs.Ev("RESET_TO_B"))

	assert.Equal(t, map[string]any{"foo": "b"}, actor.GetSnapshot().Value)
	assert.Equal(t, []string{
		"exit: foo.a",
		"exit: foo",
		"enter: foo",
		"enter: foo.b",
	}, flushTracked())
}

// JS: internal transitions > should listen to events declared at top state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/internalTransitions.test.ts#L152
func TestInternalTransitions_ShouldListenToEventsDeclaredAtTopState(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "foo",
		On: map[string]xs.Transitions{
			"CLICKED": {{Target: ".bar"}},
		},
		States: xs.States{{Key: "foo"}, {Key: "bar"}},
	})
	actor := xs.CreateActor(machine).Start()
	actor.Send(xs.Ev("CLICKED"))

	assert.Equal(t, "bar", actor.GetSnapshot().Value)
}

// JS: internal transitions > should work with targetless transitions (in conditional array)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/internalTransitions.test.ts#L171
func TestInternalTransitions_ShouldWorkWithTargetlessTransitionsInConditionalArray(t *testing.T) {
	s := newSpy()
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "foo",
		States: xs.States{
			{Key: "foo", On: map[string]xs.Transitions{
				"TARGETLESS_ARRAY": {{Actions: xs.Actions{
					xs.ActionFunc(func(a xs.ActionArgs[any]) { s.Call(a) }),
				}}},
			}},
		},
	})
	actor := xs.CreateActor(machine).Start()
	actor.Send(xs.Ev("TARGETLESS_ARRAY"))
	assert.Greater(t, s.Count(), 0)
}

// JS: internal transitions > should work with targetless transitions (in object)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/internalTransitions.test.ts#L190
func TestInternalTransitions_ShouldWorkWithTargetlessTransitionsInObject(t *testing.T) {
	s := newSpy()
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "foo",
		States: xs.States{
			{Key: "foo", On: map[string]xs.Transitions{
				"TARGETLESS_OBJECT": {{Actions: xs.Actions{
					xs.ActionFunc(func(a xs.ActionArgs[any]) { s.Call(a) }),
				}}},
			}},
		},
	})
	actor := xs.CreateActor(machine).Start()
	actor.Send(xs.Ev("TARGETLESS_OBJECT"))
	assert.Greater(t, s.Count(), 0)
}

// JS: internal transitions > should work on parent with targetless transitions (in conditional array)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/internalTransitions.test.ts#L209
func TestInternalTransitions_ShouldWorkOnParentWithTargetlessTransitionsInConditionalArray(t *testing.T) {
	s := newSpy()
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		On: map[string]xs.Transitions{
			"TARGETLESS_ARRAY": {{Actions: xs.Actions{
				xs.ActionFunc(func(a xs.ActionArgs[any]) { s.Call(a) }),
			}}},
		},
		Initial: "foo",
		States:  xs.States{{Key: "foo"}},
	})
	actor := xs.CreateActor(machine).Start()
	actor.Send(xs.Ev("TARGETLESS_ARRAY"))
	assert.Greater(t, s.Count(), 0)
}

// JS: internal transitions > should work on parent with targetless transitions (in object)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/internalTransitions.test.ts#L225
func TestInternalTransitions_ShouldWorkOnParentWithTargetlessTransitionsInObject(t *testing.T) {
	s := newSpy()
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		On: map[string]xs.Transitions{
			"TARGETLESS_OBJECT": {{Actions: xs.Actions{
				xs.ActionFunc(func(a xs.ActionArgs[any]) { s.Call(a) }),
			}}},
		},
		Initial: "foo",
		States:  xs.States{{Key: "foo"}},
	})
	actor := xs.CreateActor(machine).Start()
	actor.Send(xs.Ev("TARGETLESS_OBJECT"))
	assert.Greater(t, s.Count(), 0)
}

// JS: internal transitions > should maintain the child state when targetless transition is handled by parent
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/internalTransitions.test.ts#L241
func TestInternalTransitions_ShouldMaintainChildStateWhenTargetlessTransitionHandledByParent(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "foo",
		On: map[string]xs.Transitions{
			"PARENT_EVENT": {{Actions: xs.Actions{xs.ActionFunc(func(xs.ActionArgs[any]) {})}}},
		},
		States: xs.States{{Key: "foo"}},
	})
	actor := xs.CreateActor(machine).Start()
	actor.Send(xs.Ev("PARENT_EVENT"))

	assert.Equal(t, "foo", actor.GetSnapshot().Value)
}

// JS: internal transitions > should reenter proper descendants of a source state of an internal transition
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/internalTransitions.test.ts#L259
func TestInternalTransitions_ShouldReenterProperDescendantsOfSourceStateOfInternalTransition(t *testing.T) {
	type ctx struct {
		SourceStateEntries      int
		DirectDescendantEntries int
		DeepDescendantEntries   int
	}

	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Context: ctx{
			SourceStateEntries:      0,
			DirectDescendantEntries: 0,
			DeepDescendantEntries:   0,
		},
		Initial: "a1",
		States: xs.States{
			{
				Key:     "a1",
				Initial: "a11",
				Entry: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
					c := a.Context
					c.SourceStateEntries = a.Context.SourceStateEntries + 1
					return c
				})},
				States: xs.States{
					{
						Key:     "a11",
						Initial: "a111",
						Entry: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
							c := a.Context
							c.DirectDescendantEntries = a.Context.DirectDescendantEntries + 1
							return c
						})},
						States: xs.States{
							{
								Key: "a111",
								Entry: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
									c := a.Context
									c.DeepDescendantEntries = a.Context.DeepDescendantEntries + 1
									return c
								})},
							},
						},
					},
				},
				On: map[string]xs.Transitions{
					"REENTER": {{Target: ".a11.a111"}},
				},
			},
		},
	})

	service := xs.CreateActor(machine).Start()

	service.Send(xs.Ev("REENTER"))

	assert.Equal(t, ctx{
		SourceStateEntries:      1,
		DirectDescendantEntries: 2,
		DeepDescendantEntries:   2,
	}, service.GetSnapshot().Context)
}

// JS: internal transitions > should exit proper descendants of a source state of an internal transition
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/internalTransitions.test.ts#L315
func TestInternalTransitions_ShouldExitProperDescendantsOfSourceStateOfInternalTransition(t *testing.T) {
	type ctx struct {
		SourceStateExits      int
		DirectDescendantExits int
		DeepDescendantExits   int
	}

	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Context: ctx{
			SourceStateExits:      0,
			DirectDescendantExits: 0,
			DeepDescendantExits:   0,
		},
		Initial: "a1",
		States: xs.States{
			{
				Key:     "a1",
				Initial: "a11",
				Exit: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
					c := a.Context
					c.SourceStateExits = a.Context.SourceStateExits + 1
					return c
				})},
				States: xs.States{
					{
						Key:     "a11",
						Initial: "a111",
						Exit: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
							c := a.Context
							c.DirectDescendantExits = a.Context.DirectDescendantExits + 1
							return c
						})},
						States: xs.States{
							{
								Key: "a111",
								Exit: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
									c := a.Context
									c.DeepDescendantExits = a.Context.DeepDescendantExits + 1
									return c
								})},
							},
						},
					},
				},
				On: map[string]xs.Transitions{
					"REENTER": {{Target: ".a11.a111"}},
				},
			},
		},
	})

	service := xs.CreateActor(machine).Start()

	service.Send(xs.Ev("REENTER"))

	assert.Equal(t, ctx{
		SourceStateExits:      0,
		DirectDescendantExits: 1,
		DeepDescendantExits:   1,
	}, service.GetSnapshot().Context)
}
