package xstate_test

import (
	"encoding/json"
	"sync"
	"testing"
	"time"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// after1LightCtx mirrors the context of the shared top-level `lightMachine`.
type after1LightCtx struct{ CanTurnGreen bool }

// after1LightMachine mirrors the top-level `lightMachine` (JS L4-27). It is a
// constructor instead of a package var so a panicking CreateMachine cannot
// break package initialisation.
func after1LightMachine() *xs.StateMachine[after1LightCtx] {
	return xs.CreateMachine(xs.MachineConfig[after1LightCtx]{
		ID:      "light",
		Initial: "green",
		Context: after1LightCtx{CanTurnGreen: true},
		States: xs.States{
			{Key: "green", After: map[string]xs.Transitions{
				"1000": {{Target: "yellow"}},
			}},
			{Key: "yellow", After: map[string]xs.Transitions{
				"1000": {{Target: "red"}},
			}},
			{Key: "red", After: map[string]xs.Transitions{
				"1000": {{Target: "green"}},
			}},
		},
	})
}

// after1SpyClock mirrors `clock: { setTimeout, clearTimeout: spy }`: real
// timeouts, with clearTimeout replaced by a spy that does nothing else.
type after1SpyClock struct {
	clearSpy *spy
}

func (c *after1SpyClock) SetTimeout(fn func(), d time.Duration) xs.TimerID {
	return time.AfterFunc(d, fn)
}

func (c *after1SpyClock) ClearTimeout(id xs.TimerID) { c.clearSpy.Call(id) }

// JS: delayed transitions > should transition after delay
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/after.test.ts#L34
func TestAfter_DelayedTransitions_ShouldTransitionAfterDelay(t *testing.T) {
	clock := xs.NewSimulatedClock()

	actorRef := xs.CreateActor(after1LightMachine(), xs.WithClock(clock)).Start()
	assert.Equal(t, "green", actorRef.GetSnapshot().Value)

	clock.Increment(ms(500))
	assert.Equal(t, "green", actorRef.GetSnapshot().Value)

	clock.Increment(ms(510))
	assert.Equal(t, "yellow", actorRef.GetSnapshot().Value)
}

// JS: delayed transitions > should not try to clear an undefined timeout when exiting source state of a delayed transition
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/after.test.ts#L47
func TestAfter_DelayedTransitions_ShouldNotClearUndefinedTimeoutWhenExitingSourceStateOfDelayedTransition(t *testing.T) {
	// https://github.com/statelyai/xstate/issues/5001
	s := newSpy()

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "green",
		States: xs.States{
			{Key: "green", After: map[string]xs.Transitions{
				"1": {{Target: "yellow"}},
			}},
			{Key: "yellow"},
		},
	})

	actorRef := xs.CreateActor(machine, xs.WithClock(&after1SpyClock{clearSpy: s})).Start()

	// when the after transition gets executed it tries to clear its own timer when exiting its source state
	sleep(5)
	assert.Equal(t, "yellow", actorRef.GetSnapshot().Value)
	assert.Equal(t, 0, s.Count())
}

// JS: delayed transitions > should format transitions properly
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/after.test.ts#L76
func TestAfter_DelayedTransitions_ShouldFormatTransitionsProperly(t *testing.T) {
	greenNode := after1LightMachine().States["green"]

	// JS reads `[...greenNode.transitions.keys()]`; StateNode.On() is keyed by
	// the same event descriptors (JS `on` is derived from `transitions`).
	transitions := greenNode.On()

	keys := make([]string, 0, len(transitions))
	for k := range transitions {
		keys = append(keys, k)
	}
	assert.Equal(t, []string{"xstate.after.1000.light.green"}, keys)
}

