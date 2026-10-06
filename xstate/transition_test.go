package xstate_test

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// transition1Params returns the params of a built-in executable action as a
// JS-like object (keys use the JS names: delay, event, id, sendId, targetId,
// value, src, input, systemId).
func transition1Params(a xs.ExecutableAction) map[string]any {
	p, _ := a.Params.(map[string]any)
	return p
}

// transition1ContainsAction mirrors
// expect(actions).toContainEqual(expect.objectContaining({type, params: expect.objectContaining(...)})).
func transition1ContainsAction(actions []xs.ExecutableAction, actionType string, params map[string]any) bool {
	for _, a := range actions {
		if a.Type != actionType {
			continue
		}
		p := transition1Params(a)
		ok := true
		for k, v := range params {
			got, has := p[k]
			if !has || !assert.ObjectsAreEqual(v, got) {
				ok = false
				break
			}
		}
		if ok {
			return true
		}
	}
	return false
}

// transition1Stringify mirrors JSON.stringify(snapshot).
func transition1Stringify(t *testing.T, snap *xs.MachineSnapshot[any]) string {
	b, err := json.Marshal(snap.ToJSON())
	assert.NoError(t, err)
	return string(b)
}

// transition1Parse mirrors JSON.parse(str).
func transition1Parse(t *testing.T, s string) map[string]any {
	var out map[string]any
	assert.NoError(t, json.Unmarshal([]byte(s), &out))
	return out
}

// JS: transition function > should capture actions
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/transition.test.ts#L27
func TestTransition_TransitionFunction_ShouldCaptureActions(t *testing.T) {
	type ctx struct{ Count int }
	actionWithParams := newSpy()
	actionWithDynamicParams := newSpy()
	stringAction := newSpy()

	machine := xs.NewSetup[ctx](xs.Implementations{
		Actions: map[string]xs.Action{
			"actionWithParams": xs.ActionFunc(func(a xs.ActionArgs[ctx]) {
				actionWithParams.Call(a, a.Params)
			}),
			"actionWithDynamicParams": xs.ActionFunc(func(a xs.ActionArgs[ctx]) {
				actionWithDynamicParams.Call(a.Params)
			}),
			"stringAction": xs.ActionFunc(func(a xs.ActionArgs[ctx]) {
				stringAction.Call(a, a.Params)
			}),
		},
	}).CreateMachine(xs.MachineConfig[ctx]{
		Entry: xs.Actions{
			xs.ActionRef{Type: "actionWithParams", Params: map[string]any{"a": 1}},
			xs.ActionRef{Type: "stringAction"},
			xs.Assign(func(a xs.AssignArgs[ctx]) ctx { c := a.Context; c.Count = 100; return c }),
		},
		Context: ctx{Count: 0},
		On: map[string]xs.Transitions{
			"event": {{Actions: xs.Actions{xs.ActionRef{
				Type: "actionWithDynamicParams",
				Params: xs.NewExpr(func(a xs.ExprArgs[ctx]) any {
					return map[string]any{"msg": a.Event.(xs.E)["msg"]}
				}),
			}}}},
		},
	})

	state0, actions0 := xs.InitialTransition(machine)

	assert.Equal(t, 100, state0.Context.Count)
	require.Len(t, actions0, 2)
	assert.Equal(t, "actionWithParams", actions0[0].Type)
	assert.Equal(t, map[string]any{"a": 1}, actions0[0].Params)
	assert.Equal(t, "stringAction", actions0[1].Type)

	assert.Equal(t, 0, actionWithParams.Count())
	assert.Equal(t, 0, stringAction.Count())

	state1, actions1 := xs.Transition(machine, state0, xs.E{"type": "event", "msg": "hello"})

	assert.Equal(t, 100, state1.Context.Count)
	require.Len(t, actions1, 1)
	assert.Equal(t, "actionWithDynamicParams", actions1[0].Type)
	assert.Equal(t, map[string]any{"msg": "hello"}, actions1[0].Params)

	assert.Equal(t, 0, actionWithDynamicParams.Count())
}

