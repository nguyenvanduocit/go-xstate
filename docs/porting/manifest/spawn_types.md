> Historical translation record. Chunk tags, API-gap filenames, line numbers,
> and run counts below describe the original porting work. See the current
> [core guide](../core.md) and [architecture](../../ARCHITECTURE.md) for the maintained layout.

## Source section: spawn_types_1

Source: `references/xstate/packages/core/test/spawn.types.test.ts` lines 1-50 (`describe('spawn inside machine')`, tests at L4 and L28).
Go file: `xstate/spawn_types_test.go`.

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| spawn inside machine > input is required when defined in actor | TestSpawnTypes_SpawnInsideMachine_InputIsRequiredWhenDefinedInActor | ported | JS L4. No `expect`; implicit check (building the machines, incl. `spawn(childMachine, { input })` in `context` fn and in `assign`, does not throw) -> `assert.NotPanics`. Only the `types: { input }` requirement is TS-only and dropped. |
| spawn inside machine > input is not required when not defined in actor | TestSpawnTypes_SpawnInsideMachine_InputIsNotRequiredWhenNotDefinedInActor | ported | JS L28. No `expect`; implicit check (building the machines, incl. `spawn(childMachine)` without input, does not throw) -> `assert.NotPanics`. |

## API gaps

None.

## Ambiguities for reviewer

- Both JS tests have no `expect`. Same convention as `setup_types_1`: ported as `assert.NotPanics` around machine creation (spawn calls live in `ContextFn`/`Assign` closures that are not invoked without starting an actor, as in JS). The compile-time-only check (input required/optional by `types.input`) has no Go equivalent (`SpawnOptions.Input` is `any`); reviewer may reclassify as N/A-type if preferred.
- JS L13 and L34 use `initial: 'idle'` with state key `Idle` (case mismatch); copied verbatim. In JS this does not throw at creation (`initial` getter in StateNode.ts:389 is lazy). If the Go implementation resolves `Initial` eagerly in `CreateMachine`, these tests would fail by design faithfulness; flag for implementer.
- JS L6 and L9, L31 `types:` annotations have no Go counterpart (`MachineConfig` has no `Types` field); `ActorRefFrom<typeof childMachine>` maps to `xs.ActorRef`.