// JS: delayed transitions > should be able to transition with delay from nested initial state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/after.test.ts#L88
func TestAfter_DelayedTransitions_ShouldTransitionWithDelayFromNestedInitialState(t *testing.T) {
	sig := newSignal()

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "nested",
		States: xs.States{
			{
				Key:     "nested",
				Initial: "wait",
				States: xs.States{
					{Key: "wait", After: map[string]xs.Transitions{
						"10": {{Target: "#end"}},
					}},
				},
			},
			{Key: "end", ID: "end", Type: xs.Final},
		},
	})

	actor := xs.CreateActor(machine)
	actor.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{
		Complete: func() { sig.Resolve() },
	})
	actor.Start()

	sig.Wait(t)
}

// JS: delayed transitions > parent state should enter child state without re-entering self (relative target)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/after.test.ts#L122
func TestAfter_DelayedTransitions_ParentStateShouldEnterChildStateWithoutReenteringSelfRelativeTarget(t *testing.T) {
	sig := newSignal()

	var mu sync.Mutex
	actual := []string{}
	push := func(s string) xs.Action {
		return xs.ActionFunc(func(_ xs.ActionArgs[any]) {
			mu.Lock()
			defer mu.Unlock()
			actual = append(actual, s)
		})
	}

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "one",
		States: xs.States{
			{
				Key:     "one",
				Initial: "two",
				Entry:   xs.Actions{push("entered one")},
				States: xs.States{
					{Key: "two", Entry: xs.Actions{push("entered two")}},
					{
						Key:    "three",
						Entry:  xs.Actions{push("entered three")},
						Always: xs.Transitions{{Target: "#end"}},
					},
				},
				After: map[string]xs.Transitions{
					"10": {{Target: ".three"}},
				},
			},
			{Key: "end", ID: "end", Type: xs.Final},
		},
	})

	var atComplete []string
	actor := xs.CreateActor(machine)
	actor.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{
		Complete: func() {
			// JS asserts inside the callback; capture the value seen at
			// completion and assert after the signal on the test goroutine.
			mu.Lock()
			atComplete = append([]string(nil), actual...)
			mu.Unlock()
			sig.Resolve()
		},
	})
	actor.Start()

	sig.Wait(t)
	assert.Equal(t, []string{"entered one", "entered two", "entered three"}, atComplete)
}

// JS: delayed transitions > should defer a single send event for a delayed conditional transition (#886)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/after.test.ts#L165
func TestAfter_DelayedTransitions_ShouldDeferSingleSendEventForDelayedConditionalTransition886(t *testing.T) {
	clock := xs.NewSimulatedClock()
	s := newSpy()

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "X",
		States: xs.States{
			{Key: "X", After: map[string]xs.Transitions{
				"1": {
					{
						Target: "Y",
						Guard:  xs.GuardFunc(func(_ xs.GuardArgs[any]) bool { return true }),
					},
					{Target: "Z"},
				},
			}},
			{Key: "Y", On: map[string]xs.Transitions{
				"*": {{Actions: xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[any]) { s.Call(a) })}}},
			}},
			{Key: "Z"},
		},
	})

	xs.CreateActor(machine, xs.WithClock(clock)).Start()

	clock.Increment(ms(10))
	assert.Equal(t, 0, s.Count())
}

// JS: delayed transitions > should execute an after transition after starting from a state resolved using `.getPersistedSnapshot`
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/after.test.ts#L202
func TestAfter_DelayedTransitions_ShouldExecuteAfterTransitionAfterStartingFromGetPersistedSnapshot(t *testing.T) {
	t.Skip("skipped in JS")

	sig := newSignal()

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "machine",
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{
				"next": {{Target: "withAfter"}},
			}},
			{Key: "withAfter", After: map[string]xs.Transitions{
				"1": {{Target: "done"}},
			}},
			{Key: "done", Type: xs.Final},
		},
	})

	actorRef1 := xs.CreateActor(machine).Start()
	actorRef1.Send(xs.Ev("next"))
	withAfterState := actorRef1.GetPersistedSnapshot()

	actorRef2 := xs.CreateActor(machine, xs.WithSnapshot(withAfterState))
	actorRef2.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{Complete: func() { sig.Resolve() }})
	actorRef2.Start()

	sig.Wait(t)
}

