package xstate_test

import (
	"errors"
	"fmt"
	"testing"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/assert"
)

// transient1GreetingCtx mirrors `const greetingContext = { hour: 10 }`.
type transient1GreetingCtx struct{ Hour int }

// transient1GreetingMachine mirrors the top-level `greetingMachine`
// (transient.test.ts lines 6-28). A func so that the package can load while
// the library is stubbed.
func transient1GreetingMachine() *xs.StateMachine[transient1GreetingCtx] {
	return xs.CreateMachine(xs.MachineConfig[transient1GreetingCtx]{
		ID:      "greeting",
		Initial: "pending",
		Context: transient1GreetingCtx{Hour: 10},
		States: xs.States{
			{Key: "pending", Always: xs.Transitions{
				{Target: "morning", Guard: xs.GuardFunc(func(a xs.GuardArgs[transient1GreetingCtx]) bool { return a.Context.Hour < 12 })},
				{Target: "afternoon", Guard: xs.GuardFunc(func(a xs.GuardArgs[transient1GreetingCtx]) bool { return a.Context.Hour < 18 })},
				{Target: "evening"},
			}},
			{Key: "morning"},
			{Key: "afternoon"},
			{Key: "evening"},
		},
		On: map[string]xs.Transitions{
			"CHANGE": {{Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[transient1GreetingCtx]) transient1GreetingCtx {
				c := a.Context
				c.Hour = 20
				return c
			})}}},
			"RECHECK": {{Target: "#greeting"}},
		},
	})
}

// JS: transient states (eventless transitions) > should choose the first candidate target that matches the guard 1
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/transient.test.ts#L31
func TestTransient_ShouldChooseTheFirstCandidateTargetThatMatchesTheGuard1(t *testing.T) {
	type ctx struct{ Data bool }
	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Context: ctx{Data: false},
		Initial: "G",
		States: xs.States{
			{Key: "G", On: map[string]xs.Transitions{"UPDATE_BUTTON_CLICKED": {{Target: "E"}}}},
			{Key: "E", Always: xs.Transitions{
				{Target: "D", Guard: xs.GuardFunc(func(a xs.GuardArgs[ctx]) bool { return !a.Context.Data })},
				{Target: "F"},
			}},
			{Key: "D"},
			{Key: "F"},
		},
	})

	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.Ev("UPDATE_BUTTON_CLICKED"))

	assert.Equal(t, "D", actorRef.GetSnapshot().Value)
}

// JS: transient states (eventless transitions) > should choose the first candidate target that matches the guard 2
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/transient.test.ts#L57
func TestTransient_ShouldChooseTheFirstCandidateTargetThatMatchesTheGuard2(t *testing.T) {
	type ctx struct {
		Data   bool
		Status string
	}
	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Context: ctx{Data: false},
		Initial: "G",
		States: xs.States{
			{Key: "G", On: map[string]xs.Transitions{"UPDATE_BUTTON_CLICKED": {{Target: "E"}}}},
			{Key: "E", Always: xs.Transitions{
				{Target: "D", Guard: xs.GuardFunc(func(a xs.GuardArgs[ctx]) bool { return !a.Context.Data })},
				{Target: "F", Guard: xs.GuardFunc(func(a xs.GuardArgs[ctx]) bool { return true })},
			}},
			{Key: "D"},
			{Key: "F"},
		},
	})

	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.Ev("UPDATE_BUTTON_CLICKED"))

	assert.Equal(t, "D", actorRef.GetSnapshot().Value)
}

// JS: transient states (eventless transitions) > should choose the final candidate without a guard if none others match
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/transient.test.ts#L83
func TestTransient_ShouldChooseTheFinalCandidateWithoutAGuardIfNoneOthersMatch(t *testing.T) {
	type ctx struct {
		Data   bool
		Status string
	}
	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Context: ctx{Data: true},
		Initial: "G",
		States: xs.States{
			{Key: "G", On: map[string]xs.Transitions{"UPDATE_BUTTON_CLICKED": {{Target: "E"}}}},
			{Key: "E", Always: xs.Transitions{
				{Target: "D", Guard: xs.GuardFunc(func(a xs.GuardArgs[ctx]) bool { return !a.Context.Data })},
				{Target: "F"},
			}},
			{Key: "D"},
			{Key: "F"},
		},
	})
	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.Ev("UPDATE_BUTTON_CLICKED"))

	assert.Equal(t, "F", actorRef.GetSnapshot().Value)
}

