# timer

Ported from `references/xstate/examples/timer/src/timerMachine.ts`.

## Not ported
- `src/App.tsx`, `src/main.tsx`, `src/App.css`, `src/index.css`, `src/assets/`, `src/vite-env.d.ts`, `index.html`, `public/`, `vite.config.ts`, `tsconfig*.json`, `package.json`: React rendering, CSS, HTML and bundler setup.
- The JS `types: {} as {...}` event union: compile-time only, the Go events are plain strings.

## Go-only additions
- `NewMachine(ticks)` takes the `ticks` actor as a parameter; `Machine()` uses the production `Ticks(TickInterval)` (1 s `time.Ticker`, the Go form of `setInterval(..., 1000)`). The golden trace swaps in a no-op callback actor, as the JS script does through `machine.provide`.
- The `Ticks` cleanup does not wait for its goroutine: the library runs callback cleanup while holding the system lock and `SendBack` takes the same lock, so waiting would deadlock (see `docs/porting/notes/examples-lib-findings.md`, section timer).

## Trace coverage (`testdata/timer.golden.json`, recorded by `scripts/trace/timer/timer.ts`)
- `stopped`: `start` with guard false (`seconds == 0`) and true, `second`, `minute`, `stop` and `TICK` ignored.
- `running`: `TICK` decrement, `stop` keeps seconds, `start` / `minute` / `second` ignored, resume after stop.
- `always` guard (`seconds === 0`): reached via `TICK` down to 0 (twice) and via root `reset` while running.
- root `reset`: guard false (seconds 0, from stopped and again after a reset), guard true from stopped and from running (assign 0, then `always` returns to `stopped`).
- unknown event: no change. Child id `0.(machine).running` (the JS machine has no `id`).
- `TestRealTicker` (Go only): production ticker with a 5 ms interval counts 3 seconds down to 0, stops itself, and the ticker stops with the running state.
