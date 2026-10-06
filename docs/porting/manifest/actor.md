> Historical translation record. Chunk tags, API-gap filenames, line numbers,
> and run counts below describe the original porting work. See the current
> [core guide](../core.md) and [architecture](../../ARCHITECTURE.md) for the maintained layout.

## Source section: actor_1

Source: `references/xstate/packages/core/test/actor.test.ts` lines 1-1299 (27 `it` calls; no `it.each`, `it.skip` or `it.todo` in range).
The `it` at JS L1300 ("should not crash on child machine sync completion during self-initialization") starts outside the range and belongs to the next chunk.
Go file: `xstate/actor_test.go`.

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| spawning machines > should spawn machines | TestActor_SpawningMachines_ShouldSpawnMachines | ported | JS L123; `todoRefs` keyed by `event.id` (map[any]xs.ActorRef, key int 42) |
| spawning machines > should spawn referenced machines | TestActor_SpawningMachines_ShouldSpawnReferencedMachines | ported | JS L188 |
| spawning machines > should allow bidirectional communication between parent/child actors | TestActor_SpawningMachines_ShouldAllowBidirectionalCommunicationBetweenParentChildActors | ported | JS L225; describe-level serverMachine/clientMachine (JS L54-121) defined inside the test |
| spawning promises > should be able to spawn a promise | TestActor_SpawningPromises_ShouldBeAbleToSpawnAPromise | ported | JS L241 |
| spawning promises > should be able to spawn a referenced promise | TestActor_SpawningPromises_ShouldBeAbleToSpawnAReferencedPromise | ported | JS L293 |
| spawning callbacks > should be able to spawn an actor from a callback | TestActor_SpawningCallbacks_ShouldBeAbleToSpawnAnActorFromACallback | ported | JS L340 |
| spawning callbacks > should not deliver events sent to the parent after the callback actor gets stopped | TestActor_SpawningCallbacks_ShouldNotDeliverEventsSentToParentAfterCallbackActorGetsStopped | ported | JS L396; `sendToParent!()` -> `require.NotNil` then call; Go name drops articles |
| spawning observables > should spawn an observable | TestActor_SpawningObservables_ShouldSpawnAnObservable | ported | JS L436 |
| spawning observables > should spawn a referenced observable | TestActor_SpawningObservables_ShouldSpawnAReferencedObservable | ported | JS L481 |
| spawning observables > should read the latest snapshot of the event's origin while handling that event | TestActor_SpawningObservables_ShouldReadLatestSnapshotOfEventsOriginWhileHandlingThatEvent | ported | JS L526; `context.observableRef.getSnapshot()` -> `AnySnapshot().(*xs.ObservableSnapshot[int])` |
| spawning observables > should notify direct child listeners with final snapshot before it gets stopped | TestActor_SpawningObservables_ShouldNotifyDirectChildListenersWithFinalSnapshotBeforeStopped | ported | JS L576; child subscribe via `xs.As[*xs.ObservableSnapshot[int]]`; `toHaveBeenCalledWith(3)` -> `assert.Contains(calls, []any{3})` |
| spawning observables > should not notify direct child listeners after it gets stopped | TestActor_SpawningObservables_ShouldNotNotifyDirectChildListenersAfterItGetsStopped | ported | JS L630; `spy.mockClear()` + `not.toHaveBeenCalled()` -> count recorded at the clear point must stay unchanged after `sleep(15)` |
| spawning event observables > should spawn an event observable | TestActor_SpawningEventObservables_ShouldSpawnAnEventObservable | ported | JS L690; `interval(10).pipe(map(...))` -> `rxMap(rxInterval(10), ...)` |
| spawning event observables > should spawn a referenced event observable | TestActor_SpawningEventObservables_ShouldSpawnAReferencedEventObservable | ported | JS L734 |
| communicating with spawned actors > should treat an interpreter as an actor | TestActor_CommunicatingWithSpawnedActors_ShouldTreatAnInterpreterAsAnActor | ported | JS L782; `*xs.Actor` stored as `xs.ActorRef` in context; `origin: self` -> `xs.E{"origin": a.Self}` |
| actors > should only spawn actors defined on initial state once | TestActor_Actors_ShouldOnlySpawnActorsDefinedOnInitialStateOnce | ported | JS L852; assertion stays inside the subscriber; counter is atomic |
| actors > should spawn an actor in an initial state of a child that gets invoked in the initial state of a parent when the parent gets started | TestActor_Actors_ShouldSpawnActorInInitialStateOfChildInvokedInInitialStateOfParentOnStart | ported | JS L886; see ambiguity 1; Go name shortened |
| actors > should only spawn an initial actor once when it synchronously responds with an event | TestActor_Actors_ShouldOnlySpawnAnInitialActorOnceWhenItSynchronouslyRespondsWithAnEvent | ported | JS L930; `expect` inside context factory -> `assert.Equal` + panic (JS expect throws) |
| actors > should spawn null actors if not used within a service | TestActor_Actors_ShouldSpawnNullActorsIfNotUsedWithinAService | ported | JS L968; `ref!.send` toBeDefined -> `assert.NotNil(ref)` (any non-nil ActorRef has Send) |
| actors > should stop multiple inline spawned actors that have no explicit ids | TestActor_Actors_ShouldStopMultipleInlineSpawnedActorsThatHaveNoExplicitIds | ported | JS L988 |
| actors > should stop multiple referenced spawned actors that have no explicit ids | TestActor_Actors_ShouldStopMultipleReferencedSpawnedActorsThatHaveNoExplicitIds | ported | JS L1008 |
| actors > with actor logic > should work with a transition function logic | TestActor_Actors_WithActorLogic_ShouldWorkWithATransitionFunctionLogic | ported | JS L1037 |
| actors > with actor logic > should work with a promise logic (fulfill) | TestActor_Actors_WithActorLogic_ShouldWorkWithAPromiseLogicFulfill | ported | JS L1083; `setTimeout(() => res(42))` -> `sleep(0)` then return |
| actors > with actor logic > should work with a promise logic (reject) | TestActor_Actors_WithActorLogic_ShouldWorkWithAPromiseLogicReject | ported | JS L1132; see ambiguity 2 |
| actors > with actor logic > actor logic should have reference to the parent | TestActor_Actors_WithActorLogic_ActorLogicShouldHaveReferenceToTheParent | ported | JS L1178; custom `&xs.Logic[*xs.BasicSnapshot[any]]`; `self._parent` -> `scope.Self.Parent()` |
| actors > should be able to spawn callback actors in (lazy) initial context | TestActor_Actors_ShouldBeAbleToSpawnCallbackActorsInLazyInitialContext | ported | JS L1235 |
| actors > should be able to spawn machines in (lazy) initial context | TestActor_Actors_ShouldBeAbleToSpawnMachinesInLazyInitialContext | ported | JS L1267 |

