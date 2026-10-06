> Historical translation record. Chunk tags, API-gap filenames, line numbers,
> and run counts below describe the original porting work. See the current
> [core guide](../core.md) and [architecture](../../ARCHITECTURE.md) for the maintained layout.

## Source section: spawn_1

Source: `references/xstate/packages/core/test/spawn.test.ts` lines 1-19.

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| spawn inside machine > input is required when defined in actor | TestSpawn_SpawnInsideMachine_InputIsRequiredWhenDefinedInActor | ported | `types: { input: {} as { value: number } }` on the child is type-level only; the "required" aspect is a TS compile-time check with no Go equivalent. Runtime assertion `actor.system.get('test')` toBeDefined → `assert.NotNil`. |

## API gaps

None (no `apigap_spawn_1.go`).

## Ambiguities

- JS L5-7, L10: the test name refers to TypeScript enforcement of `input` presence when the child declares an input type. Go has no such static check; only the runtime assertion (L16-17: spawned child registered under systemId `test`) is ported.
