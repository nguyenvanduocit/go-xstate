> Historical translation record. Chunk tags, API-gap filenames, line numbers,
> and run counts below describe the original porting work. See the current
> [core guide](../core.md) and [architecture](../../ARCHITECTURE.md) for the maintained layout.

## Source section: guards_1

Source: `references/xstate/packages/core/test/guards.test.ts` lines 1-1555 (46 JS tests).
Go file: `xstate/guard_test.go`.

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| guard conditions > should transition only if condition is met | TestGuards_GuardConditions_ShouldTransitionOnlyIfConditionIsMet | ported | shared lightMachine -> `guards1LightMachine()` |
| guard conditions > should transition if condition based on event is met | TestGuards_GuardConditions_ShouldTransitionIfConditionBasedOnEventIsMet | ported | |
| guard conditions > should not transition if condition based on event is not met | TestGuards_GuardConditions_ShouldNotTransitionIfConditionBasedOnEventIsNotMet | ported | |
| guard conditions > should not transition if no condition is met | TestGuards_GuardConditions_ShouldNotTransitionIfNoConditionIsMet | ported | trackEntries copied as `guards1TrackEntries` |
| guard conditions > should work with defined string transitions | TestGuards_GuardConditions_ShouldWorkWithDefinedStringTransitions | ported | |
| guard conditions > should work with guard objects | TestGuards_GuardConditions_ShouldWorkWithGuardObjects | ported | |
| guard conditions > should work with defined string transitions (condition not met) | TestGuards_GuardConditions_ShouldWorkWithDefinedStringTransitionsConditionNotMet | ported | |
| guard conditions > should throw if string transition is not defined | TestGuards_GuardConditions_ShouldThrowIfStringTransitionIsNotDefined | ported | inline snapshot -> exact error message |
| guard conditions > should guard against transition | TestGuards_GuardConditions_ShouldGuardAgainstTransition | ported | second `guard conditions` describe |
| guard conditions > should allow a matching transition | TestGuards_GuardConditions_ShouldAllowAMatchingTransition | ported | |
| guard conditions > should check guards with interim states | TestGuards_GuardConditions_ShouldCheckGuardsWithInterimStates | ported | |
| custom guards > should evaluate custom guards | TestGuards_CustomGuards_ShouldEvaluateCustomGuards | ported | context is `map[string]int` (dynamic `prop` key) |
| custom guards > should provide the undefined params if a guard was configured using a string | TestGuards_CustomGuards_ShouldProvideUndefinedParamsIfGuardConfiguredUsingString | ported | undefined -> nil |
| custom guards > should provide the guard with resolved params when they are dynamic | TestGuards_CustomGuards_ShouldProvideGuardWithResolvedParamsWhenDynamic | ported | |
| custom guards > should resolve dynamic params using context value | TestGuards_CustomGuards_ShouldResolveDynamicParamsUsingContextValue | ported | |
| custom guards > should resolve dynamic params using event value | TestGuards_CustomGuards_ShouldResolveDynamicParamsUsingEventValue | ported | |
| custom guards > should call a referenced `not` guard that embeds an inline function guard with undefined params | TestGuards_CustomGuards_ShouldCallReferencedNotEmbeddingInlineGuardWithUndefinedParams | ported | |
| custom guards > should call a string guard referenced by referenced `not` with undefined params | TestGuards_CustomGuards_ShouldCallStringGuardReferencedByReferencedNotWithUndefinedParams | ported | |
| custom guards > should call an object guard referenced by referenced `not` with its own params | TestGuards_CustomGuards_ShouldCallObjectGuardReferencedByReferencedNotWithOwnParams | ported | |
| custom guards > should call an inline function guard embedded in referenced `and` with undefined params | TestGuards_CustomGuards_ShouldCallInlineGuardEmbeddedInReferencedAndWithUndefinedParams | ported | |
| custom guards > should call a string guard referenced by referenced `and` with undefined params | TestGuards_CustomGuards_ShouldCallStringGuardReferencedByReferencedAndWithUndefinedParams | ported | |
| custom guards > should call an object guard referenced by referenced `and` with its own params | TestGuards_CustomGuards_ShouldCallObjectGuardReferencedByReferencedAndWithOwnParams | ported | |
| referencing guards > guard should be checked when referenced by a string | TestGuards_ReferencingGuards_GuardCheckedWhenReferencedByString | ported | `vi.fn()` guard returns false (JS undefined) |
| referencing guards > guard should be checked when referenced by a parametrized guard object | TestGuards_ReferencingGuards_GuardCheckedWhenReferencedByParametrizedGuardObject | ported | `'x'` and `{type:'x'}` both map to `GuardRef{Type:"x"}` |
| referencing guards > should throw for guards with missing predicates | TestGuards_ReferencingGuards_ShouldThrowForGuardsWithMissingPredicates | ported | inline snapshot -> exact error message |
| referencing guards > should be possible to reference a composite guard that only uses inline predicates | TestGuards_ReferencingGuards_ReferenceCompositeGuardWithOnlyInlinePredicates | ported | |
| referencing guards > should be possible to reference a composite guard that references other guards recursively | TestGuards_ReferencingGuards_ReferenceCompositeGuardReferencingOtherGuardsRecursively | ported | |
| referencing guards > should be possible to resolve referenced guards recursively | TestGuards_ReferencingGuards_ResolveReferencedGuardsRecursively | ported | impl value `'ref2'` -> `GuardRef{Type:"ref2"}` |
| guards - other > should allow for a fallback target to be a simple string | TestGuards_Other_ShouldAllowFallbackTargetToBeSimpleString | ported | |
| guards - other > inline function guard should not leak into provided guards object | TestGuards_Other_InlineFunctionGuardShouldNotLeakIntoProvidedGuards | ported | Go map shared by reference, so the check is meaningful |
| guards - other > inline builtin guard should not leak into provided guards object | TestGuards_Other_InlineBuiltinGuardShouldNotLeakIntoProvidedGuards | ported | |
| not() guard > should guard with inline function | TestGuards_Not_ShouldGuardWithInlineFunction | ported | |
| not() guard > should guard with string | TestGuards_Not_ShouldGuardWithString | ported | |
| not() guard > should guard with object | TestGuards_Not_ShouldGuardWithObject | ported | |
| not() guard > should guard with nested built-in guards | TestGuards_Not_ShouldGuardWithNestedBuiltInGuards | ported | |
| not() guard > should evaluate dynamic params of the referenced guard | TestGuards_Not_ShouldEvaluateDynamicParamsOfReferencedGuard | ported | inline snapshot -> exact spy calls |
| and() guard > should guard with inline function | TestGuards_And_ShouldGuardWithInlineFunction | ported | |
| and() guard > should guard with string | TestGuards_And_ShouldGuardWithString | ported | |
| and() guard > should guard with object | TestGuards_And_ShouldGuardWithObject | ported | |
| and() guard > should guard with nested built-in guards | TestGuards_And_ShouldGuardWithNestedBuiltInGuards | ported | |
| and() guard > should evaluate dynamic params of the referenced guard | TestGuards_And_ShouldEvaluateDynamicParamsOfReferencedGuard | ported | inline snapshot -> exact spy calls |
| or() guard > should guard with inline function | TestGuards_Or_ShouldGuardWithInlineFunction | ported | |
| or() guard > should guard with string | TestGuards_Or_ShouldGuardWithString | ported | |
| or() guard > should guard with object | TestGuards_Or_ShouldGuardWithObject | ported | |
| or() guard > should guard with nested built-in guards | TestGuards_Or_ShouldGuardWithNestedBuiltInGuards | ported | |
| or() guard > should evaluate dynamic params of the referenced guard | TestGuards_Or_ShouldEvaluateDynamicParamsOfReferencedGuard | ported | inline snapshot -> exact spy calls |

