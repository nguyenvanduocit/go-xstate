package xstate_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// actorLogic1Recv receives one value from ch, failing the test after 2s.
func actorLogic1Recv[T any](t *testing.T, ch <-chan T) T {
	t.Helper()
	select {
	case v := <-ch:
		return v
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for a value")
	}
	var zero T
	return zero
}

// actorLogic1Probe collects values observed inside actor-logic callbacks so
// the test goroutine can assert on them. It mirrors `expect.assertions(n)`
// combined with `expect(...)` calls placed inside those callbacks.
type actorLogic1Probe struct {
	mu   sync.Mutex
	vals []any
	sig  *signal
}

func newActorLogic1Probe() *actorLogic1Probe { return &actorLogic1Probe{sig: newSignal()} }

func (p *actorLogic1Probe) Record(v any) {
	p.mu.Lock()
	p.vals = append(p.vals, v)
	p.mu.Unlock()
	p.sig.Resolve()
}

// Values waits for the first recorded value and returns all recorded values.
func (p *actorLogic1Probe) Values(t *testing.T) []any {
	t.Helper()
	p.sig.Wait(t)
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]any, len(p.vals))
	copy(out, p.vals)
	return out
}

// actorLogic1Deferred mirrors a `Promise.withResolvers()` created inside a
// fromPromise body, together with that body's abort signal (ctx) and self.
// "abort listener was called" maps to ctx.Err() != nil.
type actorLogic1Deferred struct {
	self    xs.ActorRef
	ctx     context.Context
	resolve chan int
}

// actorLogic1DeferredPromise mirrors
// `fromPromise(({ self, signal }) => { const deferred = Promise.withResolvers(); ...; return deferred.promise; })`:
// every invocation registers its deferred on regs.
func actorLogic1DeferredPromise(regs chan<- *actorLogic1Deferred) *xs.PromiseLogic[int] {
	return xs.FromPromise(func(ctx context.Context, a xs.PromiseArgs) (int, error) {
		d := &actorLogic1Deferred{self: a.Self, ctx: ctx, resolve: make(chan int, 1)}
		regs <- d
		return <-d.resolve, nil
	})
}

// actorLogic1RejectErr mirrors a promise rejected with a number:
// `Promise.reject(createdPromises)`.
type actorLogic1RejectErr int

func (e actorLogic1RejectErr) Error() string { return fmt.Sprintf("rejected with %d", int(e)) }

// actorLogic1Field walks a persisted machine snapshot (a JSON-like nested
// map[string]any mirroring the JS persisted object) along keys.
func actorLogic1Field(t *testing.T, v any, keys ...string) any {
	t.Helper()
	for _, k := range keys {
		m, ok := v.(map[string]any)
		require.Truef(t, ok, "expected map[string]any before key %q, got %T", k, v)
		v = m[k]
	}
	return v
}

// ---- promise logic (fromPromise) ----

// JS: promise logic (fromPromise) > should interpret a promise
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actorLogic.test.ts#L24
func TestActorLogic_FromPromise_ShouldInterpretAPromise(t *testing.T) {
	promiseLogic := xs.FromPromise(func(_ context.Context, _ xs.PromiseArgs) (string, error) {
		time.Sleep(ms(10))
		return "hello", nil
	})

	actor := xs.CreateActor(promiseLogic).Start()

	snapshot, err := xs.WaitFor(context.Background(), actor, func(s *xs.PromiseSnapshot[string]) bool {
		return s.Output == "hello"
	}).Wait()
	require.NoError(t, err)

	assert.Equal(t, "hello", snapshot.Output)
}

// JS: promise logic (fromPromise) > should resolve
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actorLogic.test.ts#L38
func TestActorLogic_FromPromise_ShouldResolve(t *testing.T) {
	sig := newSignal()
	actor := xs.CreateActor(xs.FromPromise(func(_ context.Context, _ xs.PromiseArgs) (int, error) { return 42, nil }))

	actor.SubscribeNext(func(s *xs.PromiseSnapshot[int]) {
		if s.Output == 42 {
			sig.Resolve()
		}
	})

	actor.Start()
	sig.Wait(t)
}

// JS: promise logic (fromPromise) > should resolve (observer .next)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actorLogic.test.ts#L52
func TestActorLogic_FromPromise_ShouldResolveObserverNext(t *testing.T) {
	sig := newSignal()
	actor := xs.CreateActor(xs.FromPromise(func(_ context.Context, _ xs.PromiseArgs) (int, error) { return 42, nil }))

	actor.Subscribe(xs.Observer[*xs.PromiseSnapshot[int]]{
		Next: func(s *xs.PromiseSnapshot[int]) {
			if s.Output == 42 {
				sig.Resolve()
			}
		},
	})

	actor.Start()
	sig.Wait(t)
}

// JS: promise logic (fromPromise) > should reject (observer .error)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actorLogic.test.ts#L68
func TestActorLogic_FromPromise_ShouldRejectObserverError(t *testing.T) {
	rejection := errors.New("Error")
	errs := make(chan any, 1)
	actor := xs.CreateActor(xs.FromPromise(func(_ context.Context, _ xs.PromiseArgs) (int, error) { return 0, rejection }))

	actor.Subscribe(xs.Observer[*xs.PromiseSnapshot[int]]{
		Error: func(data any) { errs <- data },
	})

	actor.Start()
	assert.Equal(t, rejection, actorLogic1Recv(t, errs))
}

// JS: promise logic (fromPromise) > should complete (observer .complete)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actorLogic.test.ts#L83
func TestActorLogic_FromPromise_ShouldCompleteObserverComplete(t *testing.T) {
	actor := xs.CreateActor(xs.FromPromise(func(_ context.Context, _ xs.PromiseArgs) (int, error) { return 42, nil }))
	actor.Start()

	snapshot, err := xs.WaitFor(context.Background(), actor, func(s *xs.PromiseSnapshot[int]) bool {
		return s.Output == 42
	}).Wait()
	require.NoError(t, err)

	assert.Equal(t, 42, snapshot.Output)
}

// JS: promise logic (fromPromise) > should not execute when reading initial state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actorLogic.test.ts#L92
func TestActorLogic_FromPromise_ShouldNotExecuteWhenReadingInitialState(t *testing.T) {
	var called atomic.Bool
	logic := xs.FromPromise(func(_ context.Context, _ xs.PromiseArgs) (int, error) {
		called.Store(true)
		return 42, nil
	})

	actor := xs.CreateActor(logic)

	actor.GetSnapshot()

	assert.False(t, called.Load())
}

