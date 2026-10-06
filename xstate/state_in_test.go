package xstate_test

import (
	"testing"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/assert"
)

// JS: transition "in" check > should transition if string state path matches current state value
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/stateIn.test.ts#L5
func TestStateIn_ShouldTransitionIfStringStatePathMatchesCurrentStateValue(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Type: xs.Parallel,
		States: xs.States{
			{
				Key:     "a",
				Initial: "a1",
				States: xs.States{
					{Key: "a1", On: map[string]xs.Transitions{
						"EVENT2": {{Target: "a2", Guard: xs.StateIn(map[string]any{"b": "b2"})}},
					}},
					{Key: "a2", ID: "a_a2"},
				},
			},
			{
				Key:     "b",
				Initial: "b2",
				States: xs.States{
					{Key: "b1", On: map[string]xs.Transitions{
						"EVENT": {{Target: "b2", Guard: xs.StateIn("#a_a2")}},
					}},
					{
						Key:  "b2",
						ID:   "b_b2",
						Type: xs.Parallel,
						States: xs.States{
							{Key: "foo", Initial: "foo2", States: xs.States{{Key: "foo1"}, {Key: "foo2"}}},
							{Key: "bar", Initial: "bar1", States: xs.States{{Key: "bar1", ID: "bar1"}, {Key: "bar2"}}},
						},
					},
				},
			},
		},
	})
	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.Ev("EVENT2"))

	assert.Equal(t, map[string]any{
		"a": "a2",
		"b": map[string]any{
			"b2": map[string]any{
				"foo": "foo2",
				"bar": "bar1",
			},
		},
	}, actorRef.GetSnapshot().Value)
}

// JS: transition "in" check > should transition if state node ID matches current state value
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/stateIn.test.ts#L76
func TestStateIn_ShouldTransitionIfStateNodeIDMatchesCurrentStateValue(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Type: xs.Parallel,
		States: xs.States{
			{
				Key:     "a",
				Initial: "a1",
				States: xs.States{
					{Key: "a1", On: map[string]xs.Transitions{
						"EVENT3": {{Target: "a2", Guard: xs.StateIn("#b_b2")}},
					}},
					{Key: "a2", ID: "a_a2"},
				},
			},
			{
				Key:     "b",
				Initial: "b2",
				States: xs.States{
					{Key: "b1"},
					{
						Key:  "b2",
						ID:   "b_b2",
						Type: xs.Parallel,
						States: xs.States{
							{Key: "foo", Initial: "foo2", States: xs.States{{Key: "foo1"}, {Key: "foo2"}}},
							{Key: "bar", Initial: "bar1", States: xs.States{{Key: "bar1", ID: "bar1"}, {Key: "bar2"}}},
						},
					},
				},
			},
		},
	})
	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.Ev("EVENT3"))

	assert.Equal(t, map[string]any{
		"a": "a2",
		"b": map[string]any{
			"b2": map[string]any{
				"foo": "foo2",
				"bar": "bar1",
			},
		},
	}, actorRef.GetSnapshot().Value)
}

// JS: transition "in" check > should not transition if string state path does not match current state value
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/stateIn.test.ts#L140
func TestStateIn_ShouldNotTransitionIfStringStatePathDoesNotMatchCurrentStateValue(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Type: xs.Parallel,
		States: xs.States{
			{
				Key:     "a",
				Initial: "a1",
				States: xs.States{
					{Key: "a1", On: map[string]xs.Transitions{
						"EVENT1": {{Target: "a2", Guard: xs.StateIn("b.b2")}},
					}},
					{Key: "a2", ID: "a_a2"},
				},
			},
			{
				Key:     "b",
				Initial: "b1",
				States: xs.States{
					{Key: "b1"},
					{
						Key:  "b2",
						ID:   "b_b2",
						Type: xs.Parallel,
						States: xs.States{
							{Key: "foo", Initial: "foo1", States: xs.States{{Key: "foo1"}, {Key: "foo2"}}},
							{Key: "bar", Initial: "bar1", States: xs.States{{Key: "bar1", ID: "bar1"}, {Key: "bar2"}}},
						},
					},
				},
			},
		},
	})
	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.Ev("EVENT1"))

	assert.Equal(t, map[string]any{
		"a": "a1",
		"b": "b1",
	}, actorRef.GetSnapshot().Value)
}

