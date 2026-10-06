> Historical translation record. Chunk tags, API-gap filenames, line numbers,
> and run counts below describe the original porting work. See the current
> [core guide](../core.md) and [architecture](../../ARCHITECTURE.md) for the maintained layout.

## Source section: deterministic_1

Source: `references/xstate/packages/core/test/deterministic.test.ts` lines 1-295 (whole file, 17 `it` calls, no `it.each`/`it.skip`).
Go file: `xstate/deterministic_test.go`.

Shared describe-level machines `lightMachine` (JS L10-48) and `testMachine` (JS L50-67) are file-local
constructors `deterministic1LightMachine()` / `deterministic1TestMachine()`, called fresh in each test.

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| deterministic machine > machine transitions > should properly transition states based on event-like object | TestDeterministic_MachineTransitions_ShouldProperlyTransitionStatesBasedOnEventLikeObject | ported | JS L70 |
| deterministic machine > machine transitions > should not transition states for illegal transitions | TestDeterministic_MachineTransitions_ShouldNotTransitionStatesForIllegalTransitions | ported | JS L83; `toBe(previousSnapshot)` -> `assert.Same` |
| deterministic machine > machine transitions > should throw an error if not given an event | TestDeterministic_MachineTransitions_ShouldThrowAnErrorIfNotGivenAnEvent | ported | JS L106; `undefined as any` event -> nil `xs.Event`; `toThrow()` -> `assert.Panics`. Snapshot resolved from testMachine but transitioned on lightMachine, as in JS |
| deterministic machine > machine transitions > should transition to nested states as target | TestDeterministic_MachineTransitions_ShouldTransitionToNestedStatesAsTarget | ported | JS L116 |
| deterministic machine > machine transitions > should throw an error for transitions from invalid states | TestDeterministic_MachineTransitions_ShouldThrowAnErrorForTransitionsFromInvalidStates | ported | JS L126; ResolveState is inside the panic closure, as in JS |
| deterministic machine > machine transitions > should throw an error for transitions from invalid substates | TestDeterministic_MachineTransitions_ShouldThrowAnErrorForTransitionsFromInvalidSubstates | ported | JS L134 |
| deterministic machine > machine transitions > should use the machine.initialState when an undefined state is given | TestDeterministic_MachineTransitions_ShouldUseInitialStateWhenUndefinedStateIsGiven | ported | JS L142; `getInitialSnapshot(m, undefined)` -> `xs.GetInitialSnapshot(m)` (no input) |
| deterministic machine > machine transitions > should use the machine.initialState when an undefined state is given (unhandled event) | TestDeterministic_MachineTransitions_ShouldUseInitialStateWhenUndefinedStateIsGivenUnhandledEvent | ported | JS L149; body identical to previous test in JS (event TIMER, expects 'yellow') |
| deterministic machine > machine transition with nested states > should properly transition a nested state | TestDeterministic_NestedStates_ShouldProperlyTransitionANestedState | ported | JS L158 |
| deterministic machine > machine transition with nested states > should transition from initial nested states | TestDeterministic_NestedStates_ShouldTransitionFromInitialNestedStates | ported | JS L168 |
| deterministic machine > machine transition with nested states > should transition from deep initial nested states | TestDeterministic_NestedStates_ShouldTransitionFromDeepInitialNestedStates | ported | JS L178; identical body to previous test in JS |
| deterministic machine > machine transition with nested states > should bubble up events that nested states cannot handle | TestDeterministic_NestedStates_ShouldBubbleUpEventsThatNestedStatesCannotHandle | ported | JS L188 |
| deterministic machine > machine transition with nested states > should not transition from illegal events | TestDeterministic_NestedStates_ShouldNotTransitionFromIllegalEvents | ported | JS L198; `toBe(previousSnapshot)` -> `assert.Same` |
| deterministic machine > machine transition with nested states > should transition to the deepest initial state | TestDeterministic_NestedStates_ShouldTransitionToTheDeepestInitialState | ported | JS L226 |
| deterministic machine > machine transition with nested states > should return the same state if no transition occurs | TestDeterministic_NestedStates_ShouldReturnTheSameStateIfNoTransitionOccurs | ported | JS L240; `toBe` -> `assert.Same` |
| deterministic machine > state key names > should work with substate nodes that have the same key | TestDeterministic_StateKeyNames_ShouldWorkWithSubstateNodesThatHaveTheSameKey | ported | JS L276; describe-level machine (JS L255-274) built inside the test; `onEntry`/`onExit` have no implementations in JS either |
| deterministic machine > forbidden events > undefined transitions should forbid events | TestDeterministic_ForbiddenEvents_UndefinedTransitionsShouldForbidEvents | ported | JS L285; `TIMER: undefined` -> `"TIMER": nil` |

Totals: 17 JS tests; 17 ported, 0 N/A-type, 0 N/A-runtime, 0 skipped-in-JS.

## API gaps

None. No `apigap_deterministic_1.go` was created.

## Ambiguities for reviewer

- JS L106-114: `transition(lightMachine, undefined as any)` is mapped to passing a nil `xs.Event` interface; the Go implementation must panic on a nil event for this test to pass.
- JS L126-140: `toThrow()` covers a throw from either `resolveState` or `transition`; the Go closure wraps both calls so either panic satisfies the assertion.
- JS L142/L149 and L168/L178 are duplicate bodies in the JS source; both copies are ported as-is.
- Test names for the "machine transition with nested states" describe use the shortened prefix `NestedStates`; names for JS L142/L149 drop the words "the machine." to stay readable.
