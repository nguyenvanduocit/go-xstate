package xstate_test

import (
	"encoding/json"
	"regexp"
	"strings"
	"sync"
	"testing"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---- shared helpers (test/utils.ts) ----

// parallel1ResolveSerializedStateValue mirrors resolveSerializedStateValue
// from test/utils.ts.
func parallel1ResolveSerializedStateValue(t *testing.T, machine *xs.StateMachine[any], serialized string) *xs.MachineSnapshot[any] {
	t.Helper()
	if serialized[0] == '{' {
		var value any
		require.NoError(t, json.Unmarshal([]byte(serialized), &value))
		return machine.ResolveState(xs.ResolveStateConfig[any]{Value: value, Context: map[string]any{}})
	}
	return machine.ResolveState(xs.ResolveStateConfig[any]{Value: serialized, Context: map[string]any{}})
}

var parallel1EventSplit = regexp.MustCompile(`,\s?`)

// parallel1TestMultiTransition mirrors testMultiTransition from test/utils.ts.
func parallel1TestMultiTransition(t *testing.T, machine *xs.StateMachine[any], fromState string, eventTypes string) *xs.MachineSnapshot[any] {
	t.Helper()
	state := parallel1ResolveSerializedStateValue(t, machine, fromState)
	for _, eventType := range parallel1EventSplit.Split(eventTypes, -1) {
		state = xs.GetNextSnapshot[*xs.MachineSnapshot[any]](machine, state, xs.Ev(eventType))
	}
	return state
}

// parallel1TrackEntries mirrors trackEntries from test/utils.ts: it prepends
// entry/exit tracking actions to every state node of the machine and returns a
// flush function that yields (and clears) the recorded log.
func parallel1TrackEntries[C any](machine *xs.StateMachine[C]) func() []string {
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

// ---- top-level machines of parallel.test.ts ----

// parallel1SelectionStatus builds the `SelectionStatus` region shared by both
// parallel states of composerMachine (JS lines 25-68 and 140-183; identical).
func parallel1SelectionStatus() xs.StateConfig {
	return xs.StateConfig{
		Key:     "SelectionStatus",
		Initial: "SelectedNone",
		On: map[string]xs.Transitions{
			"singleClickActivity": {{Target: ".SelectedActivity", Actions: xs.Actions{xs.ActionRef{Type: "selectActivity"}}}},
			"singleClickLink":     {{Target: ".SelectedLink", Actions: xs.Actions{xs.ActionRef{Type: "selectLink"}}}},
		},
		States: xs.States{
			{Key: "SelectedNone", Entry: xs.Actions{xs.ActionRef{Type: "redraw"}}},
			{
				Key:   "SelectedActivity",
				Entry: xs.Actions{xs.ActionRef{Type: "redraw"}},
				On: map[string]xs.Transitions{
					"singleClickCanvas": {{Target: "SelectedNone", Actions: xs.Actions{xs.ActionRef{Type: "selectNone"}}}},
				},
			},
			{
				Key:   "SelectedLink",
				Entry: xs.Actions{xs.ActionRef{Type: "redraw"}},
				On: map[string]xs.Transitions{
					"singleClickCanvas": {{Target: "SelectedNone", Actions: xs.Actions{xs.ActionRef{Type: "selectNone"}}}},
				},
			},
		},
	}
}

// parallel1ComposerMachine mirrors `composerMachine` (JS lines 6-189).
func parallel1ComposerMachine() *xs.StateMachine[any] {
	return xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "ReadOnly",
		States: xs.States{
			{
				Key:     "ReadOnly",
				ID:      "ReadOnly",
				Initial: "StructureEdit",
				Entry:   xs.Actions{xs.ActionRef{Type: "selectNone"}},
				States: xs.States{
					{
						Key:  "StructureEdit",
						ID:   "StructureEditRO",
						Type: xs.Parallel,
						On: map[string]xs.Transitions{
							"switchToProjectManagement": {{Target: "ProjectManagement"}},
						},
						States: xs.States{
							parallel1SelectionStatus(),
							{
								Key:     "ClipboardStatus",
								Initial: "Empty",
								States: xs.States{
									{
										Key:   "Empty",
										Entry: xs.Actions{xs.ActionRef{Type: "emptyClipboard"}},
										On: map[string]xs.Transitions{
											"cutInClipboardSuccess":  {{Target: "FilledByCut"}},
											"copyInClipboardSuccess": {{Target: "FilledByCopy"}},
										},
									},
									{
										Key: "FilledByCopy",
										On: map[string]xs.Transitions{
											"cutInClipboardSuccess":     {{Target: "FilledByCut"}},
											"copyInClipboardSuccess":    {{Target: "FilledByCopy"}},
											"pasteFromClipboardSuccess": {{Target: "FilledByCopy"}},
										},
									},
									{
										Key: "FilledByCut",
										On: map[string]xs.Transitions{
											"cutInClipboardSuccess":     {{Target: "FilledByCut"}},
											"copyInClipboardSuccess":    {{Target: "FilledByCopy"}},
											"pasteFromClipboardSuccess": {{Target: "Empty"}},
										},
									},
								},
							},
						},
					},
					{
						Key:  "ProjectManagement",
						ID:   "ProjectManagementRO",
						Type: xs.Parallel,
						On: map[string]xs.Transitions{
							"switchToStructureEdit": {{Target: "StructureEdit"}},
						},
						States: xs.States{
							parallel1SelectionStatus(),
						},
					},
				},
			},
		},
	})
}

