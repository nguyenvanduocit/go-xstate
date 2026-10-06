package xstate_test

import (
	"context"
	"errors"
	"testing"
	"time"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// emit1Receive mirrors `await new Promise((res) => actor.on('emitted', res))`:
// it returns the first event delivered to ch, failing the test after 2s.
func emit1Receive(t *testing.T, ch <-chan xs.Event) xs.Event {
	t.Helper()
	select {
	case ev := <-ch:
		return ev
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for emitted event")
		return nil
	}
}

// emit1AssertCalledWithContaining mirrors
// `expect(spy).toHaveBeenCalledWith(expect.objectContaining(want))`: some
// recorded call's single argument is an xs.E holding every key/value of want.
func emit1AssertCalledWithContaining(t *testing.T, s *spy, want xs.E) {
	t.Helper()
	for _, call := range s.Calls() {
		if len(call) != 1 {
			continue
		}
		got, ok := call[0].(xs.E)
		if !ok {
			continue
		}
		match := true
		for k, v := range want {
			if gv, has := got[k]; !has || gv != v {
				match = false
				break
			}
		}
		if match {
			return
		}
	}
	t.Errorf("expected spy to have been called with an event containing %v; calls: %v", want, s.Calls())
}

// JS: event emitter > only emits expected events if specified in setup
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/emit.test.ts#L27
func TestEmit_OnlyEmitsExpectedEventsIfSpecifiedInSetup(t *testing.T) {
	t.Skip("N/A: type-level only — `@ts-expect-error` checks that emit() only accepts event types declared in setup({ types: { emitted } })")
}

// JS: event emitter > emits any events if not specified in setup (unsafe)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/emit.test.ts#L46
func TestEmit_EmitsAnyEventsIfNotSpecifiedInSetupUnsafe(t *testing.T) {
	assert.NotPanics(t, func() {
		xs.CreateMachine(xs.MachineConfig[any]{
			Entry: xs.Actions{xs.Emit(xs.Ev("nonsense"))},
			Exit:  xs.Actions{xs.Emit(xs.E{"type": "greet", "message": 1234})},
			On: map[string]xs.Transitions{
				"someEvent": {{Actions: xs.Actions{xs.Emit(xs.E{"type": "greet", "message": "hello"})}}},
			},
		})
	})
}

// JS: event emitter > emits events that can be listened to on actorRef.on(…)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/emit.test.ts#L58
func TestEmit_EmitsEventsThatCanBeListenedToOnActorRefOn(t *testing.T) {
	machine := xs.NewSetup[any](xs.Implementations{}).CreateMachine(xs.MachineConfig[any]{
		On: map[string]xs.Transitions{
			"someEvent": {{Actions: xs.Actions{xs.Emit(xs.E{"type": "emitted", "foo": "bar"})}}},
		},
	})

	actor := xs.CreateActor(machine).Start()
	received := make(chan xs.Event, 1)
	actor.On("emitted", func(e xs.Event) {
		select {
		case received <- e:
		default:
		}
	})
	time.AfterFunc(0, func() {
		actor.Send(xs.Ev("someEvent"))
	})
	event := emit1Receive(t, received)

	assert.Equal(t, "bar", event.(xs.E)["foo"])
}

// JS: event emitter > enqueue.emit(…) emits events that can be listened to on actorRef.on(…)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/emit.test.ts#L84
func TestEmit_EnqueueEmitEmitsEventsThatCanBeListenedToOnActorRefOn(t *testing.T) {
	machine := xs.NewSetup[any](xs.Implementations{}).CreateMachine(xs.MachineConfig[any]{
		On: map[string]xs.Transitions{
			"someEvent": {{Actions: xs.Actions{xs.EnqueueActions(func(a xs.EnqueueArgs[any]) {
				a.Enqueue(xs.Emit(xs.E{"type": "emitted", "foo": "bar"}))

				a.Enqueue(xs.Emit(xs.Ev("unknown")))
			})}}},
		},
	})

	actor := xs.CreateActor(machine).Start()
	received := make(chan xs.Event, 1)
	actor.On("emitted", func(e xs.Event) {
		select {
		case received <- e:
		default:
		}
	})
	time.AfterFunc(0, func() {
		actor.Send(xs.Ev("someEvent"))
	})
	event := emit1Receive(t, received)

	assert.Equal(t, "bar", event.(xs.E)["foo"])
}

// JS: event emitter > handles errors
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/emit.test.ts#L117
func TestEmit_HandlesErrors(t *testing.T) {
	machine := xs.NewSetup[any](xs.Implementations{}).CreateMachine(xs.MachineConfig[any]{
		On: map[string]xs.Transitions{
			"someEvent": {{Actions: xs.Actions{xs.Emit(xs.E{"type": "emitted", "foo": "bar"})}}},
		},
	})

	// JS mocks reportUnhandledError to console.error; the handler stands in for that mock.
	unhandled := newSpy()
	actor := xs.CreateActor(machine, xs.WithUnhandledErrorHandler(func(err any) { unhandled.Call(err) })).Start()
	actor.On("emitted", func(xs.Event) {
		panic(errors.New("oops"))
	})
	time.AfterFunc(0, func() {
		actor.Send(xs.Ev("someEvent"))
	})

	sleep(10)

	assert.Equal(t, xs.StatusActive, actor.GetSnapshot().Status)
}

// JS: event emitter > dynamically emits events that can be listened to on actorRef.on(…)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/emit.test.ts#L145
func TestEmit_DynamicallyEmitsEventsThatCanBeListenedToOnActorRefOn(t *testing.T) {
	type ctx struct{ Count int }

	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Context: ctx{Count: 10},
		On: map[string]xs.Transitions{
			"someEvent": {{Actions: xs.Actions{xs.Emit(xs.NewExpr(func(a xs.ExprArgs[ctx]) any {
				return xs.E{"type": "emitted", "count": a.Context.Count}
			}))}}},
		},
	})

	actor := xs.CreateActor(machine).Start()
	received := make(chan xs.Event, 1)
	actor.On("emitted", func(e xs.Event) {
		select {
		case received <- e:
		default:
		}
	})
	time.AfterFunc(0, func() {
		actor.Send(xs.Ev("someEvent"))
	})
	event := emit1Receive(t, received)

	assert.Equal(t, xs.E{"type": "emitted", "count": 10}, event)
}