// JS: transient states (eventless transitions) > should carry actions from previous transitions within same step
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/transient.test.ts#L108
func TestTransient_ShouldCarryActionsFromPreviousTransitionsWithinSameStep(t *testing.T) {
	actual := []string{}
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "A",
		States: xs.States{
			{
				Key:  "A",
				Exit: xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[any]) { actual = append(actual, "exit_A") })},
				On: map[string]xs.Transitions{
					"TIMER": {{
						Target:  "T",
						Actions: xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[any]) { actual = append(actual, "timer") })},
					}},
				},
			},
			{Key: "T", Always: xs.Transitions{{Target: "B"}}},
			{Key: "B", Entry: xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[any]) { actual = append(actual, "enter_B") })}},
		},
	})

	actor := xs.CreateActor(machine).Start()

	actor.Send(xs.Ev("TIMER"))

	assert.Equal(t, []string{"exit_A", "timer", "enter_B"}, actual)
}

// JS: transient states (eventless transitions) > should execute all internal events one after the other
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/transient.test.ts#L138
func TestTransient_ShouldExecuteAllInternalEventsOneAfterTheOther(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Type: xs.Parallel,
		States: xs.States{
			{Key: "A", Initial: "A1", States: xs.States{
				{Key: "A1", On: map[string]xs.Transitions{"E": {{Target: "A2"}}}},
				{Key: "A2", Entry: xs.Actions{xs.Raise(xs.Ev("INT1"))}},
			}},
			{Key: "B", Initial: "B1", States: xs.States{
				{Key: "B1", On: map[string]xs.Transitions{"E": {{Target: "B2"}}}},
				{Key: "B2", Entry: xs.Actions{xs.Raise(xs.Ev("INT2"))}},
			}},
			{Key: "C", Initial: "C1", States: xs.States{
				{Key: "C1", On: map[string]xs.Transitions{
					"INT1": {{Target: "C2"}},
					"INT2": {{Target: "C3"}},
				}},
				{Key: "C2", On: map[string]xs.Transitions{"INT2": {{Target: "C4"}}}},
				{Key: "C3", On: map[string]xs.Transitions{"INT1": {{Target: "C4"}}}},
				{Key: "C4"},
			}},
		},
	})

	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.Ev("E"))

	assert.Equal(t, map[string]any{"A": "A2", "B": "B2", "C": "C4"}, actorRef.GetSnapshot().Value)
}

// JS: transient states (eventless transitions) > should execute all eventless transitions in the same microstep
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/transient.test.ts#L201
func TestTransient_ShouldExecuteAllEventlessTransitionsInTheSameMicrostep(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Type: xs.Parallel,
		States: xs.States{
			{Key: "A", Initial: "A1", States: xs.States{
				{Key: "A1", On: map[string]xs.Transitions{"E": {{Target: "A2"}}}}, // the external event
				{Key: "A2", Always: xs.Transitions{{Target: "A3"}}},
				{Key: "A3", Always: xs.Transitions{{Target: "A4", Guard: xs.StateIn(map[string]any{"B": "B3"})}}},
				{Key: "A4"},
			}},
			{Key: "B", Initial: "B1", States: xs.States{
				{Key: "B1", On: map[string]xs.Transitions{"E": {{Target: "B2"}}}},
				{Key: "B2", Always: xs.Transitions{{Target: "B3", Guard: xs.StateIn(map[string]any{"A": "A2"})}}},
				{Key: "B3", Always: xs.Transitions{{Target: "B4", Guard: xs.StateIn(map[string]any{"A": "A3"})}}},
				{Key: "B4"},
			}},
		},
	})

	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.Ev("E"))

	assert.Equal(t, map[string]any{"A": "A4", "B": "B4"}, actorRef.GetSnapshot().Value)
}