// parallel1WakMachine mirrors `wakMachine` (JS lines 191-233).
func parallel1WakMachine() *xs.StateMachine[any] {
	return xs.CreateMachine(xs.MachineConfig[any]{
		ID:   "wakMachine",
		Type: xs.Parallel,
		States: xs.States{
			{
				Key:     "wak1",
				Initial: "wak1sonA",
				States: xs.States{
					{Key: "wak1sonA", Entry: xs.Actions{xs.ActionRef{Type: "wak1sonAenter"}}, Exit: xs.Actions{xs.ActionRef{Type: "wak1sonAexit"}}},
					{Key: "wak1sonB", Entry: xs.Actions{xs.ActionRef{Type: "wak1sonBenter"}}, Exit: xs.Actions{xs.ActionRef{Type: "wak1sonBexit"}}},
				},
				On: map[string]xs.Transitions{
					"WAK1": {{Target: ".wak1sonB"}},
				},
				Entry: xs.Actions{xs.ActionRef{Type: "wak1enter"}},
				Exit:  xs.Actions{xs.ActionRef{Type: "wak1exit"}},
			},
			{
				Key:     "wak2",
				Initial: "wak2sonA",
				States: xs.States{
					{Key: "wak2sonA", Entry: xs.Actions{xs.ActionRef{Type: "wak2sonAenter"}}, Exit: xs.Actions{xs.ActionRef{Type: "wak2sonAexit"}}},
					{Key: "wak2sonB", Entry: xs.Actions{xs.ActionRef{Type: "wak2sonBenter"}}, Exit: xs.Actions{xs.ActionRef{Type: "wak2sonBexit"}}},
				},
				On: map[string]xs.Transitions{
					"WAK2": {{Target: ".wak2sonB"}},
				},
				Entry: xs.Actions{xs.ActionRef{Type: "wak2enter"}},
				Exit:  xs.Actions{xs.ActionRef{Type: "wak2exit"}},
			},
		},
	})
}

// parallel1WordMachine mirrors `wordMachine` (JS lines 235-290).
func parallel1WordMachine() *xs.StateMachine[any] {
	toggle := func(key, event string) xs.StateConfig {
		return xs.StateConfig{
			Key:     key,
			Initial: "off",
			States: xs.States{
				{Key: "on", On: map[string]xs.Transitions{event: {{Target: "off"}}}},
				{Key: "off", On: map[string]xs.Transitions{event: {{Target: "on"}}}},
			},
		}
	}
	return xs.CreateMachine(xs.MachineConfig[any]{
		ID:   "word",
		Type: xs.Parallel,
		States: xs.States{
			toggle("bold", "TOGGLE_BOLD"),
			toggle("underline", "TOGGLE_UNDERLINE"),
			toggle("italics", "TOGGLE_ITALICS"),
			{
				Key:     "list",
				Initial: "none",
				States: xs.States{
					{Key: "none", On: map[string]xs.Transitions{
						"BULLETS": {{Target: "bullets"}},
						"NUMBERS": {{Target: "numbers"}},
					}},
					{Key: "bullets", On: map[string]xs.Transitions{
						"NONE":    {{Target: "none"}},
						"NUMBERS": {{Target: "numbers"}},
					}},
					{Key: "numbers", On: map[string]xs.Transitions{
						"BULLETS": {{Target: "bullets"}},
						"NONE":    {{Target: "none"}},
					}},
				},
			},
		},
		On: map[string]xs.Transitions{
			"RESET": {{Target: "#word"}}, // TODO: this should be 'word' or [{ internal: false }]
		},
	})
}

// parallel1FlatParallelMachine mirrors `flatParallelMachine` (JS lines 292-305).
func parallel1FlatParallelMachine() *xs.StateMachine[any] {
	return xs.CreateMachine(xs.MachineConfig[any]{
		Type: xs.Parallel,
		States: xs.States{
			{Key: "foo"},
			{Key: "bar"},
			{
				Key:     "baz",
				Initial: "one",
				States: xs.States{
					{Key: "one", On: map[string]xs.Transitions{"E": {{Target: "two"}}}},
					{Key: "two"},
				},
			},
		},
	})
}

