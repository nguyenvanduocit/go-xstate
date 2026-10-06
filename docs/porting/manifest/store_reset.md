> Historical translation record. Chunk tags, API-gap filenames, line numbers,
> and run counts below describe the original porting work. See the current
> [core guide](../core.md) and [architecture](../../ARCHITECTURE.md) for the maintained layout.

# store_reset_1 manifest

Source: `references/xstate/packages/xstate-store/test/reset.test.ts` lines 1-176.
Go: `store/reset_test.go` (tag `port_store_reset_1`).

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| reset extension > should reset to initial context | TestStoreReset_ShouldResetToInitialContext | ported | |
| reset extension > should reset multiple fields to initial context | TestStoreReset_ShouldResetMultipleFieldsToInitialContext | ported | `toEqual` on struct context |
| reset extension > should support partial reset via `to` option | TestStoreReset_ShouldSupportPartialResetViaToOption | ported | `user: string \| null` is `*string` (nil = null) |
| reset extension > should be idempotent when no changes have been made | TestStoreReset_ShouldBeIdempotentWhenNoChangesHaveBeenMade | ported | |
| reset extension > should preserve snapshot status | TestStoreReset_ShouldPreserveSnapshotStatus | ported | |
| reset extension > should notify subscribers on reset | TestStoreReset_ShouldNotifySubscribersOnReset | ported | |
| reset extension > should work with undoRedo (reset is undoable) | TestStoreReset_ShouldWorkWithUndoRedoResetIsUndoable | ported | |
| reset extension > should allow resetting after multiple operations | TestStoreReset_ShouldAllowResettingAfterMultipleOperations | ported | |
| reset extension > should detect reset event collisions in development | TestStoreReset_ShouldDetectResetEventCollisionsInDevelopment | ported | `toThrow(string)` is a substring match, so `assert.Contains` on `panicMessage`; construction and `.With(Reset)` both sit inside the panicking closure |
| reset extension > should return initial snapshot from getInitialSnapshot | TestStoreReset_ShouldReturnInitialSnapshotFromGetInitialSnapshot | ported | |

## API gaps

None (`apigap_store_reset_1.go` not created).

## Reviewer notes

- Collision test: JS handler `reset: (ctx) => ctx` ported as `return c, true`. The snapshot-identity limitation does not matter (the test only checks the thrown message).
- `setName`/`login` read payloads via `ev.(xs.E)["name"].(string)`, assuming `Trigger` delivers an `xs.E` event with the payload keys.
