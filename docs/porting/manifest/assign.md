> Historical translation record. Chunk tags, API-gap filenames, line numbers,
> and run counts below describe the original porting work. See the current
> [core guide](../core.md) and [architecture](../../ARCHITECTURE.md) for the maintained layout.

## Source section: assign_1

Source: `references/xstate/packages/core/test/assign.test.ts` lines 1-367 (14 tests: 14 `it`, no `it.each`/`it.skip`/`it.todo`).
Go file: `xstate/assign_test.go`. No gap file.

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| assign > applies the assignment to the external state (property assignment) | `TestAssign_AppliesAssignmentToExternalStatePropertyAssignment` | ported | uses assign1CreateCounterMachine |
| assign > applies the assignment to the external state | `TestAssign_AppliesAssignmentToExternalState` | ported | uses assign1CreateCounterMachine |
| assign > applies the assignment to multiple properties (property assignment) | `TestAssign_AppliesAssignmentToMultiplePropertiesPropertyAssignment` | ported | partial-object assigner expressed as whole-context Assign |
| assign > applies the assignment to multiple properties (static) | `TestAssign_AppliesAssignmentToMultiplePropertiesStatic` | ported | static partial assign expressed as whole-context Assign |
| assign > applies the assignment to multiple properties (static + prop assignment) | `TestAssign_AppliesAssignmentToMultiplePropertiesStaticPlusPropAssignment` | ported | mixed static/function assign expressed as whole-context Assign |
| assign > applies the assignment to multiple properties | `TestAssign_AppliesAssignmentToMultipleProperties` | ported |  |
| assign > applies the assignment to the explicit external state (property assignment) | `TestAssign_AppliesAssignmentToExplicitExternalStatePropertyAssignment` | ported |  |
| assign > applies the assignment to the explicit external state | `TestAssign_AppliesAssignmentToExplicitExternalState` | ported |  |
| assign > should maintain state after unhandled event | `TestAssign_ShouldMaintainStateAfterUnhandledEvent` | ported | `context` toBeDefined -> assert.NotNil(snapshot) (context is a Go value type) |
| assign > sets undefined properties | `TestAssign_SetsUndefinedProperties` | ported | optional `maybe` modelled as `*string`; toBeDefined -> NotNil |
| assign > can assign from event | `TestAssign_CanAssignFromEvent` | ported |  |
| assign meta > should provide the parametrized action to the assigner | `TestAssign_Meta_ShouldProvideParametrizedActionToAssigner` | ported | params via `a.Params.(map[string]any)` |
| assign meta > should provide the action parameters to the partial assigner | `TestAssign_Meta_ShouldProvideActionParametersToPartialAssigner` | ported | params via `a.Params.(map[string]any)` |
| assign meta > a parameterized action that resolves to assign() should be provided the params | `TestAssign_Meta_ParameterizedActionResolvingToAssignShouldBeProvidedParams` | ported | Promise.withResolvers -> newSignal |

## API gaps

None.

## Ambiguities for reviewers

- L9-91 `createCounterMachine(context: Partial<CounterContext> = {})` spreads `{count: 0, foo: 'bar', ...context}`. Every caller in range passes nothing or both `count` and `foo` (L176, L193, L196, L208, L222, L225), so the Go helper `assign1CreateCounterMachine` takes the full context and no-arg callers pass `{Count: 0, Foo: "bar"}`.
- L37-79: JS distinguishes property-assigner (`assign({count: () => 100})`), static (`assign({count: 100})`), mixed, and function (`assign(() => ({...}))`) forms. The Go contract has only `xs.Assign(fn) C` (whole next context), so the WIN_PROP/WIN_STATIC/WIN_MIX/WIN tests exercise the same Go form; the observable assertions are unchanged. The original JS form is kept as a comment above each transition.
- L5 `maybe?: string`: modelled as `Maybe *string`; JS `toEqual({count, foo})` ignores an undefined `maybe`, which matches `Maybe: nil`.
- L252 `expect(nextState.context).toBeDefined()`: the Go context is a struct value (always defined); translated as `assert.NotNil(t, nextState)`.
- L344 `createMachine` with no context: Go uses `MachineConfig[any]` and `AssignArgs[any]`, returning `a.Context` unchanged.