// parallel1RaisingParallelMachine mirrors `raisingParallelMachine` (JS lines 307-372).
func parallel1RaisingParallelMachine() *xs.StateMachine[any] {
	return xs.CreateMachine(xs.MachineConfig[any]{
		Type: xs.Parallel,
		States: xs.States{
			{
				Key:     "OUTER1",
				Initial: "C",
				States: xs.States{
					{
						Key:   "A",
						Entry: xs.Actions{xs.Raise(xs.Ev("TURN_OFF"))},
						On: map[string]xs.Transitions{
							"EVENT_OUTER1_B": {{Target: "B"}},
							"EVENT_OUTER1_C": {{Target: "C"}},
						},
					},
					{
						Key:   "B",
						Entry: xs.Actions{xs.Raise(xs.Ev("TURN_ON"))},
						On: map[string]xs.Transitions{
							"EVENT_OUTER1_A": {{Target: "A"}},
							"EVENT_OUTER1_C": {{Target: "C"}},
						},
					},
					{
						Key:   "C",
						Entry: xs.Actions{xs.Raise(xs.Ev("CLEAR"))},
						On: map[string]xs.Transitions{
							"EVENT_OUTER1_A": {{Target: "A"}},
							"EVENT_OUTER1_B": {{Target: "B"}},
						},
					},
				},
			},
			{
				Key:  "OUTER2",
				Type: xs.Parallel,
				States: xs.States{
					{
						Key:     "INNER1",
						Initial: "ON",
						States: xs.States{
							{Key: "OFF", On: map[string]xs.Transitions{"TURN_ON": {{Target: "ON"}}}},
							{Key: "ON", On: map[string]xs.Transitions{"CLEAR": {{Target: "OFF"}}}},
						},
					},
					{
						Key:     "INNER2",
						Initial: "OFF",
						States: xs.States{
							{Key: "OFF", On: map[string]xs.Transitions{"TURN_ON": {{Target: "ON"}}}},
							{Key: "ON", On: map[string]xs.Transitions{"TURN_OFF": {{Target: "OFF"}}}},
						},
					},
				},
			},
		},
	})
}

// parallel1NestedParallelState mirrors `nestedParallelState` (JS lines 374-455).
func parallel1NestedParallelState() *xs.StateMachine[any] {
	return xs.CreateMachine(xs.MachineConfig[any]{
		Type: xs.Parallel,
		States: xs.States{
			{
				Key:     "OUTER1",
				Initial: "STATE_OFF",
				States: xs.States{
					{Key: "STATE_OFF", On: map[string]xs.Transitions{
						"EVENT_COMPLEX": {{Target: "STATE_ON"}},
						"EVENT_SIMPLE":  {{Target: "STATE_ON"}},
					}},
					{
						Key:  "STATE_ON",
						Type: xs.Parallel,
						States: xs.States{
							{
								Key:     "STATE_NTJ0",
								Initial: "STATE_IDLE_0",
								States: xs.States{
									{Key: "STATE_IDLE_0", On: map[string]xs.Transitions{
										"EVENT_STATE_NTJ0_WORK": {{Target: "STATE_WORKING_0"}},
									}},
									{Key: "STATE_WORKING_0", On: map[string]xs.Transitions{
										"EVENT_STATE_NTJ0_IDLE": {{Target: "STATE_IDLE_0"}},
									}},
								},
							},
							{
								Key:     "STATE_NTJ1",
								Initial: "STATE_IDLE_1",
								States: xs.States{
									{Key: "STATE_IDLE_1", On: map[string]xs.Transitions{
										"EVENT_STATE_NTJ1_WORK": {{Target: "STATE_WORKING_1"}},
									}},
									{Key: "STATE_WORKING_1", On: map[string]xs.Transitions{
										"EVENT_STATE_NTJ1_IDLE": {{Target: "STATE_IDLE_1"}},
									}},
								},
							},
						},
					},
				},
			},
			{
				Key:     "OUTER2",
				Initial: "STATE_OFF",
				States: xs.States{
					{Key: "STATE_OFF", On: map[string]xs.Transitions{
						"EVENT_COMPLEX": {{Target: "STATE_ON_COMPLEX"}},
						"EVENT_SIMPLE":  {{Target: "STATE_ON_SIMPLE"}},
					}},
					{Key: "STATE_ON_SIMPLE"},
					{
						Key:  "STATE_ON_COMPLEX",
						Type: xs.Parallel,
						States: xs.States{
							{
								Key:     "STATE_INNER1",
								Initial: "STATE_OFF",
								States:  xs.States{{Key: "STATE_OFF"}, {Key: "STATE_ON"}},
							},
							{
								Key:     "STATE_INNER2",
								Initial: "STATE_OFF",
								States:  xs.States{{Key: "STATE_OFF"}, {Key: "STATE_ON"}},
							},
						},
					},
				},
			},
		},
	})
}

// parallel1DeepFlatParallelMachine mirrors `deepFlatParallelMachine` (JS lines 457-492).
func parallel1DeepFlatParallelMachine() *xs.StateMachine[any] {
	return xs.CreateMachine(xs.MachineConfig[any]{
		Type: xs.Parallel,
		States: xs.States{
			{Key: "X"},
			{
				Key:     "V",
				Initial: "A",
				On: map[string]xs.Transitions{
					"a": {{Target: "V.A"}},
					"b": {{Target: "V.B"}},
					"c": {{Target: "V.C"}},
				},
				States: xs.States{
					{Key: "A"},
					{
						Key:     "B",
						Initial: "BB",
						States: xs.States{
							{
								Key:    "BB",
								Type:   xs.Parallel,
								States: xs.States{{Key: "BBB_A"}, {Key: "BBB_B"}},
							},
						},
					},
					{Key: "C"},
				},
			},
		},
	})
}

// ---- tests ----

