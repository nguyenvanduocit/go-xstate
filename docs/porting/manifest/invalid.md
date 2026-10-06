> Historical translation record. Chunk tags, API-gap filenames, line numbers,
> and run counts below describe the original porting work. See the current
> [core guide](../core.md) and [architecture](../../ARCHITECTURE.md) for the maintained layout.

## Source section: invalid_1

Source: `references/xstate/packages/core/test/invalid.test.ts` lines 1-173 (whole file, 6 `it` calls, no `it.each`/`it.skip`/`it.todo`).
Go file: `xstate/invalid_test.go`.

The parallel machine declared identically in each of the 5 tests of `invalid or resolved states` (JS L5-23, L36-54, ...) is the
file-local constructor `invalid1ParallelMachine()`, called fresh in each test.

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| invalid or resolved states > should resolve a String state | TestInvalid_ResolvedStates_ShouldResolveAStringState | ported | JS L4; `resolveState({value: 'A'})` on a parallel root |
| invalid or resolved states > should resolve transitions from empty states | TestInvalid_ResolvedStates_ShouldResolveTransitionsFromEmptyStates | ported | JS L35; `{A: {}, B: {}}` -> nested empty `map[string]any{}` |
| invalid or resolved states > should allow transitioning from valid states | TestInvalid_ResolvedStates_ShouldAllowTransitioningFromValidStates | ported | JS L66; JS has no explicit expect (passes when nothing throws) -> `assert.NotPanics` |
| invalid or resolved states > should reject transitioning from bad state configs | TestInvalid_ResolvedStates_ShouldRejectTransitioningFromBadStateConfigs | ported | JS L92; `toThrow()` -> `assert.Panics`; ResolveState is inside the panic closure, as in JS |
| invalid or resolved states > should resolve transitioning from partially valid states | TestInvalid_ResolvedStates_ShouldResolveTransitioningFromPartiallyValidStates | ported | JS L120 |
| invalid transition > should throw when attempting to create a machine with a sibling target on the root node | TestInvalid_Transition_ShouldThrowWhenCreatingMachineWithSiblingTargetOnRootNode | ported | JS L154; `toThrowError(/invalid target/i)` -> `assert.Panics` + case-insensitive regexp on `panicMessage` |

Counts: 6 JS tests; 6 ported, 0 N/A-type, 0 N/A-runtime, 0 skipped-in-JS.

## API gaps

None. Uses existing contract APIs: `xs.GetNextSnapshot` (util.go:73), `StateMachine.ResolveState` (machine.go:135).

## Ambiguities for review

- JS L66-90: test body has no `expect`; Go wraps the call in `assert.NotPanics`, which is the implicit JS assertion.
- JS L154-172: Go test name shortened ("attempting to create" -> "creating") to stay under ~100 chars.
- JS L154-172: machine creation runs twice in Go (once for `assert.Panics`, once for `panicMessage`); CreateMachine is side-effect free so this does not change semantics.
