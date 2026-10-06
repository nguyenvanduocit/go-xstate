> Historical translation record. Chunk tags, API-gap filenames, line numbers,
> and run counts below describe the original porting work. See the current
> [core guide](../core.md) and [architecture](../../ARCHITECTURE.md) for the maintained layout.

# transient_1 — packages/core/test/transient.test.ts lines 1-804 (whole file, 803 lines)

Go file: `xstate/transient_test.go`. Shared top-level `greetingMachine`
(JS lines 6-28) is ported as `transient1GreetingMachine()` (a func, so package init does not hit stubs).

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| transient states (eventless transitions) > should choose the first candidate target that matches the guard 1 | TestTransient_ShouldChooseTheFirstCandidateTargetThatMatchesTheGuard1 | ported | |
| transient states (eventless transitions) > should choose the first candidate target that matches the guard 2 | TestTransient_ShouldChooseTheFirstCandidateTargetThatMatchesTheGuard2 | ported | |
| transient states (eventless transitions) > should choose the final candidate without a guard if none others match | TestTransient_ShouldChooseTheFinalCandidateWithoutAGuardIfNoneOthersMatch | ported | |
| transient states (eventless transitions) > should carry actions from previous transitions within same step | TestTransient_ShouldCarryActionsFromPreviousTransitionsWithinSameStep | ported | |
| transient states (eventless transitions) > should execute all internal events one after the other | TestTransient_ShouldExecuteAllInternalEventsOneAfterTheOther | ported | |
| transient states (eventless transitions) > should execute all eventless transitions in the same microstep | TestTransient_ShouldExecuteAllEventlessTransitionsInTheSameMicrostep | ported | `stateIn({B:'B3'})` → `xs.StateIn(map[string]any{"B":"B3"})` |
| transient states (eventless transitions) > should check for automatic transitions even after microsteps are done | TestTransient_ShouldCheckForAutomaticTransitionsEvenAfterMicrostepsAreDone | ported | |
| transient states (eventless transitions) > should determine the resolved initial state from the transient state | TestTransient_ShouldDetermineTheResolvedInitialStateFromTheTransientState | ported | actor not started, as in JS |
| transient states (eventless transitions) > should determine the resolved state from an initial transient state | TestTransient_ShouldDetermineTheResolvedStateFromAnInitialTransientState | ported | |
| transient states (eventless transitions) > should select eventless transition before processing raised events | TestTransient_ShouldSelectEventlessTransitionBeforeProcessingRaisedEvents | ported | |
| transient states (eventless transitions) > should not select wildcard for eventless transition | TestTransient_ShouldNotSelectWildcardForEventlessTransition | ported | |
| transient states (eventless transitions) > should work with transient transition on root | TestTransient_ShouldWorkWithTransientTransitionOnRoot | ported | |
| transient states (eventless transitions) > shouldn't crash when invoking a machine with initial transient transition depending on custom data | TestTransient_ShouldntCrashWhenInvokingMachineWithInitialTransientTransitionOnCustomData | ported | name shortened (~100 char limit); `not.toThrow()` → `assert.NotPanics` |
| transient states (eventless transitions) > should be taken even in absence of other transitions | TestTransient_ShouldBeTakenEvenInAbsenceOfOtherTransitions | ported | |
| transient states (eventless transitions) > should select subsequent transient transitions even in absence of other transitions | TestTransient_ShouldSelectSubsequentTransientTransitionsEvenInAbsenceOfOtherTransitions | ported | |
| transient states (eventless transitions) > events that trigger eventless transitions should be preserved in guards | TestTransient_EventsThatTriggerEventlessTransitionsShouldBePreservedInGuards | ported | in-guard `expect` → `assert.Equal` inside the guard |
| transient states (eventless transitions) > events that trigger eventless transitions should be preserved in actions | TestTransient_EventsThatTriggerEventlessTransitionsShouldBePreservedInActions | ported | `expect.assertions(3)` → counter asserted == 3 |
| transient states (eventless transitions) > should avoid infinite loops with eventless transitions | TestTransient_ShouldAvoidInfiniteLoopsWithEventlessTransitions | ported | `options.maxIterations` → gap `WithOptions(MachineOptions{MaxIterations:100})`; `expect.assertions(1)` → error-callback counter == 1; `/infinite loop/i` → `assert.Regexp("(?i)infinite loop")` |
| transient states (eventless transitions) > should avoid infinite loops with raised events | TestTransient_ShouldAvoidInfiniteLoopsWithRaisedEvents | ported | same as above |
| transient states (eventless transitions) > shouldn't end up in an infinite loop when selecting the fallback target | TestTransient_ShouldntEndUpInAnInfiniteLoopWhenSelectingTheFallbackTarget | ported | |
| transient states (eventless transitions) > shouldn't end up in an infinite loop when selecting a guarded target | TestTransient_ShouldntEndUpInAnInfiniteLoopWhenSelectingAGuardedTarget | ported | |
| transient states (eventless transitions) > shouldn't end up in an infinite loop when executing a fire-and-forget action that doesn't change state | TestTransient_ShouldntEndUpInInfiniteLoopWhenExecutingFireAndForgetActionNoStateChange | ported | name shortened; JS `throw new Error` → `panic(errors.New(...))` |
| transient states (eventless transitions) > should loop (but not infinitely) for assign actions | TestTransient_ShouldLoopButNotInfinitelyForAssignActions | ported | |
| transient states (eventless transitions) > should execute an always transition after a raised transition even if that raised transition doesn't change the state | TestTransient_ShouldExecuteAlwaysTransitionAfterRaisedTransitionEvenIfNoStateChange | ported | `spy.mockClear()` → record `s.Count()` after start, assert `s.Calls()[cleared:]` |

Totals: 24 JS tests — 24 ported, 0 N/A-type, 0 N/A-runtime, 0 skipped-in-JS.

## API gaps (`apigap_transient_1.go`)

- `type MachineOptions struct{ MaxIterations int }` — mirrors JS `MachineOptions` (the `options` key of the
  createMachine config; 0 = Infinity, the JS default).
- `func (m *StateMachine[C]) WithOptions(opts MachineOptions) *StateMachine[C]` — mirrors setting `options` in
  the createMachine config. `MachineConfig` has no `Options` field and contract files cannot be edited; at
  integration this may be better folded into `MachineConfig[C].Options`.

## Ambiguities for review

- JS lines 569-641: errors delivered to `subscribe({ error })` may be an `error` or another panic value; the Go
  test matches the regexp against `err.Error()` when it is an `error`, otherwise `fmt.Sprint(err)`.
- JS lines 535-567 / 569-641: `expect.assertions(n)` is mapped to a counter incremented in the callback(s)
  where the `expect` ran, asserted at the end of the test (synchronous in JS and Go).
- JS line 27 `RECHECK: '#greeting'` (root self-target) is ported as `Target: "#greeting"`.
- JS line 410 test: the timer machine input is a local struct `timerInput{Duration}` read via
  `a.Input.(timerInput)` in `ContextFn`.
