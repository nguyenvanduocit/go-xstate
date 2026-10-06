> Historical translation record. Chunk tags, API-gap filenames, line numbers,
> and run counts below describe the original porting work. See the current
> [core guide](../core.md) and [architecture](../../ARCHITECTURE.md) for the maintained layout.

## Source section: multiple_1

Source: `references/xstate/packages/core/test/multiple.test.ts` lines 1-213. Go file: `xstate/multiple_test.go` (tag `port_multiple_1`).

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| multiple > transitions to parallel states > should enter initial states of parallel states | TestMultiple_TransitionsToParallelStates_ShouldEnterInitialStatesOfParallelStates | ported | |
| multiple > transitions to parallel states > should enter specific states in one region | TestMultiple_TransitionsToParallelStates_ShouldEnterSpecificStatesInOneRegion | ported | |
| multiple > transitions to parallel states > should enter specific states in all regions | TestMultiple_TransitionsToParallelStates_ShouldEnterSpecificStatesInAllRegions | ported | |
| multiple > transitions to parallel states > should enter specific states in some regions | TestMultiple_TransitionsToParallelStates_ShouldEnterSpecificStatesInSomeRegions | ported | |
| multiple > transitions to parallel states > should reject two targets in the same region | TestMultiple_TransitionsToParallelStates_ShouldRejectTwoTargetsInTheSameRegion | skipped-in-JS | `it.skip`; body kept after `t.Skip` |
| multiple > transitions to parallel states > should reject targets inside and outside a region | TestMultiple_TransitionsToParallelStates_ShouldRejectTargetsInsideAndOutsideARegion | skipped-in-JS | `it.skip`; body kept after `t.Skip` |
| multiple > transitions to parallel states > should reject two targets in different regions | TestMultiple_TransitionsToParallelStates_ShouldRejectTwoTargetsInDifferentRegions | skipped-in-JS | `it.skip`; body kept after `t.Skip` |
| multiple > transitions to parallel states > should reject two targets in different regions at different levels | TestMultiple_TransitionsToParallelStates_ShouldRejectTwoTargetsInDifferentRegionsAtDifferentLevels | skipped-in-JS | `it.skip`; body kept after `t.Skip` |
| multiple > transitions to parallel states > should reject two deep targets in different regions at top level | TestMultiple_TransitionsToParallelStates_ShouldRejectTwoDeepTargetsInDifferentRegionsAtTopLevel | skipped-in-JS | `it.skip`; JS body duplicates the previous test (sends BROKEN_DIFFERENT_REGIONS_3), JS TODO at line 199 |
| multiple > transitions to parallel states > should reject two deep targets in different regions at different levels | TestMultiple_TransitionsToParallelStates_ShouldRejectTwoDeepTargetsInDifferentRegionsAtDifferentLevels | skipped-in-JS | `it.skip`; body kept after `t.Skip` |

Totals: 10 JS tests: 4 ported, 0 N/A-type, 0 N/A-runtime, 6 skipped-in-JS.

## API gaps

None. No `apigap_multiple_1.go` was created.

## Ambiguities for review

- JS lines 4-136: the shared `machine` is built once per describe in JS. In Go, `multiple1Machine()` builds a fresh machine for each test. The machine is stateless, so behaviour is the same.
- JS lines 10-22: single-element transition arrays with an array target (`[{ target: [...] }]`) map to `{{Targets: []string{...}}}`. The string shorthand (`DEEP_M`, `INITIAL`) maps to `Target`.
- Skipped tests use `toThrow()` with no message. They map to `assert.Panics`.
