package xstate_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// errors1Reporter stands in for the JS global `window` error listener that the
// tests install with installGlobalOnErrorHandler. In JS, reportUnhandledError
// is mocked to dispatch an ErrorEvent; in Go the root actor receives the same
// errors through xs.WithUnhandledErrorHandler.
type errors1Reporter struct {
	ch chan any
}

func newErrors1Reporter() *errors1Reporter { return &errors1Reporter{ch: make(chan any, 64)} }

// handle is passed to xs.WithUnhandledErrorHandler.
func (r *errors1Reporter) handle(err any) { r.ch <- err }

// waitN waits for n globally reported errors (in arrival order).
func (r *errors1Reporter) waitN(t *testing.T, n int) []any {
	t.Helper()
	out := make([]any, 0, n)
	timeout := time.After(2 * time.Second)
	for len(out) < n {
		select {
		case err := <-r.ch:
			out = append(out, err)
		case <-timeout:
			t.Fatalf("timed out waiting for %d globally reported errors, got %d: %v", n, len(out), out)
		}
	}
	return out
}

// assertNoneWithin mirrors the JS pattern: the global handler rejects with
// 'Fail' and a setTimeout(resolve, d) resolves the test.
func (r *errors1Reporter) assertNoneWithin(t *testing.T, d int) {
	t.Helper()
	sleep(d)
	select {
	case err := <-r.ch:
		t.Fatalf("Fail: unexpected globally reported error: %v", err)
	default:
	}
}

// errors1Message mirrors `err.message`; it fails when err is not an error.
func errors1Message(t *testing.T, err any) string {
	t.Helper()
	e, ok := err.(error)
	require.Truef(t, ok, "expected an error value, got %T (%v)", err, err)
	return e.Error()
}

// errors1AssertErrorCalls mirrors
// `expect(errorSpy.mock.calls).toMatchInlineSnapshot('[[ [Error: msg] ]]')`.
func errors1AssertErrorCalls(t *testing.T, s *spy, msg string) {
	t.Helper()
	calls := s.Calls()
	require.Len(t, calls, 1)
	require.Len(t, calls[0], 1)
	assert.Equal(t, msg, errors1Message(t, calls[0][0]))
}

// errors1OnlyChild mirrors `Object.values(snapshot.children)[0]`.
func errors1OnlyChild(t *testing.T, children map[string]xs.ActorRef) xs.ActorRef {
	t.Helper()
	require.Len(t, children, 1)
	for _, c := range children {
		return c
	}
	return nil
}

// JS: error handling > does not cause an infinite loop when an error is thrown in subscribe
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/errors.test.ts#L42
func TestErrors_DoesNotCauseInfiniteLoopWhenErrorIsThrownInSubscribe(t *testing.T) {
	type ctx struct{ Count int }
	reporter := newErrors1Reporter()
	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		ID:      "machine",
		Initial: "initial",
		Context: ctx{Count: 0},
		States: xs.States{
			{Key: "initial", On: map[string]xs.Transitions{"activate": {{Target: "active"}}}},
			{Key: "active"},
		},
	})

	s := newSpy()

	actor := xs.CreateActor(machine, xs.WithUnhandledErrorHandler(reporter.handle)).Start()

	actor.SubscribeNext(func(snap *xs.MachineSnapshot[ctx]) {
		s.Call(snap)
		panic(errors.New("no_infinite_loop_when_error_is_thrown_in_subscribe"))
	})
	actor.Send(xs.Ev("activate"))

	assert.Equal(t, 1, s.Count())

	errs := reporter.waitN(t, 1)
	assert.Equal(t, "no_infinite_loop_when_error_is_thrown_in_subscribe", errors1Message(t, errs[0]))
}

