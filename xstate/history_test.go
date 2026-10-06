package xstate_test

import (
	"encoding/json"
	"strings"
	"sync"
	"testing"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// history1TrackEntries mirrors trackEntries from test/utils.ts: it prepends
// entry/exit tracking actions to every state node of the machine and returns a
// flush function that yields (and clears) the recorded log.
func history1TrackEntries[C any](machine *xs.StateMachine[C]) func() []string {
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

// history1SpyAction wraps a spy as an action (JS passes `vi.fn()` directly as
// an action).
func history1SpyAction(s *spy) xs.Action {
	return xs.ActionFunc(func(xs.ActionArgs[any]) { s.Call() })
}

// JS: history states > should go to the most recently visited state (explicit shallow history type)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/history.test.ts#L6
func TestHistory_ShouldGoToMostRecentlyVisitedStateExplicitShallowHistoryType(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "on",
		States: xs.States{
			{
				Key:     "on",
				Initial: "first",
				States: xs.States{
					{Key: "first", On: map[string]xs.Transitions{"SWITCH": {{Target: "second"}}}},
					{Key: "second"},
					{Key: "hist", Type: xs.History, History: xs.Shallow},
				},
				On: map[string]xs.Transitions{"POWER": {{Target: "off"}}},
			},
			{Key: "off", On: map[string]xs.Transitions{"POWER": {{Target: "on.hist"}}}},
		},
	})

	actorRef := xs.CreateActor(machine).Start()

	actorRef.Send(xs.Ev("SWITCH"))
	actorRef.Send(xs.Ev("POWER"))
	actorRef.Send(xs.Ev("POWER"))

	assert.Equal(t, map[string]any{"on": "second"}, actorRef.GetSnapshot().Value)
}

// JS: history states > should go to the most recently visited state (no explicit history type)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/history.test.ts#L41
func TestHistory_ShouldGoToMostRecentlyVisitedStateNoExplicitHistoryType(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "on",
		States: xs.States{
			{
				Key:     "on",
				Initial: "first",
				States: xs.States{
					{Key: "first", On: map[string]xs.Transitions{"SWITCH": {{Target: "second"}}}},
					{Key: "second"},
					{Key: "hist", Type: xs.History},
				},
				On: map[string]xs.Transitions{"POWER": {{Target: "off"}}},
			},
			{Key: "off", On: map[string]xs.Transitions{"POWER": {{Target: "on.hist"}}}},
		},
	})
	actorRef := xs.CreateActor(machine).Start()

	actorRef.Send(xs.Ev("SWITCH"))
	actorRef.Send(xs.Ev("POWER"))
	actorRef.Send(xs.Ev("POWER"))

	assert.Equal(t, map[string]any{"on": "second"}, actorRef.GetSnapshot().Value)
}

// JS: history states > should go to the initial state when no history present (explicit shallow history type)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/history.test.ts#L74
func TestHistory_ShouldGoToInitialStateWhenNoHistoryPresentExplicitShallowHistoryType(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "off",
		States: xs.States{
			{Key: "off", On: map[string]xs.Transitions{"POWER": {{Target: "on.hist"}}}},
			{
				Key:     "on",
				Initial: "first",
				States: xs.States{
					{Key: "first"},
					{Key: "second"},
					{Key: "hist", Type: xs.History, History: xs.Shallow},
				},
			},
		},
	})
	actorRef := xs.CreateActor(machine).Start()

	actorRef.Send(xs.Ev("POWER"))

	assert.Equal(t, map[string]any{"on": "first"}, actorRef.GetSnapshot().Value)
}

// JS: history states > should go to the initial state when no history present (no explicit history type)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/history.test.ts#L101
func TestHistory_ShouldGoToInitialStateWhenNoHistoryPresentNoExplicitHistoryType(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "off",
		States: xs.States{
			{Key: "off", On: map[string]xs.Transitions{"POWER": {{Target: "on.hist"}}}},
			{
				Key:     "on",
				Initial: "first",
				States: xs.States{
					{Key: "first"},
					{Key: "second"},
					{Key: "hist", Type: xs.History},
				},
			},
		},
	})
	actorRef := xs.CreateActor(machine).Start()

	actorRef.Send(xs.Ev("POWER"))

	assert.Equal(t, map[string]any{"on": "first"}, actorRef.GetSnapshot().Value)
}

// JS: history states > should go to the most recently visited state by a transient transition
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/history.test.ts#L127
func TestHistory_ShouldGoToMostRecentlyVisitedStateByTransientTransition(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "idle",
		States: xs.States{
			{
				Key:     "idle",
				ID:      "idle",
				Initial: "absent",
				States: xs.States{
					{Key: "absent", On: map[string]xs.Transitions{
						"DEPLOY": {{Target: "#deploy"}},
					}},
					{Key: "present", On: map[string]xs.Transitions{
						"DEPLOY":  {{Target: "#deploy"}},
						"DESTROY": {{Target: "#destroy"}},
					}},
					{Key: "hist", Type: xs.History},
				},
			},
			{
				Key: "deploy",
				ID:  "deploy",
				On: map[string]xs.Transitions{
					"SUCCESS": {{Target: "idle.present"}},
					"FAILURE": {{Target: "idle.hist"}},
				},
			},
			{
				Key:    "destroy",
				ID:     "destroy",
				Always: xs.Transitions{{Target: "idle.absent"}},
			},
		},
	})

	actorRef := xs.CreateActor(machine).Start()

	actorRef.Send(xs.Ev("DEPLOY"))
	actorRef.Send(xs.Ev("SUCCESS"))
	actorRef.Send(xs.Ev("DESTROY"))
	actorRef.Send(xs.Ev("DEPLOY"))
	actorRef.Send(xs.Ev("FAILURE"))

	assert.Equal(t, map[string]any{"idle": "absent"}, actorRef.GetSnapshot().Value)
}