// JS: transient states (eventless transitions) > should check for automatic transitions even after microsteps are done
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/transient.test.ts#L258
func TestTransient_ShouldCheckForAutomaticTransitionsEvenAfterMicrostepsAreDone(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Type: xs.Parallel,
		States: xs.States{
			{Key: "A", Initial: "A1", States: xs.States{
				{Key: "A1", On: map[string]xs.Transitions{"A": {{Target: "A2"}}}},
				{Key: "A2"},
			}},
			{Key: "B", Initial: "B1", States: xs.States{
				{Key: "B1", Always: xs.Transitions{{Target: "B2", Guard: xs.StateIn(map[string]any{"A": "A2"})}}},
				{Key: "B2"},
			}},
			{Key: "C", Initial: "C1", States: xs.States{
				{Key: "C1", Always: xs.Transitions{{Target: "C2", Guard: xs.StateIn(map[string]any{"A": "A2"})}}},
				{Key: "C2"},
			}},
		},
	})

	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.Ev("A"))

	assert.Equal(t, map[string]any{"A": "A2", "B": "B2", "C": "C2"}, actorRef.GetSnapshot().Value)
}

// JS: transient states (eventless transitions) > should determine the resolved initial state from the transient state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/transient.test.ts#L306
func TestTransient_ShouldDetermineTheResolvedInitialStateFromTheTransientState(t *testing.T) {
	assert.Equal(t, "morning", xs.CreateActor(transient1GreetingMachine()).GetSnapshot().Value)
}

// JS: transient states (eventless transitions) > should determine the resolved state from an initial transient state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/transient.test.ts#L310
func TestTransient_ShouldDetermineTheResolvedStateFromAnInitialTransientState(t *testing.T) {
	actorRef := xs.CreateActor(transient1GreetingMachine()).Start()

	actorRef.Send(xs.Ev("CHANGE"))
	assert.Equal(t, "morning", actorRef.GetSnapshot().Value)

	actorRef.Send(xs.Ev("RECHECK"))
	assert.Equal(t, "evening", actorRef.GetSnapshot().Value)
}

// JS: transient states (eventless transitions) > should select eventless transition before processing raised events
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/transient.test.ts#L320
func TestTransient_ShouldSelectEventlessTransitionBeforeProcessingRaisedEvents(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{"FOO": {{Target: "b"}}}},
			{
				Key:    "b",
				Entry:  xs.Actions{xs.Raise(xs.Ev("BAR"))},
				Always: xs.Transitions{{Target: "c"}},
				On:     map[string]xs.Transitions{"BAR": {{Target: "d"}}},
			},
			{Key: "c", On: map[string]xs.Transitions{"BAR": {{Target: "e"}}}},
			{Key: "d"},
			{Key: "e"},
		},
	})

	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.Ev("FOO"))

	assert.Equal(t, "e", actorRef.GetSnapshot().Value)
}

// JS: transient states (eventless transitions) > should not select wildcard for eventless transition
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/transient.test.ts#L352
func TestTransient_ShouldNotSelectWildcardForEventlessTransition(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{"FOO": {{Target: "b"}}}},
			{
				Key:    "b",
				Always: xs.Transitions{{Target: "pass"}},
				On:     map[string]xs.Transitions{"*": {{Target: "fail"}}},
			},
			{Key: "fail"},
			{Key: "pass"},
		},
	})

	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.Ev("FOO"))

	assert.Equal(t, "pass", actorRef.GetSnapshot().Value)
}