// JS: error handling > doesn't crash the actor when an error is thrown in subscribe
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/errors.test.ts#L80
func TestErrors_DoesntCrashActorWhenErrorIsThrownInSubscribe(t *testing.T) {
	type ctx struct{ Count int }
	reporter := newErrors1Reporter()
	s := newSpy()

	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		ID:      "machine",
		Initial: "initial",
		Context: ctx{Count: 0},
		States: xs.States{
			{Key: "initial", On: map[string]xs.Transitions{"activate": {{Target: "active"}}}},
			{Key: "active", On: map[string]xs.Transitions{
				"do": {{Actions: xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[ctx]) { s.Call(a.Event) })}}},
			}},
		},
	})

	subscriber := newSpy()

	actor := xs.CreateActor(machine, xs.WithUnhandledErrorHandler(reporter.handle)).Start()

	actor.SubscribeNext(func(snap *xs.MachineSnapshot[ctx]) {
		subscriber.Call(snap)
		// mockImplementationOnce: only the first call throws
		if subscriber.Count() == 1 {
			panic(errors.New("doesnt_crash_actor_when_error_is_thrown_in_subscribe"))
		}
	})
	actor.Send(xs.Ev("activate"))

	assert.Equal(t, 1, subscriber.Count())
	assert.Equal(t, xs.StatusActive, actor.GetSnapshot().Status)

	errs := reporter.waitN(t, 1)
	assert.Equal(t, "doesnt_crash_actor_when_error_is_thrown_in_subscribe", errors1Message(t, errs[0]))

	actor.Send(xs.Ev("do"))
	assert.Equal(t, 1, s.Count())
}

// JS: error handling > doesn't notify error listener when an error is thrown in subscribe
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/errors.test.ts#L131
func TestErrors_DoesntNotifyErrorListenerWhenErrorIsThrownInSubscribe(t *testing.T) {
	type ctx struct{ Count int }
	reporter := newErrors1Reporter()
	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		ID:      "machine",
		Initial: "initial",
		Context: ctx{Count: 0},
		States: xs.States{
			{Key: "initial", On: map[string]xs.Transitions{"activate": {{Target: "active"}}}},
			{Key: "active"},
		},
	})

	nextSpy := newSpy()
	errorSpy := newSpy()

	actor := xs.CreateActor(machine, xs.WithUnhandledErrorHandler(reporter.handle)).Start()

	actor.Subscribe(xs.Observer[*xs.MachineSnapshot[ctx]]{
		Next: func(snap *xs.MachineSnapshot[ctx]) {
			nextSpy.Call(snap)
			panic(errors.New("doesnt_notify_error_listener_when_error_is_thrown_in_subscribe"))
		},
		Error: func(err any) { errorSpy.Call(err) },
	})
	actor.Send(xs.Ev("activate"))

	assert.Equal(t, 1, nextSpy.Count())
	assert.Equal(t, 0, errorSpy.Count())

	errs := reporter.waitN(t, 1)
	assert.Equal(t, "doesnt_notify_error_listener_when_error_is_thrown_in_subscribe", errors1Message(t, errs[0]))
}

// JS: error handling > unhandled sync errors thrown when starting a child actor should be reported globally
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/errors.test.ts#L176
func TestErrors_UnhandledSyncErrorsWhenStartingChildActorShouldBeReportedGlobally(t *testing.T) {
	reporter := newErrors1Reporter()
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "pending",
		States: xs.States{
			{Key: "pending", Invoke: []xs.InvokeConfig{{
				Logic: xs.FromCallback(func(xs.CallbackArgs) func() {
					panic(errors.New("unhandled_sync_error_in_actor_start"))
				}),
				OnDone: xs.Transitions{{Target: "success"}},
			}}},
			{Key: "success", Type: xs.Final},
		},
	})

	xs.CreateActor(machine, xs.WithUnhandledErrorHandler(reporter.handle)).Start()

	errs := reporter.waitN(t, 1)
	assert.Equal(t, "unhandled_sync_error_in_actor_start", errors1Message(t, errs[0]))
}

