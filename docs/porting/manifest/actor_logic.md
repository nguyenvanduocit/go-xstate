> Historical translation record. Chunk tags, API-gap filenames, line numbers,
> and run counts below describe the original porting work. See the current
> [core guide](../core.md) and [architecture](../../ARCHITECTURE.md) for the maintained layout.

## Source section: actor_logic_1

Source: `references/xstate/packages/core/test/actorLogic.test.ts` lines 1-1278 (whole file; 1278 lines).
Go file: `xstate/logic_test.go`. Gap file: `apigap_actor_logic_1.go`.

Totals: 49 JS tests; 49 ported, 0 N/A-type, 0 N/A-runtime, 0 skipped-in-JS.

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| promise logic (fromPromise) > should interpret a promise | TestActorLogic_FromPromise_ShouldInterpretAPromise | ported |  |
| promise logic (fromPromise) > should resolve | TestActorLogic_FromPromise_ShouldResolve | ported |  |
| promise logic (fromPromise) > should resolve (observer .next) | TestActorLogic_FromPromise_ShouldResolveObserverNext | ported |  |
| promise logic (fromPromise) > should reject (observer .error) | TestActorLogic_FromPromise_ShouldRejectObserverError | ported | JS rejects with string 'Error'; Go returns errors.New("Error") and asserts equality of the error value. |
| promise logic (fromPromise) > should complete (observer .complete) | TestActorLogic_FromPromise_ShouldCompleteObserverComplete | ported |  |
| promise logic (fromPromise) > should not execute when reading initial state | TestActorLogic_FromPromise_ShouldNotExecuteWhenReadingInitialState | ported |  |
| promise logic (fromPromise) > should persist an unresolved promise | TestActorLogic_FromPromise_ShouldPersistAnUnresolvedPromise | ported |  |
| promise logic (fromPromise) > should persist a resolved promise | TestActorLogic_FromPromise_ShouldPersistAResolvedPromise | ported | setTimeout(5) -> sleep(5) then assertions inline. Inline snapshot -> &PromiseSnapshot[int]{Status: done, Output: 42} (assumes promise persisted snapshot is the snapshot itself, as in JS promise.ts). |
| promise logic (fromPromise) > should not invoke a resolved promise again | TestActorLogic_FromPromise_ShouldNotInvokeAResolvedPromiseAgain | ported | Persisted snapshot asserted as *PromiseSnapshot[int] (see assumptions). |
| promise logic (fromPromise) > should not invoke a rejected promise again | TestActorLogic_FromPromise_ShouldNotInvokeARejectedPromiseAgain | ported | JS rejects with number 1; Go rejects with actorLogic1RejectErr(1). Output undefined -> zero int 0. |
| promise logic (fromPromise) > should have access to the system | TestActorLogic_FromPromise_ShouldHaveAccessToTheSystem | ported | expect.assertions(1) -> probe records exactly 1 value; Go body runs on a goroutine so the test waits for it. |
| promise logic (fromPromise) > should have reference to self | TestActorLogic_FromPromise_ShouldHaveReferenceToSelf | ported | `self.send` defined -> a.Self not nil. |
| promise logic (fromPromise) > should abort when stopping | TestActorLogic_FromPromise_ShouldAbortWhenStopping | ported | abort listener called -> ctx.Err() != nil (sync, deterministic). Never-settling promise body is released at t.Cleanup. |
| promise logic (fromPromise) > should not abort when stopped if promise is resolved/rejected | TestActorLogic_FromPromise_ShouldNotAbortWhenStoppedIfPromiseIsResolvedRejected | ported | listener not called -> ctx.Err() == nil after Stop. Rejected+catch -> body returns (0, nil) after rejection signal. |
| promise logic (fromPromise) > should not reuse the same signal for different actors with same logic | TestActorLogic_FromPromise_ShouldNotReuseSignalForDifferentActorsWithSameLogic | ported | Bodies run on goroutines: deferreds collected from a channel and keyed by self.ID(). |
| promise logic (fromPromise) > should not reuse the same signal for different actors with same logic and id | TestActorLogic_FromPromise_ShouldNotReuseSignalForDifferentActorsWithSameLogicAndID | ported | JS deferredList order = synchronous creation order; Go recovers it from @xstate.actor inspection events (SessionID). |
| promise logic (fromPromise) > should not reuse the same signal for the same actor when restarted | TestActorLogic_FromPromise_ShouldNotReuseSignalForSameActorWhenRestarted | ported | Listener mapped to ctx.Err(). |
| transition function logic (fromTransition) > should interpret a transition function | TestActorLogic_FromTransition_ShouldInterpretATransitionFunction | ported |  |
| transition function logic (fromTransition) > should persist a transition function | TestActorLogic_FromTransition_ShouldPersistATransitionFunction | ported | toEqual -> &TransitionSnapshot[state]{Status: active, Context: {on}}. |
| transition function logic (fromTransition) > should have access to the system | TestActorLogic_FromTransition_ShouldHaveAccessToTheSystem | ported |  |
| transition function logic (fromTransition) > should have reference to self | TestActorLogic_FromTransition_ShouldHaveReferenceToSelf | ported |  |
| observable logic (fromObservable) > should interpret an observable | TestActorLogic_FromObservable_ShouldInterpretAnObservable | ported |  |
| observable logic (fromObservable) > should resolve | TestActorLogic_FromObservable_ShouldResolve | ported | toHaveBeenCalledWith(42) -> Contains(calls, [42]). |
| observable logic (fromObservable) > should resolve (observer .next) | TestActorLogic_FromObservable_ShouldResolveObserverNext | ported | toHaveBeenCalledWith(42) -> Contains(calls, [42]). |
| observable logic (fromObservable) > should reject (observer .error) | TestActorLogic_FromObservable_ShouldRejectObserverError | ported |  |
| observable logic (fromObservable) > should complete (observer .complete) | TestActorLogic_FromObservable_ShouldCompleteObserverComplete | ported |  |
| observable logic (fromObservable) > should not execute when reading initial state | TestActorLogic_FromObservable_ShouldNotExecuteWhenReadingInitialState | ported |  |
| observable logic (fromObservable) > should have access to the system | TestActorLogic_FromObservable_ShouldHaveAccessToTheSystem | ported |  |
| observable logic (fromObservable) > should have reference to self | TestActorLogic_FromObservable_ShouldHaveReferenceToSelf | ported |  |
| eventObservable logic (fromEventObservable) > should have access to the system | TestActorLogic_FromEventObservable_ShouldHaveAccessToTheSystem | ported |  |
| eventObservable logic (fromEventObservable) > should have reference to self | TestActorLogic_FromEventObservable_ShouldHaveReferenceToSelf | ported |  |
| callback logic (fromCallback) > should interpret a callback | TestActorLogic_FromCallback_ShouldInterpretACallback | ported |  |
| callback logic (fromCallback) > should have access to the system | TestActorLogic_FromCallback_ShouldHaveAccessToTheSystem | ported |  |
| callback logic (fromCallback) > should have reference to self | TestActorLogic_FromCallback_ShouldHaveReferenceToSelf | ported |  |
| callback logic (fromCallback) > can send self reference in an event to parent | TestActorLogic_FromCallback_CanSendSelfReferenceInAnEventToParent | ported |  |
| callback logic (fromCallback) > should persist the input of a callback | TestActorLogic_FromCallback_ShouldPersistTheInputOfACallback | ported | spy.mockClear() -> slice calls after the clear index. |
| machine logic > should persist a machine | TestActorLogic_MachineLogic_ShouldPersistAMachine | ported | Persisted machine snapshot navigated as nested map[string]any (children.<id>.snapshot). objectContaining checked field by field; nested `children: {reducer}` checked exactly (len 1). |
| machine logic > should persist and restore a nested machine | TestActorLogic_MachineLogic_ShouldPersistAndRestoreANestedMachine | ported |  |
| machine logic > should return the initial persisted state of a non-started actor | TestActorLogic_MachineLogic_ShouldReturnInitialPersistedStateOfNonStartedActor | ported | objectContaining({value}) -> field check. |
| machine logic > the initial state of a child is available before starting the parent | TestActorLogic_MachineLogic_InitialStateOfChildAvailableBeforeStartingParent | ported | objectContaining({value}) -> field check. |
| machine logic > should not invoke an actor if it is missing in persisted state | TestActorLogic_MachineLogic_ShouldNotInvokeActorIfMissingInPersistedState | ported | delete persisted.children['child'] -> delete on the persisted children map. |
| machine logic > should persist a spawned actor with referenced src | TestActorLogic_MachineLogic_ShouldPersistASpawnedActorWithReferencedSrc | ported | toBe on refs -> assert.Same. |
| machine logic > should not persist a spawned actor with inline src | TestActorLogic_MachineLogic_ShouldNotPersistASpawnedActorWithInlineSrc | ported | toThrowErrorMatchingInlineSnapshot -> PanicsWithError exact message. |
| machine logic > should have access to the system | TestActorLogic_MachineLogic_ShouldHaveAccessToTheSystem | ported |  |
| composable actor logic > should work with machines | TestActorLogic_Composable_ShouldWorkWithMachines | ported | `{...actorLogic, transition}` -> copy of xs.LogicOf(...) with Transition overridden (gap). |
| composable actor logic > should work with promises | TestActorLogic_Composable_ShouldWorkWithPromises | ported | Uses LogicOf gap. |
| composable actor logic > should work with functions | TestActorLogic_Composable_ShouldWorkWithFunctions | ported | Uses LogicOf gap. |
| composable actor logic > should work with observables | TestActorLogic_Composable_ShouldWorkWithObservables | ported | Uses LogicOf gap; complete callback copies logs, assertion in test goroutine. |
| composable actor logic > higher-level logic wrapping a machine should be able to persist a snapshot | TestActorLogic_Composable_HigherLevelLogicWrappingMachineCanPersistSnapshot | ported | Uses LogicOf gap; status compared as xs.StatusActive. |

