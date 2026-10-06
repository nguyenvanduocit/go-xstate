> Historical translation record. Chunk tags, API-gap filenames, line numbers,
> and run counts below describe the original porting work. See the current
> [core guide](../core.md) and [architecture](../../ARCHITECTURE.md) for the maintained layout.

# Port manifest: graph

JS sources: `references/xstate/packages/core/src/graph/test/*.test.ts` (10 files) and `references/xstate/packages/core/src/graph/types.test.ts`.
Go package: `graph/` (stubs in `types.go`, `graph.go`, `test_model.go`; every body panics `graph: not implemented`).
Go tests: `graph/*_test.go`, `package graph_test`, build tag `port_graph`. Shared helpers: `graph/helpers_test.go` (mirrors `test/testUtils.ts` and `getPathsSnapshot`).

Totals: 85 JS tests; 71 ported, 11 N/A-type, 0 N/A-runtime, 3 skipped-in-JS.

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| test/adjacency.test.ts: adjacency maps > model generates an adjacency map (converted to an array) | `TestAdjacency_ModelGeneratesAnAdjacencyMapConvertedToAnArray` | ported | Order re-derived with sorted own events (JS: `on` insertion order). Expected list computed by running the reference graph code with `StateNode.ownEvents` sorted. |
| test/adjacency.test.ts: adjacency maps > function generates an adjacency map (converted to an array) | `TestAdjacency_FunctionGeneratesAnAdjacencyMapConvertedToAnArray` | ported |  |
| test/events.test.ts: events > should execute events (`exec` property) | `TestEvents_ShouldExecuteEventsExecProperty` | ported | Async executors → sync funcs returning error. |
| test/events.test.ts: events > should execute events (function) | `TestEvents_ShouldExecuteEventsFunction` | ported |  |
| test/forbiddenAttributes.test.ts: Forbidden attributes > Should not let you declare invocations on your test machine | `TestForbiddenAttributes_ShouldNotLetYouDeclareInvocationsOnYourTestMachine` | ported |  |
| test/forbiddenAttributes.test.ts: Forbidden attributes > Should not let you declare after on your test machine | `TestForbiddenAttributes_ShouldNotLetYouDeclareAfterOnYourTestMachine` | ported |  |
| test/forbiddenAttributes.test.ts: Forbidden attributes > Should not let you delayed actions on your machine | `TestForbiddenAttributes_ShouldNotLetYouDelayedActionsOnYourMachine` | ported | Implementation needs gap `xs.ActionDelay`. |
| test/states.test.ts: states > should test states by key | `TestStates_ShouldTestStatesByKey` | ported |  |
| test/states.test.ts: states > should test states by ID | `TestStates_ShouldTestStatesByID` | ported |  |
| test/testModel.test.ts: custom test models > tests any logic | `TestTestModel_TestsAnyLogic` | ported |  |
| test/testModel.test.ts: custom test models > tests states for any logic | `TestTestModel_TestsStatesForAnyLogic` | ported |  |
| test/graph.test.ts: @xstate/graph > getStateNodes() > should return an array of all nodes | `TestGraph_GetStateNodes_ShouldReturnAnArrayOfAllNodes` | ported | `getStateNodes(machine)` → `GetStateNodes(machine.Root)`; `instanceof StateNode` → `assert.IsType`. |
| test/graph.test.ts: @xstate/graph > getStateNodes() > should return an array of all nodes (parallel) | `TestGraph_GetStateNodes_ShouldReturnAnArrayOfAllNodesParallel` | ported | Same as above. |
| test/graph.test.ts: @xstate/graph > getShortestPaths() > should return a mapping of shortest paths to all states | `TestGraph_GetShortestPaths_ShouldReturnAMappingOfShortestPathsToAllStates` | ported | Snapshot `shortest paths 1` inlined; path order re-derived with sorted own events (POWER_OUTAGE before TIMER). |
| test/graph.test.ts: @xstate/graph > getShortestPaths() > should return a mapping of shortest paths to all states (parallel) | `TestGraph_GetShortestPaths_ShouldReturnAMappingOfShortestPathsToAllStatesParallel` | ported | Snapshot `shortest paths parallel 1` inlined (unchanged: numeric keys already sorted in JS). |
| test/graph.test.ts: @xstate/graph > getShortestPaths() > the initial state should have a single-length path | `TestGraph_GetShortestPaths_TheInitialStateShouldHaveASingleLengthPath` | ported |  |
| test/graph.test.ts: @xstate/graph > getShortestPaths() > should not throw when a condition is present | `TestGraph_GetShortestPaths_ShouldNotThrowWhenAConditionIsPresent` | skipped-in-JS | `it.skip` in JS. |
| test/graph.test.ts: @xstate/graph > getShortestPaths() > should represent conditional paths based on context | `TestGraph_GetShortestPaths_ShouldRepresentConditionalPathsBasedOnContext` | skipped-in-JS | `it.skip` in JS. |
| test/graph.test.ts: @xstate/graph > getSimplePaths() > should return a mapping of arrays of simple paths to all states | `TestGraph_GetSimplePaths_ShouldReturnAMappingOfArraysOfSimplePathsToAllStates` | ported | Inline snapshot + snapshot `...all states 2` inlined; order re-derived with sorted own events. |
| test/graph.test.ts: @xstate/graph > getSimplePaths() > should return a mapping of simple paths to all states (parallel) | `TestGraph_GetSimplePaths_ShouldReturnAMappingOfSimplePathsToAllStatesParallel` | ported | Snapshot `simple paths parallel 1` inlined. |
| test/graph.test.ts: @xstate/graph > getSimplePaths() > should return multiple paths for equivalent transitions | `TestGraph_GetSimplePaths_ShouldReturnMultiplePathsForEquivalentTransitions` | ported | Snapshot `simple paths equal transitions 1` inlined; BAR path precedes FOO (sorted own events). |
| test/graph.test.ts: @xstate/graph > getSimplePaths() > should return a single-length path for the initial state | `TestGraph_GetSimplePaths_ShouldReturnASingleLengthPathForTheInitialState` | ported |  |
| test/graph.test.ts: @xstate/graph > getSimplePaths() > should return value-based paths | `TestGraph_GetSimplePaths_ShouldReturnValueBasedPaths` | ported | Snapshot `simple paths context 1` inlined. |
| test/graph.test.ts: @xstate/graph > getSimplePaths() > should support filtering disabled events | `TestGraph_GetSimplePaths_ShouldSupportFilteringDisabledEvents` | ported |  |
| test/graph.test.ts: @xstate/graph > getPathFromEvents() > should return a path to the last entered state by the event sequence | `TestGraph_GetPathFromEvents_ShouldReturnAPathToTheLastEnteredStateByTheEventSequence` | ported | Snapshot `path from events 1` inlined. |
| test/graph.test.ts: @xstate/graph > getPathFromEvents() > should throw when an invalid event sequence is provided | `TestGraph_GetPathFromEvents_ShouldThrowWhenAnInvalidEventSequenceIsProvided` | skipped-in-JS | `it.skip` in JS. |
| test/graph.test.ts: @xstate/graph > getPathFromEvents() > should return a path from a specified from-state | `TestGraph_GetPathFromEvents_ShouldReturnAPathFromASpecifiedFromState` | ported |  |
| test/graph.test.ts: @xstate/graph > toDirectedGraph > should represent a statechart as a directed graph | `TestGraph_ToDirectedGraph_ShouldRepresentAStatechartAsADirectedGraph` | ported | Snapshot inlined as `DirectedGraphNodeJSON` (pretty-format calls toJSON()). `toDirectedGraph(machine)` → `ToDirectedGraph(machine.Root)`. |
| test/graph.test.ts: simple paths for transition functions | `TestGraph_SimplePathsForTransitionFunctions` | ported | Snapshot `simple paths for transition functions 1` inlined (JS test calls getShortestPaths). |
| test/graph.test.ts: shortest paths for transition functions | `TestGraph_ShortestPathsForTransitionFunctions` | ported | Snapshot `shortest paths for transition functions 1` inlined (JS test calls getSimplePaths). |
| test/graph.test.ts: filtering > should not traverse past filtered states | `TestGraph_Filtering_ShouldNotTraversePastFilteredStates` | ported |  |
| test/graph.test.ts: should provide previous state for serializeState() | `TestGraph_ShouldProvidePreviousStateForSerializeState` | ported |  |
| test/graph.test.ts: from-state can be specified (it.each([getShortestPaths, getSimplePaths])) | `TestGraph_FromStateCanBeSpecified` | ported | `it.each([getShortestPaths, getSimplePaths])` → one Go test with 2 subtests. |
| test/graph.test.ts: joinPaths() > should join two paths | `TestGraph_JoinPaths_ShouldJoinTwoPaths` | ported |  |
| test/graph.test.ts: joinPaths() > should not join two paths with mismatched source/target states | `TestGraph_JoinPaths_ShouldNotJoinTwoPathsWithMismatchedSourceTargetStates` | ported |  |
| test/index.test.ts: events > should allow for representing many cases | `TestIndex_Events_ShouldAllowForRepresentingManyCases` | ported |  |
| test/index.test.ts: events > should not throw an error for unimplemented events | `TestIndex_Events_ShouldNotThrowAnErrorForUnimplementedEvents` | ported | JS `expect(async fn).not.toThrow()` only checks sync throws; Go runs paths synchronously under `assert.NotPanics` and also requires no path error. |
| test/index.test.ts: events > should allow for dynamic generation of cases based on state | `TestIndex_Events_ShouldAllowForDynamicGenerationOfCasesBasedOnState` | ported |  |
| test/index.test.ts: state limiting > should limit states with filter option | `TestIndex_StateLimiting_ShouldLimitStatesWithFilterOption` | ported |  |
| test/index.test.ts: prevents infinite recursion based on a provided limit | `TestIndex_PreventsInfiniteRecursionBasedOnAProvidedLimit` | ported | `toThrowErrorMatchingInlineSnapshot` → `assert.PanicsWithError("Traversal limit exceeded")`. |
| test/index.test.ts: test model options > options.testState(...) should test state | `TestIndex_TestModelOptions_OptionsTestStateShouldTestState` | ported |  |
| test/index.test.ts: tests transitions | `TestIndex_TestsTransitions` | ported | `expect.assertions(2)` → executor call count asserted == 1 (2 assertions each). |
| test/index.test.ts: Event in event executor should contain payload from case | `TestIndex_EventInEventExecutorShouldContainPayloadFromCase` | ported | `toEqual` with a function value: compared field by field, function by pointer identity (reflect.DeepEqual cannot compare funcs). Default serializeEvent must tolerate non-JSON values (JS JSON.stringify drops functions). |
| test/index.test.ts: state tests > should test states | `TestIndex_StateTests_ShouldTestStates` | ported | `expect.assertions(2)` → callback count asserted == 2. |
| test/index.test.ts: state tests > should test wildcard state for non-matching states | `TestIndex_StateTests_ShouldTestWildcardStateForNonMatchingStates` | ported | `expect.assertions(4)` → callback count asserted == 4. |
| test/index.test.ts: state tests > should test nested states | `TestIndex_StateTests_ShouldTestNestedStates` | ported | State test keys iterate sorted in Go (`a`, `b`, `b.b1`), equal to JS key order here. |
| test/index.test.ts: state tests > should test with input | `TestIndex_StateTests_ShouldTestWithInput` | ported |  |
| test/paths.test.ts: testModel.testPaths(...) > custom path generators can be provided | `TestPaths_TestPaths_CustomPathGeneratorsCanBeProvided` | ported |  |
| test/paths.test.ts: testModel.testPaths(...) > When the machine only has one path > Should only follow that path | `TestPaths_TestPaths_WhenTheMachineOnlyHasOnePath_ShouldOnlyFollowThatPath` | ported |  |
| test/paths.test.ts: testModel.testPaths(...) > getSimplePaths > Should dedup simple path paths | `TestPaths_TestPaths_GetSimplePaths_ShouldDedupSimplePathPaths` | ported |  |
| test/paths.test.ts: testModel.testPaths(...) > getSimplePaths > Should not dedup simple path paths if deduplicate: false | `TestPaths_TestPaths_GetSimplePaths_ShouldNotDedupSimplePathPathsIfDeduplicateFalse` | ported |  |
| test/paths.test.ts: testModel.testPaths(...) > getSimplePaths > should support filtering disabled events | `TestPaths_TestPaths_GetSimplePaths_ShouldSupportFilteringDisabledEvents` | ported |  |
| test/paths.test.ts: path.description > Should write a readable description including the target state and the path | `TestPaths_PathDescription_ShouldWriteAReadableDescriptionIncludingTheTargetStateAndThePath` | ported |  |
| test/paths.test.ts: transition coverage > path generation should cover all transitions by default | `TestPaths_TransitionCoverage_PathGenerationShouldCoverAllTransitionsByDefault` | ported | Expectation re-derived with sorted own events: JS `NEXT → PREV`, `NEXT → RESTART`, `END` becomes `END → PREV`, `END → RESTART`, `NEXT`. |
| test/paths.test.ts: transition coverage > transition coverage should consider guarded transitions | `TestPaths_TransitionCoverage_TransitionCoverageShouldConsiderGuardedTransitions` | ported |  |
| test/paths.test.ts: transition coverage > transition coverage should consider multiple transitions with the same target | `TestPaths_TransitionCoverage_TransitionCoverageShouldConsiderMultipleTransitionsWithTheSameTarget` | ported |  |
| test/paths.test.ts: getShortestPathsTo > Should find a path to a non-initial target state | `TestPaths_GetShortestPathsTo_ShouldFindAPathToANonInitialTargetState` | ported |  |
| test/paths.test.ts: getShortestPathsTo > Should find a path to an initial target state | `TestPaths_GetShortestPathsTo_ShouldFindAPathToAnInitialTargetState` | ported |  |
| test/paths.test.ts: getShortestPathsFrom > should get shortest paths from array of paths | `TestPaths_GetShortestPathsFrom_ShouldGetShortestPathsFromArrayOfPaths` | ported |  |
| test/paths.test.ts: getShortestPathsFrom > getSimplePathsFrom > should get simple paths from array of paths | `TestPaths_GetShortestPathsFrom_GetSimplePathsFrom_ShouldGetSimplePathsFromArrayOfPaths` | ported |  |
| test/shortestPaths.test.ts: getShortestPaths > finds the shortest paths to a state without continuing traversal from that state | `TestShortestPaths_FindsTheShortestPathsToAStateWithoutContinuingTraversalFromThatState` | ported |  |
| test/shortestPaths.test.ts: getShortestPaths > finds the shortest paths from a state to another state | `TestShortestPaths_FindsTheShortestPathsFromAStateToAnotherState` | ported |  |
| test/shortestPaths.test.ts: getShortestPaths > handles event cases | `TestShortestPaths_HandlesEventCases` | ported | JS `expect(filtered).toBeDefined()` is vacuous; Go asserts the filtered slice is non-nil (reference run yields 8 matches). |
| test/shortestPaths.test.ts: getShortestPaths > should work for machines with delays | `TestShortestPaths_ShouldWorkForMachinesWithDelays` | ported |  |
| test/dieHard.test.ts: die hard example > testing a model (shortestPathsTo) > should generate the right number of paths | `TestDieHard_ShortestPathsTo_ShouldGenerateTheRightNumberOfPaths` | ported |  |
| test/dieHard.test.ts: die hard example > testing a model (shortestPathsTo) > path ${getDescription(path.state)} > path ${getDescription(path.state)} | `TestDieHard_ShortestPathsTo_Path` | ported | Dynamic `it` per generated path → one subtest per path (fresh jugs per subtest, mirrors beforeEach). |
| test/dieHard.test.ts: die hard example > testing a model (simplePathsTo) > should generate the right number of paths | `TestDieHard_SimplePathsTo_ShouldGenerateTheRightNumberOfPaths` | ported |  |
| test/dieHard.test.ts: die hard example > testing a model (simplePathsTo) > reaches state ${value} (${context}) > path ${getDescription(path.state)} | `TestDieHard_SimplePathsTo_ReachesStatePath` | ported | Dynamic `it` per generated path → subtests. |
| test/dieHard.test.ts: die hard example > testing a model (getPathFromEvents) > reaches state ${value} (${context}) > path ${getDescription(path.state)} | `TestDieHard_GetPathFromEvents_ReachesStatePath` | ported | Single generated `it` → subtest. |
| test/dieHard.test.ts: die hard example > testing a model (getPathFromEvents) > should return no paths if the target does not match the last entered state | `TestDieHard_GetPathFromEvents_ShouldReturnNoPathsIfTheTargetDoesNotMatchTheLastEnteredState` | ported |  |
| test/dieHard.test.ts: die hard example > .testPath(path) > should generate the right number of paths | `TestDieHard_TestPath_ShouldGenerateTheRightNumberOfPaths` | ported |  |
| test/dieHard.test.ts: die hard example > .testPath(path) > reaches state ${value} (${context}) > path ${getDescription(path.state)} > reaches the target state | `TestDieHard_TestPath_ReachesTheTargetState` | ported | Dynamic `it` per generated path → subtests. |
| test/dieHard.test.ts: error path trace > should return trace for failed state > should generate the right number of paths | `TestDieHard_ErrorPathTrace_ShouldGenerateTheRightNumberOfPaths` | ported |  |
| test/dieHard.test.ts: error path trace > should return trace for failed state > should show an error path trace | `TestDieHard_ErrorPathTrace_ShouldShowAnErrorPathTrace` | ported | Inline snapshot of err.message inlined verbatim (verified against reference run). JS throw in state test → state func returns error; TestPath returns the annotated error. |
| types.test.ts: getShortestPath types > `getEvents` should be allowed to return a mutable array | `TestTypes_GetShortestPathTypes_GetEventsShouldBeAllowedToReturnAMutableArray` | N/A-type | checked that a mutable event array type-checks as `events` |
| types.test.ts: getShortestPath types > `getEvents` should be allowed to return a readonly array | `TestTypes_GetShortestPathTypes_GetEventsShouldBeAllowedToReturnAReadonlyArray` | N/A-type | checked that a readonly event array type-checks as `events` |
| types.test.ts: getShortestPath types > `events` should allow known event | `TestTypes_GetShortestPathTypes_EventsShouldAllowKnownEvent` | N/A-type | checked that a known event with payload type-checks in `events` |
| types.test.ts: getShortestPath types > `events` should not require all event types (array literal expression) | `TestTypes_GetShortestPathTypes_EventsShouldNotRequireAllEventTypesArrayLiteral` | N/A-type | checked that `events` may list a subset of event types (array literal) |
| types.test.ts: getShortestPath types > `events` should not require all event types (tuple) | `TestTypes_GetShortestPathTypes_EventsShouldNotRequireAllEventTypesTuple` | N/A-type | checked that `events` may list a subset of event types (readonly tuple) |
| types.test.ts: getShortestPath types > `events` should not require all event types (function) | `TestTypes_GetShortestPathTypes_EventsShouldNotRequireAllEventTypesFunction` | N/A-type | checked that an `events` function may return a subset of event types |
| types.test.ts: getShortestPath types > `events` should not allow unknown events | `TestTypes_GetShortestPathTypes_EventsShouldNotAllowUnknownEvents` | N/A-type | @ts-expect-error on an unknown event type in `events` |
| types.test.ts: getShortestPath types > `events` should only allow props of a specific event | `TestTypes_GetShortestPathTypes_EventsShouldOnlyAllowPropsOfASpecificEvent` | N/A-type | @ts-expect-error on a prop belonging to another event type |
| types.test.ts: getShortestPath types > `serializeEvent` should be allowed to return plain string | `TestTypes_GetShortestPathTypes_SerializeEventShouldBeAllowedToReturnPlainString` | N/A-type | checked that serializeEvent may return a plain (unbranded) string |
| types.test.ts: getShortestPath types > `serializeState` should be allowed to return plain string | `TestTypes_GetShortestPathTypes_SerializeStateShouldBeAllowedToReturnPlainString` | N/A-type | checked that serializeState may return a plain (unbranded) string |
| types.test.ts: createTestModel types > `EventExecutor` should be passed event with type that corresponds to its key | `TestTypes_CreateTestModelTypes_EventExecutorShouldBePassedEventWithTypeOfItsKey` | N/A-type | @ts-expect-error narrowing of event.type per EventExecutor key |