// JS: error handling > unhandled rejection of a promise actor should be reported globally in absence of error listener
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/errors.test.ts#L206
func TestErrors_UnhandledRejectionOfPromiseActorReportedGloballyWithoutErrorListener(t *testing.T) {
	reporter := newErrors1Reporter()
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "pending",
		States: xs.States{
			{Key: "pending", Invoke: []xs.InvokeConfig{{
				Logic: xs.FromPromise(func(context.Context, xs.PromiseArgs) (any, error) {
					return nil, errors.New("unhandled_rejection_in_promise_actor_without_error_listener")
				}),
				OnDone: xs.Transitions{{Target: "success"}},
			}}},
			{Key: "success", Type: xs.Final},
		},
	})

	xs.CreateActor(machine, xs.WithUnhandledErrorHandler(reporter.handle)).Start()

	errs := reporter.waitN(t, 1)
	assert.Equal(t, "unhandled_rejection_in_promise_actor_without_error_listener", errors1Message(t, errs[0]))
}

// JS: error handling > unhandled rejection of a promise actor should be reported to the existing error listener of its parent
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/errors.test.ts#L242
func TestErrors_UnhandledRejectionOfPromiseActorReportedToParentErrorListener(t *testing.T) {
	errorSpy := newSpy()
	gotError := newSignal()

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "pending",
		States: xs.States{
			{Key: "pending", Invoke: []xs.InvokeConfig{{
				Logic: xs.FromPromise(func(context.Context, xs.PromiseArgs) (any, error) {
					return nil, errors.New("unhandled_rejection_in_promise_actor_with_parent_listener")
				}),
				OnDone: xs.Transitions{{Target: "success"}},
			}}},
			{Key: "success", Type: xs.Final},
		},
	})

	actorRef := xs.CreateActor(machine)
	actorRef.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{
		Error: func(err any) {
			errorSpy.Call(err)
			gotError.Resolve()
		},
	})
	actorRef.Start()

	// await sleep(0): the rejection settles on another goroutine in Go.
	gotError.Wait(t)

	errors1AssertErrorCalls(t, errorSpy, "unhandled_rejection_in_promise_actor_with_parent_listener")
}

// JS: error handling > unhandled rejection of a promise actor should be reported to the existing error listener of its grandparent
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/errors.test.ts#L283
func TestErrors_UnhandledRejectionOfPromiseActorReportedToGrandparentErrorListener(t *testing.T) {
	errorSpy := newSpy()
	gotError := newSignal()

	child := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "pending",
		States: xs.States{
			{Key: "pending", Invoke: []xs.InvokeConfig{{
				Logic: xs.FromPromise(func(context.Context, xs.PromiseArgs) (any, error) {
					return nil, errors.New("unhandled_rejection_in_promise_actor_with_grandparent_listener")
				}),
				OnDone: xs.Transitions{{Target: "success"}},
			}}},
			{Key: "success", Type: xs.Final},
		},
	})

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "pending",
		States: xs.States{
			{Key: "pending", Invoke: []xs.InvokeConfig{{
				Logic:  child,
				OnDone: xs.Transitions{{Target: "success"}},
			}}},
			{Key: "success", Type: xs.Final},
		},
	})

	actorRef := xs.CreateActor(machine)
	actorRef.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{
		Error: func(err any) {
			errorSpy.Call(err)
			gotError.Resolve()
		},
	})
	actorRef.Start()

	// await sleep(0): the rejection settles on another goroutine in Go.
	gotError.Wait(t)

	errors1AssertErrorCalls(t, errorSpy, "unhandled_rejection_in_promise_actor_with_grandparent_listener")
}

// JS: error handling > handled sync errors thrown when starting a child actor should not be reported globally
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/errors.test.ts#L339
func TestErrors_HandledSyncErrorsWhenStartingChildActorShouldNotBeReportedGlobally(t *testing.T) {
	reporter := newErrors1Reporter()
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "pending",
		States: xs.States{
			{Key: "pending", Invoke: []xs.InvokeConfig{{
				Logic: xs.FromCallback(func(xs.CallbackArgs) func() {
					panic(errors.New("handled_sync_error_in_actor_start"))
				}),
				OnError: xs.Transitions{{Target: "failed"}},
			}}},
			{Key: "failed", Type: xs.Final},
		},
	})

	xs.CreateActor(machine, xs.WithUnhandledErrorHandler(reporter.handle)).Start()

	reporter.assertNoneWithin(t, 10)
}