// JS: event emitter > listener should be able to read the updated snapshot of the emitting actor
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/emit.test.ts#L174
func TestEmit_ListenerShouldBeAbleToReadUpdatedSnapshotOfEmittingActor(t *testing.T) {
	s := newSpy()

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{
				"ev": {{Actions: xs.Actions{xs.Emit(xs.Ev("someEvent"))}, Target: "b"}},
			}},
			{Key: "b"},
		},
	})

	actor := xs.CreateActor(machine)
	actor.On("someEvent", func(xs.Event) {
		s.Call(actor.GetSnapshot().Value)
	})

	actor.Start()
	actor.Send(xs.Ev("ev"))

	assert.Equal(t, 1, s.Count())
	assert.Contains(t, s.Calls(), []any{"b"})
}

// JS: event emitter > wildcard listeners should be able to receive all emitted events
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/emit.test.ts#L204
func TestEmit_WildcardListenersShouldBeAbleToReceiveAllEmittedEvents(t *testing.T) {
	s := newSpy()

	machine := xs.NewSetup[any](xs.Implementations{}).CreateMachine(xs.MachineConfig[any]{
		On: map[string]xs.Transitions{
			"event": {{Actions: xs.Actions{xs.Emit(xs.Ev("emitted"))}}},
		},
	})

	actor := xs.CreateActor(machine)

	actor.On("*", func(ev xs.Event) {
		s.Call(ev)
	})

	actor.Start()

	actor.Send(xs.Ev("event"))

	assert.Equal(t, 1, s.Count())
}

// JS: event emitter > events can be emitted from promise logic
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/emit.test.ts#L237
func TestEmit_EventsCanBeEmittedFromPromiseLogic(t *testing.T) {
	s := newSpy()
	called := newSignal()

	logic := xs.FromPromise(func(_ context.Context, a xs.PromiseArgs) (any, error) {
		a.Emit(xs.E{"type": "emitted", "msg": "hello"})
		return nil, nil
	})

	actor := xs.CreateActor(logic)

	actor.On("emitted", func(ev xs.Event) {
		s.Call(ev)
		called.Resolve()
	})

	actor.Start()

	// FromPromise runs fn on its own goroutine; JS calls emit synchronously
	// during start(). Wait for the listener before asserting.
	called.Wait(t)
	emit1AssertCalledWithContaining(t, s, xs.E{"type": "emitted", "msg": "hello"})
}