// JS: promise logic (fromPromise) > should persist an unresolved promise
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actorLogic.test.ts#L106
func TestActorLogic_FromPromise_ShouldPersistAnUnresolvedPromise(t *testing.T) {
	promiseLogic := xs.FromPromise(func(_ context.Context, _ xs.PromiseArgs) (int, error) {
		time.Sleep(ms(10))
		return 42, nil
	})

	actor := xs.CreateActor(promiseLogic)
	actor.Start()

	resolvedPersistedState := actor.GetPersistedSnapshot()
	actor.Stop()

	restoredActor := xs.CreateActor(promiseLogic, xs.WithSnapshot(resolvedPersistedState)).Start()

	sleep(20)
	assert.Equal(t, 42, restoredActor.GetSnapshot().Output)
}

// JS: promise logic (fromPromise) > should persist a resolved promise
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actorLogic.test.ts#L128
func TestActorLogic_FromPromise_ShouldPersistAResolvedPromise(t *testing.T) {
	promiseLogic := xs.FromPromise(func(_ context.Context, _ xs.PromiseArgs) (int, error) {
		return 42, nil
	})

	actor := xs.CreateActor(promiseLogic)
	actor.Start()

	// JS: setTimeout(() => { ...assertions...; resolve() }, 5); return promise;
	sleep(5)

	resolvedPersistedState := actor.GetPersistedSnapshot()

	assert.Equal(t, &xs.PromiseSnapshot[int]{
		Error:  nil,
		Input:  nil,
		Output: 42,
		Status: xs.StatusDone,
	}, resolvedPersistedState)

	restoredActor := xs.CreateActor(promiseLogic, xs.WithSnapshot(resolvedPersistedState)).Start()
	assert.Equal(t, 42, restoredActor.GetSnapshot().Output)
}

// JS: promise logic (fromPromise) > should not invoke a resolved promise again
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actorLogic.test.ts#L161
func TestActorLogic_FromPromise_ShouldNotInvokeAResolvedPromiseAgain(t *testing.T) {
	var createdPromises atomic.Int32
	promiseLogic := xs.FromPromise(func(_ context.Context, _ xs.PromiseArgs) (int, error) {
		return int(createdPromises.Add(1)), nil
	})
	actor := xs.CreateActor(promiseLogic)
	actor.Start()

	sleep(5)

	resolvedPersistedState := actor.GetPersistedSnapshot()
	assert.Equal(t, &xs.PromiseSnapshot[int]{
		Error:  nil,
		Input:  nil,
		Output: 1,
		Status: xs.StatusDone,
	}, resolvedPersistedState)
	assert.Equal(t, int32(1), createdPromises.Load())

	restoredActor := xs.CreateActor(promiseLogic, xs.WithSnapshot(resolvedPersistedState)).Start()

	assert.Equal(t, 1, restoredActor.GetSnapshot().Output)
	assert.Equal(t, int32(1), createdPromises.Load())
}

// JS: promise logic (fromPromise) > should not invoke a rejected promise again
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actorLogic.test.ts#L191
func TestActorLogic_FromPromise_ShouldNotInvokeARejectedPromiseAgain(t *testing.T) {
	var createdPromises atomic.Int32
	promiseLogic := xs.FromPromise(func(_ context.Context, _ xs.PromiseArgs) (int, error) {
		return 0, actorLogic1RejectErr(createdPromises.Add(1))
	})
	actorRef := xs.CreateActor(promiseLogic)
	actorRef.Subscribe(xs.Observer[*xs.PromiseSnapshot[int]]{Error: func(any) {}}) // preventUnhandledErrorListener
	actorRef.Start()

	sleep(5)

	rejectedPersistedState := actorRef.GetPersistedSnapshot()
	assert.Equal(t, &xs.PromiseSnapshot[int]{
		Error:  actorLogic1RejectErr(1),
		Input:  nil,
		Output: 0,
		Status: xs.StatusError,
	}, rejectedPersistedState)
	assert.Equal(t, int32(1), createdPromises.Load())

	actorRef2 := xs.CreateActor(promiseLogic, xs.WithSnapshot(rejectedPersistedState))
	actorRef2.Subscribe(xs.Observer[*xs.PromiseSnapshot[int]]{Error: func(any) {}}) // preventUnhandledErrorListener
	actorRef2.Start()

	assert.Equal(t, int32(1), createdPromises.Load())
}

// JS: promise logic (fromPromise) > should have access to the system
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actorLogic.test.ts#L223
func TestActorLogic_FromPromise_ShouldHaveAccessToTheSystem(t *testing.T) {
	probe := newActorLogic1Probe()
	promiseLogic := xs.FromPromise(func(_ context.Context, a xs.PromiseArgs) (int, error) {
		probe.Record(a.System)
		return 42, nil
	})

	xs.CreateActor(promiseLogic).Start()

	vals := probe.Values(t)
	require.Len(t, vals, 1) // expect.assertions(1)
	assert.NotNil(t, vals[0])
}

// JS: promise logic (fromPromise) > should have reference to self
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actorLogic.test.ts#L233
func TestActorLogic_FromPromise_ShouldHaveReferenceToSelf(t *testing.T) {
	probe := newActorLogic1Probe()
	promiseLogic := xs.FromPromise(func(_ context.Context, a xs.PromiseArgs) (int, error) {
		probe.Record(a.Self)
		return 42, nil
	})

	xs.CreateActor(promiseLogic).Start()

	vals := probe.Values(t)
	require.Len(t, vals, 1) // expect.assertions(1)
	assert.NotNil(t, vals[0])
}

// JS: promise logic (fromPromise) > should abort when stopping
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actorLogic.test.ts#L244
func TestActorLogic_FromPromise_ShouldAbortWhenStopping(t *testing.T) {
	// The JS promise never settles; the Go body blocks until test cleanup.
	regs := make(chan *actorLogic1Deferred, 1)
	promiseLogic := actorLogic1DeferredPromise(regs)

	actor := xs.CreateActor(promiseLogic).Start()
	d := actorLogic1Recv(t, regs) // the JS body runs synchronously inside start()
	t.Cleanup(func() { d.resolve <- 0 })

	actor.Stop()

	assert.Error(t, d.ctx.Err(), "abort listener should have been called")
}