// JS: error handling > handled sync errors thrown when starting a child actor should be reported globally when not all of its own observers come with an error listener
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/errors.test.ts#L371
func TestErrors_HandledSyncErrorsChildReportedGloballyWhenNotAllObserversHaveErrorListener(t *testing.T) {
	reporter := newErrors1Reporter()
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "pending",
		States: xs.States{
			{Key: "pending", Invoke: []xs.InvokeConfig{{
				Logic: xs.FromCallback(func(xs.CallbackArgs) func() {
					panic(errors.New("handled_sync_error_in_actor_start"))
				}),
				OnError: xs.Transitions{{Target: "failed"}},
			}}},
			{Key: "failed", Type: xs.Final},
		},
	})

	actorRef := xs.CreateActor(machine, xs.WithUnhandledErrorHandler(reporter.handle))
	childActorRef := errors1OnlyChild(t, actorRef.GetSnapshot().Children)
	childActorRef.SubscribeAny(xs.Observer[xs.Snapshot]{
		Error: func(any) {}, // preventUnhandledErrorListener
	})
	childActorRef.SubscribeAny(xs.Observer[xs.Snapshot]{Next: func(xs.Snapshot) {}})
	actorRef.Start()

	errs := reporter.waitN(t, 1)
	assert.Equal(t, "handled_sync_error_in_actor_start", errors1Message(t, errs[0]))
}

// JS: error handling > handled sync errors thrown when starting a child actor should not be reported globally when all of its own observers come with an error listener
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/errors.test.ts#L407
func TestErrors_HandledSyncErrorsChildNotReportedGloballyWhenAllObserversHaveErrorListener(t *testing.T) {
	reporter := newErrors1Reporter()
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "pending",
		States: xs.States{
			{Key: "pending", Invoke: []xs.InvokeConfig{{
				Logic: xs.FromCallback(func(xs.CallbackArgs) func() {
					panic(errors.New("handled_sync_error_in_actor_start"))
				}),
				OnError: xs.Transitions{{Target: "failed"}},
			}}},
			{Key: "failed", Type: xs.Final},
		},
	})

	actorRef := xs.CreateActor(machine, xs.WithUnhandledErrorHandler(reporter.handle))
	childActorRef := errors1OnlyChild(t, actorRef.GetSnapshot().Children)
	childActorRef.SubscribeAny(xs.Observer[xs.Snapshot]{
		Error: func(any) {}, // preventUnhandledErrorListener
	})
	childActorRef.SubscribeAny(xs.Observer[xs.Snapshot]{
		Error: func(any) {}, // preventUnhandledErrorListener
	})
	actorRef.Start()

	reporter.assertNoneWithin(t, 10)
}

// JS: error handling > unhandled sync errors thrown when starting a child actor should be reported twice globally when not all of its own observers come with an error listener and when the root has no error listener of its own
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/errors.test.ts#L447
func TestErrors_UnhandledSyncErrorsChildReportedTwiceGloballyWhenRootHasNoErrorListener(t *testing.T) {
	reporter := newErrors1Reporter()
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "pending",
		States: xs.States{
			{Key: "pending", Invoke: []xs.InvokeConfig{{
				Logic: xs.FromCallback(func(xs.CallbackArgs) func() {
					panic(errors.New("handled_sync_error_in_actor_start"))
				}),
			}}},
		},
	})

	actorRef := xs.CreateActor(machine, xs.WithUnhandledErrorHandler(reporter.handle))
	childActorRef := errors1OnlyChild(t, actorRef.GetSnapshot().Children)
	childActorRef.SubscribeAny(xs.Observer[xs.Snapshot]{
		Error: func(any) {}, // preventUnhandledErrorListener
	})
	childActorRef.SubscribeAny(xs.Observer[xs.Snapshot]{})
	actorRef.Start()

	errs := reporter.waitN(t, 2)
	actual := []string{errors1Message(t, errs[0]), errors1Message(t, errs[1])}
	assert.Equal(t, []string{
		"handled_sync_error_in_actor_start",
		"handled_sync_error_in_actor_start",
	}, actual)
}