Totals: 27 JS tests: 27 ported, 0 N/A-type, 0 N/A-runtime, 0 skipped-in-JS.

## API gaps

None. No `apigap_actor_1.go` was created; every test is expressed with the existing contract.

## Ambiguities for review

1. JS L925 (`expect(spawnCounter).toBe(1)` right after `start()`): in JS the `fromPromise` creator runs synchronously during start. `logic.go` `FromPromise` says "fn runs on its own goroutine", so an immediate check would race. The Go test waits with `assert.Eventually` until the counter is at least 1, sleeps 10ms, then asserts it equals exactly 1. A double spawn still fails the test.
2. JS L1148/L1160: JS rejects with the string `errorMessage` and the guard checks `event.error === errorMessage`. In Go a promise rejection is an `error`, so the test rejects with one `errors.New("An error occurred")` value and the guard compares the `ErrorActorEvent.Error` against that same value by identity.
3. JS L681 (`spy.mockClear()`): `xstate/helpers_test.go` `spy` has no reset. The Go test saves `s.Count()` at the clear point and asserts the count is the same after `sleep(15)`.
4. JS L879-881: the subscriber asserts `count == 1` on every notification. The Go test keeps the assert inside the subscriber, as JS does. Notifications that arrive after the test returns (promise children resolving) are not checked, which matches JS, where `it` also returns synchronously.
5. JS L620/L673: `children.childActor!.subscribe(...)`. The Go test uses `require.NotNil` on the child, then subscribes through `xs.As[*xs.ObservableSnapshot[int]]`.


## Source section: actor_2