// JS: promise logic (fromPromise) > should not abort when stopped if promise is resolved/rejected
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actorLogic.test.ts#L262
func TestActorLogic_FromPromise_ShouldNotAbortWhenStoppedIfPromiseIsResolvedRejected(t *testing.T) {
	resolvedRegs := make(chan *actorLogic1Deferred, 1)
	resolvedPromiseLogic := actorLogic1DeferredPromise(resolvedRegs)

	rejectedDeferred := make(chan error, 1)
	rejectedCtx := make(chan context.Context, 1)
	rejectedPromiseLogic := xs.FromPromise(func(ctx context.Context, _ xs.PromiseArgs) (int, error) {
		rejectedCtx <- ctx
		<-rejectedDeferred // rejectedDeferred.promise.catch(() => {}) resolves with undefined
		return 0, nil
	})

	actor := xs.CreateActor(resolvedPromiseLogic).Start()
	resolvedDeferred := actorLogic1Recv(t, resolvedRegs)
	resolvedDeferred.resolve <- 42
	_, err := xs.WaitFor(context.Background(), actor, func(s *xs.PromiseSnapshot[int]) bool {
		return s.Status == xs.StatusDone
	}).Wait()
	require.NoError(t, err)
	actor.Stop()
	assert.NoError(t, resolvedDeferred.ctx.Err(), "resolvedSignalListener should not have been called")

	actor2 := xs.CreateActor(rejectedPromiseLogic).Start()
	ctx2 := actorLogic1Recv(t, rejectedCtx)

	rejectedDeferred <- errors.New("50")
	_, err = xs.WaitFor(context.Background(), actor2, func(s *xs.PromiseSnapshot[int]) bool {
		return s.Status == xs.StatusDone
	}).Wait()
	require.NoError(t, err)
	actor2.Stop()
	assert.NoError(t, ctx2.Err(), "rejectedSignalListener should not have been called")
}

// JS: promise logic (fromPromise) > should not reuse the same signal for different actors with same logic
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actorLogic.test.ts#L292
func TestActorLogic_FromPromise_ShouldNotReuseSignalForDifferentActorsWithSameLogic(t *testing.T) {
	regs := make(chan *actorLogic1Deferred, 2)
	p := actorLogic1DeferredPromise(regs)
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Type: xs.Parallel,
		States: xs.States{
			{
				Key:     "p1",
				Initial: "running",
				States: xs.States{
					{
						Key:    "running",
						Invoke: []xs.InvokeConfig{{Logic: p, ID: "p1"}},
						On:     map[string]xs.Transitions{"CANCEL_1": {{Target: "canceled"}}},
					},
					{Key: "canceled"},
				},
			},
			{
				Key:     "p2",
				Initial: "running",
				States: xs.States{
					{
						Key:    "running",
						Invoke: []xs.InvokeConfig{{Logic: p, ID: "p2", OnDone: xs.Transitions{{Target: "done"}}}},
					},
					{Key: "done"},
				},
			},
		},
	})
	actor := xs.CreateActor(machine).Start()

	deferredMap := map[string]*actorLogic1Deferred{}
	for range 2 {
		d := actorLogic1Recv(t, regs)
		deferredMap[d.self.ID()] = d
	}
	p1Deferred := deferredMap["p1"]
	p2Deferred := deferredMap["p2"]
	require.NotNil(t, p1Deferred)
	require.NotNil(t, p2Deferred)

	actor.Send(xs.Ev("CANCEL_1"))
	p1Deferred.resolve <- 42
	p2Deferred.resolve <- 42
	w1 := xs.WaitFor(context.Background(), actor, func(s *xs.MachineSnapshot[any]) bool { return s.Matches("p1.canceled") })
	w2 := xs.WaitFor(context.Background(), actor, func(s *xs.MachineSnapshot[any]) bool { return s.Matches("p2.done") })
	_, err := w1.Wait()
	require.NoError(t, err)
	_, err = w2.Wait()
	require.NoError(t, err)

	assert.Error(t, p1Deferred.ctx.Err(), "p1 signal listener should have been called")
	assert.NoError(t, p2Deferred.ctx.Err(), "p2 signal listener should not have been called")
}

// JS: promise logic (fromPromise) > should not reuse the same signal for different actors with same logic and id
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actorLogic.test.ts#L352
func TestActorLogic_FromPromise_ShouldNotReuseSignalForDifferentActorsWithSameLogicAndID(t *testing.T) {
	regs := make(chan *actorLogic1Deferred, 2)
	p := actorLogic1DeferredPromise(regs)
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Type: xs.Parallel,
		States: xs.States{
			{
				Key:     "p1",
				Initial: "running",
				States: xs.States{
					{
						Key:    "running",
						Invoke: []xs.InvokeConfig{{Logic: p, ID: "p"}},
						On:     map[string]xs.Transitions{"CANCEL_1": {{Target: "canceled"}}},
					},
					{Key: "canceled"},
				},
			},
			{
				Key:     "p2",
				Initial: "running",
				States: xs.States{
					{
						Key:    "running",
						Invoke: []xs.InvokeConfig{{Logic: p, ID: "p", OnDone: xs.Transitions{{Target: "done"}}}},
					},
					{Key: "done"},
				},
			},
		},
	})

	// JS pushes to deferredList synchronously inside the promise creator,
	// which runs when each child actor STARTS (promise.ts start). Go promise
	// bodies run on their own goroutines, so start order is recovered from the
	// `xstate.init` @xstate.event each actor emits on start (createActor.ts:545).
	var mu sync.Mutex
	var creationOrder []string
	actor := xs.CreateActor(machine, xs.WithInspect(func(ev xs.InspectionEvent) {
		if ev.Type == xs.InspectEvent && ev.Event.EventType() == "xstate.init" {
			mu.Lock()
			creationOrder = append(creationOrder, ev.ActorRef.SessionID())
			mu.Unlock()
		}
	})).Start()

	bySession := map[string]*actorLogic1Deferred{}
	for range 2 {
		d := actorLogic1Recv(t, regs)
		bySession[d.self.SessionID()] = d
	}
	var deferredList []*actorLogic1Deferred
	mu.Lock()
	for _, sid := range creationOrder {
		if d, ok := bySession[sid]; ok {
			deferredList = append(deferredList, d)
		}
	}
	mu.Unlock()
	require.Len(t, deferredList, 2)

	p1Deferred := deferredList[0]
	p2Deferred := deferredList[1]

	actor.Send(xs.Ev("CANCEL_1"))
	p1Deferred.resolve <- 42
	p2Deferred.resolve <- 42

	w1 := xs.WaitFor(context.Background(), actor, func(s *xs.MachineSnapshot[any]) bool { return s.Matches("p1.canceled") })
	w2 := xs.WaitFor(context.Background(), actor, func(s *xs.MachineSnapshot[any]) bool { return s.Matches("p2.done") })
	_, err := w1.Wait()
	require.NoError(t, err)
	_, err = w2.Wait()
	require.NoError(t, err)

	assert.Error(t, p1Deferred.ctx.Err(), "p1Fn should have been called")
	assert.NoError(t, p2Deferred.ctx.Err(), "p2Fn should not have been called")
}