// JS: history states > should reenter persisted state during reentering transition targeting a history state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/history.test.ts#L176
func TestHistory_ShouldReenterPersistedStateDuringReenteringTransitionTargetingHistoryState(t *testing.T) {
	var mu sync.Mutex
	actual := []string{}
	push := func(s string) xs.Action {
		return xs.ActionFunc(func(xs.ActionArgs[any]) {
			mu.Lock()
			defer mu.Unlock()
			actual = append(actual, s)
		})
	}

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{
				Key: "a",
				On: map[string]xs.Transitions{
					"REENTER": {{Target: "#b_hist", Reenter: true}},
				},
				Initial: "a1",
				States: xs.States{
					{Key: "a1", On: map[string]xs.Transitions{"NEXT": {{Target: "a2"}}}},
					{
						Key:   "a2",
						Entry: xs.Actions{push("a2 entered")},
						Exit:  xs.Actions{push("a2 exited")},
					},
					{Key: "a3", Type: xs.History, ID: "b_hist"},
				},
			},
		},
	})

	actorRef := xs.CreateActor(machine).Start()

	actorRef.Send(xs.Ev("NEXT"))

	mu.Lock()
	actual = actual[:0]
	mu.Unlock()
	actorRef.Send(xs.Ev("REENTER"))

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, []string{"a2 exited", "a2 entered"}, actual)
}

// JS: history states > should go to the configured default target when a history state is the initial state of the machine
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/history.test.ts#L219
func TestHistory_ShouldGoToDefaultTargetWhenHistoryStateIsInitialStateOfMachine(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "foo",
		States: xs.States{
			{Key: "foo", Type: xs.History, Target: "bar"},
			{Key: "bar"},
		},
	})

	actorRef := xs.CreateActor(machine).Start()

	assert.Equal(t, "bar", actorRef.GetSnapshot().Value)
}

// JS: history states > should go to the configured default target when a history state is the initial state of the transition's target
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/history.test.ts#L236
func TestHistory_ShouldGoToDefaultTargetWhenHistoryStateIsInitialStateOfTransitionTarget(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "foo",
		States: xs.States{
			{Key: "foo", On: map[string]xs.Transitions{"NEXT": {{Target: "bar"}}}},
			{
				Key:     "bar",
				Initial: "baz",
				States: xs.States{
					{Key: "baz", Type: xs.History, Target: "qwe"},
					{Key: "qwe"},
				},
			},
		},
	})

	actorRef := xs.CreateActor(machine).Start()

	actorRef.Send(xs.Ev("NEXT"))

	assert.Equal(t, map[string]any{"bar": "qwe"}, actorRef.GetSnapshot().Value)
}

// JS: history states > should execute actions of the initial transition when a history state without a default target is targeted and its parent state was never visited yet
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/history.test.ts#L267
func TestHistory_ShouldExecuteInitialActionsWhenHistoryWithoutDefaultTargetTargetedAndParentNeverVisited(t *testing.T) {
	s := newSpy()

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{"NEXT": {{Target: "#hist"}}}},
			xs.WithStateInitialActions(xs.StateConfig{
				Key:     "b",
				Initial: "b1",
				States: xs.States{
					{Key: "b1"},
					{Key: "b2", ID: "hist", Type: xs.History},
				},
			}, history1SpyAction(s)),
		},
	})

	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.Ev("NEXT"))

	assert.Equal(t, 1, s.Count())
}

// JS: history states > should enter the parallel default configuration when a deep history state without a default target is targeted and its parent parallel state was never visited yet
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/history.test.ts#L298
func TestHistory_ShouldEnterParallelDefaultConfigWhenDeepHistoryWithoutDefaultTargetTargetedAndParentNeverVisited(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "off",
		States: xs.States{
			{Key: "off", On: map[string]xs.Transitions{"GO": {{Target: "on.hist"}}}},
			{
				Key:  "on",
				Type: xs.Parallel,
				States: xs.States{
					{Key: "regA", Initial: "a1", States: xs.States{{Key: "a1"}, {Key: "a2"}}},
					{Key: "regB", Initial: "b1", States: xs.States{{Key: "b1"}, {Key: "b2"}}},
					{Key: "hist", Type: xs.History, History: xs.Deep},
				},
			},
		},
	})

	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.Ev("GO"))

	assert.Equal(t, map[string]any{
		"on": map[string]any{"regA": "a1", "regB": "b1"},
	}, actorRef.GetSnapshot().Value)
}