// JS: transition "in" check > should not transition if state value matches current state value
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/stateIn.test.ts#L199
func TestStateIn_ShouldNotTransitionIfStateValueMatchesCurrentStateValue(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Type: xs.Parallel,
		States: xs.States{
			{
				Key:     "a",
				Initial: "a1",
				States: xs.States{
					{Key: "a1", On: map[string]xs.Transitions{
						"EVENT2": {{Target: "a2", Guard: xs.StateIn(map[string]any{"b": "b2"})}},
					}},
					{Key: "a2", ID: "a_a2"},
				},
			},
			{
				Key:     "b",
				Initial: "b2",
				States: xs.States{
					{Key: "b1"},
					{
						Key:  "b2",
						ID:   "b_b2",
						Type: xs.Parallel,
						States: xs.States{
							{Key: "foo", Initial: "foo2", States: xs.States{{Key: "foo1"}, {Key: "foo2"}}},
							{Key: "bar", Initial: "bar1", States: xs.States{{Key: "bar1", ID: "bar1"}, {Key: "bar2"}}},
						},
					},
				},
			},
		},
	})
	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.Ev("EVENT2"))

	assert.Equal(t, map[string]any{
		"a": "a2",
		"b": map[string]any{
			"b2": map[string]any{
				"foo": "foo2",
				"bar": "bar1",
			},
		},
	}, actorRef.GetSnapshot().Value)
}

// JS: transition "in" check > matching should be relative to grandparent (match)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/stateIn.test.ts#L263
func TestStateIn_MatchingShouldBeRelativeToGrandparentMatch(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Type: xs.Parallel,
		States: xs.States{
			{
				Key:     "a",
				Initial: "a1",
				States: xs.States{
					{Key: "a1"},
					{Key: "a2", ID: "a_a2"},
				},
			},
			{
				Key:     "b",
				Initial: "b2",
				States: xs.States{
					{Key: "b1"},
					{
						Key:  "b2",
						ID:   "b_b2",
						Type: xs.Parallel,
						States: xs.States{
							{
								Key:     "foo",
								Initial: "foo1",
								States: xs.States{
									{Key: "foo1", On: map[string]xs.Transitions{
										"EVENT_DEEP": {{Target: "foo2", Guard: xs.StateIn("#bar1")}},
									}},
									{Key: "foo2"},
								},
							},
							{Key: "bar", Initial: "bar1", States: xs.States{{Key: "bar1", ID: "bar1"}, {Key: "bar2"}}},
						},
					},
				},
			},
		},
	})
	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.Ev("EVENT_DEEP"))

	assert.Equal(t, map[string]any{
		"a": "a1",
		"b": map[string]any{
			"b2": map[string]any{
				"foo": "foo2",
				"bar": "bar1",
			},
		},
	}, actorRef.GetSnapshot().Value)
}

// JS: transition "in" check > matching should be relative to grandparent (no match)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/stateIn.test.ts#L324
func TestStateIn_MatchingShouldBeRelativeToGrandparentNoMatch(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Type: xs.Parallel,
		States: xs.States{
			{
				Key:     "a",
				Initial: "a1",
				States: xs.States{
					{Key: "a1"},
					{Key: "a2", ID: "a_a2"},
				},
			},
			{
				Key:     "b",
				Initial: "b2",
				States: xs.States{
					{Key: "b1"},
					{
						Key:  "b2",
						ID:   "b_b2",
						Type: xs.Parallel,
						States: xs.States{
							{
								Key:     "foo",
								Initial: "foo1",
								States: xs.States{
									{Key: "foo1", On: map[string]xs.Transitions{
										"EVENT_DEEP": {{Target: "foo2", Guard: xs.StateIn("#bar1")}},
									}},
									{Key: "foo2"},
								},
							},
							{Key: "bar", Initial: "bar2", States: xs.States{{Key: "bar1", ID: "bar1"}, {Key: "bar2"}}},
						},
					},
				},
			},
		},
	})
	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.Ev("EVENT_DEEP"))

	assert.Equal(t, map[string]any{
		"a": "a1",
		"b": map[string]any{
			"b2": map[string]any{
				"foo": "foo1",
				"bar": "bar2",
			},
		},
	}, actorRef.GetSnapshot().Value)
}

