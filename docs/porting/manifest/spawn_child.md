> Historical translation record. Chunk tags, API-gap filenames, line numbers,
> and run counts below describe the original porting work. See the current
> [core guide](../core.md) and [architecture](../../ARCHITECTURE.md) for the maintained layout.

## Source section: spawn_child_1

Source: `references/xstate/packages/core/test/spawnChild.test.ts` lines 1-119 (whole file).

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| spawnChild action > can spawn | TestSpawnChild_CanSpawn | ported | `toBeDefined` -> `assert.NotNil` on `Children["child"]` |
| spawnChild action > can spawn from named actor | TestSpawnChild_CanSpawnFromNamedActor | ported | `types.actors` (type-level) dropped; runtime assertion kept |
| spawnChild action > should accept `syncSnapshot` option | TestSpawnChild_ShouldAcceptSyncSnapshotOption | ported | `await promise` -> `sig.Wait(t)` (2s default; JS needs ~60ms) |
| spawnChild action > should handle a dynamic id | TestSpawnChild_ShouldHandleADynamicID | ported | `id: ({context}) => context.childId` -> `xs.NewExpr` |

## API gaps

None. All needed APIs exist in the contract (`SpawnChild`, `SpawnOptions{ID, Input, SyncSnapshot}`, `SnapshotEvent`, `ObservableSnapshot`).

## Ambiguities

- JS L55-60: context field `observableRef` is never assigned or read; kept as `ObservableRef xs.ActorRef` (nil) for fidelity.
- JS L65-68: guard reads `event.snapshot.context`; Go type-asserts `xs.SnapshotEvent` and `*xs.ObservableSnapshot[int]`.
