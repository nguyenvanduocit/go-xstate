> Historical translation record. Chunk tags, API-gap filenames, line numbers,
> and run counts below describe the original porting work. See the current
> [core guide](../core.md) and [architecture](../../ARCHITECTURE.md) for the maintained layout.

## Source section: rehydration_1

Source: `references/xstate/packages/core/test/rehydration.test.ts` lines 1-536 (whole file; 18 `it` calls, no `it.each`/`it.skip`).
Go file: `xstate/rehydration_test.go`.

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| rehydration > using persisted state > should be able to use `hasTag` immediately | TestRehydration_UsingPersistedState_ShouldBeAbleToUseHasTagImmediately | ported | JS L14; JSON round-trip via `rehydration1JSONRoundTrip` |
| rehydration > using persisted state > should not call exit actions when machine gets stopped immediately | TestRehydration_UsingPersistedState_ShouldNotCallExitActionsWhenStoppedImmediately | ported | JS L35 |
| rehydration > using persisted state > should get correct result back from `can` immediately | TestRehydration_UsingPersistedState_ShouldGetCorrectResultFromCanImmediately | ported | JS L58; `JSON.stringify(snapshot)` -> `json.Marshal(snap.ToJSON())` |
| rehydration > using state value > should be able to use `hasTag` immediately | TestRehydration_UsingStateValue_ShouldBeAbleToUseHasTagImmediately | ported | JS L80 |
| rehydration > using state value > should not call exit actions when machine gets stopped immediately | TestRehydration_UsingStateValue_ShouldNotCallExitActionsWhenStoppedImmediately | ported | JS L103 |
| rehydration > using state value > should error on incompatible state value (shallow) | TestRehydration_UsingStateValue_ShouldErrorOnIncompatibleStateValueShallow | ported | JS L127; `toThrowError(/invalid/)` -> `assert.Regexp("invalid", panicMessage(fn))` |
| rehydration > using state value > should error on incompatible state value (deep) | TestRehydration_UsingStateValue_ShouldErrorOnIncompatibleStateValueDeep | ported | JS L140 |
| rehydration > should not replay actions when starting from a persisted state | TestRehydration_ShouldNotReplayActionsWhenStartingFromPersistedState | ported | JS L159 |
| rehydration > should be able to stop a rehydrated child | TestRehydration_ShouldBeAbleToStopRehydratedChild | ported | JS L178; `not.toThrow` -> `assert.NotPanics` |
| rehydration > a rehydrated active child should be registered in the system | TestRehydration_RehydratedActiveChildShouldBeRegisteredInSystem | ported | JS L213; `toBeDefined` -> `assert.NotNil` |
| rehydration > a rehydrated done child should not be registered in the system | TestRehydration_RehydratedDoneChildShouldNotBeRegisteredInSystem | ported | JS L241; `toBeUndefined` -> `assert.Nil` |
| rehydration > a rehydrated done child should not re-notify the parent about its completion | TestRehydration_RehydratedDoneChildShouldNotReNotifyParentAboutCompletion | ported | JS L269; `mockClear` -> baseline count |
| rehydration > should be possible to persist a rehydrated actor that got its children rehydrated | TestRehydration_ShouldBePossibleToPersistRehydratedActorWithRehydratedChildren | ported | JS L306; `(persisted as any).children` read after JSON round-trip |
| rehydration > should complete on a rehydrated final state | TestRehydration_ShouldCompleteOnRehydratedFinalState | ported | JS L332 |
| rehydration > should error on a rehydrated error state | TestRehydration_ShouldErrorOnRehydratedErrorState | ported | JS L359; `await sleep(0)` -> wait on signal resolved by the error observer |
| rehydration > shouldn't re-notify the parent about the error when rehydrating | TestRehydration_ShouldNotReNotifyParentAboutErrorWhenRehydrating | ported | JS L392; `await sleep(0)` -> wait on signal resolved in onError action; `mockClear` -> baseline count |
| rehydration > should continue syncing snapshots | TestRehydration_ShouldContinueSyncingSnapshots | ported | JS L426; BehaviorSubject -> `newBehaviorSubject`; `mockClear` -> slice calls after baseline |
| rehydration > should be able to rehydrate an actor deep in the tree | TestRehydration_ShouldBeAbleToRehydrateActorDeepInTree | ported | JS L469 |

## API gaps

None. No `apigap_rehydration_1.go` was created.

## Ambiguities for review

- JS L378/L415 `await sleep(0)`: JS waits one macrotask so the rejected promise is processed. Go promises run on goroutines, so `sleep(0)` would be racy. The port waits on a signal: resolved by the existing error observer (L374, `preventUnhandledErrorListener`) in the first test, and by the onError action, next to the spy call, in the second.
- `spy.mockClear()` (L297, L418, L461): `xstate/helpers_test.go` has no clear method, so the port records a baseline count and asserts on calls made after it. The resulting assertions are the same.
- L326-329 `Object.keys(children).length` / `Object.values(children)[0].src`: the persisted snapshot is JSON round-tripped to `map[string]any` and then read. If the Go persisted shape is not JSON-serializable (e.g. it holds an ActorRef), this test needs a typed accessor instead.
- L35-56/L103-125 exit-action tests: `actual` is appended inside actions without a mutex. That is safe if exit actions run synchronously on Stop, which is the expected behavior.
