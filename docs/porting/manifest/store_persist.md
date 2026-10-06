> Historical translation record. Chunk tags, API-gap filenames, line numbers,
> and run counts below describe the original porting work. See the current
> [core guide](../core.md) and [architecture](../../ARCHITECTURE.md) for the maintained layout.

# store_persist_1 manifest

JS file: `references/xstate/packages/xstate-store/test/persist.test.ts` lines 1-1292 (57 `it`/`it.each` entries; the entry at line 1293 "should migrate events when version differs" belongs to the next chunk).
Go file: `store/persist_test.go` (tag `port_store_persist_1`).

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| persistence lifecycle regressions > preserves an eligible snapshot when a nested event is filtered (throttle %i) | TestStorePersist_LifecycleRegressions_PreservesEligibleSnapshotWhenNestedEventIsFiltered | ported | it.each: one Go test, one subtest per parameter |
| persistence lifecycle regressions > orders new async writes after a queued clear | TestStorePersist_LifecycleRegressions_OrdersNewAsyncWritesAfterQueuedClear | ported |  |
| persistence lifecycle regressions > waits for writes queued by onDone callbacks | TestStorePersist_LifecycleRegressions_WaitsForWritesQueuedByOnDoneCallbacks | ported |  |
| persistence lifecycle regressions > restarts event history after clearStorage | TestStorePersist_LifecycleRegressions_RestartsEventHistoryAfterClearStorage | ported |  |
| persistence lifecycle regressions > reports a rejected initial async read (%s) | TestStorePersist_LifecycleRegressions_ReportsRejectedInitialAsyncRead | ported | it.each: one Go test, one subtest per parameter |
| persistence lifecycle regressions > applies pick once when flushing a throttled update | TestStorePersist_LifecycleRegressions_AppliesPickOnceWhenFlushingThrottledUpdate | ported |  |
| persistence lifecycle regressions > preserves updates triggered by onDone during a throttled flush (%s) | TestStorePersist_LifecycleRegressions_PreservesUpdatesTriggeredByOnDoneDuringThrottledFlush | ported | it.each: one Go test, one subtest per parameter |
| persistence lifecycle regressions > cancels buffered writes when storage is cleared (%s) | TestStorePersist_LifecycleRegressions_CancelsBufferedWritesWhenStorageIsCleared | ported | it.each: one Go test, one subtest per parameter |
| persistence lifecycle regressions > removes storage after an in-flight async write finishes | TestStorePersist_LifecycleRegressions_RemovesStorageAfterInFlightAsyncWriteFinishes | ported |  |
| persist > should persist context to storage after each event | TestStorePersist_Persist_ShouldPersistContextToStorageAfterEachEvent | ported |  |
| persist > should restore context from storage on creation | TestStorePersist_Persist_ShouldRestoreContextFromStorageOnCreation | ported |  |
| persist > should set _persist.hydrated to true on sync hydration | TestStorePersist_Persist_ShouldSetHydratedToTrueOnSyncHydration | ported |  |
| persist > should set _persist.hydrated to true when storage is empty | TestStorePersist_Persist_ShouldSetHydratedToTrueWhenStorageIsEmpty | ported |  |
| persist > should preserve _persist metadata across transitions | TestStorePersist_Persist_ShouldPreservePersistMetadataAcrossTransitions | ported |  |
| persist - pick > should only persist selected fields | TestStorePersist_Pick_ShouldOnlyPersistSelectedFields | ported |  |
| persist - pick > should merge picked data with full context on restore | TestStorePersist_Pick_ShouldMergePickedDataWithFullContextOnRestore | ported |  |
| persist - version + migrate > should migrate persisted state when version differs | TestStorePersist_VersionMigrate_ShouldMigratePersistedStateWhenVersionDiffers | ported | persisted version read back from JSON is float64; test compares via fmt.Sprint for numeric versions |
| persist - version + migrate > should migrate with string versions | TestStorePersist_VersionMigrate_ShouldMigrateWithStringVersions | ported | persisted version read back from JSON is float64; test compares via fmt.Sprint for numeric versions |
| persist - version + migrate > should not migrate when version matches | TestStorePersist_VersionMigrate_ShouldNotMigrateWhenVersionMatches | ported | persisted version read back from JSON is float64; test compares via fmt.Sprint for numeric versions |
| persist - merge > should use custom merge strategy | TestStorePersist_Merge_ShouldUseCustomMergeStrategy | ported |  |
| persist - serialize / deserialize > should use custom serializer and deserializer | TestStorePersist_SerializeDeserialize_ShouldUseCustomSerializerAndDeserializer | ported |  |
| persist - throttle > should batch writes with throttle | TestStorePersist_Throttle_ShouldBatchWritesWithThrottle | ported | vi.useFakeTimers -> xs.NewSimulatedClock via PersistOptions.Clock |
| persist - throttle > should not write again if no events between throttle intervals | TestStorePersist_Throttle_ShouldNotWriteAgainIfNoEventsBetweenThrottleIntervals | ported | vi.useFakeTimers -> xs.NewSimulatedClock via PersistOptions.Clock |
| persist - flushStorage > should force immediate write of pending throttled context | TestStorePersist_FlushStorage_ShouldForceImmediateWriteOfPendingThrottledContext | ported | vi.useFakeTimers -> xs.NewSimulatedClock via PersistOptions.Clock |
| persist - flushStorage > should throw when store has no persist extension | TestStorePersist_FlushStorage_ShouldThrowWhenStoreHasNoPersistExtension | ported |  |
| persist - flushStorage > should return a promise for async storage flushes | TestStorePersist_FlushStorage_ShouldReturnAPromiseForAsyncStorageFlushes | ported | vi.useFakeTimers -> xs.NewSimulatedClock via PersistOptions.Clock |
| persist - onDone / onError > should call onDone after successful write | TestStorePersist_OnDoneOnError_ShouldCallOnDoneAfterSuccessfulWrite | ported | onDone data compared by JSON form (struct or map accepted) |
| persist - onDone / onError > should call onDone with picked context when pick is used | TestStorePersist_OnDoneOnError_ShouldCallOnDoneWithPickedContextWhenPickIsUsed | ported | onDone data compared by JSON form (struct or map accepted) |
| persist - onDone / onError > should call onError on write failure | TestStorePersist_OnDoneOnError_ShouldCallOnErrorOnWriteFailure | ported | onDone data compared by JSON form (struct or map accepted) |
| persist - onDone / onError > should call onError on read failure during hydration | TestStorePersist_OnDoneOnError_ShouldCallOnErrorOnReadFailureDuringHydration | ported | onDone data compared by JSON form (struct or map accepted) |
| persist - filter > should skip persisting when filter returns false | TestStorePersist_Filter_ShouldSkipPersistingWhenFilterReturnsFalse | ported |  |
| persist - skipHydration > should not hydrate when skipHydration is true | TestStorePersist_SkipHydration_ShouldNotHydrateWhenSkipHydrationIsTrue | ported |  |
| persist - rehydrateStore > should rehydrate from sync storage | TestStorePersist_RehydrateStore_ShouldRehydrateFromSyncStorage | ported |  |
| persist - rehydrateStore > should rehydrate from async storage | TestStorePersist_RehydrateStore_ShouldRehydrateFromAsyncStorage | ported |  |
| persist - rehydrateStore > should merge with events sent before rehydration | TestStorePersist_RehydrateStore_ShouldMergeWithEventsSentBeforeRehydration | ported |  |
| persist - rehydrateStore > should handle empty storage gracefully | TestStorePersist_RehydrateStore_ShouldHandleEmptyStorageGracefully | ported |  |
| persist - rehydrateStore > should apply migration during rehydration | TestStorePersist_RehydrateStore_ShouldApplyMigrationDuringRehydration | ported |  |
| persist - rehydrateStore > should throw when store has no persist extension | TestStorePersist_RehydrateStore_ShouldThrowWhenStoreHasNoPersistExtension | ported | JS promise rejects; Go contract documents a panic; test accepts either (panic or rejected promise) and checks the message |
| persist - clearStorage > should remove persisted data from storage | TestStorePersist_ClearStorage_ShouldRemovePersistedDataFromStorage | ported |  |
| persist - clearStorage > should throw when store has no persist extension | TestStorePersist_ClearStorage_ShouldThrowWhenStoreHasNoPersistExtension | ported |  |
| persist - clearStorage > should return a promise for async storage removals | TestStorePersist_ClearStorage_ShouldReturnAPromiseForAsyncStorageRemovals | ported |  |
| persist - createJSONStorage > should return noop storage when getStorage throws | TestStorePersist_CreateJSONStorage_ShouldReturnNoopStorageWhenGetStorageThrows | ported |  |
| persist - createJSONStorage > should wrap a working storage adapter | TestStorePersist_CreateJSONStorage_ShouldWrapAWorkingStorageAdapter | ported |  |
| persist - createJSONStorage > should preserve async storage semantics | TestStorePersist_CreateJSONStorage_ShouldPreserveAsyncStorageSemantics | ported |  |
| persist - SSR-safe defaults > should not require localStorage when using default storage | TestStorePersist_SSRSafeDefaults_ShouldNotRequireLocalStorageWhenUsingDefaultStorage | ported | JS stubs a throwing globalThis.localStorage; Go default (nil Storage) is the no-op storage, so test uses Storage unset |
| persist - SSR-safe defaults > should detect persist rehydrate event collisions in development | TestStorePersist_SSRSafeDefaults_ShouldDetectPersistRehydrateEventCollisionsInDevelopment | ported |  |
| persist - async storage auto-detection > should detect async storage and skip sync hydration | TestStorePersist_AsyncStorageAutoDetection_ShouldDetectAsyncStorageAndSkipSyncHydration | ported | asyncMemoryStorage resolves immediately; the IsHydrated==false check right after With relies on hydration completing off the calling path (JS: microtask) |
| persist - composability > should work with undoRedo extension | TestStorePersist_Composability_ShouldWorkWithUndoRedoExtension | ported |  |
| persist - composability > should persist and restore correctly across store instances | TestStorePersist_Composability_ShouldPersistAndRestoreCorrectlyAcrossStoreInstances | ported |  |
| persist - strategy: event > should persist events to storage | TestStorePersist_StrategyEvent_ShouldPersistEventsToStorage | ported |  |
| persist - strategy: event > should restore state by replaying events | TestStorePersist_StrategyEvent_ShouldRestoreStateByReplayingEvents | ported |  |
| persist - strategy: event > should restore state with event payloads | TestStorePersist_StrategyEvent_ShouldRestoreStateWithEventPayloads | ported |  |
| persist - strategy: event > should set _persist.hydrated to true on sync hydration | TestStorePersist_StrategyEvent_ShouldSetHydratedToTrueOnSyncHydration | ported |  |
| persist - strategy: event > should respect maxEvents option | TestStorePersist_StrategyEvent_ShouldRespectMaxEventsOption | ported |  |
| persist - strategy: event > should not hydrate when skipHydration is true | TestStorePersist_StrategyEvent_ShouldNotHydrateWhenSkipHydrationIsTrue | ported |  |
| persist - strategy: event > should rehydrate from async storage | TestStorePersist_StrategyEvent_ShouldRehydrateFromAsyncStorage | ported |  |
| persist - strategy: event > should continue accumulating events after rehydration | TestStorePersist_StrategyEvent_ShouldContinueAccumulatingEventsAfterRehydration | ported |  |

