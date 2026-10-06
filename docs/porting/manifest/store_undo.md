> Historical translation record. Chunk tags, API-gap filenames, line numbers,
> and run counts below describe the original porting work. See the current
> [core guide](../core.md) and [architecture](../../ARCHITECTURE.md) for the maintained layout.

# store_undo_1 manifest

Source: `references/xstate/packages/xstate-store/test/undo.test.ts` lines 1-1244 (whole file).
Go: `store/undo_test.go` (tag `port_store_undo_1`).

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| preserves persistence metadata through snapshot undo and redo (%s) | TestStoreUndo_PreservesPersistenceMetadataThroughSnapshotUndoAndRedo | ported | `it.each` over 2 orders becomes `t.Run` subtests inside one Go test; `storage` is `StorageFuncs` with spies; `expect(() => flushStorage).not.toThrow()` is `assert.NotPanics` |
| preserves live extension metadata through custom restore triggers | TestStoreUndo_PreservesLiveExtensionMetadataThroughCustomRestoreTriggers | ported | `toHaveBeenCalledExactlyOnceWith` is `Count()==1` plus exact args `["counter", toJSON({context:{count:1},version:0})]`; context has json tag `count` |
| keeps metadata updates produced by custom restore triggers | TestStoreUndo_KeepsMetadataUpdatesProducedByCustomRestoreTriggers | ported | Symbol-keyed `revision` becomes `ExtensionState["revision"]` (int), copy-on-write via `storeUndo1WithExt`; custom extension overrides `GetInitialSnapshot` and `Transition` |
| should undo a single event | TestStoreUndo_ShouldUndoASingleEvent | ported |  |
| should redo a previously undone event | TestStoreUndo_ShouldRedoAPreviouslyUndoneEvent | ported |  |
| should undo/redo multiple events, non-transactional | TestStoreUndo_ShouldUndoRedoMultipleEventsNonTransactional | ported |  |
| should group events by transaction ID | TestStoreUndo_ShouldGroupEventsByTransactionID | ported |  |
| should maintain correct state when interleaving undo/redo with new events | TestStoreUndo_ShouldMaintainCorrectStateWhenInterleavingUndoRedoWithNewEvents | ported |  |
| should do nothing when undoing with empty history | TestStoreUndo_ShouldDoNothingWhenUndoingWithEmptyHistory | ported | `toEqual(initialSnapshot)` is `assert.Equal` on snapshot pointers (deep equality of pointees) |
| should do nothing when redoing with empty undo stack | TestStoreUndo_ShouldDoNothingWhenRedoingWithEmptyUndoStack | ported | `toEqual(initialSnapshot)` is `assert.Equal` on snapshot pointers (deep equality of pointees) |
| should clear redo stack when new events occur after undo | TestStoreUndo_ShouldClearRedoStackWhenNewEventsOccurAfterUndo | ported |  |
| should preserve emitted events during undo/redo | TestStoreUndo_ShouldPreserveEmittedEventsDuringUndoRedo | ported |  |
| should preserve context and event types | TestStoreUndo_ShouldPreserveContextAndEventTypes | N/A-type | Only `satisfies` / `@ts-expect-error`; no runtime assertions |
| should skip non-undoable events during undo | TestStoreUndo_ShouldSkipNonUndoableEventsDuringUndo | ported | handler `(ctx) => ctx` returns `(c, true)`; Go always creates a new snapshot (docs/porting/store.md limitation), JS identity is not asserted here |
| should skip non-redoable events during redo | TestStoreUndo_ShouldSkipNonRedoableEventsDuringRedo | ported | handler `(ctx) => ctx` returns `(c, true)`; Go always creates a new snapshot (docs/porting/store.md limitation), JS identity is not asserted here |
| should skip events with transaction grouping | TestStoreUndo_ShouldSkipEventsWithTransactionGrouping | ported | handler `(ctx) => ctx` returns `(c, true)`; Go always creates a new snapshot (docs/porting/store.md limitation), JS identity is not asserted here |
| should handle mixed undoable and non-undoable events | TestStoreUndo_ShouldHandleMixedUndoableAndNonUndoableEvents | ported | handler `(ctx) => ctx` returns `(c, true)`; Go always creates a new snapshot (docs/porting/store.md limitation), JS identity is not asserted here |
| should not replay emitted events for skipped events during undo/redo | TestStoreUndo_ShouldNotReplayEmittedEventsForSkippedEventsDuringUndoRedo | ported |  |
| should skip events with transaction grouping (second test of this name; getTransactionId reads the snapshot) | TestStoreUndo_ShouldSkipEventsWithTransactionGrouping_2 | ported | Second JS test with the same name (JS L486). `transactionId: string | null` is `*string`; null maps to `""` in `GetTransactionID` |
| should use the snapshot in the skipEvent function | TestStoreUndo_ShouldUseTheSnapshotInTheSkipEventFunction | ported |  |
| emit event types should be correct | TestStoreUndo_EmitEventTypesShouldBeCorrect | N/A-type | Only `@ts-expect-error` / `satisfies`; handlers never run |
| should detect undo/redo event collisions in development | TestStoreUndo_ShouldDetectUndoRedoEventCollisionsInDevelopment | ported | `toThrow(string)` is a substring match: `assert.Contains` on `panicMessage` |
| undoRedo with snapshot strategy > undoRedo with snapshot strategy > should undo a single event | TestStoreUndo_Snapshot_ShouldUndoASingleEvent | ported |  |
| undoRedo with snapshot strategy > undoRedo with snapshot strategy > should redo a previously undone event | TestStoreUndo_Snapshot_ShouldRedoAPreviouslyUndoneEvent | ported |  |
| undoRedo with snapshot strategy > undoRedo with snapshot strategy > should undo/redo multiple events, non-transactional | TestStoreUndo_Snapshot_ShouldUndoRedoMultipleEventsNonTransactional | ported |  |
| undoRedo with snapshot strategy > undoRedo with snapshot strategy > should undo back into history after a redo | TestStoreUndo_Snapshot_ShouldUndoBackIntoHistoryAfterARedo | ported |  |
| undoRedo with snapshot strategy > undoRedo with snapshot strategy > should group events by transaction ID | TestStoreUndo_Snapshot_ShouldGroupEventsByTransactionID | ported |  |
| undoRedo with snapshot strategy > undoRedo with snapshot strategy > should undo back into history after redoing a transaction | TestStoreUndo_Snapshot_ShouldUndoBackIntoHistoryAfterRedoingATransaction | ported |  |
| undoRedo with snapshot strategy > undoRedo with snapshot strategy > should maintain correct state when interleaving undo/redo with new events | TestStoreUndo_Snapshot_ShouldMaintainCorrectStateWhenInterleavingUndoRedoWithNewEvents | ported |  |
| undoRedo with snapshot strategy > undoRedo with snapshot strategy > should do nothing when undoing with empty history | TestStoreUndo_Snapshot_ShouldDoNothingWhenUndoingWithEmptyHistory | ported |  |
| undoRedo with snapshot strategy > undoRedo with snapshot strategy > should do nothing when redoing with empty future stack | TestStoreUndo_Snapshot_ShouldDoNothingWhenRedoingWithEmptyFutureStack | ported |  |
| undoRedo with snapshot strategy > undoRedo with snapshot strategy > should clear redo stack when new events occur after undo | TestStoreUndo_Snapshot_ShouldClearRedoStackWhenNewEventsOccurAfterUndo | ported |  |
| undoRedo with snapshot strategy > undoRedo with snapshot strategy > should skip non-undoable events | TestStoreUndo_Snapshot_ShouldSkipNonUndoableEvents | ported | handler `(ctx) => ctx` returns `(c, true)`; Go always creates a new snapshot (docs/porting/store.md limitation), JS identity is not asserted here |
| undoRedo with snapshot strategy > undoRedo with snapshot strategy > should respect historyLimit | TestStoreUndo_Snapshot_ShouldRespectHistoryLimit | ported |  |
| undoRedo with snapshot strategy > undoRedo with snapshot strategy > should apply historyLimit during redo | TestStoreUndo_Snapshot_ShouldApplyHistoryLimitDuringRedo | ported |  |
| undoRedo with snapshot strategy > undoRedo with snapshot strategy > should preserve context with skipped events | TestStoreUndo_Snapshot_ShouldPreserveContextWithSkippedEvents | ported |  |
| undoRedo with snapshot strategy > undoRedo with snapshot strategy > should handle transaction grouping with historyLimit | TestStoreUndo_Snapshot_ShouldHandleTransactionGroupingWithHistoryLimit | ported |  |
| undoRedo with snapshot strategy > undoRedo with snapshot strategy > should use compare function to skip duplicate snapshots | TestStoreUndo_Snapshot_ShouldUseCompareFunctionToSkipDuplicateSnapshots | ported | handler `(ctx) => ctx` returns `(c, true)`; Go always creates a new snapshot (docs/porting/store.md limitation), JS identity is not asserted here |
| undoRedo with snapshot strategy > undoRedo with snapshot strategy > should save all snapshots when no compare function is provided | TestStoreUndo_Snapshot_ShouldSaveAllSnapshotsWhenNoCompareFunctionIsProvided | ported | handler `(ctx) => ctx` returns `(c, true)`; Go always creates a new snapshot (docs/porting/store.md limitation), JS identity is not asserted here |
| undoRedo with snapshot strategy > undoRedo with snapshot strategy > should preserve orthogonal context during undo and redo | TestStoreUndo_Snapshot_ShouldPreserveOrthogonalContextDuringUndoAndRedo | ported |  |
| undoRedo with snapshot strategy > undoRedo with snapshot strategy > should pass current, next, and direction to restore | TestStoreUndo_Snapshot_ShouldPassCurrentNextAndDirectionToRestore | ported |  |
| undoRedo with snapshot strategy > undoRedo with snapshot strategy > should restore once for a transaction group | TestStoreUndo_Snapshot_ShouldRestoreOnceForATransactionGroup | ported |  |
| undoRedo with snapshot strategy > undoRedo with snapshot strategy > should run emitted events after restored context commits | TestStoreUndo_Snapshot_ShouldRunEmittedEventsAfterRestoredContextCommits | ported |  |
| undoRedo with snapshot strategy > undoRedo with snapshot strategy > should run effects after restored context commits | TestStoreUndo_Snapshot_ShouldRunEffectsAfterRestoredContextCommits | ported |  |
| undoRedo with snapshot strategy > undoRedo with snapshot strategy > should apply triggered transitions and effects once without history | TestStoreUndo_Snapshot_ShouldApplyTriggeredTransitionsAndEffectsOnceWithoutHistory | ported |  |
| undoRedo with snapshot strategy > undoRedo with snapshot strategy > should not execute enqueued effects when checking can | TestStoreUndo_Snapshot_ShouldNotExecuteEnqueuedEffectsWhenCheckingCan | ported |  |
| undoRedo with snapshot strategy > undoRedo with snapshot strategy > should infer restore context, emitted events, and triggers | TestStoreUndo_Snapshot_ShouldInferRestoreContextEmittedEventsAndTriggers | N/A-type | Only `satisfies` / `@ts-expect-error`; restore callback never runs |