// JS: event emitter > events can be emitted from transition logic
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/emit.test.ts#L272
func TestEmit_EventsCanBeEmittedFromTransitionLogic(t *testing.T) {
	s := newSpy()

	logic := xs.FromTransition(func(st map[string]any, e xs.Event, scope *xs.ActorScope) map[string]any {
		if e.EventType() == "emit" {
			scope.Emit(xs.E{"type": "emitted", "msg": "hello"})
		}
		return st
	}, func(xs.TransitionInitArgs) map[string]any { return map[string]any{} })

	actor := xs.CreateActor(logic)

	actor.On("emitted", func(ev xs.Event) {
		s.Call(ev)
	})

	actor.Start()

	actor.Send(xs.Ev("emit"))

	emit1AssertCalledWithContaining(t, s, xs.E{"type": "emitted", "msg": "hello"})
}

// JS: event emitter > events can be emitted from observable logic
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/emit.test.ts#L316
func TestEmit_EventsCanBeEmittedFromObservableLogic(t *testing.T) {
	s := newSpy()

	logic := xs.FromObservable(func(a xs.ObservableArgs) xs.Subscribable[any] {
		a.Emit(xs.E{"type": "emitted", "msg": "hello"})

		return &observable[any]{subscribe: func(xs.Observer[any]) func() {
			return func() {}
		}}
	})

	actor := xs.CreateActor(logic)

	actor.On("emitted", func(ev xs.Event) {
		s.Call(ev)
	})

	actor.Start()

	emit1AssertCalledWithContaining(t, s, xs.E{"type": "emitted", "msg": "hello"})
}

// JS: event emitter > events can be emitted from event observable logic
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/emit.test.ts#L359
func TestEmit_EventsCanBeEmittedFromEventObservableLogic(t *testing.T) {
	s := newSpy()

	logic := xs.FromEventObservable(func(a xs.ObservableArgs) xs.Subscribable[xs.Event] {
		a.Emit(xs.E{"type": "emitted", "msg": "hello"})

		return &observable[xs.Event]{subscribe: func(xs.Observer[xs.Event]) func() {
			return func() {}
		}}
	})

	actor := xs.CreateActor(logic)

	actor.On("emitted", func(ev xs.Event) {
		s.Call(ev)
	})

	actor.Start()

	emit1AssertCalledWithContaining(t, s, xs.E{"type": "emitted", "msg": "hello"})
}

// JS: event emitter > events can be emitted from callback logic
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/emit.test.ts#L404
func TestEmit_EventsCanBeEmittedFromCallbackLogic(t *testing.T) {
	s := newSpy()

	logic := xs.FromCallback(func(a xs.CallbackArgs) func() {
		a.Emit(xs.E{"type": "emitted", "msg": "hello"})
		return nil
	})

	actor := xs.CreateActor(logic)

	actor.On("emitted", func(ev xs.Event) {
		s.Call(ev)
	})

	actor.Start()

	emit1AssertCalledWithContaining(t, s, xs.E{"type": "emitted", "msg": "hello"})
}

// JS: event emitter > events can be emitted from callback logic (restored root)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/emit.test.ts#L439
func TestEmit_EventsCanBeEmittedFromCallbackLogicRestoredRoot(t *testing.T) {
	s := newSpy()

	logic := xs.FromCallback(func(a xs.CallbackArgs) func() {
		a.Emit(xs.E{"type": "emitted", "msg": "hello"})
		return nil
	})

	machine := xs.NewSetup[any](xs.Implementations{
		Actors: map[string]xs.ActorLogic{"logic": logic},
	}).CreateMachine(xs.MachineConfig[any]{
		Invoke: []xs.InvokeConfig{{ID: "cb", Src: "logic"}},
	})

	actor := xs.CreateActor(machine)

	// Persist the root actor
	persistedSnapshot := actor.GetPersistedSnapshot()

	// Rehydrate a new instance of the root actor using the persisted snapshot
	restoredActor := xs.CreateActor(machine, xs.WithSnapshot(persistedSnapshot))

	cb := restoredActor.GetSnapshot().Children["cb"]
	require.NotNil(t, cb)
	xs.As[*xs.CallbackSnapshot](cb).On("emitted", func(ev xs.Event) {
		s.Call(ev)
	})

	restoredActor.Start()

	emit1AssertCalledWithContaining(t, s, xs.E{"type": "emitted", "msg": "hello"})
}
