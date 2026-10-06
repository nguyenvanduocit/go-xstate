> Historical translation record. Chunk tags, API-gap filenames, line numbers,
> and run counts below describe the original porting work. See the current
> [core guide](../core.md) and [architecture](../../ARCHITECTURE.md) for the maintained layout.

## Source section: clock_1

Source: `references/xstate/packages/core/test/clock.test.ts` lines 1-35.
Go file: `xstate/clock_test.go`.

Totals: 1 JS test; 1 ported, 0 N/A-type, 0 N/A-runtime, 0 skipped-in-JS.

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| clock > system clock should be default clock for actors (invoked from machine) | TestClock_SystemClockShouldBeDefaultClockForActorsInvokedFromMachine | ported | `children.child.getSnapshot().value` mapped to `machineSnap[any](snap.Children["child"]).Value` (child has no context → `MachineSnapshot[any]`). |

## API gaps

None.

## Ambiguities

- None. The test checks that the clock passed to the root actor (`createActor(machine, {clock})`, JS lines 24-26) is used by the invoked child's `after` timer (JS lines 14-16).
