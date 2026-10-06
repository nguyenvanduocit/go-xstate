> Historical translation record. Chunk tags, API-gap filenames, line numbers,
> and run counts below describe the original porting work. See the current
> [core guide](../core.md) and [architecture](../../ARCHITECTURE.md) for the maintained layout.

## Source section: actions_1

Source: `references/xstate/packages/core/test/actions.test.ts` lines 1-1233.
Go file: `xstate/action_test.go`.

Totals: 38 JS tests; 38 ported, 0 N/A-type, 0 N/A-runtime, 0 skipped-in-JS.

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| entry/exit actions > State.actions > should return the entry actions of an initial state | TestActions_StateActions_ShouldReturnEntryActionsOfInitialState | ported |  |
| entry/exit actions > State.actions > should return the entry actions of an initial state (deep) | TestActions_StateActions_ShouldReturnEntryActionsOfInitialStateDeep | ported |  |
| entry/exit actions > State.actions > should return the entry actions of an initial state (parallel) | TestActions_StateActions_ShouldReturnEntryActionsOfInitialStateParallel | ported |  |
| entry/exit actions > State.actions > should return the entry and exit actions of a transition | TestActions_StateActions_ShouldReturnEntryAndExitActionsOfTransition | ported |  |
| entry/exit actions > State.actions > should return the entry and exit actions of a deep transition | TestActions_StateActions_ShouldReturnEntryAndExitActionsOfDeepTransition | ported |  |
| entry/exit actions > State.actions > should return the entry and exit actions of a nested transition | TestActions_StateActions_ShouldReturnEntryAndExitActionsOfNestedTransition | ported |  |
| entry/exit actions > State.actions > should not have actions for unhandled events (shallow) | TestActions_StateActions_ShouldNotHaveActionsForUnhandledEventsShallow | ported |  |
| entry/exit actions > State.actions > should not have actions for unhandled events (deep) | TestActions_StateActions_ShouldNotHaveActionsForUnhandledEventsDeep | ported |  |
| entry/exit actions > State.actions > should exit and enter the state for reentering self-transitions (shallow) | TestActions_StateActions_ShouldExitAndEnterStateForReenteringSelfTransitionsShallow | ported |  |
| entry/exit actions > State.actions > should exit and enter the state for reentering self-transitions (deep) | TestActions_StateActions_ShouldExitAndEnterStateForReenteringSelfTransitionsDeep | ported |  |
| entry/exit actions > State.actions > should return actions for parallel machines | TestActions_StateActions_ShouldReturnActionsForParallelMachines | ported | `actual.length = 0` -> `actual = actual[:0]`. |
| entry/exit actions > State.actions > should return nested actions in the correct (child to parent) order | TestActions_StateActions_ShouldReturnNestedActionsInChildToParentOrder | ported |  |
| entry/exit actions > State.actions > should ignore parent state actions for same-parent substates | TestActions_StateActions_ShouldIgnoreParentStateActionsForSameParentSubstates | ported |  |
| entry/exit actions > State.actions > should work with function actions | TestActions_StateActions_ShouldWorkWithFunctionActions | ported | `toHaveBeenCalled` -> `assert.Positive(spy.Count())`. |
| entry/exit actions > State.actions > should exit children of parallel state nodes | TestActions_StateActions_ShouldExitChildrenOfParallelStateNodes | ported |  |
| entry/exit actions > State.actions > should reenter targeted ancestor (as it's a descendant of the transition domain) | TestActions_StateActions_ShouldReenterTargetedAncestor | ported |  |
| entry/exit actions > State.actions > shouldn't use a referenced custom action over a builtin one when there is a naming conflict | TestActions_StateActions_ShouldNotUseReferencedCustomActionOverBuiltinOne | ported |  |
| entry/exit actions > State.actions > shouldn't use a referenced custom action over an inline one when there is a naming conflict | TestActions_StateActions_ShouldNotUseReferencedCustomActionOverInlineOne | ported | JS named function `myFn` (L591) bound to Go var `myFn`; Go funcs carry no name, so the test checks the inline action runs and the `myFn` implementation does not. |
| entry/exit actions > State.actions > root entry/exit actions should be called on root reentering transitions | TestActions_StateActions_RootEntryExitActionsCalledOnRootReenteringTransitions | ported | `mockClear()` (L636-637) mapped to call-count deltas (`assert.Greater` vs count before send). |
| entry/exit actions > State.actions > should ignore same-parent state actions (sparse) > with a relative transition | TestActions_StateActions_IgnoreSameParentSparse_WithRelativeTransition | ported |  |
| entry/exit actions > State.actions > should ignore same-parent state actions (sparse) > with an absolute transition | TestActions_StateActions_IgnoreSameParentSparse_WithAbsoluteTransition | ported |  |
| entry/exit actions > entry/exit actions > should return the entry actions of an initial state | TestActions_EntryExitActions_ShouldReturnEntryActionsOfInitialState | ported |  |
| entry/exit actions > entry/exit actions > should return the entry and exit actions of a transition | TestActions_EntryExitActions_ShouldReturnEntryAndExitActionsOfTransition | ported |  |
| entry/exit actions > entry/exit actions > should return the entry and exit actions of a deep transition | TestActions_EntryExitActions_ShouldReturnEntryAndExitActionsOfDeepTransition | ported |  |
| entry/exit actions > entry/exit actions > should return the entry and exit actions of a nested transition | TestActions_EntryExitActions_ShouldReturnEntryAndExitActionsOfNestedTransition | ported |  |
| entry/exit actions > entry/exit actions > should keep the same state for unhandled events (shallow) | TestActions_EntryExitActions_ShouldKeepSameStateForUnhandledEventsShallow | ported |  |
| entry/exit actions > entry/exit actions > should keep the same state for unhandled events (deep) | TestActions_EntryExitActions_ShouldKeepSameStateForUnhandledEventsDeep | ported |  |
| entry/exit actions > entry/exit actions > should exit and enter the state for reentering self-transitions (shallow) | TestActions_EntryExitActions_ShouldExitAndEnterStateForReenteringSelfTransitionsShallow | ported |  |
| entry/exit actions > entry/exit actions > should exit and enter the state for reentering self-transitions (deep) | TestActions_EntryExitActions_ShouldExitAndEnterStateForReenteringSelfTransitionsDeep | ported |  |
| entry/exit actions > entry/exit actions > should exit current node and enter target node when target is not a descendent or ancestor of current | TestActions_EntryExitActions_ExitCurrentEnterTargetWhenTargetNotDescendantOrAncestor | ported |  |
| entry/exit actions > entry/exit actions > should exit current node and reenter target node when target is ancestor of current | TestActions_EntryExitActions_ExitCurrentReenterTargetWhenTargetIsAncestor | ported |  |
| entry/exit actions > entry/exit actions > should enter all descendents when target is a descendent of the source when using an reentering transition | TestActions_EntryExitActions_EnterAllDescendantsWhenTargetIsDescendantWithReenter | ported |  |
| entry/exit actions > entry/exit actions > should exit deep descendant during a default self-transition | TestActions_EntryExitActions_ShouldExitDeepDescendantDuringDefaultSelfTransition | ported |  |
| entry/exit actions > entry/exit actions > should exit deep descendant during a reentering self-transition | TestActions_EntryExitActions_ShouldExitDeepDescendantDuringReenteringSelfTransition | ported |  |
| entry/exit actions > entry/exit actions > should not reenter leaf state during its default self-transition | TestActions_EntryExitActions_ShouldNotReenterLeafStateDuringDefaultSelfTransition | ported |  |
| entry/exit actions > entry/exit actions > should reenter leaf state during its reentering self-transition | TestActions_EntryExitActions_ShouldReenterLeafStateDuringReenteringSelfTransition | ported |  |
| entry/exit actions > entry/exit actions > should not enter exited state when targeting its ancestor and when its former descendant gets selected through initial state | TestActions_EntryExitActions_NotEnterExitedStateTargetingAncestorFormerDescendant | ported |  |
| entry/exit actions > entry/exit actions > should not enter exited state when targeting its ancestor and when its latter descendant gets selected through initial state | TestActions_EntryExitActions_NotEnterExitedStateTargetingAncestorLatterDescendant | ported |  |

## API gaps

None. No `apigap_actions_1.go` was created.

## Shared helper copied

- `trackEntries` (`test/utils.ts:76-111`) is copied as `actions1TrackEntries[C]` in `xstate/action_test.go`. It prepends
  tracking actions to `StateNode.Entry` / `StateNode.Exit` (JS `state.entry.unshift(...)`) of the root and of every
  descendant reached through `StateNode.ChildStates()` (document order, mirroring `Object.values(state.states)`),
  labelling them with `strings.Join(child.Path, ".")`.
- The JS helper's guard against registering the same machine twice (`seen` WeakSet, utils.ts:77-80) is omitted:
  it is a misuse guard, not a test assertion, and would need package-level state.

## Ambiguities for review

- `actions1TrackEntries` relies on the implementation reading `StateNode.Entry`/`Exit` at execution time (the
  contract exposes them as mutable exported fields, `machine.go` StateNode). If the implementation snapshots
  entry/exit actions elsewhere at `CreateMachine` time, these 33 trackEntries-based tests need a different hook.
  Equivalent risk exists in JS (helper mutates `state.entry` arrays in place).
- The tracking actions are `xs.ActionFunc[C]` where `C` is the machine context type (all tracked machines here use `any`).
- JS L555-580: `'xstate.assign': spy` custom implementation must not override the built-in `assign`; ported as an
  `Implementations.Actions["xstate.assign"]` entry.


## Source section: actions_2

Source: `references/xstate/packages/core/test/actions.test.ts` lines 1234-2533 (37 tests: 36 `it`, 1 `it.skip`).
Go file: `xstate/action_test.go`. Gap file: `apigap_actions_2.go`.

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| entry/exit actions > parallel states > should return entry action defined on parallel state | `TestActions_ParallelStates_ShouldReturnEntryActionDefinedOnParallelState` | ported | uses actions2TrackEntries (copy of utils.ts trackEntries) |
| entry/exit actions > parallel states > should reenter parallel region when a parallel state gets reentered while targeting another region | `TestActions_ParallelStates_ShouldReenterRegionWhenParallelStateGetsReenteredTargetingAnotherRegion` | ported | uses actions2TrackEntries (copy of utils.ts trackEntries) |
| entry/exit actions > parallel states > should reenter parallel region when a parallel state is reentered while targeting another region | `TestActions_ParallelStates_ShouldReenterRegionWhenParallelStateIsReenteredTargetingAnotherRegion` | ported | uses actions2TrackEntries (copy of utils.ts trackEntries) |
| entry/exit actions > targetless transitions > shouldn't exit a state on a parent's targetless transition | `TestActions_TargetlessTransitions_ShouldNotExitStateOnParentsTargetlessTransition` | ported | uses actions2TrackEntries (copy of utils.ts trackEntries) |
| entry/exit actions > targetless transitions > shouldn't exit (and reenter) state on targetless delayed transition | `TestActions_TargetlessTransitions_ShouldNotExitAndReenterStateOnTargetlessDelayedTransition` | ported | uses actions2TrackEntries (copy of utils.ts trackEntries) |
| entry/exit actions > when reaching a final state > exit actions should be called when invoked machine reaches its final state | `TestActions_WhenReachingFinalState_ExitActionsShouldBeCalledWhenInvokedMachineReachesFinalState` | ported |  |
| entry/exit actions > when stopped > exit actions should not be called when stopping a machine | `TestActions_WhenStopped_ExitActionsShouldNotBeCalledWhenStoppingMachine` | ported |  |
| entry/exit actions > when stopped > an exit action executed when an interpreter reaches its final state should be called with the last received event | `TestActions_WhenStopped_ExitActionOnFinalStateShouldBeCalledWithLastReceivedEvent` | ported |  |
| entry/exit actions > when stopped > stopping an interpreter that receives events from its children exit handlers should not throw | `TestActions_WhenStopped_StoppingInterpreterReceivingEventsFromChildExitHandlersShouldNotThrow` | ported |  |
| entry/exit actions > when stopped > sent events from exit handlers of a stopped child should not be received by the parent | `TestActions_WhenStopped_SentEventsFromExitHandlersOfStoppedChildShouldNotBeReceivedByParent` | skipped-in-JS | it.skip in JS (L1541); body translated after t.Skip |
| entry/exit actions > when stopped > sent events from exit handlers of a done child should be received by the parent | `TestActions_WhenStopped_SentEventsFromExitHandlersOfDoneChildShouldBeReceivedByParent` | ported |  |
| entry/exit actions > when stopped > sent events from exit handlers of a stopped child should not be received by its children | `TestActions_WhenStopped_SentEventsFromExitHandlersOfStoppedChildShouldNotBeReceivedByItsChildren` | ported |  |
| entry/exit actions > when stopped > sent events from exit handlers of a done child should be received by its children | `TestActions_WhenStopped_SentEventsFromExitHandlersOfDoneChildShouldBeReceivedByItsChildren` | ported |  |
| entry/exit actions > when stopped > actors spawned in exit handlers of a stopped child should not be started | `TestActions_WhenStopped_ActorsSpawnedInExitHandlersOfStoppedChildShouldNotBeStarted` | ported | JS has no explicit expect; implicit 'no unhandled error' made explicit via WithUnhandledErrorHandler + t.Errorf |
| entry/exit actions > when stopped > should note execute referenced custom actions correctly when stopping an interpreter | `TestActions_WhenStopped_ShouldNoteExecuteReferencedCustomActionsWhenStoppingInterpreter` | ported |  |
| entry/exit actions > when stopped > should not execute builtin actions when stopping an interpreter | `TestActions_WhenStopped_ShouldNotExecuteBuiltinActionsWhenStoppingInterpreter` | ported |  |
| entry/exit actions > when stopped > should clear all scheduled events when the interpreter gets stopped | `TestActions_WhenStopped_ShouldClearAllScheduledEventsWhenInterpreterGetsStopped` | ported | JS has no explicit expect; implicit 'no unhandled error' made explicit via WithUnhandledErrorHandler + t.Errorf |
| entry/exit actions > when stopped > should execute exit actions of the settled state of the last initiated microstep | `TestActions_WhenStopped_ShouldExecuteExitActionsOfSettledStateOfLastInitiatedMicrostep` | ported |  |
| entry/exit actions > when stopped > should not execute exit actions of the settled state of the last initiated microstep after executing all actions from that microstep | `TestActions_WhenStopped_ShouldNotExecuteExitActionsOfSettledStateAfterAllMicrostepActions` | ported |  |
| initial actions > should support initial actions | `TestActions_InitialActions_ShouldSupportInitialActions` | ported | gap: WithMachineInitialActions |
| initial actions > should support initial actions from transition | `TestActions_InitialActions_ShouldSupportInitialActionsFromTransition` | ported | gap: WithStateInitialActions |
| initial actions > should execute actions of initial transitions only once when taking an explicit transition | `TestActions_InitialActions_ShouldExecuteInitialTransitionActionsOnlyOnceOnExplicitTransition` | ported | gap: WithStateInitialActions |
| initial actions > should execute actions of all initial transitions resolving to the initial state value | `TestActions_InitialActions_ShouldExecuteActionsOfAllInitialTransitionsResolvingToInitialValue` | ported | gaps: WithMachineInitialActions, WithStateInitialActions |
| initial actions > should execute actions of the initial transition when taking a root reentering self-transition | `TestActions_InitialActions_ShouldExecuteInitialTransitionActionsOnRootReenteringSelfTransition` | ported | gap: WithMachineInitialActions; spy.mockClear() -> replace spy with newSpy() |
| actions on invalid transition > should not recall previous actions | `TestActions_ActionsOnInvalidTransition_ShouldNotRecallPreviousActions` | ported |  |
| actions config > should reference actions defined in actions parameter of machine options (entry actions) | `TestActions_ActionsConfig_ShouldReferenceActionsFromMachineOptionsEntryActions` | ported |  |
| actions config > should reference actions defined in actions parameter of machine options (initial state) | `TestActions_ActionsConfig_ShouldReferenceActionsFromMachineOptionsInitialState` | ported |  |
| actions config > should be able to reference action implementations from action objects | `TestActions_ActionsConfig_ShouldReferenceActionImplementationsFromActionObjects` | ported |  |
| actions config > should work with anonymous functions (with warning) | `TestActions_ActionsConfig_ShouldWorkWithAnonymousFunctions` | ported |  |
| action meta > should provide the original params | `TestActions_ActionMeta_ShouldProvideOriginalParams` | ported | toHaveBeenCalledWith -> assert.Contains(spy.Calls(), args) |
| action meta > should provide undefined params when it was configured as string | `TestActions_ActionMeta_ShouldProvideUndefinedParamsWhenConfiguredAsString` | ported | toHaveBeenCalledWith -> assert.Contains(spy.Calls(), args) |
| action meta > should provide the action with resolved params when they are dynamic | `TestActions_ActionMeta_ShouldProvideResolvedParamsWhenDynamic` | ported | toHaveBeenCalledWith -> assert.Contains(spy.Calls(), args) |
| action meta > should resolve dynamic params using context value | `TestActions_ActionMeta_ShouldResolveDynamicParamsUsingContextValue` | ported | toHaveBeenCalledWith -> assert.Contains(spy.Calls(), args) |
| action meta > should resolve dynamic params using event value | `TestActions_ActionMeta_ShouldResolveDynamicParamsUsingEventValue` | ported | toHaveBeenCalledWith -> assert.Contains(spy.Calls(), args) |
| forwardTo() > should forward an event to a service | `TestActions_ForwardTo_ShouldForwardEventToService` | ported |  |
| forwardTo() > should forward an event to a service (dynamic) | `TestActions_ForwardTo_ShouldForwardEventToServiceDynamic` | ported |  |
| forwardTo() > should not cause an infinite loop when forwarding to undefined | `TestActions_ForwardTo_ShouldNotCauseInfiniteLoopWhenForwardingToUndefined` | ported |  |

## API gaps (`apigap_actions_2.go`)

- `WithStateInitialActions(cfg StateConfig, actions ...Action) StateConfig` — JS `initial: { target, actions }` on a state node. `StateConfig.Initial` is a string with no slot for initial-transition actions; at integration this should become an `InitialActions Actions` field.
- `WithMachineInitialActions[C](cfg MachineConfig[C], actions ...Action) MachineConfig[C]` — same for the root config (`MachineConfig.Initial`).

## File-local helpers

- `actions2TrackEntries[C]` — port of `trackEntries` from `core/test/utils.ts`: prepends tracker actions to `StateNode.Entry`/`Exit` of `machine.Root` and every descendant (via `StateNode.States`, path joined with `.`). Assumes the runtime reads `StateNode.Entry/Exit` (JS mutates `state.entry`/`state.exit` arrays). The JS "same machine twice" guard is omitted (not an assertion). Mutex-guarded.
- `actions2CameraMachine`, `actions2ForwardChild` — shared machine builders for identical JS configs.

## Ambiguities for review

- L1318 duplicates L1271 verbatim except for the it-name; both ported.
- L1719 ("actors spawned in exit handlers of a stopped child should not be started") and L1794 ("should clear all scheduled events ...") have no `expect`; JS fails only via a thrown/unhandled error. Go adds `xs.WithUnhandledErrorHandler(t.Errorf)` on the root actor; this assumes errors of a spawned child reach the root's handler (JS `reportUnhandledError` is global).
- L2032: `spy.mockClear()` has no `spy` equivalent; the closure reads a mutex-guarded `s` that is replaced with `newSpy()`.
- L2238-2385: `toHaveBeenCalledWith(x)` → `assert.Contains(t, s.Calls(), []any{x})`; params are compared as `map[string]any`, so the runtime must pass static params through unchanged and Expr params as returned.
- L2508: inline snapshot of `errorSpy.mock.calls` → exactly one call whose single arg is an `error` with message `Attempted to forward event to undefined actor. This risks an infinite loop in the sender.`; `forwardTo(undefined as any)` → `xs.ForwardTo(nil)`.
- L2387 / L2448: `guard: ({event}) => event.value === 42` → `a.Event.(xs.E)["value"] == 42` (int 42 sent).
- L1488: `receivedEvent` toEqual `{type:'NEXT'}` → `assert.Equal(xs.Ev("NEXT"), receivedEvent)`.
- L1390 uses real timers (`await sleep(50)`) in JS; ported with `sleep(50)`, no simulated clock.


## Source section: actions_3

Source: `references/xstate/packages/core/test/actions.test.ts` lines 2534-3791.
Go file: `xstate/action_test.go`. JS tests in range: 38. Ported: 37, N/A-type: 1, N/A-runtime: 0, skipped-in-JS: 0.

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| log() > should log a string | TestActions_Log_ShouldLogAString | ported | |
| log() > should log an expression | TestActions_Log_ShouldLogAnExpression | ported | |
| enqueueActions > should execute a simple referenced action | TestActions_EnqueueActions_ShouldExecuteASimpleReferencedAction | ported | |
| enqueueActions > should execute multiple different referenced actions | TestActions_EnqueueActions_ShouldExecuteMultipleDifferentReferencedActions | ported | |
| enqueueActions > should execute multiple same referenced actions | TestActions_EnqueueActions_ShouldExecuteMultipleSameReferencedActions | ported | |
| enqueueActions > should execute a parameterized action | TestActions_EnqueueActions_ShouldExecuteAParameterizedAction | ported | |
| enqueueActions > should execute a function | TestActions_EnqueueActions_ShouldExecuteAFunction | ported | |
| enqueueActions > should execute a builtin action using its own action creator | TestActions_EnqueueActions_ShouldExecuteBuiltinActionUsingItsOwnActionCreator | ported | |
| enqueueActions > should execute a builtin action using its bound action creator | TestActions_EnqueueActions_ShouldExecuteBuiltinActionUsingItsBoundActionCreator | ported | Contract has no bound creators (`enqueue.raise`); uses `a.Enqueue(xs.Raise(...))`, so the body matches the previous test |
| enqueueActions > should execute assigns when resolving the initial snapshot | TestActions_EnqueueActions_ShouldExecuteAssignsWhenResolvingInitialSnapshot | ported | `enqueue.assign` → `a.Enqueue(xs.Assign(...))` |
| enqueueActions > should be able to check a simple referenced guard | TestActions_EnqueueActions_ShouldBeAbleToCheckASimpleReferencedGuard | ported | |
| enqueueActions > should be able to check a parameterized guard | TestActions_EnqueueActions_ShouldBeAbleToCheckAParameterizedGuard | ported | |
| enqueueActions > should provide self | TestActions_EnqueueActions_ShouldProvideSelf | ported | `expect.assertions(1)` → counter asserted == 1 |
| enqueueActions > should be able to communicate with the parent using params | TestActions_EnqueueActions_ShouldBeAbleToCommunicateWithParentUsingParams | ported | `enqueue.sendTo` → `a.Enqueue(xs.SendTo(...))`; console.log → t.Log |
| enqueueActions > should enqueue.sendParent | TestActions_EnqueueActions_ShouldEnqueueSendParent | ported | `enqueue.sendParent` → `a.Enqueue(xs.SendParent(...))` |
| sendParent > TS: should compile for any event | TestActions_SendParent_TSShouldCompileForAnyEvent | ported | TS part dropped; runtime `toBeTruthy` kept as NotNil |
| sendTo > should be able to send an event to an actor | TestActions_SendTo_ShouldBeAbleToSendAnEventToAnActor | ported | |
| sendTo > should be able to send an event from expression to an actor | TestActions_SendTo_ShouldBeAbleToSendAnEventFromExpressionToAnActor | ported | |
| sendTo > should report a type error for an invalid event | TestActions_SendTo_ShouldReportATypeErrorForAnInvalidEvent | N/A-type | Only assertion is `@ts-expect-error` |
| sendTo > should be able to send an event to a named actor | TestActions_SendTo_ShouldBeAbleToSendAnEventToANamedActor | ported | |
| sendTo > should be able to send an event directly to an ActorRef | TestActions_SendTo_ShouldBeAbleToSendAnEventDirectlyToAnActorRef | ported | |
| sendTo > should be able to read from event | TestActions_SendTo_ShouldBeAbleToReadFromEvent | ported | `expect.assertions(1)` → counter asserted == 1 right after Send |
| sendTo > should error if given a string | TestActions_SendTo_ShouldErrorIfGivenAString | ported | `xs.SendTo("child", "a string")`; asserts single error observer call with exact message |
| sendTo > a self-event "handler" of an event sent using sendTo should be able to read updated snapshot of self | TestActions_SendTo_SelfEventHandlerShouldReadUpdatedSnapshotOfSelf | ported | |
| sendTo > should not attempt to deliver a delayed event to the spawned actor's ID that was stopped since the event was scheduled | TestActions_SendTo_ShouldNotDeliverDelayedEventToStoppedSpawnedActorID | ported | Uses gap `WithWarnHandler`; session id `x:113` captured via WithInspect (first `@xstate.actor` with id myChild) |
| sendTo > should not attempt to deliver a delayed event to the invoked actor's ID that was stopped since the event was scheduled | TestActions_SendTo_ShouldNotDeliverDelayedEventToStoppedInvokedActorID | ported | Same as above (`x:116`) |
| raise > should be able to send a delayed event to itself | TestActions_Raise_ShouldBeAbleToSendADelayedEventToItself | ported | |
| raise > should be able to send a delayed event to itself with delay = 0 | TestActions_Raise_ShouldBeAbleToSendADelayedEventToItselfWithDelay0 | ported | `await sleep(0)` → `sleep(10)` (see ambiguities) |
| raise > should be able to raise an event and respond to it in the same state | TestActions_Raise_ShouldBeAbleToRaiseAnEventAndRespondToItInTheSameState | ported | |
| raise > should be able to raise a delayed event and respond to it in the same state | TestActions_Raise_ShouldBeAbleToRaiseADelayedEventAndRespondToItInTheSameState | ported | |
| raise > should accept event expression | TestActions_Raise_ShouldAcceptEventExpression | ported | |
| raise > should be possible to access context in the event expression | TestActions_Raise_ShouldBePossibleToAccessContextInTheEventExpression | ported | |
| raise > should error if given a string | TestActions_Raise_ShouldErrorIfGivenAString | ported | `xs.Raise("a string")` |
| cancel > should be possible to cancel a raised delayed event | TestActions_Cancel_ShouldBePossibleToCancelARaisedDelayedEvent | ported | |
| cancel > should cancel only the delayed event in the machine that scheduled it when canceling the event with the same ID in the machine that sent it first | TestActions_Cancel_ShouldCancelOnlyInSchedulingMachineWhenSameIDSentFirst | ported | |
| cancel > should cancel only the delayed event in the machine that scheduled it when canceling the event with the same ID in the machine that sent it second | TestActions_Cancel_ShouldCancelOnlyInSchedulingMachineWhenSameIDSentSecond | ported | |
| cancel > should not try to clear an undefined timeout when canceling an unscheduled timer | TestActions_Cancel_ShouldNotClearUndefinedTimeoutWhenCancelingUnscheduledTimer | ported | Custom clock `actions3ClearSpyClock` (real `time.AfterFunc`, spied ClearTimeout) |
| cancel > should be able to cancel a just scheduled delayed event to a just invoked child | TestActions_Cancel_ShouldBeAbleToCancelJustScheduledDelayedEventToJustInvokedChild | ported | `delay: 0` → `Delay: ms(0)` (non-nil) |

## API gaps (`apigap_actions_3.go`)

- `WithWarnHandler(fn func(args ...any)) ActorOption` — mirrors `vi.spyOn(console, 'warn')`. Identical signature to the gap in `apigap_actions_4.go`; deduplicate at integration.

## Ambiguities for review

- JS 2717-2740 (`enqueue.raise` bound creator): EnqueueArgs has only `Enqueue`/`Check`; the Go test is identical to the "own action creator" test. Bound creators `enqueue.assign` (2748), `enqueue.sendTo` (2855), `enqueue.sendParent` (2908) likewise map to `a.Enqueue(xs.Assign/SendTo/SendParent(...))`.
- JS 3302 and 3376: inline snapshot hardcodes session ids `x:113` / `x:116` (test-order artifact). Go builds the expected string from the session id of the first actor with id `myChild` observed via `WithInspect` (`@xstate.actor`). Assumes the warn handler set on the root actor receives warnings from every actor in the system (JS spies the global console.warn).
- JS 3445: `await sleep(0)` relies on JS macrotask FIFO ordering after a `setTimeout(..., 0)`; Go uses `sleep(10)` because timers fire on other goroutines.
- JS 3171-3195 / 3567-3591: the error observer receives a JS `Error`; Go asserts the value implements `error` with the exact message.
- JS 3136-3169: `expect.assertions(1)` with a synchronous test; Go asserts the receive-callback assertion ran exactly once immediately after `Send`, assuming synchronous delivery to the spawned callback actor.
- Range boundary: the test at JS line 3792 ("should not be able to cancel a just scheduled non-delayed event to a just invoked child") is outside this chunk (ported in actions_4 as `TestActions_Cancel_ShouldNotBeAbleToCancelJustScheduledNonDelayedEventToJustInvokedChild`).


## Source section: actions_4

Source: `references/xstate/packages/core/test/actions.test.ts` lines 3792-4360.
Go file: `xstate/action_test.go`.

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| cancel > should not be able to cancel a just scheduled non-delayed event to a just invoked child | TestActions_Cancel_ShouldNotBeAbleToCancelJustScheduledNonDelayedEventToJustInvokedChild | ported | JS L3792 |
| assign action order > should preserve action order | TestActions_AssignActionOrder_ShouldPreserveActionOrder | ported | `captured` array recorded via spy; compared as `[][]any{{0},{1},{2}}` |
| assign action order > should deeply preserve action order | TestActions_AssignActionOrder_ShouldDeeplyPreserveActionOrder | ported | |
| assign action order > should capture correct context values on subsequent transitions | TestActions_AssignActionOrder_ShouldCaptureCorrectContextValuesOnSubsequentTransitions | ported | |
| types > assign actions should be inferred correctly | TestActions_Types_AssignActionsShouldBeInferredCorrectly | N/A-type | Only `@ts-expect-error` checks; no runtime expectations |
| action meta > base action objects should have meta.action as the same base action object | TestActions_ActionMeta_BaseActionObjectsShouldHaveMetaActionAsSameBaseActionObject | skipped-in-JS | `it.todo` |
| action meta > should provide self | TestActions_ActionMeta_ShouldProvideSelf | ported | `expect.assertions(1)` → entry action called exactly once; `self.send` defined → `Self` non-nil |
| actions > should call transition actions in document order for same-level parallel regions | TestActions_Actions_ShouldCallTransitionActionsInDocOrderForSameLevelParallelRegions | ported | |
| actions > should call transition actions in document order for states at different levels of parallel regions | TestActions_Actions_ShouldCallTransitionActionsInDocOrderForDifferentLevelParallelRegions | ported | |
| actions > should call an inline action responding to an initial raise with the raised event | TestActions_Actions_ShouldCallInlineActionOnInitialRaiseWithRaisedEvent | ported | `toHaveBeenCalledWith` → `assert.Contains(calls, args)` |
| actions > should call a referenced action responding to an initial raise with the raised event | TestActions_Actions_ShouldCallReferencedActionOnInitialRaiseWithRaisedEvent | ported | |
| actions > should call an inline action responding to an initial raise with updated (non-initial) context | TestActions_Actions_ShouldCallInlineActionOnInitialRaiseWithUpdatedContext | ported | |
| actions > should call a referenced action responding to an initial raise with updated (non-initial) context | TestActions_Actions_ShouldCallReferencedActionOnInitialRaiseWithUpdatedContext | ported | |
| actions > should call inline entry custom action with undefined parametrized action object | TestActions_Actions_ShouldCallInlineEntryCustomActionWithUndefinedParams | ported | `undefined` → `nil` |
| actions > should call inline entry builtin action with undefined parametrized action object | TestActions_Actions_ShouldCallInlineEntryBuiltinActionWithUndefinedParams | ported | JS `return {}` → return `a.Context` unchanged |
| actions > should call inline transition custom action with undefined parametrized action object | TestActions_Actions_ShouldCallInlineTransitionCustomActionWithUndefinedParams | ported | |
| actions > should call inline transition builtin action with undefined parameters | TestActions_Actions_ShouldCallInlineTransitionBuiltinActionWithUndefinedParams | ported | |
| actions > should call a referenced custom action with undefined params when it has no params and it is referenced using a string | TestActions_Actions_ShouldCallReferencedCustomActionWithUndefinedParamsWhenStringRef | ported | |
| actions > should call a referenced builtin action with undefined params when it has no params and it is referenced using a string | TestActions_Actions_ShouldCallReferencedBuiltinActionWithUndefinedParamsWhenStringRef | ported | |
| actions > should call a referenced custom action with the provided parametrized action object | TestActions_Actions_ShouldCallReferencedCustomActionWithProvidedParams | ported | params `{foo:'bar'}` → `map[string]any{"foo":"bar"}` |
| actions > should call a referenced builtin action with the provided parametrized action object | TestActions_Actions_ShouldCallReferencedBuiltinActionWithProvidedParams | ported | |
| actions > should warn if called in custom action | TestActions_Actions_ShouldWarnIfCalledInCustomAction | ported | console.warn spy → `xs.WithWarnHandler`; inline snapshot asserted exactly |
| actions > inline actions should not leak into provided actions object | TestActions_Actions_InlineActionsShouldNotLeakIntoProvidedActionsObject | ported | |

Totals: 23 JS tests — 21 ported, 1 N/A-type, 0 N/A-runtime, 1 skipped-in-JS.

## API gaps (`apigap_actions_4.go`)

- `func WithWarnHandler(fn func(args ...any)) ActorOption` — mirrors `vi.spyOn(console, 'warn')`, signature as prescribed by `docs/porting/core.md`. If another chunk adds the same gap, the duplicates must be merged at integration.

## Ambiguities for review

- JS L4314 (`should warn if called in custom action`): in JS the warnings come from global `console.warn`, triggered when `assign()/raise()/sendTo()/emit()` are *constructed* inside a custom action. In Go the handler is an actor option, so the implementation must detect "constructor called while executing a custom action of this actor" and route the warning to that actor's warn handler. The test asserts exactly four warnings in order (toMatchInlineSnapshot equals the full call list).
- JS L3981 (`should provide self`): `expect.assertions(1)` is translated as "entry action ran exactly once" + `Self != nil`.
- `toHaveBeenCalledWith(x)` is translated as `assert.Contains(spy.Calls(), []any{x})` (at least one call with exactly these args), matching vitest semantics. JS passes `(args, params)` to spies; the Go spies record only the asserted value.
- JS L3792 (cancel test) has no `await` before its assertion, so the Go test asserts synchronously after `Send`.