// JS: transient states (eventless transitions) > should work with transient transition on root
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/transient.test.ts#L376
func TestTransient_ShouldWorkWithTransientTransitionOnRoot(t *testing.T) {
	type ctx struct{ Count int }
	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		ID:      "machine",
		Initial: "first",
		Context: ctx{Count: 0},
		States: xs.States{
			{Key: "first", On: map[string]xs.Transitions{
				"ADD": {{Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
					c := a.Context
					c.Count = a.Context.Count + 1
					return c
				})}}},
			}},
			{Key: "success", Type: xs.Final},
		},
		Always: xs.Transitions{{
			Target: ".success",
			Guard: xs.GuardFunc(func(a xs.GuardArgs[ctx]) bool {
				return a.Context.Count > 0
			}),
		}},
	})

	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.Ev("ADD"))

	assert.Equal(t, xs.StatusDone, actorRef.GetSnapshot().Status)
}

// JS: transient states (eventless transitions) > shouldn't crash when invoking a machine with initial transient transition depending on custom data
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/transient.test.ts#L410
func TestTransient_ShouldntCrashWhenInvokingMachineWithInitialTransientTransitionOnCustomData(t *testing.T) {
	type timerInput struct{ Duration int }
	type timerCtx struct{ Duration int }
	type parentCtx struct{ CustomDuration int }

	timerMachine := xs.CreateMachine(xs.MachineConfig[timerCtx]{
		Initial: "initial",
		ContextFn: func(a xs.ContextArgs) timerCtx {
			return timerCtx{Duration: a.Input.(timerInput).Duration}
		},
		States: xs.States{
			{Key: "initial", Always: xs.Transitions{
				{Target: "finished", Guard: xs.GuardFunc(func(a xs.GuardArgs[timerCtx]) bool { return a.Context.Duration < 1000 })},
				{Target: "active"},
			}},
			{Key: "active"},
			{Key: "finished", Type: xs.Final},
		},
	})

	machine := xs.CreateMachine(xs.MachineConfig[parentCtx]{
		Initial: "active",
		Context: parentCtx{CustomDuration: 3000},
		States: xs.States{
			{Key: "active", Invoke: []xs.InvokeConfig{{
				Logic: timerMachine,
				Input: xs.NewExpr(func(a xs.ExprArgs[parentCtx]) any {
					return timerInput{Duration: a.Context.CustomDuration}
				}),
			}}},
		},
	})

	actorRef := xs.CreateActor(machine)
	assert.NotPanics(t, func() { actorRef.Start() })
}

// JS: transient states (eventless transitions) > should be taken even in absence of other transitions
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/transient.test.ts#L457
func TestTransient_ShouldBeTakenEvenInAbsenceOfOtherTransitions(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", Always: xs.Transitions{{
				Target: "b",
				Guard:  xs.GuardFunc(func(a xs.GuardArgs[any]) bool { return a.Event.EventType() == "WHATEVER" }),
			}}},
			{Key: "b"},
		},
	})
	actorRef := xs.CreateActor(machine).Start()

	actorRef.Send(xs.Ev("WHATEVER"))

	assert.Equal(t, "b", actorRef.GetSnapshot().Value)
}

// JS: transient states (eventless transitions) > should select subsequent transient transitions even in absence of other transitions
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/transient.test.ts#L477
func TestTransient_ShouldSelectSubsequentTransientTransitionsEvenInAbsenceOfOtherTransitions(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", Always: xs.Transitions{{
				Target: "b",
				Guard:  xs.GuardFunc(func(a xs.GuardArgs[any]) bool { return a.Event.EventType() == "WHATEVER" }),
			}}},
			{Key: "b", Always: xs.Transitions{{
				Target: "c",
				Guard:  xs.GuardFunc(func(a xs.GuardArgs[any]) bool { return true }),
			}}},
			{Key: "c"},
		},
	})

	actorRef := xs.CreateActor(machine).Start()

	actorRef.Send(xs.Ev("WHATEVER"))

	assert.Equal(t, "c", actorRef.GetSnapshot().Value)
}