// JS: history states > should enter the parallel default configuration when a shallow history state without a default target is targeted and its parent parallel state was never visited yet
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/history.test.ts#L324
func TestHistory_ShouldEnterParallelDefaultConfigWhenShallowHistoryWithoutDefaultTargetTargetedAndParentNeverVisited(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "off",
		States: xs.States{
			{Key: "off", On: map[string]xs.Transitions{"GO": {{Target: "on.hist"}}}},
			{
				Key:  "on",
				Type: xs.Parallel,
				States: xs.States{
					{Key: "regA", Initial: "a1", States: xs.States{{Key: "a1"}, {Key: "a2"}}},
					{Key: "regB", Initial: "b1", States: xs.States{{Key: "b1"}, {Key: "b2"}}},
					{Key: "hist", Type: xs.History, History: xs.Shallow},
				},
			},
		},
	})

	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.Ev("GO"))

	assert.Equal(t, map[string]any{
		"on": map[string]any{"regA": "a1", "regB": "b1"},
	}, actorRef.GetSnapshot().Value)
}

// JS: history states > should not execute actions of the initial transition when a history state with a default target is targeted and its parent state was never visited yet
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/history.test.ts#L350
func TestHistory_ShouldNotExecuteInitialActionsWhenHistoryWithDefaultTargetTargetedAndParentNeverVisited(t *testing.T) {
	s := newSpy()
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{"NEXT": {{Target: "#hist"}}}},
			xs.WithStateInitialActions(xs.StateConfig{
				Key:     "b",
				Initial: "b1",
				States: xs.States{
					{Key: "b1"},
					{Key: "b2", ID: "hist", Type: xs.History, Target: "b3"},
					{Key: "b3"},
				},
			}, history1SpyAction(s)),
		},
	})

	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.Ev("NEXT"))

	assert.Equal(t, 0, s.Count())
}

// JS: history states > should execute entry actions of a parent of the targeted history state when its parent state was never visited yet
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/history.test.ts#L382
func TestHistory_ShouldExecuteParentEntryActionsOfTargetedHistoryStateWhenParentNeverVisited(t *testing.T) {
	s := newSpy()
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{"NEXT": {{Target: "#hist"}}}},
			{
				Key:     "b",
				Entry:   xs.Actions{history1SpyAction(s)},
				Initial: "b1",
				States: xs.States{
					{Key: "b1"},
					{Key: "b2", ID: "hist", Type: xs.History, Target: "b3"},
					{Key: "b3"},
				},
			},
		},
	})

	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.Ev("NEXT"))

	assert.Equal(t, 1, s.Count())
}

// JS: history states > should execute actions of the initial transition when it select a history state as the initial state of its parent
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/history.test.ts#L412
func TestHistory_ShouldExecuteInitialActionsWhenItSelectsHistoryStateAsInitialStateOfParent(t *testing.T) {
	s := newSpy()
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{"NEXT": {{Target: "b"}}}},
			xs.WithStateInitialActions(xs.StateConfig{
				Key:     "b",
				Initial: "b1",
				States: xs.States{
					{Key: "b1", ID: "hist", Type: xs.History, Target: "b2"},
					{Key: "b2"},
				},
			}, history1SpyAction(s)),
		},
	})

	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.Ev("NEXT"))

	assert.Equal(t, 1, s.Count())
}

// JS: history states > should execute actions of the initial transition when a history state without a default target is targeted and its parent state was already visited
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/history.test.ts#L443
func TestHistory_ShouldExecuteInitialActionsWhenHistoryWithoutDefaultTargetTargetedAndParentAlreadyVisited(t *testing.T) {
	s := newSpy()

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{"NEXT": {{Target: "#hist"}}}},
			xs.WithStateInitialActions(xs.StateConfig{
				Key:     "b",
				Initial: "b1",
				States: xs.States{
					{Key: "b1"},
					{Key: "b2", ID: "hist", Type: xs.History},
				},
				On: map[string]xs.Transitions{"NEXT": {{Target: "a"}}},
			}, history1SpyAction(s)),
		},
	})

	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.Ev("NEXT"))
	// spy.mockClear(): count calls from this point on.
	before := s.Count()

	actorRef.Send(xs.Ev("NEXT"))
	actorRef.Send(xs.Ev("NEXT"))

	assert.Equal(t, 0, s.Count()-before)
}

// JS: history states > should not execute actions of the initial transition when a history state with a default target is targeted and its parent state was already visited
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/history.test.ts#L481
func TestHistory_ShouldNotExecuteInitialActionsWhenHistoryWithDefaultTargetTargetedAndParentAlreadyVisited(t *testing.T) {
	s := newSpy()
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{"NEXT": {{Target: "#hist"}}}},
			xs.WithStateInitialActions(xs.StateConfig{
				Key:     "b",
				Initial: "b1",
				States: xs.States{
					{Key: "b1"},
					{Key: "b2", ID: "hist", Type: xs.History, Target: "b3"},
					{Key: "b3"},
				},
				On: map[string]xs.Transitions{"NEXT": {{Target: "a"}}},
			}, history1SpyAction(s)),
		},
	})

	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.Ev("NEXT"))
	// spy.mockClear(): count calls from this point on.
	before := s.Count()

	actorRef.Send(xs.Ev("NEXT"))
	actorRef.Send(xs.Ev("NEXT"))

	assert.Equal(t, 0, s.Count()-before)
}

