> Historical translation record. Chunk tags, API-gap filenames, line numbers,
> and run counts below describe the original porting work. See the current
> [core guide](../core.md) and [architecture](../../ARCHITECTURE.md) for the maintained layout.

# predictable_exec_1

Source: `references/xstate/packages/core/test/predictableExec.test.ts` lines 1-574 (the whole file; it has 573 lines, 17 `it()` calls, no `it.each`/`it.skip`/`it.todo`).
Go file: `xstate/predictable_exec_test.go`.

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| predictableExec > should call mixed custom and builtin actions in the definitions order (L12) | TestPredictableExec_ShouldCallMixedCustomAndBuiltinActionsInTheDefinitionsOrder | ported | `context: {}` → empty struct `ctx{}` |
| predictableExec > should call initial custom actions when starting a service (L42) | TestPredictableExec_ShouldCallInitialCustomActionsWhenStartingAService | ported | |
| predictableExec > should resolve initial assign actions before starting a service (L57) | TestPredictableExec_ShouldResolveInitialAssignActionsBeforeStartingAService | ported | |
| predictableExec > should call raised transition custom actions with raised event (L72) | TestPredictableExec_ShouldCallRaisedTransitionCustomActionsWithRaisedEvent | ported | `require.NotNil` guards the `eventArg.type` deref (JS would throw TypeError) |
| predictableExec > should call raised transition builtin actions with raised event (L101) | TestPredictableExec_ShouldCallRaisedTransitionBuiltinActionsWithRaisedEvent | ported | same as above |
| predictableExec > should call invoke creator with raised event (L134) | TestPredictableExec_ShouldCallInvokeCreatorWithRaisedEvent | ported | input `({event}) => ({event})` → `NewExpr` returning `map[string]any{"event": a.Event}` |
| predictableExec > invoked child should be available on the new state (L168) | TestPredictableExec_InvokedChildShouldBeAvailableOnTheNewState | ported | `toBeDefined` → `assert.NotNil` |
| predictableExec > invoked child should not be available on the state after leaving invoking state (L193) | TestPredictableExec_InvokedChildShouldNotBeAvailableAfterLeavingInvokingState | ported | `not.toBeDefined` → `assert.Nil` on map lookup |
| predictableExec > should correctly provide intermediate context value to a custom action executed in between assign actions (L223) | TestPredictableExec_ShouldProvideIntermediateContextToCustomActionBetweenAssigns | ported | name shortened (>100 chars) |
| predictableExec > initial actions should receive context updated only by preceding assign actions (L252) | TestPredictableExec_InitialActionsShouldReceiveContextUpdatedOnlyByPrecedingAssigns | ported | the three identical push actions share one `ActionFunc` value |
| predictableExec > parent should be able to read the updated state of a child when receiving an event from it (L271) | TestPredictableExec_ParentShouldReadUpdatedStateOfChildWhenReceivingEventFromIt | ported | assertion inside `complete` observer, then `sig.Wait` |
| predictableExec > should be possible to send immediate events to initially invoked actors (L337) | TestPredictableExec_ShouldBePossibleToSendImmediateEventsToInitiallyInvokedActors | ported | |
| predictableExec > should create invoke based on context updated by entry actions of the same state (L370) | TestPredictableExec_ShouldCreateInvokeBasedOnContextUpdatedByEntryActionsOfSameState | ported | assertion inside the promise fn (goroutine; `assert`, not `require`) |
| predictableExec > should deliver events sent from the entry actions to a service invoked in the same state (L406) | TestPredictableExec_ShouldDeliverEntrySentEventsToServiceInvokedInSameState | ported | `'*'` wildcard → `On["*"]` |
| predictableExec > parent should be able to read the updated state of a child when receiving an event from it (L444, duplicate name) | TestPredictableExec_ParentShouldReadUpdatedStateOfChildWhenReceivingEventFromIt_2 | ported | JS duplicate of L271 (only guard formatting differs); `_2` suffix for uniqueness |
| predictableExec > should be possible to send immediate events to initially invoked actors (L507, duplicate name) | TestPredictableExec_ShouldBePossibleToSendImmediateEventsToInitiallyInvokedActors_2 | ported | JS duplicate of L337 (identical body); `_2` suffix |
| predictableExec > should deliver events sent from the exit actions to a service invoked in the same state (L541) | TestPredictableExec_ShouldDeliverExitSentEventsToServiceInvokedInSameState | ported | |

Totals: 17 JS tests; 17 ported, 0 N/A-type, 0 N/A-runtime, 0 skipped-in-JS.

## API gaps

None. No `apigap_predictable_exec_1.go` was created.

## Ambiguities for review

- L271 / L444 and L337 / L507: the JS file has two pairs of tests with the same names. Both copies are ported; the second copy of each gets a `_2` suffix.
- L72-L98, L101-L131, L134-L165: JS reads `eventArg.type` directly. Go adds `require.NotNil(t, eventArg)` before calling `EventType()` so a missing event fails the test instead of crashing the test binary. JS would also fail here, with a TypeError.
- L193-L220: JS `children.myChild` is `undefined` → Go `Children["myChild"]` is nil (key absent). This is asserted with `assert.Nil`.
- L292-L297 / L465-L467: the guard reads the child snapshot through `machineSnap[any](ref).Value == "b"`, so it assumes the child machine's context type is `any`.
- L370-L403: the promise input is a `map[string]any{"updated": ...}` built by `NewExpr`. `expect(input.updated).toBe(true)` → `assert.Equal(t, true, ...)` on the map value.
