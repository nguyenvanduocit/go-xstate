> Historical translation record. Chunk tags, API-gap filenames, line numbers,
> and run counts below describe the original porting work. See the current
> [core guide](../core.md) and [architecture](../../ARCHITECTURE.md) for the maintained layout.

## Source section: machine_1

Source: `references/xstate/packages/core/test/machine.test.ts` lines 1-422 (whole file).
Go file: `xstate/machine_test.go`. Gap file: `apigap_machine_1.go`.

21 JS tests (20 `it()` + 1 `it.skip`), no `it.each` or generated tests. Top-level `pedestrianStates` /
`lightMachine` (lines 3-44) and the describe-scoped `resolveMachine` (lines 247-286) are helper funcs
`machine1PedestrianStates`, `machine1LightMachine`, `machine1ResolveMachine` (funcs, not package vars,
so the stubbed `CreateMachine` panic stays inside tests).

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| machine > machine.states > should properly register machine states | TestMachine_MachineStates_ShouldProperlyRegisterMachineStates | ported | `Object.keys(machine.states)` -> keys of `Root.ChildStates()` (document order) |
| machine > machine.events > should return the set of events accepted by machine | TestMachine_MachineEvents_ShouldReturnTheSetOfEventsAcceptedByMachine | ported | expectation sorted (Go API returns sorted events) |
| machine > machine.config > state node config should reference original machine config | TestMachine_MachineConfig_StateNodeConfigShouldReferenceOriginalMachineConfig | ported | identity (`toBe`) -> `assert.Equal`; mutation propagation assertion not expressible (see ambiguities) |
| machine > machine.provide > should override an action | TestMachine_MachineProvide_ShouldOverrideAnAction | ported | |
| machine > machine.provide > should override a guard | TestMachine_MachineProvide_ShouldOverrideAGuard | ported | |
| machine > machine.provide > should not override context if not defined | TestMachine_MachineProvide_ShouldNotOverrideContextIfNotDefined | ported | |
| machine > machine.provide > should override context (second argument) | TestMachine_MachineProvide_ShouldOverrideContextSecondArgument | skipped-in-JS | `it.skip`, body fully commented out |
| machine > machine.provide > should throw if initial state is missing in a compound state | TestMachine_MachineProvide_ShouldThrowIfInitialStateIsMissingInACompoundState | ported | `toThrow()` with no message -> `assert.Panics` |
| machine > machine.provide > machines defined without context should have a default empty object for context | TestMachine_MachineProvide_MachinesWithoutContextShouldHaveDefaultEmptyObjectContext | ported | context type `map[string]any`; expects `map[string]any{}` (not nil) |
| machine > machine.provide > should lazily create context for all interpreter instances created from the same machine template created by `provide` | TestMachine_MachineProvide_ShouldLazilyCreateContextForAllInstancesFromProvide | ported | `Foo` is a pointer so `not.toBe` -> `assert.NotSame` |
| machine > machine function context > context from a function should be lazily evaluated | TestMachine_MachineFunctionContext_ContextFromAFunctionShouldBeLazilyEvaluated | ported | `not.toBe` on context -> `NotSame` on nested pointer `Foo` |
| machine > machine.resolveStateValue() > should resolve the state value | TestMachine_MachineResolveStateValue_ShouldResolveTheStateValue | ported | |
| machine > machine.resolveStateValue() > should resolve `status: done` | TestMachine_MachineResolveStateValue_ShouldResolveStatusDone | ported | |
| machine > initial state > should follow always transition | TestMachine_InitialState_ShouldFollowAlwaysTransition | ported | |
| machine > versioning > should allow a version to be specified | TestMachine_Versioning_ShouldAllowAVersionToBeSpecified | ported | |
| machine > id > should represent the ID | TestMachine_ID_ShouldRepresentTheID | ported | |
| machine > id > should represent the ID (state node) | TestMachine_ID_ShouldRepresentTheIDStateNode | ported | |
| machine > id > should use the key as the ID if no ID is provided (state node) | TestMachine_ID_ShouldUseTheKeyAsTheIDIfNoIDIsProvidedStateNode | ported | |
| machine > combinatorial machines > should support combinatorial machines (single-state) | TestMachine_CombinatorialMachines_ShouldSupportCombinatorialMachinesSingleState | ported | unstarted snapshot value `{}` -> `map[string]any{}` |
| machine > should pass through schemas | TestMachine_ShouldPassThroughSchemas | ported | uses gaps `Setup.WithSchemas`, `StateMachine.Schemas` |
| StateNode > should list transitions | TestMachine_StateNode_ShouldListTransitions | ported | uses gap `StateNode.Transitions`; keys sorted |

Totals: 21 JS tests; 20 ported, 0 N/A-type, 0 N/A-runtime, 1 skipped-in-JS.

## API gaps (`apigap_machine_1.go`)

- `func (s *Setup[C]) WithSchemas(schemas any) *Setup[C]` — mirrors `setup({ schemas })` (setup.ts:476 passes `{ ...config, schemas }` to createMachine). Integration: `Schemas any` field on Setup.
- `func (m *StateMachine[C]) Schemas() any` — mirrors `machine.schemas` (StateMachine.ts:150). Integration: `Schemas any` field on StateMachine.
- `func (n *StateNode) Transitions() map[string][]*TransitionDefinition` — mirrors `stateNode.transitions` (StateNode.ts:157,241). Distinct from `On()` (`stateNode.on`, StateNode.ts:362), which drops forbidden events with no transitions; `transitions` keeps `FORBIDDEN_EVENT` (JS line 418 expects it).

## Ambiguities for review

- JS lines 67-92 (machine.config): JS asserts object identity (`oneState.config toBe machine.config.states.one`) and that mutating `deepState.config.meta` is visible via `machine.config`. `StateNode.Config` is a `StateConfig` value and `MachineConfig.States` a value slice in the contract, so identity and mutation propagation cannot hold; the Go test asserts value equality only. The mutation assertion (line 89-91) is dropped. If reference semantics matter, the contract would need `StateNode.Config *StateConfig` pointing into `MachineConfig.States`.
- JS line 189 (default empty context): `MachineConfig[any]` would leave context nil; the test uses `MachineConfig[map[string]any]` and expects `map[string]any{}`, i.e. the library must default a nil map context to an empty map.
- JS lines 207, 229 (`not.toBe` on context): Go struct contexts are always distinct values, so identity is checked on a nested pointer field, which is what JS's lazy context function guarantees (a fresh object per actor).
- JS lines 58-63 and 416-420: JS orders follow key insertion; Go expectations are sorted per docs/porting/core.md.