// JS: history states > should execute entry actions of a parent of the targeted history state when its parent state was already visited
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/history.test.ts#L520
func TestHistory_ShouldExecuteParentEntryActionsOfTargetedHistoryStateWhenParentAlreadyVisited(t *testing.T) {
	s := newSpy()
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{"NEXT": {{Target: "#hist"}}}},
			{
				Key:     "b",
				Entry:   xs.Actions{history1SpyAction(s)},
				Initial: "b1",
				States: xs.States{
					{Key: "b1"},
					{Key: "b2", ID: "hist", Type: xs.History, Target: "b3"},
					{Key: "b3"},
				},
				On: map[string]xs.Transitions{"NEXT": {{Target: "a"}}},
			},
		},
	})

	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.Ev("NEXT"))
	// spy.mockClear(): count calls from this point on.
	before := s.Count()

	actorRef.Send(xs.Ev("NEXT"))
	actorRef.Send(xs.Ev("NEXT"))

	assert.Equal(t, 1, s.Count()-before)
}

// JS: history states > should invoke an actor when reentering the stored configuration through the history state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/history.test.ts#L557
func TestHistory_ShouldInvokeActorWhenReenteringStoredConfigurationThroughHistoryState(t *testing.T) {
	s := newSpy()

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "running",
		States: xs.States{
			{
				Key: "running",
				On: map[string]xs.Transitions{
					"PING": {{Target: "refresh"}},
				},
				Invoke: []xs.InvokeConfig{{
					// fromCallback(spy): the spy is the callback body.
					Logic: xs.FromCallback(func(a xs.CallbackArgs) func() {
						s.Call(a)
						return nil
					}),
				}},
			},
			{Key: "refresh", Type: xs.History},
		},
	})
	actorRef := xs.CreateActor(machine).Start()
	// spy.mockClear(): count calls from this point on.
	before := s.Count()

	actorRef.Send(xs.Ev("PING"))

	assert.Equal(t, 1, s.Count()-before)
}

// JS: history states > should not enter ancestors of the entered history state that lie outside of the transition domain when entering the default history configuration
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/history.test.ts#L586
func TestHistory_ShouldNotEnterAncestorsOutsideTransitionDomainWhenEnteringDefaultHistoryConfiguration(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "closed",
		States: xs.States{
			{
				Key: "closed",
				On: map[string]xs.Transitions{
					"BUTTON.CLICK": {{Target: "open.hist"}},
				},
			},
			{
				Key: "open",
				On: map[string]xs.Transitions{
					"BUTTON.CLICK": {{Target: "closed"}},
				},
				Initial: "first",
				States: xs.States{
					{Key: "hist", Type: xs.History},
					{Key: "first"},
					{Key: "second"},
				},
			},
		},
	})

	flushTracked := history1TrackEntries(machine)

	actorRef := xs.CreateActor(machine).Start()
	flushTracked()

	actorRef.Send(xs.Ev("BUTTON.CLICK"))
	assert.Equal(t, []string{
		"exit: closed",
		"enter: open",
		"enter: open.first",
	}, flushTracked())
}

// JS: history states > should not enter ancestors of the entered history state that lie outside of the transition domain when restoring the stored history configuration
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/history.test.ts#L622
func TestHistory_ShouldNotEnterAncestorsOutsideTransitionDomainWhenRestoringStoredHistoryConfiguration(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "closed",
		States: xs.States{
			{
				Key: "closed",
				ID:  "closed",
				On: map[string]xs.Transitions{
					"BUTTON.CLICK": {{Target: "open.hist"}},
				},
			},
			{
				Key: "open",
				On: map[string]xs.Transitions{
					"BUTTON.CLICK": {{Target: "closed"}},
				},
				Initial: "first",
				States: xs.States{
					{Key: "hist", Type: xs.History},
					{Key: "first", On: map[string]xs.Transitions{"NEXT": {{Target: "second"}}}},
					{Key: "second", On: map[string]xs.Transitions{"CLOSE": {{Target: "#closed"}}}},
				},
			},
		},
	})

	flushTracked := history1TrackEntries(machine)

	actorRef := xs.CreateActor(machine).Start()

	actorRef.Send(xs.Ev("BUTTON.CLICK"))
	actorRef.Send(xs.Ev("NEXT"))
	actorRef.Send(xs.Ev("CLOSE"))

	flushTracked()

	actorRef.Send(xs.Ev("BUTTON.CLICK"))
	assert.Equal(t, []string{
		"exit: closed",
		"enter: open",
		"enter: open.second",
	}, flushTracked())
}

// history1DeepMachine builds the machine shared (verbatim) by the
// "deep history states" tests; pInner adds `on: { INNER: 'Q' }` to state P
// (only "should go to the deepest history" has it) and historyType is the
// `history` value of the `history` state.
//
// NOTE: JS `history: { history: 'shallow' }` has no `type`; StateNode.ts:188-194
// infers type 'history' from the `history` property, so Type is left empty.
func history1DeepMachine(historyType xs.HistoryType, pInner bool) *xs.StateMachine[any] {
	p := xs.StateConfig{Key: "P"}
	if pInner {
		p.On = map[string]xs.Transitions{"INNER": {{Target: "Q"}}}
	}
	return xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "on",
		States: xs.States{
			{
				Key: "off",
				On: map[string]xs.Transitions{
					"POWER": {{Target: "on.history"}},
				},
			},
			{
				Key:     "on",
				Initial: "first",
				States: xs.States{
					{Key: "first", On: map[string]xs.Transitions{"SWITCH": {{Target: "second"}}}},
					{
						Key:     "second",
						Initial: "A",
						States: xs.States{
							{Key: "A", On: map[string]xs.Transitions{"INNER": {{Target: "B"}}}},
							{
								Key:     "B",
								Initial: "P",
								States:  xs.States{p, {Key: "Q"}},
							},
						},
					},
					{Key: "history", History: historyType},
				},
				On: map[string]xs.Transitions{"POWER": {{Target: "off"}}},
			},
		},
	})
}

