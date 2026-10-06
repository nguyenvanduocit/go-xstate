> Historical translation record. Chunk tags, API-gap filenames, line numbers,
> and run counts below describe the original porting work. See the current
> [core guide](../core.md) and [architecture](../../ARCHITECTURE.md) for the maintained layout.

## Source section: activities_1

Source: `references/xstate/packages/core/test/activities.test.ts` lines 1-442 (whole file, 13 `it` calls).
Go file: `xstate/activities_test.go`.

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| invocations (activities) > identifies initial root invocations | TestActivities_IdentifiesInitialRootInvocations | ported | JS L8 |
| invocations (activities) > identifies initial invocations | TestActivities_IdentifiesInitialInvocations | ported | JS L22 |
| invocations (activities) > identifies initial deep invocations | TestActivities_IdentifiesInitialDeepInvocations | ported | JS L41 |
| invocations (activities) > identifies start invocations | TestActivities_IdentifiesStartInvocations | ported | JS L65 |
| invocations (activities) > identifies start invocations for child states and active invocations | TestActivities_IdentifiesStartInvocationsForChildStatesAndActiveInvocations | ported | JS L92 |
| invocations (activities) > identifies stop invocations for child states | TestActivities_IdentifiesStopInvocationsForChildStates | ported | JS L130 |
| invocations (activities) > identifies multiple stop invocations for child and parent states | TestActivities_IdentifiesMultipleStopInvocationsForChildAndParentStates | ported | JS L173 |
| invocations (activities) > should activate even if there are subsequent always but blocked transition | TestActivities_ShouldActivateEvenIfThereAreSubsequentAlwaysButBlockedTransition | ported | JS L219 |
| invocations (activities) > should remember the invocations even after an ignored event | TestActivities_ShouldRememberTheInvocationsEvenAfterAnIgnoredEvent | ported | JS L248; `not.toBeCalled()` -> `cleanupSpy.Count() == 0` |
| invocations (activities) > should remember the invocations when transitioning within the invoking state | TestActivities_ShouldRememberTheInvocationsWhenTransitioningWithinTheInvokingState | ported | JS L281; `not.toBeCalled()` -> `cleanupSpy.Count() == 0` |
| invocations (activities) > should start a new actor when leaving an invoking state and entering a new one that invokes the same actor type | TestActivities_ShouldStartNewActorWhenLeavingInvokingStateAndEnteringNewOneInvokingSameActorType | ported | JS L317; Go name shortened (articles dropped) to stay ~100 chars |
| invocations (activities) > should start a new actor when reentering the invoking state during a reentering self transition | TestActivities_ShouldStartNewActorWhenReenteringInvokingStateDuringReenteringSelfTransition | ported | JS L361; Go name shortened (articles dropped) |
| invocations (activities) > should have stopped after automatic transitions | TestActivities_ShouldHaveStoppedAfterAutomaticTransitions | ported | JS L403 |

## API gaps

None. No `apigap_activities_1.go` created.

## Notes for reviewer

- JS `let active = false` flags are `atomic.Bool` and the `actual` log is mutex-guarded (`activities1Log`) because callback actors may run off the test goroutine under `-race`; assertions are unchanged (`toBe(true/false)` -> `assert.Equal(t, true/false, ...)`).
- JS `counter++` with `localId = counter` before increment -> `counter.Add(1) - 1` (same 0-based ids).
- `fromCallback(() => { active = true; })` (no cleanup) -> callback returning `nil`.
- `setup({actors: {fooActor}})` -> `xs.NewSetup[any](xs.Implementations{Actors: ...})`.