// JS: promise logic (fromPromise) > should not reuse the same signal for the same actor when restarted
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actorLogic.test.ts#L416
func TestActorLogic_FromPromise_ShouldNotReuseSignalForSameActorWhenRestarted(t *testing.T) {
	regs := make(chan *actorLogic1Deferred, 2)
	p := actorLogic1DeferredPromise(regs)
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "running",
		States: xs.States{
			{
				Key:    "running",
				Invoke: []xs.InvokeConfig{{Logic: p, ID: "p", OnDone: xs.Transitions{{Target: "done"}}}},
				On:     map[string]xs.Transitions{"cancel": {{Target: "canceled"}}},
			},
			{Key: "done", On: map[string]xs.Transitions{"restart": {{Target: "running"}}}},
			{Key: "canceled", On: map[string]xs.Transitions{"restart": {{Target: "running"}}}},
		},
	})
	actor := xs.CreateActor(machine).Start()
	waitMatches := func(state string) {
		t.Helper()
		_, err := xs.WaitFor(context.Background(), actor, func(s *xs.MachineSnapshot[any]) bool { return s.Matches(state) }).Wait()
		require.NoError(t, err)
	}

	// resolve the first promise and no canceling
	waitMatches("running")
	deferred1 := actorLogic1Recv(t, regs)
	deferred1.resolve <- 42
	waitMatches("done")
	assert.NoError(t, deferred1.ctx.Err(), "fn1 should not have been called")

	actor.Send(xs.Ev("restart"))

	// cancel while running
	waitMatches("running")
	actor.Send(xs.Ev("cancel"))
	waitMatches("canceled")

	deferred2 := actorLogic1Recv(t, regs)
	deferred2.resolve <- 42
	assert.Error(t, deferred2.ctx.Err(), "fn2 should have been called")
}

// ---- transition function logic (fromTransition) ----

// JS: transition function logic (fromTransition) > should interpret a transition function
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actorLogic.test.ts#L478
func TestActorLogic_FromTransition_ShouldInterpretATransitionFunction(t *testing.T) {
	type state struct{ Enabled string }
	transitionLogic := xs.FromTransition(
		func(s state, e xs.Event, _ *xs.ActorScope) state {
			if e.EventType() == "toggle" {
				if s.Enabled == "on" {
					s.Enabled = "off"
				} else {
					s.Enabled = "on"
				}
				return s
			}
			return s
		},
		func(xs.TransitionInitArgs) state { return state{Enabled: "on"} },
	)

	actor := xs.CreateActor(transitionLogic).Start()

	assert.Equal(t, "on", actor.GetSnapshot().Context.Enabled)

	actor.Send(xs.Ev("toggle"))

	assert.Equal(t, "off", actor.GetSnapshot().Context.Enabled)
}

// JS: transition function logic (fromTransition) > should persist a transition function
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actorLogic.test.ts#L502
func TestActorLogic_FromTransition_ShouldPersistATransitionFunction(t *testing.T) {
	type state struct{ Enabled string }
	logic := xs.FromTransition(
		func(s state, e xs.Event, _ *xs.ActorScope) state {
			if e.EventType() == "activate" {
				return state{Enabled: "on"}
			}
			return s
		},
		func(xs.TransitionInitArgs) state { return state{Enabled: "off"} },
	)
	actor := xs.CreateActor(logic).Start()
	actor.Send(xs.Ev("activate"))
	persistedSnapshot := actor.GetPersistedSnapshot()

	assert.Equal(t, &xs.TransitionSnapshot[state]{
		Status:  xs.StatusActive,
		Output:  nil,
		Error:   nil,
		Context: state{Enabled: "on"},
	}, persistedSnapshot)

	restoredActor := xs.CreateActor(logic, xs.WithSnapshot(persistedSnapshot))

	restoredActor.Start()

	assert.Equal(t, "on", restoredActor.GetSnapshot().Context.Enabled)
}

// JS: transition function logic (fromTransition) > should have access to the system
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actorLogic.test.ts#L534
func TestActorLogic_FromTransition_ShouldHaveAccessToTheSystem(t *testing.T) {
	probe := newActorLogic1Probe()
	transitionLogic := xs.FromTransition(
		func(_ int, _ xs.Event, scope *xs.ActorScope) int {
			probe.Record(scope.System)
			return 42
		},
		func(xs.TransitionInitArgs) int { return 0 },
	)

	actor := xs.CreateActor(transitionLogic).Start()

	actor.Send(xs.Ev("a"))

	vals := probe.Values(t)
	require.Len(t, vals, 1) // expect.assertions(1)
	assert.NotNil(t, vals[0])
}

// JS: transition function logic (fromTransition) > should have reference to self
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actorLogic.test.ts#L546
func TestActorLogic_FromTransition_ShouldHaveReferenceToSelf(t *testing.T) {
	probe := newActorLogic1Probe()
	transitionLogic := xs.FromTransition(
		func(_ int, _ xs.Event, scope *xs.ActorScope) int {
			probe.Record(scope.Self)
			return 42
		},
		func(xs.TransitionInitArgs) int { return 0 },
	)

	actor := xs.CreateActor(transitionLogic).Start()

	actor.Send(xs.Ev("a"))

	vals := probe.Values(t)
	require.Len(t, vals, 1) // expect.assertions(1)
	assert.NotNil(t, vals[0])
}

// ---- observable logic (fromObservable) ----

// JS: observable logic (fromObservable) > should interpret an observable
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actorLogic.test.ts#L560
func TestActorLogic_FromObservable_ShouldInterpretAnObservable(t *testing.T) {
	observableLogic := xs.FromObservable(func(xs.ObservableArgs) xs.Subscribable[int] {
		return rxTake(rxInterval(10), 4)
	})

	actor := xs.CreateActor(observableLogic).Start()

	snapshot, err := xs.WaitFor(context.Background(), actor, func(s *xs.ObservableSnapshot[int]) bool {
		return s.Status == xs.StatusDone
	}).Wait()
	require.NoError(t, err)

	assert.Equal(t, 3, snapshot.Context)
}