// JS: deep history states > should go to the shallow history
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/history.test.ts#L674
func TestHistory_Deep_ShouldGoToShallowHistory(t *testing.T) {
	machine := history1DeepMachine(xs.Shallow, false)
	actorRef := xs.CreateActor(machine).Start()

	actorRef.Send(xs.Ev("SWITCH"))
	actorRef.Send(xs.Ev("INNER"))
	actorRef.Send(xs.Ev("POWER"))
	actorRef.Send(xs.Ev("POWER"))

	assert.Equal(t, map[string]any{
		"on": map[string]any{
			"second": "A",
		},
	}, actorRef.GetSnapshot().Value)
}

// JS: deep history states > should go to the deep history (explicit)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/history.test.ts#L726
func TestHistory_Deep_ShouldGoToDeepHistoryExplicit(t *testing.T) {
	machine := history1DeepMachine(xs.Deep, false)
	actorRef := xs.CreateActor(machine).Start()

	actorRef.Send(xs.Ev("SWITCH"))
	actorRef.Send(xs.Ev("INNER"))
	actorRef.Send(xs.Ev("POWER"))
	actorRef.Send(xs.Ev("POWER"))

	assert.Equal(t, map[string]any{
		"on": map[string]any{
			"second": map[string]any{"B": "P"},
		},
	}, actorRef.GetSnapshot().Value)
}

// JS: deep history states > should go to the deepest history
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/history.test.ts#L780
func TestHistory_Deep_ShouldGoToDeepestHistory(t *testing.T) {
	machine := history1DeepMachine(xs.Deep, true)
	actorRef := xs.CreateActor(machine).Start()

	actorRef.Send(xs.Ev("SWITCH"))
	actorRef.Send(xs.Ev("INNER"))
	actorRef.Send(xs.Ev("INNER"))
	actorRef.Send(xs.Ev("POWER"))
	actorRef.Send(xs.Ev("POWER"))

	assert.Equal(t, map[string]any{
		"on": map[string]any{
			"second": map[string]any{"B": "Q"},
		},
	}, actorRef.GetSnapshot().Value)
}

// history1ParallelOn builds the `on` parallel state used by the later
// "parallel history states" tests (JS L976, L1065, L1155, L1245: identical config in each):
// regions A and K each with a nested compound state, a shallow `hist` and a
// deep `deepHistory`, plus `hist`, `shallowHistory` and `deepHistory` history
// children of the parallel state itself.
//
// NOTE: JS `history: true` (no `type`) maps to History: xs.Shallow
// (StateNode.ts:227-228) with type inferred as 'history' (StateNode.ts:188-194).
func history1ParallelOn() xs.StateConfig {
	return xs.StateConfig{
		Key:  "on",
		Type: xs.Parallel,
		States: xs.States{
			{
				Key:     "A",
				Initial: "B",
				States: xs.States{
					{Key: "B", On: map[string]xs.Transitions{"INNER_A": {{Target: "C"}}}},
					{
						Key:     "C",
						Initial: "D",
						States: xs.States{
							{Key: "D", On: map[string]xs.Transitions{"INNER_A": {{Target: "E"}}}},
							{Key: "E"},
						},
					},
					{Key: "hist", History: xs.Shallow},
					{Key: "deepHistory", History: xs.Deep},
				},
			},
			{
				Key:     "K",
				Initial: "L",
				States: xs.States{
					{Key: "L", On: map[string]xs.Transitions{"INNER_K": {{Target: "M"}}}},
					{
						Key:     "M",
						Initial: "N",
						States: xs.States{
							{Key: "N", On: map[string]xs.Transitions{"INNER_K": {{Target: "O"}}}},
							{Key: "O"},
						},
					},
					{Key: "hist", History: xs.Shallow},
					{Key: "deepHistory", History: xs.Deep},
				},
			},
			{Key: "hist", History: xs.Shallow},
			{Key: "shallowHistory", History: xs.Shallow},
			{Key: "deepHistory", History: xs.Deep},
		},
		On: map[string]xs.Transitions{"POWER": {{Target: "off"}}},
	}
}

// JS: parallel history states > should ignore parallel state history
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/history.test.ts#L839
func TestHistory_Parallel_ShouldIgnoreParallelStateHistory(t *testing.T) {
	// `history: true` (no `type`) → History: xs.Shallow, type inferred.
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "off",
		States: xs.States{
			{
				Key: "off",
				On: map[string]xs.Transitions{
					"SWITCH": {{Target: "on"}},
					"POWER":  {{Target: "on.hist"}},
				},
			},
			{
				Key:  "on",
				Type: xs.Parallel,
				States: xs.States{
					{
						Key:     "A",
						Initial: "B",
						States: xs.States{
							{Key: "B", On: map[string]xs.Transitions{"INNER_A": {{Target: "C"}}}},
							{
								Key:     "C",
								Initial: "D",
								States:  xs.States{{Key: "D"}, {Key: "E"}},
							},
							{Key: "hist", History: xs.Shallow},
						},
					},
					{
						Key:     "K",
						Initial: "L",
						States: xs.States{
							{Key: "L"},
							{Key: "M"},
							{Key: "hist", History: xs.Shallow},
							{Key: "deepHistory", History: xs.Deep},
						},
					},
					{Key: "hist", History: xs.Shallow},
				},
				On: map[string]xs.Transitions{"POWER": {{Target: "off"}}},
			},
		},
	})

	actorRef := xs.CreateActor(machine).Start()

	actorRef.Send(xs.Ev("SWITCH"))
	actorRef.Send(xs.Ev("INNER_A"))
	actorRef.Send(xs.Ev("POWER"))
	actorRef.Send(xs.Ev("POWER"))

	assert.Equal(t, map[string]any{
		"on": map[string]any{
			"A": "B",
			"K": "L",
		},
	}, actorRef.GetSnapshot().Value)
}

