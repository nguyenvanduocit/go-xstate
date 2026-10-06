> Historical translation record. Chunk tags, API-gap filenames, line numbers,
> and run counts below describe the original porting work. See the current
> [core guide](../core.md) and [architecture](../../ARCHITECTURE.md) for the maintained layout.

## Source section: parallel_1

Source: `references/xstate/packages/core/test/parallel.test.ts` lines 1-1346 (whole file; the chunk range 1-1347 covers it).
Go file: `xstate/parallel_test.go`.

The file has 24 literal `it()` calls plus one template-literal `it()` (line 548) inside a nested
`Object.keys(expected).forEach` loop over `expected` (lines 506-542). The loop generates
1 + 1 + 2 = 4 tests (no integer-like keys, so `Object.keys` order is source order).
Total: 28 JS tests. No `it.skip`/`it.todo`, no type-level-only tests.

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| parallel states > should have initial parallel states | TestParallel_ShouldHaveInitialParallelStates | ported | |
| parallel states > should go from {"bold": "off"} to {"bold":"on","italics":"off","underline":"off","list":"none"} on TOGGLE_BOLD | TestParallel_ShouldGoFromBoldOffToBoldOnOnTOGGLE_BOLD | ported | loop-generated (line 548); uses parallel1TestMultiTransition |
| parallel states > should go from {"bold": "on"} to {"bold":"off","italics":"off","underline":"off","list":"none"} on TOGGLE_BOLD | TestParallel_ShouldGoFromBoldOnToBoldOffOnTOGGLE_BOLD | ported | loop-generated |
| parallel states > should go from {"bold":"off","italics":"off","underline":"on","list":"bullets"} to {"bold":"on","italics":"on","underline":"on","list":"bullets"} on TOGGLE_BOLD, TOGGLE_ITALICS | TestParallel_ShouldGoFromUnderlineOnBulletsToBoldItalicsOnOnTOGGLE_BOLD_TOGGLE_ITALICS | ported | loop-generated; multi-event chain |
| parallel states > should go from {"bold":"off","italics":"off","underline":"on","list":"bullets"} to {"bold":"off","italics":"off","underline":"off","list":"none"} on RESET | TestParallel_ShouldGoFromUnderlineOnBulletsToAllOffOnRESET | ported | loop-generated |
| parallel states > should have all parallel states represented in the state value | TestParallel_ShouldHaveAllParallelStatesRepresentedInTheStateValue | ported | |
| parallel states > should have all parallel states represented in the state value (2) | TestParallel_ShouldHaveAllParallelStatesRepresentedInTheStateValue2 | ported | |
| parallel states > should work with regions without states (line 603) | TestParallel_ShouldWorkWithRegionsWithoutStates | ported | duplicate JS name; first occurrence (initial snapshot) |
| parallel states > should work with regions without states (line 611) | TestParallel_ShouldWorkWithRegionsWithoutStates2 | ported | duplicate JS name; second occurrence (after E) |
| parallel states > should properly transition to relative substate | TestParallel_ShouldProperlyTransitionToRelativeSubstate | ported | |
| parallel states > should properly transition according to entry events on an initial state | TestParallel_ShouldProperlyTransitionAccordingToEntryEventsOnAnInitialState | ported | |
| parallel states > should properly transition when raising events for a parallel state | TestParallel_ShouldProperlyTransitionWhenRaisingEventsForAParallelState | ported | |
| parallel states > should handle simultaneous orthogonal transitions | TestParallel_ShouldHandleSimultaneousOrthogonalTransitions | ported | `types: {}` dropped (type-only); runtime asserts kept |
| parallel states > should execute actions of the initial transition of a parallel region when entering the initial state nodes of a machine | TestParallel_ShouldExecuteInitialTransitionActionsOfParallelRegionWhenEnteringInitialStateNodesOfMachine | ported | `initial: {target, actions}` via gap WithStateInitialActions |
| parallel states > should execute actions of the initial transition of a parallel region when the parallel state is targeted with an explicit transition | TestParallel_ShouldExecuteInitialTransitionActionsOfParallelRegionWhenParallelStateIsTargetedExplicitly | ported | same gap |
| parallel states > transitions with nested parallel states > should properly transition when in a simple nested state | TestParallel_TransitionsWithNestedParallelStates_ShouldProperlyTransitionWhenInASimpleNestedState | ported | |
| parallel states > transitions with nested parallel states > should properly transition when in a complex nested state | TestParallel_TransitionsWithNestedParallelStates_ShouldProperlyTransitionWhenInAComplexNestedState | ported | |
| parallel states > nested flat parallel states > should represent the flat nested parallel states in the state value | TestParallel_NestedFlatParallelStates_ShouldRepresentTheFlatNestedParallelStatesInTheStateValue | ported | describe-level machine moved into the test |
| parallel states > deep flat parallel states > should properly evaluate deep flat parallel states | TestParallel_DeepFlatParallelStates_ShouldProperlyEvaluateDeepFlatParallelStates | ported | |
| parallel states > deep flat parallel states > should not overlap resolved state nodes in state resolution | TestParallel_DeepFlatParallelStates_ShouldNotOverlapResolvedStateNodesInStateResolution | ported | `not.toThrow` -> assert.NotPanics |
| parallel states > other > regions should be able to transition to orthogonal regions | TestParallel_Other_RegionsShouldBeAbleToTransitionToOrthogonalRegions | ported | |
| parallel states > other > should calculate the entry set for reentering transitions in parallel states | TestParallel_Other_ShouldCalculateTheEntrySetForReenteringTransitionsInParallelStates | ported | |
| parallel states > should raise a "xstate.done.state.*" event when all child states reach final state | TestParallel_ShouldRaiseDoneStateEventWhenAllChildStatesReachFinalState | ported | returned promise -> signal.Wait |
| parallel states > should raise a "xstate.done.state.*" event when a pseudostate of a history type is directly on a parallel state | TestParallel_ShouldRaiseDoneStateEventWhenHistoryPseudostateIsDirectlyOnParallelState | ported | |
| parallel states > source parallel region should be reentered when a transition within it targets another parallel region (parallel root) | TestParallel_SourceParallelRegionShouldBeReenteredWhenTargetingAnotherParallelRegion_ParallelRoot | ported | uses parallel1TrackEntries |
| parallel states > source parallel region should be reentered when a transition within it targets another parallel region (nested parallel) | TestParallel_SourceParallelRegionShouldBeReenteredWhenTargetingAnotherParallelRegion_NestedParallel | ported | uses parallel1TrackEntries |
| parallel states > targetless transition on a parallel state should not enter nor exit any states | TestParallel_TargetlessTransitionOnAParallelStateShouldNotEnterNorExitAnyStates | ported | |
| parallel states > targetless transition in one of the parallel regions should not enter nor exit any states | TestParallel_TargetlessTransitionInOneOfTheParallelRegionsShouldNotEnterNorExitAnyStates | ported | |