// JS: observable logic (fromObservable) > should resolve
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actorLogic.test.ts#L570
func TestActorLogic_FromObservable_ShouldResolve(t *testing.T) {
	actor := xs.CreateActor(xs.FromObservable(func(xs.ObservableArgs) xs.Subscribable[int] { return rxOf(42) }))
	s := newSpy()

	actor.SubscribeNext(func(snapshot *xs.ObservableSnapshot[int]) { s.Call(snapshot.Context) })

	actor.Start()

	assert.Contains(t, s.Calls(), []any{42})
}

// JS: observable logic (fromObservable) > should resolve (observer .next)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actorLogic.test.ts#L581
func TestActorLogic_FromObservable_ShouldResolveObserverNext(t *testing.T) {
	actor := xs.CreateActor(xs.FromObservable(func(xs.ObservableArgs) xs.Subscribable[int] { return rxOf(42) }))
	s := newSpy()

	actor.Subscribe(xs.Observer[*xs.ObservableSnapshot[int]]{
		Next: func(snapshot *xs.ObservableSnapshot[int]) { s.Call(snapshot.Context) },
	})

	actor.Start()
	assert.Contains(t, s.Calls(), []any{42})
}

// JS: observable logic (fromObservable) > should reject (observer .error)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actorLogic.test.ts#L593
func TestActorLogic_FromObservable_ShouldRejectObserverError(t *testing.T) {
	actor := xs.CreateActor(xs.FromObservable(func(xs.ObservableArgs) xs.Subscribable[int] {
		return rxThrowError[int]("Observable error.")
	}))
	s := newSpy()

	actor.Subscribe(xs.Observer[*xs.ObservableSnapshot[int]]{
		Error: func(err any) { s.Call(err) },
	})

	actor.Start()
	assert.Equal(t, [][]any{{"Observable error."}}, s.Calls())
}

// JS: observable logic (fromObservable) > should complete (observer .complete)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actorLogic.test.ts#L613
func TestActorLogic_FromObservable_ShouldCompleteObserverComplete(t *testing.T) {
	actor := xs.CreateActor(xs.FromObservable(func(xs.ObservableArgs) xs.Subscribable[int] { return rxEmpty[int]() }))
	s := newSpy()

	actor.Subscribe(xs.Observer[*xs.ObservableSnapshot[int]]{
		Complete: func() { s.Call() },
	})

	actor.Start()

	assert.GreaterOrEqual(t, s.Count(), 1)
}

// JS: observable logic (fromObservable) > should not execute when reading initial state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actorLogic.test.ts#L626
func TestActorLogic_FromObservable_ShouldNotExecuteWhenReadingInitialState(t *testing.T) {
	var called atomic.Bool
	logic := xs.FromObservable(func(xs.ObservableArgs) xs.Subscribable[int] {
		called.Store(true)
		return rxEmpty[int]()
	})

	actor := xs.CreateActor(logic)

	actor.GetSnapshot()

	assert.False(t, called.Load())
}

// JS: observable logic (fromObservable) > should have access to the system
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actorLogic.test.ts#L640
func TestActorLogic_FromObservable_ShouldHaveAccessToTheSystem(t *testing.T) {
	probe := newActorLogic1Probe()
	observableLogic := xs.FromObservable(func(a xs.ObservableArgs) xs.Subscribable[int] {
		probe.Record(a.System)
		return rxOf(42)
	})

	xs.CreateActor(observableLogic).Start()

	vals := probe.Values(t)
	require.Len(t, vals, 1) // expect.assertions(1)
	assert.NotNil(t, vals[0])
}

// JS: observable logic (fromObservable) > should have reference to self
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actorLogic.test.ts#L650
func TestActorLogic_FromObservable_ShouldHaveReferenceToSelf(t *testing.T) {
	probe := newActorLogic1Probe()
	observableLogic := xs.FromObservable(func(a xs.ObservableArgs) xs.Subscribable[int] {
		probe.Record(a.Self)
		return rxOf(42)
	})

	xs.CreateActor(observableLogic).Start()

	vals := probe.Values(t)
	require.Len(t, vals, 1) // expect.assertions(1)
	assert.NotNil(t, vals[0])
}

// ---- eventObservable logic (fromEventObservable) ----

// JS: eventObservable logic (fromEventObservable) > should have access to the system
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actorLogic.test.ts#L662
func TestActorLogic_FromEventObservable_ShouldHaveAccessToTheSystem(t *testing.T) {
	probe := newActorLogic1Probe()
	observableLogic := xs.FromEventObservable(func(a xs.ObservableArgs) xs.Subscribable[xs.Event] {
		probe.Record(a.System)
		return rxOf[xs.Event](xs.Ev("a"))
	})

	xs.CreateActor(observableLogic).Start()

	vals := probe.Values(t)
	require.Len(t, vals, 1) // expect.assertions(1)
	assert.NotNil(t, vals[0])
}

// JS: eventObservable logic (fromEventObservable) > should have reference to self
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actorLogic.test.ts#L672
func TestActorLogic_FromEventObservable_ShouldHaveReferenceToSelf(t *testing.T) {
	probe := newActorLogic1Probe()
	observableLogic := xs.FromEventObservable(func(a xs.ObservableArgs) xs.Subscribable[xs.Event] {
		probe.Record(a.Self)
		return rxOf[xs.Event](xs.Ev("a"))
	})

	xs.CreateActor(observableLogic).Start()

	vals := probe.Values(t)
	require.Len(t, vals, 1) // expect.assertions(1)
	assert.NotNil(t, vals[0])
}

// ---- callback logic (fromCallback) ----

// JS: callback logic (fromCallback) > should interpret a callback
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actorLogic.test.ts#L684
func TestActorLogic_FromCallback_ShouldInterpretACallback(t *testing.T) {
	probe := newActorLogic1Probe()
	callbackLogic := xs.FromCallback(func(a xs.CallbackArgs) func() {
		a.Receive(func(event xs.Event) {
			probe.Record(event)
		})
		return nil
	})

	actor := xs.CreateActor(callbackLogic).Start()

	actor.Send(xs.Ev("a"))

	vals := probe.Values(t)
	require.Len(t, vals, 1) // expect.assertions(1)
	assert.Equal(t, xs.Ev("a"), vals[0])
}

// JS: callback logic (fromCallback) > should have access to the system
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actorLogic.test.ts#L698
func TestActorLogic_FromCallback_ShouldHaveAccessToTheSystem(t *testing.T) {
	probe := newActorLogic1Probe()
	callbackLogic := xs.FromCallback(func(a xs.CallbackArgs) func() {
		probe.Record(a.System)
		return nil
	})

	xs.CreateActor(callbackLogic).Start()

	vals := probe.Values(t)
	require.Len(t, vals, 1) // expect.assertions(1)
	assert.NotNil(t, vals[0])
}

