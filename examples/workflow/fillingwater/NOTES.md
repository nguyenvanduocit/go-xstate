# workflow-filling-water

Ported from `references/xstate/examples/workflow-filling-water/main.ts` (serverless workflow "filling a glass of water").

- `machine.go`: the `workflow` machine (`Machine()`, `NewMachine(delay)`), `Input`/`Context` types with `json` tags equal to the JS keys, and the entry of `main.ts` (`Run`, `RunWith`): one actor with input `{current: 0, max: 10}`, a subscriber printing state and context for every snapshot, `workflow completed undefined` on completion.

## Not ported
- `package.json`, `pnpm-lock.yaml`, `tsconfig.json`: Node/vite-node/pnpm setup (`cockatiel` is declared in package.json but never imported by main.ts).
- `types: {} as {...}` (events `WaterAddedEvent`, context, input): type-level only; carried by the Go structs. `WaterAddedEvent` is never handled by any state.
- `console.log` of the context object: Go prints the fixed shape of this context in Bun's `console.log` format (multi-line, two-space indent, trailing commas) by hand; there is no general formatter because this example prints only `{ counts: { current, max } }`.
- `after: { 500: ... }` takes a numeric key in JS; Go `After` keys are strings, so `NewMachine(delay)` formats the delay in whole milliseconds (`Run` is the real 500 ms, `RunWith` injects a shorter delay for the test).

## Trace coverage
Scripts in `scripts/trace/workflow-filling-water/` import `main.ts` with console output muted (it starts a demo actor at import) and drive `workflow` with a SimulatedClock; the golden files record `input` and `clock: true`.
- `testdata/fill.golden.json` (`fill.ts`, input `{current: 0, max: 2}`): initial `CheckIfFull` -> always[0] -> `AddWater`; `advance 499` stays; `advance 1` fires `after 500` (assign `current` 1) -> `CheckIfFull` always[0] -> `AddWater`; `advance 500` assigns `current` 2 -> always[1] (guard false) -> `GlassFull` (status done); `advance 500` changes nothing.
- `testdata/already-full.golden.json` (`already-full.ts`, input `{current: 7, max: 5}`): initial `CheckIfFull` takes always[1] at once, `AddWater` is never entered; a later `advance 500` changes nothing.
- `testdata/workflow-filling-water.stdout.txt` (`workflow-filling-water.stdout.ts`): the real `main.ts` with its 500 ms delay (11 snapshots, 10 hops); `TestStdout` compares it with `RunWith` using a 5 ms delay.
- All 3 states, both `always` branches (guard true and false), the `after` transition with its assign and the final state are covered. The machine has no events, so event handling is not exercised.