// JS: parallel states > should have initial parallel states
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/parallel.test.ts#L495
func TestParallel_ShouldHaveInitialParallelStates(t *testing.T) {
	initialState := xs.CreateActor(parallel1WordMachine()).GetSnapshot()

	assert.Equal(t, map[string]any{
		"bold":      "off",
		"italics":   "off",
		"underline": "off",
		"list":      "none",
	}, initialState.Value)
}

// JS: parallel states > should go from {"bold": "off"} to {"bold":"on","italics":"off","underline":"off","list":"none"} on TOGGLE_BOLD
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/parallel.test.ts#L548
// JS case: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/parallel.test.ts#L508
func TestParallel_ShouldGoFromBoldOffToBoldOnOnTOGGLE_BOLD(t *testing.T) {
	resultState := parallel1TestMultiTransition(t, parallel1WordMachine(), `{"bold": "off"}`, "TOGGLE_BOLD")

	assert.Equal(t, map[string]any{
		"bold":      "on",
		"italics":   "off",
		"underline": "off",
		"list":      "none",
	}, resultState.Value)
}

// JS: parallel states > should go from {"bold": "on"} to {"bold":"off","italics":"off","underline":"off","list":"none"} on TOGGLE_BOLD
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/parallel.test.ts#L548
// JS case: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/parallel.test.ts#L516
func TestParallel_ShouldGoFromBoldOnToBoldOffOnTOGGLE_BOLD(t *testing.T) {
	resultState := parallel1TestMultiTransition(t, parallel1WordMachine(), `{"bold": "on"}`, "TOGGLE_BOLD")

	assert.Equal(t, map[string]any{
		"bold":      "off",
		"italics":   "off",
		"underline": "off",
		"list":      "none",
	}, resultState.Value)
}

// JS: parallel states > should go from {"bold":"off","italics":"off","underline":"on","list":"bullets"} to {"bold":"on","italics":"on","underline":"on","list":"bullets"} on TOGGLE_BOLD, TOGGLE_ITALICS
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/parallel.test.ts#L548
// JS case: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/parallel.test.ts#L529
func TestParallel_ShouldGoFromUnderlineOnBulletsToBoldItalicsOnOnTOGGLE_BOLD_TOGGLE_ITALICS(t *testing.T) {
	resultState := parallel1TestMultiTransition(t, parallel1WordMachine(),
		`{"bold":"off","italics":"off","underline":"on","list":"bullets"}`, "TOGGLE_BOLD, TOGGLE_ITALICS")

	assert.Equal(t, map[string]any{
		"bold":      "on",
		"italics":   "on",
		"underline": "on",
		"list":      "bullets",
	}, resultState.Value)
}

// JS: parallel states > should go from {"bold":"off","italics":"off","underline":"on","list":"bullets"} to {"bold":"off","italics":"off","underline":"off","list":"none"} on RESET
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/parallel.test.ts#L548
// JS case: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/parallel.test.ts#L535
func TestParallel_ShouldGoFromUnderlineOnBulletsToAllOffOnRESET(t *testing.T) {
	resultState := parallel1TestMultiTransition(t, parallel1WordMachine(),
		`{"bold":"off","italics":"off","underline":"on","list":"bullets"}`, "RESET")

	assert.Equal(t, map[string]any{
		"bold":      "off",
		"italics":   "off",
		"underline": "off",
		"list":      "none",
	}, resultState.Value)
}

// JS: parallel states > should have all parallel states represented in the state value
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/parallel.test.ts#L562
func TestParallel_ShouldHaveAllParallelStatesRepresentedInTheStateValue(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Type: xs.Parallel,
		States: xs.States{
			{
				Key:     "wak1",
				Initial: "wak1sonA",
				States:  xs.States{{Key: "wak1sonA"}, {Key: "wak1sonB"}},
				On: map[string]xs.Transitions{
					"WAK1": {{Target: ".wak1sonB"}},
				},
			},
			{
				Key:     "wak2",
				Initial: "wak2sonA",
				States:  xs.States{{Key: "wak2sonA"}},
			},
		},
	})
	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.Ev("WAK1"))

	assert.Equal(t, map[string]any{
		"wak1": "wak1sonB",
		"wak2": "wak2sonA",
	}, actorRef.GetSnapshot().Value)
}

// JS: parallel states > should have all parallel states represented in the state value (2)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/parallel.test.ts#L593
func TestParallel_ShouldHaveAllParallelStatesRepresentedInTheStateValue2(t *testing.T) {
	actorRef := xs.CreateActor(parallel1WakMachine()).Start()
	actorRef.Send(xs.Ev("WAK2"))

	assert.Equal(t, map[string]any{
		"wak1": "wak1sonA",
		"wak2": "wak2sonB",
	}, actorRef.GetSnapshot().Value)
}

// JS: parallel states > should work with regions without states
// (first of two JS tests with this exact name; this one checks the initial snapshot)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/parallel.test.ts#L603
func TestParallel_ShouldWorkWithRegionsWithoutStates(t *testing.T) {
	assert.Equal(t, map[string]any{
		"foo": map[string]any{},
		"bar": map[string]any{},
		"baz": "one",
	}, xs.CreateActor(parallel1FlatParallelMachine()).GetSnapshot().Value)
}

