> Historical translation record. Chunk tags, API-gap filenames, line numbers,
> and run counts below describe the original porting work. See the current
> [core guide](../core.md) and [architecture](../../ARCHITECTURE.md) for the maintained layout.

# type_helpers_1 — `packages/core/test/typeHelpers.test.ts` lines 1-226

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| ContextFrom > should return context of a machine | TestTypeHelpers_ContextFrom_ShouldReturnContextOfAMachine | N/A-type | JS L17-40: only `@ts-expect-error` checks on `ContextFrom<typeof machine>`; no `expect`. |
| EventFrom > should return events for a machine | TestTypeHelpers_EventFrom_ShouldReturnEventsForAMachine | N/A-type | JS L44-65: `EventFrom` union acceptance + `@ts-expect-error` on `UNKNOWN_EVENT`; no `expect`. Go events are untyped `xs.Event`. |
| EventFrom > should return events for an interpreter | TestTypeHelpers_EventFrom_ShouldReturnEventsForAnInterpreter | N/A-type | JS L67-90: same as above via `createActor(machine)` (never started); no `expect`. |
| MachineImplementationsFrom > should return implementations for a machine | TestTypeHelpers_MachineImplementationsFrom_ShouldReturnImplementationsForAMachine | N/A-type | JS L94-136: only type acceptance of implementations objects + `@ts-expect-error` on `100`; the `assign` callbacks are never executed. |
| StateValueFrom > should return any from a machine | TestTypeHelpers_StateValueFrom_ShouldReturnAnyFromAMachine | N/A-type | JS L140-146: `StateValueFrom` accepts any string; no runtime code beyond `createMachine({})`. |
| SnapshotFrom > should return state type from a service that has concrete event type | TestTypeHelpers_SnapshotFrom_ShouldReturnStateTypeFromAServiceThatHasConcreteEventType | ported | JS L150-164: runtime `createActor(...).getSnapshot()` ported; `SnapshotFrom` maps to Go static type `*xs.MachineSnapshot[any]` (checked at compile time). `@ts-expect-error acceptState("isn't any")` is a Go compile error — dropped (type-only). `types.events` has no Go counterpart. |
| SnapshotFrom > should return state from a machine without context | TestTypeHelpers_SnapshotFrom_ShouldReturnStateFromAMachineWithoutContext | ported | JS L166-174: same shape; `@ts-expect-error` dropped (type-only). |
| SnapshotFrom > should return state from a machine with context | TestTypeHelpers_SnapshotFrom_ShouldReturnStateFromAMachineWithContext | ported | JS L176-188: context `{counter: 0}` → `ctx{Counter: 0}`; accept param typed `*xs.MachineSnapshot[ctx]`; `@ts-expect-error` dropped (type-only). |
| ActorRefFrom > should return `ActorRef` based on actor logic | TestTypeHelpers_ActorRefFrom_ShouldReturnActorRefBasedOnActorLogic | ported | JS L192-208: custom logic → `&xs.Logic[*xs.BasicSnapshot[any]]{...}`; `ActorRefFrom<typeof logic>` → `*xs.Actor[*xs.BasicSnapshot[any]]`; runtime `start()` + `send({type:'TEST'})` ported. |
| tags > derives string from StateMachine | TestTypeHelpers_Tags_DerivesStringFromStateMachine | N/A-type | JS L212-224: `TagsFrom` accepts any string; no runtime expectations. |

Totals: 10 JS tests — 4 ported, 6 N/A-type, 0 N/A-runtime, 0 skipped-in-JS.

## API gaps

None (no `apigap_type_helpers_1.go`).

## Ambiguities for reviewer

- No test in this chunk has an `expect(...)` call. The 4 ported tests contain runtime calls (`getSnapshot`, `start`, `send`) whose implicit assertion is "does not throw"; they are ported with no extra assertions so as not to invent expectations. The type-helper part of each is expressed via Go static parameter types (compile-time), which is the closest Go equivalent.
- JS L77 (`EventFrom` interpreter test) also calls `createActor(machine)` at runtime but never starts or uses it; classified N/A-type for consistency with the other `createMachine`-only tests rather than ported as a no-assertion construction test.
- JS L150-157: `types: { events: {} as { type: 'FOO' } }` is type-only; the Go machine is `xs.MachineConfig[any]{}`.
