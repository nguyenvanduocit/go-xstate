> Historical translation record. Chunk tags, API-gap filenames, line numbers,
> and run counts below describe the original porting work. See the current
> [core guide](../core.md) and [architecture](../../ARCHITECTURE.md) for the maintained layout.

## Source section: invoke_1

Source: `references/xstate/packages/core/test/invoke.test.ts` lines 1-1365. 13 `it` calls outside the parametrized describe + 13 `it` calls inside `promiseTypes.forEach(... describe(`with promises (${type})`))` (L758), which runs for 2 types (`Promise`, `PromiseLike`) = 26 tests. Total 39 JS tests. No `it.each`, `it.skip` or `it.todo` in range. `describe('with callbacks')` at L1366 belongs to the next chunk.
Go file: `xstate/invoke_test.go`.

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| invoke > child can immediately respond to the parent with multiple events | TestInvoke_ChildCanImmediatelyRespondToTheParentWithMultipleEvents | ported | JS L31; identical to example_port_test.go contract example (Go name differs to avoid collision) |
| invoke > should start services (explicit machine, invoke = config) | TestInvoke_ShouldStartServicesExplicitMachineInvokeConfig | ported | JS L107; `user` const (L28) -> `invoke1UserData`; `context.userId !== undefined` -> `UserID *string` non-nil |
| invoke > should start services (explicit machine, invoke = machine) | TestInvoke_ShouldStartServicesExplicitMachineInvokeMachine | ported | JS L199 |
| invoke > should start services (machine as invoke config) | TestInvoke_ShouldStartServicesMachineAsInvokeConfig | ported | JS L252 |
| invoke > should start deeply nested service (machine as invoke config) | TestInvoke_ShouldStartDeeplyNestedServiceMachineAsInvokeConfig | ported | JS L296 |
| invoke > should use the service overwritten by .provide(...) | TestInvoke_ShouldUseTheServiceOverwrittenByProvide | ported | JS L346 |
| invoke > parent to child > should communicate with the child machine (invoke on machine) | TestInvoke_ParentToChild_ShouldCommunicateWithTheChildMachineInvokeOnMachine | ported | JS L429; describe-level `subMachine` (L416) -> `invoke1ParentToChildSubMachine()` |
| invoke > parent to child > should communicate with the child machine (invoke on state) | TestInvoke_ParentToChild_ShouldCommunicateWithTheChildMachineInvokeOnState | ported | JS L459; same `subMachine` helper |
| invoke > parent to child > should transition correctly if child invocation causes it to directly go to final state | TestInvoke_ParentToChild_ShouldTransitionCorrectlyIfChildInvocationGoesDirectlyToFinal | ported | JS L489; Go name shortened |
| invoke > parent to child > should work with invocations defined in orthogonal state nodes | TestInvoke_ParentToChild_ShouldWorkWithInvocationsDefinedInOrthogonalStateNodes | ported | JS L529 |
| invoke > parent to child > should not reinvoke root-level invocations on root non-reentering transitions | TestInvoke_ParentToChild_ShouldNotReinvokeRootLevelInvocationsOnRootNonReenteringTransitions | ported | JS L577; counters are atomic.Int32 |
| invoke > parent to child > should stop a child actor when reaching a final state | TestInvoke_ParentToChild_ShouldStopAChildActorWhenReachingAFinalState | ported | JS L624 |
| invoke > parent to child > child should not invoke an actor when it transitions to an invoking state when it gets stopped by its parent | TestInvoke_ParentToChild_ChildShouldNotInvokeActorWhenTransitioningToInvokingStateWhileStoppedByParent | ported | JS L652; `setTimeout(() => sendBack(...))` -> `time.AfterFunc(0, ...)`; `forwardTo(SpecialTargets.Parent)` -> `xs.ForwardTo("#_parent")` (enum value, no contract constant); Go name shortened |
| invoke > with promises (Promise) > should be invoked with a promise factory and resolve through onDone | TestInvoke_WithPromisesPromise_ShouldBeInvokedWithAPromiseFactoryAndResolveThroughOnDone | ported | JS L803; body `invoke1Promise*` shared by both promise types |
| invoke > with promises (Promise) > should be invoked with a promise factory and reject with ErrorExecution | TestInvoke_WithPromisesPromise_ShouldBeInvokedWithAPromiseFactoryAndRejectWithErrorExecution | ported | JS L833; describe-level `invokePromiseMachine` (L759) built inside the shared body; `{id:42, succeed:true, ...input}` merge done field by field; body `invoke1Promise*` shared by both promise types |
| invoke > with promises (Promise) > should be invoked with a promise factory and surface any unhandled errors | TestInvoke_WithPromisesPromise_ShouldBeInvokedWithAPromiseFactoryAndSurfaceAnyUnhandledErrors | ported | JS L843; `stringMatching(/test/)` -> `assert.Regexp`; body `invoke1Promise*` shared by both promise types |
| invoke > with promises (Promise) > should be invoked with a promise factory and stop on unhandled onError target | TestInvoke_WithPromisesPromise_ShouldBeInvokedWithAPromiseFactoryAndStopOnUnhandledOnErrorTarget | ported | JS L877; `toBeInstanceOf(Error)` -> value implements `error`; body `invoke1Promise*` shared by both promise types |
| invoke > with promises (Promise) > should be invoked with a promise factory and resolve through onDone for compound state nodes | TestInvoke_WithPromisesPromise_PromiseFactoryResolveThroughOnDoneForCompoundStateNodes | ported | JS L916; Go name shortened; body `invoke1Promise*` shared by both promise types |
| invoke > with promises (Promise) > should be invoked with a promise service and resolve through onDone for compound state nodes | TestInvoke_WithPromisesPromise_PromiseServiceResolveThroughOnDoneForCompoundStateNodes | ported | JS L950; Go name shortened; body `invoke1Promise*` shared by both promise types |
| invoke > with promises (Promise) > should assign the resolved data when invoked with a promise factory | TestInvoke_WithPromisesPromise_ShouldAssignTheResolvedDataWhenInvokedWithAPromiseFactory | ported | JS L990; body `invoke1Promise*` shared by both promise types |
| invoke > with promises (Promise) > should assign the resolved data when invoked with a promise service | TestInvoke_WithPromisesPromise_ShouldAssignTheResolvedDataWhenInvokedWithAPromiseService | ported | JS L1028; body `invoke1Promise*` shared by both promise types |
| invoke > with promises (Promise) > should provide the resolved data when invoked with a promise factory | TestInvoke_WithPromisesPromise_ShouldProvideTheResolvedDataWhenInvokedWithAPromiseFactory | ported | JS L1073; captured `count` is atomic.Int64; body `invoke1Promise*` shared by both promise types |
| invoke > with promises (Promise) > should provide the resolved data when invoked with a promise service | TestInvoke_WithPromisesPromise_ShouldProvideTheResolvedDataWhenInvokedWithAPromiseService | ported | JS L1112; captured `count` is atomic.Int64; body `invoke1Promise*` shared by both promise types |
| invoke > with promises (Promise) > should be able to specify a Promise as a service | TestInvoke_WithPromisesPromise_ShouldBeAbleToSpecifyAPromiseAsAService | ported | JS L1157; `reject()` (undefined reason) -> `invoke1Rejection{nil}` error (branch not taken); body `invoke1Promise*` shared by both promise types |
| invoke > with promises (Promise) > should be able to reuse the same promise logic multiple times and create unique promise for each created actor | TestInvoke_WithPromisesPromise_ShouldReuseSamePromiseLogicAndCreateUniquePromisePerActor | ported | JS L1225; `null` -> `*float64` nil; `typeof === 'number'` -> NotNil; Go name shortened; body `invoke1Promise*` shared by both promise types |
| invoke > with promises (Promise) > should not emit onSnapshot if stopped | TestInvoke_WithPromisesPromise_ShouldNotEmitOnSnapshotIfStopped | ported | JS L1321; JS fails via the unhandled error thrown by the `'*'` action -> `WithUnhandledErrorHandler` records errors, asserted empty after `sleep(10)` (plus status != error, same failure path); `event.snapshot` truthy -> event is `xs.SnapshotEvent`; body `invoke1Promise*` shared by both promise types |
| invoke > with promises (PromiseLike) > should be invoked with a promise factory and resolve through onDone | TestInvoke_WithPromisesPromiseLike_ShouldBeInvokedWithAPromiseFactoryAndResolveThroughOnDone | ported | JS L803; body `invoke1Promise*` shared by both promise types |
| invoke > with promises (PromiseLike) > should be invoked with a promise factory and reject with ErrorExecution | TestInvoke_WithPromisesPromiseLike_ShouldBeInvokedWithAPromiseFactoryAndRejectWithErrorExecution | ported | JS L833; describe-level `invokePromiseMachine` (L759) built inside the shared body; `{id:42, succeed:true, ...input}` merge done field by field; body `invoke1Promise*` shared by both promise types |
| invoke > with promises (PromiseLike) > should be invoked with a promise factory and surface any unhandled errors | TestInvoke_WithPromisesPromiseLike_ShouldBeInvokedWithAPromiseFactoryAndSurfaceAnyUnhandledErrors | ported | JS L843; `stringMatching(/test/)` -> `assert.Regexp`; body `invoke1Promise*` shared by both promise types |
| invoke > with promises (PromiseLike) > should be invoked with a promise factory and stop on unhandled onError target | TestInvoke_WithPromisesPromiseLike_ShouldBeInvokedWithAPromiseFactoryAndStopOnUnhandledOnErrorTarget | ported | JS L877; `toBeInstanceOf(Error)` -> value implements `error`; body `invoke1Promise*` shared by both promise types |
| invoke > with promises (PromiseLike) > should be invoked with a promise factory and resolve through onDone for compound state nodes | TestInvoke_WithPromisesPromiseLike_PromiseFactoryResolveThroughOnDoneForCompoundStateNodes | ported | JS L916; Go name shortened; body `invoke1Promise*` shared by both promise types |
| invoke > with promises (PromiseLike) > should be invoked with a promise service and resolve through onDone for compound state nodes | TestInvoke_WithPromisesPromiseLike_PromiseServiceResolveThroughOnDoneForCompoundStateNodes | ported | JS L950; Go name shortened; body `invoke1Promise*` shared by both promise types |
| invoke > with promises (PromiseLike) > should assign the resolved data when invoked with a promise factory | TestInvoke_WithPromisesPromiseLike_ShouldAssignTheResolvedDataWhenInvokedWithAPromiseFactory | ported | JS L990; body `invoke1Promise*` shared by both promise types |
| invoke > with promises (PromiseLike) > should assign the resolved data when invoked with a promise service | TestInvoke_WithPromisesPromiseLike_ShouldAssignTheResolvedDataWhenInvokedWithAPromiseService | ported | JS L1028; body `invoke1Promise*` shared by both promise types |
| invoke > with promises (PromiseLike) > should provide the resolved data when invoked with a promise factory | TestInvoke_WithPromisesPromiseLike_ShouldProvideTheResolvedDataWhenInvokedWithAPromiseFactory | ported | JS L1073; captured `count` is atomic.Int64; body `invoke1Promise*` shared by both promise types |
| invoke > with promises (PromiseLike) > should provide the resolved data when invoked with a promise service | TestInvoke_WithPromisesPromiseLike_ShouldProvideTheResolvedDataWhenInvokedWithAPromiseService | ported | JS L1112; captured `count` is atomic.Int64; body `invoke1Promise*` shared by both promise types |
| invoke > with promises (PromiseLike) > should be able to specify a Promise as a service | TestInvoke_WithPromisesPromiseLike_ShouldBeAbleToSpecifyAPromiseAsAService | ported | JS L1157; `reject()` (undefined reason) -> `invoke1Rejection{nil}` error (branch not taken); body `invoke1Promise*` shared by both promise types |
| invoke > with promises (PromiseLike) > should be able to reuse the same promise logic multiple times and create unique promise for each created actor | TestInvoke_WithPromisesPromiseLike_ShouldReuseSamePromiseLogicAndCreateUniquePromisePerActor | ported | JS L1225; `null` -> `*float64` nil; `typeof === 'number'` -> NotNil; Go name shortened; body `invoke1Promise*` shared by both promise types |
| invoke > with promises (PromiseLike) > should not emit onSnapshot if stopped | TestInvoke_WithPromisesPromiseLike_ShouldNotEmitOnSnapshotIfStopped | ported | JS L1321; JS fails via the unhandled error thrown by the `'*'` action -> `WithUnhandledErrorHandler` records errors, asserted empty after `sleep(10)` (plus status != error, same failure path); `event.snapshot` truthy -> event is `xs.SnapshotEvent`; body `invoke1Promise*` shared by both promise types |

