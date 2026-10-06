> Historical translation record. Chunk tags, API-gap filenames, line numbers,
> and run counts below describe the original porting work. See the current
> [core guide](../core.md) and [architecture](../../ARCHITECTURE.md) for the maintained layout.

## Source section: state_1

Source: `references/xstate/packages/core/test/state.test.ts` lines 1-511 (whole file, 21 `it` calls, no `it.each`/`it.skip`/`it.todo`).
Go file: `xstate/state_test.go`.

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| State > status > should show that a machine has not reached its final state | TestState_Status_ShouldShowMachineHasNotReachedFinalState | ported | JS L110; uses `state1ExampleMachine()` |
| State > status > should show that a machine has reached its final state | TestState_Status_ShouldShowMachineHasReachedFinalState | ported | JS L114 |
| State > .can > should return true for a simple event that results in a transition to a different state | TestState_Can_ShouldReturnTrueForSimpleEventTransitioningToDifferentState | ported | JS L122 |
| State > .can > should return true for an event object that results in a transition to a different state | TestState_Can_ShouldReturnTrueForEventObjectTransitioningToDifferentState | ported | JS L140; body identical to L122 in JS |
| State > .can > should return true for an event object that results in a new action | TestState_Can_ShouldReturnTrueForEventObjectResultingInNewAction | ported | JS L158; `'newAction'` -> `xs.ActionRef{Type: "newAction"}` (no implementation, as in JS) |
| State > .can > should return true for an event object that results in a context change | TestState_Can_ShouldReturnTrueForEventObjectResultingInContextChange | ported | JS L177 |
| State > .can > should return true for a reentering self-transition without actions | TestState_Can_ShouldReturnTrueForReenteringSelfTransitionWithoutActions | ported | JS L197 |
| State > .can > should return true for a reentering self-transition with reentry action | TestState_Can_ShouldReturnTrueForReenteringSelfTransitionWithReentryAction | ported | JS L212 |
| State > .can > should return true for a reentering self-transition with transition action | TestState_Can_ShouldReturnTrueForReenteringSelfTransitionWithTransitionAction | ported | JS L228 |
| State > .can > should return true for a targetless transition with actions | TestState_Can_ShouldReturnTrueForTargetlessTransitionWithActions | ported | JS L246 |
| State > .can > should return false for a forbidden transition | TestState_Can_ShouldReturnFalseForForbiddenTransition | ported | JS L263; `EV: undefined` -> `"EV": nil` |
| State > .can > should return false for an unknown event | TestState_Can_ShouldReturnFalseForUnknownEvent | ported | JS L280 |
| State > .can > should return true when a guarded transition allows the transition | TestState_Can_ShouldReturnTrueWhenGuardedTransitionAllowsTransition | ported | JS L298 |
| State > .can > should return false when a guarded transition disallows the transition | TestState_Can_ShouldReturnFalseWhenGuardedTransitionDisallowsTransition | ported | JS L321 |
| State > .can > should not spawn actors when determining if an event is accepted | TestState_Can_ShouldNotSpawnActorsWhenDeterminingIfEventIsAccepted | ported | JS L344; `spawned` is `atomic.Bool` (callback may run on another goroutine) |
| State > .can > should not execute assignments when used with non-started actor | TestState_Can_ShouldNotExecuteAssignmentsWithNonStartedActor | ported | JS L372; `toBeTruthy`/`toBeFalsy` -> `assert.True`/`assert.False` |
| State > .can > should not execute assignments when used with started actor | TestState_Can_ShouldNotExecuteAssignmentsWithStartedActor | ported | JS L394 |
| State > .can > should return true when non-first parallel region changes value | TestState_Can_ShouldReturnTrueWhenNonFirstParallelRegionChangesValue | ported | JS L416; `target: ['#foo', '#bar']` -> `Targets` |
| State > .can > should return true when transition targets a state that is already part of the current configuration but the final state value changes | TestState_Can_ShouldReturnTrueWhenTargetAlreadyActiveButFinalStateValueChanges | ported | JS L449; name shortened |
| State > .hasTag > should be able to check a tag after recreating a persisted state | TestState_HasTag_ShouldCheckTagAfterRecreatingPersistedState | ported | JS L480 |
| State > .status > should be 'stopped' after a running actor gets stopped | TestState_Status_ShouldBeStoppedAfterRunningActorGetsStopped | ported | JS L502 |

Totals: 21 JS tests -> 21 ported, 0 N/A-type, 0 N/A-runtime, 0 skipped-in-JS.

## API gaps

None. No `apigap_state_1.go` created.

## Ambiguities for review

- JS L22-107 top-level `exampleMachine` is a constructor `state1ExampleMachine()` (a package-level var would call the panicking `CreateMachine` at init). Each of the two tests that use it builds its own instance; JS tests do not depend on shared identity. `types: {} as { events: Events }` (L23-25) and the `Events` union (L5-20) are type-only and dropped.
- JS L35 `INERT: {}` -> `"INERT": {{}}` (one empty transition config: targetless, no actions).
- JS L72, L83, L91, L98 `'.'` targets -> `Target: "."` verbatim. The contract does not document `"."`; the implementation must resolve it as JS does (target = the source state itself, re-entered).
- JS L29/L37 `'enter'` / `'doSomething'` string actions have no implementation in JS either; kept as `xs.ActionRef` with no `Implementations`.
- JS L353-359: JS `assign(({spawn}) => ({ ref: spawn(...) }))` returns a partial context; Go returns the whole `ctx{Ref: ...}`. JS context starts as `{}`; Go uses `ctx{}` with a nil `Ref`.
- JS L378/L400: `assign((ctx) => { ...; return ctx; })` receives the assign-args object (named `ctx` in JS) and returns it; Go returns `a.Context`. Behaviourally irrelevant to the assertion (`executed` must stay false).
- JS L503-507 chains `.start().stop().getSnapshot()`; Go chains `Start().Stop().GetSnapshot()` (both return `*Actor[S]` in the contract).
