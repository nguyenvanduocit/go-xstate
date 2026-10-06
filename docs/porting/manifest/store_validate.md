> Historical translation record. Chunk tags, API-gap filenames, line numbers,
> and run counts below describe the original porting work. See the current
> [core guide](../core.md) and [architecture](../../ARCHITECTURE.md) for the maintained layout.

# store_validate_1 manifest

Source: `references/xstate/packages/xstate-store/test/validate.test.ts` lines 1-348.
Go: `store/validate_test.go` (tag `port_store_validate_1`).

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| validates initial context when the extension is applied | TestStoreValidate_ValidatesInitialContextWhenTheExtensionIsApplied | ported | `count: 'nope' as any` is `Count any` in a local context type; `toThrow(StoreValidationError)` is `ErrorAs` on the recovered panic |
| validates event payloads before transitions | TestStoreValidate_ValidatesEventPayloadsBeforeTransitions | ported | |
| validates final context after a macrostep | TestStoreValidate_ValidatesFinalContextAfterAMacrostep | ported | handler returns `Count: "nope"` through an `any` field |
| validates emitted payloads before running effects | TestStoreValidate_ValidatesEmittedPayloadsBeforeRunningEffects | ported | JS `enq.effect(effectSpy)` is a closure calling the spy; `return ctx` is `(c, true)` |
| validates no-payload events and emitted events as empty objects | TestStoreValidate_ValidatesNoPayloadEventsAndEmittedEventsAsEmptyObjects | ported | `toHaveBeenCalledWith` is `assert.Contains` on the spy calls |
| throws for unknown events and emitted events by default | TestStoreValidate_ThrowsForUnknownEventsAndEmittedEventsByDefault | ported | `toMatchObject` is per-field asserts; `payload: {}` is `JSONEq` against `{}` (also fails when Payload is nil) |
| returns false from can for validation errors | TestStoreValidate_ReturnsFalseFromCanForValidationErrors | ported | |
| exposes validation error details | TestStoreValidate_ExposesValidationErrorDetails | ported | `name: 'StoreValidationError'` is `IsStoreValidationError(thrown)` plus the `*StoreValidationError` type assertion; `issues: expect.any(Array)` is `NotEmpty` (stricter) |
| throws for unknown emitted events by default | TestStoreValidate_ThrowsForUnknownEmittedEventsByDefault | ported | |
| throws for unknown events by default | TestStoreValidate_ThrowsForUnknownEventsByDefault | ported | |
| can ignore unknown events and emitted events | TestStoreValidate_CanIgnoreUnknownEventsAndEmittedEvents | ported | `UnknownIgnore` for both policies |
| allows extension-added event types without schemas | TestStoreValidate_AllowsExtensionAddedEventTypesWithoutSchemas | ported | `.With(Reset).With(ValidateSchemas)` |
| can opt out of individual validation areas | TestStoreValidate_CanOptOutOfIndividualValidationAreas | ported | `SkipContext`, `SkipEvents` |
| warns and no-ops in dev when there are no schemas | TestStoreValidate_WarnsAndNoOpsInDevWhenThereAreNoSchemas | ported | `console.warn` spy is `ValidateSchemasOptions.Warn`; the exact warning string is asserted |
| throws a validation error for async schemas | TestStoreValidate_ThrowsAValidationErrorForAsyncSchemas | ported | `.refine(async ...)` is `zAsync(...)` |

## API gaps

None (`apigap_store_validate_1.go` not created).

## Ambiguities for review

- `payload` of `StoreValidationError`: JS asserts `{}` for events without payload and `{by: 'nope'}` otherwise. The Go type of `Payload` is `any`; tests compare its JSON encoding, so `xs.E`, `map[string]any` and similar all pass, but nil fails.
- Handlers returning `ctx` (JS `return ctx`) are ported as `(c, true)`; no test here asserts snapshot identity.
- Test names use the `TestStoreValidate_` prefix, so the later chunk of validate.test.ts (line 348 onward) must pick distinct `it` names or a different prefix.