Source: `references/xstate/packages/core/test/actor.test.ts` lines 1300-1841 (all inside `describe('actors')`).
Go file: `xstate/actor_test.go`. 14 JS tests: 14 ported.

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| actors > should not crash on child machine sync completion during self-initialization (L1300) | TestActor_Actors_ShouldNotCrashOnChildMachineSyncCompletionDuringSelfInitialization | ported | `not.toThrow` → `assert.NotPanics` |
| actors > should not crash on child promise-like sync completion during self-initialization (L1341) | TestActor_Actors_ShouldNotCrashOnChildPromiseLikeSyncCompletionDuringSelfInitialization | ported | JS thenable that resolves synchronously → Go promise func that returns `(nil, nil)` immediately (see ambiguities) |
| actors > should not crash on child observable sync completion during self-initialization (L1362) | TestActor_Actors_ShouldNotCrashOnChildObservableSyncCompletionDuringSelfInitialization | ported | custom subscribable `actor2SyncCompleteObservable` completes synchronously inside Subscribe |
| actors > should receive done event from an immediately completed observable when self-initializing (L1390) | TestActor_Actors_ShouldReceiveDoneEventFromImmediatelyCompletedObservableWhenSelfInitializing | ported | `EMPTY` → `rxEmpty[any]()` |
| actors > should not restart a completed observable (L1422) | TestActor_Actors_ShouldNotRestartACompletedObservable | ported | counter is `atomic.Int32` |
| actors > should not restart a completed event observable (L1445) | TestActor_Actors_ShouldNotRestartACompletedEventObservable | ported | `fromEventObservable` → `xs.FromEventObservable` |
| actors > should be able to restart a spawned actor within a single macrostep (L1468) | TestActor_Actors_ShouldBeAbleToRestartASpawnedActorWithinASingleMacrostep | ported | `actual` → mutex-guarded `actor2Recorder`; `actual.length = 0` → `reset()` |
| actors > should be able to restart a named spawned actor within a single macrostep when stopping by a ref (L1535) | TestActor_Actors_ShouldRestartNamedSpawnedActorWithinSingleMacrostepWhenStoppingByRef | ported | `stopChild(({context}) => context.actorRef)` → `xs.StopChild(xs.NewExpr(...))` |
| actors > should be able to restart a named spawned actor within a single macrostep when stopping by static name (L1600) | TestActor_Actors_ShouldRestartNamedSpawnedActorWithinSingleMacrostepWhenStoppingByStaticName | ported | `xs.StopChild("my_name")` |
| actors > should be able to restart a named spawned actor within a single macrostep when stopping by resolved name (L1665) | TestActor_Actors_ShouldRestartNamedSpawnedActorWithinSingleMacrostepWhenStoppingByResolvedName | ported | `stopChild(() => 'my_name')` → Expr returning `"my_name"` |
| actors > should be possible to pass `self` as input to a child machine from within the context factory (L1730) | TestActor_Actors_ShouldBePossibleToPassSelfAsInputToChildMachineFromContextFactory | ported | input is a local `childInput` struct |
| actors > catches errors from spawned promise actors (L1766) | TestActor_Actors_CatchesErrorsFromSpawnedPromiseActors | ported | `expect.assertions(1)` → atomic counter asserted == 1 after waiting on a signal |
| actors > same-position invokes should not leak between machines (L1792) | TestActor_Actors_SamePositionInvokesShouldNotLeakBetweenMachines | ported | `toHaveBeenCalledWith('foo')` → `assert.Contains(spy.Calls(), []any{"foo"})` |
| actors > inline invokes should not leak into provided actors object (L1824) | TestActor_Actors_InlineInvokesShouldNotLeakIntoProvidedActorsObject | ported | `expect(actors).toEqual({})` → equals empty `map[string]xs.ActorLogic{}` |

## API gaps

None. No `apigap_actor_2.go` was created.

## Ambiguities for review

- L1342-1344: JS `fromPromise(() => ({ then: (fn) => fn(null) }))` returns a thenable that resolves synchronously. Go `FromPromise` runs on its own goroutine and cannot resolve synchronously. The port returns `(nil, nil)` immediately. That keeps the "spawned promise completes immediately during self-initialization" scenario, but not the synchronous timing.
- L1766-1790: the JS test is synchronous with `expect.assertions(1)`, so the rejection is observed in a microtask before vitest checks the count. Go waits on a signal (default 2s timeout) and then asserts the error observer ran exactly once. The observer also checks that the value is an `error`, so `.message` can be compared through `Error()`.
- L1792-1822: `await sleep(1)` → `sleep(1)`. The promise goroutine has 1ms to resolve and deliver `onDone`. This can be timing-sensitive under `-race`.
- L1730-1764: the JS spy receives the action args; the Go spy records `ActionArgs`. Only the call count is asserted, as in JS.
- L1300-1339: `entry: 'setup'` → `xs.ActionRef{Type: "setup"}`, with the `assign` passed through `Implementations.Actions`.
