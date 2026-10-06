> Historical translation record. Chunk tags, API-gap filenames, line numbers,
> and run counts below describe the original porting work. See the current
> [core guide](../core.md) and [architecture](../../ARCHITECTURE.md) for the maintained layout.

## Source section: meta_1

Source: `references/xstate/packages/core/test/meta.test.ts` lines 1-674 (whole file, 673 lines).
Go file: `xstate/meta_test.go`. Gap file: `apigap_meta_1.go`.

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| state meta data > states should aggregate meta data | TestMeta_StateMetaData_StatesShouldAggregateMetaData | ported | JS L75. Uses shared `meta1LightMachine()` (JS L4-73) |
| state meta data > states should aggregate meta data (deep) | TestMeta_StateMetaData_StatesShouldAggregateMetaDataDeep | ported | JS L89 |
| state meta data > services started from a persisted state should calculate meta data | TestMeta_StateMetaData_ServicesStartedFromPersistedStateShouldCalculateMetaData | ported | JS L109. `machine.resolveState({ value: 'second' })` → `ResolveState(ResolveStateConfig[any]{Value: "second"})` |
| state meta data > meta keys are strongly-typed | TestMeta_StateMetaData_MetaKeysAreStronglyTyped | N/A-type | JS L139. Only `satisfies` / `@ts-expect-error`; no `expect` |
| state meta data > TS should error with unexpected meta property | TestMeta_StateMetaData_TSShouldErrorWithUnexpectedMetaProperty | N/A-type | JS L186. Only `@ts-expect-error` |
| state meta data > TS should error with wrong meta value type | TestMeta_StateMetaData_TSShouldErrorWithWrongMetaValueType | N/A-type | JS L211. Only `@ts-expect-error` |
| state meta data > should allow states to omit meta | TestMeta_StateMetaData_ShouldAllowStatesToOmitMeta | N/A-type | JS L236. Compile-only check; no `expect` |
| state meta data > TS should error with unexpected transition meta property | TestMeta_StateMetaData_TSShouldErrorWithUnexpectedTransitionMetaProperty | N/A-type | JS L256. Only `@ts-expect-error` |
| state meta data > TS should error with wrong transition meta value type | TestMeta_StateMetaData_TSShouldErrorWithWrongTransitionMetaValueType | N/A-type | JS L280. Only `@ts-expect-error` |
| state meta data > should support typing meta properties (no ts-expected errors) | TestMeta_StateMetaData_ShouldSupportTypingMetaPropertiesNoTSExpectedErrors | N/A-type | JS L304. Only `satisfies`; no `expect` |
| state meta data > should strongly type the state IDs in snapshot.getMeta() | TestMeta_StateMetaData_ShouldStronglyTypeTheStateIDsInGetMeta | N/A-type | JS L344. Only `@ts-expect-error` |
| state meta data > should strongly type the state IDs in snapshot.getMeta() (no root ID) | TestMeta_StateMetaData_ShouldStronglyTypeTheStateIDsInGetMetaNoRootID | N/A-type | JS L381. Only `@ts-expect-error` |
| transition meta data > infers distinct metadata types with createMachine | TestMeta_TransitionMetaData_InfersDistinctMetadataTypesWithCreateMachine | N/A-type | JS L420. Only `satisfies` |
| transition meta data > supports distinct state and transition meta types | TestMeta_TransitionMetaData_SupportsDistinctStateAndTransitionMetaTypes | N/A-type | JS L438. Only `satisfies` / `@ts-expect-error` |
| transition meta data > rejects state and transition metadata in the wrong positions | TestMeta_TransitionMetaData_RejectsStateAndTransitionMetadataInTheWrongPositions | N/A-type | JS L477. Only `@ts-expect-error` |
| transition meta data > keeps types.meta as the shared metadata type for compatibility | TestMeta_TransitionMetaData_KeepsTypesMetaAsTheSharedMetadataTypeForCompatibility | N/A-type | JS L500. Only `satisfies` |
| transition meta data > preserves transition meta on all transition definitions | TestMeta_TransitionMetaData_PreservesTransitionMetaOnAllTransitionDefinitions | ported | JS L518. All 6 runtime `expect`s ported; type-level `satisfies` / `@ts-expect-error` dropped. Uses gap `WithMachineInitialMeta` |
| transition meta data > TS should error with unexpected transition meta property | TestMeta_TransitionMetaData_TSShouldErrorWithUnexpectedTransitionMetaProperty | N/A-type | JS L597. Only `@ts-expect-error` (duplicate of L256 under another describe) |
| transition meta data > TS should error with wrong transition meta value type | TestMeta_TransitionMetaData_TSShouldErrorWithWrongTransitionMetaValueType | N/A-type | JS L621. Only `@ts-expect-error` (duplicate of L280 under another describe) |
| state description > state node should have its description | TestMeta_StateDescription_StateNodeShouldHaveItsDescription | ported | JS L647 |
| transition description > state node should have its description | TestMeta_TransitionDescription_StateNodeShouldHaveItsDescription | ported | JS L662. `machine.root.on['EVENT'][0]` → `Root.On()["EVENT"][0]` |

Totals: 21 JS tests; 6 ported, 15 N/A-type, 0 N/A-runtime, 0 skipped-in-JS.

## API gaps (`apigap_meta_1.go`)

- `WithMachineInitialMeta[C any](cfg MachineConfig[C], meta any) MachineConfig[C]` — mirrors a root `initial: { target, meta }` initial-transition object (JS L527-530). `MachineConfig.Initial` is a string, so the initial transition's meta has no field. Same shape as the existing `WithMachineInitialActions` gap (`apigap_actions_2.go`); at integration both could merge into one initial-transition field on MachineConfig.

## Ambiguities for review

- JS L4-73: describe-level `pedestrianStates` / `lightMachine` are built by file-local funcs `meta1PedestrianStates()` / `meta1LightMachine()` (a fresh machine per test; JS shares one instance, which is stateless, so behaviour is the same). The `...pedestrianStates` spread copies `Initial` and `States` into the `red` StateConfig.
- JS L39-56: entry/exit actions are strings (`'enter_green'`, …) with no implementation; translated to `xs.ActionRef{Type: ...}` with no provided implementations, as in JS (unresolved named actions are no-ops in v5).
- JS L83-84: `'light.green' in getMeta()` falsy → `assert.NotContains(t, meta, "light.green")` (map key check).
- JS L578-581: `machine.definition.initial?.meta` assumes `Definition()["initial"]` is a JSON-like `map[string]any`; the test `require`s that type. `JSON.parse(JSON.stringify(machine))` → `json.Marshal(machine.ToJSON())` + `json.Unmarshal` into `map[string]any`.
- JS L584-595: `expect.arrayContaining([...])` over `[...idle.transitions.values()].flat().map(t => t.meta)` → `assert.Subset` over the metas of all `idle.On()` transitions. This assumes Go `StateNode.On()` includes the invoke onDone/onError/onSnapshot transitions as JS `stateNode.transitions` does (keys `xstate.done.actor.*`, `xstate.error.actor.*`, `xstate.snapshot.*`).
- JS L139, L304, L344, L381: these tests do create machines/actors and call `getMeta()` at runtime but have no `expect`; classified N/A-type (consistent with the emit_1 / actions_3 manifests).
