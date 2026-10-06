> Historical translation record. Chunk tags, API-gap filenames, line numbers,
> and run counts below describe the original porting work. See the current
> [core guide](../core.md) and [architecture](../../ARCHITECTURE.md) for the maintained layout.

## Source section: assert_1

Source: `references/xstate/packages/core/test/assert.test.ts` lines 1-107.
Go file: `xstate/assert_test.go`.

Totals: 2 JS tests; 2 ported, 0 N/A-type, 0 N/A-runtime, 0 skipped-in-JS.

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| assertion helpers > assertEvent asserts the correct event type | TestAssert_AssertEventAssertsCorrectEventType | ported | `@ts-expect-error` property accesses and `satisfies` checks (JS L22-30) are type-level only and dropped; runtime `assertEvent` call and inline-snapshot error message kept. |
| assertion helpers > assertEvent asserts multiple event types | TestAssert_AssertEventAssertsMultipleEventTypes | ported | `@ts-expect-error` / `satisfies` lines (JS L71-85) dropped as type-level; both runtime `assertEvent` calls (`['greet','notify']`, then `['notify']`) kept. |

## API gaps

None. Uses existing contract `xs.AssertEvent(event, types...)` (`util.go:44`).

## Reviewer notes

- `toMatchInlineSnapshot(\`[Error: msg]\`)` (JS L39-41, L91-93) is translated as: the observer's `err` must be a Go `error` whose `Error()` equals `msg` exactly. The JS message includes `JSON.stringify(event)`; for `xs.E{"type":"count","value":42}` the implementation must serialize keys as `{"type":"count","value":42}` (Go `encoding/json` sorts map keys, which coincides here).
- Single-element vs array form of `assertEvent` (JS `'greet'` vs `['greet','notify']`) both map to the variadic `AssertEvent`. The wording is chosen by count after `toArray` in JS (`packages/core/src/assert.ts:42-44`: `types.length === 1` gives "type matching", otherwise "one of types matching"), so variadic is equivalent.