// JS: callback logic (fromCallback) > should have reference to self
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actorLogic.test.ts#L707
func TestActorLogic_FromCallback_ShouldHaveReferenceToSelf(t *testing.T) {
	probe := newActorLogic1Probe()
	callbackLogic := xs.FromCallback(func(a xs.CallbackArgs) func() {
		probe.Record(a.Self)
		return nil
	})

	xs.CreateActor(callbackLogic).Start()

	vals := probe.Values(t)
	require.Len(t, vals, 1) // expect.assertions(1)
	assert.NotNil(t, vals[0])
}

// JS: callback logic (fromCallback) > can send self reference in an event to parent
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actorLogic.test.ts#L716
func TestActorLogic_FromCallback_CanSendSelfReferenceInAnEventToParent(t *testing.T) {
	sig := newSignal()
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Invoke: []xs.InvokeConfig{{
			Logic: xs.FromCallback(func(a xs.CallbackArgs) func() {
				a.Receive(func(event xs.Event) {
					switch event.EventType() {
					case "PONG":
						sig.Resolve()
					}
				})

				a.SendBack(xs.E{"type": "PING", "ref": a.Self})
				return nil
			}),
		}},
		On: map[string]xs.Transitions{
			"PING": {{Actions: xs.Actions{xs.SendTo(
				xs.NewExpr(func(a xs.ExprArgs[any]) any { return a.Event.(xs.E)["ref"] }),
				xs.NewExpr(func(xs.ExprArgs[any]) any { return xs.Ev("PONG") }),
			)}}},
		},
	})

	xs.CreateActor(machine).Start()
	sig.Wait(t)
}

// JS: callback logic (fromCallback) > should persist the input of a callback
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actorLogic.test.ts#L752
func TestActorLogic_FromCallback_ShouldPersistTheInputOfACallback(t *testing.T) {
	s := newSpy()
	machine := xs.CreateMachine(
		xs.MachineConfig[any]{
			Initial: "a",
			States: xs.States{
				{Key: "a", On: map[string]xs.Transitions{"EV": {{Target: "b"}}}},
				{Key: "b", Invoke: []xs.InvokeConfig{{
					Src:   "cb",
					Input: xs.NewExpr(func(a xs.ExprArgs[any]) any { return a.Event.(xs.E)["data"] }),
				}}},
			},
		},
		xs.Implementations{Actors: map[string]xs.ActorLogic{
			"cb": xs.FromCallback(func(a xs.CallbackArgs) func() {
				s.Call(a.Input)
				return nil
			}),
		}},
	)

	actor := xs.CreateActor(machine)
	actor.Start()
	actor.Send(xs.E{"type": "EV", "data": 13})

	snapshot := actor.GetPersistedSnapshot()

	actor.Stop()

	cleared := s.Count() // spy.mockClear()

	restoredActor := xs.CreateActor(machine, xs.WithSnapshot(snapshot))

	restoredActor.Start()

	calls := s.Calls()[cleared:]
	assert.Len(t, calls, 1)
	assert.Contains(t, calls, []any{13})
}

// ---- machine logic ----

// JS: machine logic > should persist a machine
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actorLogic.test.ts#L804
func TestActorLogic_MachineLogic_ShouldPersistAMachine(t *testing.T) {
	type childCtx struct{ Count int }
	childMachine := xs.CreateMachine(xs.MachineConfig[childCtx]{
		Context: childCtx{Count: 55},
		Initial: "start",
		States: xs.States{
			{Key: "start", Invoke: []xs.InvokeConfig{{
				ID: "reducer",
				Logic: xs.FromTransition(
					func(s any, _ xs.Event, _ *xs.ActorScope) any { return s },
					func(xs.TransitionInitArgs) any { return nil },
				),
			}}},
		},
	})

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "waiting",
		Invoke: []xs.InvokeConfig{
			{
				ID:     "a",
				Logic:  xs.FromPromise(func(_ context.Context, _ xs.PromiseArgs) (int, error) { return 42, nil }),
				OnDone: xs.Transitions{{Actions: xs.Actions{xs.Raise(xs.Ev("done"))}}},
			},
			{ID: "b", Logic: childMachine},
		},
		States: xs.States{
			{Key: "waiting", On: map[string]xs.Transitions{"done": {{Target: "success"}}}},
			{Key: "success"},
		},
	})

	actor := xs.CreateActor(machine).Start()

	_, err := xs.WaitFor(context.Background(), actor, func(s *xs.MachineSnapshot[any]) bool {
		return s.Matches("success")
	}).Wait()
	require.NoError(t, err)

	persistedState := actor.GetPersistedSnapshot()

	assert.Equal(t, &xs.PromiseSnapshot[int]{
		Error:  nil,
		Input:  nil,
		Output: 42,
		Status: xs.StatusDone,
	}, actorLogic1Field(t, persistedState, "children", "a", "snapshot"))

	// expect.objectContaining({ context, value, children: { reducer: objectContaining({ snapshot: { status: 'active' } }) } })
	b := actorLogic1Field(t, persistedState, "children", "b", "snapshot")
	assert.Equal(t, childCtx{Count: 55}, actorLogic1Field(t, b, "context"))
	assert.Equal(t, "start", actorLogic1Field(t, b, "value"))
	bChildren, ok := actorLogic1Field(t, b, "children").(map[string]any)
	require.True(t, ok)
	assert.Len(t, bChildren, 1)
	assert.Equal(t, &xs.TransitionSnapshot[any]{Status: xs.StatusActive},
		actorLogic1Field(t, bChildren, "reducer", "snapshot"))
}

// JS: machine logic > should persist and restore a nested machine
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actorLogic.test.ts#L877
func TestActorLogic_MachineLogic_ShouldPersistAndRestoreANestedMachine(t *testing.T) {
	childMachine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{"NEXT": {{Target: "b"}}}},
			{Key: "b", On: map[string]xs.Transitions{"LAST": {{Target: "c"}}}},
			{Key: "c"},
		},
	})

	parentMachine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "idle",
		States: xs.States{
			{Key: "idle", On: map[string]xs.Transitions{"START": {{Target: "invoked"}}}},
			{
				Key:    "invoked",
				Invoke: []xs.InvokeConfig{{ID: "child", Logic: childMachine}},
				On: map[string]xs.Transitions{
					"NEXT": {{Actions: xs.Actions{xs.SendTo("child", xs.Ev("NEXT"))}}},
					"LAST": {{Actions: xs.Actions{xs.SendTo("child", xs.Ev("LAST"))}}},
				},
			},
		},
	})

	actor := xs.CreateActor(parentMachine).Start()

	// parent is at 'idle'
	actor.Send(xs.Ev("START"))
	// parent is at 'invoked'; child is at 'a'
	actor.Send(xs.Ev("NEXT"))
	// child is at 'b'

	persistedSnapshot := actor.GetPersistedSnapshot()
	newActor := xs.CreateActor(parentMachine, xs.WithSnapshot(persistedSnapshot)).Start()
	newSnapshot := newActor.GetSnapshot()

	assert.Equal(t, "b", machineSnap[any](newSnapshot.Children["child"]).Value)

	// Ensure that the child actor is started
	// LAST is sent to parent which sends LAST to child
	newActor.Send(xs.Ev("LAST"))
	// child is at 'c'

	assert.Equal(t, "c", machineSnap[any](newActor.GetSnapshot().Children["child"]).Value)
}