Totals: 46 JS tests; 46 ported, 0 N/A-type, 0 N/A-runtime, 0 skipped-in-JS.

## API gaps

None. No `apigap_guards_1.go` was created.

## Ambiguities for review

- JS L18-79 (`lightMachine`): `context: ({ input = {} }) => ({ elapsed: input.elapsed ?? 0 })`. Go input is `map[string]any{"elapsed": n}`; `ContextFn` falls back to 0 when input is nil or the key is missing. The machine is built per test via `guards1LightMachine()`, not as a package var, because building it at package init would panic while the stubs are unimplemented.
- JS L228-260 and L864-895: the error-observer inline snapshot is asserted as exactly one call whose single argument is an `error` with the full message (including the `\n` between the two lines and the trailing `'.`). The error is assumed to arrive as a Go `error` value.
- JS L474-499 and similar: `toHaveBeenCalledWith(undefined)` becomes `assert.Contains(spy.Calls(), []any{nil})`, so the guard's `a.Params` must be an untyped nil.
- JS L406-472: guard params are a typed struct (`customParams`), and context is `map[string]int` because the guard indexes `context[prop]`.
- JS L806-862: `checkStuff: vi.fn()` becomes a `GuardFunc` that records the call and returns `false` (vi.fn returns undefined, which is falsy). Only the call count is asserted, as in JS.
- JS L1110-1142, L1267-1305, L1438-1476: `params: { value: N }` becomes a local struct `greaterThan10Params{Value}`, and the guard type-asserts `a.Params` to it.
- The dynamic `params` Exprs (L501+, L1174+) return `map[string]any`, and the spy expectation is the same map type.