## API gaps

None. No `apigap_invoke_1.go` was created.

## Ambiguities for review

- L734-757 `promiseTypes`: Go has no Promise vs thenable distinction. `invoke1PromiseTypes` models `createPromise(executor)` awaited inside the fromPromise creator: the executor runs synchronously, first resolve/reject wins, a panic rejects (mirrors a throw inside `new Promise`). The `PromiseLike` variant relays the settlement through an extra goroutine hop to model `then(onfulfilled, onrejected)`. Both variants are ported as separate Go tests calling the same body, so the 39 JS tests map 1:1 to 39 Go tests.
- L780 / L853 / L889: `throw new Error(...)` inside the executor -> `panic(errors.New(...))`/`panic(fmt.Errorf(...))` inside the executor, turned into the FromPromise `error` by the helper.
- L688 `forwardTo(SpecialTargets.Parent)`: the contract has no `SpecialTargets` constant; the enum value `"#_parent"` is passed as a string target (JS accepts the same string).
- L1321 `should not emit onSnapshot if stopped` has no explicit `expect`; its JS failure mode is the uncaught error from the `'*'` action. The Go port makes that explicit with `WithUnhandledErrorHandler` + `assert.Empty`, and also asserts the actor status is not `error`.
- L107: child `context.user` is set from the raised event's `user` payload (`invoke1User` value); the parent guard reads `event.output.user.name` via type assertion.
- Implicit JS timeouts: tests that `return promise` wait via `sig.Wait(t)` (2s default).


