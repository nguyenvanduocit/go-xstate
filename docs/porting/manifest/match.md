> Historical translation record. Chunk tags, API-gap filenames, line numbers,
> and run counts below describe the original porting work. See the current
> [core guide](../core.md) and [architecture](../../ARCHITECTURE.md) for the maintained layout.

## Source section: match_1

Source: `references/xstate/packages/core/test/match.test.ts` lines 1-123 (whole file, 122 lines; 13 `it` calls, no `it.each`/`it.skip`/`it.todo`).
Go file: `xstate/match_test.go`.

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| matchesState() > should return true if two states are equivalent | TestMatch_MatchesState_ShouldReturnTrueIfTwoStatesAreEquivalent | ported | JS L4; `toBe(false)` -> `assert.False` |
| matchesState() > should return true if two state values are equivalent | TestMatch_MatchesState_ShouldReturnTrueIfTwoStateValuesAreEquivalent | ported | JS L12 |
| matchesState() > should return true if two parallel states are equivalent | TestMatch_MatchesState_ShouldReturnTrueIfTwoParallelStatesAreEquivalent | ported | JS L17 |
| matchesState() > should return true if a state is a substate of a superstate | TestMatch_MatchesState_ShouldReturnTrueIfAStateIsASubstateOfASuperstate | ported | JS L37 |
| matchesState() > should return true if a state value is a substate of a superstate value | TestMatch_MatchesState_ShouldReturnTrueIfAStateValueIsASubstateOfASuperstateValue | ported | JS L43 |
| matchesState() > should return true if a parallel state value is a substate of a superstate value | TestMatch_MatchesState_ShouldReturnTrueIfAParallelStateValueIsASubstateOfASuperstateValue | ported | JS L51 |
| matchesState() > should return false if two states are not equivalent | TestMatch_MatchesState_ShouldReturnFalseIfTwoStatesAreNotEquivalent | ported | JS L62; `expect(!x).toBeTruthy()` kept as `assert.True(t, !x)` |
| matchesState() > should return false if parent state is more specific than child state | TestMatch_MatchesState_ShouldReturnFalseIfParentStateIsMoreSpecificThanChildState | ported | JS L68 |
| matchesState() > should return false if two state values are not equivalent | TestMatch_MatchesState_ShouldReturnFalseIfTwoStateValuesAreNotEquivalent | ported | JS L74 |
| matchesState() > should return false if a state is not a substate of a superstate | TestMatch_MatchesState_ShouldReturnFalseIfAStateIsNotASubstateOfASuperstate | ported | JS L78 |
| matchesState() > should return false if a state value is not a substate of a superstate value | TestMatch_MatchesState_ShouldReturnFalseIfAStateValueIsNotASubstateOfASuperstateValue | ported | JS L84; JS object key `false` is the string `"false"` |
| matchesState() > should mix/match string state values and object state values | TestMatch_MatchesState_ShouldMixMatchStringStateValuesAndObjectStateValues | ported | JS L92 |
| matches() method > should execute matchesState on a State given the parent state value | TestMatch_MatchesMethod_ShouldExecuteMatchesStateOnAStateGivenTheParentStateValue | ported | JS L98; actor not started (`getSnapshot()` before `start()`), mirrored as-is |

## API gaps

None. Uses existing `xs.MatchesState` (`util.go:34`) and `MachineSnapshot.Matches` (`snapshot.go:42`).

## Ambiguities for review

- JS L116: `createActor(machine).getSnapshot()` is called on an unstarted actor; the Go port calls `GetSnapshot()` without `Start()` to match. The implementation must return the initial snapshot for unstarted actors.
- JS L88: `{ foo: { false: 'baz' } }` uses the identifier `false` as an object key, which JS coerces to the string `"false"`; ported as `map[string]any{"false": "baz"}`.
