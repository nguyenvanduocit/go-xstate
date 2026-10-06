> Historical translation record. Chunk tags, API-gap filenames, line numbers,
> and run counts below describe the original porting work. See the current
> [core guide](../core.md) and [architecture](../../ARCHITECTURE.md) for the maintained layout.

## Source section: issue5454_1

Source: `references/xstate/packages/core/test/issue5454.test.ts` lines 1-124.
Go file: `xstate/issue5454_test.go`.

Totals: 5 JS tests; 5 ported, 0 N/A-type, 0 N/A-runtime, 0 skipped-in-JS.

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| initialTransition / transition with invoke systemId (issue #5454) > does not throw when the initial state has an invoke with systemId | TestIssue5454_DoesNotThrowWhenInitialStateHasInvokeWithSystemID | ported | `not.toThrow()` → `assert.NotPanics` |
| initialTransition / transition with invoke systemId (issue #5454) > returns the correct initial snapshot when invoke has systemId | TestIssue5454_ReturnsCorrectInitialSnapshotWhenInvokeHasSystemID | ported | `toHaveLength(1)` → `assert.Len(actions, 1)` |
| initialTransition / transition with invoke systemId (issue #5454) > is idempotent: repeated calls do not throw | TestIssue5454_IsIdempotentRepeatedCallsDoNotThrow | ported |  |
| initialTransition / transition with invoke systemId (issue #5454) > transition() does not throw when the target state has an invoke with systemId | TestIssue5454_TransitionDoesNotThrowWhenTargetStateHasInvokeWithSystemID | ported | `fromTransition(reducer, 0)` → `xs.FromTransition(reducer, func(TransitionInitArgs) int { return 0 })` |
| initialTransition / transition with invoke systemId (issue #5454) > works with multiple invokes each having a distinct systemId | TestIssue5454_WorksWithMultipleInvokesEachHavingDistinctSystemID | ported |  |

## API gaps

None. All APIs used exist in the contract (`InvokeConfig.SystemID`, `InitialTransition`, `Transition`, `FromPromise`, `FromTransition`).

## Ambiguities

- JS line 51: `expect(actions).toHaveLength(1)` relies on `initialTransition` returning exactly one executable action (the `spawnChild` for the invoke) and no entry actions; the Go `[]ExecutableAction` must likewise contain exactly one element.
- Test func prefix is `TestIssue5454_` (area name); no file-local helpers were needed, so the `issue54541` identifier prefix was not used.
