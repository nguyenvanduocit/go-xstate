> Historical translation record. Chunk tags, API-gap filenames, line numbers,
> and run counts below describe the original porting work. See the current
> [core guide](../core.md) and [architecture](../../ARCHITECTURE.md) for the maintained layout.

## Source section: id_1

Source: `references/xstate/packages/core/test/id.test.ts` lines 1-216 (whole file).
Go file: `xstate/id_test.go`.

The file has 4 literal `it()` calls (lines 76, 117, 150, 186) plus `testAll(idMachine, expected)`
(line 74, defined in `test/utils.ts`), which generates one `it()` per (fromState, eventTypes) pair of
`expected` (lines 57-72): 2 + 1 + 1 + 2 = 6 tests. Total: 10 JS tests. Rows follow `Object.keys`
order (no integer-like keys, so source order).

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| State node IDs > should go from A to {"A":"bar"} on NEXT | TestId_StateNodeIDs_ShouldGoFromAToABarOnNEXT | ported | testAll-generated; fromState "A" resolves to A.foo |
| State node IDs > should go from A to {"B":"bar"} on NEXT_DOT_RESOLVE | TestId_StateNodeIDs_ShouldGoFromAToBBarOnNEXT_DOT_RESOLVE | ported | testAll-generated; target `#B.bar` (ID + relative path) |
| State node IDs > should go from {"A":"foo"} to {"A":"bar"} on NEXT | TestId_StateNodeIDs_ShouldGoFromAFooToABarOnNEXT | ported | testAll-generated |
| State node IDs > should go from {"A":"bar"} to {"B":"foo"} on NEXT | TestId_StateNodeIDs_ShouldGoFromABarToBFooOnNEXT | ported | testAll-generated |
| State node IDs > should go from {"B":"foo"} to {"A":"foo"} on NEXT,NEXT | TestId_StateNodeIDs_ShouldGoFromBFooToAFooOnNEXT_NEXT | ported | testAll-generated; multi-event chain |
| State node IDs > should go from {"B":"foo"} to {"B":"dot"} on NEXT_DOT | TestId_StateNodeIDs_ShouldGoFromBFooToBDotOnNEXT_DOT | ported | testAll-generated |
| State node IDs > should work with ID + relative path | TestId_StateNodeIDs_ShouldWorkWithIDPlusRelativePath | ported | |
| State node IDs > should work with keys that have escaped periods | TestId_StateNodeIDs_ShouldWorkWithKeysThatHaveEscapedPeriods | ported | |
| State node IDs > should work with IDs that have escaped periods | TestId_StateNodeIDs_ShouldWorkWithIDsThatHaveEscapedPeriods | ported | |
| State node IDs > should not treat escaped backslash as period's escape | TestId_StateNodeIDs_ShouldNotTreatEscapedBackslashAsPeriodsEscape | ported | |

## API gaps

None. Uses `StateMachine.ResolveState`, `xs.GetNextSnapshot`, `xs.GetInitialSnapshot`,
`xs.MatchesState`, `xs.CreateActor`.

## Ambiguities for review

- `testAll` / `testMultiTransition` / `resolveSerializedStateValue` from `test/utils.ts` are copied as
  file-local helpers with prefix `id1` (same shape as `examples6161*` helpers in
  `xstate/spec_examples_test.go`); candidates for consolidation into `xstate/helpers_test.go` at integration.
- The JS `idMachine` is shared at module scope; Go builds a fresh machine per test via
  `id1IDMachine()` (machine config is stateless).
- JS string escapes are translated to their runtime values using Go raw strings:
  `'foo\\.bar'` (line 123) → `` `foo\.bar` ``; `'#foo\\.bar'` (line 156) → `` `#foo\.bar` ``;
  `'#some\\\\.thing'` (line 192) → `` `#some\\.thing` ``; `'some\\.thing'` (line 196) → `` `some\.thing` ``;
  `'some\\'` (line 199) → `` `some\` ``.
- Line 95: state `quux` has the literal id `'#bar.qux.quux'` (with leading `#`); kept verbatim.
- `resolveState({ value, context: {} })`: context `{}` is passed as `map[string]any{}` with `C = any`.
- The `undefined` and `string` branches of testAll are unused by this file's data but kept for fidelity.
