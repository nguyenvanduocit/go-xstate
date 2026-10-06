> Historical translation record. Chunk tags, API-gap filenames, line numbers,
> and run counts below describe the original porting work. See the current
> [core guide](../core.md) and [architecture](../../ARCHITECTURE.md) for the maintained layout.

## Source section: microstep_1

Source: `references/xstate/packages/core/test/microstep.test.ts` lines 1-335 (whole file).
Go file: `xstate/microstep_test.go`.

Totals: 11 JS tests; 11 ported, 0 N/A-type, 0 N/A-runtime, 0 skipped-in-JS.

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| machine.microstep() > should return an array of states from all microsteps | TestMicrostep_MachineMicrostep_ShouldReturnAnArrayOfStatesFromAllMicrosteps | ported | `getInitialSnapshot(actorScope)` → `GetInitialSnapshot(actorScope, nil)` |
| machine.microstep() > should return the states from microstep (transient) | TestMicrostep_MachineMicrostep_ShouldReturnTheStatesFromMicrostepTransient | ported | `resolveState({value:'first'})` → `ResolveState(ResolveStateConfig[any]{Value: "first"})` |
| machine.microstep() > should return the states from microstep (raised event) | TestMicrostep_MachineMicrostep_ShouldReturnTheStatesFromMicrostepRaisedEvent | ported |  |
| machine.microstep() > should return a single-item array for normal transitions | TestMicrostep_MachineMicrostep_ShouldReturnASingleItemArrayForNormalTransitions | ported |  |
| machine.microstep() > each state should preserve their internal queue | TestMicrostep_MachineMicrostep_EachStateShouldPreserveTheirInternalQueue | ported |  |
| getMicrosteps > should return microsteps with actions | TestMicrostep_GetMicrosteps_ShouldReturnMicrostepsWithActions | ported | `microsteps[i][0]` → `.Snapshot`, `microsteps[i][1]` → `.Actions` |
| getMicrosteps > should capture actions from raised events | TestMicrostep_GetMicrosteps_ShouldCaptureActionsFromRaisedEvents | ported |  |
| getInitialMicrosteps > should return initial microsteps with entry actions | TestMicrostep_GetInitialMicrosteps_ShouldReturnInitialMicrostepsWithEntryActions | ported |  |
| getInitialMicrosteps > should capture actions from initial always transitions | TestMicrostep_GetInitialMicrosteps_ShouldCaptureActionsFromInitialAlwaysTransitions | ported |  |
| getInitialMicrosteps > should work with nested initial states | TestMicrostep_GetInitialMicrosteps_ShouldWorkWithNestedInitialStates | ported | `{parent:'child'}` → `map[string]any{"parent": "child"}` |
| getInitialMicrosteps > should pass input to context function | TestMicrostep_GetInitialMicrosteps_ShouldPassInputToContextFunction | ported | input `{value: 42}` → local `input{Value: 42}`; context → `ctx{Count: 42}` |

## API gaps (`apigap_microstep_1.go`)

- `func CreateInertActorScope(logic ActorLogic) *ActorScope` — mirrors `createInertActorScope` from `src/getNextSnapshot.ts:14` (imported by the test at JS line 7). The contract exposes `StateMachine.Microstep/GetInitialSnapshot(scope *ActorScope, ...)` but no way to build an inert scope. If another chunk (e.g. a later microstep chunk) adds the same name, dedupe at integration.

## Ambiguities

- File-local helpers: `microstep1Values` (maps `states.map(s => s.value)`) and `microstep1Noop` (the `() => {}` inline actions).
- `toHaveLength(n)` on the microsteps array → `require.Len` (JS `expect` aborts the test on failure; guards later indexing). JS line 326 (`should pass input to context function`) indexes `microsteps[0]` without a length check; Go adds `require.NotEmpty` only to avoid an index-out-of-range panic — no assertion weakened.
- Action-count assertions (JS lines 213, 217, 263, 293, 297, 318) depend on `GetMicrosteps`/`GetInitialMicrosteps` returning exactly the executable custom actions (no raise/internal actions counted). In JS line 228 test (`raise` in actions) only values are asserted, so no count ambiguity there.
