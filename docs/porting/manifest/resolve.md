> Historical translation record. Chunk tags, API-gap filenames, line numbers,
> and run counts below describe the original porting work. See the current
> [core guide](../core.md) and [architecture](../../ARCHITECTURE.md) for the maintained layout.

## Source section: resolve_1

Source: `references/xstate/packages/core/test/resolve.test.ts` lines 1-74 (file has 73 lines).
Go file: `xstate/resolve_test.go`.

Totals: 1 JS test; 1 ported, 0 N/A-type, 0 N/A-runtime, 0 skipped-in-JS.

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| resolve() > should resolve parallel states with flat child states | TestResolve_ShouldResolveParallelStatesWithFlatChildStates | ported | Top-level `flatParallelMachine` (JS lines 5-58) copied as helper `resolve1FlatParallelMachine`. `flatParallelMachine.root` -> `.Root`. JS `{}` for atomic parallel regions -> `map[string]any{}`. |

## API gaps (`apigap_resolve_1.go`)

- `ResolveStateValue(root *StateNode, value StateValue) StateValue` mirrors internal `resolveStateValue` from `src/stateUtils.ts:1870` (imported by the test from `../src/stateUtils`).

## Ambiguities

- JS line 72 expects atomic children of parallel states to resolve to `{}` (not a string). The Go expectation uses an empty `map[string]any{}` for those; the implementation must produce exactly that (not `nil`) for `assert.Equal` to pass.
