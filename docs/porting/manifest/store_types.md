> Historical translation record. Chunk tags, API-gap filenames, line numbers,
> and run counts below describe the original porting work. See the current
> [core guide](../core.md) and [architecture](../../ARCHITECTURE.md) for the maintained layout.

# store_types_1 manifest

JS: `references/xstate/packages/xstate-store/test/types.test.tsx` lines 1-521 (20 tests).
Go: `store/types_test.go` (tag `port_store_types_1`). No `apigap_store_types_1.go` needed.

`types.test.tsx` is a type-checking file: its only assertions are `satisfies` and `@ts-expect-error`
(both compile-time). The test bodies still execute at runtime under vitest, so every test whose body
calls the library is ported as a runtime test: the same calls in the same order, with `NotPanics`
(or `NotNil` for pure construction) standing in for "the test body does not throw". The type
assertions are dropped (the Go compiler checks types), and where the JS `satisfies` pins a value
whose runtime result is unambiguous, the value is asserted (marked "added assertion" below).

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| emitted > can emit a known event | TestStoreTypes_Emitted_CanEmitAKnownEvent | ported | Construction only (the handler is never invoked in JS). Asserts the store is created with `Schemas.Emitted`. |
| emitted > can't emit an unknown event | TestStoreTypes_Emitted_CantEmitAnUnknownEvent | ported | Construction only. `enq.emit.unknown()` is `@ts-expect-error`; kept as `enq.Emit("unknown")` inside the never-invoked handler. |
| emitted > can't emit a known event with wrong payload | TestStoreTypes_Emitted_CantEmitAKnownEventWithWrongPayload | ported | Construction only. The `'bazinga'` payload is kept in the never-invoked handler. |
| emitted > can subscribe to a known event | TestStoreTypes_Emitted_CanSubscribeToAKnownEvent | ported | `s.On("increased", ...)` must not panic; the `satisfies` inside the callback is type-only. |
| emitted > can't subscribe to a unknown event | TestStoreTypes_Emitted_CantSubscribeToAUnknownEvent | ported | Both `On("increased")` and `On("unknown")` run in JS (the latter is only a compile error); both are called. |
| emitted > wildcard listener receives union of all emitted events | TestStoreTypes_Emitted_WildcardListenerReceivesUnionOfAllEmittedEvents | ported | `s.On("*", ...)` must not panic; callback body is type-only. |
| emitted > works with a discriminated union event payload | TestStoreTypes_Emitted_WorksWithADiscriminatedUnionEventPayload | ported | Construction only. `z.discriminatedUnion` is `storeTypes1LogSchema()` (file-local SchemaFunc built from `zObject`/`zLiteral`/`zString`). |
| trigger > works with a distributive event payload | TestStoreTypes_Trigger_WorksWithADistributiveEventPayload | ported | Three `Trigger("log", ...)` calls, including the `@ts-expect-error` one, which runs at runtime in JS. |
| trigger > uses schema-declared events for trigger typing | TestStoreTypes_Trigger_UsesSchemaDeclaredEventsForTriggerTyping | ported | `on: {}` with only `schemas.events.log`; three `Trigger` calls must not panic (an event with no handler is a no-op in JS). Reviewer: confirm `Trigger` accepts schema-declared event types with no handler. |
| trigger > preserves inferred trigger typing when only emitted schemas are declared | TestStoreTypes_Trigger_PreservesInferredTriggerTypingWhenOnlyEmittedSchemasAreDeclared | ported | `Trigger("log", {message: "hello"})` runs; the `if (false)` block (type-only) is not executed in JS and is noted in a comment. |
| trigger > uses schema-declared events for enqueued trigger typing | TestStoreTypes_Trigger_UsesSchemaDeclaredEventsForEnqueuedTriggerTyping | ported | Construction only; the `flush` handler (which would re-trigger itself) is never invoked in JS, nor in Go. |
| can > uses event payload types | TestStoreTypes_Can_UsesEventPayloadTypes | ported | All five `Can` calls run (including the two `@ts-expect-error` ones). The handler reads `by` with comma-ok so a missing/string payload yields 0 and never panics, matching JS (`undefined` / `'one'` do not throw). Return values are not asserted (JS `satisfies boolean` is type-only). |
| logic selectors > infers selected values from a store | TestStoreTypes_LogicSelectors_InfersSelectedValuesFromAStore | ported | Added assertion: `count.Get() == 0`, `label.Get() == "Count: 0"`. `if (false)` block is type-only and skipped. |
| logic selectors > infers input and selector values from reusable store logic | TestStoreTypes_LogicSelectors_InfersInputAndSelectorValuesFromReusableStoreLogic | ported | Added assertion: `Selectors()["count"].Get() == 1`, `["label"] == "Count: 1"`. Input is a local `input` struct passed through `CreateStore(input...)`. `if (false)` block is type-only and skipped. |
| schemas > requires event and emitted schemas to define object payloads | TestStoreTypes_Schemas_RequiresEventAndEmittedSchemasToDefineObjectPayloads | N/A-type | Both `createStore` calls are `@ts-expect-error` (object-payload constraint on schemas); Go's `Schema` interface has no such constraint. |
| schemas > uses schema-declared context for snapshot typing | TestStoreTypes_Schemas_UsesSchemaDeclaredContextForSnapshotTyping | ported | Added assertions: `Context.Label == "ready"`, `assert.Same(schemas, s.Schemas())` (JS `store.schemas` is the config's schemas object). |
| schemas > merges schema-declared context with inferred event types | TestStoreTypes_Schemas_MergesSchemaDeclaredContextWithInferredEventTypes | ported | Added assertion: after `Trigger("rename", {label: "done"})`, `Context.Label == "done"`. `if (false)` block skipped. |
| fromStore schemas > preserves inferred event types when only emitted schemas are declared | TestStoreTypes_FromStoreSchemas_PreservesInferredEventTypesWhenOnlyEmittedSchemasAreDeclared | ported | Actor is created with `WithInput(1)` and, like JS, never started; `Send` then `On("increased")` must not panic. |
| fromStore schemas > uses schema-declared events for send typing | TestStoreTypes_FromStoreSchemas_UsesSchemaDeclaredEventsForSendTyping | ported | Two `Send` calls on an unstarted actor must not panic (`reset` has no handler). |
| fromStore schemas > uses schema-declared context for snapshot typing | TestStoreTypes_FromStoreSchemas_UsesSchemaDeclaredContextForSnapshotTyping | ported | `logic.GetInitialSnapshot(&xs.ActorScope{}, nil)` mirrors `getInitialSnapshot({} as any, undefined as never)`. Added assertion: `Context.Label == "ready"`. |

## API gaps

None.

## Things for a reviewer to check

- Added assertions (see Notes) go beyond the JS file. They are runtime facts the JS `satisfies` clauses imply; remove them if a strict 1:1 assertion count is wanted.
- 8 tests (the "construction only" rows) assert nothing beyond a non-nil store: in JS they exist purely to compile-check the generics. They are ported rather than `N/A-type` because `CreateStore` with `Schemas` runs at runtime in JS; reclassify as `N/A-type` if the project prefers.
- The `if (false)` blocks never execute in JS, so they have no Go counterpart other than comments.
- Go's `fmt.Sprintf("Count: %d", ...)` stands in for the template literal.
- `xs.CreateActor(xstore.FromStore(...))` relies on type inference of `S` from `*xs.Logic[*StoreSnapshot[C]]`; `go vet` accepts it.
