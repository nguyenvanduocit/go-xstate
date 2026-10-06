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

// waitFor1DoneSpyCtx is the Go stand-in for `signal.addEventListener = spy`:
// listening for cancellation of a context.Context (a goroutine selecting on
// Done(), or context.AfterFunc) requires calling Done(), so every Done() call
// is recorded. Err()/Value() (the `signal.aborted` / `signal.reason` analogs)
// are delegated untouched, so context.Cause still yields the abort reason.
type waitFor1DoneSpyCtx struct {
	context.Context
	doneSpy *spy
}

func (c waitFor1DoneSpyCtx) Done() <-chan struct{} {
	c.doneSpy.Call()
	return c.Context.Done()
}

// waitFor1ABMachine is the machine shared by most tests:
// a --NEXT--> b, with b final when bFinal is true.
func waitFor1ABMachine(bFinal bool) *xs.StateMachine[any] {
	b := xs.StateConfig{Key: "b"}
	if bFinal {
		b.Type = xs.Final
	}
	return xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{"NEXT": {{Target: "b"}}}},
			b,
		},
	})
}

// waitFor1ABCMachine is a --NEXT--> b --NEXT--> c (no final states).
func waitFor1ABCMachine() *xs.StateMachine[any] {
	return xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{"NEXT": {{Target: "b"}}}},
			{Key: "b", On: map[string]xs.Transitions{"NEXT": {{Target: "c"}}}},
			{Key: "c"},
		},
	})
}

// JS: waitFor > should wait for a condition to be true and return the emitted value
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/waitFor.test.ts#L4
func TestWaitFor_ShouldWaitForAConditionToBeTrueAndReturnTheEmittedValue(t *testing.T) {
	service := xs.CreateActor(waitFor1ABMachine(false)).Start()

	time.AfterFunc(ms(10), func() { service.Send(xs.Ev("NEXT")) })

	state, err := xs.WaitFor(context.Background(), service, func(s *xs.MachineSnapshot[any]) bool {
		return s.Matches("b")
	}).Wait()
	require.NoError(t, err)

	assert.Equal(t, "b", state.Value)
}

// JS: waitFor > should throw an error after a timeout
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/waitFor.test.ts#L24
func TestWaitFor_ShouldThrowAnErrorAfterATimeout(t *testing.T) {
	service := xs.CreateActor(waitFor1ABCMachine()).Start()

	_, err := xs.WaitFor(context.Background(), service, func(s *xs.MachineSnapshot[any]) bool {
		return s.Matches("c")
	}, xs.WaitForOptions{Timeout: ms(10)}).Wait()

	// JS: catch (e) { expect(e).toBeInstanceOf(Error) }
	assert.Error(t, err)
}

// JS: waitFor > should not reject immediately when passing Infinity as timeout
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/waitFor.test.ts#L47
func TestWaitFor_ShouldNotRejectImmediatelyWhenPassingInfinityAsTimeout(t *testing.T) {
	service := xs.CreateActor(waitFor1ABCMachine()).Start()

	// timeout: Infinity → zero Timeout (no timeout).
	p := xs.WaitFor(context.Background(), service, func(s *xs.MachineSnapshot[any]) bool {
		return s.Matches("c")
	}, xs.WaitForOptions{Timeout: 0})

	// Promise.race([waitFor(...), sleep(10).then(() => 'timeout')])
	var result any
	select {
	case <-p.Done():
		s, err := p.Wait()
		if err != nil {
			result = err
		} else {
			result = s
		}
	case <-time.After(ms(10)):
		result = "timeout"
	}

	assert.Equal(t, "timeout", result)
	service.Stop()
}

// JS: waitFor > should throw an error when reaching a final state that does not match the predicate
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/waitFor.test.ts#L72
func TestWaitFor_ShouldThrowAnErrorWhenReachingAFinalStateThatDoesNotMatchThePredicate(t *testing.T) {
	service := xs.CreateActor(waitFor1ABMachine(true)).Start()

	time.AfterFunc(ms(10), func() { service.Send(xs.Ev("NEXT")) })

	_, err := xs.WaitFor(context.Background(), service, func(s *xs.MachineSnapshot[any]) bool {
		return s.Matches("never")
	}).Wait()

	assert.EqualError(t, err, "Actor terminated without satisfying predicate")
}

