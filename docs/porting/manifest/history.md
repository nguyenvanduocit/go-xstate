> Historical translation record. Chunk tags, API-gap filenames, line numbers,
> and run counts below describe the original porting work. See the current
> [core guide](../core.md) and [architecture](../../ARCHITECTURE.md) for the maintained layout.

# history_1 — packages/core/test/history.test.ts L1-1517

Test file: `xstate/history_test.go` (tag `port_history_1`). Gap file: `apigap_history_1.go`.
JS tests in range: 35 (`it(` count; no it.each / it.skip / it.todo). Ported: 35.

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| history states > should go to the most recently visited state (explicit shallow history type) | TestHistory_ShouldGoToMostRecentlyVisitedStateExplicitShallowHistoryType | ported | L6 |
| history states > should go to the most recently visited state (no explicit history type) | TestHistory_ShouldGoToMostRecentlyVisitedStateNoExplicitHistoryType | ported | L41 |
| history states > should go to the initial state when no history present (explicit shallow history type) | TestHistory_ShouldGoToInitialStateWhenNoHistoryPresentExplicitShallowHistoryType | ported | L74 |
| history states > should go to the initial state when no history present (no explicit history type) | TestHistory_ShouldGoToInitialStateWhenNoHistoryPresentNoExplicitHistoryType | ported | L101 |
| history states > should go to the most recently visited state by a transient transition | TestHistory_ShouldGoToMostRecentlyVisitedStateByTransientTransition | ported | L127 |
| history states > should reenter persisted state during reentering transition targeting a history state | TestHistory_ShouldReenterPersistedStateDuringReenteringTransitionTargetingHistoryState | ported | L176; `actual.length = 0` -> slice reset under mutex |
| history states > should go to the configured default target when a history state is the initial state of the machine | TestHistory_ShouldGoToDefaultTargetWhenHistoryStateIsInitialStateOfMachine | ported | L219 |
| history states > should go to the configured default target when a history state is the initial state of the transition's target | TestHistory_ShouldGoToDefaultTargetWhenHistoryStateIsInitialStateOfTransitionTarget | ported | L236 |
| history states > should execute actions of the initial transition when a history state without a default target is targeted and its parent state was never visited yet | TestHistory_ShouldExecuteInitialActionsWhenHistoryWithoutDefaultTargetTargetedAndParentNeverVisited | ported | L267; `initial: {target, actions}` -> gap `WithStateInitialActions` |
| history states > should enter the parallel default configuration when a deep history state without a default target is targeted and its parent parallel state was never visited yet | TestHistory_ShouldEnterParallelDefaultConfigWhenDeepHistoryWithoutDefaultTargetTargetedAndParentNeverVisited | ported | L298 |
| history states > should enter the parallel default configuration when a shallow history state without a default target is targeted and its parent parallel state was never visited yet | TestHistory_ShouldEnterParallelDefaultConfigWhenShallowHistoryWithoutDefaultTargetTargetedAndParentNeverVisited | ported | L324 |
| history states > should not execute actions of the initial transition when a history state with a default target is targeted and its parent state was never visited yet | TestHistory_ShouldNotExecuteInitialActionsWhenHistoryWithDefaultTargetTargetedAndParentNeverVisited | ported | L350; gap `WithStateInitialActions`; `not.toHaveBeenCalled` -> Count()==0 |
| history states > should execute entry actions of a parent of the targeted history state when its parent state was never visited yet | TestHistory_ShouldExecuteParentEntryActionsOfTargetedHistoryStateWhenParentNeverVisited | ported | L382 |
| history states > should execute actions of the initial transition when it select a history state as the initial state of its parent | TestHistory_ShouldExecuteInitialActionsWhenItSelectsHistoryStateAsInitialStateOfParent | ported | L412; gap `WithStateInitialActions` |
| history states > should execute actions of the initial transition when a history state without a default target is targeted and its parent state was already visited | TestHistory_ShouldExecuteInitialActionsWhenHistoryWithoutDefaultTargetTargetedAndParentAlreadyVisited | ported | L443; `spy.mockClear()` -> baseline count; JS asserts 0 calls despite the "should execute" title — kept as 0 |
| history states > should not execute actions of the initial transition when a history state with a default target is targeted and its parent state was already visited | TestHistory_ShouldNotExecuteInitialActionsWhenHistoryWithDefaultTargetTargetedAndParentAlreadyVisited | ported | L481; mockClear -> baseline count |
| history states > should execute entry actions of a parent of the targeted history state when its parent state was already visited | TestHistory_ShouldExecuteParentEntryActionsOfTargetedHistoryStateWhenParentAlreadyVisited | ported | L520; mockClear -> baseline count |
| history states > should invoke an actor when reentering the stored configuration through the history state | TestHistory_ShouldInvokeActorWhenReenteringStoredConfigurationThroughHistoryState | ported | L557; `fromCallback(spy)` -> FromCallback whose body calls the spy; mockClear -> baseline count |
| history states > should not enter ancestors of the entered history state that lie outside of the transition domain when entering the default history configuration | TestHistory_ShouldNotEnterAncestorsOutsideTransitionDomainWhenEnteringDefaultHistoryConfiguration | ported | L586; `trackEntries` (utils.ts:76) copied as `history1TrackEntries` |
| history states > should not enter ancestors of the entered history state that lie outside of the transition domain when restoring the stored history configuration | TestHistory_ShouldNotEnterAncestorsOutsideTransitionDomainWhenRestoringStoredHistoryConfiguration | ported | L622 |
| deep history states > should go to the shallow history | TestHistory_Deep_ShouldGoToShallowHistory | ported | L674; machine built by `history1DeepMachine(xs.Shallow, false)` (configs of the 3 deep tests differ only in history type and P's INNER) |
| deep history states > should go to the deep history (explicit) | TestHistory_Deep_ShouldGoToDeepHistoryExplicit | ported | L726 |
| deep history states > should go to the deepest history | TestHistory_Deep_ShouldGoToDeepestHistory | ported | L780 |
| parallel history states > should ignore parallel state history | TestHistory_Parallel_ShouldIgnoreParallelStateHistory | ported | L839 |
| parallel history states > should remember first level state history | TestHistory_Parallel_ShouldRememberFirstLevelStateHistory | ported | L905 |
| parallel history states > should re-enter each regions of parallel state correctly | TestHistory_Parallel_ShouldReenterEachRegionsOfParallelStateCorrectly | ported | L976; `on` state from `history1ParallelOn()` (identical config in L976/L1062/L1148/L1234 tests) |
| parallel history states > should re-enter multiple history states | TestHistory_Parallel_ShouldReenterMultipleHistoryStates | ported | L1065; `target: [..]` -> Targets |
| parallel history states > should re-enter a parallel with partial history | TestHistory_Parallel_ShouldReenterParallelWithPartialHistory | ported | L1155 |
| parallel history states > should re-enter a parallel with full history | TestHistory_Parallel_ShouldReenterParallelWithFullHistory | ported | L1245 |
| internal transition to a history state should enter default history state configuration if the containing state has never been exited yet | TestHistory_InternalTransitionToHistoryStateShouldEnterDefaultConfigIfContainingStateNeverExited | ported | L1338 (top-level it) |
| multistage history states > should go to the most recently visited state | TestHistory_Multistage_ShouldGoToMostRecentlyVisitedState | ported | L1376 |
| revive history states > should restore from stringified snapshot | TestHistory_Revive_ShouldRestoreFromStringifiedSnapshot | ported | L1456; describe-level setup -> `history1ReviveSetup`; JSON.parse(JSON.stringify(getPersistedSnapshot())) -> json round-trip into map[string]any |
| revive history states > should ignore unresolved ids as-is and log a warning | TestHistory_Revive_ShouldIgnoreUnresolvedIdsAsIsAndLogWarning | ported | L1467; `vi.spyOn(console,'warn')` -> gap `WithWarnHandler`; `toHaveBeenCalledWith(msg)` -> `assert.Contains(calls, []any{msg})`; `(getPersistedSnapshot() as any).historyValue` read via json round-trip |
| revive history states > should not re-resolve already-instantiated StateNode | TestHistory_Revive_ShouldNotReResolveAlreadyInstantiatedStateNode | ported | L1488; `toBeInstanceOf(StateNode)` -> non-empty + `IsType(&xs.StateNode{})` + NotNil (HistoryValue is statically `[]*StateNode`) |
| revive history states > should handle null, undefined, and primitive values | TestHistory_Revive_ShouldHandleNullUndefinedAndPrimitiveValues | ported | L1502; forEach -> t.Run subtests; `undefined` modelled as key absent |

## API gaps (`apigap_history_1.go`)

- `WithStateInitialActions(cfg StateConfig, actions ...Action) StateConfig` — JS `initial: { target, actions }` on a state node. Same signature as `apigap_actions_2.go`.
- `WithWarnHandler(fn func(args ...any)) ActorOption` — `vi.spyOn(console, 'warn')`. Same signature as `apigap_actions_3.go` / `apigap_actions_4.go` / `apigap_event_descriptors_1.go`.

## Ambiguities for the reviewer

- JS `history: 'shallow' | 'deep' | true` without `type: 'history'` (L704, L756, L812, L865ff, L1354, L1390): StateNode.ts:188-194 infers type `history`; `true` means shallow (StateNode.ts:227-228). Translated as `{Key, History: xs.Shallow|xs.Deep}` with `Type` left empty, so the Go implementation must infer `Type == History` when `History != ""`.
- L443 test title says "should execute" but asserts `toHaveBeenCalledTimes(0)`; translated as asserted (0).
- Revive describe (L1419-1454) runs its setup once at describe level; the Go tests each run the same deterministic setup via `history1ReviveSetup`.
- L1502 `undefined` element: a spread with `historyValue: undefined` is modelled as the key being absent from the persisted map; `null` as an explicit `nil` value.
- `getPersistedSnapshot()` returns `any` in the contract; tests read its `historyValue` through a JSON round-trip (expects `{}`, i.e. `map[string]any{}`, not `null`).