// JS: error handling > handled sync errors shouldn't notify the error listener
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/errors.test.ts#L488
func TestErrors_HandledSyncErrorsShouldntNotifyErrorListener(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "pending",
		States: xs.States{
			{Key: "pending", Invoke: []xs.InvokeConfig{{
				Logic: xs.FromCallback(func(xs.CallbackArgs) func() {
					panic(errors.New("handled_sync_error_in_actor_start"))
				}),
				OnError: xs.Transitions{{Target: "failed"}},
			}}},
			{Key: "failed", Type: xs.Final},
		},
	})

	errorSpy := newSpy()

	actorRef := xs.CreateActor(machine)
	actorRef.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{
		Error: func(err any) { errorSpy.Call(err) },
	})
	actorRef.Start()

	assert.Equal(t, 0, errorSpy.Count())
}

// JS: error handling > unhandled sync errors should notify the root error listener
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/errors.test.ts#L517
func TestErrors_UnhandledSyncErrorsShouldNotifyRootErrorListener(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "pending",
		States: xs.States{
			{Key: "pending", Invoke: []xs.InvokeConfig{{
				Logic: xs.FromCallback(func(xs.CallbackArgs) func() {
					panic(errors.New("unhandled_sync_error_in_actor_start_with_root_error_listener"))
				}),
				OnDone: xs.Transitions{{Target: "success"}},
			}}},
			{Key: "success", Type: xs.Final},
		},
	})

	errorSpy := newSpy()

	actorRef := xs.CreateActor(machine)
	actorRef.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{
		Error: func(err any) { errorSpy.Call(err) },
	})
	actorRef.Start()

	errors1AssertErrorCalls(t, errorSpy, "unhandled_sync_error_in_actor_start_with_root_error_listener")
}

// JS: error handling > unhandled sync errors should not notify the global listener when the root error listener is present
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/errors.test.ts#L554
func TestErrors_UnhandledSyncErrorsShouldNotNotifyGlobalListenerWhenRootErrorListenerPresent(t *testing.T) {
	reporter := newErrors1Reporter()
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "pending",
		States: xs.States{
			{Key: "pending", Invoke: []xs.InvokeConfig{{
				Logic: xs.FromCallback(func(xs.CallbackArgs) func() {
					panic(errors.New("unhandled_sync_error_in_actor_start_with_root_error_listener"))
				}),
				OnDone: xs.Transitions{{Target: "success"}},
			}}},
			{Key: "success", Type: xs.Final},
		},
	})

	errorSpy := newSpy()

	actorRef := xs.CreateActor(machine, xs.WithUnhandledErrorHandler(reporter.handle))
	actorRef.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{
		Error: func(err any) { errorSpy.Call(err) },
	})
	actorRef.Start()

	assert.Equal(t, 1, errorSpy.Count())

	reporter.assertNoneWithin(t, 10)
}

// JS: error handling > handled sync errors thrown when starting an actor shouldn't crash the parent
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/errors.test.ts#L596
func TestErrors_HandledSyncErrorsWhenStartingActorShouldntCrashParent(t *testing.T) {
	s := newSpy()

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "pending",
		States: xs.States{
			{Key: "pending", Invoke: []xs.InvokeConfig{{
				Logic: xs.FromCallback(func(xs.CallbackArgs) func() {
					panic(errors.New("handled_sync_error_in_actor_start"))
				}),
				OnError: xs.Transitions{{Target: "failed"}},
			}}},
			{Key: "failed", On: map[string]xs.Transitions{
				"do": {{Actions: xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[any]) { s.Call(a.Event) })}}},
			}},
		},
	})

	actorRef := xs.CreateActor(machine)
	actorRef.Start()

	assert.Equal(t, xs.StatusActive, actorRef.GetSnapshot().Status)

	actorRef.Send(xs.Ev("do"))
	assert.Equal(t, 1, s.Count())
}

