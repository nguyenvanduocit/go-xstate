> Historical translation record. Chunk tags, API-gap filenames, line numbers,
> and run counts below describe the original porting work. See the current
> [core guide](../core.md) and [architecture](../../ARCHITECTURE.md) for the maintained layout.

## Source section: interpreter_1

Source: `references/xstate/packages/core/test/interpreter.test.ts` lines 1-1276.
Go file: `xstate/interpreter_test.go`. JS tests in range: 37. Ported: 37, N/A-type: 0, N/A-runtime: 0, skipped-in-JS: 0.

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| interpreter > initial state > .getSnapshot returns the initial state | TestInterpreter_InitialState_GetSnapshotReturnsTheInitialState | ported | |
| interpreter > initial state > initially spawned actors should not be spawned when reading initial state | TestInterpreter_InitialState_InitiallySpawnedActorsNotSpawnedWhenReadingInitialState | ported | Promise executor → FromPromise body that increments an atomic counter and blocks until ctx is cancelled (never settles) |
| interpreter > initial state > does not execute actions from a restored state | TestInterpreter_InitialState_DoesNotExecuteActionsFromARestoredState | ported | |
| interpreter > initial state > should not execute actions that are not part of the actual persisted state | TestInterpreter_InitialState_ShouldNotExecuteActionsNotPartOfActualPersistedState | ported | |
| interpreter > subscribing > should not notify subscribers of the current state upon subscription (subscribe) | TestInterpreter_Subscribing_ShouldNotNotifySubscribersOfCurrentStateUponSubscription | ported | |
| interpreter > send with delay > can send an event after a delay | TestInterpreter_SendWithDelay_CanSendAnEventAfterADelay | ported | |
| interpreter > send with delay > can send an event after a delay (expression) | TestInterpreter_SendWithDelay_CanSendAnEventAfterADelayExpression | ported | |
| interpreter > send with delay > can send an event after a delay (expression using _event) | TestInterpreter_SendWithDelay_CanSendAnEventAfterADelayExpressionUsingEvent | ported | |
| interpreter > send with delay > can send an event after a delay (delayed transitions) | TestInterpreter_SendWithDelay_CanSendAnEventAfterADelayDelayedTransitions | ported | |
| interpreter > activities (deprecated) > should start activities | TestInterpreter_Activities_ShouldStartActivities | ported | `fromCallback(spy)` → callback that calls the spy; `toHaveBeenCalled` → `Count() > 0` |
| interpreter > activities (deprecated) > should stop activities | TestInterpreter_Activities_ShouldStopActivities | ported | |
| interpreter > activities (deprecated) > should stop activities upon stopping the service | TestInterpreter_Activities_ShouldStopActivitiesUponStoppingTheService | ported | |
| interpreter > activities (deprecated) > should restart activities from a compound state | TestInterpreter_Activities_ShouldRestartActivitiesFromACompoundState | ported | |
| interpreter > can cancel a delayed event | TestInterpreter_CanCancelADelayedEvent | ported | |
| interpreter > can cancel a delayed event using expression to resolve send id | TestInterpreter_CanCancelADelayedEventUsingExpressionToResolveSendID | ported | |
| interpreter > should not throw an error if an event is sent to an uninitialized interpreter | TestInterpreter_ShouldNotThrowIfEventSentToUninitializedInterpreter | ported | |
| interpreter > should defer events sent to an uninitialized service | TestInterpreter_ShouldDeferEventsSentToAnUninitializedService | ported | |
| interpreter > should throw an error if initial state sent to interpreter is invalid | TestInterpreter_ShouldThrowAnErrorIfInitialStateSentToInterpreterIsInvalid | ported | Asserts the exact inline-snapshot error message via `error.Error()` |
| interpreter > should not update when stopped | TestInterpreter_ShouldNotUpdateWhenStopped | ported | Gap `WithWarnHandler`; inline snapshot `x:27 (x:27)` built from `SessionID()` |
| interpreter > should be able to log (log action) | TestInterpreter_ShouldBeAbleToLogLogAction | ported | Logger records `args[0]` (JS `(msg) => logs.push(msg)`) |
| interpreter > should receive correct event (log action) | TestInterpreter_ShouldReceiveCorrectEventLogAction | ported | Logger records `args[0]` |
| interpreter > send() event expressions > should resolve send event expressions | TestInterpreter_SendEventExpressions_ShouldResolveSendEventExpressions | ported | |
| interpreter > sendParent() event expressions > should resolve sendParent event expressions | TestInterpreter_SendParentEventExpressions_ShouldResolveSendParentEventExpressions | ported | `typeof child.send === "function"` → `assert.NotNil(child)` |
| interpreter > .send() > can send events with a string | TestInterpreter_Send_CanSendEventsWithAString | ported | |
| interpreter > .send() > can send events with an object | TestInterpreter_Send_CanSendEventsWithAnObject | ported | |
| interpreter > .send() > can send events with an object with payload | TestInterpreter_Send_CanSendEventsWithAnObjectWithPayload | ported | |
| interpreter > .send() > should receive and process all events sent simultaneously | TestInterpreter_Send_ShouldReceiveAndProcessAllEventsSentSimultaneously | ported | |
| interpreter > .start() > should initialize the service | TestInterpreter_Start_ShouldInitializeTheService | ported | `context: contextSpy` → `ContextFn` calling the spy and returning nil |
| interpreter > .start() > should not reinitialize a started service | TestInterpreter_Start_ShouldNotReinitializeAStartedService | ported | |
| interpreter > .start() > should be able to be initialized at a custom state | TestInterpreter_Start_ShouldBeAbleToBeInitializedAtACustomState | ported | |
| interpreter > .start() > should be able to be initialized at a custom state value | TestInterpreter_Start_ShouldBeAbleToBeInitializedAtACustomStateValue | ported | |
| interpreter > .start() > should be able to resolve a custom initialized state | TestInterpreter_Start_ShouldBeAbleToResolveACustomInitializedState | ported | |
| interpreter > .stop() > should cancel delayed events | TestInterpreter_Stop_ShouldCancelDelayedEvents | ported | |
| interpreter > .stop() > should not execute transitions after being stopped | TestInterpreter_Stop_ShouldNotExecuteTransitionsAfterBeingStopped | ported | Gap `WithWarnHandler`; inline snapshot `x:43 (x:43)` built from `SessionID()` |
| interpreter > .stop() > should not throw when sending an unserializable event to a stopped actor | TestInterpreter_Stop_ShouldNotThrowWhenSendingUnserializableEventToStoppedActor | ported | Gap `WithWarnHandler`; circular `xs.E` (map containing itself) |
| interpreter > .stop() > stopping a not-started interpreter should not crash | TestInterpreter_Stop_StoppingANotStartedInterpreterShouldNotCrash | ported | |
| interpreter > .unsubscribe() > should remove transition listeners | TestInterpreter_Unsubscribe_ShouldRemoveTransitionListeners | ported | |

