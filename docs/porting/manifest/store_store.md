> Historical translation record. Chunk tags, API-gap filenames, line numbers,
> and run counts below describe the original porting work. See the current
> [core guide](../core.md) and [architecture](../../ARCHITECTURE.md) for the maintained layout.

# store_store_1 — `xstate-store/test/store.test.ts` lines 1-1537

Go file: `store/store_test.go` (tag `port_store_store_1`). Gap file: `store/apigap_store_store_1.go`.

Test names are `TestStoreStore_<CamelCaseOfItName>`; tests under `describe('store.trigger')` / `describe('store.transition')` / `describe('types')` carry the `Trigger_` / `Transition_` / `Types_` segment.

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| processes triggered events breadth-first when handlers append more events | TestStoreStore_ProcessesTriggeredEventsBreadthFirstWhenHandlersAppendMoreEvents | ported | `start` returns undefined -> `(c, false)`; payload ids are `int` |
| updates a store with an event without mutating original context | TestStoreStore_UpdatesAStoreWithAnEventWithoutMutatingOriginalContext | ported |  |
| can update context | TestStoreStore_CanUpdateContext | ported |  |
| handles unknown events sent via store.send (does not do anything) | TestStoreStore_HandlesUnknownEventsSentViaStoreSendDoesNotDoAnything | ported | `{type:"unknown"} as any` -> `xs.Ev("unknown")` |
| updates state from sent events | TestStoreStore_UpdatesStateFromSentEvents | ported |  |
| can be observed | TestStoreStore_CanBeObserved | ported |  |
| does not expose atom internals at runtime | TestStoreStore_DoesNotExposeAtomInternalsAtRuntime | N/A-runtime | `'_snapshot' in store` probes JS object own-properties; Go struct fields are unexported |
| exposes schemas at runtime | TestStoreStore_ExposesSchemasAtRuntime | ported | `toBe(schemas)` -> `assert.Same` on `*StoreSchemas` |
| exposes schemas after extension | TestStoreStore_ExposesSchemasAfterExtension | ported | `.with(reset())` -> `.With(xstore.Reset[...]())` |
| can be inspected | TestStoreStore_CanBeInspected | ported | `objectContaining` -> field-by-field asserts on `StoreInspectionEvent` (Type, Event, Snapshot.Context); `require.Len(evs, 2)` first |
| inspection with @statelyai/inspect typechecks correctly | TestStoreStore_InspectionWithStatelyaiInspectTypechecksCorrectly | ported | JS asserts nothing. `createBrowserInspector` (@statelyai/inspect) has no Go form and is dropped; the runtime part (empty store + `store.Inspect(fn)`) runs as a smoke test (`NotPanics`, then `Unsubscribe`) |
| emitted events can be subscribed to | TestStoreStore_EmittedEventsCanBeSubscribedTo | ported | `toHaveBeenCalledWith` -> `assert.Contains(spy.Calls(), ...)` |
| emitted events can be unsubscribed to | TestStoreStore_EmittedEventsCanBeUnsubscribedTo | ported |  |
| emitted events occur after the snapshot is updated | TestStoreStore_EmittedEventsOccurAfterTheSnapshotIsUpdated | ported | `expect.assertions(1)` -> handler appends to a slice, asserted to equal `[1]` (exactly one call, snapshot already updated) |
| events can be emitted with no payload | TestStoreStore_EventsCanBeEmittedWithNoPayload | ported | `hasPayload` handler (never triggered, @ts-expect-error in JS) is kept as `enq.Emit("expectsPayload")` |
| events can be emitted with optional payloads (type check) | TestStoreStore_EventsCanBeEmittedWithOptionalPayloadsTypeCheck | ported | JS asserts nothing and never invokes the handler. Runtime part ported as a smoke test: `CreateStore` with an emitted schema `{payload: optional string}` and the three `Emit` forms (no payload, `{payload}`, `{}`) in an uninvoked handler. The `emit.optionalPayload('foo')` @ts-expect-error call has no Go form and is dropped |
| effects can be enqueued | TestStoreStore_EffectsCanBeEnqueued | ported | `setTimeout(5)` -> goroutine blocked on a `gate` signal that the test resolves right after the synchronous `Count == 1` assertion (no timing dependence); the final `await setTimeout(10)` becomes `done.Wait(t)`; shared store accessed from two goroutines (impl must be race-safe) |
| events can be enqueued from transitions | TestStoreStore_EventsCanBeEnqueuedFromTransitions | ported | `return ctx` -> `(c, true)`; snapshot identity not asserted |
| effect-only transitions should execute effects | TestStoreStore_EffectOnlyTransitionsShouldExecuteEffects | ported | `enq.effect(spy)` -> effect closure calling `spy.Call()`; handler returns `(c, false)` |
| emits-only transitions should emit events | TestStoreStore_EmitsOnlyTransitionsShouldEmitEvents | ported |  |
| checks whether events can transition | TestStoreStore_ChecksWhetherEventsCanTransition | ported | `noop: (ctx) => ctx` -> `(c, true)` (allowed); handlers returning undefined -> `(c, false)`; `store.can.x(p)` -> `store.Can("x", p)` |
| checks whether Immer transitions can transition without changing context | TestStoreStore_ChecksWhetherImmerTransitionsCanTransitionWithoutChangingContext | ported | Immer `produce` has no Go equivalent: replaced by a value copy (`next := c; next.Count += by`) returning `(next, true)`; runtime behavior (can/trigger, `toBe(snapshot)` -> `assert.Same` after `Can`) fully ported |
| wildcard listener receives all emitted events | TestStoreStore_WildcardListenerReceivesAllEmittedEvents | ported |  |
| wildcard listener can be unsubscribed | TestStoreStore_WildcardListenerCanBeUnsubscribed | ported |  |
| wildcard listener is called after specific listener | TestStoreStore_WildcardListenerIsCalledAfterSpecificListener | ported |  |
| async effects can be enqueued | TestStoreStore_AsyncEffectsCanBeEnqueued | ported | async effect -> goroutine + `gate` + signal (see "effects can be enqueued") |
| effects receive an enqueue object to trigger events (no closure needed) | TestStoreStore_EffectsReceiveAnEnqueueObjectToTriggerEventsNoClosureNeeded | ported | `createStoreLogic({context: () => ...})` -> `CreateStoreLogic` with `ContextFn`; async effect -> goroutine + `gate` + signal |
| effects can read fresh state after awaiting via enq.getSnapshot() | TestStoreStore_EffectsCanReadFreshStateAfterAwaitingViaEnqGetSnapshot | ported | async effect -> goroutine blocked on a `gate` resolved after `bump`; `seen` is appended before `e.Trigger("done")` (JS order) and read after `done.Wait`; `done: ctx` -> `(c, true)` |
| sync effects read the current transition snapshot via enq.getSnapshot() | TestStoreStore_SyncEffectsReadTheCurrentTransitionSnapshotViaEnqGetSnapshot | ported |  |
| effects can use enq.send to dispatch events | TestStoreStore_EffectsCanUseEnqSendToDispatchEvents | ported | `await setTimeout(0)` -> `sleep(5)` |
| rejects async handlers in createStoreTransition(...) | TestStoreStore_RejectsAsyncHandlersInCreateStoreTransition | ported (PARTIAL) | First assertion (`store.transition(snapshot, {type:"bad"})` does not throw) ported as `assert.NotPanics`. Second assertion (`createStoreTransition({bad: async ...})` throws `Async transition unsupported here`) is N/A-runtime and NOT ported: Go handlers are synchronous (docs/porting/store.md "Untranslatable tests"), there is no async assigner form. Not skipped so the first half stays active. An additive sync check (`CreateStoreTransition` with a sync "bad" handler does not panic and returns Count 1) keeps `CreateStoreTransition` exercised; it is not a JS assertion. See "Partial ports" |
| store.trigger > should allow triggering events with a fluent API | TestStoreStore_Trigger_ShouldAllowTriggeringEventsWithAFluentApi | ported |  |
| store.trigger > should provide type safety for event payloads | TestStoreStore_Trigger_ShouldProvideTypeSafetyForEventPayloads | ported | runtime part ported (valid `reset()` and `increment({by:1})` triggers; JS has no assertions); the `if (false)` @ts-expect-error block never runs in JS |
| store.trigger > should be equivalent to store.send | TestStoreStore_Trigger_ShouldBeEquivalentToStoreSend | ported | `vi.spyOn(store, "send")` cannot be expressed on Go methods (no gap method invented for it). Ported as observational equivalence: two identical stores, one driven by `Trigger("increment", {by:5})`, one by `Send({type:"increment", by:5})`; the inspected event sequence must equal exactly `[@xstate.init, {type:"increment", by:5}]` on both and the contexts must match (Count 5). Still not a call spy: an implementation where `Trigger` bypasses `Send` but feeds the same transition would pass |
| store.trigger > should fail fast for unknown trigger names on config-based stores | TestStoreStore_Trigger_ShouldFailFastForUnknownTriggerNamesOnConfigBasedStores | ported | `Object.keys(store.trigger)` -> API gap `Store.EventTypes()` (sorted); `toThrow(TypeError)` -> panic message check via `panicMessage`: non-empty, not the stub's "not implemented", and containing the unknown event name `unknown` (Go has no TypeError; the message-names-the-trigger rule is a contract this test pins) |
| store.trigger > should include extension events in the concrete trigger object | TestStoreStore_Trigger_ShouldIncludeExtensionEventsInTheConcreteTriggerObject | ported | `Object.keys(store.trigger)` -> gap `EventTypes()`; sorted order equals JS insertion order here (`increment`, `reset`) |
| store.trigger > should include schema-declared events in the concrete trigger object | TestStoreStore_Trigger_ShouldIncludeSchemaDeclaredEventsInTheConcreteTriggerObject | ported | `Object.keys(store.trigger)` -> gap `EventTypes()`; sorted order equals JS insertion order here |
| works with typestates | TestStoreStore_WorksWithTypestates | N/A-type | `satisfies` / @ts-expect-error narrowing only |
| the emit type is not overridden by the payload | TestStoreStore_TheEmitTypeIsNotOverriddenByThePayload | ported | drawer is `*drawer`; the whole incoming event (type `openDrawer`) is passed to `Emit` and the emitted type must stay `drawerOpened` |
| store.transition > returns next state and effects for a given state and event | TestStoreStore_Transition_ReturnsNextStateAndEffectsForAGivenStateAndEvent | ported | `effects[0]` toEqual event -> `effects[0].Emitted` |
| store.transition > returns unchanged state and empty effects for unknown events | TestStoreStore_Transition_ReturnsUnchangedStateAndEmptyEffectsForUnknownEvents | ported | `toBe(currentState)` -> `assert.Same`; `toEqual([])` -> `assert.Empty` (nil or empty slice both accepted) |
| store.transition > collects enqueued effects | TestStoreStore_Transition_CollectsEnqueuedEffects | ported | `typeof effects[0] === "function"` -> `effects[0].Run != nil` |
| store.transition > resolves enqueued trigger events and collects effects in pure transitions | TestStoreStore_Transition_ResolvesEnqueuedTriggerEventsAndCollectsEffectsInPureTransitions | ported | `effect()` -> `effect.Run(nil)`; `toHaveBeenNthCalledWith` -> `spy.Calls()[i]`; `incTwice: return ctx` -> `(c, true)` |
| can be created with a logic object | TestStoreStore_CanBeCreatedWithALogicObject | ported | `CreateStoreFromLogic` with `GetInitialSnapshot`/`Transition`; `EventTypes` left nil so `Trigger("unknown")` (an un-guarded call in JS) must not panic; `satisfies` lines are type-only |
| can select from a store | TestStoreStore_CanSelectFromAStore | ported | `store.select` -> `xstore.Select`; `toHaveBeenCalledWith` -> `assert.Contains` |
| can create reusable store logic with selectors | TestStoreStore_CanCreateReusableStoreLogicWithSelectors | ported | `context.missing` (@ts-expect-error) is type-only and dropped; selector results are `any` (`int`) |
| preserves selectors through store extensions | TestStoreStore_PreservesSelectorsThroughStoreExtensions | ported |  |
| should not trigger update if the snapshot is the same | TestStoreStore_ShouldNotTriggerUpdateIfTheSnapshotIsTheSame | ported | See "Snapshot identity". JS `(ctx) => ctx` keeps the identical snapshot so subscribers are not notified. Ported as `return c, true` with 0 subscriber calls, per docs/porting/store.md:109-111 ("port the assertion and note it in the manifest") |
| should not trigger update if the snapshot is the same even if there are effects | TestStoreStore_ShouldNotTriggerUpdateIfTheSnapshotIsTheSameEvenIfThereAreEffects | ported | Same as above (an enqueued effect does not make the snapshot change); see "Snapshot identity" |
| types > AnyStoreConfig | TestStoreStore_Types_AnyStoreConfig | N/A-type | type-only (AnyStoreConfig, @ts-expect-error) |
| types > EventFromStoreConfig | TestStoreStore_Types_EventFromStoreConfig | N/A-type | type-only (`satisfies`) |
| types > ContextFromStoreConfig | TestStoreStore_Types_ContextFromStoreConfig | N/A-type | type-only (`satisfies`) |
| types > generics can be provided | TestStoreStore_Types_GenericsCanBeProvided | ported | runtime calls ported (`Emit` with/without payload, including the @ts-expect-error `beansGround()` call which runs in JS); JS has no assertions; the `if (false)` block never runs |
| types > localizes TypeScript errors to the specific transition | TestStoreStore_Types_LocalizesTypeScriptErrorsToTheSpecificTransition | N/A-type | only a @ts-expect-error check |
| emitted events work with store extensions | TestStoreStore_EmittedEventsWorkWithStoreExtensions | ported |  |