// JS: transition function > should not execute a referenced serialized action
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/transition.test.ts#L90
func TestTransition_TransitionFunction_ShouldNotExecuteAReferencedSerializedAction(t *testing.T) {
	type ctx struct{ Count int }
	foo := newSpy()

	machine := xs.NewSetup[ctx](xs.Implementations{
		Actions: map[string]xs.Action{
			"foo": xs.ActionFunc(func(a xs.ActionArgs[ctx]) { foo.Call(a, a.Params) }),
		},
	}).CreateMachine(xs.MachineConfig[ctx]{
		Entry:   xs.Actions{xs.ActionRef{Type: "foo"}},
		Context: ctx{Count: 0},
	})

	_, actions := xs.InitialTransition(machine)
	_ = actions

	assert.Equal(t, 0, foo.Count())
}

// JS: transition function > should capture enqueued actions
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/transition.test.ts#L107
func TestTransition_TransitionFunction_ShouldCaptureEnqueuedActions(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Entry: xs.Actions{
			xs.EnqueueActions(func(x xs.EnqueueArgs[any]) {
				x.Enqueue(xs.ActionRef{Type: "stringAction"})
				x.Enqueue(xs.ActionRef{Type: "objectAction"})
			}),
		},
	})

	_, actions := xs.InitialTransition(machine)

	require.Len(t, actions, 2)
	assert.Equal(t, "stringAction", actions[0].Type)
	assert.Equal(t, "objectAction", actions[1].Type)
}

// JS: transition function > delayed raise actions should be returned
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/transition.test.ts#L125
func TestTransition_TransitionFunction_DelayedRaiseActionsShouldBeReturned(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{
				Key:   "a",
				Entry: xs.Actions{xs.Raise(xs.Ev("NEXT"), xs.SendOptions{Delay: ms(10)})},
				On:    map[string]xs.Transitions{"NEXT": {{Target: "b"}}},
			},
			{Key: "b"},
		},
	})

	state, actions := xs.InitialTransition(machine)

	assert.Equal(t, "a", state.Value)

	require.NotEmpty(t, actions)
	assert.Equal(t, "xstate.raise", actions[0].Type)
	p := transition1Params(actions[0])
	assert.Equal(t, ms(10), p["delay"])
	assert.Equal(t, xs.Ev("NEXT"), p["event"])
}

// JS: transition function > raise actions related to delayed transitions should be returned
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/transition.test.ts#L154
func TestTransition_TransitionFunction_RaiseActionsRelatedToDelayedTransitionsShouldBeReturned(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", After: map[string]xs.Transitions{"10": {{Target: "b"}}}},
			{Key: "b"},
		},
	})

	state, actions := xs.InitialTransition(machine)

	assert.Equal(t, "a", state.Value)

	require.NotEmpty(t, actions)
	assert.Equal(t, "xstate.raise", actions[0].Type)
	p := transition1Params(actions[0])
	assert.Equal(t, ms(10), p["delay"])
	// JS: event: { type: 'xstate.after.10.(machine).a' } — the after event carries only its type.
	ev, ok := p["event"].(xs.Event)
	require.True(t, ok, "params.event must be an event")
	assert.Equal(t, "xstate.after.10.(machine).a", ev.EventType())
	if m, isMap := ev.(xs.E); isMap {
		assert.Equal(t, xs.Ev("xstate.after.10.(machine).a"), m)
	}
}

// JS: transition function > cancel action should be returned
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/transition.test.ts#L180
func TestTransition_TransitionFunction_CancelActionShouldBeReturned(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{
				Key:   "a",
				Entry: xs.Actions{xs.Raise(xs.Ev("NEXT"), xs.SendOptions{Delay: ms(10), ID: "myRaise"})},
				On: map[string]xs.Transitions{
					"NEXT": {{Target: "b", Actions: xs.Actions{xs.Cancel("myRaise")}}},
				},
			},
			{Key: "b"},
		},
	})

	state, _ := xs.InitialTransition(machine)

	assert.Equal(t, "a", state.Value)

	_, actions := xs.Transition(machine, state, xs.Ev("NEXT"))

	assert.True(t, transition1ContainsAction(actions, "xstate.cancel", map[string]any{"sendId": "myRaise"}),
		"expected an xstate.cancel action with sendId 'myRaise', got %#v", actions)
}