## Shared helpers

- `interpreter1LightMachine()` — top-level `lightMachine` (JS 21-46), a func so `CreateMachine` is not called at package init.
- `interpreter1SendMachine()` — `sendMachine` of the `.send()` describe (JS 933-950).
- The `subscribing` describe-level machine (JS 175-180) and the `send() event expressions` describe-level machine (JS 831-854) are inlined into their only test.

## API gaps (`apigap_interpreter_1.go`)

- `WithWarnHandler(fn func(args ...any)) ActorOption` — mirrors `vi.spyOn(console, 'warn')`. Identical signature to the gaps in `apigap_actions_3.go`, `apigap_actions_4.go`, `apigap_event_descriptors_1.go`, `apigap_history_1.go`; deduplicate at integration.

## Ambiguities for review

- JS 703-728: in JS `createMachine(invalidMachine)` does not throw; the error surfaces as `snapshot.status === 'error'` from `createActor(...).getSnapshot()`. The contract doc on `CreateMachine` says it "panics on an invalid config"; this test requires that an unresolvable `initial` target NOT panic at `CreateMachine` time but produce an error snapshot, matching JS.
- JS 744-754 and 1172-1180: inline snapshots hardcode `x:27 (x:27)` / `x:43 (x:43)` (session-id counter, test-order artifact). The message format is `${actor.id} (${actor.sessionId})` (createActor.ts:743) and the default id is the session id, so Go builds the expected string from `service.SessionID()` twice.
- JS 741-746: the JS `try/catch` around `send` after `stop` is mirrored with a `recover` that asserts the value stays `yellow` only if `Send` panics.
- JS 1191-1221: the circular event becomes an `xs.E` that contains itself; Go's `encoding/json` reports a cycle error rather than overflowing, so the implementation must format the warning without panicking.
- JS 193-220, 611-656, 1117-1147, 1149-1189: real timers (`setTimeout`) → `sleep(n)`; timings kept identical to JS (10/5/10 ms, 100/200 ms, 60 ms, 10 ms), so these can be timing-sensitive under `-race`.
- JS 63-107: `setTimeout(..., 100)` then assert → `sleep(100)` then assert; the spawned promise blocks until its ctx is cancelled (it never settles in JS either).
- JS 893-899: `expect(typeof childActor!.send).toBe('function')` → `assert.NotNil(childActor)`: in Go a non-nil `ActorRef` always has `Send`; a missing child would throw in JS.
- JS 222-289: `'wait' in event ? event.wait : 0` → checks the `"wait"` key of `xs.E`.
- Range boundary: `describe('transient states')` starts at JS line 1277, outside this chunk.