## Source section: invoke_2

Source: `references/xstate/packages/core/test/invoke.test.ts` lines 1366-2630 (29 `it` calls; no `it.each`, `it.skip`, `it.todo` or template-literal names in range).
The describe `multiple simultaneous services` starts at JS L2631 (outside the range) and belongs to the next chunk.
Go file: `xstate/invoke_test.go`. All JS describe paths start with `invoke >`.

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| invoke > with callbacks > should be able to specify a callback as a service | TestInvoke_WithCallbacks_ShouldBeAbleToSpecifyACallbackAsAService | ported | JS L1367; invoke input `{foo, event}` -> local struct `callbackInput`; guard `event.data === 42` via `xs.E` |
| invoke > with callbacks > should transition correctly if callback function sends an event | TestInvoke_WithCallbacks_ShouldTransitionCorrectlyIfCallbackFunctionSendsAnEvent | ported | JS L1461; JS index loop -> `require.GreaterOrEqual(len)` (JS would compare against `undefined`) + per-index `assert.Equal` |
| invoke > with callbacks > should transition correctly if callback function invoked from start and sends an event | TestInvoke_WithCallbacks_ShouldTransitionCorrectlyIfCallbackInvokedFromStartAndSendsAnEvent | ported | JS L1504; same loop translation |
| invoke > with callbacks > should transition correctly if transient transition happens before current state invokes callback function and sends an event | TestInvoke_WithCallbacks_ShouldTransitionCorrectlyIfTransientTransitionHappensBeforeInvokingCallback | ported | JS L1545; Go name shortened (<100 chars) |
| invoke > with callbacks > should treat a callback source as an event stream | TestInvoke_WithCallbacks_ShouldTreatACallbackSourceAsAnEventStream | ported | JS L1594; `setInterval`/`clearInterval` -> `invoke2Ticker` goroutine + stop func |
| invoke > with callbacks > should dispose of the callback (if disposal function provided) | TestInvoke_WithCallbacks_ShouldDisposeOfTheCallbackIfDisposalFunctionProvided | ported | JS L1636; `toHaveBeenCalled()` -> `Count() >= 1` |
| invoke > with callbacks > callback should be able to receive messages from parent | TestInvoke_WithCallbacks_CallbackShouldBeAbleToReceiveMessagesFromParent | ported | JS L1661 |
| invoke > with callbacks > should call onError upon error (sync) | TestInvoke_WithCallbacks_ShouldCallOnErrorUponErrorSync | ported | JS L1694; `throw new Error('test')` -> `panic(errors.New("test"))`; `instanceof Error && message === 'test'` -> `invoke2ErrorMessage` (error type + `Error()`) |
| invoke > with callbacks > should transition correctly upon error (sync) | TestInvoke_WithCallbacks_ShouldTransitionCorrectlyUponErrorSync | ported | JS L1727 |
| invoke > with callbacks > should call onError only on the state which has invoked failed service | TestInvoke_WithCallbacks_ShouldCallOnErrorOnlyOnTheStateWhichHasInvokedFailedService | ported | JS L1751 |
| invoke > with callbacks > should be able to be stringified | TestInvoke_WithCallbacks_ShouldBeAbleToBeStringified | ported | JS L1809; `JSON.stringify(snapshot)` (which calls `snapshot.toJSON()`) -> `json.Marshal(snapshot.ToJSON())` with `assert.NotPanics` + `assert.NoError` |
| invoke > with callbacks > should result in an error notification if callback actor throws when it starts and the error stays unhandled by the machine | TestInvoke_WithCallbacks_ShouldResultInErrorNotificationIfCallbackThrowsOnStartUnhandled | ported | JS L1834; inline snapshot `[[ [Error: test] ]]` -> exactly 1 call with 1 arg, an `error` with message "test" |
| invoke > with callbacks > should work with input | TestInvoke_WithCallbacks_ShouldWorkWithInput | ported | JS L1866; `input: ({context}) => context` -> Expr returning context struct; `toEqual({foo:'bar'})` -> `ctx{Foo: "bar"}` |
| invoke > with callbacks > sub invoke race condition ends on the completed state | TestInvoke_WithCallbacks_SubInvokeRaceConditionEndsOnTheCompletedState | ported | JS L1891 |
| invoke > with observables > should work with an infinite observable | TestInvoke_WithObservables_ShouldWorkWithAnInfiniteObservable | ported | JS L1935; `count: number \| undefined` -> `*int` |
| invoke > with observables > should work with a finite observable | TestInvoke_WithObservables_ShouldWorkWithAFiniteObservable | ported | JS L1977; `interval(10).pipe(take(5))` -> `rxTake(rxInterval(10), 5)` |
| invoke > with observables > should receive an emitted error | TestInvoke_WithObservables_ShouldReceiveAnEmittedError | ported | JS L2024; `map` that throws at 5 -> file-local operator `invoke2MapOrThrow` (rxMap cannot throw); `expect` inside guard kept as `assert.Equal` |
| invoke > with observables > should work with input | TestInvoke_WithObservables_ShouldWorkWithInput | ported | JS L2085; root-level `invoke` -> `MachineConfig.Invoke` |
| invoke > with event observables > should work with an infinite event observable | TestInvoke_WithEventObservables_ShouldWorkWithAnInfiniteEventObservable | ported | JS L2128 |
| invoke > with event observables > should work with a finite event observable | TestInvoke_WithEventObservables_ShouldWorkWithAFiniteEventObservable | ported | JS L2172 |
| invoke > with event observables > should receive an emitted error | TestInvoke_WithEventObservables_ShouldReceiveAnEmittedError | ported | JS L2226; uses `invoke2MapOrThrow` |
| invoke > with event observables > should work with input | TestInvoke_WithEventObservables_ShouldWorkWithInput | ported | JS L2287 |
| invoke > with logic > should work with actor logic | TestInvoke_WithLogic_ShouldWorkWithActorLogic | ported | JS L2315; custom logic -> `&xs.Logic[*xs.BasicSnapshot[int]]`; `children['count']?.getSnapshot().context` -> nil check + `AnySnapshot().(*xs.BasicSnapshot[int])` |
| invoke > with logic > logic should have reference to the parent | TestInvoke_WithLogic_LogicShouldHaveReferenceToTheParent | ported | JS L2369; `self._parent?.send` -> `scope.Self.Parent()` nil check |
| invoke > with transition functions > should work with a transition function | TestInvoke_WithTransitionFunctions_ShouldWorkWithATransitionFunction | ported | JS L2418 |
| invoke > with transition functions > should schedule events in a FIFO queue | TestInvoke_WithTransitionFunctions_ShouldScheduleEventsInAFIFOQueue | ported | JS L2457; `self.send` -> `scope.Self.Send` |
| invoke > with transition functions > should emit onSnapshot | TestInvoke_WithTransitionFunctions_ShouldEmitOnSnapshot | ported | JS L2501; `{ delay: 10 }` -> `SendOptions{Delay: 10 * time.Millisecond}` |
| invoke > with machines > should create invocations from machines in nested states | TestInvoke_WithMachines_ShouldCreateInvocationsFromMachinesInNestedStates | ported | JS L2582; describe-level `pongMachine`/`pingMachine` (JS L2538-2580) defined inside the test (only user) |
| invoke > with machines > should emit onSnapshot | TestInvoke_WithMachines_ShouldEmitOnSnapshot | ported | JS L2590 |