// JS: transition function > sendTo action should be returned
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/transition.test.ts#L213
func TestTransition_TransitionFunction_SendToActionShouldBeReturned(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		Invoke: []xs.InvokeConfig{{
			Logic: xs.CreateMachine(xs.MachineConfig[any]{}),
			ID:    "someActor",
		}},
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{
				"NEXT": {{Actions: xs.Actions{xs.SendTo("someActor", xs.Ev("someEvent"))}}},
			}},
		},
	})

	state, actions0 := xs.InitialTransition(machine)

	assert.Equal(t, "a", state.Value)

	assert.True(t, transition1ContainsAction(actions0, "xstate.spawnChild", map[string]any{"id": "someActor"}),
		"expected an xstate.spawnChild action with id 'someActor', got %#v", actions0)

	_, actions := xs.Transition(machine, state, xs.Ev("NEXT"))

	assert.True(t, transition1ContainsAction(actions, "xstate.sendTo", map[string]any{"targetId": "someActor"}),
		"expected an xstate.sendTo action with targetId 'someActor', got %#v", actions)
}

// JS: transition function > emit actions should be returned
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/transition.test.ts#L256
func TestTransition_TransitionFunction_EmitActionsShouldBeReturned(t *testing.T) {
	type ctx struct{ Count int }
	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Initial: "a",
		Context: ctx{Count: 10},
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{
				"NEXT": {{Actions: xs.Actions{xs.Emit(xs.NewExpr(func(a xs.ExprArgs[ctx]) any {
					return xs.E{"type": "counted", "count": a.Context.Count}
				}))}}},
			}},
		},
	})

	state, _ := xs.InitialTransition(machine)

	assert.Equal(t, "a", state.Value)

	_, nextActions := xs.Transition(machine, state, xs.Ev("NEXT"))

	assert.True(t, transition1ContainsAction(nextActions, "xstate.emit", map[string]any{
		"event": xs.E{"type": "counted", "count": 10},
	}), "expected an xstate.emit action with event {type: counted, count: 10}, got %#v", nextActions)
}

// JS: transition function > log actions should be returned
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/transition.test.ts#L290
func TestTransition_TransitionFunction_LogActionsShouldBeReturned(t *testing.T) {
	type ctx struct{ Count int }
	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Initial: "a",
		Context: ctx{Count: 10},
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{
				"NEXT": {{Actions: xs.Actions{xs.Log(xs.NewExpr(func(a xs.ExprArgs[ctx]) any {
					return fmt.Sprintf("count: %d", a.Context.Count)
				}))}}},
			}},
		},
	})

	state, _ := xs.InitialTransition(machine)

	assert.Equal(t, "a", state.Value)

	_, nextActions := xs.Transition(machine, state, xs.Ev("NEXT"))

	assert.True(t, transition1ContainsAction(nextActions, "xstate.log", map[string]any{"value": "count: 10"}),
		"expected an xstate.log action with value 'count: 10', got %#v", nextActions)
}

// JS: transition function > should calculate the next snapshot for transition logic
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/transition.test.ts#L321
func TestTransition_TransitionFunction_ShouldCalculateTheNextSnapshotForTransitionLogic(t *testing.T) {
	type state struct{ Count int }
	logic := xs.FromTransition(
		func(s state, e xs.Event, _ *xs.ActorScope) state {
			if e.EventType() == "next" {
				return state{Count: s.Count + 1}
			}
			return s
		},
		func(xs.TransitionInitArgs) state { return state{Count: 0} },
	)

	init, _ := xs.InitialTransitionActorLogic(logic)
	s1, _ := xs.TransitionActorLogic(logic, init, xs.Ev("next"))
	assert.Equal(t, 1, s1.Context.Count)
	s2, _ := xs.TransitionActorLogic(logic, s1, xs.Ev("next"))
	assert.Equal(t, 2, s2.Context.Count)
}

