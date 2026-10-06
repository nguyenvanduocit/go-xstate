> Historical translation record. Chunk tags, API-gap filenames, line numbers,
> and run counts below describe the original porting work. See the current
> [core guide](../core.md) and [architecture](../../ARCHITECTURE.md) for the maintained layout.

## Source section: wait_for_1

Source: `references/xstate/packages/core/test/waitFor.test.ts` lines 1-401 (whole file, 400 lines).
Go file: `xstate/wait_for_test.go`.

Totals: 16 JS tests; 12 ported, 0 N/A-type, 4 N/A-runtime, 0 skipped-in-JS.

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| waitFor > should wait for a condition to be true and return the emitted value | TestWaitFor_ShouldWaitForAConditionToBeTrueAndReturnTheEmittedValue | ported | `setTimeout(..., 10)` → `time.AfterFunc(ms(10), ...)`. |
| waitFor > should throw an error after a timeout | TestWaitFor_ShouldThrowAnErrorAfterATimeout | ported | JS asserts `toBeInstanceOf(Error)` only inside `catch`; Go asserts `assert.Error(err)` (see ambiguities). |
| waitFor > should not reject immediately when passing Infinity as timeout | TestWaitFor_ShouldNotRejectImmediatelyWhenPassingInfinityAsTimeout | ported | `timeout: Infinity` → `WaitForOptions{Timeout: 0}` (zero = no timeout per `util.go:22`). `Promise.race` → `select` on `p.Done()` vs `time.After(ms(10))`. |
| waitFor > should throw an error when reaching a final state that does not match the predicate | TestWaitFor_ShouldThrowAnErrorWhenReachingAFinalStateThatDoesNotMatchThePredicate | ported | Inline snapshot `[Error: Actor terminated without satisfying predicate]` → `assert.EqualError`. |
| waitFor > should resolve correctly when the predicate immediately matches the current state | TestWaitFor_ShouldResolveCorrectlyWhenThePredicateImmediatelyMatchesTheCurrentState | ported | `resolves.toHaveProperty('value', 'a')` → NoError + `Equal("a", state.Value)`. |
| waitFor > should not subscribe when the predicate immediately matches | TestWaitFor_ShouldNotSubscribeWhenThePredicateImmediatelyMatches | N/A-runtime | Only assertion is a `vi.fn()` replacing `actorRef.subscribe` on the instance (JS L118); Go methods on `*xs.Actor` cannot be swapped. Setup translated after `t.Skip`. |
| waitFor > should internally unsubscribe when the predicate immediately matches the current state | TestWaitFor_ShouldInternallyUnsubscribeWhenThePredicateImmediatelyMatchesTheCurrentState | ported | `count` is `atomic.Int32` (predicate may run on another goroutine under `-race`). |
| waitFor > should immediately resolve for an actor in its final state that matches the predicate | TestWaitFor_ShouldImmediatelyResolveForAnActorInItsFinalStateThatMatchesThePredicate | ported | |
| waitFor > should immediately reject for an actor in its final state that does not match the predicate | TestWaitFor_ShouldImmediatelyRejectForAnActorInItsFinalStateThatDoesNotMatchThePredicate | ported | `assert.EqualError(..., "Actor terminated without satisfying predicate")`. |
| waitFor > should not subscribe to the actor when it receives an aborted signal | TestWaitFor_ShouldNotSubscribeToTheActorWhenItReceivesAnAbortedSignal | N/A-runtime | Only assertion is the `service.subscribe = spy` monkey-patch (JS L221, L226). Setup translated after `t.Skip`. |
| waitFor > should not listen for the "abort" event when it receives an aborted signal | TestWaitFor_ShouldNotListenForTheAbortEventWhenItReceivesAnAbortedSignal | ported | `AbortController.abort(new Error('Aborted!'))` → `context.WithCancelCause` + `cancel(errors.New("Aborted!"))`. `signal.addEventListener` spy → `waitFor1DoneSpyCtx` counting `Done()` calls. |
| waitFor > should not listen for the "abort" event for actor in its final state that matches the predicate | TestWaitFor_ShouldNotListenForTheAbortEventForActorInItsFinalStateThatMatchesThePredicate | ported | Same `Done()` spy; JS `await` outside try → `require.NoError`. |
| waitFor > should immediately reject when it receives an aborted signal | TestWaitFor_ShouldImmediatelyRejectWhenItReceivesAnAbortedSignal | ported | Rejection is `signal.reason` → `assert.EqualError(err, "Aborted!")` (i.e. `context.Cause(ctx)`). |
| waitFor > should reject when the signal is aborted while waiting | TestWaitFor_ShouldRejectWhenTheSignalIsAbortedWhileWaiting | ported | `setTimeout(() => controller.abort(...), 10)` → `time.AfterFunc(ms(10), func(){ cancel(errors.New("Aborted!")) })`. |
| waitFor > should stop listening for the "abort" event upon successful completion | TestWaitFor_ShouldStopListeningForTheAbortEventUponSuccessfulCompletion | N/A-runtime | Asserts `signal.removeEventListener` called once (JS L362, L366). `context.Context` has no removal hook to observe. Observable part (resolves) translated after `t.Skip`. |
| waitFor > should stop listening for the "abort" event upon failure | TestWaitFor_ShouldStopListeningForTheAbortEventUponFailure | N/A-runtime | Same as above (JS L391, L397). |

## API gaps

None. Uses `xs.WaitFor(ctx, actor, pred, xs.WaitForOptions{...}).Wait()` and `Promise.Done()` (`util.go`). The JS `signal` option is the `ctx` argument, as the contract states (`util.go:27-28`).

## Ambiguities for review

- **Abort reason (JS L219, L311, L332)**: Go tests require the rejection to be `context.Cause(ctx)` (the error passed to `cancel`), not `ctx.Err()` (`context.Canceled`). That mirrors JS rejecting with `signal.reason`.
- **addEventListener → `Done()` spy (JS L253, L285)**: `waitFor1DoneSpyCtx` wraps the ctx and counts `Done()` calls, because any way of listening for cancellation (goroutine `select`, `context.AfterFunc`) calls `Done()`. `Err()`/`Value()` are delegated, so `ctx.Err()` is the `signal.aborted` analog and `context.Cause` still works. The proxy is stricter than JS: an implementation that checks "already aborted" with a non-blocking `select { case <-ctx.Done(): }` instead of `ctx.Err()` fails these tests, while JS reads `signal.aborted` without touching `addEventListener`. The implementation must therefore check `ctx.Err()` first (JS order: aborted check → predicate check → listen).
- **removeEventListener (JS L339-399)**: marked N/A-runtime. Making it observable would need a contract change, such as a custom signal type or a test hook. Using `ctx` was a deliberate contract choice, so this port adds no gap for it.
- **subscribe monkey-patch (JS L113-123, L199-228)**: marked N/A-runtime. A port would need an introspection API, such as a subscriber count on `*xs.Actor`, which the contract does not have. Test L125 still covers the related "no further predicate calls after an immediate match" behaviour.
- **Timeout test (JS L24-45)**: JS asserts only inside `catch`, so a resolved promise would pass silently. Go uses `assert.Error(err)` unconditionally. This is stricter, but the JS semantics guarantee it: state `c` is unreachable without events, so the 10ms timeout must reject. The message `Timeout of 10 ms exceeded` is not asserted in JS and is not asserted here.
- **JS L230 / L199**: the try/catch runs the catch block whether waitFor rejects or resolves (`throw new Error('Should not be reached')` inside try is caught). So the outcome is not asserted, only the spy. Go discards the result the same way.