// JS: machine logic > should return the initial persisted state of a non-started actor
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actorLogic.test.ts#L947
func TestActorLogic_MachineLogic_ShouldReturnInitialPersistedStateOfNonStartedActor(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "idle",
		States:  xs.States{{Key: "idle"}},
	})

	actor := xs.CreateActor(machine)

	// expect.objectContaining({ value: 'idle' })
	assert.Equal(t, "idle", actorLogic1Field(t, actor.GetPersistedSnapshot(), "value"))
}

// JS: machine logic > the initial state of a child is available before starting the parent
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actorLogic.test.ts#L964
func TestActorLogic_MachineLogic_InitialStateOfChildAvailableBeforeStartingParent(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Invoke: []xs.InvokeConfig{{
			ID: "child",
			Logic: xs.CreateMachine(xs.MachineConfig[any]{
				Initial: "inner",
				States:  xs.States{{Key: "inner"}},
			}),
		}},
	})

	actor := xs.CreateActor(machine)

	// expect.objectContaining({ value: 'inner' })
	assert.Equal(t, "inner", actorLogic1Field(t, actor.GetPersistedSnapshot(), "children", "child", "snapshot", "value"))
}

// JS: machine logic > should not invoke an actor if it is missing in persisted state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actorLogic.test.ts#L986
func TestActorLogic_MachineLogic_ShouldNotInvokeActorIfMissingInPersistedState(t *testing.T) {
	type childCtx struct{ Value string }
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{"NEXT": {{Target: "b"}}}},
			{Key: "b", Invoke: []xs.InvokeConfig{{
				ID: "child",
				Logic: xs.CreateMachine(xs.MachineConfig[childCtx]{
					ContextFn: func(a xs.ContextArgs) childCtx {
						// this is only meant to showcase why we can't invoke this actor when it's missing in the persisted state
						// because we don't have access to the right input as it depends on the event that was used to enter state `b`
						deep := a.Input.(map[string]any)["deep"].(map[string]any)
						return childCtx{Value: deep["prop"].(string)}
					},
				}),
				Input: xs.NewExpr(func(a xs.ExprArgs[any]) any { return a.Event.(xs.E)["data"] }),
			}}},
		},
	})

	actor := xs.CreateActor(machine).Start()

	actor.Send(xs.E{
		"type": "NEXT",
		"data": map[string]any{
			"deep": map[string]any{
				"prop": "value",
			},
		},
	})

	assert.NotNil(t, actor.GetSnapshot().Children["child"])
	assert.Equal(t, childCtx{Value: "value"}, machineSnap[childCtx](actor.GetSnapshot().Children["child"]).Context)

	persisted := actor.GetPersistedSnapshot()

	persistedChildren, ok := actorLogic1Field(t, persisted, "children").(map[string]any)
	require.True(t, ok)
	delete(persistedChildren, "child")

	rehydratedActor := xs.CreateActor(machine, xs.WithSnapshot(persisted)).Start()

	assert.Nil(t, rehydratedActor.GetSnapshot().Children["child"])
}

// JS: machine logic > should persist a spawned actor with referenced src
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actorLogic.test.ts#L1038
func TestActorLogic_MachineLogic_ShouldPersistASpawnedActorWithReferencedSrc(t *testing.T) {
	type reducerState struct{ Count int }
	type ctx struct{ Ref xs.ActorRef }
	reducer := xs.FromTransition(
		func(s reducerState, _ xs.Event, _ *xs.ActorScope) reducerState { return s },
		func(xs.TransitionInitArgs) reducerState { return reducerState{Count: 42} },
	)
	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		ContextFn: func(a xs.ContextArgs) ctx {
			return ctx{Ref: a.Spawn("reducer", xs.SpawnOptions{ID: "child"})}
		},
	}).Provide(xs.Implementations{
		Actors: map[string]xs.ActorLogic{"reducer": reducer},
	})

	actor := xs.CreateActor(machine).Start()

	persistedSnapshot := actor.GetPersistedSnapshot()

	childSnapshot, ok := actorLogic1Field(t, persistedSnapshot, "children", "child", "snapshot").(*xs.TransitionSnapshot[reducerState])
	require.True(t, ok)
	assert.Equal(t, reducerState{Count: 42}, childSnapshot.Context)

	newActor := xs.CreateActor(machine, xs.WithSnapshot(persistedSnapshot)).Start()

	snapshot := newActor.GetSnapshot()

	assert.Same(t, snapshot.Children["child"], snapshot.Context.Ref)

	assert.Equal(t, 42, xs.As[*xs.TransitionSnapshot[reducerState]](snapshot.Context.Ref).GetSnapshot().Context.Count)
}

// JS: machine logic > should not persist a spawned actor with inline src
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actorLogic.test.ts#L1079
func TestActorLogic_MachineLogic_ShouldNotPersistASpawnedActorWithInlineSrc(t *testing.T) {
	type ctx struct{ ChildRef xs.ActorRef }
	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		ContextFn: func(a xs.ContextArgs) ctx {
			return ctx{ChildRef: a.Spawn(xs.CreateMachine(xs.MachineConfig[any]{}))}
		},
	})

	actorRef := xs.CreateActor(machine).Start()

	assert.PanicsWithError(t, "An inline child actor cannot be persisted.", func() {
		actorRef.GetPersistedSnapshot()
	})
}

// JS: machine logic > should have access to the system
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actorLogic.test.ts#L1097
func TestActorLogic_MachineLogic_ShouldHaveAccessToTheSystem(t *testing.T) {
	probe := newActorLogic1Probe()
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Entry: xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[any]) {
			probe.Record(a.System)
		})},
	})

	xs.CreateActor(machine).Start()

	vals := probe.Values(t)
	require.Len(t, vals, 1) // expect.assertions(1)
	assert.NotNil(t, vals[0])
}

