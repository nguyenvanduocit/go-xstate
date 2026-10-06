> Historical translation record. Chunk tags, API-gap filenames, line numbers,
> and run counts below describe the original porting work. See the current
> [core guide](../core.md) and [architecture](../../ARCHITECTURE.md) for the maintained layout.

## Source section: internal_transitions_1

Source: `references/xstate/packages/core/test/internalTransitions.test.ts` lines 1-371 (12 tests, single `describe('internal transitions')`, no `it.each`/`it.skip`).
Go file: `xstate/internal_transitions_test.go`.

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| internal transitions > parent state should enter child state without re-entering self | TestInternalTransitions_ParentStateShouldEnterChildStateWithoutReenteringSelf | ported | uses `internalTransitions1TrackEntries` |
| internal transitions > parent state should re-enter self upon transitioning to child state if transition is reentering | TestInternalTransitions_ParentStateShouldReenterSelfUponTransitioningToChildIfReentering | ported | |
| internal transitions > parent state should only exit/reenter if there is an explicit self-transition | TestInternalTransitions_ParentStateShouldOnlyExitReenterIfExplicitSelfTransition | ported | |
| internal transitions > parent state should only exit/reenter if there is an explicit self-transition (to child) | TestInternalTransitions_ParentStateShouldOnlyExitReenterIfExplicitSelfTransitionToChild | ported | |
| internal transitions > should listen to events declared at top state | TestInternalTransitions_ShouldListenToEventsDeclaredAtTopState | ported | |
| internal transitions > should work with targetless transitions (in conditional array) | TestInternalTransitions_ShouldWorkWithTargetlessTransitionsInConditionalArray | ported | `toHaveBeenCalled` -> `assert.Greater(Count, 0)` |
| internal transitions > should work with targetless transitions (in object) | TestInternalTransitions_ShouldWorkWithTargetlessTransitionsInObject | ported | object vs array form collapse to the same Go `Transitions` literal |
| internal transitions > should work on parent with targetless transitions (in conditional array) | TestInternalTransitions_ShouldWorkOnParentWithTargetlessTransitionsInConditionalArray | ported | |
| internal transitions > should work on parent with targetless transitions (in object) | TestInternalTransitions_ShouldWorkOnParentWithTargetlessTransitionsInObject | ported | same collapse as above |
| internal transitions > should maintain the child state when targetless transition is handled by parent | TestInternalTransitions_ShouldMaintainChildStateWhenTargetlessTransitionHandledByParent | ported | |
| internal transitions > should reenter proper descendants of a source state of an internal transition | TestInternalTransitions_ShouldReenterProperDescendantsOfSourceStateOfInternalTransition | ported | `types: {} as ...` dropped (type-only) |
| internal transitions > should exit proper descendants of a source state of an internal transition | TestInternalTransitions_ShouldExitProperDescendantsOfSourceStateOfInternalTransition | ported | `types: {} as ...` dropped (type-only) |

Totals: 12 JS tests, 12 ported, 0 N/A-type, 0 N/A-runtime, 0 skipped-in-JS.

## API gaps

None. No `apigap_internal_transitions_1.go` created.

## Reviewer notes

- `trackEntries` (test/utils.ts:76) is copied as file-local `internalTransitions1TrackEntries`, identical to `actions1TrackEntries` in `xstate/action_test.go`. It mutates `StateNode.Entry/Exit` after `CreateMachine`, same as JS `state.entry.unshift(...)`; the implementation must read actions from the `StateNode` fields at runtime for this to work.
- JS lines 166-219: the "in conditional array" (`[{actions}]`) and "in object" (`{actions}`) variants map to the same Go literal, because Go has no shorthand for a single transition object. The tests stay separate so every JS test has a Go counterpart.