// JS: parallel states > should work with regions without states
// (second of two JS tests with this exact name; this one sends E)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/parallel.test.ts#L611
func TestParallel_ShouldWorkWithRegionsWithoutStates2(t *testing.T) {
	actorRef := xs.CreateActor(parallel1FlatParallelMachine()).Start()
	actorRef.Send(xs.Ev("E"))
	assert.Equal(t, map[string]any{
		"foo": map[string]any{},
		"bar": map[string]any{},
		"baz": "two",
	}, actorRef.GetSnapshot().Value)
}

// JS: parallel states > should properly transition to relative substate
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/parallel.test.ts#L621
func TestParallel_ShouldProperlyTransitionToRelativeSubstate(t *testing.T) {
	actorRef := xs.CreateActor(parallel1ComposerMachine()).Start()
	actorRef.Send(xs.Ev("singleClickActivity"))

	assert.Equal(t, map[string]any{
		"ReadOnly": map[string]any{
			"StructureEdit": map[string]any{
				"SelectionStatus": "SelectedActivity",
				"ClipboardStatus": "Empty",
			},
		},
	}, actorRef.GetSnapshot().Value)
}

// JS: parallel states > should properly transition according to entry events on an initial state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/parallel.test.ts#L637
func TestParallel_ShouldProperlyTransitionAccordingToEntryEventsOnAnInitialState(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Type: xs.Parallel,
		States: xs.States{
			{
				Key:     "OUTER1",
				Initial: "B",
				States: xs.States{
					{Key: "A"},
					{Key: "B", Entry: xs.Actions{xs.Raise(xs.Ev("CLEAR"))}},
				},
			},
			{
				Key:  "OUTER2",
				Type: xs.Parallel,
				States: xs.States{
					{
						Key:     "INNER1",
						Initial: "ON",
						States: xs.States{
							{Key: "OFF"},
							{Key: "ON", On: map[string]xs.Transitions{"CLEAR": {{Target: "OFF"}}}},
						},
					},
					{
						Key:     "INNER2",
						Initial: "OFF",
						States:  xs.States{{Key: "OFF"}, {Key: "ON"}},
					},
				},
			},
		},
	})
	assert.Equal(t, map[string]any{
		"OUTER1": "B",
		"OUTER2": map[string]any{
			"INNER1": "OFF",
			"INNER2": "OFF",
		},
	}, xs.CreateActor(machine).GetSnapshot().Value)
}

// JS: parallel states > should properly transition when raising events for a parallel state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/parallel.test.ts#L684
func TestParallel_ShouldProperlyTransitionWhenRaisingEventsForAParallelState(t *testing.T) {
	actorRef := xs.CreateActor(parallel1RaisingParallelMachine()).Start()
	actorRef.Send(xs.Ev("EVENT_OUTER1_B"))

	assert.Equal(t, map[string]any{
		"OUTER1": "B",
		"OUTER2": map[string]any{
			"INNER1": "ON",
			"INNER2": "ON",
		},
	}, actorRef.GetSnapshot().Value)
}

// JS: parallel states > should handle simultaneous orthogonal transitions
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/parallel.test.ts#L699
func TestParallel_ShouldHandleSimultaneousOrthogonalTransitions(t *testing.T) {
	type ctx struct{ Value string }

	simultaneousMachine := xs.CreateMachine(xs.MachineConfig[ctx]{
		ID:      "yamlEditor",
		Type:    xs.Parallel,
		Context: ctx{Value: ""},
		States: xs.States{
			{
				Key: "editing",
				On: map[string]xs.Transitions{
					"CHANGE": {{Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
						c := a.Context
						c.Value = a.Event.(xs.E)["value"].(string)
						return c
					})}}},
				},
			},
			{
				Key:     "status",
				Initial: "unsaved",
				States: xs.States{
					{Key: "unsaved", On: map[string]xs.Transitions{
						"SAVE": {{Target: "saved", Actions: xs.Actions{xs.ActionRef{Type: "save"}}}},
					}},
					{Key: "saved", On: map[string]xs.Transitions{
						"CHANGE": {{Target: "unsaved"}},
					}},
				},
			},
		},
	})

	actorRef := xs.CreateActor(simultaneousMachine).Start()
	actorRef.Send(xs.Ev("SAVE"))
	actorRef.Send(xs.E{"type": "CHANGE", "value": "something"})

	assert.Equal(t, map[string]any{
		"editing": map[string]any{},
		"status":  "unsaved",
	}, actorRef.GetSnapshot().Value)

	assert.Equal(t, ctx{Value: "something"}, actorRef.GetSnapshot().Context)
}

// JS: parallel states > should execute actions of the initial transition of a parallel region when entering the initial state nodes of a machine
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/parallel.test.ts#L758
func TestParallel_ShouldExecuteInitialTransitionActionsOfParallelRegionWhenEnteringInitialStateNodesOfMachine(t *testing.T) {
	s := newSpy()

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Type: xs.Parallel,
		States: xs.States{
			xs.WithStateInitialActions(xs.StateConfig{
				Key:     "a",
				Initial: "a1",
				States:  xs.States{{Key: "a1"}},
			}, xs.ActionFunc(func(xs.ActionArgs[any]) { s.Call() })),
		},
	})

	xs.CreateActor(machine).Start()

	assert.Equal(t, 1, s.Count())
}