// JS: parallel history states > should remember first level state history
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/history.test.ts#L905
func TestHistory_Parallel_ShouldRememberFirstLevelStateHistory(t *testing.T) {
	// `history: true` (no `type`) → History: xs.Shallow, type inferred.
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "off",
		States: xs.States{
			{
				Key: "off",
				On: map[string]xs.Transitions{
					"SWITCH":     {{Target: "on"}},
					"DEEP_POWER": {{Target: "on.deepHistory"}},
				},
			},
			{
				Key:  "on",
				Type: xs.Parallel,
				States: xs.States{
					{
						Key:     "A",
						Initial: "B",
						States: xs.States{
							{Key: "B", On: map[string]xs.Transitions{"INNER_A": {{Target: "C"}}}},
							{
								Key:     "C",
								Initial: "D",
								States:  xs.States{{Key: "D"}, {Key: "E"}},
							},
							{Key: "hist", History: xs.Shallow},
							{Key: "deepHistory", History: xs.Deep},
						},
					},
					{
						Key:     "K",
						Initial: "L",
						States: xs.States{
							{Key: "L"},
							{Key: "M"},
							{Key: "hist", History: xs.Shallow},
							{Key: "deepHistory", History: xs.Deep},
						},
					},
					{Key: "deepHistory", History: xs.Deep},
				},
				On: map[string]xs.Transitions{"POWER": {{Target: "off"}}},
			},
		},
	})

	actorRef := xs.CreateActor(machine).Start()

	actorRef.Send(xs.Ev("SWITCH"))
	actorRef.Send(xs.Ev("INNER_A"))
	actorRef.Send(xs.Ev("POWER"))
	actorRef.Send(xs.Ev("DEEP_POWER"))

	assert.Equal(t, map[string]any{
		"on": map[string]any{
			"A": map[string]any{"C": "D"},
			"K": "L",
		},
	}, actorRef.GetSnapshot().Value)
}

// JS: parallel history states > should re-enter each regions of parallel state correctly
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/history.test.ts#L976
func TestHistory_Parallel_ShouldReenterEachRegionsOfParallelStateCorrectly(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "off",
		States: xs.States{
			{
				Key: "off",
				On: map[string]xs.Transitions{
					"SWITCH":     {{Target: "on"}},
					"DEEP_POWER": {{Target: "on.deepHistory"}},
				},
			},
			history1ParallelOn(),
		},
	})
	actorRef := xs.CreateActor(machine).Start()

	actorRef.Send(xs.Ev("SWITCH"))
	actorRef.Send(xs.Ev("INNER_A"))
	actorRef.Send(xs.Ev("INNER_A"))
	actorRef.Send(xs.Ev("INNER_K"))
	actorRef.Send(xs.Ev("INNER_K"))
	actorRef.Send(xs.Ev("POWER"))
	actorRef.Send(xs.Ev("DEEP_POWER"))

	assert.Equal(t, map[string]any{
		"on": map[string]any{
			"A": map[string]any{"C": "E"},
			"K": map[string]any{"M": "O"},
		},
	}, actorRef.GetSnapshot().Value)
}

// JS: parallel history states > should re-enter multiple history states
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/history.test.ts#L1065
func TestHistory_Parallel_ShouldReenterMultipleHistoryStates(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "off",
		States: xs.States{
			{
				Key: "off",
				On: map[string]xs.Transitions{
					"SWITCH":           {{Target: "on"}},
					"PARALLEL_HISTORY": {{Targets: []string{"on.A.hist", "on.K.hist"}}},
				},
			},
			history1ParallelOn(),
		},
	})

	actorRef := xs.CreateActor(machine).Start()

	actorRef.Send(xs.Ev("SWITCH"))
	actorRef.Send(xs.Ev("INNER_A"))
	actorRef.Send(xs.Ev("INNER_A"))
	actorRef.Send(xs.Ev("INNER_K"))
	actorRef.Send(xs.Ev("INNER_K"))
	actorRef.Send(xs.Ev("POWER"))
	actorRef.Send(xs.Ev("PARALLEL_HISTORY"))

	assert.Equal(t, map[string]any{
		"on": map[string]any{
			"A": map[string]any{"C": "D"},
			"K": map[string]any{"M": "N"},
		},
	}, actorRef.GetSnapshot().Value)
}