// JS: error handling > unhandled sync errors thrown when starting an actor should crash the parent
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/errors.test.ts#L629
func TestErrors_UnhandledSyncErrorsWhenStartingActorShouldCrashParent(t *testing.T) {
	reporter := newErrors1Reporter()
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "pending",
		States: xs.States{
			{Key: "pending", Invoke: []xs.InvokeConfig{{
				Logic: xs.FromCallback(func(xs.CallbackArgs) func() {
					panic(errors.New("unhandled_sync_error_in_actor_start"))
				}),
			}}},
		},
	})

	actorRef := xs.CreateActor(machine, xs.WithUnhandledErrorHandler(reporter.handle))
	actorRef.Start()

	assert.Equal(t, xs.StatusError, actorRef.GetSnapshot().Status)

	errs := reporter.waitN(t, 1)
	assert.Equal(t, "unhandled_sync_error_in_actor_start", errors1Message(t, errs[0]))
}

// JS: error handling > error thrown by the error listener should be reported globally
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/errors.test.ts#L658
func TestErrors_ErrorThrownByErrorListenerShouldBeReportedGlobally(t *testing.T) {
	reporter := newErrors1Reporter()
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "pending",
		States: xs.States{
			{Key: "pending", Invoke: []xs.InvokeConfig{{
				Logic: xs.FromCallback(func(xs.CallbackArgs) func() {
					panic(errors.New("handled_sync_error_in_actor_start"))
				}),
			}}},
		},
	})

	actorRef := xs.CreateActor(machine, xs.WithUnhandledErrorHandler(reporter.handle))
	actorRef.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{
		Error: func(any) {
			panic(errors.New("error_thrown_by_error_listener"))
		},
	})
	actorRef.Start()

	errs := reporter.waitN(t, 1)
	assert.Equal(t, "error_thrown_by_error_listener", errors1Message(t, errs[0]))
}

// JS: error handling > error should be reported globally if not every observer comes with an error listener
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/errors.test.ts#L690
func TestErrors_ErrorReportedGloballyIfNotEveryObserverHasErrorListener(t *testing.T) {
	reporter := newErrors1Reporter()
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "pending",
		States: xs.States{
			{Key: "pending", Invoke: []xs.InvokeConfig{{
				Logic: xs.FromCallback(func(xs.CallbackArgs) func() {
					panic(errors.New("error_thrown_when_not_every_observer_comes_with_an_error_listener"))
				}),
			}}},
		},
	})

	actorRef := xs.CreateActor(machine, xs.WithUnhandledErrorHandler(reporter.handle))
	actorRef.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{
		Error: func(any) {}, // preventUnhandledErrorListener
	})
	actorRef.SubscribeNext(func(*xs.MachineSnapshot[any]) {})
	actorRef.Start()

	errs := reporter.waitN(t, 1)
	assert.Equal(t, "error_thrown_when_not_every_observer_comes_with_an_error_listener", errors1Message(t, errs[0]))
}

// JS: error handling > uncaught error and an error thrown by the error listener should both be reported globally when not every observer comes with an error listener
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/errors.test.ts#L724
func TestErrors_UncaughtErrorAndErrorListenerErrorBothReportedGlobally(t *testing.T) {
	reporter := newErrors1Reporter()
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "pending",
		States: xs.States{
			{Key: "pending", Invoke: []xs.InvokeConfig{{
				Logic: xs.FromCallback(func(xs.CallbackArgs) func() {
					panic(errors.New("error_thrown_when_not_every_observer_comes_with_an_error_listener"))
				}),
			}}},
		},
	})

	actorRef := xs.CreateActor(machine, xs.WithUnhandledErrorHandler(reporter.handle))
	actorRef.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{
		Error: func(any) {
			panic(errors.New("error_thrown_by_error_listener"))
		},
	})
	actorRef.SubscribeNext(func(*xs.MachineSnapshot[any]) {})
	actorRef.Start()

	errs := reporter.waitN(t, 2)
	actual := []string{errors1Message(t, errs[0]), errors1Message(t, errs[1])}
	assert.Equal(t, []string{
		"error_thrown_by_error_listener",
		"error_thrown_when_not_every_observer_comes_with_an_error_listener",
	}, actual)
}