// JS: waitFor > should resolve correctly when the predicate immediately matches the current state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/waitFor.test.ts#L98
func TestWaitFor_ShouldResolveCorrectlyWhenThePredicateImmediatelyMatchesTheCurrentState(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States:  xs.States{{Key: "a"}},
	})

	service := xs.CreateActor(machine).Start()

	state, err := xs.WaitFor(context.Background(), service, func(s *xs.MachineSnapshot[any]) bool {
		return s.Matches("a")
	}).Wait()
	require.NoError(t, err)

	assert.Equal(t, "a", state.Value)
}

// JS: waitFor > should not subscribe when the predicate immediately matches
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/waitFor.test.ts#L113
func TestWaitFor_ShouldNotSubscribeWhenThePredicateImmediatelyMatches(t *testing.T) {
	t.Skip("N/A: JS replaces actorRef.subscribe with a vi.fn() spy on the instance; *xs.Actor methods cannot be replaced in Go and the contract exposes no subscriber count")

	machine := xs.CreateMachine(xs.MachineConfig[any]{})
	actorRef := xs.CreateActor(machine).Start()

	_ = xs.WaitFor(context.Background(), actorRef, func(_ *xs.MachineSnapshot[any]) bool { return true })
	// JS: expect(spy /* actorRef.subscribe */).not.toHaveBeenCalled();
}

// JS: waitFor > should internally unsubscribe when the predicate immediately matches the current state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/waitFor.test.ts#L125
func TestWaitFor_ShouldInternallyUnsubscribeWhenThePredicateImmediatelyMatchesTheCurrentState(t *testing.T) {
	var count atomic.Int32

	service := xs.CreateActor(waitFor1ABMachine(false)).Start()

	_, err := xs.WaitFor(context.Background(), service, func(s *xs.MachineSnapshot[any]) bool {
		count.Add(1)
		return s.Matches("a")
	}).Wait()
	require.NoError(t, err)

	service.Send(xs.Ev("NEXT"))

	assert.Equal(t, int32(1), count.Load())
}

// JS: waitFor > should immediately resolve for an actor in its final state that matches the predicate
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/waitFor.test.ts#L151
func TestWaitFor_ShouldImmediatelyResolveForAnActorInItsFinalStateThatMatchesThePredicate(t *testing.T) {
	service := xs.CreateActor(waitFor1ABMachine(true)).Start()
	service.Send(xs.Ev("NEXT"))

	state, err := xs.WaitFor(context.Background(), service, func(s *xs.MachineSnapshot[any]) bool {
		return s.Matches("b")
	}).Wait()
	require.NoError(t, err)

	assert.Equal(t, "b", state.Value)
}

// JS: waitFor > should immediately reject for an actor in its final state that does not match the predicate
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/waitFor.test.ts#L174
func TestWaitFor_ShouldImmediatelyRejectForAnActorInItsFinalStateThatDoesNotMatchThePredicate(t *testing.T) {
	service := xs.CreateActor(waitFor1ABMachine(true)).Start()
	service.Send(xs.Ev("NEXT"))

	_, err := xs.WaitFor(context.Background(), service, func(s *xs.MachineSnapshot[any]) bool {
		return s.Matches("a")
	}).Wait()

	assert.EqualError(t, err, "Actor terminated without satisfying predicate")
}

// JS: waitFor > should not subscribe to the actor when it receives an aborted signal
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/waitFor.test.ts#L199
func TestWaitFor_ShouldNotSubscribeToTheActorWhenItReceivesAnAbortedSignal(t *testing.T) {
	t.Skip("N/A: JS replaces service.subscribe with a vi.fn() spy on the instance; *xs.Actor methods cannot be replaced in Go and the contract exposes no subscriber count")

	service := xs.CreateActor(waitFor1ABMachine(true)).Start()
	service.Send(xs.Ev("NEXT"))

	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(errors.New("Aborted!"))

	_, _ = xs.WaitFor(ctx, service, func(s *xs.MachineSnapshot[any]) bool {
		return s.Matches("b")
	}).Wait()
	// JS (catch block): expect(spy /* service.subscribe */).not.toHaveBeenCalled();
}

// JS: waitFor > should not listen for the "abort" event when it receives an aborted signal
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/waitFor.test.ts#L230
func TestWaitFor_ShouldNotListenForTheAbortEventWhenItReceivesAnAbortedSignal(t *testing.T) {
	service := xs.CreateActor(waitFor1ABMachine(true)).Start()
	service.Send(xs.Ev("NEXT"))

	parent, cancel := context.WithCancelCause(context.Background())
	cancel(errors.New("Aborted!"))

	addEventListenerSpy := newSpy()
	ctx := waitFor1DoneSpyCtx{Context: parent, doneSpy: addEventListenerSpy}

	// JS awaits inside try/catch and asserts only in the catch block, which
	// runs whether waitFor rejects or resolves (via 'Should not be reached').
	_, _ = xs.WaitFor(ctx, service, func(s *xs.MachineSnapshot[any]) bool {
		return s.Matches("b")
	}).Wait()

	assert.Equal(t, 0, addEventListenerSpy.Count())
}