// JS: transition function > should calculate the next snapshot for machine logic
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/transition.test.ts#L340
func TestTransition_TransitionFunction_ShouldCalculateTheNextSnapshotForMachineLogic(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{"NEXT": {{Target: "b"}}}},
			{Key: "b", On: map[string]xs.Transitions{"NEXT": {{Target: "c"}}}},
			{Key: "c"},
		},
	})

	init, _ := xs.InitialTransition(machine)
	s1, _ := xs.Transition(machine, init, xs.Ev("NEXT"))

	assert.Equal(t, "b", s1.Value)

	s2, _ := xs.Transition(machine, s1, xs.Ev("NEXT"))

	assert.Equal(t, "c", s2.Value)
}

// JS: transition function > should not execute entry actions
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/transition.test.ts#L368
func TestTransition_TransitionFunction_ShouldNotExecuteEntryActions(t *testing.T) {
	fn := newSpy()

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		Entry:   xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[any]) { fn.Call(a, a.Params) })},
		States: xs.States{
			{Key: "a"},
			{Key: "b"},
		},
	})

	xs.InitialTransition(machine)

	assert.Equal(t, 0, fn.Count())
}

// JS: transition function > should not execute transition actions
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/transition.test.ts#L385
func TestTransition_TransitionFunction_ShouldNotExecuteTransitionActions(t *testing.T) {
	fn := newSpy()

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{
				"event": {{
					Target:  "b",
					Actions: xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[any]) { fn.Call(a, a.Params) })},
				}},
			}},
			{Key: "b"},
		},
	})

	init, _ := xs.InitialTransition(machine)
	nextSnapshot, _ := xs.Transition(machine, init, xs.Ev("event"))

	assert.Equal(t, 0, fn.Count())
	assert.Equal(t, "b", nextSnapshot.Value)
}

// JS: transition function > delayed events example (experimental)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/transition.test.ts#L410
func TestTransition_TransitionFunction_DelayedEventsExampleExperimental(t *testing.T) {
	var dbMu sync.Mutex
	var dbState string
	completed := make(chan struct{})

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "start",
		States: xs.States{
			{Key: "start", On: map[string]xs.Transitions{"next": {{Target: "waiting"}}}},
			{Key: "waiting", After: map[string]xs.Transitions{"10": {{Target: "done"}}}},
			{Key: "done", Type: xs.Final},
		},
	})

	var postEvent func(event xs.Event)

	// execute blocks for the remaining delay (JS: `await new Promise(setTimeout)`).
	execute := func(action xs.ExecutableAction) {
		p := transition1Params(action)
		delay, _ := p["delay"].(time.Duration)
		if action.Type == "xstate.raise" && delay != 0 {
			currentTime := time.Now()
			startedAt := currentTime
			elapsed := currentTime.Sub(startedAt)
			timeRemaining := delay - elapsed
			if timeRemaining < 0 {
				timeRemaining = 0
			}

			time.Sleep(timeRemaining)
			postEvent(p["event"].(xs.Event))
		}
	}

	// POST /workflow
	postStart := func() {
		state, actions := xs.InitialTransition(machine)

		dbMu.Lock()
		dbState = transition1Stringify(t, state)
		dbMu.Unlock()

		// execute actions
		for _, action := range actions {
			execute(action)
		}
	}

	// POST /workflow/{sessionId}
	// JS calls postEvent without awaiting: the synchronous part (transition +
	// db write) runs immediately, the awaited action execution continues
	// asynchronously.
	postEvent = func(event xs.Event) {
		dbMu.Lock()
		parsed := transition1Parse(t, dbState)
		nextState, actions := xs.Transition(
			machine,
			machine.ResolveState(xs.ResolveStateConfig[any]{Value: parsed["value"], Context: parsed["context"]}),
			event,
		)
		dbState = transition1Stringify(t, nextState)
		dbMu.Unlock()
		if nextState.Status == xs.StatusDone {
			close(completed)
		}

		go func() {
			for _, action := range actions {
				execute(action)
			}
		}()
	}

	postStart()
	postEvent(xs.Ev("next"))

	select {
	case <-completed:
	case <-time.After(3 * time.Second):
		t.Fatal("delayed workflow did not complete")
	}
	dbMu.Lock()
	final := transition1Parse(t, dbState)
	dbMu.Unlock()
	assert.Equal(t, "done", final["status"])
}

