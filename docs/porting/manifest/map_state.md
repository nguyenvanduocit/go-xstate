> Historical translation record. Chunk tags, API-gap filenames, line numbers,
> and run counts below describe the original porting work. See the current
> [core guide](../core.md) and [architecture](../../ARCHITECTURE.md) for the maintained layout.

# map_state_1 — packages/core/test/mapState.test.ts (lines 1-524)

Go file: `xstate/map_state_test.go`. Gap file: `apigap_map_state_1.go`.

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| mapState > should map context from root state | TestMapState_ShouldMapContextFromRootState | ported | JS L4 |
| mapState > should map context from nested states | TestMapState_ShouldMapContextFromNestedStates | ported | JS L26. `results.find(...)?.result` → local `find` returning (value, found); found asserted then value compared |
| mapState > should only call mappers for active states | TestMapState_ShouldOnlyCallMappersForActiveStates | ported | JS L72 |
| mapState > should work with parallel states | TestMapState_ShouldWorkWithParallelStates | ported | JS L106 |
| mapState > should handle states without mappers | TestMapState_ShouldHandleStatesWithoutMappers | ported | JS L165 |
| mapState > should work with final states | TestMapState_ShouldWorkWithFinalStates | ported | JS L206 |
| mapState > should include stateNode in results | TestMapState_ShouldIncludeStateNodeInResults | ported | JS L238. `path toEqual([])` → `assert.Empty` (nil or empty slice) |
| mapState > type safety > should accept valid state keys | TestMapState_TypeSafety_ShouldAcceptValidStateKeys | ported | JS L276. No explicit assertion in JS; runtime call ported as `assert.NotPanics` |
| mapState > type safety > should error on invalid state keys | TestMapState_TypeSafety_ShouldErrorOnInvalidStateKeys | ported | JS L310. `@ts-expect-error` on key `nonexistent` dropped (Go map keys are untyped strings); runtime call ported as `assert.NotPanics` |
| mapState > type safety > should error on invalid nested state keys | TestMapState_TypeSafety_ShouldErrorOnInvalidNestedStateKeys | ported | JS L340. `@ts-expect-error` on key `invalidChild` dropped; runtime call ported as `assert.NotPanics` |
| mapState > type safety > should infer snapshot type in map function | TestMapState_TypeSafety_ShouldInferSnapshotTypeInMapFunction | ported | JS L380. Typed locals are checked by the Go compiler; runtime call as `assert.NotPanics` |
| mapState > type safety > should enforce consistent TResult type across all map functions | TestMapState_TypeSafety_ShouldEnforceConsistentTResultTypeAcrossAllMapFunctions | ported | JS L405. `StateMapper[ctx, int]` fixes TResult; runtime call as `assert.NotPanics` |
| mapState > type safety > should error when nested map returns wrong type | TestMapState_TypeSafety_ShouldErrorWhenNestedMapReturnsWrongType | N/A-type | JS L441. Only check is `@ts-expect-error` (boolean returned where number expected); Go generics make it uncompilable |
| mapState > type safety > should error when deeply nested map returns wrong type | TestMapState_TypeSafety_ShouldErrorWhenDeeplyNestedMapReturnsWrongType | N/A-type | JS L467. Only check is `@ts-expect-error` (number returned where string expected); uncompilable in Go |
| mapState > type safety > should infer result type in return value | TestMapState_TypeSafety_ShouldInferResultTypeInReturnValue | ported | JS L503. `satisfies number` → `var _ int = results[0].Result`; `@ts-expect-error ... satisfies string` dropped. `require.NotEmpty` mirrors JS throwing on `results[0]` being undefined |

Totals: 15 JS tests; 13 ported, 2 N/A-type, 0 N/A-runtime, 0 skipped-in-JS.

## API gaps (`apigap_map_state_1.go`)

- `type StateMapper[C, R any] struct { Map func(*MachineSnapshot[C]) R; States map[string]StateMapper[C, R] }` mirrors `StateSchemaMapper`.
- `type StateMapResult[R any] struct { StateNode *StateNode; Result R }` mirrors `{ stateNode, result }`.
- `func MapSnapshot[C, R any](s *MachineSnapshot[C], mapper StateMapper[C, R]) []StateMapResult[R]` mirrors v5 `mapState(snapshot, mapper)` (`packages/core/src/mapState.ts`). The contract name `MapState` (util.go:47) is taken by the v4-style `mapState(stateMap, stateId)` signature, which does not match the v5 API these tests exercise.

## For reviewer

- The contract `xs.MapState` (util.go:46-47) has a different signature from v5 `mapState` in `references/xstate/packages/core/src/mapState.ts:35`. At integration, decide whether to replace the contract `MapState` with `MapSnapshot` or keep both.
- JS L64-66: root state node key is expected to be `'(machine)'` (default machine id); Go test asserts `StateNode.Key == "(machine)"` for a machine with no `ID`.
- Type-safety tests (JS L276-522) with no explicit runtime expectation were ported as `assert.NotPanics` around the call (vitest fails a test that throws); only the two whose sole content is an uncompilable-in-Go `@ts-expect-error` were marked N/A-type.