## API gaps added

- `apigap_graph.go` (package `xstate`, tag `port_graph`): `func ActionDelay(action Action) (delay any, ok bool)`. It mirrors the JS check `typeof action.delay === 'number'` in `graph/validateMachine.ts`. `createTestModel` needs it to reject delayed actions (`TestForbiddenAttributes_ShouldNotLetYouDelayedActionsOnYourMachine`). No other core gap: traversal uses `xs.GetInitialSnapshot` / `xs.GetNextSnapshot`. Event descriptors come from `snapshot.StateNodes()` + `StateNode.OwnEvents()`, and descriptions from `GetMeta()` / `Matches` / `Can`.

## Graph package API (new, in `graph/`)

- Generic over the snapshot type `S xs.Snapshot` (`*xs.MachineSnapshot[C]`, `*xs.TransitionSnapshot[T]`, ...); logic is `xs.TypedActorLogic[S]`.
- `TraversalOptions[S]`: JS `events` (array | function) → `Events` / `EventsFn` (fn wins). `limit: Infinity` → `Limit: 0`. `fromState` absent → zero (nil) `FromState`. `serializeState(state, undefined, undefined)` → `event == nil`, `prevState == nil`.
- `AdjacencyMap` / `AdjacencyValue` keep JS object insertion order explicitly (`Keys`, `EventKeys`), because `adjacencyMapToArray` and all path generators iterate in that order.
- `TestParam.States` / `.Events` are maps: state test keys run in sorted order (JS: object key order). Executors and state tests return `error` (JS: throw / rejected promise). `TestModel.TestPath` returns `(TestPathResult, error)`, and the error message gets the formatted path trace appended.
- `getStateNodes(machine)` / `toDirectedGraph(machine)` → `GetStateNodes(machine.Root)` / `ToDirectedGraph(machine.Root)`.
- `toJSON()` of the directed graph → typed `DirectedGraphNodeJSON` / `DirectedGraphEdgeJSON`, with `Children` / `Edges` never nil (JSON `[]`).
- `ValidateMachine` and `CreateShortestPathsGen` / `CreateSimplePathsGen` are exported too (`validateMachine` is internal in JS).

