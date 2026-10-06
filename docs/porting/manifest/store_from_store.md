> Historical translation record. Chunk tags, API-gap filenames, line numbers,
> and run counts below describe the original porting work. See the current
> [core guide](../core.md) and [architecture](../../ARCHITECTURE.md) for the maintained layout.

## Source section: store_from_store_1

Source: `references/xstate/packages/xstate-store/test/fromStore.test.ts` lines 1-118.
Go file: `store/from_store_test.go`.

Totals: 4 JS tests; 4 ported, 0 N/A-type, 0 N/A-runtime, 0 skipped-in-JS.

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| fromStore > creates an actor from store logic with input | TestStoreFromStore_CreatesAnActorFromStoreLogicWithInput | ported | `context: (count) => ({count})` -> `ContextFn` reading `input.(int)`. |
| fromStore > emits events | TestStoreFromStore_EmitsEvents | ported | `schemas.emitted.increased` via `zObject({"upBy": zNumber()})`; `toHaveBeenCalledWith` -> exact `spy.Calls()` equality with `xs.E{"type":"increased","upBy":8}`. |
| fromStore > enq.getSnapshot() in a sync effect reflects the post-transition state (matches createStore) | TestStoreFromStore_EnqGetSnapshotInASyncEffectReflectsThePostTransitionStateMatchesCreateStore | ported | `seen` is a `*int` so "never set" (undefined) fails `NotNil` instead of comparing to 0. |
| fromStore > enq.getSnapshot() in an async effect reflects the latest committed state | TestStoreFromStore_EnqGetSnapshotInAnAsyncEffectReflectsTheLatestCommittedState | ported | A channel releases the async effect after `bump` commits; the test receives the observed count with a bounded wait. `bump` handler returns `(ctx, true)`. |

## API gaps

None (no `apigap_store_from_store_1.go`).

## Reviewer notes

- Handlers use the whole-context return convention; no JS assertion depends on snapshot identity (`toBe`), so the "same snapshot on `return ctx`" limitation does not apply.
- JS handler `bump: (ctx) => ({count: ctx.count + 10})` ignores the event; Go keeps the same signature with `_` params.
- Input type: ContextFn asserts `input.(int)`; `xs.WithInput(42)` passes an `int`.
