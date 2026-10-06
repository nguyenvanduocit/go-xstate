# stopwatch

Ported from `references/xstate/examples/stopwatch/src/stopwatchMachine.ts`.

## Not ported
- `src/main.ts`, `index.html`, `src/style.css`, `public/`, `package.json`, `tsconfig.json`: DOM rendering (output element, start/stop/reset buttons) and bundler setup.

## Deterministic trace
The JS `ticks` actor uses wall-clock `setInterval(..., 10)`. The golden trace swaps it for a no-op callback actor (`machine.provide` in
`scripts/trace/stopwatch/stopwatch.ts`; `idleTicks` in `machine_test.go`) and sends `TICK` explicitly, which reaches the same
`running.on.TICK` transition. The real ticker (`Ticks` in `machine.go`) is covered by `TestRealTicker` with polling and tolerance, not by the golden.

## Trace coverage (`testdata/stopwatch.golden.json`, recorded by `scripts/trace/stopwatch/stopwatch.ts`)
- States: `stopped` (initial), `running` (invokes `ticks`; `children` lists it).
- Transitions: `stopped --start--> running`, `running --stop--> stopped` (elapsed kept), `running --TICK--> assign elapsed+1`, root `reset` from `running` and from `stopped` (assign 0, target `.stopped`).
- Ignored events: `stop`/`TICK` in `stopped`, `start` in `running`, unknown event.
- The machine has no guards, delays, `after` or `onDone`/`onError`.