## API gaps

None. No `apigap_store_persist_1.go` was created.

## Ambiguities for a reviewer

- `PersistOptions.OnDone` / `OnError` take `any`; assertions on onDone data compare the JSON form, so either a typed context or a decoded map passes. `Pick` returns `map[string]any` or a struct.
- `Migrate` receives the persisted value as `any` and `version` as `any`; the stored version is a JSON number (float64) when read back, so tests compare numeric versions via `fmt.Sprint`; string versions compare directly.
- Throttle tests use `xs.NewSimulatedClock()` through `PersistOptions.Clock` in place of vi fake timers; the two lifecycle tests with real timers in JS (`throttle 100`, `pick once`) also use a simulated clock because they only use `flushStorage`.
- Asynchronous waits that JS does with `await Promise.resolve()`: "reports a rejected initial async read" polls until onError fires then sleeps 20ms to confirm it fired exactly once; "async storage ... clearStorage" uses `sleep(1)`.
- Event-strategy replay passes events decoded from JSON, so numeric payloads arrive as float64; the `add` handler accepts int or float64.
- Context field `Items []string` in the merge test: the stored `items` array is merged through JSON overlay; the test assumes the default Go serializer is encoding/json.
- Verified: `go vet -tags port_store_persist_1 ./store/` is clean, `gofmt -l store/` is clean, and `go test -tags port_store_persist_1` fails with `store: not implemented` panics (no build errors).