Totals: 55 JS tests = 49 ported (1 partial: rejects async handlers), 5 N/A-type, 1 N/A-runtime, 0 skipped-in-JS.

## API gaps

- `store/apigap_store_store_1.go`: `func (s *Store[C]) EventTypes() []string` — mirrors `Object.keys(store.trigger)` (sorted event types accepted by `Trigger`/`Can`: `on` keys, `schemas.events` keys, extension events such as `reset`; for logic-object stores, `StoreLogic.EventTypes`). Used by three `store.trigger` tests. If the contract exposes this differently, replace the three call sites. The method returns sorted names; JS `Object.keys` returns insertion order, which coincides with sorted order in all three tests (`increment`, `reset`).

## Partial ports

- `rejects async handlers in createStoreTransition(...)`: the `createStoreTransition({bad: async ...})` -> `toThrow('Async transition unsupported here')` assertion (store.test.ts:846-862) is N/A-runtime and not ported; only the first assertion is. Counted as `ported` because the first half runs.

## Snapshot identity

The two `should not trigger update if the snapshot is the same ...` tests port `(ctx) => ctx` as `return c, true` and assert 0 subscriber calls, as docs/porting/store.md:109-111 prescribes for tests that rely on `return ctx` keeping the snapshot. JS decides with `nextContext !== currentContext` (store.ts:851): an identical context object keeps `currentSnapshot` and notifies nobody.

