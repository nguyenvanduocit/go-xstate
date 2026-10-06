> Historical translation record. Chunk tags, API-gap filenames, line numbers,
> and run counts below describe the original porting work. See the current
> [core guide](../core.md) and [architecture](../../ARCHITECTURE.md) for the maintained layout.

## Source section: json_1

Source: `references/xstate/packages/core/test/json.test.ts` lines 1-193 (3 tests, no `it.each`/`it.skip`).
Go file: `xstate/json_test.go`.

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| json > should serialize the machine | TestJSON_ShouldSerializeTheMachine | ported | `types: {} as {context}` dropped (type-only). Ajv schema validation -> gap `ValidateMachineSchema`; `expect(validate.errors).toBeNull()` -> `assert.Nil(errs)`. `JSON.parse(JSON.stringify(machine.definition))` -> json.Marshal/Unmarshal of `machine.Definition()` (marshal error fails the test). |
| json > should detect an invalid machine | TestJSON_ShouldDetectAnInvalidMachine | ported | `validate(invalid); expect(validate.errors).not.toBeNull()` -> `assert.NotNil(ValidateMachineSchema(invalid))`. |
| json > should not double-serialize invoke transitions | TestJSON_ShouldNotDoubleSerializeInvokeTransitions | ported | `createMachine(JSON.parse(JSON.stringify(machine)))` -> gap `CreateMachineFromJSON(machine.ToJSON() round-tripped)`. Inline snapshot asserted field by field on `TransitionDefinition` and on the JSON-normalized `ToJSON()` (actions `[]`, eventType, guard nil, reenter false, source `#active`, target `["#(machine).foo"]`/`["#(machine).bar"]`). `.transitions` -> `StateNode.On()` (JS `on` is derived from `transitions` with the same content), flattened in sorted descriptor order. |

## API gaps (`apigap_json_1.go`)

- `ValidateMachineSchema(v any) []string` — mirrors `ajv.compile(machine.schema.json)` + `validate(v)` + `validate.errors`; nil when valid.
- `CreateMachineFromJSON(config map[string]any, impl ...Implementations) *StateMachine[any]` — mirrors `createMachine(plainJSONObject)` for a revived `machine.toJSON()`.

## Ambiguities for review

- JS L33-39: `{ type: 'objectActionTypeWithExec', exec, other: 'any' }` -> `xs.ActionRef{Type: "objectActionTypeWithExec"}`; `exec`/`other` have no ActionRef field (they are dropped by serialization in JS anyway).
- JS L46-48: `assign((ctx) => ({ ...ctx }))` spreads the assign-args object (v5 signature), so it is translated as returning a map of the args fields (context/event/self/system/spawn). Never executed by the test.
- JS L54-57: guard `!!context.string` -> non-empty string check on `context["string"]`.
- JS L152-187: snapshot order EVENT, done, error equals sorted descriptor order, so the sorted Go order matches JS. The JSON type of `ToJSON()["target"]` is asserted after a JSON round-trip (`[]any{"#(machine).foo"}`) to avoid depending on the Go slice type; `guard: undefined` is asserted as nil/absent.
- `ValidateMachineSchema` returns `[]string`; Ajv errors are objects. Only nil/non-nil is asserted, so the element type does not affect the test.