// JS: transition function > serverless workflow example (experimental)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/transition.test.ts#L480
func TestTransition_TransitionFunction_ServerlessWorkflowExampleExperimental(t *testing.T) {
	var dbMu sync.Mutex
	var dbState string

	var callsMu sync.Mutex
	calls := []string{}

	machine := xs.NewSetup[any](xs.Implementations{
		Actors: map[string]xs.ActorLogic{
			"sendWelcomeEmail": xs.FromPromise(func(_ context.Context, _ xs.PromiseArgs) (any, error) {
				callsMu.Lock()
				calls = append(calls, "sendWelcomeEmail")
				callsMu.Unlock()
				return map[string]any{"status": "sent"}, nil
			}),
		},
	}).CreateMachine(xs.MachineConfig[any]{
		Initial: "sendingWelcomeEmail",
		States: xs.States{
			{Key: "sendingWelcomeEmail", Invoke: []xs.InvokeConfig{{
				Src: "sendWelcomeEmail",
				Input: xs.NewExpr(func(xs.ExprArgs[any]) any {
					return map[string]any{"message": "hello world", "subject": "hi"}
				}),
				OnDone: xs.Transitions{{Target: "logSent"}},
			}}},
			{Key: "logSent", Invoke: []xs.InvokeConfig{{
				Logic:  xs.FromPromise(func(context.Context, xs.PromiseArgs) (any, error) { return nil, nil }),
				OnDone: xs.Transitions{{Target: "finish"}},
			}}},
			{Key: "finish"},
		},
	})

	var postEvent func(event xs.Event)

	execute := func(action xs.ExecutableAction) {
		switch action.Type {
		case "xstate.spawnChild":
			p := transition1Params(action)
			var logic xs.ActorLogic
			if src, isString := p["src"].(string); isString {
				logic = xs.ResolveReferencedActor(machine, src)
			} else {
				logic, _ = p["src"].(xs.ActorLogic)
			}
			// JS: assert('transition' in logic) — both actors here are fromPromise logic.
			typed, ok := logic.(xs.TypedActorLogic[*xs.PromiseSnapshot[any]])
			if !assert.True(t, ok, "spawned logic must be actor logic, got %#v", logic) {
				return
			}
			id, _ := p["id"].(string)
			opts := []xs.ActorOption{xs.WithID(id), xs.WithInput(p["input"])}
			if systemID, _ := p["systemId"].(string); systemID != "" {
				opts = append(opts, xs.WithSystemID(systemID))
			}
			output, err := xs.ToPromise(xs.CreateActor(typed, opts...).Start()).Wait()
			assert.NoError(t, err)
			postEvent(xs.DoneActorEvent{ActorID: id, Output: output})
		default:
		}
	}

	// POST /workflow
	postStart := func() {
		state, actions := xs.InitialTransition(machine)

		dbMu.Lock()
		dbState = transition1Stringify(t, state)
		dbMu.Unlock()

		// execute actions
		for _, action := range actions {
			execute(action)
		}
	}

	// POST /workflow/{sessionId}
	// JS calls postEvent without awaiting: the synchronous part (transition +
	// db write) runs immediately, the awaited action execution continues
	// asynchronously.
	postEvent = func(event xs.Event) {
		dbMu.Lock()
		parsed := transition1Parse(t, dbState)
		nextState, actions := xs.Transition(
			machine,
			machine.ResolveState(xs.ResolveStateConfig[any]{Value: parsed["value"], Context: parsed["context"]}),
			event,
		)
		dbState = transition1Stringify(t, nextState)
		dbMu.Unlock()

		// "sync" built-in actions: assign, raise, cancel, stop
		// "external" built-in actions: sendTo, raise w/delay, log
		go func() {
			for _, action := range actions {
				execute(action)
			}
		}()
	}

	postStart()
	postEvent(xs.Ev("sent"))

	callsMu.Lock()
	assert.Equal(t, []string{"sendWelcomeEmail"}, calls)
	callsMu.Unlock()

	sleep(10)
	dbMu.Lock()
	final := transition1Parse(t, dbState)
	dbMu.Unlock()
	assert.Equal(t, "finish", final["value"])
}

// transition1EventTypes mirrors transitions.map((t) => t.eventType).
func transition1EventTypes(ts []*xs.TransitionDefinition) []string {
	out := make([]string, len(ts))
	for i, tr := range ts {
		out[i] = tr.EventType
	}
	return out
}