## Source section: interpreter_2

Source: `references/xstate/packages/core/test/interpreter.test.ts` lines 1277-1948 (21 `it` calls; no `it.each`, `it.skip` or `it.todo` in range).
Lines 1277-1862 are inside `describe('interpreter')`; the last 4 tests (JS L1864-1948) are top-level (no describe).
Go file: `xstate/interpreter_test.go`. No API gaps.

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| interpreter > transient states > should transition in correct order | TestInterpreter_TransientStates_ShouldTransitionInCorrectOrder | ported | JS L1278; index guard added so a short slice fails instead of panicking |
| interpreter > transient states > should transition in correct order when there is a condition | TestInterpreter_TransientStates_ShouldTransitionInCorrectOrderWhenThereIsACondition | ported | JS L1302 |
| interpreter > observable > should be subscribable | TestInterpreter_Observable_ShouldBeSubscribable | ported | JS L1371; describe-level `intervalMachine` (JS L1342-1369) -> `interpreter2IntervalMachine()`; `typeof subscribe === 'function'` -> compile-time `var _ xs.Subscribable[...] = intervalService` |
| interpreter > observable > should be interoperable with RxJS, etc. via Symbol.observable | TestInterpreter_Observable_ShouldBeInteroperableWithRxJSViaSymbolObservable | ported | JS L1392; `from(intervalService)` -> actor used through the `xs.Subscribable[S]` interface (Go has no Symbol.observable; interface satisfaction is the interop) |
| interpreter > observable > should be unsubscribable | TestInterpreter_Observable_ShouldBeUnsubscribable | ported | JS L1414 |
| interpreter > observable > should call complete() once a final state is reached | TestInterpreter_Observable_ShouldCallCompleteOnceAFinalStateIsReached | ported | JS L1459 |
| interpreter > observable > should call complete() once the interpreter is stopped | TestInterpreter_Observable_ShouldCallCompleteOnceTheInterpreterIsStopped | ported | JS L1484 |
| interpreter > actors > doesn't crash cryptically on undefined return from the actor creator | TestInterpreter_Actors_DoesntCrashCrypticallyOnUndefinedReturnFromTheActorCreator | ported | JS L1502; callback returns nil cleanup; `not.toThrow()` -> `assert.NotPanics` |
| interpreter > children > state.children should reference invoked child actors (machine) | TestInterpreter_Children_StateChildrenShouldReferenceInvokedChildActorsMachine | ported | JS L1536; `children.childActor.send` -> `require.NotNil` then Send; `not.toHaveProperty` -> `assert.NotContains` |
| interpreter > children > state.children should reference invoked child actors (promise) | TestInterpreter_Children_StateChildrenShouldReferenceInvokedChildActorsPromise | ported | JS L1570; `toHaveProperty('send')` -> `assert.NotNil` (every ActorRef has Send); guard reads `xs.DoneActorEvent.Output == 42` |
| interpreter > children > state.children should reference invoked child actors (observable) | TestInterpreter_Children_StateChildrenShouldReferenceInvokedChildActorsObservable | ported | JS L1637; guard reads `xs.SnapshotEvent.Snapshot.(*xs.ObservableSnapshot[int]).Context == 3` |
| interpreter > children > state.children should reference spawned actors | TestInterpreter_Children_StateChildrenShouldReferenceSpawnedActors | ported | JS L1693 |
| interpreter > children > stopped spawned actors should be cleaned up in parent | TestInterpreter_Children_StoppedSpawnedActorsShouldBeCleanedUpInParent | ported | JS L1717; never-settling `new Promise(() => {})` -> promise blocking on `ctx.Done()`; `toBeUndefined()` -> `assert.Nil(children[key])` |
| interpreter > shouldn't execute actions when reading a snapshot of not started actor | TestInterpreter_ShouldntExecuteActionsWhenReadingASnapshotOfNotStartedActor | ported | JS L1798 |
| interpreter > should execute entry actions when starting the actor after reading its snapshot first | TestInterpreter_ShouldExecuteEntryActionsWhenStartingTheActorAfterReadingItsSnapshotFirst | ported | JS L1813; `toHaveBeenCalled()` -> `assert.Greater(count, 0)` |
| interpreter > the first state of an actor should be its initial state | TestInterpreter_TheFirstStateOfAnActorShouldBeItsInitialState | ported | JS L1830; `toBe` -> `assert.Same` |
| interpreter > should call an onDone callback immediately if the service is already done | TestInterpreter_ShouldCallAnOnDoneCallbackImmediatelyIfTheServiceIsAlreadyDone | ported | JS L1840 |
| should throw if an event is received | TestInterpreter_ShouldThrowIfAnEventIsReceived | N/A-type | JS L1864; JS sends a bare string `'EVENT'` (with `@ts-ignore`) and expects a throw; `Send(event xs.Event)` rejects a string at compile time |
| should not process events sent directly to own actor ref before initial entry actions are processed | TestInterpreter_ShouldNotProcessEventsSentDirectlyToOwnActorRefBeforeInitialEntryActionsAreProcessed | ported | JS L1877; `actorRef` declared before the machine so the entry closure can reference it |
| should not notify the completion observer for an active logic when it gets subscribed before starting | TestInterpreter_ShouldNotNotifyCompletionObserverForActiveLogicSubscribedBeforeStarting | ported | JS L1915; Go name shortened (drops articles/"when it gets") |
| should notify the error observer for an errored logic when it gets subscribed after it errors | TestInterpreter_ShouldNotifyErrorObserverForErroredLogicSubscribedAfterItErrors | ported | JS L1924; inline snapshot `[[Error: error]]` -> `assert.Equal([][]any{{errors.New("error")}}, spy.Calls())` (deep-equal on message) |

Totals: 21 JS tests; 20 ported, 1 N/A-type, 0 N/A-runtime, 0 skipped-in-JS.

## API gaps

None (`apigap_interpreter_2.go` not created).

## Ambiguities for review

- JS L1392 (Symbol.observable interop): ported as consumption through `xs.Subscribable[S]` rather than marked N/A. Reviewer may prefer N/A-runtime.
- JS L1924: assumes the error observer receives the recovered panic value unchanged (per docs/porting/core.md "Snapshot `Error` holds the recovered value"). If the implementation wraps panics, the assertion needs `EqualError`.
- JS L1570: assumes the onDone event reaches the guard as the value type `xs.DoneActorEvent` (contract `event.go` uses value receivers).
- JS L1864: marked N/A-type. An alternative is `Send(nil)` expecting a panic, but JS does not test nil, so that would add behaviour the JS test does not check.
