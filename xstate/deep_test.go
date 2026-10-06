package xstate_test

import (
	"strings"
	"sync"
	"testing"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/assert"
)

// deep1TrackEntries mirrors trackEntries from test/utils.ts: it prepends
// entry/exit tracking actions to every state node of the machine and returns a
// flush function that yields (and clears) the recorded log.
func deep1TrackEntries[C any](machine *xs.StateMachine[C]) func() []string {
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

// JS: deep transitions > exiting super/substates > should exit all substates when superstates exits
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/deep.test.ts#L6
func TestDeep_ExitingSuperSubstates_ShouldExitAllSubstatesWhenSuperstatesExits(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "root",
		Initial: "A",
		States: xs.States{
			{Key: "DONE"},
			{Key: "FAIL"},
			{
				Key: "A",
				On: map[string]xs.Transitions{
					"A_EVENT": {{Target: "#root.DONE"}},
				},
				Initial: "B",
				States: xs.States{
					{
						Key:     "B",
						Initial: "C",
						States: xs.States{
							{
								Key:     "C",
								Initial: "D",
								States:  xs.States{{Key: "D"}},
							},
						},
					},
				},
			},
		},
	})

	flushTracked := deep1TrackEntries(machine)

	actor := xs.CreateActor(machine).Start()
	flushTracked()

	actor.Send(xs.Ev("A_EVENT"))

	assert.Equal(t, []string{
		"exit: A.B.C.D",
		"exit: A.B.C",
		"exit: A.B",
		"exit: A",
		"enter: DONE",
	}, flushTracked())
}

// JS: deep transitions > exiting super/substates > should exit substates and superstates when exiting (B_EVENT)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/deep.test.ts#L53
func TestDeep_ExitingSuperSubstates_ShouldExitSubstatesAndSuperstatesWhenExitingBEvent(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "root",
		Initial: "A",
		States: xs.States{
			{Key: "DONE"},
			{
				Key:     "A",
				Initial: "B",
				States: xs.States{
					{
						Key: "B",
						On: map[string]xs.Transitions{
							"B_EVENT": {{Target: "#root.DONE"}},
						},
						Initial: "C",
						States: xs.States{
							{
								Key:     "C",
								Initial: "D",
								States:  xs.States{{Key: "D"}},
							},
						},
					},
				},
			},
		},
	})

	flushTracked := deep1TrackEntries(machine)

	actor := xs.CreateActor(machine).Start()
	flushTracked()

	actor.Send(xs.Ev("B_EVENT"))

	assert.Equal(t, []string{
		"exit: A.B.C.D",
		"exit: A.B.C",
		"exit: A.B",
		"exit: A",
		"enter: DONE",
	}, flushTracked())
}

// JS: deep transitions > exiting super/substates > should exit substates and superstates when exiting (C_EVENT)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/deep.test.ts#L99
func TestDeep_ExitingSuperSubstates_ShouldExitSubstatesAndSuperstatesWhenExitingCEvent(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "root",
		Initial: "A",
		States: xs.States{
			{Key: "DONE"},
			{
				Key:     "A",
				Initial: "B",
				States: xs.States{
					{
						Key:     "B",
						Initial: "C",
						States: xs.States{
							{
								Key: "C",
								On: map[string]xs.Transitions{
									"C_EVENT": {{Target: "#root.DONE"}},
								},
								Initial: "D",
								States:  xs.States{{Key: "D"}},
							},
						},
					},
				},
			},
		},
	})

	flushTracked := deep1TrackEntries(machine)

	actor := xs.CreateActor(machine).Start()
	flushTracked()

	actor.Send(xs.Ev("C_EVENT"))

	assert.Equal(t, []string{
		"exit: A.B.C.D",
		"exit: A.B.C",
		"exit: A.B",
		"exit: A",
		"enter: DONE",
	}, flushTracked())
}

