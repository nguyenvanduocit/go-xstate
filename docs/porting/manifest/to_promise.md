> Historical translation record. Chunk tags, API-gap filenames, line numbers,
> and run counts below describe the original porting work. See the current
> [core guide](../core.md) and [architecture](../../ARCHITECTURE.md) for the maintained layout.

## Source section: to_promise_1

Source: `references/xstate/packages/core/test/toPromise.test.ts` lines 1-142.
Go file: `xstate/to_promise_test.go`.

Totals: 6 JS tests; 5 ported, 0 N/A-type, 0 N/A-runtime, 1 skipped-in-JS.

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| toPromise > should be awaitable | TestToPromise_ShouldBeAwaitable | ported | `result satisfies number` (JS L10) is type-only; runtime value asserted with Equal(42). |
| toPromise > should await actors | TestToPromise_ShouldAwaitActors | ported | `types.output` and `data satisfies` are type-only; `setTimeout(...,1)` → `time.AfterFunc(ms(1), ...)`. |
| toPromise > should await already done actors | TestToPromise_ShouldAwaitAlreadyDoneActors | ported | type-only `satisfies` dropped. |
| toPromise > should handle errors | TestToPromise_ShouldHandleErrors | skipped-in-JS | `it.skip` (JS L72); body translated after `t.Skip`. |
| toPromise > should immediately resolve for a done actor | TestToPromise_ShouldImmediatelyResolveForADoneActor | ported | |
| toPromise > should immediately reject for an actor that had an error | TestToPromise_ShouldImmediatelyRejectForAnActorThatHadAnError | ported | `rejects.toEqual(new Error('oh noes'))` → `assert.Equal(errors.New("oh noes"), err)` from `Wait()`. |

## API gaps

None. Uses existing `xs.ToPromise(actor).Wait()` (`util.go`).

## Ambiguities for review

- Machine `output: { count: 42 }` is a static `map[string]any{"count": 42}`; expectations compare against the same map type. If the implementation normalizes outputs differently, the expectation type is the thing to check.
- Error equality (JS L136, L138): `assert.Equal(errors.New("oh noes"), ...)` requires the snapshot `Error` and the promise rejection to be the panicked `*errors.errorString` value itself (not wrapped). JS `toEqual` on Error compares message/name, so a wrapping implementation would need `EqualError` instead.
