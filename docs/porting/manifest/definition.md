> Historical translation record. Chunk tags, API-gap filenames, line numbers,
> and run counts below describe the original porting work. See the current
> [core guide](../core.md) and [architecture](../../ARCHITECTURE.md) for the maintained layout.

## Source section: definition_1

Source: `references/xstate/packages/core/test/definition.test.ts` lines 1-28.

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| definition > should provide invoke definitions | TestDefinition_ShouldProvideInvokeDefinitions | ported | `types: {} as {actors: ...}` (JS L6-16) is type-level only and dropped; runtime assertion `root.definition.invoke.length === 2` ported as `assert.Len(def["invoke"], 2)`. |

## API gaps

None. Uses `StateMachine.Root` and `(*StateNode).Definition()` (machine.go:28) from the contract.

## Ambiguities

- `StateNodeDefinition` is `map[string]any`; the concrete type of `def["invoke"]` is not fixed by the contract. `assert.Len` works on any slice type via reflection, so the test does not pin the element type. A `require.Contains` guard was added so a missing key fails clearly instead of a Len-on-nil message (it does not weaken the JS assertion: JS would throw on `undefined.length`).
