> Historical translation record. Chunk tags, API-gap filenames, line numbers,
> and run counts below describe the original porting work. See the current
> [core guide](../core.md) and [architecture](../../ARCHITECTURE.md) for the maintained layout.

## Source section: select_1

Source: `references/xstate/packages/core/test/select.test.ts` lines 1-276 (whole file, 275 lines).
Go file: `xstate/select_test.go`.

Totals: 6 JS tests; 6 ported, 0 N/A-type, 0 N/A-runtime, 0 skipped-in-JS.

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| select > should get current value | TestSelect_ShouldGetCurrentValue | ported |  |
| select > should subscribe to changes | TestSelect_ShouldSubscribeToChanges | ported | `toHaveBeenCalledWith(43)` -> `assert.Contains(spy.Calls(), []any{43})`. |
| select > should not notify if selected value has not changed | TestSelect_ShouldNotNotifyIfSelectedValueHasNotChanged | ported | `not.toHaveBeenCalled()` -> count 0. |
| select > should support custom equality function | TestSelect_ShouldSupportCustomEqualityFunction | ported | Selector returns a local `selected{Name, Age}` struct. |
| select > should unsubscribe correctly | TestSelect_ShouldUnsubscribeCorrectly | ported |  |
| select > should handle updates with multiple subscribers | TestSelect_ShouldHandleUpdatesWithMultipleSubscribers | ported | `toHaveBeenLastCalledWith` -> compare the last entry of `spy.Calls()` (local closure `lastCall`). Unused JS `interface PositionContext` (L163-168) is dropped: it is type-only. |

## API gaps (`apigap_select_1.go`)

- `type Readable[T any] interface { Subscribable[T]; Get() T }`: mirrors JS `Readable<T>` (types.ts:2050).
- `func Select[S Snapshot, T any](actor *Actor[S], selector func(S) T, equal ...func(a, b T) bool) Readable[T]`:
  mirrors `actor.select(selector, equalityFn)` (createActor.ts:493). It is a package-level function because Go
  methods cannot have type parameters.

## Ambiguities for review

- The JS default equality is `Object.is` (createActor.ts:495). The gap documents the Go default as `==` for
  comparable values and `reflect.DeepEqual` otherwise. In JS the selected `position` object is compared by reference;
  in Go it is a value struct. The assertions hold either way: every `UPDATE_POSITION` in the test (L227, L241,
  L252) carries a different value, and `UPDATE_USER` (L264) leaves `position` unchanged in both reference and value.
- JS `selection.subscribe(callback)` passes a plain function. Go uses `Subscribe(xs.Observer[T]{Next: ...})`, the
  only form `Subscribable[T]` provides. The gap does not add a `SubscribeNext` variant.
- Event payloads are typed Go values inside `xs.E` (`"position": position{...}`, `"user": user{...}`). The assign
  actions read them back with type assertions.