// JS: error handling > error thrown in initial custom entry action should error the actor
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/errors.test.ts#L768
func TestErrors_ErrorThrownInInitialCustomEntryActionShouldErrorActor(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Entry: xs.Actions{xs.ActionFunc(func(xs.ActionArgs[any]) {
			panic(errors.New("error_thrown_in_initial_entry_action"))
		})},
	})

	errorSpy := newSpy()

	actorRef := xs.CreateActor(machine)
	actorRef.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{
		Error: func(err any) { errorSpy.Call(err) },
	})
	actorRef.Start()

	snapshot := actorRef.GetSnapshot()
	assert.Equal(t, xs.StatusError, snapshot.Status)
	assert.Equal(t, "error_thrown_in_initial_entry_action", errors1Message(t, snapshot.Error))
	errors1AssertErrorCalls(t, errorSpy, "error_thrown_in_initial_entry_action")
}

// JS: error handling > error thrown when resolving initial builtin entry action should error the actor immediately
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/errors.test.ts#L797
func TestErrors_ErrorThrownResolvingInitialBuiltinEntryActionShouldErrorActorImmediately(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Entry: xs.Actions{xs.Assign(func(xs.AssignArgs[any]) any {
			panic(errors.New("error_thrown_when_resolving_initial_entry_action"))
		})},
	})

	errorSpy := newSpy()

	actorRef := xs.CreateActor(machine)

	snapshot := actorRef.GetSnapshot()
	assert.Equal(t, xs.StatusError, snapshot.Status)
	assert.Equal(t, "error_thrown_when_resolving_initial_entry_action", errors1Message(t, snapshot.Error))

	actorRef.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{
		Error: func(err any) { errorSpy.Call(err) },
	})
	actorRef.Start()

	errors1AssertErrorCalls(t, errorSpy, "error_thrown_when_resolving_initial_entry_action")
}

// JS: error handling > error thrown by a custom entry action when transitioning should error the actor
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/errors.test.ts#L828
func TestErrors_ErrorThrownByCustomEntryActionWhenTransitioningShouldErrorActor(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{"NEXT": {{Target: "b"}}}},
			{Key: "b", Entry: xs.Actions{xs.ActionFunc(func(xs.ActionArgs[any]) {
				panic(errors.New("error_thrown_in_a_custom_entry_action_when_transitioning"))
			})}},
		},
	})

	errorSpy := newSpy()

	actorRef := xs.CreateActor(machine)
	actorRef.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{
		Error: func(err any) { errorSpy.Call(err) },
	})
	actorRef.Start()
	actorRef.Send(xs.Ev("NEXT"))

	snapshot := actorRef.GetSnapshot()
	assert.Equal(t, xs.StatusError, snapshot.Status)
	assert.Equal(t, "error_thrown_in_a_custom_entry_action_when_transitioning", errors1Message(t, snapshot.Error))
	errors1AssertErrorCalls(t, errorSpy, "error_thrown_in_a_custom_entry_action_when_transitioning")
}

// JS: error handling > shouldn't execute deferred initial actions that come after an action that errors
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/errors.test.ts#L870
func TestErrors_ShouldntExecuteDeferredInitialActionsAfterActionThatErrors(t *testing.T) {
	s := newSpy()

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Entry: xs.Actions{
			xs.ActionFunc(func(xs.ActionArgs[any]) {
				panic(errors.New("error_thrown_in_initial_entry_action"))
			}),
			xs.ActionFunc(func(a xs.ActionArgs[any]) { s.Call(a.Event) }),
		},
	})

	actorRef := xs.CreateActor(machine)
	actorRef.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{
		Error: func(any) {}, // preventUnhandledErrorListener
	})
	actorRef.Start()

	assert.Equal(t, 0, s.Count())
}

