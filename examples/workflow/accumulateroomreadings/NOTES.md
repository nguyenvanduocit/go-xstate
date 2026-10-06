# workflow-accumulate-room-readings

Ported from `references/xstate/examples/workflow-accumulate-room-readings/main.ts` (machine `roomreadings`, actor `produceReport`,
the console entry). Go: `machine.go` (`NewMachine`, `Machine`, `Run`, `RunWith`).

## Not ported
- `package.json`, `tsconfig.json`, `pnpm-lock.yaml`, `.gitignore`: vite-node/tsc/vite scripts and dependency setup (`cockatiel` is declared but never imported).
- `delay(ms, errorProbability)`: the `Math.random()` rejection path (`{ type: 'ServiceNotAvailable' }`) is dead code in this app (the only
  call passes probability 0), so Go keeps only the `setTimeout` part, as a `time.Timer` that also honours context cancellation
  (Go idiom for abandoning the promise when `GenerateReport` is exited).
- The demo's own `delay(1000/11_000/...)` waits and the `actor.subscribe({complete})` handler: `RunWith` sleeps the same amounts; `complete`
  never fires (the machine has no final state), so nothing is printed for it.
- `main.ts` hardcodes 10_000 ms (`PT1H`) and 1_000 ms (`produceReport`); Go takes them as `Timing` (`Machine()` uses `RealTiming`)
  and `RunWith(w, unit)` scales every duration of the entry, so tests need no 22 s run.
- The JS entry's `console.log` object output uses the runtime's formatter; `formatInput` reproduces that layout (bun: multi-line, trailing
  commas, `null`) only for the `{temperature, humidity}` input shape.
- Import side effect: `main.ts` starts the demo at import time (about 14 s) and exports `workflow`. The JS trace script mutes
  `console.log`, imports it and traces the exported machine; Go has no import-time side effect.
- `TODO: make this per room`: `roomId` is carried by the events but unused by the machine, as in JS.

## Trace coverage
`testdata/workflow-accumulate-room-readings.golden.json`, recorded by `scripts/trace/workflow-accumulate-room-readings/workflow-accumulate-room-readings.ts`
(SimulatedClock for `PT1H`; the real, unmodified `produceReport` with a 1100 ms real wait):
- states `ConsumeReading` (initial) and `GenerateReport`; the invoked child appears in `children` as `0.roomreadings.GenerateReport` only while generating.
- entry `assign` null/null: initial context and the reset after `onDone`.
- `TemperatureEvent` / `HumidityEvent` assigning, and a second `TemperatureEvent` overwriting the first; an unknown event; a `TemperatureEvent` ignored while generating.
- `after: PT1H` guard true: not due at 5000 ms and 9999 ms, fires at 10000 ms -> `GenerateReport`.
- `onDone` -> `ConsumeReading`.
- `after: PT1H` guard false (humidity set, temperature null): stays in `ConsumeReading`; the spent timer is not re-armed (a further
  10000 ms advance after both readings are set changes nothing).

`testdata/workflow-accumulate-room-readings.stdout.txt`, recorded from `main.ts` with real timers (about 21 s): two reports, `20/50` at 10 s and
`10/30` at 21 s (after the script itself has returned at 14 s; the last `PT1H` timer fires with both readings set). `TestStdout` runs `RunWith` with
one JS millisecond = 100 us (about 2.2 s); `RunWith` waits for the second report to complete, then stops the actor.