Counts: 28 JS tests; 28 ported; 0 N/A-type; 0 N/A-runtime; 0 skipped-in-JS.

## API gaps (`apigap_parallel_1.go`)

- `WithStateInitialActions(cfg StateConfig, actions ...Action) StateConfig` — JS `initial: { target, actions }`
  on a state node (lines 765-768, 796-799). Same signature as the existing gap in `apigap_actions_2.go` /
  `apigap_history_1.go`; maps to an `InitialActions` field on StateConfig at integration.

## Helpers copied from `test/utils.ts` (area prefix `parallel1`)

- `parallel1ResolveSerializedStateValue`, `parallel1TestMultiTransition` (testMultiTransition), `parallel1TrackEntries`
  (trackEntries). Same shape as `id1TestMultiTransition` / `deep1TrackEntries` in other chunks.
- Module-level JS machines (`composerMachine`, `wakMachine`, `wordMachine`, `flatParallelMachine`,
  `raisingParallelMachine`, `nestedParallelState`, `deepFlatParallelMachine`) became constructor funcs
  (`parallel1WordMachine()`, ...) so each test gets a fresh machine. JS shares the instance; machines are
  immutable configs there, so no behaviour depends on sharing.

## Reviewer notes

- Lines 603/611: two JS tests share the exact name "should work with regions without states"; Go suffixes the second with `2`.
- Line 506-507/515: the fromState keys `{"bold": "off"}` / `{"bold": "on"}` are partial state values parsed with
  JSON and resolved via `machine.ResolveState` (utils.ts resolveSerializedStateValue); the Go helper does the same.
- composerMachine: `SelectionStatus` region is identical in both parallel parents (JS lines 25-68 vs 140-183) and is
  built by one helper `parallel1SelectionStatus()`. The `ProjectManagement` parent has only that region.
- Named actions (`selectNone`, `redraw`, `wak1enter`, `save`, ...) are never provided in JS; translated as
  `xs.ActionRef{Type: ...}` without implementations, matching JS (unresolved named actions are skipped).
- Line 1180/1229 tests are `async` in JS but await nothing; translated synchronously.
- Line 1312/1344: `toEqual([])` -> `assert.Equal(t, []string{}, ...)`; the tracker's flush returns a non-nil empty slice.