// JS: error handling > should error the parent on errored initial state of a child
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/errors.test.ts#L889
func TestErrors_ShouldErrorParentOnErroredInitialStateOfChild(t *testing.T) {
	// JS: fromTransition((_) => undefined, undefined) with getInitialSnapshot
	// overridden. The Go TransitionLogic has no overridable fields, so the same
	// logic is expressed as custom logic producing a TransitionSnapshot.
	immediateFailure := &xs.Logic[*xs.TransitionSnapshot[any]]{
		Transition: func(s *xs.TransitionSnapshot[any], _ xs.Event, _ *xs.ActorScope) *xs.TransitionSnapshot[any] {
			return &xs.TransitionSnapshot[any]{Status: s.Status, Context: nil, Output: s.Output, Error: s.Error}
		},
		GetInitialSnapshot: func(*xs.ActorScope, any) *xs.TransitionSnapshot[any] {
			return &xs.TransitionSnapshot[any]{
				Status:  xs.StatusError,
				Output:  nil,
				Error:   "immediate error!",
				Context: nil,
			}
		},
	}

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Invoke: []xs.InvokeConfig{{Src: "failure"}},
	}, xs.Implementations{Actors: map[string]xs.ActorLogic{"failure": immediateFailure}})

	actorRef := xs.CreateActor(machine)
	actorRef.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{
		Error: func(any) {}, // preventUnhandledErrorListener
	})
	actorRef.Start()

	snapshot := actorRef.GetSnapshot()

	assert.Equal(t, xs.StatusError, snapshot.Status)
	assert.Equal(t, "immediate error!", snapshot.Error)
}

// JS: error handling > should error when a guard throws when transitioning
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/errors.test.ts#L921
func TestErrors_ShouldErrorWhenGuardThrowsWhenTransitioning(t *testing.T) {
	s := newSpy()
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{
				"NEXT": {{
					Guard: xs.GuardFunc(func(xs.GuardArgs[any]) bool {
						panic(errors.New("error_thrown_in_guard_when_transitioning"))
					}),
					Target: "b",
				}},
			}},
			{Key: "b"},
		},
	})

	actorRef := xs.CreateActor(machine)
	actorRef.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{
		Error: func(err any) { s.Call(err) },
	})
	actorRef.Start()
	actorRef.Send(xs.Ev("NEXT"))

	snapshot := actorRef.GetSnapshot()
	assert.Equal(t, xs.StatusError, snapshot.Status)
	assert.Equal(t,
		"Unable to evaluate guard in transition for event 'NEXT' in state node '(machine).a':\n"+
			"error_thrown_in_guard_when_transitioning",
		errors1Message(t, snapshot.Error))
}

// JS: error handling > actor continues to work normally after emit callback errors
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/errors.test.ts#L955
func TestErrors_ActorContinuesToWorkNormallyAfterEmitCallbackErrors(t *testing.T) {
	// types.emitted is type-level only.
	machine := xs.NewSetup[any](xs.Implementations{}).CreateMachine(xs.MachineConfig[any]{
		On: map[string]xs.Transitions{
			"someEvent": {{Actions: xs.Actions{xs.Emit(xs.E{"type": "emitted", "foo": "bar"})}}},
		},
	})

	actor := xs.CreateActor(machine).Start()
	var errorThrown atomic.Bool

	actor.On("emitted", func(xs.Event) {
		errorThrown.Store(true)
		panic(errors.New("oops"))
	})

	// Send first event - should trigger error but actor should remain active
	actor.Send(xs.Ev("someEvent"))
	sleep(10)

	assert.True(t, errorThrown.Load())
	assert.Equal(t, xs.StatusActive, actor.GetSnapshot().Status)

	// Send second event - should work normally without error
	received := make(chan xs.Event, 1)
	actor.On("emitted", func(e xs.Event) {
		select {
		case received <- e:
		default:
		}
	})
	actor.Send(xs.Ev("someEvent"))

	var event xs.Event
	select {
	case event = <-received:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for the emitted event")
	}

	e, ok := event.(xs.E)
	require.True(t, ok)
	assert.Equal(t, "bar", e["foo"])
	assert.Equal(t, xs.StatusActive, actor.GetSnapshot().Status)
}
