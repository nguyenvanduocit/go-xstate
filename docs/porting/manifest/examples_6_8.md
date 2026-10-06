> Historical translation record. Chunk tags, API-gap filenames, line numbers,
> and run counts below describe the original porting work. See the current
> [core guide](../core.md) and [architecture](../../ARCHITECTURE.md) for the maintained layout.

## Source section: examples_6_8_1

Source: `references/xstate/packages/core/test/examples/6.8.test.ts` lines 1-80 (whole file, 79 lines).
Go file: `xstate/spec_examples_test.go`.

The file has one literal `it()` (line 70). `testAll(machine, expected)` (line 68, defined in
`test/utils.ts`) generates one `it()` per (fromState, eventTypes) pair of `expected` (lines 37-66):
2 + 3 + 3 + 3 + 4 + 1 = 16 tests. Total: 17. JS `Object.keys` puts integer-like keys first in
ascending order, then `FAKE`; rows follow that order. Test names use `JSON.stringify(toState)`, so
string targets appear quoted (`"A.C"`) and `undefined` appears as `undefined`.

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| Example 6.8 > should go from A to "A.C" on 1 | TestExamples6_8_ShouldGoFromAToACOn1 | ported | string → MatchesState |
| Example 6.8 > should go from A to "F" on 6 | TestExamples6_8_ShouldGoFromAToFOn6 | ported | |
| Example 6.8 > should go from {"A":"B"} to "A.C" on 1 | TestExamples6_8_ShouldGoFromABToACOn1 | ported | |
| Example 6.8 > should go from {"A":"B"} to "F" on 6 | TestExamples6_8_ShouldGoFromABToFOn6 | ported | |
| Example 6.8 > should go from {"A":"B"} to undefined on FAKE | TestExamples6_8_ShouldGoFromABToUndefinedOnFAKE | ported | undefined → value unchanged |
| Example 6.8 > should go from {"A":"C"} to "A.E" on 2 | TestExamples6_8_ShouldGoFromACToAEOn2 | ported | |
| Example 6.8 > should go from {"A":"C"} to "F" on 6 | TestExamples6_8_ShouldGoFromACToFOn6 | ported | |
| Example 6.8 > should go from {"A":"C"} to undefined on FAKE | TestExamples6_8_ShouldGoFromACToUndefinedOnFAKE | ported | |
| Example 6.8 > should go from {"A":"D"} to "A.B" on 3 | TestExamples6_8_ShouldGoFromADToABOn3 | ported | |
| Example 6.8 > should go from {"A":"D"} to "F" on 6 | TestExamples6_8_ShouldGoFromADToFOn6 | ported | |
| Example 6.8 > should go from {"A":"D"} to undefined on FAKE | TestExamples6_8_ShouldGoFromADToUndefinedOnFAKE | ported | |
| Example 6.8 > should go from {"A":"E"} to "A.B" on 4 | TestExamples6_8_ShouldGoFromAEToABOn4 | ported | |
| Example 6.8 > should go from {"A":"E"} to "A.D" on 5 | TestExamples6_8_ShouldGoFromAEToADOn5 | ported | |
| Example 6.8 > should go from {"A":"E"} to "F" on 6 | TestExamples6_8_ShouldGoFromAEToFOn6 | ported | |
| Example 6.8 > should go from {"A":"E"} to undefined on FAKE | TestExamples6_8_ShouldGoFromAEToUndefinedOnFAKE | ported | |
| Example 6.8 > should go from F to "A.B" on 5 | TestExamples6_8_ShouldGoFromFToABOn5 | ported | history with no recorded value → A's initial B |
| Example 6.8 > should respect the history mechanism | TestExamples6_8_ShouldRespectTheHistoryMechanism | ported | |

## API gaps

None. Uses `StateMachine.ResolveState`, `xs.GetNextSnapshot`, `xs.MatchesState`, `xs.History` + `xs.Shallow`.

## Ambiguities for review

- `hist: { history: true }` (line 26): JS StateNode maps `history: true` to type `'history'` and
  `history: 'shallow'` (`StateNode.ts:192,227`). Go: `Type: xs.History, History: xs.Shallow`.
- `testAll` / `testMultiTransition` / `resolveSerializedStateValue` from `test/utils.ts` are copied
  as file-local helpers with prefix `examples681` (same shape as `xstate/spec_examples_test.go`).
- The JS machine is shared across the describe; Go builds a fresh machine per test (stateless config).
- Numeric event keys (`on: { 6: 'F' }`) become string event types `"6"`, matching JS where object keys
  and `{ type: '6' }` are strings.