// JS: delayed transitions > should execute an after transition after starting from a persisted state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/after.test.ts#L236
func TestAfter_DelayedTransitions_ShouldExecuteAfterTransitionAfterStartingFromPersistedState(t *testing.T) {
	sig := newSignal()
	createMyMachine := func() *xs.StateMachine[any] {
		return xs.CreateMachine(xs.MachineConfig[any]{
			Initial: "A",
			States: xs.States{
				{Key: "A", On: map[string]xs.Transitions{
					"NEXT": {{Target: "B"}},
				}},
				{Key: "B", After: map[string]xs.Transitions{
					"1": {{Target: "C"}},
				}},
				{Key: "C", Type: xs.Final},
			},
		})
	}

	service := xs.CreateActor(createMyMachine()).Start()

	// JSON.parse(JSON.stringify(service.getSnapshot())): JSON.stringify uses
	// snapshot.toJSON().
	raw, err := json.Marshal(service.GetSnapshot().ToJSON())
	require.NoError(t, err)
	var persistedSnapshot map[string]any
	require.NoError(t, json.Unmarshal(raw, &persistedSnapshot))

	service = xs.CreateActor(createMyMachine(), xs.WithSnapshot(persistedSnapshot)).Start()

	service.Send(xs.Ev("NEXT"))

	service.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{Complete: func() { sig.Resolve() }})

	sig.Wait(t)
}

// JS: delayed transitions > delay expressions > should evaluate the expression (function) to determine the delay
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/after.test.ts#L274
func TestAfter_DelayExpressions_ShouldEvaluateExpressionFunctionToDetermineDelay(t *testing.T) {
	type ctx struct{ Delay int }

	clock := xs.NewSimulatedClock()
	s := newSpy()
	context := ctx{Delay: 500}

	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Initial: "inactive",
		Context: context,
		States: xs.States{
			{Key: "inactive", After: map[string]xs.Transitions{
				"myDelay": {{Target: "active"}},
			}},
			{Key: "active"},
		},
	}, xs.Implementations{
		Delays: map[string]any{
			"myDelay": xs.NewExpr(func(a xs.ExprArgs[ctx]) any {
				s.Call(a.Context)
				return ms(a.Context.Delay)
			}),
		},
	})

	actor := xs.CreateActor(machine, xs.WithClock(clock)).Start()

	assert.Contains(t, s.Calls(), []any{context}) // toBeCalledWith(context)
	assert.Equal(t, "inactive", actor.GetSnapshot().Value)

	clock.Increment(ms(300))
	assert.Equal(t, "inactive", actor.GetSnapshot().Value)

	clock.Increment(ms(200))
	assert.Equal(t, "active", actor.GetSnapshot().Value)
}

// JS: delayed transitions > delay expressions > should evaluate the expression (string) to determine the delay
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/after.test.ts#L313
func TestAfter_DelayExpressions_ShouldEvaluateExpressionStringToDetermineDelay(t *testing.T) {
	clock := xs.NewSimulatedClock()
	s := newSpy()

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "inactive",
		States: xs.States{
			{Key: "inactive", On: map[string]xs.Transitions{
				"ACTIVATE": {{Target: "active"}},
			}},
			{Key: "active", After: map[string]xs.Transitions{
				"someDelay": {{Target: "inactive"}},
			}},
		},
	}, xs.Implementations{
		Delays: map[string]any{
			"someDelay": xs.NewExpr(func(a xs.ExprArgs[any]) any {
				s.Call(a.Event)
				return ms(a.Event.(xs.E)["delay"].(int))
			}),
		},
	})

	actor := xs.CreateActor(machine, xs.WithClock(clock)).Start()

	event := xs.E{"type": "ACTIVATE", "delay": 500}
	actor.Send(event)

	assert.Contains(t, s.Calls(), []any{event}) // toBeCalledWith(event)
	assert.Equal(t, "active", actor.GetSnapshot().Value)

	clock.Increment(ms(300))
	assert.Equal(t, "active", actor.GetSnapshot().Value)

	clock.Increment(ms(200))
	assert.Equal(t, "inactive", actor.GetSnapshot().Value)
}