// JS: deep transitions > exiting super/substates > should exit superstates when exiting (D_EVENT)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/deep.test.ts#L145
func TestDeep_ExitingSuperSubstates_ShouldExitSuperstatesWhenExitingDEvent(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "root",
		Initial: "A",
		States: xs.States{
			{Key: "DONE"},
			{
				Key:     "A",
				Initial: "B",
				States: xs.States{
					{
						Key:     "B",
						Initial: "C",
						States: xs.States{
							{
								Key:     "C",
								Initial: "D",
								States: xs.States{
									{
										Key: "D",
										On: map[string]xs.Transitions{
											"D_EVENT": {{Target: "#root.DONE"}},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	})

	flushTracked := deep1TrackEntries(machine)

	actor := xs.CreateActor(machine).Start()
	flushTracked()

	actor.Send(xs.Ev("D_EVENT"))

	assert.Equal(t, []string{
		"exit: A.B.C.D",
		"exit: A.B.C",
		"exit: A.B",
		"exit: A",
		"enter: DONE",
	}, flushTracked())
}

// JS: deep transitions > exiting super/substates > should exit substate when machine handles event (MACHINE_EVENT)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/deep.test.ts#L192
func TestDeep_ExitingSuperSubstates_ShouldExitSubstateWhenMachineHandlesEventMachineEvent(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "deep",
		Initial: "A",
		On: map[string]xs.Transitions{
			"MACHINE_EVENT": {{Target: "#deep.DONE"}},
		},
		States: xs.States{
			{Key: "DONE"},
			{
				Key:     "A",
				Initial: "B",
				States: xs.States{
					{
						Key:     "B",
						Initial: "C",
						States: xs.States{
							{
								Key:     "C",
								Initial: "D",
								States:  xs.States{{Key: "D"}},
							},
						},
					},
				},
			},
		},
	})

	flushTracked := deep1TrackEntries(machine)

	actor := xs.CreateActor(machine).Start()
	flushTracked()

	actor.Send(xs.Ev("MACHINE_EVENT"))

	assert.Equal(t, []string{
		"exit: A.B.C.D",
		"exit: A.B.C",
		"exit: A.B",
		"exit: A",
		"enter: DONE",
	}, flushTracked())
}

// JS: deep transitions > exiting super/substates > should exit deep and enter deep (A_S)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/deep.test.ts#L238
func TestDeep_ExitingSuperSubstates_ShouldExitDeepAndEnterDeepAS(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "root",
		Initial: "A",
		States: xs.States{
			{
				Key: "A",
				On: map[string]xs.Transitions{
					"A_S": {{Target: "#root.P.Q.R.S"}},
				},
				Initial: "B",
				States: xs.States{
					{
						Key:     "B",
						Initial: "C",
						States: xs.States{
							{
								Key:     "C",
								Initial: "D",
								States:  xs.States{{Key: "D"}},
							},
						},
					},
				},
			},
			{
				Key:     "P",
				Initial: "Q",
				States: xs.States{
					{
						Key:     "Q",
						Initial: "R",
						States: xs.States{
							{
								Key:     "R",
								Initial: "S",
								States:  xs.States{{Key: "S"}},
							},
						},
					},
				},
			},
		},
	})
	flushTracked := deep1TrackEntries(machine)

	actor := xs.CreateActor(machine).Start()
	flushTracked()

	actor.Send(xs.Ev("A_S"))

	assert.Equal(t, []string{
		"exit: A.B.C.D",
		"exit: A.B.C",
		"exit: A.B",
		"exit: A",
		"enter: P",
		"enter: P.Q",
		"enter: P.Q.R",
		"enter: P.Q.R.S",
	}, flushTracked())
}

// JS: deep transitions > exiting super/substates > should exit deep and enter deep (D_P)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/deep.test.ts#L301
func TestDeep_ExitingSuperSubstates_ShouldExitDeepAndEnterDeepDP(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "deep",
		Initial: "A",
		States: xs.States{
			{
				Key:     "A",
				Initial: "B",
				States: xs.States{
					{
						Key:     "B",
						Initial: "C",
						States: xs.States{
							{
								Key:     "C",
								Initial: "D",
								States: xs.States{
									{
										Key: "D",
										On: map[string]xs.Transitions{
											"D_P": {{Target: "#deep.P"}},
										},
									},
								},
							},
						},
					},
				},
			},
			{
				Key:     "P",
				Initial: "Q",
				States: xs.States{
					{
						Key:     "Q",
						Initial: "R",
						States: xs.States{
							{
								Key:     "R",
								Initial: "S",
								States:  xs.States{{Key: "S"}},
							},
						},
					},
				},
			},
		},
	})

	flushTracked := deep1TrackEntries(machine)

	actor := xs.CreateActor(machine).Start()
	flushTracked()

	actor.Send(xs.Ev("D_P"))

	assert.Equal(t, []string{
		"exit: A.B.C.D",
		"exit: A.B.C",
		"exit: A.B",
		"exit: A",
		"enter: P",
		"enter: P.Q",
		"enter: P.Q.R",
		"enter: P.Q.R.S",
	}, flushTracked())
}

// JS: deep transitions > exiting super/substates > should exit deep and enter deep when targeting an ancestor of the final resolved deep target
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/deep.test.ts#L366
func TestDeep_ExitingSuperSubstates_ShouldExitDeepAndEnterDeepWhenTargetingAncestorOfFinalTarget(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "root",
		Initial: "A",
		States: xs.States{
			{
				Key: "A",
				On: map[string]xs.Transitions{
					"A_P": {{Target: "#root.P"}},
				},
				Initial: "B",
				States: xs.States{
					{
						Key:     "B",
						Initial: "C",
						States: xs.States{
							{
								Key:     "C",
								Initial: "D",
								States:  xs.States{{Key: "D"}},
							},
						},
					},
				},
			},
			{
				Key:     "P",
				Initial: "Q",
				States: xs.States{
					{
						Key:     "Q",
						Initial: "R",
						States: xs.States{
							{
								Key:     "R",
								Initial: "S",
								States:  xs.States{{Key: "S"}},
							},
						},
					},
				},
			},
		},
	})

	flushTracked := deep1TrackEntries(machine)

	actor := xs.CreateActor(machine).Start()
	flushTracked()

	actor.Send(xs.Ev("A_P"))

	assert.Equal(t, []string{
		"exit: A.B.C.D",
		"exit: A.B.C",
		"exit: A.B",
		"exit: A",
		"enter: P",
		"enter: P.Q",
		"enter: P.Q.R",
		"enter: P.Q.R.S",
	}, flushTracked())
}

