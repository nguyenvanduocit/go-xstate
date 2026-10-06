> Historical translation record. Chunk tags, API-gap filenames, line numbers,
> and run counts below describe the original porting work. See the current
> [core guide](../core.md) and [architecture](../../ARCHITECTURE.md) for the maintained layout.

# event_descriptors_1 — eventDescriptors.test.ts lines 1-460

Source: `references/xstate/packages/core/test/eventDescriptors.test.ts` (459 lines, whole file).
Go file: `xstate/event_descriptors_test.go` (tag `port_event_descriptors_1`).

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| event descriptors > should fallback to using wildcard transition definition (if specified) | TestEventDescriptors_ShouldFallbackToUsingWildcardTransitionDefinitionIfSpecified | ported | |
| event descriptors > should prioritize explicit descriptor even if wildcard comes first | TestEventDescriptors_ShouldPrioritizeExplicitDescriptorEvenIfWildcardComesFirst | ported | key order irrelevant in Go map |
| event descriptors > should prioritize explicit descriptor even if a partial one comes first | TestEventDescriptors_ShouldPrioritizeExplicitDescriptorEvenIfAPartialOneComesFirst | ported | |
| event descriptors > should prioritize a longer descriptor even if the shorter one comes first | TestEventDescriptors_ShouldPrioritizeALongerDescriptorEvenIfTheShorterOneComesFirst | ported | |
| event descriptors > should use a shorter descriptor if the longer one doesn't match | TestEventDescriptors_ShouldUseAShorterDescriptorIfTheLongerOneDoesntMatch | ported | |
| event descriptors > should fall back to wildcard descriptor when exact descriptor guard fails | TestEventDescriptors_ShouldFallBackToWildcardDescriptorWhenExactDescriptorGuardFails | ported | |
| event descriptors > should NOT support non-tokenized wildcards | TestEventDescriptors_ShouldNOTSupportNonTokenizedWildcards | ported | |
| event descriptors > should support prefix matching with wildcards (+0) | TestEventDescriptors_ShouldSupportPrefixMatchingWithWildcardsPlus0 | ported | |
| event descriptors > should support prefix matching with wildcards (+1) | TestEventDescriptors_ShouldSupportPrefixMatchingWithWildcardsPlus1 | ported | |
| event descriptors > should support prefix matching with wildcards (+n) | TestEventDescriptors_ShouldSupportPrefixMatchingWithWildcardsPlusN | ported | |
| event descriptors > should support prefix matching with wildcards (+n, multi-prefix) | TestEventDescriptors_ShouldSupportPrefixMatchingWithWildcardsPlusNMultiPrefix | ported | |
| event descriptors > should not match infix wildcards | TestEventDescriptors_ShouldNotMatchInfixWildcards | ported | console.warn spy → `WithWarnHandler` per actor; warning order written with `on` keys sorted (see ambiguity 1) |
| event descriptors > should not match wildcards as part of tokens | TestEventDescriptors_ShouldNotMatchWildcardsAsPartOfTokens | ported | same as above |
| event descriptors > should allow assertEvent to use partial descriptors | TestEventDescriptors_ShouldAllowAssertEventToUsePartialDescriptors | ported | `satisfies` type checks dropped (type-level only) |
| event descriptors > should throw if assertEvent partial descriptor does not match | TestEventDescriptors_ShouldThrowIfAssertEventPartialDescriptorDoesNotMatch | ported | `toThrowErrorMatchingInlineSnapshot` → `assert.PanicsWithError` |

Totals: 15 JS tests — 15 ported, 0 N/A-type, 0 N/A-runtime, 0 skipped-in-JS.

## API gaps (`apigap_event_descriptors_1.go`)

- `WithWarnHandler(fn func(args ...any)) ActorOption` — mirrors `vi.spyOn(console, 'warn')`. Same signature
  as the existing gaps in `apigap_actions_3.go` and `apigap_actions_4.go`.

## Ambiguities for review

1. JS lines 260-333 and 335-389: the warnings come from `matchesEventDescriptor` (utils.ts:297-345), called by
   `getCandidates` (stateUtils.ts:213-230) over `stateNode.transitions.keys()` in insertion order. Go maps have
   no insertion order, so following docs/porting/core.md "Ordering differences" the expected warning sequences are
   written with the descriptors visited in sorted key order (`"*.event.*"` before `"event.*.bar.*"`,
   `"*event.*"` before `"event*.bar.*"`). Within one descriptor the JS order (wildcard warning, then infix
   warning) is kept. If the implementation preserves document order instead, swap the groups back to JS order.
2. `warnSpy.mockClear()` (JS lines 293, 360) is expressed by giving each actor its own warn spy, since
   `WithWarnHandler` is per-actor in Go while JS spies on the global console. Warnings are emitted only during
   `send` (transition selection), not at actor creation, so the per-actor split captures the same calls.
3. `toHaveBeenNthCalledWith(n, {...})` (JS lines 425-432): the spy records `a.Event`, compared against
   `xs.E{...}` with `rate` as Go `int` 5. The implementation must deliver the sent `xs.E` value unchanged.