// transition1TargetKeys mirrors transitions.map((t) => t.target?.[0]?.key).
// A missing target yields "" (JS undefined).
func transition1TargetKeys(ts []*xs.TransitionDefinition) []string {
	out := make([]string, len(ts))
	for i, tr := range ts {
		if len(tr.Target) > 0 && tr.Target[0] != nil {
			out[i] = tr.Target[0].Key
		}
	}
	return out
}

// JS: getNextTransitions > should return all transitions from current state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/transition.test.ts#L575
func TestTransition_GetNextTransitions_ShouldReturnAllTransitionsFromCurrentState(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{
				"GO_B": {{Target: "b"}},
				"GO_C": {{Target: "c"}},
			}},
			{Key: "b"},
			{Key: "c"},
		},
	})

	actor := xs.CreateActor(machine)
	actor.Start()
	state := actor.GetSnapshot()

	transitions := xs.GetNextTransitions(state)

	assert.Len(t, transitions, 2)
	// Order should be deterministic: transitions appear in the order they're defined
	assert.Equal(t, []string{"GO_B", "GO_C"}, transition1EventTypes(transitions))
}

// JS: getNextTransitions > should include guarded transitions regardless of guard result
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/transition.test.ts#L601
func TestTransition_GetNextTransitions_ShouldIncludeGuardedTransitionsRegardlessOfGuardResult(t *testing.T) {
	type ctx struct{ Count int }
	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Initial: "a",
		Context: ctx{Count: 100},
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{
				"GO_B": {
					{
						Guard:  xs.GuardFunc(func(a xs.GuardArgs[ctx]) bool { return a.Context.Count < 10 }),
						Target: "b",
					},
					{Target: "d"},
				},
				"GO_C": {{
					Guard:  xs.GuardFunc(func(a xs.GuardArgs[ctx]) bool { return a.Context.Count > 50 }),
					Target: "c",
				}},
			}},
			{Key: "b"},
			{Key: "c"},
			{Key: "d"},
		},
	})

	actor := xs.CreateActor(machine)
	actor.Start()
	state := actor.GetSnapshot()

	transitions := xs.GetNextTransitions(state)

	assert.Len(t, transitions, 3)
	// Order should be deterministic: all GO_B transitions first (in order), then GO_C
	assert.Equal(t, []string{"GO_B", "GO_B", "GO_C"}, transition1EventTypes(transitions))
	// Verify targets match the order
	assert.Equal(t, []string{"b", "d", "c"}, transition1TargetKeys(transitions))
}

// JS: getNextTransitions > should include always (eventless) transitions
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/transition.test.ts#L646
func TestTransition_GetNextTransitions_ShouldIncludeAlwaysEventlessTransitions(t *testing.T) {
	type ctx struct{ Count int }
	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Initial: "a",
		Context: ctx{Count: 5},
		States: xs.States{
			{
				Key: "a",
				Always: xs.Transitions{
					{Guard: xs.GuardFunc(func(a xs.GuardArgs[ctx]) bool { return a.Context.Count > 10 }), Target: "b"},
					{Guard: xs.GuardFunc(func(xs.GuardArgs[ctx]) bool { return false }), Target: "c"},
				},
				On: map[string]xs.Transitions{"GO_D": {{Target: "d"}}},
			},
			{Key: "b"},
			{Key: "c"},
			{Key: "d"},
		},
	})

	actor := xs.CreateActor(machine)
	actor.Start()
	state := actor.GetSnapshot()

	transitions := xs.GetNextTransitions(state)

	assert.Len(t, transitions, 3)
	// Order: on transitions first, then always transitions (in order they appear)
	assert.Equal(t, []string{"GO_D", "", ""}, transition1EventTypes(transitions))
	assert.Equal(t, []string{"d", "b", "c"}, transition1TargetKeys(transitions))
}

