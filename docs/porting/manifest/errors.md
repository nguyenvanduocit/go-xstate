> Historical translation record. Chunk tags, API-gap filenames, line numbers,
> and run counts below describe the original porting work. See the current
> [core guide](../core.md) and [architecture](../../ARCHITECTURE.md) for the maintained layout.

## Source section: errors_1

Source: `references/xstate/packages/core/test/errors.test.ts` lines 1-993.
Go file: `xstate/errors_test.go`. JS tests in range: 26. Ported: 26, N/A-type: 0, N/A-runtime: 0, skipped-in-JS: 0.

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| error handling > does not cause an infinite loop when an error is thrown in subscribe | TestErrors_DoesNotCauseInfiniteLoopWhenErrorIsThrownInSubscribe | ported |  |
| error handling > doesn't crash the actor when an error is thrown in subscribe | TestErrors_DoesntCrashActorWhenErrorIsThrownInSubscribe | ported | `mockImplementationOnce` -> panic only on first call; `actor.send(do)` + spy assert moved after waiting for the reported error (JS runs them inside the async global handler) |
| error handling > doesn't notify error listener when an error is thrown in subscribe | TestErrors_DoesntNotifyErrorListenerWhenErrorIsThrownInSubscribe | ported |  |
| error handling > unhandled sync errors thrown when starting a child actor should be reported globally | TestErrors_UnhandledSyncErrorsWhenStartingChildActorShouldBeReportedGlobally | ported |  |
| error handling > unhandled rejection of a promise actor should be reported globally in absence of error listener | TestErrors_UnhandledRejectionOfPromiseActorReportedGloballyWithoutErrorListener | ported |  |
| error handling > unhandled rejection of a promise actor should be reported to the existing error listener of its parent | TestErrors_UnhandledRejectionOfPromiseActorReportedToParentErrorListener | ported | `await sleep(0)` -> wait on a signal resolved by the error observer (promise settles on a goroutine), then assert the exact calls |
| error handling > unhandled rejection of a promise actor should be reported to the existing error listener of its grandparent | TestErrors_UnhandledRejectionOfPromiseActorReportedToGrandparentErrorListener | ported | same as above |
| error handling > handled sync errors thrown when starting a child actor should not be reported globally | TestErrors_HandledSyncErrorsWhenStartingChildActorShouldNotBeReportedGlobally | ported |  |
| error handling > handled sync errors thrown when starting a child actor should be reported globally when not all of its own observers come with an error listener | TestErrors_HandledSyncErrorsChildReportedGloballyWhenNotAllObserversHaveErrorListener | ported |  |
| error handling > handled sync errors thrown when starting a child actor should not be reported globally when all of its own observers come with an error listener | TestErrors_HandledSyncErrorsChildNotReportedGloballyWhenAllObserversHaveErrorListener | ported |  |
| error handling > unhandled sync errors thrown when starting a child actor should be reported twice globally when not all of its own observers come with an error listener and when the root has no error listener of its own | TestErrors_UnhandledSyncErrorsChildReportedTwiceGloballyWhenRootHasNoErrorListener | ported |  |
| error handling > handled sync errors shouldn't notify the error listener | TestErrors_HandledSyncErrorsShouldntNotifyErrorListener | ported |  |
| error handling > unhandled sync errors should notify the root error listener | TestErrors_UnhandledSyncErrorsShouldNotifyRootErrorListener | ported |  |
| error handling > unhandled sync errors should not notify the global listener when the root error listener is present | TestErrors_UnhandledSyncErrorsShouldNotNotifyGlobalListenerWhenRootErrorListenerPresent | ported |  |
| error handling > handled sync errors thrown when starting an actor shouldn't crash the parent | TestErrors_HandledSyncErrorsWhenStartingActorShouldntCrashParent | ported |  |
| error handling > unhandled sync errors thrown when starting an actor should crash the parent | TestErrors_UnhandledSyncErrorsWhenStartingActorShouldCrashParent | ported |  |
| error handling > error thrown by the error listener should be reported globally | TestErrors_ErrorThrownByErrorListenerShouldBeReportedGlobally | ported |  |
| error handling > error should be reported globally if not every observer comes with an error listener | TestErrors_ErrorReportedGloballyIfNotEveryObserverHasErrorListener | ported |  |
| error handling > uncaught error and an error thrown by the error listener should both be reported globally when not every observer comes with an error listener | TestErrors_UncaughtErrorAndErrorListenerErrorBothReportedGlobally | ported |  |
| error handling > error thrown in initial custom entry action should error the actor | TestErrors_ErrorThrownInInitialCustomEntryActionShouldErrorActor | ported |  |
| error handling > error thrown when resolving initial builtin entry action should error the actor immediately | TestErrors_ErrorThrownResolvingInitialBuiltinEntryActionShouldErrorActorImmediately | ported |  |
| error handling > error thrown by a custom entry action when transitioning should error the actor | TestErrors_ErrorThrownByCustomEntryActionWhenTransitioningShouldErrorActor | ported |  |
| error handling > shouldn't execute deferred initial actions that come after an action that errors | TestErrors_ShouldntExecuteDeferredInitialActionsAfterActionThatErrors | ported |  |
| error handling > should error the parent on errored initial state of a child | TestErrors_ShouldErrorParentOnErroredInitialStateOfChild | ported | `fromTransition` + overridden `getInitialSnapshot` expressed as `&xs.Logic[*xs.TransitionSnapshot[any]]` (TransitionLogic has no overridable fields) |
| error handling > should error when a guard throws when transitioning | TestErrors_ShouldErrorWhenGuardThrowsWhenTransitioning | ported |  |
| error handling > actor continues to work normally after emit callback errors | TestErrors_ActorContinuesToWorkNormallyAfterEmitCallbackErrors | ported | `types.emitted` is type-level only; dropped |

## API gaps

None. No `apigap_errors_1.go` was created.

## Ambiguities for review

- Lines 16-38: JS mocks `reportUnhandledError` to dispatch a `window` ErrorEvent asynchronously, and tests install the listener (`installGlobalOnErrorHandler`) after `start()`. Go passes `xs.WithUnhandledErrorHandler(reporter.handle)` to the root `CreateActor` instead. The file-local `errors1Reporter` collects reported errors on a buffered channel; `waitN` waits for N of them with a 2s timeout, and `assertNoneWithin(10)` stands in for the "reject on any global error, resolve after setTimeout 10ms" pattern.
- Lines 371-486: errors reported by a child actor (child observers missing an error listener) are expected to reach the handler set on the ROOT actor. This assumes `WithUnhandledErrorHandler` covers the whole actor system, not just the root actor.
- Lines 391, 427, 463: `Object.values(actorRef.getSnapshot().children)[0]` before `start()` -> `errors1OnlyChild`, which also requires exactly one child.
- Lines 785-787, 810-812, 858-860, 949-952: `toMatchInlineSnapshot('[Error: msg]')` on `snapshot.error` -> the value must be a Go `error` whose `Error()` equals msg. Guard error message (line 950-951) asserted as `"Unable to evaluate guard in transition for event 'NEXT' in state node '(machine).a':\nerror_thrown_in_guard_when_transitioning"` (format from `src/StateNode.ts:459-465`).
- Line 955-991: no global handler is installed in JS; Go also passes none, so the panic from the first `emitted` listener (it stays registered and fires again on the second send) goes to the default unhandled-error handling (logging).