// JS: parallel history states > should re-enter a parallel with partial history
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/history.test.ts#L1155
func TestHistory_Parallel_ShouldReenterParallelWithPartialHistory(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "off",
		States: xs.States{
			{
				Key: "off",
				On: map[string]xs.Transitions{
					"SWITCH":                {{Target: "on"}},
					"PARALLEL_SOME_HISTORY": {{Targets: []string{"on.A.C", "on.K.hist"}}},
				},
			},
			history1ParallelOn(),
		},
	})

	actorRef := xs.CreateActor(machine).Start()

	actorRef.Send(xs.Ev("SWITCH"))
	actorRef.Send(xs.Ev("INNER_A"))
	actorRef.Send(xs.Ev("INNER_A"))
	actorRef.Send(xs.Ev("INNER_K"))
	actorRef.Send(xs.Ev("INNER_K"))
	actorRef.Send(xs.Ev("POWER"))
	actorRef.Send(xs.Ev("PARALLEL_SOME_HISTORY"))

	assert.Equal(t, map[string]any{
		"on": map[string]any{
			"A": map[string]any{"C": "D"},
			"K": map[string]any{"M": "N"},
		},
	}, actorRef.GetSnapshot().Value)
}

// JS: parallel history states > should re-enter a parallel with full history
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/history.test.ts#L1245
func TestHistory_Parallel_ShouldReenterParallelWithFullHistory(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "off",
		States: xs.States{
			{
				Key: "off",
				On: map[string]xs.Transitions{
					"SWITCH":                {{Target: "on"}},
					"PARALLEL_DEEP_HISTORY": {{Targets: []string{"on.A.deepHistory", "on.K.deepHistory"}}},
				},
			},
			history1ParallelOn(),
		},
	})

	actorRef := xs.CreateActor(machine).Start()

	actorRef.Send(xs.Ev("SWITCH"))
	actorRef.Send(xs.Ev("INNER_A"))
	actorRef.Send(xs.Ev("INNER_A"))
	actorRef.Send(xs.Ev("INNER_K"))
	actorRef.Send(xs.Ev("INNER_K"))
	actorRef.Send(xs.Ev("POWER"))
	actorRef.Send(xs.Ev("PARALLEL_DEEP_HISTORY"))

	assert.Equal(t, map[string]any{
		"on": map[string]any{
			"A": map[string]any{"C": "E"},
			"K": map[string]any{"M": "O"},
		},
	}, actorRef.GetSnapshot().Value)
}

// JS: internal transition to a history state should enter default history state configuration if the containing state has never been exited yet
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/history.test.ts#L1338
func TestHistory_InternalTransitionToHistoryStateShouldEnterDefaultConfigIfContainingStateNeverExited(t *testing.T) {
	service := xs.CreateActor(xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "first",
		States: xs.States{
			{Key: "first", On: map[string]xs.Transitions{"NEXT": {{Target: "second.other"}}}},
			{
				Key:     "second",
				Initial: "nested",
				States: xs.States{
					{Key: "nested"},
					{Key: "other"},
					// `history: true` (no `type`) → History: xs.Shallow, type inferred.
					{Key: "hist", History: xs.Shallow},
				},
				On: map[string]xs.Transitions{"NEXT": {{Target: ".hist"}}},
			},
		},
	})).Start()

	service.Send(xs.Ev("NEXT"))
	service.Send(xs.Ev("NEXT"))

	assert.Equal(t, map[string]any{"second": "nested"}, service.GetSnapshot().Value)
}

// JS: multistage history states > should go to the most recently visited state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/history.test.ts#L1376
func TestHistory_Multistage_ShouldGoToMostRecentlyVisitedState(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "running",
		States: xs.States{
			{
				Key:     "running",
				Initial: "normal",
				States: xs.States{
					{Key: "normal", On: map[string]xs.Transitions{"SWITCH_TURBO": {{Target: "turbo"}}}},
					{Key: "turbo", On: map[string]xs.Transitions{"SWITCH_TURBO": {{Target: "normal"}}}},
					// `history: true` (no `type`) → History: xs.Shallow, type inferred.
					{Key: "H", History: xs.Shallow},
				},
				On: map[string]xs.Transitions{"POWER": {{Target: "off"}}},
			},
			{Key: "starting", On: map[string]xs.Transitions{"STARTED": {{Target: "running.H"}}}},
			{Key: "off", On: map[string]xs.Transitions{"POWER": {{Target: "starting"}}}},
		},
	})

	actorRef := xs.CreateActor(machine).Start()

	actorRef.Send(xs.Ev("SWITCH_TURBO"))
	actorRef.Send(xs.Ev("POWER"))
	actorRef.Send(xs.Ev("POWER"))
	actorRef.Send(xs.Ev("STARTED"))

	assert.Equal(t, map[string]any{"running": "turbo"}, actorRef.GetSnapshot().Value)
}

// history1JSONRoundTrip mirrors JSON.parse(JSON.stringify(v)).
func history1JSONRoundTrip(t *testing.T, v any) map[string]any {
	t.Helper()
	raw, err := json.Marshal(v)
	require.NoError(t, err)
	var out map[string]any
	require.NoError(t, json.Unmarshal(raw, &out))
	return out
}

// history1With mirrors `{ ...base, [key]: val }`; omit=true mirrors
// `[key]: undefined` (the key is dropped by JSON-like consumers).
func history1With(base map[string]any, key string, val any, omit bool) map[string]any {
	out := make(map[string]any, len(base)+1)
	for k, v := range base {
		out[k] = v
	}
	if omit {
		delete(out, key)
	} else {
		out[key] = val
	}
	return out
}