// JS: getNextTransitions > should include after (delayed) transitions
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/transition.test.ts#L678
func TestTransition_GetNextTransitions_ShouldIncludeAfterDelayedTransitions(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{
				Key:   "a",
				After: map[string]xs.Transitions{"1000": {{Target: "b"}}},
				On:    map[string]xs.Transitions{"GO_C": {{Target: "c"}}},
			},
			{Key: "b"},
			{Key: "c"},
		},
	})

	actor := xs.CreateActor(machine)
	actor.Start()
	state := actor.GetSnapshot()

	transitions := xs.GetNextTransitions(state)

	assert.Len(t, transitions, 2)
	// Order: on transitions first (in definition order), then after transitions
	assert.Equal(t, []string{"GO_C", "xstate.after.1000.(machine).a"}, transition1EventTypes(transitions))
	assert.Equal(t, []string{"c", "b"}, transition1TargetKeys(transitions))
}

// JS: getNextTransitions > should include transitions from parent states in depth-first order
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/transition.test.ts#L710
func TestTransition_GetNextTransitions_ShouldIncludeTransitionsFromParentStatesInDepthFirstOrder(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "parent",
		States: xs.States{
			{
				Key:     "parent",
				Initial: "child",
				On:      map[string]xs.Transitions{"PARENT_EVENT": {{Target: "other"}}},
				States: xs.States{
					{Key: "child", On: map[string]xs.Transitions{"CHILD_EVENT": {{Target: "sibling"}}}},
					{Key: "sibling"},
				},
			},
			{Key: "other"},
		},
	})

	actor := xs.CreateActor(machine)
	actor.Start()
	state := actor.GetSnapshot()

	transitions := xs.GetNextTransitions(state)

	// Order: child state transitions first, then parent state transitions
	assert.Equal(t, []string{"CHILD_EVENT", "PARENT_EVENT"}, transition1EventTypes(transitions))
}

// JS: getNextTransitions > should include all guarded transitions from different state nodes with same event type
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/transition.test.ts#L745
func TestTransition_GetNextTransitions_ShouldIncludeAllGuardedTransitionsFromDifferentStateNodesWithSameEventType(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "parent",
		States: xs.States{
			{
				Key:     "parent",
				Initial: "child",
				On: map[string]xs.Transitions{
					"SAME_EVENT": {
						{
							Guard:  xs.GuardFunc(func(xs.GuardArgs[any]) bool { return false }),
							Target: "parentTarget",
						},
						{Target: "parentTarget2"},
					},
				},
				States: xs.States{
					{Key: "child", On: map[string]xs.Transitions{
						"SAME_EVENT": {{Target: "childTarget"}},
					}},
					{Key: "childTarget"},
				},
			},
			{Key: "parentTarget"},
			{Key: "parentTarget2"},
		},
	})

	actor := xs.CreateActor(machine)
	actor.Start()
	state := actor.GetSnapshot()

	transitions := xs.GetNextTransitions(state)

	// Should include all transitions: 2 from parent, 1 from child = 3 total
	assert.Len(t, transitions, 3)
	var sameEventTransitions []*xs.TransitionDefinition
	for _, tr := range transitions {
		if tr.EventType == "SAME_EVENT" {
			sameEventTransitions = append(sameEventTransitions, tr)
		}
	}
	require.Len(t, sameEventTransitions, 3)
	// Order: child state transitions first, then parent state transitions
	keys := transition1TargetKeys(sameEventTransitions)
	assert.Equal(t, "childTarget", keys[0])
	assert.Equal(t, "parentTarget", keys[1])
	assert.Equal(t, "parentTarget2", keys[2])
}

// JS: getNextTransitions > should return transitions from parallel states in document order
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/transition.test.ts#L796
func TestTransition_GetNextTransitions_ShouldReturnTransitionsFromParallelStatesInDocumentOrder(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Type: xs.Parallel,
		States: xs.States{
			{
				Key:     "regionA",
				Initial: "a1",
				On:      map[string]xs.Transitions{"REGION_A_EVENT": {{Target: ".a2"}}},
				States: xs.States{
					{Key: "a1", On: map[string]xs.Transitions{"A1_EVENT": {{Target: "a2"}}}},
					{Key: "a2"},
				},
			},
			{
				Key:     "regionB",
				Initial: "b1",
				On:      map[string]xs.Transitions{"REGION_B_EVENT": {{Target: ".b2"}}},
				States: xs.States{
					{Key: "b1", On: map[string]xs.Transitions{"B1_EVENT": {{Target: "b2"}}}},
					{Key: "b2"},
				},
			},
		},
	})

	actor := xs.CreateActor(machine)
	actor.Start()
	state := actor.GetSnapshot()

	transitions := xs.GetNextTransitions(state)

	// Order: regionA atomic state first (depth-first), then regionB atomic state
	// Within each: child transitions first, then parent transitions
	assert.Equal(t, []string{
		"A1_EVENT",       // regionA.a1 (atomic)
		"REGION_A_EVENT", // regionA (parent)
		"B1_EVENT",       // regionB.b1 (atomic)
		"REGION_B_EVENT", // regionB (parent)
	}, transition1EventTypes(transitions))
}