## API gaps

None (`store/apigap_store_undo_1.go` not needed).

## Reviewer notes

- `it.each(...)` test (1 JS test, 2 cases) is one Go test with `t.Run` subtests `persist-first` / `undo-first`.
- Counts: 47 JS tests: 44 ported, 3 N/A-type, 0 N/A-runtime, 0 skipped-in-JS.
- Handler `(ctx) => ctx` (log/noop) maps to `return c, true`. Go always makes a new snapshot for `(ctx, true)` (limitation in docs/porting/store.md). The undo tests never assert snapshot identity (`toBe`), so assertions are unchanged; the compare/noop tests rely on the undo extension recording a history entry per non-skipped event regardless of identity, as in JS undo.ts.
- Context fields use the Go zero value for `logs: [] as string[]`: tests use `[]string{}` (non-nil) so `assert.Equal(t, []string{}, ...)` distinguishes empty from nil as JS `toEqual([])` does.
- Payloads from `Trigger` keep Go types: `by`, `viewport` are `int` and are read with `.(int)`.
- `getTransactionId` returning null/undefined is `""` (`UndoRedoOptions.GetTransactionID`).
- Emitted event payloads compare as `xs.E{"type": ..., "value": 1}` with `int` values; if the implementation normalizes numbers differently, adjust here.
- `restore` spy test compares `xstore.UndoRedoRestoreArgs[ctx]` values (Current/Next/Direction) instead of JS object literals.
- Effect spies (`vi.fn()` passed to `enqueue.effect`) are wrapped: `enq.Effect(func(e){ effect.Call(e) })`.