// JS: parallel states > should execute actions of the initial transition of a parallel region when the parallel state is targeted with an explicit transition
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/parallel.test.ts#L781
func TestParallel_ShouldExecuteInitialTransitionActionsOfParallelRegionWhenParallelStateIsTargetedExplicitly(t *testing.T) {
	s := newSpy()

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{"NEXT": {{Target: "b"}}}},
			{
				Key:  "b",
				Type: xs.Parallel,
				States: xs.States{
					xs.WithStateInitialActions(xs.StateConfig{
						Key:     "c",
						Initial: "c1",
						States:  xs.States{{Key: "c1"}},
					}, xs.ActionFunc(func(xs.ActionArgs[any]) { s.Call() })),
				},
			},
		},
	})

	actorRef := xs.CreateActor(machine).Start()

	actorRef.Send(xs.Ev("NEXT"))

	assert.Equal(t, 1, s.Count())
}

// JS: parallel states > transitions with nested parallel states > should properly transition when in a simple nested state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/parallel.test.ts#L817
func TestParallel_TransitionsWithNestedParallelStates_ShouldProperlyTransitionWhenInASimpleNestedState(t *testing.T) {
	actorRef := xs.CreateActor(parallel1NestedParallelState()).Start()
	actorRef.Send(xs.Ev("EVENT_SIMPLE"))
	actorRef.Send(xs.Ev("EVENT_STATE_NTJ0_WORK"))

	assert.Equal(t, map[string]any{
		"OUTER1": map[string]any{
			"STATE_ON": map[string]any{
				"STATE_NTJ0": "STATE_WORKING_0",
				"STATE_NTJ1": "STATE_IDLE_1",
			},
		},
		"OUTER2": "STATE_ON_SIMPLE",
	}, actorRef.GetSnapshot().Value)
}

// JS: parallel states > transitions with nested parallel states > should properly transition when in a complex nested state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/parallel.test.ts#L837
func TestParallel_TransitionsWithNestedParallelStates_ShouldProperlyTransitionWhenInAComplexNestedState(t *testing.T) {
	actorRef := xs.CreateActor(parallel1NestedParallelState()).Start()
	actorRef.Send(xs.Ev("EVENT_COMPLEX"))
	actorRef.Send(xs.Ev("EVENT_STATE_NTJ0_WORK"))

	assert.Equal(t, map[string]any{
		"OUTER1": map[string]any{
			"STATE_ON": map[string]any{
				"STATE_NTJ0": "STATE_WORKING_0",
				"STATE_NTJ1": "STATE_IDLE_1",
			},
		},
		"OUTER2": map[string]any{
			"STATE_ON_COMPLEX": map[string]any{
				"STATE_INNER1": "STATE_OFF",
				"STATE_INNER2": "STATE_OFF",
			},
		},
	}, actorRef.GetSnapshot().Value)
}

// JS: parallel states > nested flat parallel states > should represent the flat nested parallel states in the state value
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/parallel.test.ts#L886
func TestParallel_NestedFlatParallelStates_ShouldRepresentTheFlatNestedParallelStatesInTheStateValue(t *testing.T) {
	// machine declared in the describe body (JS lines 865-884)
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "A",
		States: xs.States{
			{Key: "A", On: map[string]xs.Transitions{"to-B": {{Target: "B"}}}},
			{
				Key:    "B",
				Type:   xs.Parallel,
				States: xs.States{{Key: "C"}, {Key: "D"}},
			},
		},
		On: map[string]xs.Transitions{
			"to-A": {{Target: ".A"}},
		},
	})

	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.Ev("to-B"))

	assert.Equal(t, map[string]any{
		"B": map[string]any{
			"C": map[string]any{},
			"D": map[string]any{},
		},
	}, actorRef.GetSnapshot().Value)
}

// JS: parallel states > deep flat parallel states > should properly evaluate deep flat parallel states
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/parallel.test.ts#L902
func TestParallel_DeepFlatParallelStates_ShouldProperlyEvaluateDeepFlatParallelStates(t *testing.T) {
	actorRef := xs.CreateActor(parallel1DeepFlatParallelMachine()).Start()

	actorRef.Send(xs.Ev("a"))
	actorRef.Send(xs.Ev("c"))
	actorRef.Send(xs.Ev("b"))

	assert.Equal(t, map[string]any{
		"V": map[string]any{
			"B": map[string]any{
				"BB": map[string]any{
					"BBB_A": map[string]any{},
					"BBB_B": map[string]any{},
				},
			},
		},
		"X": map[string]any{},
	}, actorRef.GetSnapshot().Value)
}

