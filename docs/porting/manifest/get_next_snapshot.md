> Historical translation record. Chunk tags, API-gap filenames, line numbers,
> and run counts below describe the original porting work. See the current
> [core guide](../core.md) and [architecture](../../ARCHITECTURE.md) for the maintained layout.

## Source section: get_next_snapshot_1

Source: `references/xstate/packages/core/test/getNextSnapshot.test.ts` lines 1-79.
Go file: `xstate/get_next_snapshot_test.go`.

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| getNextSnapshot > should calculate the next snapshot for transition logic | TestGetNextSnapshot_ShouldCalculateTheNextSnapshotForTransitionLogic | ported | `{count: 0}` → local `state{Count int}`; `s1.context.count` → `s1.Context.Count` |
| getNextSnapshot > should calculate the next snapshot for machine logic | TestGetNextSnapshot_ShouldCalculateTheNextSnapshotForMachineLogic | ported | |
| getNextSnapshot > should not execute actions | TestGetNextSnapshot_ShouldNotExecuteActions | ported | `vi.fn()` action → `xs.ActionFunc` calling `newSpy()`; `not.toHaveBeenCalled()` → `Count() == 0` |

## API gaps

None. Uses existing contract `xs.GetInitialSnapshot` / `xs.GetNextSnapshot` (`util.go:72-76`) and `xs.FromTransition` (`logic.go:194`).

## Ambiguities

- JS lines 21, 45, 73 call `getInitialSnapshot(logic, undefined)`. Go calls `xs.GetInitialSnapshot(logic)` with no input (variadic omitted), treating explicit `undefined` as "no input".
