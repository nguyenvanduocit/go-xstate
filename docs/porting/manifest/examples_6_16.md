> Historical translation record. Chunk tags, API-gap filenames, line numbers,
> and run counts below describe the original porting work. See the current
> [core guide](../core.md) and [architecture](../../ARCHITECTURE.md) for the maintained layout.

## Source section: examples_6_16_1

Source: `references/xstate/packages/core/test/examples/6.16.test.ts` lines 1-54.
Go file: `xstate/spec_examples_test.go`.

The file has no literal `it()`/`test()` call: `testAll(machine, expected)` (line 52, defined in
`test/utils.ts`) generates one `it()` per (fromState, eventTypes) pair of `expected` (lines 34-50),
i.e. 3 + 3 + 3 = 9 tests. JS iterates `Object.keys`, which puts integer-like keys first in
ascending order; the rows below follow that order (it matches source order here).

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| Example 6.16 > should go from {"A":"D","B":"F"} to {"A":"C","B":"E"} on 1 | TestExamples6_16_ShouldGoFromADBFToACBEOn1 | ported | |
| Example 6.16 > should go from {"A":"D","B":"F"} to undefined on 2 | TestExamples6_16_ShouldGoFromADBFToUndefinedOn2 | ported | undefined → value unchanged |
| Example 6.16 > should go from {"A":"D","B":"F"} to {"A":"C","B":"F"} on 1, 5, 3 | TestExamples6_16_ShouldGoFromADBFToACBFOn1_5_3 | ported | multi-event chain |
| Example 6.16 > should go from {"A":"C","B":"E"} to undefined on 1 | TestExamples6_16_ShouldGoFromACBEToUndefinedOn1 | ported | |
| Example 6.16 > should go from {"A":"C","B":"E"} to {"A":"D","B":"E"} on 2 | TestExamples6_16_ShouldGoFromACBEToADBEOn2 | ported | stateIn('#E') guard |
| Example 6.16 > should go from {"A":"C","B":"E"} to {"A":"C","B":"G"} on 5 | TestExamples6_16_ShouldGoFromACBEToACBGOn5 | ported | |
| Example 6.16 > should go from {"A":"C","B":"G"} to undefined on 1 | TestExamples6_16_ShouldGoFromACBGToUndefinedOn1 | ported | |
| Example 6.16 > should go from {"A":"C","B":"G"} to undefined on 2 | TestExamples6_16_ShouldGoFromACBGToUndefinedOn2 | ported | guard stateIn('#E') false |
| Example 6.16 > should go from {"A":"C","B":"G"} to {"A":"C","B":"F"} on 3 | TestExamples6_16_ShouldGoFromACBGToACBFOn3 | ported | |

## API gaps

None. Uses `xs.StateIn`, `StateMachine.ResolveState`, `xs.GetNextSnapshot`, `xs.MatchesState`.

## Ambiguities for review

- `testAll` / `testMultiTransition` / `resolveSerializedStateValue` live in `test/utils.ts`; they
  are copied as file-local helpers with prefix `examples6161`. Other example chunks will likely need
  the same helpers; at integration they could be consolidated into `xstate/helpers_test.go`.
- The JS machine is shared across the describe; the Go port builds a fresh machine per test (the
  machine is stateless config, so behaviour is the same).
- `resolveState({ value, context: {} })`: context `{}` is passed as `map[string]any{}` with `C = any`.
- The `typeof toState === 'string'` branch of testAll is unused by this file's data but kept for
  fidelity.