Totals: 29 JS tests; 29 ported, 0 N/A-type, 0 N/A-runtime, 0 skipped-in-JS.

## API gaps

None. No `apigap_invoke_2.go` was created.

## Ambiguities for review

- JS L1809 "should be able to be stringified": Go has no `JSON.stringify` equivalent for a snapshot holding funcs; ported as `json.Marshal(snapshot.ToJSON())` must not panic or error. This assumes `ToJSON()` returns a JSON-serialisable map (as JS `toJSON()` does).
- JS L1834 inline snapshot `[Error: test]`: asserted as a Go `error` whose `Error()` is "test". The callback panics with `errors.New("test")`, so the implementation must surface the recovered value unchanged.
- JS L2024 / L2226: rxjs `map` throwing inside the pipe is modelled by `invoke2MapOrThrow`, which calls the observer's `Error(errors.New("some error"))` and unsubscribes the source, matching rxjs semantics.
- JS L1461/1504/1545: state values are collected under a mutex because the Go runtime may deliver callback-actor events on another goroutine; assertions still run synchronously right after `Send`, as in JS.


## Source section: invoke_3

Source: `references/xstate/packages/core/test/invoke.test.ts` lines 2631-3535 (22 JS tests: 19 `it` calls + one `it.each` with 3 rows at L3078; no `it.skip`/`it.todo`).
Go file: `xstate/invoke_test.go`. `go test -tags port_invoke_3 -list .` lists 22 tests.

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| invoke > multiple simultaneous services > should start all services at once | TestInvoke_MultipleSimultaneousServices_ShouldStartAllServicesAtOnce | ported | JS L2678; describe-level `multiple` machine (L2632-2676) built inside the test |
| invoke > multiple simultaneous services > should run services in parallel | TestInvoke_MultipleSimultaneousServices_ShouldRunServicesInParallel | ported | JS L2755; describe-level `parallel` machine (L2693-2753) built inside the test |
| invoke > multiple simultaneous services > should not invoke an actor if it gets stopped immediately by transitioning away in immediate microstep | TestInvoke_MultipleSimultaneousServices_ShouldNotInvokeIfStoppedInImmediateMicrostep | ported | JS L2772 |
| invoke > multiple simultaneous services > should not invoke an actor if it gets stopped immediately by transitioning away in subsequent microstep | TestInvoke_MultipleSimultaneousServices_ShouldNotInvokeIfStoppedInSubsequentMicrostep | ported | JS L2802 |
| invoke > multiple simultaneous services > should invoke a service if other service gets stopped in subsequent microstep (#1180) | TestInvoke_MultipleSimultaneousServices_ShouldInvokeIfOtherServiceStoppedInSubsequentMicrostep | ported | JS L2840 |
| invoke > multiple simultaneous services > should invoke an actor when reentering invoking state within a single macrostep | TestInvoke_MultipleSimultaneousServices_ShouldInvokeWhenReenteringInvokingStateInOneMacrostep | ported | JS L2903 |
| invoke > invoke `src` can be used with invoke `input` | TestInvoke_SrcCanBeUsedWithInvokeInput | ported | JS L2939; input is `map[string]any` |
| invoke > invoke `src` can be used with dynamic invoke `input` | TestInvoke_SrcCanBeUsedWithDynamicInvokeInput | ported | JS L2986 |
| invoke > invoke generated ID should be predictable based on the state node where it is defined | TestInvoke_GeneratedIDShouldBePredictableBasedOnDefiningStateNode | ported | JS L3036 |
| invoke > invoke config defined as src with string reference should register unique and predictable child in state | TestInvoke_InvokeConfigAsSrcWithStringReference_ShouldRegisterUniquePredictableChild | ported | JS L3078 it.each row 1; shared body `invoke3AssertUniqueChild` |
| invoke > invoke config defined as src containing a machine directly should register unique and predictable child in state | TestInvoke_InvokeConfigAsSrcContainingMachine_ShouldRegisterUniquePredictableChild | ported | JS L3078 it.each row 2 (the commented-out `['machine', ...]` row at L3080 is not a test) |
| invoke > invoke config defined as src containing a callback actor directly should register unique and predictable child in state | TestInvoke_InvokeConfigAsSrcContainingCallback_ShouldRegisterUniquePredictableChild | ported | JS L3078 it.each row 3 |
| invoke > xstate.done.actor events should only select onDone transition on the invoking state when invokee is referenced using a string | TestInvoke_DoneActorEventsShouldOnlySelectOnDoneOnInvokingStateWithStringSrc | ported | JS L3122; `sleep(0)` → `sleep(10)` |
| invoke > xstate.done.actor events should have unique names when invokee is a machine with an id property | TestInvoke_DoneActorEventsShouldHaveUniqueNamesWhenInvokeeIsMachineWithID | ported | JS L3176; `sleep(0)` → `sleep(10)` |
| invoke > should get reinstantiated after reentering the invoking state in a microstep | TestInvoke_ShouldGetReinstantiatedAfterReenteringInvokingStateInMicrostep | ported | JS L3242 |
| invoke > invocations should be stopped when the machine reaches done state | TestInvoke_InvocationsShouldBeStoppedWhenMachineReachesDoneState | ported | JS L3270 |
| invoke > deep invocations should be stopped when the machine reaches done state | TestInvoke_DeepInvocationsShouldBeStoppedWhenMachineReachesDoneState | ported | JS L3298 |
| invoke > root invocations should restart on root reentering transitions | TestInvoke_RootInvocationsShouldRestartOnRootReenteringTransitions | ported | JS L3332; see ambiguity 2 |
| invoke > should be able to restart an invoke when reentering the invoking state | TestInvoke_ShouldRestartInvokeWhenReenteringInvokingState | ported | JS L3365 |
| invoke > should be able to receive a delayed event sent by the entry action of the invoking state | TestInvoke_ShouldReceiveDelayedEventSentByEntryActionOfInvokingState | ported | JS L3410; `sleep(3)` → `sleep(10)` |
| invoke input > should provide input to an actor creator | TestInvokeInput_ShouldProvideInputToAnActorCreator | ported | JS L3458 |
| invoke input > should provide self to input mapper | TestInvokeInput_ShouldProvideSelfToInputMapper | ported | JS L3517 |

Totals: 22 JS tests: 22 ported, 0 N/A-type, 0 N/A-runtime, 0 skipped-in-JS.

## API gaps

None. No `apigap_invoke_3.go` was created; every test is expressed with the existing contract.

## Ambiguities for review

1. JS L3172 / L3225 (`await sleep(0)`) and L3453 (`await sleep(3)` after a 1ms delayed send): Go promise actors and timers settle on other goroutines, so the Go tests wait `sleep(10)` (same approach as `actions_test.go:745`). The assertions are unchanged.
2. JS L3362 (`expect(count).toEqual(2)` right after `send`): JS runs the `fromPromise` creator synchronously; `logic.go` `FromPromise` runs it on its own goroutine. The Go test waits with `assert.Eventually` for `count >= 2`, sleeps 10ms, then asserts exactly 2 (same approach as `actor_test.go:760`).
3. JS L3226-3237: the expected order of the two `xstate.done.actor.*` events (first, then second) is kept. In Go the two grandchild promises resolve on separate goroutines, so the implementation must preserve deterministic order for this to pass.
4. JS L3122 (#464): the never-resolving second promise is modelled as blocking on `ctx.Done()` (the JS `signal`), returning `ctx.Err()` only after the actor stops.
5. JS L3078-3119 (`it.each`): ported as 3 Go test funcs sharing `invoke3AssertUniqueChild`. `toBeDefined()` → `assert.NotNil` on `Children["0.machine.a"]`.
6. JS L3519 (`expect(input.responder.send).toBeDefined()`): Go asserts `input["responder"]` is an `xs.ActorRef` (which has `Send`) and is non-nil.
7. JS L2633 / L2694 (`context: { one?: string; two?: string }`, initial `{}`): modelled as `invoke3OneTwoCtx{One, Two string}` with zero values; expected `{one: 'one', two: 'two'}` → `invoke3OneTwoCtx{One: "one", Two: "two"}`.
8. JS L3177: `Promise.withResolvers` is declared but unused in JS; not ported.