// JS: parallel states > deep flat parallel states > should not overlap resolved state nodes in state resolution
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/parallel.test.ts#L922
func TestParallel_DeepFlatParallelStates_ShouldNotOverlapResolvedStateNodesInStateResolution(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		ID:   "pipeline",
		Type: xs.Parallel,
		States: xs.States{
			{
				Key: "foo",
				On: map[string]xs.Transitions{
					"UPDATE": {{Actions: xs.Actions{xs.ActionFunc(func(xs.ActionArgs[any]) {
						/* do nothing */
					})}}},
				},
			},
			{
				Key: "bar",
				On: map[string]xs.Transitions{
					"UPDATE": {{Target: ".baz"}},
				},
				Initial: "idle",
				States:  xs.States{{Key: "idle"}, {Key: "baz"}},
			},
		},
	})

	actorRef := xs.CreateActor(machine).Start()
	assert.NotPanics(t, func() {
		actorRef.Send(xs.Ev("UPDATE"))
	})
}

// JS: parallel states > other > regions should be able to transition to orthogonal regions
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/parallel.test.ts#L960
func TestParallel_Other_RegionsShouldBeAbleToTransitionToOrthogonalRegions(t *testing.T) {
	testMachine := xs.CreateMachine(xs.MachineConfig[any]{
		Type: xs.Parallel,
		States: xs.States{
			{
				Key:     "Pages",
				Initial: "About",
				States: xs.States{
					{Key: "About", ID: "About"},
					{Key: "Dashboard", ID: "Dashboard"},
				},
			},
			{
				Key:     "Menu",
				Initial: "Closed",
				States: xs.States{
					{Key: "Closed", ID: "Closed", On: map[string]xs.Transitions{
						"toggle": {{Target: "#Opened"}},
					}},
					{Key: "Opened", ID: "Opened", On: map[string]xs.Transitions{
						"toggle":          {{Target: "#Closed"}},
						"go to dashboard": {{Targets: []string{"#Dashboard", "#Opened"}}},
					}},
				},
			},
		},
	})

	actorRef := xs.CreateActor(testMachine).Start()

	actorRef.Send(xs.Ev("toggle"))
	actorRef.Send(xs.Ev("go to dashboard"))

	assert.True(t, actorRef.GetSnapshot().Matches(map[string]any{"Menu": "Opened", "Pages": "Dashboard"}))
}

// JS: parallel states > other > should calculate the entry set for reentering transitions in parallel states
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/parallel.test.ts#L1009
func TestParallel_Other_ShouldCalculateTheEntrySetForReenteringTransitionsInParallelStates(t *testing.T) {
	type ctx struct{ Log []string }

	testMachine := xs.CreateMachine(xs.MachineConfig[ctx]{
		ID:      "test",
		Context: ctx{Log: []string{}},
		Type:    xs.Parallel,
		States: xs.States{
			{
				Key:     "foo",
				Initial: "foobar",
				States: xs.States{
					{Key: "foobar", On: map[string]xs.Transitions{
						"GOTO_FOOBAZ": {{Target: "foobaz"}},
					}},
					{
						Key: "foobaz",
						Entry: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
							next := make([]string, 0, len(a.Context.Log)+1)
							next = append(next, a.Context.Log...)
							next = append(next, "entered foobaz")
							return ctx{Log: next}
						})},
						On: map[string]xs.Transitions{
							"GOTO_FOOBAZ": {{Target: "foobaz", Reenter: true}},
						},
					},
				},
			},
			{Key: "bar"},
		},
	})

	actorRef := xs.CreateActor(testMachine).Start()

	actorRef.Send(xs.Ev("GOTO_FOOBAZ"))
	actorRef.Send(xs.Ev("GOTO_FOOBAZ"))

	assert.Equal(t, 2, len(actorRef.GetSnapshot().Context.Log))
}

// JS: parallel states > should raise a "xstate.done.state.*" event when all child states reach final state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/parallel.test.ts#L1054
func TestParallel_ShouldRaiseDoneStateEventWhenAllChildStatesReachFinalState(t *testing.T) {
	sig := newSignal()

	region := func(key string) xs.StateConfig {
		return xs.StateConfig{
			Key:     key,
			Initial: "idle",
			States: xs.States{
				{Key: "idle", On: map[string]xs.Transitions{"FINISH": {{Target: "finished"}}}},
				{Key: "finished", Type: xs.Final},
			},
		}
	}

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "test",
		Initial: "p",
		States: xs.States{
			{
				Key:    "p",
				Type:   xs.Parallel,
				States: xs.States{region("a"), region("b"), region("c")},
				OnDone: xs.Transitions{{Target: "success"}},
			},
			{Key: "success", Type: xs.Final},
		},
	})

	service := xs.CreateActor(machine)
	service.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{
		Complete: func() {
			sig.Resolve()
		},
	})
	service.Start()

	service.Send(xs.Ev("FINISH"))

	sig.Wait(t)
}

// JS: parallel states > should raise a "xstate.done.state.*" event when a pseudostate of a history type is directly on a parallel state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/parallel.test.ts#L1125
func TestParallel_ShouldRaiseDoneStateEventWhenHistoryPseudostateIsDirectlyOnParallelState(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "parallelSteps",
		States: xs.States{
			{
				Key:  "parallelSteps",
				Type: xs.Parallel,
				States: xs.States{
					{Key: "hist", Type: xs.History},
					{
						Key:     "one",
						Initial: "wait_one",
						States: xs.States{
							{Key: "wait_one", On: map[string]xs.Transitions{
								"finish_one": {{Target: "done"}},
							}},
							{Key: "done", Type: xs.Final},
						},
					},
					{
						Key:     "two",
						Initial: "wait_two",
						States: xs.States{
							{Key: "wait_two", On: map[string]xs.Transitions{
								"finish_two": {{Target: "done"}},
							}},
							{Key: "done", Type: xs.Final},
						},
					},
				},
				OnDone: xs.Transitions{{Target: "finished"}},
			},
			{Key: "finished"},
		},
	})

	service := xs.CreateActor(machine).Start()

	service.Send(xs.Ev("finish_one"))
	service.Send(xs.Ev("finish_two"))

	assert.Equal(t, "finished", service.GetSnapshot().Value)
}

