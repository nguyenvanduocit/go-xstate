> Historical translation record. Chunk tags, API-gap filenames, line numbers,
> and run counts below describe the original porting work. See the current
> [core guide](../core.md) and [architecture](../../ARCHITECTURE.md) for the maintained layout.

## Source section: tags_1

Source: `references/xstate/packages/core/test/tags.test.ts` lines 1-149 (whole file, 6 `it` calls, 1 `describe`, no `it.each`/`it.skip`).
Go file: `xstate/tags_test.go`.

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| tags > supports tagging states | TestTags_SupportsTaggingStates | ported | JS L4 |
| tags > supports tags in compound states | TestTags_SupportsTagsInCompoundStates | ported | JS L34 |
| tags > supports tags in parallel states | TestTags_SupportsTagsInParallelStates | ported | JS L65; `tags: 'yes'` -> `xs.Tags{"yes"}`; Set equality -> `assert.ElementsMatch` on sorted `snap.Tags` |
| tags > sets tags correctly after not selecting any transition | TestTags_SetsTagsCorrectlyAfterNotSelectingAnyTransition | ported | JS L104 |
| tags > tags can be single (not array) | TestTags_TagsCanBeSingleNotArray | ported | JS L121; Go contract `Tags` is `[]string` only, so the single-string form is a one-element slice (the string-vs-array normalization itself has no Go equivalent) |
| tags > stringifies to an array | TestTags_StringifiesToAnArray | ported | JS L134; `toJSON().tags` compared via `json.Marshal` + `assert.JSONEq` with `["go","light"]` (order-sensitive, element-type agnostic) |

## API gaps

None.

## Ambiguities for reviewer

- JS L121-132: the test's intent is that a scalar `tags: 'go'` is normalized to an array. Go's typed `xs.Tags` makes the scalar form inexpressible; the port keeps the runtime assertion (`hasTag('go')` on an unstarted actor's snapshot).
- JS L144-146: `toJSON()` returns `map[string]any`; the element type of `"tags"` is unspecified by the contract, hence the JSON comparison. Expected order `go, light` matches both JS Set insertion order and the contract's sorted order.
