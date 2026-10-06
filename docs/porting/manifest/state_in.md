> Historical translation record. Chunk tags, API-gap filenames, line numbers,
> and run counts below describe the original porting work. See the current
> [core guide](../core.md) and [architecture](../../ARCHITECTURE.md) for the maintained layout.

## Source section: state_in_1

Source: `references/xstate/packages/core/test/stateIn.test.ts` lines 1-503 (9 JS tests; the file has 502 lines, so this chunk is the whole file).
Go file: `xstate/state_in_test.go`.

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| transition "in" check > should transition if string state path matches current state value | TestStateIn_ShouldTransitionIfStringStatePathMatchesCurrentStateValue | ported | `stateIn({ b: 'b2' })` -> `xs.StateIn(map[string]any{"b": "b2"})` |
| transition "in" check > should transition if state node ID matches current state value | TestStateIn_ShouldTransitionIfStateNodeIDMatchesCurrentStateValue | ported | |
| transition "in" check > should not transition if string state path does not match current state value | TestStateIn_ShouldNotTransitionIfStringStatePathDoesNotMatchCurrentStateValue | ported | |
| transition "in" check > should not transition if state value matches current state value | TestStateIn_ShouldNotTransitionIfStateValueMatchesCurrentStateValue | ported | JS name says "not transition" but the assertion expects a transition to `a2`; ported verbatim |
| transition "in" check > matching should be relative to grandparent (match) | TestStateIn_MatchingShouldBeRelativeToGrandparentMatch | ported | |
| transition "in" check > matching should be relative to grandparent (no match) | TestStateIn_MatchingShouldBeRelativeToGrandparentNoMatch | ported | |
| transition "in" check > should work to forbid events | TestStateIn_ShouldWorkToForbidEvents | ported | |
| transition "in" check > should be possible to use a referenced `stateIn` guard | TestStateIn_ShouldBePossibleToUseAReferencedStateInGuard | ported | `selected: {}` state value -> `map[string]any{}` |
| transition "in" check > should be possible to check an ID with a path | TestStateIn_ShouldBePossibleToCheckAnIDWithAPath | ported | `vi.fn()` action -> `xs.ActionFunc` calling `newSpy().Call` |

## API gaps

None. No `apigap_state_in_1.go` was created.

## Ambiguities for review

- JS line 199 (`should not transition if state value matches current state value`): the test name contradicts its assertion (JS lines 252-260 expect `a: 'a2'`). The Go port keeps the JS assertion.
- JS line 463 (`selected: {}`): the Go port expects an empty atomic-parallel-region value of `map[string]any{}`, matching JS `toEqual({ selected: {}, ... })`.