// JS: getNextTransitions > should return transitions from deeply nested compound states in depth-first order
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/transition.test.ts#L847
func TestTransition_GetNextTransitions_ShouldReturnTransitionsFromDeeplyNestedCompoundStatesInDepthFirstOrder(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "level1",
		On:      map[string]xs.Transitions{"ROOT_EVENT": {{Target: ".level1"}}},
		States: xs.States{
			{
				Key:     "level1",
				Initial: "level2",
				On:      map[string]xs.Transitions{"LEVEL1_EVENT": {{Target: ".level2"}}},
				States: xs.States{
					{
						Key:     "level2",
						Initial: "level3",
						On:      map[string]xs.Transitions{"LEVEL2_EVENT": {{Target: ".level3"}}},
						States: xs.States{
							{Key: "level3", On: map[string]xs.Transitions{"LEVEL3_EVENT": {{Target: "level3"}}}},
						},
					},
				},
			},
		},
	})

	actor := xs.CreateActor(machine)
	actor.Start()
	state := actor.GetSnapshot()

	transitions := xs.GetNextTransitions(state)

	// Order: deepest state first, then ancestors up to root
	assert.Equal(t, []string{
		"LEVEL3_EVENT", // level3 (atomic, deepest)
		"LEVEL2_EVENT", // level2 (parent of level3)
		"LEVEL1_EVENT", // level1 (grandparent)
		"ROOT_EVENT",   // root (great-grandparent)
	}, transition1EventTypes(transitions))
}

// JS: getNextTransitions > should return transitions from parallel states with nested compound states
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/transition.test.ts#L893
func TestTransition_GetNextTransitions_ShouldReturnTransitionsFromParallelStatesWithNestedCompoundStates(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Type: xs.Parallel,
		On:   map[string]xs.Transitions{"ROOT_EVENT": {{}}},
		States: xs.States{
			{
				Key:     "regionA",
				Initial: "nested",
				On:      map[string]xs.Transitions{"REGION_A_EVENT": {{Target: ".nested"}}},
				States: xs.States{
					{
						Key:     "nested",
						Initial: "deep",
						On:      map[string]xs.Transitions{"NESTED_A_EVENT": {{Target: ".deep"}}},
						States: xs.States{
							{Key: "deep", On: map[string]xs.Transitions{"DEEP_A_EVENT": {{Target: "deep"}}}},
						},
					},
				},
			},
			{
				Key:     "regionB",
				Initial: "leaf",
				On:      map[string]xs.Transitions{"REGION_B_EVENT": {{Target: ".leaf"}}},
				States: xs.States{
					{Key: "leaf", On: map[string]xs.Transitions{"LEAF_B_EVENT": {{Target: "leaf"}}}},
				},
			},
		},
	})

	actor := xs.CreateActor(machine)
	actor.Start()
	state := actor.GetSnapshot()

	transitions := xs.GetNextTransitions(state)

	// Order: regionA's atomic state (depth-first up to regionA),
	// then regionB's atomic state (depth-first up to regionB),
	// then root
	assert.Equal(t, []string{
		"DEEP_A_EVENT",   // regionA.nested.deep (atomic)
		"NESTED_A_EVENT", // regionA.nested
		"REGION_A_EVENT", // regionA
		"ROOT_EVENT",     // root
		"LEAF_B_EVENT",   // regionB.leaf (atomic)
		"REGION_B_EVENT", // regionB
	}, transition1EventTypes(transitions))
}
