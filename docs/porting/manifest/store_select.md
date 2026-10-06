> Historical translation record. Chunk tags, API-gap filenames, line numbers,
> and run counts below describe the original porting work. See the current
> [core guide](../core.md) and [architecture](../../ARCHITECTURE.md) for the maintained layout.

# store_select_1 — `xstate-store/src/select.test.ts` lines 1-244

Go file: `store/select_test.go` (tag `port_store_select_1`). No apigap file needed.

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| select > should get current value | TestStoreSelect_ShouldGetCurrentValue | ported | `store.select(fn).get()` -> `xstore.Select(s, fn).Get()` |
| select > should subscribe to changes | TestStoreSelect_ShouldSubscribeToChanges | ported | `toHaveBeenCalledWith('Jane')` -> `assert.Contains(calls, []any{"Jane"})` |
| select > should not notify if selected value has not changed | TestStoreSelect_ShouldNotNotifyIfSelectedValueHasNotChanged | ported | |
| select > should support custom equality function | TestStoreSelect_ShouldSupportCustomEqualityFunction | ported | custom `equalityFn` passed as the variadic `equal` arg of `Select` |
| select > should unsubscribe correctly | TestStoreSelect_ShouldUnsubscribeCorrectly | ported | `subscription.unsubscribe()` -> `Unsubscribe()` |
| select > should handle updates with multiple subscribers | TestStoreSelect_ShouldHandleUpdatesWithMultipleSubscribers | ported | `toHaveBeenLastCalledWith` -> last element of `spy.Calls()`; event payloads carry Go struct values (`position`, `storeSelect1User`), read back via type assertion |

## API gaps
None.

## Review notes
- Selected `position` struct is comparable, so the default `Object.is`-style equality (`==`) matches JS where a new `{x, y}` object with a changed `y` is a different reference; the "y only changed" step expects `renderCallback` to fire (count 3) and `loggerCallback` not to, which holds under `==` on both selectors.
- JS `toHaveBeenCalledWith` checks any call; ported as `assert.Contains` on calls where the spy had exactly one call (count asserted first).
- Handlers always return `(next, true)`; none of these tests relies on snapshot identity.