## Ambiguities for the reviewer

1. **Event order (the main deviation).** JS visits each state's events in `on` key insertion order (`__unsafe_getAllOwnEventDescriptors` → `stateNode.ownEvents`). docs/porting/core.md says the Go API returns own events sorted. This changes the *order* (and for TestModel shortest paths, the *chosen* path) in 5 tests. Their expectations were re-derived, not hand-edited: the reference TS sources were copied to a scratch dir, `StateNode.ownEvents` was patched to `Array.from(events).sort()`, and the test scenarios ran under bun. Unpatched, the same harness reproduced every original JS inline/file snapshot, which confirms the harness is faithful. Affected tests: `TestAdjacency_ModelGeneratesAnAdjacencyMapConvertedToAnArray`, `TestGraph_GetShortestPaths_ShouldReturnAMappingOfShortestPathsToAllStates`, `TestGraph_GetSimplePaths_ShouldReturnAMappingOfArraysOfSimplePathsToAllStates`, `TestGraph_GetSimplePaths_ShouldReturnMultiplePathsForEquivalentTransitions`, `TestPaths_TransitionCoverage_PathGenerationShouldCoverAllTransitionsByDefault`. The Go impl is assumed to build the all-own-events list by concatenating each active node's sorted `OwnEvents()` in `StateNodes()` order, deduplicated. Any other order changes these 5 expectations.
2. Every other order-sensitive expectation (die hard counts and descriptions, dynamic cases, state-test order, parallel `"1"/"2"/"3"` keys, which JS already orders numerically) is identical under both orders. This was verified with the same harness.
3. JSON: `serializeSnapshot` / descriptions use `JSON.stringify`. Context structs carry `json:"..."` tags in JS key order (`{"allowed":true}`, `{"three":..,"five":..}`). `xs.E` payloads marshal with sorted keys, which matches JS for the single-payload-key cases asserted (`{"value":0}`). The default `serializeEvent` must not fail on func-valued payloads (JS drops them; see `TestIndex_EventInEventExecutorShouldContainPayloadFromCase`).
4. die hard: JS generates one `it` per path at collection time. Go uses one test func per source `it`, with `t.Run` per path named by `path.Description` (JS names come from internal `getDescription`, which is not exported).
5. `types.test.ts`: all 11 tests are `types: {} as ...` / `@ts-expect-error` checks with no runtime expectation → N/A-type per docs/porting/core.md.
