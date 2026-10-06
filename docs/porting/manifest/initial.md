> Historical translation record. Chunk tags, API-gap filenames, line numbers,
> and run counts below describe the original porting work. See the current
> [core guide](../core.md) and [architecture](../../ARCHITECTURE.md) for the maintained layout.

## Source section: initial_1

Source: `references/xstate/packages/core/test/initial.test.ts` lines 1-165 (whole file, 164 lines).
Go file: `xstate/initial_test.go`.

Totals: 3 JS tests; 3 ported, 0 N/A-type, 0 N/A-runtime, 0 skipped-in-JS.

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| Initial states > should return the correct initial state | TestInitial_ShouldReturnTheCorrectInitialState | ported |  |
| Initial states > should return the correct initial state (parallel) | TestInitial_ShouldReturnTheCorrectInitialStateParallel | ported | Identical inline `foo`/`bar` configs built by a test-local `region` closure; structure unchanged. |
| Initial states > should return the correct initial state (deep parallel) | TestInitial_ShouldReturnTheCorrectInitialStateDeepParallel | ported | Same test-local `region` closure for the four identical `foo`/`bar` configs. |

## API gaps

None.

## Ambiguities for reviewers

- JS `toEqual` on `snapshot.value` is asserted against nested `map[string]any` / `string`, per the contract's `snap.Value` convention.