// JS: deep transitions > exiting super/substates > should exit deep and enter deep when targeting a deep state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/deep.test.ts#L430
func TestDeep_ExitingSuperSubstates_ShouldExitDeepAndEnterDeepWhenTargetingDeepState(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "root",
		Initial: "A",
		States: xs.States{
			{
				Key:     "A",
				Initial: "B",
				States: xs.States{
					{
						Key:     "B",
						Initial: "C",
						States: xs.States{
							{
								Key:     "C",
								Initial: "D",
								States: xs.States{
									{
										Key: "D",
										On: map[string]xs.Transitions{
											"D_S": {{Target: "#root.P.Q.R.S"}},
										},
									},
								},
							},
						},
					},
				},
			},
			{
				Key:     "P",
				Initial: "Q",
				States: xs.States{
					{
						Key:     "Q",
						Initial: "R",
						States: xs.States{
							{
								Key:     "R",
								Initial: "S",
								States:  xs.States{{Key: "S"}},
							},
						},
					},
				},
			},
		},
	})

	flushTracked := deep1TrackEntries(machine)

	actor := xs.CreateActor(machine).Start()
	flushTracked()

	actor.Send(xs.Ev("D_S"))

	assert.Equal(t, []string{
		"exit: A.B.C.D",
		"exit: A.B.C",
		"exit: A.B",
		"exit: A",
		"enter: P",
		"enter: P.Q",
		"enter: P.Q.R",
		"enter: P.Q.R.S",
	}, flushTracked())
}