// ---- composable actor logic ----

// JS: composable actor logic > should work with machines
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actorLogic.test.ts#L1110
func TestActorLogic_Composable_ShouldWorkWithMachines(t *testing.T) {
	var mu sync.Mutex
	logs := []string{}

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{"to_b": {{Target: "b"}}}},
			{Key: "b", On: map[string]xs.Transitions{"to_c": {{Target: "c"}}}},
			{Key: "c", On: map[string]xs.Transitions{"to_a": {{Target: "a"}}}},
		},
	})

	// withLogs(machine): { ...actorLogic, transition: ... }
	inner := xs.LogicOf[*xs.MachineSnapshot[any]](machine)
	withLogs := *inner
	withLogs.Transition = func(s *xs.MachineSnapshot[any], e xs.Event, scope *xs.ActorScope) *xs.MachineSnapshot[any] {
		mu.Lock()
		logs = append(logs, e.EventType())
		mu.Unlock()
		return inner.Transition(s, e, scope)
	}

	actor := xs.CreateActor(&withLogs).Start()

	actor.Send(xs.Ev("to_b"))
	actor.Send(xs.Ev("to_c"))
	actor.Send(xs.Ev("to_a"))

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, []string{"to_b", "to_c", "to_a"}, logs)
}

// JS: composable actor logic > should work with promises
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actorLogic.test.ts#L1148
func TestActorLogic_Composable_ShouldWorkWithPromises(t *testing.T) {
	var mu sync.Mutex
	logs := []any{}

	promiseLogic := xs.FromPromise(func(_ context.Context, _ xs.PromiseArgs) (int, error) { return 42, nil })

	inner := xs.LogicOf[*xs.PromiseSnapshot[int]](promiseLogic)
	withLogs := *inner
	withLogs.Transition = func(state *xs.PromiseSnapshot[int], e xs.Event, scope *xs.ActorScope) *xs.PromiseSnapshot[int] {
		s := inner.Transition(state, e, scope)
		mu.Lock()
		logs = append(logs, s.Output)
		mu.Unlock()
		return s
	}

	actor := xs.CreateActor(&withLogs).Start()

	_, err := xs.WaitFor(context.Background(), actor, func(s *xs.PromiseSnapshot[int]) bool {
		return s.Status == xs.StatusDone
	}).Wait()
	require.NoError(t, err)

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, []any{42}, logs)
}

// JS: composable actor logic > should work with functions
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actorLogic.test.ts#L1172
func TestActorLogic_Composable_ShouldWorkWithFunctions(t *testing.T) {
	var mu sync.Mutex
	logs := []any{}

	transitionLogic := xs.FromTransition(
		func(_ int, ev xs.Event, _ *xs.ActorScope) int { return ev.(xs.E)["value"].(int) },
		func(xs.TransitionInitArgs) int { return 0 },
	)

	inner := xs.LogicOf[*xs.TransitionSnapshot[int]](transitionLogic)
	withLogs := *inner
	withLogs.Transition = func(state *xs.TransitionSnapshot[int], e xs.Event, scope *xs.ActorScope) *xs.TransitionSnapshot[int] {
		s := inner.Transition(state, e, scope)
		mu.Lock()
		logs = append(logs, s.Context)
		mu.Unlock()
		return s
	}

	actor := xs.CreateActor(&withLogs).Start()

	actor.Send(xs.E{"type": "a", "value": 42})

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, []any{42}, logs)
}

// JS: composable actor logic > should work with observables
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actorLogic.test.ts#L1199
func TestActorLogic_Composable_ShouldWorkWithObservables(t *testing.T) {
	var mu sync.Mutex
	logs := []any{}

	observableLogic := xs.FromObservable(func(xs.ObservableArgs) xs.Subscribable[int] {
		return rxTake(rxInterval(10), 4)
	})

	inner := xs.LogicOf[*xs.ObservableSnapshot[int]](observableLogic)
	withLogs := *inner
	withLogs.Transition = func(state *xs.ObservableSnapshot[int], e xs.Event, scope *xs.ActorScope) *xs.ObservableSnapshot[int] {
		s := inner.Transition(state, e, scope)

		if s.Status == xs.StatusActive {
			mu.Lock()
			logs = append(logs, s.Context)
			mu.Unlock()
		}

		return s
	}

	actor := xs.CreateActor(&withLogs).Start()

	completed := make(chan []any, 1)
	actor.Subscribe(xs.Observer[*xs.ObservableSnapshot[int]]{
		Complete: func() {
			mu.Lock()
			got := append([]any(nil), logs...)
			mu.Unlock()
			completed <- got
		},
	})

	assert.Equal(t, []any{0, 1, 2, 3}, actorLogic1Recv(t, completed))
}

// JS: composable actor logic > higher-level logic wrapping a machine should be able to persist a snapshot
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actorLogic.test.ts#L1231
func TestActorLogic_Composable_HigherLevelLogicWrappingMachineCanPersistSnapshot(t *testing.T) {
	var mu sync.Mutex
	logged := []string{}

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "start",
		States: xs.States{
			{Key: "start", On: map[string]xs.Transitions{"next": {{Target: "working"}}}},
			{Key: "working", On: map[string]xs.Transitions{"more": {{Target: "done"}}}},
			{Key: "done"},
		},
	})

	// withLogging(machine): { ...actorLogic, transition: ... }
	inner := xs.LogicOf[*xs.MachineSnapshot[any]](machine)
	enhancedLogic := *inner
	enhancedLogic.Transition = func(s *xs.MachineSnapshot[any], e xs.Event, scope *xs.ActorScope) *xs.MachineSnapshot[any] {
		mu.Lock()
		logged = append(logged, e.EventType())
		mu.Unlock()
		return inner.Transition(s, e, scope)
	}

	actor := xs.CreateActor(&enhancedLogic).Start()

	actor.Send(xs.Ev("next"))
	actor.Send(xs.Ev("more"))

	mu.Lock()
	assert.Equal(t, []string{"next", "more"}, logged)
	mu.Unlock()

	assert.Equal(t, "done", actor.GetSnapshot().Value)

	assert.NotPanics(t, func() {
		actor.GetPersistedSnapshot()
	})

	// expect.objectContaining({ status: 'active', value: 'done' })
	persisted := actor.GetPersistedSnapshot()
	assert.Equal(t, xs.StatusActive, actorLogic1Field(t, persisted, "status"))
	assert.Equal(t, "done", actorLogic1Field(t, persisted, "value"))
}
