> Historical translation record. Chunk tags, API-gap filenames, line numbers,
> and run counts below describe the original porting work. See the current
> [core guide](../core.md) and [architecture](../../ARCHITECTURE.md) for the maintained layout.

## Source section: logger_1

Source: `references/xstate/packages/core/test/logger.test.ts` lines 1-42.
Go file: `xstate/logger_test.go`.

| JS test (describe > it) | Go test | Status | Notes |
|---|---|---|---|
| logger > system logger should be default logger for actors (invoked from machine) | TestLogger_SystemLoggerShouldBeDefaultLoggerForActorsInvokedFromMachine | ported | `expect.assertions(1)` with the assertion inside the logger → logger called exactly once; first arg `"hello"`. |
| logger > system logger should be default logger for actors (spawned from machine) | TestLogger_SystemLoggerShouldBeDefaultLoggerForActorsSpawnedFromMachine | ported | Same as above, child spawned via `xs.SpawnChild(logic)`. |

## API gaps

None. Uses existing `xs.WithLogger`, `xs.Log`, `xs.SpawnChild`, `xs.InvokeConfig.Logic`.

## Ambiguities for review

- JS lines 6, 25: `expect.assertions(1)` counts assertions made inside the logger callback, so it implicitly requires exactly one logger call. Go records calls with `newSpy()` and asserts `len == 1` plus `calls[0][0] == "hello"`.
- JS lines 15-20, 34-39: logger callback reads only the first argument (`arg`); JS `log('hello')` with no label calls `logger(value)` (`src/actions/log.ts:51`), so Go checks only the first argument.
- JS lines 19, 38: the redundant second `actor.start()` is kept (`actor.Start()`), since it is part of the scenario (start on a running actor must be a no-op and must not log again).