## API gaps

- `LogicOf[S Snapshot](logic TypedActorLogic[S]) *Logic[S]` — mirrors object spread `{ ...actorLogic }` so composable-logic tests (JS lines 1109-1278) can override `transition` while keeping the wrapped logic's other methods. The contract exposes no `Transition` method on PromiseLogic/TransitionLogic/ObservableLogic and no `Start` on StateMachine, so a spread cannot be built from existing API.

## Assumptions / ambiguities for reviewer

- Giả định: `GetPersistedSnapshot()` of promise / transition logic returns the snapshot pointer itself (`*PromiseSnapshot[O]`, `*TransitionSnapshot[T]`), because JS `getPersistedSnapshot` for these logics returns the snapshot object (promise.ts, transition.ts). Affects JS lines 140-147, 176-183, 207-214, 515-522, 860-867, 874, 1063.
- Giả định: a machine's persisted snapshot is a JSON-like `map[string]any` mirroring the JS object: keys `status` (xs.Status), `value` (StateValue), `context` (typed C), `children` -> `map[string]any` of `map[string]any{"snapshot": <child persisted snapshot>, ...}`. Affects JS lines 858-889, 958-961, 979-983, 1028, 1060-1063, 1270-1275.
- Abort-signal listeners (`signal.addEventListener('abort', fn)`, JS lines 244-475) are mapped to `ctx.Err() != nil` on the promise body's context: deterministic and equivalent to 'listener was called'. A Go implementation that cancels the ctx after the promise settles (idiomatic `defer cancel()`) will fail these tests, exactly as JS would.
- fromPromise bodies run on goroutines in Go; JS runs them synchronously inside `start()`. Tests that read state created by the body right after start (JS lines 244-475) wait on a channel instead.
- JS line 352 (`same logic and id`): deferredList order is recovered from `@xstate.actor` inspection events because both children share id `p` and goroutine entry order is nondeterministic.
- JS line 68 / 191: promise rejections with non-Error values ('Error' string, number 1) are mapped to `errors.New("Error")` and `actorLogic1RejectErr(1)`.
- `TestActorLogic_FromPromise_ShouldResolve` duplicates `example_port_test.go:TestActorLogic_PromiseShouldResolve` (tag `port_example`); names differ, so no collision when tags are removed.