## Source section: store_persist_2

Source: `references/xstate/packages/xstate-store/test/persist.test.ts` lines 1293-1616.
Go file: `store/persist_test.go`. No apigap file needed.

Totals: 10 JS tests; 10 ported, 0 N/A-type, 0 N/A-runtime, 0 skipped-in-JS.

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| persist - strategy: event > should migrate events when version differs | TestStorePersist_StrategyEvent_ShouldMigrateEventsWhenVersionDiffers | ported | `migrate` -> `MigrateEvents`; `{...e, type: 'inc'}` copies the `xs.E` map and rewrites `type`. |
| persist - strategy: event > should work with clearStorage | TestStorePersist_StrategyEvent_ShouldWorkWithClearStorage | ported | |
| persist - strategy: event > should handle empty storage gracefully | TestStorePersist_StrategyEvent_ShouldHandleEmptyStorageGracefully | ported | |
| persist - strategy: event > should call onError on read failure during hydration | TestStorePersist_StrategyEvent_ShouldCallOnErrorOnReadFailureDuringHydration | ported | throwing `getItem` -> `GetItemFunc` that panics with `errors.New("read failed")`; `toHaveBeenCalled` -> `assert.Positive(count)`. |
| persist - strategy: event > should work with throttle | TestStorePersist_StrategyEvent_ShouldWorkWithThrottle | ported | `vi.useFakeTimers()` / `advanceTimersByTime(100)` -> `xs.NewSimulatedClock()` passed as `Clock`, `clock.Increment(ms(100))`. |
| persist - strategy: event > should work with flushStorage | TestStorePersist_StrategyEvent_ShouldWorkWithFlushStorage | ported | same fake-clock mapping; `Throttle: ms(1000)`. |
| persist committed %s writes > persists nested events in commit order (throttle %i, %s) | TestStorePersist_CommittedWrites_PersistsNestedEventsInCommitOrder | ported | `describe.each` x `it.each` -> one Go test, 8 subtests `<strategy>/throttle N, <nestedFrom>`. Real clock (JS uses real timers, no `Clock` set). `await flushStorage` -> `Wait` via a timeout helper. `enqueue.effect(({trigger}) => trigger.inner())` -> `enq.Effect(func(e){ e.Trigger("inner") })`. |
| persist committed %s writes > does not buffer %s evaluations | TestStorePersist_CommittedWrites_DoesNotBufferEvaluations | ported | 6 subtests `<strategy>/<evaluation>`. `advanceTimersByTimeAsync(100)` -> `clock.Increment(ms(100))`. `toThrow(StoreValidationError)` -> `recovered` + `assert.ErrorAs`. Context schema via `zObject({count: zNumberMax(1)})`. |
| persist committed %s writes > flushes throttled data after an already pending asynchronous write | TestStorePersist_CommittedWrites_FlushesThrottledDataAfterAlreadyPendingAsynchronousWrite | ported | 2 subtests (strategy). Pending `setItem` promise from `xstore.NewPromise`; microtask settling after `advanceTimersByTimeAsync` -> `assert.Eventually` on `len(writes)` (positive) and `sleep(20)` before the final "no third write" check. |
| persist committed %s writes > orders asynchronous writes and recovers after failure: %s | TestStorePersist_CommittedWrites_OrdersAsynchronousWritesAndRecoversAfterFailure | ported | 4 subtests `<strategy>/<failFirst>`. `Promise.resolve(flushed).then(completed)` -> goroutine that waits on `flushed` then calls the spy. `vi.waitFor(len == 2)` -> `assert.Eventually`. `flushed` `toBeInstanceOf(Promise)` -> `assert.NotNil`. |

## API gaps
None.

## Review notes
- Event strategy test "does not buffer evaluations" asserts `saved.checkpoint` is `null` (key present). The contract struct `PersistEventStorageValue.Checkpoint` carries `json:"checkpoint,omitempty"`, which would drop a nil checkpoint. The Go test asserts the key is present with a nil value (faithful to `toBeNull`, which fails on `undefined`); the implementation must write `"checkpoint": null` for it to pass.
- Handlers always return `(next, true)`; no test relies on snapshot identity.
- Persisted event `{type: 'inner'}` is compared as `[]any{map[string]any{"type": "inner"}}` (JSON-decoded form); numbers are `float64`.
- `strategy: 'snapshot'` runs reuse `storePersist2EventStore` for the two async-write tests; `Strategy` selects the behaviour (`PersistSnapshot` = `"snapshot"`), matching the JS `persist({ strategy })`.
- Timing: JS relies on microtask flushing (`await`); the Go port uses polling with a 2s ceiling for positive conditions and a short `sleep(20)` for the single negative check ("still 2 writes"), so that assertion can only pass vacuously if persist writes from a goroutine slower than 20ms.
