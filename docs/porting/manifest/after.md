> Historical translation record. Chunk tags, API-gap filenames, line numbers,
> and run counts below describe the original porting work. See the current
> [core guide](../core.md) and [architecture](../../ARCHITECTURE.md) for the maintained layout.

## Source section: after_1

Source: `references/xstate/packages/core/test/after.test.ts` lines 1-361 (whole file, 10 `it` calls incl. 1 `it.skip`).
Go file: `xstate/after_test.go`.

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| delayed transitions > should transition after delay | TestAfter_DelayedTransitions_ShouldTransitionAfterDelay | ported | JS L34; fake timers -> SimulatedClock |
| delayed transitions > should not try to clear an undefined timeout when exiting source state of a delayed transition | TestAfter_DelayedTransitions_ShouldNotClearUndefinedTimeoutWhenExitingSourceStateOfDelayedTransition | ported | JS L47; custom clock `after1SpyClock` (real `time.AfterFunc` + spy ClearTimeout); name shortened |
| delayed transitions > should format transitions properly | TestAfter_DelayedTransitions_ShouldFormatTransitionsProperly | ported | JS L76; `transitions.keys()` -> keys of `StateNode.On()` (JS `on` is derived from `transitions`, StateNode.ts:362) |
| delayed transitions > should be able to transition with delay from nested initial state | TestAfter_DelayedTransitions_ShouldTransitionWithDelayFromNestedInitialState | ported | JS L88 |
| delayed transitions > parent state should enter child state without re-entering self (relative target) | TestAfter_DelayedTransitions_ParentStateShouldEnterChildStateWithoutReenteringSelfRelativeTarget | ported | JS L122; assertion made on test goroutine with value captured inside `complete` |
| delayed transitions > should defer a single send event for a delayed conditional transition (#886) | TestAfter_DelayedTransitions_ShouldDeferSingleSendEventForDelayedConditionalTransition886 | ported | JS L165 |
| delayed transitions > should execute an after transition after starting from a state resolved using `.getPersistedSnapshot` | TestAfter_DelayedTransitions_ShouldExecuteAfterTransitionAfterStartingFromGetPersistedSnapshot | skipped-in-JS | JS L202 `it.skip`; body translated after `t.Skip` |
| delayed transitions > should execute an after transition after starting from a persisted state | TestAfter_DelayedTransitions_ShouldExecuteAfterTransitionAfterStartingFromPersistedState | ported | JS L236; `JSON.parse(JSON.stringify(snap))` -> json round-trip of `snap.ToJSON()` |
| delayed transitions > delay expressions > should evaluate the expression (function) to determine the delay | TestAfter_DelayExpressions_ShouldEvaluateExpressionFunctionToDetermineDelay | ported | JS L274; `toBeCalledWith` -> `assert.Contains(s.Calls(), []any{context})` |
| delayed transitions > delay expressions > should evaluate the expression (string) to determine the delay | TestAfter_DelayExpressions_ShouldEvaluateExpressionStringToDetermineDelay | ported | JS L313; `toBeCalledWith` -> `assert.Contains(s.Calls(), []any{event})` |

## API gaps

None. No `apigap_after_1.go` created.

## Ambiguities for review

- JS L4-27 top-level `lightMachine` is a constructor `after1LightMachine()` (a package-level var would call the panicking `CreateMachine` at init). Each test builds its own instance; the JS tests do not depend on shared identity.
- JS L76 "should format transitions properly": the contract has no `StateNode.transitions`; `On()` has the same key set. If the implementation's `On()` omits `xstate.after.*` descriptors, a `Transitions()` accessor gap would be needed.
- JS L47 uses real timers (1ms after + `await sleep(5)`); Go keeps the same timings, so it carries the same small scheduling margin.
- JS L236: `service.subscribe({complete})` is registered after `send(NEXT)`, like JS. The 1ms real timer makes it very unlikely to complete before subscribe, but Go timers run on another goroutine, unlike the JS event loop.
- JS L165: `actions: spy` receives action args; Go spy records `a` but only `Count()` is asserted, matching `not.toHaveBeenCalled()`.
