> Historical translation record. Chunk tags, API-gap filenames, line numbers,
> and run counts below describe the original porting work. See the current
> [core guide](../core.md) and [architecture](../../ARCHITECTURE.md) for the maintained layout.

## Source section: store_persist_clear_1

Source: `references/xstate/packages/xstate-store/test/persistClear.test.ts` lines 1-385 (whole file).
Go file: `store/persist_clear_test.go`.

Totals: 10 JS tests (`it` / `it.each`); 10 ported, 0 N/A-type, 0 N/A-runtime, 0 skipped-in-JS.
Each `it.each` is one Go test with one `t.Run` subtest per JS case (throttle 0/100, can/transition, snapshot/event).

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| clearStorage event history > starts a new event log from the live context (throttle %i) | TestStorePersistClear_StartsNewEventLogFromLiveContext | ported | Subtests `throttle 0`, `throttle 100`. `vi.useFakeTimers`/`advanceTimersByTime` -> `xs.NewSimulatedClock()` passed as `PersistOptions.Clock`, `clock.Increment(ms(100))`. `Object.freeze` has no Go equivalent (dropped; identity via `assert.Same` and content assertions kept). `clearStorage({getSnapshot})` ported as `ClearStorage(store)` (Go takes `*Store`). |
| clearStorage event history > replaces an existing checkpoint and still truncates the new log correctly | TestStorePersistClear_ReplacesExistingCheckpointAndTruncatesNewLog | ported | |
| clearStorage event history > does not consume the reset during %s evaluations | TestStorePersistClear_DoesNotConsumeResetDuringEvaluations | ported | Subtests `can`, `transition`. |
| clearStorage event history > keeps history cleared when rehydrating empty storage | TestStorePersistClear_KeepsHistoryClearedWhenRehydratingEmptyStorage | ported | `await rehydrateStore` -> `RehydrateStore(store).Wait()`. |
| clearStorage event history > keeps newly hydrated history after clearing an earlier log | TestStorePersistClear_KeepsNewlyHydratedHistoryAfterClearingEarlierLog | ported | |
| clearStorage event history > starts a fresh log after each clear, including repeated clears without an event | TestStorePersistClear_StartsFreshLogAfterEachClearIncludingRepeatedClears | ported | |
| clearStorage event history > invalidates %s persistence effects computed before clearing | TestStorePersistClear_InvalidatesPersistenceEffectsComputedBeforeClearing | ported | Subtests `snapshot`, `event`. `typeof effect === 'function'` -> `effect.Run != nil`, called as `Run(nil)`. |
| clearStorage event history > discards a %s read that started before clearing | TestStorePersistClear_DiscardsReadThatStartedBeforeClearing | ported | Subtests `snapshot`, `event`. Deferred read via `xstore.NewPromise[*string]`; `finishRead` captured under a mutex. Assumes `RehydrateStore` calls `GetItem` synchronously (as JS does), otherwise `finishRead` is nil and the test fails at `require.NotNil`. |
| clearStorage event history > does not restore a pre-clear %s from its deferred persistence effect | TestStorePersistClear_DoesNotRestorePreClearFromDeferredPersistenceEffect | ported | Subtests `snapshot`, `event`. `ClearStorage` is called from inside a store subscriber, so the implementation must tolerate re-entrancy. |
| clearStorage event history > orders new history after an in-flight write and asynchronous removal | TestStorePersistClear_OrdersNewHistoryAfterInFlightWriteAndAsyncRemoval | ported | `removalStarted` promise -> `newSignal()`; `await cleared` -> `cleared.Wait()`; `await flushStorage` -> `FlushStorage(store).Wait()`. Storage built with `xstore.StorageFuncs`; shared state mutex-guarded for `-race`. |

## API gaps

None (no `apigap_store_persist_clear_1.go`).

## Reviewer notes

- Context type uses json tag `count`; events are `xs.E{"amount": int}`, so the handler reads `ev.(xs.E)["amount"].(int)`.
- Expected stored JSON is compared as parsed `map[string]any` (numbers `float64`), through `storedJSON`.
- `assert.Same(beforeClear, store.GetSnapshot())` after `ClearStorage` checks that clearing does not replace the snapshot. No handler returns the same context object, so the identity limitation in docs/porting/store.md does not apply.