// JS: waitFor > should not listen for the "abort" event for actor in its final state that matches the predicate
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/waitFor.test.ts#L263
func TestWaitFor_ShouldNotListenForTheAbortEventForActorInItsFinalStateThatMatchesThePredicate(t *testing.T) {
	service := xs.CreateActor(waitFor1ABMachine(true)).Start()
	service.Send(xs.Ev("NEXT"))

	parent, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)

	addEventListenerSpy := newSpy()
	ctx := waitFor1DoneSpyCtx{Context: parent, doneSpy: addEventListenerSpy}

	_, err := xs.WaitFor(ctx, service, func(s *xs.MachineSnapshot[any]) bool {
		return s.Matches("b")
	}).Wait()
	require.NoError(t, err)

	assert.Equal(t, 0, addEventListenerSpy.Count())
}

// JS: waitFor > should immediately reject when it receives an aborted signal
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/waitFor.test.ts#L291
func TestWaitFor_ShouldImmediatelyRejectWhenItReceivesAnAbortedSignal(t *testing.T) {
	service := xs.CreateActor(waitFor1ABMachine(true)).Start()
	service.Send(xs.Ev("NEXT"))

	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(errors.New("Aborted!"))

	_, err := xs.WaitFor(ctx, service, func(s *xs.MachineSnapshot[any]) bool {
		return s.Matches("b")
	}).Wait()

	// rejects.toMatchInlineSnapshot(`[Error: Aborted!]`) — rejects with signal.reason.
	assert.EqualError(t, err, "Aborted!")
}

// JS: waitFor > should reject when the signal is aborted while waiting
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/waitFor.test.ts#L318
func TestWaitFor_ShouldRejectWhenTheSignalIsAbortedWhileWaiting(t *testing.T) {
	service := xs.CreateActor(waitFor1ABMachine(false)).Start()

	ctx, cancel := context.WithCancelCause(context.Background())
	time.AfterFunc(ms(10), func() { cancel(errors.New("Aborted!")) })

	_, err := xs.WaitFor(ctx, service, func(s *xs.MachineSnapshot[any]) bool {
		return s.Matches("b")
	}).Wait()

	// rejects.toMatchInlineSnapshot(`[Error: Aborted!]`) — rejects with signal.reason.
	assert.EqualError(t, err, "Aborted!")
}

// JS: waitFor > should stop listening for the "abort" event upon successful completion
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/waitFor.test.ts#L339
func TestWaitFor_ShouldStopListeningForTheAbortEventUponSuccessfulCompletion(t *testing.T) {
	t.Skip("N/A: JS spies on signal.removeEventListener; context.Context has no removal hook (de-registration is an internal AfterFunc stop()/goroutine exit), so the assertion is not observable through the contract")

	service := xs.CreateActor(waitFor1ABMachine(true)).Start()
	time.AfterFunc(ms(10), func() { service.Send(xs.Ev("NEXT")) })

	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)

	_, err := xs.WaitFor(ctx, service, func(s *xs.MachineSnapshot[any]) bool {
		return s.Matches("b")
	}).Wait()
	require.NoError(t, err)
	// JS: expect(spy /* signal.removeEventListener */).toHaveBeenCalledTimes(1);
}

// JS: waitFor > should stop listening for the "abort" event upon failure
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/waitFor.test.ts#L369
func TestWaitFor_ShouldStopListeningForTheAbortEventUponFailure(t *testing.T) {
	t.Skip("N/A: JS spies on signal.removeEventListener; context.Context has no removal hook (de-registration is an internal AfterFunc stop()/goroutine exit), so the assertion is not observable through the contract")

	service := xs.CreateActor(waitFor1ABMachine(true)).Start()
	time.AfterFunc(ms(10), func() { service.Send(xs.Ev("NEXT")) })

	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)

	_, _ = xs.WaitFor(ctx, service, func(s *xs.MachineSnapshot[any]) bool {
		return s.Matches("never")
	}).Wait()
	// JS (catch block): expect(spy /* signal.removeEventListener */).toHaveBeenCalledTimes(1);
}
