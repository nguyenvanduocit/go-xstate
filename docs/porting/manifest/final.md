> Historical translation record. Chunk tags, API-gap filenames, line numbers,
> and run counts below describe the original porting work. See the current
> [core guide](../core.md) and [architecture](../../ARCHITECTURE.md) for the maintained layout.

## Source section: final_1

Source: `references/xstate/packages/core/test/final.test.ts` lines 1-1294 (whole file, 1293 lines).
Go file: `xstate/final_test.go`.

Totals: 33 JS tests; 33 ported, 0 N/A-type, 0 N/A-runtime, 0 skipped-in-JS.

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| final states > status of a machine with a root state being final should be done | TestFinal_StatusOfMachineWithRootStateBeingFinalShouldBeDone | ported |  |
| final states > output of a machine with a root state being final should be called with a "xstate.done.state.ROOT_ID" event | TestFinal_OutputOfMachineWithRootStateBeingFinalCalledWithDoneStateRootEvent | ported | Inline snapshot `{output: undefined, type: 'xstate.done.state.(machine)'}` -> `xs.DoneStateEvent{StateID: "(machine)", Output: nil}`. |
| final states > should emit the "xstate.done.state.*" event when all nested states are in their final states | TestFinal_ShouldEmitDoneStateEventWhenAllNestedStatesAreInFinalStates | ported |  |
| final states > should execute final child state actions first | TestFinal_ShouldExecuteFinalChildStateActionsFirst | ported |  |
| final states > should call output expressions on nested final nodes | TestFinal_ShouldCallOutputExpressionsOnNestedFinalNodes | ported | `revealedSecret: undefined` -> zero-value string field; `Promise.withResolvers` -> `newSignal`. |
| final states > should only call data expression once when entering root's final state | TestFinal_ShouldOnlyCallDataExpressionOnceWhenEnteringRootsFinalState | ported |  |
| final states > output mapper should receive self | TestFinal_OutputMapperShouldReceiveSelf | ported | `expect(output.selfRef.send).toBeDefined()` -> output is a local struct with `SelfRef xs.ActorRef`, asserted non-nil (Send is part of the interface). |
| final states > state output should be able to use context updated by the entry action of the reached final state | TestFinal_StateOutputShouldUseContextUpdatedByEntryActionOfReachedFinalState | ported |  |
| final states > should emit a done state event for a parallel state when its parallel children reach their final states | TestFinal_ShouldEmitDoneStateEventForParallelWhenParallelChildrenReachFinal | ported |  |
| final states > should emit a done state event for a parallel state when its compound child reaches its final state when the other parallel child region is already in its final state | TestFinal_DoneStateForParallelWhenCompoundChildFinalAfterParallelRegionFinal | ported |  |
| final states > should emit a done state event for a parallel state when its parallel child reaches its final state when the other compound child region is already in its final state | TestFinal_DoneStateForParallelWhenParallelChildFinalAfterCompoundRegionFinal | ported |  |
| final states > should reach a final state when a parallel state reaches its final state and transitions to a top-level final state in response to that | TestFinal_ShouldReachFinalWhenParallelReachesFinalAndTransitionsToTopLevelFinal | ported | Machine is identical to the next test in JS; shared via `final1ParallelReachesFinalMachine`. |
| final states > should reach a final state when a parallel state nested in a parallel state reaches its final state and transitions to a top-level final state in response to that | TestFinal_ShouldReachFinalWhenNestedParallelReachesFinalAndTransitionsToTopLevelFinal | ported | Machine identical to the previous test in JS (JS duplicate). |
| final states > root output should be called with a "xstate.done.state.*" event of the parallel root when a direct final child of that parallel root is reached | TestFinal_RootOutputCalledWithParallelRootDoneEventWhenDirectFinalChildReached | ported | Inline snapshot -> `xs.DoneStateEvent{StateID: "(machine)"}`. |
| final states > root output should be called with a "xstate.done.state.*" event of the parallel root when a final child of its compound child is reached | TestFinal_RootOutputCalledWithParallelRootDoneEventWhenCompoundChildFinalReached | ported | Inline snapshot -> `xs.DoneStateEvent{StateID: "(machine)"}`. |
| final states > root output should be called with a "xstate.done.state.*" event of the parallel root when a final descendant is reached 2 parallel levels deep | TestFinal_RootOutputCalledWithParallelRootDoneEventWhenFinalDescendant2LevelsDeep | ported | Inline snapshot -> `xs.DoneStateEvent{StateID: "(machine)"}`. |
| final states > onDone of an outer parallel state should be called with its own "xstate.done.state.*" event when its direct parallel child completes | TestFinal_OnDoneOfOuterParallelCalledWithOwnDoneEventWhenDirectParallelChildCompletes | ported | Inline snapshot -> `xs.DoneStateEvent{StateID: "(machine).a"}`. |
| final states > onDone should not be called when the machine reaches its final state | TestFinal_OnDoneShouldNotBeCalledWhenMachineReachesItsFinalState | ported |  |
| final states > machine should not complete when a parallel child of a compound state completes | TestFinal_MachineShouldNotCompleteWhenParallelChildOfCompoundStateCompletes | ported | JS declares an unused `vi.fn()`; omitted (no assertion uses it). |
| final states > root output should only be called once when multiple parallel regions complete at once | TestFinal_RootOutputOnlyCalledOnceWhenMultipleParallelRegionsCompleteAtOnce | ported |  |
| final states > onDone of a parallel state should only be called once when multiple parallel regions complete at once | TestFinal_OnDoneOfParallelOnlyCalledOnceWhenMultipleRegionsCompleteAtOnce | ported |  |
| final states > should call exit actions in reversed document order when the machines reaches its final state | TestFinal_ShouldCallExitActionsInReversedDocOrderWhenMachineReachesFinalState | ported | `trackEntries` copied as `final1TrackEntries`. |
| final states > should call exit actions of parallel states in reversed document order when the machines reaches its final state after earlier region transition | TestFinal_ExitActionsOfParallelInReversedDocOrderAfterEarlierRegionTransition | ported | `trackEntries` copied as `final1TrackEntries`; machine via `final1TwoRegionMachine("EV2","EV1")`. |
| final states > should call exit actions of parallel states in reversed document order when the machines reaches its final state after later region transition | TestFinal_ExitActionsOfParallelInReversedDocOrderAfterLaterRegionTransition | ported | Identical to the 'earlier region' test in JS (same machine, same events); kept as-is. |
| final states > should call exit actions of parallel states in reversed document order when the machines reaches its final state after multiple regions transition | TestFinal_ExitActionsOfParallelInReversedDocOrderAfterMultipleRegionsTransition | ported | `final1TwoRegionMachine("EV","EV")`. |
| final states > should not complete a parallel root immediately when only some of its regions are in their final states (final state reached in a compound region) | TestFinal_ShouldNotCompleteParallelRootWhenOnlySomeRegionsFinalCompoundRegion | ported |  |
| final states > should not complete a parallel root immediately when only some of its regions are in their final states (a direct final child state reached) | TestFinal_ShouldNotCompleteParallelRootWhenOnlySomeRegionsFinalDirectFinalChild | ported |  |
| final states > should not resolve output of a final state if its parent is a parallel state | TestFinal_ShouldNotResolveOutputOfFinalStateIfParentIsParallel | ported |  |
| final states > should only call exit actions once when a child machine reaches its final state and sends an event to its parent that ends up stopping that child | TestFinal_OnlyCallExitActionsOnceWhenChildReachesFinalAndSendsEventStoppingIt | ported | Inline child logic -> `InvokeConfig.Logic`. |
| final states > should deliver final outgoing events (from final entry action) to the parent before delivering the `xstate.done.actor.*` event | TestFinal_DeliverFinalOutgoingEventsFromFinalEntryBeforeDoneActorEvent | ported |  |
| final states > should deliver final outgoing events (from root exit action) to the parent before delivering the `xstate.done.actor.*` event | TestFinal_DeliverFinalOutgoingEventsFromRootExitBeforeDoneActorEvent | ported |  |
| final states > should be possible to complete with a null output (directly on root) | TestFinal_ShouldBePossibleToCompleteWithNullOutputDirectlyOnRoot | ported | JS `output: null` vs absent output both map to Go `nil`; `toBe(null)` -> `assert.Nil`. Added `Status == done` assertion so the test is not vacuous. |
| final states > should be possible to complete with a null output (resolving with final state's output) | TestFinal_ShouldBePossibleToCompleteWithNullOutputResolvingFinalStateOutput | ported | Same null/undefined collapse as above; added `Status == done` assertion. |

## API gaps

None. No `apigap_final_1.go` was created.

## Ambiguities for review

- JS lines 1249-1293 (null-output tests): Go cannot distinguish `output: null` from no output (`Output any` is nil in both cases). The `toBe(null)` assertion is kept as `assert.Nil`; an extra `Status == done` assertion was added so the test still checks that completion with a nil output happens.
- JS lines 27, 599, 632, 670, 714 (inline snapshots of done-state events): translated assuming the engine passes `xs.DoneStateEvent` by value to output/onDone expressions (contract `event.go` defines value-receiver `EventType`).
- JS lines 161, 256, 1285 (`event.output`): read via `a.Event.(xs.DoneStateEvent).Output`.
- JS tests at lines 520 and 552 are identical tests in JS; both ported with one shared machine builder. Same for lines 869 vs 925 (identical machine and event sequence).
- `final1TrackEntries` (copied from `test/utils.ts:76`) mutates `StateNode.Entry/Exit` after machine creation, same as the existing `deep1TrackEntries`/`actions1TrackEntries`; the JS `seen` guard (throw on reuse) is not replicated.
