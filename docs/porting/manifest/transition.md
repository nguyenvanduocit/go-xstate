> Historical translation record. Chunk tags, API-gap filenames, line numbers,
> and run counts below describe the original porting work. See the current
> [core guide](../core.md) and [architecture](../../ARCHITECTURE.md) for the maintained layout.

## Source section: transition_1

Source: `references/xstate/packages/core/test/transition.test.ts` lines 1-956 (whole file, 24 tests).
Go file: `xstate/transition_test.go`. Gap file: `apigap_transition_1.go`.

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| transition function > should capture actions | TestTransition_TransitionFunction_ShouldCaptureActions | ported | `toEqual([objectContaining...])` → exact length + Type/Params per element |
| transition function > should not execute a referenced serialized action | TestTransition_TransitionFunction_ShouldNotExecuteAReferencedSerializedAction | ported | |
| transition function > should capture enqueued actions | TestTransition_TransitionFunction_ShouldCaptureEnqueuedActions | ported | |
| transition function > delayed raise actions should be returned | TestTransition_TransitionFunction_DelayedRaiseActionsShouldBeReturned | ported | params.delay asserted as `ms(10)` (time.Duration) |
| transition function > raise actions related to delayed transitions should be returned | TestTransition_TransitionFunction_RaiseActionsRelatedToDelayedTransitionsShouldBeReturned | ported | after event compared by EventType (plus full equality when it is an `xs.E`) |
| transition function > cancel action should be returned | TestTransition_TransitionFunction_CancelActionShouldBeReturned | ported | toContainEqual → `transition1ContainsAction` |
| transition function > sendTo action should be returned | TestTransition_TransitionFunction_SendToActionShouldBeReturned | ported | |
| transition function > emit actions should be returned | TestTransition_TransitionFunction_EmitActionsShouldBeReturned | ported | |
| transition function > log actions should be returned | TestTransition_TransitionFunction_LogActionsShouldBeReturned | ported | |
| transition function > should calculate the next snapshot for transition logic | TestTransition_TransitionFunction_ShouldCalculateTheNextSnapshotForTransitionLogic | ported | uses gap `InitialTransitionActorLogic` / `TransitionActorLogic` |
| transition function > should calculate the next snapshot for machine logic | TestTransition_TransitionFunction_ShouldCalculateTheNextSnapshotForMachineLogic | ported | |
| transition function > should not execute entry actions | TestTransition_TransitionFunction_ShouldNotExecuteEntryActions | ported | |
| transition function > should not execute transition actions | TestTransition_TransitionFunction_ShouldNotExecuteTransitionActions | ported | |
| transition function > delayed events example (experimental) | TestTransition_TransitionFunction_DelayedEventsExampleExperimental | ported | JSON.stringify/parse → `json.Marshal(snap.ToJSON())`/`json.Unmarshal`; non-awaited `postEvent` → sync transition+store, then goroutine for action execution; db guarded by mutex |
| transition function > serverless workflow example (experimental) | TestTransition_TransitionFunction_ServerlessWorkflowExampleExperimental | ported | uses gap `ResolveReferencedActor`; both promise actors typed `FromPromise[any]` so the resolved logic can be cast to `TypedActorLogic[*PromiseSnapshot[any]]` |
| getNextTransitions > should return all transitions from current state | TestTransition_GetNextTransitions_ShouldReturnAllTransitionsFromCurrentState | ported | |
| getNextTransitions > should include guarded transitions regardless of guard result | TestTransition_GetNextTransitions_ShouldIncludeGuardedTransitionsRegardlessOfGuardResult | ported | |
| getNextTransitions > should include always (eventless) transitions | TestTransition_GetNextTransitions_ShouldIncludeAlwaysEventlessTransitions | ported | |
| getNextTransitions > should include after (delayed) transitions | TestTransition_GetNextTransitions_ShouldIncludeAfterDelayedTransitions | ported | |
| getNextTransitions > should include transitions from parent states in depth-first order | TestTransition_GetNextTransitions_ShouldIncludeTransitionsFromParentStatesInDepthFirstOrder | ported | |
| getNextTransitions > should include all guarded transitions from different state nodes with same event type | TestTransition_GetNextTransitions_ShouldIncludeAllGuardedTransitionsFromDifferentStateNodesWithSameEventType | ported | |
| getNextTransitions > should return transitions from parallel states in document order | TestTransition_GetNextTransitions_ShouldReturnTransitionsFromParallelStatesInDocumentOrder | ported | |
| getNextTransitions > should return transitions from deeply nested compound states in depth-first order | TestTransition_GetNextTransitions_ShouldReturnTransitionsFromDeeplyNestedCompoundStatesInDepthFirstOrder | ported | |
| getNextTransitions > should return transitions from parallel states with nested compound states | TestTransition_GetNextTransitions_ShouldReturnTransitionsFromParallelStatesWithNestedCompoundStates | ported | |

Totals: 24 JS tests — 24 ported, 0 N/A-type, 0 N/A-runtime, 0 skipped-in-JS.

## API gaps (`apigap_transition_1.go`)

- `TransitionActorLogic[S Snapshot](logic TypedActorLogic[S], s S, e Event) (S, []ExecutableAction)` — JS `transition()` accepts any actor logic; contract `Transition` is machine-only.
- `InitialTransitionActorLogic[S Snapshot](logic TypedActorLogic[S], input ...any) (S, []ExecutableAction)` — same for `initialTransition()`.
- `ResolveReferencedActor[C any](m *StateMachine[C], src string) ActorLogic` — mirrors `resolveReferencedActor` from `src/utils.ts`, imported directly by the test (JS line 24).

## Ambiguities for review

- `ExecutableAction.Params` of built-in actions is assumed to be `map[string]any` with the JS key names (`delay`, `event`, `id`, `sendId`, `targetId`, `value`, `src`, `input`, `systemId`); `delay` is assumed to be `time.Duration`. The contract only says `Params any` (JS lines 145-150, 170-176, 202-209, 234-252, 276-284, 309-316, 433-441, 507-518).
- Referenced-action params (`{ a: 1 }`, dynamic `{ msg }`) are written as `map[string]any` (JS lines 46, 54-56, 66, 82-86).
- JS lines 170-176: the after-event is `{ type: 'xstate.after.10.(machine).a' }`; Go asserts `EventType()` always, and full `xs.E` equality only when the implementation uses `xs.E`.
- JS lines 459-479 / 547-569: `postEvent` is never awaited in JS. Go runs its synchronous part (resolve + transition + db write) inline and the action loop in a goroutine, which reproduces JS ordering (db holds `logSent` before `postEvent({type:'sent'})` reads it). Timing sleeps (15 ms / 10 ms) are kept as in JS and are equally tight.
- JS line 518 `assert('transition' in logic)`: Go checks the cast to `TypedActorLogic[*PromiseSnapshot[any]]`, which is stricter (needed to call the typed `CreateActor`). JS passes the whole `spawnAction.params` as actor options; Go forwards `id`, `input` and (when set) `systemId` — `actorRef`/`src` are not actor options with an effect here.
- JS line 532 (`switch` case without `break`) falls through to `default: break` — no behaviour, mirrored with an empty `default`.
- Go-map ordering: every `on`-order expectation in the getNextTransitions tests is already alphabetically sorted (GO_B < GO_C; GO_D only one key per node), so no expectation was reordered.
