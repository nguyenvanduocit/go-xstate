# workflow-monitor-patient

Ported from `references/xstate/examples/workflow-monitor-patient/main.ts` (serverless workflow "monitor patient vital signs").

- `machine.go`: the machine (`patientVitalsWorkflow`, single state `MonitorVitals`; `org.monitor.highBodyTemp` runs `sendTylenolOrder`, `org.monitor.highBloodPressure` runs `callNurse`, `org.monitor.highRespirationRate` runs `callPulmonologist`; none changes state), `Input`/`Context` (`{patientId}`), `VitalEvent` (the event shape of main.ts's `events` union) and the entry (`Run`, `RunWith`).
- `main.ts` builds the machine, starts an actor for `patient1` and sends a random vital-sign event every 3 s forever. Go `Run(ctx, w)` is that entry (3 s ticker, `rand.Float64`) and stops when `ctx` is cancelled; `RunWith` injects the ticks and the random source for tests.

## Not ported
- `package.json`, `pnpm-lock.yaml`, `tsconfig.json`, `.gitignore`: Node/pnpm/vite-node setup.
- `console.log(...)` in the three actions: written to the `io.Writer` given to `NewMachine`.
- `actor.subscribe({ complete() { console.log('workflow completed', ...) } })`: the machine has no final state, so it never runs in JS or Go; omitted.
- `setInterval` that never ends and `new Date().toISOString()` / `Math.random()` as sources: replaced by the injected `ticks` and `random` of `RunWith`; `Run` uses a real ticker and `rand.Float64`. Nothing is recorded from a real clock or real randomness.
- TypeScript types (`types: { input, context, events }`): Go uses the `Input` and `Context` structs; events are `xs.E` built by `VitalEvent`.

## Trace coverage
- `testdata/workflow-monitor-patient.golden.json` (`scripts/trace/workflow-monitor-patient/workflow-monitor-patient.ts`, input `{patientId: "patient1"}`): initial snapshot (context from `input`), then each of the three events (state stays `MonitorVitals`, status `active`), an unknown event (ignored), and a repeat of `highBodyTemp`. This is the only state and every transition; the machine has no guards, delays or invokes. Action effects are not part of a snapshot, so they are covered by the next file.
- `testdata/actions.stdout.txt` (`actions.stdout.ts`): the machine driven by hand with patient `patient42`: the exact text printed by `callPulmonologist`, `sendTylenolOrder`, `callNurse`, and nothing for an unknown event.
- `testdata/workflow-monitor-patient.stdout.txt` (`workflow-monitor-patient.stdout.ts`): main.ts's own entry run with `Math.random` fixed to `0, .5, .9, .4, .7, .1` and `setInterval` fired six times by the script; `TestStdout` feeds the same sequence to `RunWith`.