Implementation requirement these tests pin: when a handler returns `(ctx, true)` and the returned context equals the current context (`==`/deep equality; value-typed `storeStore1Counter` here), the snapshot is NOT replaced and subscribers are NOT called, with or without enqueued effects. This narrows the sentence "A snapshot is replaced whenever a handler returns `(ctx, true)`" in docs/porting/store.md:109-110. `allowed` (used by `Can`) is unaffected: `noop: (ctx) => ctx` still counts as allowed (JS `assignerResult !== undefined`). Rewriting the tests as `(c, false)` is rejected because it would stop exercising the returned-same-context path.

## Reviewer notes

- `Trigger("unknown")` on a config-based store must panic with a message naming the event; `Can("unavailable")` for a schema-declared event without handler is asserted `false` (no panic). `Trigger` on a logic-object store with nil `EventTypes` is asserted not to panic.
- `store.can.noop()` (`(ctx) => ctx`) is ported as `(c, true)` and asserted allowed; handlers that JS leaves returning `undefined` are `(c, false)`.
- Async effects run in goroutines that wait on a `gate` signal released after the synchronous assertions instead of fixed sleeps, so the "before" assertions do not depend on scheduler timing. The store is accessed from multiple goroutines, so the implementation needs a mutex; run with `-race` once the implementation exists.
- `effects can use enq.send to dispatch events` keeps `sleep(5)` for `await setTimeout(0)`; it only waits and asserts after the fact.
- `emitted events can be unsubscribed to` and others use `assert.Contains(spy.Calls(), []any{event})` for `toHaveBeenCalledWith`; call counts are asserted separately where JS does.