// JS: transition "in" check > should work to forbid events
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/stateIn.test.ts#L385
func TestStateIn_ShouldWorkToForbidEvents(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "green",
		States: xs.States{
			{Key: "green", On: map[string]xs.Transitions{"TIMER": {{Target: "yellow"}}}},
			{Key: "yellow", On: map[string]xs.Transitions{"TIMER": {{Target: "red"}}}},
			{
				Key:     "red",
				Initial: "walk",
				States: xs.States{
					{Key: "walk", On: map[string]xs.Transitions{"TIMER": {{Target: "wait"}}}},
					{Key: "wait", On: map[string]xs.Transitions{"TIMER": {{Target: "stop"}}}},
					{Key: "stop"},
				},
				On: map[string]xs.Transitions{
					"TIMER": {
						{Target: "green", Guard: xs.StateIn(map[string]any{"red": "stop"})},
					},
				},
			},
		},
	})

	actorRef := xs.CreateActor(machine).Start()

	actorRef.Send(xs.Ev("TIMER"))
	actorRef.Send(xs.Ev("TIMER"))
	actorRef.Send(xs.Ev("TIMER"))
	assert.Equal(t, map[string]any{"red": "wait"}, actorRef.GetSnapshot().Value)

	actorRef.Send(xs.Ev("TIMER"))
	assert.Equal(t, map[string]any{"red": "stop"}, actorRef.GetSnapshot().Value)

	actorRef.Send(xs.Ev("TIMER"))
	assert.Equal(t, "green", actorRef.GetSnapshot().Value)
}

// JS: transition "in" check > should be possible to use a referenced `stateIn` guard
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/stateIn.test.ts#L428
func TestStateIn_ShouldBePossibleToUseAReferencedStateInGuard(t *testing.T) {
	machine := xs.CreateMachine(
		xs.MachineConfig[any]{
			Type: xs.Parallel,
			// machine definition,
			States: xs.States{
				{Key: "selected"},
				{
					Key:     "location",
					Initial: "home",
					States: xs.States{
						{Key: "home", On: map[string]xs.Transitions{
							"NEXT": {{Target: "success", Guard: xs.GuardRef{Type: "hasSelection"}}},
						}},
						{Key: "success"},
					},
				},
			},
		},
		xs.Implementations{
			Guards: map[string]xs.Guard{
				"hasSelection": xs.StateIn("selected"),
			},
		},
	)

	actor := xs.CreateActor(machine).Start()
	actor.Send(xs.Ev("NEXT"))
	assert.Equal(t, map[string]any{
		"selected": map[string]any{},
		"location": "success",
	}, actor.GetSnapshot().Value)
}

// JS: transition "in" check > should be possible to check an ID with a path
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/stateIn.test.ts#L468
func TestStateIn_ShouldBePossibleToCheckAnIDWithAPath(t *testing.T) {
	s := newSpy()
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Type: xs.Parallel,
		States: xs.States{
			{
				Key:     "A",
				Initial: "A1",
				States: xs.States{
					{Key: "A1", On: map[string]xs.Transitions{
						"MY_EVENT": {{
							Guard:   xs.StateIn("#b.B1"),
							Actions: xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[any]) { s.Call(a) })},
						}},
					}},
				},
			},
			{
				Key:     "B",
				ID:      "b",
				Initial: "B1",
				States:  xs.States{{Key: "B1"}},
			},
		},
	})

	xs.CreateActor(machine).Start().Send(xs.Ev("MY_EVENT"))

	assert.Equal(t, 1, s.Count())
}
