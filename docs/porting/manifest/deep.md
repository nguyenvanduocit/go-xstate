> Historical translation record. Chunk tags, API-gap filenames, line numbers,
> and run counts below describe the original porting work. See the current
> [core guide](../core.md) and [architecture](../../ARCHITECTURE.md) for the maintained layout.

## Source section: deep_1

Source: `references/xstate/packages/core/test/deep.test.ts` lines 1-496 (whole file, 495 lines).
Go file: `xstate/deep_test.go`.

Totals: 9 JS tests; 9 ported, 0 N/A-type, 0 N/A-runtime, 0 skipped-in-JS.

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| deep transitions > exiting super/substates > should exit all substates when superstates exits | TestDeep_ExitingSuperSubstates_ShouldExitAllSubstatesWhenSuperstatesExits | ported |  |
| deep transitions > exiting super/substates > should exit substates and superstates when exiting (B_EVENT) | TestDeep_ExitingSuperSubstates_ShouldExitSubstatesAndSuperstatesWhenExitingBEvent | ported |  |
| deep transitions > exiting super/substates > should exit substates and superstates when exiting (C_EVENT) | TestDeep_ExitingSuperSubstates_ShouldExitSubstatesAndSuperstatesWhenExitingCEvent | ported |  |
| deep transitions > exiting super/substates > should exit superstates when exiting (D_EVENT) | TestDeep_ExitingSuperSubstates_ShouldExitSuperstatesWhenExitingDEvent | ported |  |
| deep transitions > exiting super/substates > should exit substate when machine handles event (MACHINE_EVENT) | TestDeep_ExitingSuperSubstates_ShouldExitSubstateWhenMachineHandlesEventMachineEvent | ported |  |
| deep transitions > exiting super/substates > should exit deep and enter deep (A_S) | TestDeep_ExitingSuperSubstates_ShouldExitDeepAndEnterDeepAS | ported |  |
| deep transitions > exiting super/substates > should exit deep and enter deep (D_P) | TestDeep_ExitingSuperSubstates_ShouldExitDeepAndEnterDeepDP | ported |  |
| deep transitions > exiting super/substates > should exit deep and enter deep when targeting an ancestor of the final resolved deep target | TestDeep_ExitingSuperSubstates_ShouldExitDeepAndEnterDeepWhenTargetingAncestorOfFinalTarget | ported | Go name shortened ("final resolved deep target" -> "FinalTarget") to stay under ~100 chars. |
| deep transitions > exiting super/substates > should exit deep and enter deep when targeting a deep state | TestDeep_ExitingSuperSubstates_ShouldExitDeepAndEnterDeepWhenTargetingDeepState | ported |  |

## API gaps

None. No `apigap_deep_1.go` file was created.

## Notes for reviewers

- `trackEntries` (test/utils.ts:76-111) is copied as the file-local helper `deep1TrackEntries`, using the same approach as `actions1TrackEntries` in `xstate/action_test.go`. It prepends tracking actions to `StateNode.Entry`/`Exit` and walks `ChildStates()` in document order. The JS helper also throws when it gets the same machine twice; the Go copy leaves that check out because none of these tests pass a machine twice.
- JS `on: { EV: '#root.X' }` maps to `On: {"EV": {{Target: "#root.X"}}}`. The order of `states` keys follows the JS source (for example `DONE`, `FAIL`, `A` at JS lines 10-12).