// history1ReviveSetup mirrors the describe-level setup of
// "revive history states" (JS L1419-1454). JS runs it once for the describe
// block; it is deterministic, so each Go test runs it on its own.
func history1ReviveSetup(t *testing.T) (*xs.StateMachine[any], map[string]any, *xs.MachineSnapshot[any]) {
	t.Helper()
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "on",
		States: xs.States{
			{
				Key:     "on",
				Initial: "first",
				States: xs.States{
					{Key: "first", On: map[string]xs.Transitions{"SWITCH": {{Target: "second"}}}},
					{Key: "second"},
					{Key: "hist", Type: xs.History},
				},
				On: map[string]xs.Transitions{"POWER": {{Target: "off"}}},
			},
			{Key: "off", On: map[string]xs.Transitions{"POWER": {{Target: "on.hist"}}}},
		},
	})

	sourceRef := xs.CreateActor(machine).Start()

	sourceRef.Send(xs.Ev("SWITCH"))
	sourceRef.Send(xs.Ev("POWER"))

	persistedSnapshot := history1JSONRoundTrip(t, sourceRef.GetPersistedSnapshot())
	snapshot := sourceRef.GetSnapshot()

	sourceRef.Stop()

	return machine, persistedSnapshot, snapshot
}

// JS: revive history states > should restore from stringified snapshot
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/history.test.ts#L1456
func TestHistory_Revive_ShouldRestoreFromStringifiedSnapshot(t *testing.T) {
	machine, persistedSnapshot, _ := history1ReviveSetup(t)

	assert.Equal(t, "off", persistedSnapshot["value"])

	actorRef := xs.CreateActor(machine, xs.WithSnapshot(persistedSnapshot)).Start()
	actorRef.Send(xs.Ev("POWER"))

	assert.Equal(t, map[string]any{"on": "second"}, actorRef.GetSnapshot().Value)
}

// JS: revive history states > should ignore unresolved ids as-is and log a warning
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/history.test.ts#L1467
func TestHistory_Revive_ShouldIgnoreUnresolvedIdsAsIsAndLogWarning(t *testing.T) {
	machine, persistedSnapshot, _ := history1ReviveSetup(t)

	consoleSpy := newSpy()
	fakeSnapshot := history1With(persistedSnapshot, "historyValue", map[string]any{
		"(machine).on.hist": []any{map[string]any{"id": "nonexistent"}},
	}, false)
	assert.Equal(t, "off", fakeSnapshot["value"])

	actorRef := xs.CreateActor(machine,
		xs.WithSnapshot(fakeSnapshot),
		xs.WithWarnHandler(func(args ...any) { consoleSpy.Call(args...) }),
	).Start()
	actorRef.Send(xs.Ev("POWER"))

	assert.Contains(t, consoleSpy.Calls(), []any{"Could not resolve StateNode for id: nonexistent"})
	assert.Equal(t, map[string]any{"on": "first"}, actorRef.GetSnapshot().Value)
	// (actorRef.getPersistedSnapshot() as any).historyValue
	assert.Equal(t, map[string]any{}, history1JSONRoundTrip(t, actorRef.GetPersistedSnapshot())["historyValue"])
}

// JS: revive history states > should not re-resolve already-instantiated StateNode
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/history.test.ts#L1488
func TestHistory_Revive_ShouldNotReResolveAlreadyInstantiatedStateNode(t *testing.T) {
	machine, _, snapshot := history1ReviveSetup(t)

	assert.Equal(t, "off", snapshot.Value)
	// toBeInstanceOf(StateNode): HistoryValue entries are *xs.StateNode.
	require.NotEmpty(t, snapshot.HistoryValue["(machine).on.hist"])
	assert.IsType(t, &xs.StateNode{}, snapshot.HistoryValue["(machine).on.hist"][0])
	assert.NotNil(t, snapshot.HistoryValue["(machine).on.hist"][0])

	actorRef := xs.CreateActor(machine, xs.WithSnapshot(snapshot)).Start()
	actorRef.Send(xs.Ev("POWER"))

	assert.Equal(t, map[string]any{"on": "second"}, actorRef.GetSnapshot().Value)
}

// JS: revive history states > should handle null, undefined, and primitive values
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/history.test.ts#L1502
func TestHistory_Revive_ShouldHandleNullUndefinedAndPrimitiveValues(t *testing.T) {
	machine, persistedSnapshot, _ := history1ReviveSetup(t)

	// JS: [null, undefined, 42, 'foo', true, false]; `undefined` is modelled
	// as the key being absent from the spread object.
	cases := []struct {
		name string
		val  any
		omit bool
	}{
		{"null", nil, false},
		{"undefined", nil, true},
		{"42", 42, false},
		{"foo", "foo", false},
		{"true", true, false},
		{"false", false, false},
	}
	for _, c := range cases {
		// JS test (shared case definition): https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/history.test.ts#L1502
		t.Run(c.name, func(t *testing.T) {
			fakeSnapshot := history1With(persistedSnapshot, "historyValue", c.val, c.omit)
			assert.Equal(t, "off", fakeSnapshot["value"])

			actorRef := xs.CreateActor(machine, xs.WithSnapshot(fakeSnapshot)).Start()
			actorRef.Send(xs.Ev("POWER"))

			assert.Equal(t, map[string]any{"on": "first"}, actorRef.GetSnapshot().Value)
			assert.Equal(t, map[string]any{}, history1JSONRoundTrip(t, actorRef.GetPersistedSnapshot())["historyValue"])
		})
	}
}