// JS: parallel states > source parallel region should be reentered when a transition within it targets another parallel region (parallel root)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/parallel.test.ts#L1180
func TestParallel_SourceParallelRegionShouldBeReenteredWhenTargetingAnotherParallelRegion_ParallelRoot(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Type: xs.Parallel,
		States: xs.States{
			{
				Key:     "Operation",
				Initial: "Waiting",
				States: xs.States{
					{Key: "Waiting", On: map[string]xs.Transitions{
						"TOGGLE_MODE": {{Target: "#Demo"}},
					}},
					{Key: "Fetching"},
				},
			},
			{
				Key:     "Mode",
				Initial: "Normal",
				States: xs.States{
					{Key: "Normal"},
					{Key: "Demo", ID: "Demo"},
				},
			},
		},
	})

	flushTracked := parallel1TrackEntries(machine)

	actor := xs.CreateActor(machine)
	actor.Start()
	flushTracked()

	actor.Send(xs.Ev("TOGGLE_MODE"))

	assert.Equal(t, []string{
		"exit: Mode.Normal",
		"exit: Mode",
		"exit: Operation.Waiting",
		"exit: Operation",
		"enter: Operation",
		"enter: Operation.Waiting",
		"enter: Mode",
		"enter: Mode.Demo",
	}, flushTracked())
}

// JS: parallel states > source parallel region should be reentered when a transition within it targets another parallel region (nested parallel)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/parallel.test.ts#L1229
func TestParallel_SourceParallelRegionShouldBeReenteredWhenTargetingAnotherParallelRegion_NestedParallel(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{
				Key:  "a",
				Type: xs.Parallel,
				States: xs.States{
					{
						Key:     "Operation",
						Initial: "Waiting",
						States: xs.States{
							{Key: "Waiting", On: map[string]xs.Transitions{
								"TOGGLE_MODE": {{Target: "#Demo"}},
							}},
							{Key: "Fetching"},
						},
					},
					{
						Key:     "Mode",
						Initial: "Normal",
						States: xs.States{
							{Key: "Normal"},
							{Key: "Demo", ID: "Demo"},
						},
					},
				},
			},
		},
	})

	flushTracked := parallel1TrackEntries(machine)

	actor := xs.CreateActor(machine)
	actor.Start()
	flushTracked()

	actor.Send(xs.Ev("TOGGLE_MODE"))

	assert.Equal(t, []string{
		"exit: a.Mode.Normal",
		"exit: a.Mode",
		"exit: a.Operation.Waiting",
		"exit: a.Operation",
		"enter: a.Operation",
		"enter: a.Operation.Waiting",
		"enter: a.Mode",
		"enter: a.Mode.Demo",
	}, flushTracked())
}

// JS: parallel states > targetless transition on a parallel state should not enter nor exit any states
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/parallel.test.ts#L1283
func TestParallel_TargetlessTransitionOnAParallelStateShouldNotEnterNorExitAnyStates(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		ID:   "test",
		Type: xs.Parallel,
		States: xs.States{
			{
				Key:     "first",
				Initial: "disabled",
				States:  xs.States{{Key: "disabled"}, {Key: "enabled"}},
			},
			{Key: "second"},
		},
		On: map[string]xs.Transitions{
			"MY_EVENT": {{Actions: xs.Actions{xs.ActionFunc(func(xs.ActionArgs[any]) {})}}},
		},
	})

	flushTracked := parallel1TrackEntries(machine)

	actor := xs.CreateActor(machine)
	actor.Start()
	flushTracked()

	actor.Send(xs.Ev("MY_EVENT"))

	assert.Equal(t, []string{}, flushTracked())
}

// JS: parallel states > targetless transition in one of the parallel regions should not enter nor exit any states
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/parallel.test.ts#L1315
func TestParallel_TargetlessTransitionInOneOfTheParallelRegionsShouldNotEnterNorExitAnyStates(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		ID:   "test",
		Type: xs.Parallel,
		States: xs.States{
			{
				Key:     "first",
				Initial: "disabled",
				States:  xs.States{{Key: "disabled"}, {Key: "enabled"}},
				On: map[string]xs.Transitions{
					"MY_EVENT": {{Actions: xs.Actions{xs.ActionFunc(func(xs.ActionArgs[any]) {})}}},
				},
			},
			{Key: "second"},
		},
	})

	flushTracked := parallel1TrackEntries(machine)

	actor := xs.CreateActor(machine)
	actor.Start()
	flushTracked()

	actor.Send(xs.Ev("MY_EVENT"))

	assert.Equal(t, []string{}, flushTracked())
}