// JS: transient states (eventless transitions) > events that trigger eventless transitions should be preserved in guards
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/transient.test.ts#L504
func TestTransient_EventsThatTriggerEventlessTransitionsShouldBePreservedInGuards(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{"EVENT": {{Target: "b"}}}},
			{Key: "b", Always: xs.Transitions{{Target: "c"}}},
			{Key: "c", Always: xs.Transitions{{
				Guard: xs.GuardFunc(func(a xs.GuardArgs[any]) bool {
					assert.Equal(t, "EVENT", a.Event.EventType())
					return a.Event.EventType() == "EVENT"
				}),
				Target: "d",
			}}},
			{Key: "d", Type: xs.Final},
		},
	})

	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.Ev("EVENT"))

	assert.Equal(t, xs.StatusDone, actorRef.GetSnapshot().Status)
}

// JS: transient states (eventless transitions) > events that trigger eventless transitions should be preserved in actions
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/transient.test.ts#L535
func TestTransient_EventsThatTriggerEventlessTransitionsShouldBePreservedInActions(t *testing.T) {
	// expect.assertions(3)
	assertions := 0
	expected := xs.E{"type": "EVENT", "value": 42}

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{"EVENT": {{Target: "b"}}}},
			{
				Key: "b",
				Always: xs.Transitions{{
					Target: "c",
					Actions: xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[any]) {
						assertions++
						assert.Equal(t, expected, a.Event)
					})},
				}},
				Exit: xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[any]) {
					assertions++
					assert.Equal(t, expected, a.Event)
				})},
			},
			{Key: "c", Entry: xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[any]) {
				assertions++
				assert.Equal(t, expected, a.Event)
			})}},
		},
	})

	service := xs.CreateActor(machine).Start()
	service.Send(xs.E{"type": "EVENT", "value": 42})

	assert.Equal(t, 3, assertions)
}

// JS: transient states (eventless transitions) > should avoid infinite loops with eventless transitions
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/transient.test.ts#L569
func TestTransient_ShouldAvoidInfiniteLoopsWithEventlessTransitions(t *testing.T) {
	// expect.assertions(1)
	assertions := 0
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", Always: xs.Transitions{{Target: "b"}}},
			{Key: "b", Always: xs.Transitions{{Target: "c"}}},
			{Key: "c", Always: xs.Transitions{{Target: "a"}}},
		},
	}).WithOptions(xs.MachineOptions{MaxIterations: 100})
	actor := xs.CreateActor(machine)

	actor.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{
		Error: func(err any) {
			assertions++
			msg := fmt.Sprint(err)
			if e, ok := err.(error); ok {
				msg = e.Error()
			}
			assert.Regexp(t, `(?i)infinite loop`, msg)
		},
	})

	actor.Start()

	assert.Equal(t, 1, assertions)
}

// JS: transient states (eventless transitions) > should avoid infinite loops with raised events
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/transient.test.ts#L605
func TestTransient_ShouldAvoidInfiniteLoopsWithRaisedEvents(t *testing.T) {
	// expect.assertions(1)
	assertions := 0
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", Always: xs.Transitions{{Target: "b"}}},
			{
				Key:   "b",
				Entry: xs.Actions{xs.Raise(xs.Ev("EVENT"))},
				On:    map[string]xs.Transitions{"EVENT": {{Target: "c"}}},
			},
			{Key: "c", Always: xs.Transitions{{Target: "a"}}},
		},
	}).WithOptions(xs.MachineOptions{MaxIterations: 100})
	actor := xs.CreateActor(machine)

	actor.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{
		Error: func(err any) {
			assertions++
			msg := fmt.Sprint(err)
			if e, ok := err.(error); ok {
				msg = e.Error()
			}
			assert.Regexp(t, `(?i)infinite loop`, msg)
		},
	})

	actor.Start()

	assert.Equal(t, 1, assertions)
}

// JS: transient states (eventless transitions) > shouldn't end up in an infinite loop when selecting the fallback target
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/transient.test.ts#L644
func TestTransient_ShouldntEndUpInAnInfiniteLoopWhenSelectingTheFallbackTarget(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "idle",
		States: xs.States{
			{Key: "idle", On: map[string]xs.Transitions{"event": {{Target: "active"}}}},
			{
				Key:     "active",
				Initial: "a",
				States:  xs.States{{Key: "a"}, {Key: "b"}},
				Always: xs.Transitions{
					{Guard: xs.GuardFunc(func(a xs.GuardArgs[any]) bool { return false }), Target: ".a"},
					{Target: ".b"},
				},
			},
		},
	})
	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.Ev("event"))

	assert.Equal(t, map[string]any{"active": "b"}, actorRef.GetSnapshot().Value)
}

