> Historical translation record. Chunk tags, API-gap filenames, line numbers,
> and run counts below describe the original porting work. See the current
> [core guide](../core.md) and [architecture](../../ARCHITECTURE.md) for the maintained layout.

# store_broadcast_1 manifest

JS: `references/xstate/packages/xstate-store/test/broadcast.test.ts` lines 1-195 (5 tests).
Go: `store/broadcast_test.go` (tag `port_store_broadcast_1`).

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| broadcast storage > broadcasts queued persisted writes only after each async write completes | TestStoreBroadcast_BroadcastsQueuedPersistedWritesOnlyAfterEachAsyncWriteCompletes | ported | `await waitForMicrotask()` is `sleep(20)`; `await flushed` is `flushed.Wait()` (nil-safe). Shared slices guarded by a mutex (`-race`). |
| broadcast storage > broadcasts writes to other storage adapters on the same channel | TestStoreBroadcast_BroadcastsWritesToOtherStorageAdaptersOnTheSameChannel | ported | Expected message is `xstore.BroadcastMessage{Type: "xstate-store-update", Name: "counter"}` (value, not pointer). |
| broadcast storage > rehydrates subscribed stores when another tab writes persisted state | TestStoreBroadcast_RehydratesSubscribedStoresWhenAnotherTabWritesPersistedState | ported | |
| broadcast storage > does not rehydrate from unrelated storage names | TestStoreBroadcast_DoesNotRehydrateFromUnrelatedStorageNames | ported | Sender posts `xstore.BroadcastMessage` as the JS plain object. |
| broadcast storage > throws when subscribing a store without broadcast storage | TestStoreBroadcast_ThrowsWhenSubscribingAStoreWithoutBroadcastStorage | ported | `toThrow(string)` is a substring match, so `assert.Contains` on `panicMessage`. |

## Mock BroadcastChannel

The JS `MockBroadcastChannel` plus the `globalThis.BroadcastChannel` stub become a file-local
`storeBroadcast1Registry` (one per test, cleaned up with `t.Cleanup`, mirroring beforeEach/afterEach)
whose `New` method is passed as `BroadcastStorageOptions.NewChannel`. The channels implement
`xstore.BroadcastChannel`; `addEventListener`/`removeEventListener` map to `OnMessage` returning a Subscription.
Delivery is synchronous, as in JS.

## API gaps

None (no `apigap_store_broadcast_1.go`).

## Reviewer checks

- Test 1 relies on `FlushStorage` returning a promise that settles only after the queued second async write completes.
- The Go `BroadcastMessage` type is assumed to be the posted payload type (value); if the implementation posts a pointer or map, the assertion in test 2 and the filter in test 4 need adjusting.
- Test 3/4 delays are `sleep(20)` as a stand-in for `setTimeout(0)`; implementation rehydration is async.
- `-race`: not run (stubs panic).