// JS: transient states (eventless transitions) > shouldn't end up in an infinite loop when selecting a guarded target
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/transient.test.ts#L679
func TestTransient_ShouldntEndUpInAnInfiniteLoopWhenSelectingAGuardedTarget(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "idle",
		States: xs.States{
			{Key: "idle", On: map[string]xs.Transitions{"event": {{Target: "active"}}}},
			{
				Key:     "active",
				Initial: "a",
				States:  xs.States{{Key: "a"}, {Key: "b"}},
				Always: xs.Transitions{
					{Guard: xs.GuardFunc(func(a xs.GuardArgs[any]) bool { return true }), Target: ".a"},
					{Target: ".b"},
				},
			},
		},
	})
	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.Ev("event"))

	assert.Equal(t, map[string]any{"active": "a"}, actorRef.GetSnapshot().Value)
}

// JS: transient states (eventless transitions) > shouldn't end up in an infinite loop when executing a fire-and-forget action that doesn't change state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/transient.test.ts#L714
func TestTransient_ShouldntEndUpInInfiniteLoopWhenExecutingFireAndForgetActionNoStateChange(t *testing.T) {
	count := 0
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "idle",
		States: xs.States{
			{Key: "idle", On: map[string]xs.Transitions{"event": {{Target: "active"}}}},
			{
				Key:     "active",
				Initial: "a",
				States:  xs.States{{Key: "a"}},
				Always: xs.Transitions{{
					Actions: xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[any]) {
						count++
						if count > 5 {
							panic(errors.New("Infinite loop detected"))
						}
					})},
					Target: ".a",
				}},
			},
		},
	})

	actorRef := xs.CreateActor(machine)

	actorRef.Start()
	actorRef.Send(xs.Ev("event"))

	assert.Equal(t, map[string]any{"active": "a"}, actorRef.GetSnapshot().Value)
	assert.Equal(t, 1, count)
}

// JS: transient states (eventless transitions) > should loop (but not infinitely) for assign actions
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/transient.test.ts#L755
func TestTransient_ShouldLoopButNotInfinitelyForAssignActions(t *testing.T) {
	type ctx struct{ Count int }
	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Context: ctx{Count: 0},
		Initial: "counting",
		States: xs.States{
			{Key: "counting", Always: xs.Transitions{{
				Guard: xs.GuardFunc(func(a xs.GuardArgs[ctx]) bool { return a.Context.Count < 5 }),
				Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
					c := a.Context
					c.Count = a.Context.Count + 1
					return c
				})},
			}}},
		},
	})

	actorRef := xs.CreateActor(machine).Start()

	assert.Equal(t, 5, actorRef.GetSnapshot().Context.Count)
}

// JS: transient states (eventless transitions) > should execute an always transition after a raised transition even if that raised transition doesn't change the state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/transient.test.ts#L774
func TestTransient_ShouldExecuteAlwaysTransitionAfterRaisedTransitionEvenIfNoStateChange(t *testing.T) {
	s := newSpy()
	counter := 0
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Always: xs.Transitions{{
			Actions: xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[any]) { s.Call(counter) })},
		}},
		On: map[string]xs.Transitions{
			"EV": {{Actions: xs.Actions{xs.Raise(xs.Ev("RAISED"))}}},
			"RAISED": {{Actions: xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[any]) {
				counter++
			})}}},
		},
	})
	actorRef := xs.CreateActor(machine).Start()
	// spy.mockClear(): only calls recorded after this point are asserted.
	cleared := s.Count()
	actorRef.Send(xs.Ev("EV"))

	assert.Equal(t, [][]any{
		// called in response to the `EV` event
		{0},
		// called in response to the `RAISED` event
		{1},
	}, s.Calls()[cleared:])
}
