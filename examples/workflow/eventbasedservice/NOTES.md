# workflow-event-based-service

Ported from `references/xstate/examples/workflow-event-based-service/main.ts` (serverless workflow "event-based service invocation").

- `machine.go`: `NewMachine` (machine `VetAppointmentWorkflow`: `Idle` -> `MakeVetAppointmentState` on `MakeVetAppointment` with `assign` of
  `patientInfo`; the invoked `MakeAppointmentAction` `onDone` -> `Idle` with `assign` of `appointmentInfo`), the real actor
  (`MakeAppointmentAction`) and the entry (`Run`, `RunWith`).
- `main.ts` builds the machine and runs a demo actor at import time; Go `Run(w)` is that entry. `RunWith(w, delay, now)` injects the 2000 ms
  delay and the clock for tests.
- `context.appointmentInfo` holds the whole actor output `{appointmentInfo: {...}}` (the JS `assign` takes `event.output`), so the snapshot
  nests it twice; `AppointmentResult` keeps that shape.

## Not ported
- `package.json`, `pnpm-lock.yaml`, `tsconfig.json`, `.gitignore`: Node/pnpm/vite-node setup.
- `console.log` object formatting: the Go actor prints the two objects in the layout Bun's `console.log` produced when the golden was recorded
  (multi-line, double quotes, trailing commas; Node prints `{ name: 'Jenny', ... }` on one line). The layout is hand-written for these two
  shapes and uses `%q`, which matches JS quoting only for strings without special characters.
- `new Date().toISOString()`: `Date.now` is injected (`now func() time.Time`, formatted with `DateLayout`); the JS stdout script pins `Date` to
  `2026-01-02T03:04:05.678Z` and the Go test passes the same instant.
- The `complete` observer is wired but never fires: the machine has no final state, so the JS entry never prints `workflow completed`. The JS
  process exits when no timer is left; `RunWith` returns once the machine is back in `Idle` with the appointment assigned, and unsubscribes
  before `Stop` (xstate's `stop()` completes the actor in JS as well, which would print the line).
- The machine's `input` (`{person: {name: 'Jenny'}}` in `main.ts`) is not read by the machine; it is passed anyway and recorded in the golden.
- The JS promise cannot be cancelled; Go selects on `ctx.Done()` so stopping the actor abandons the action (`TestMakeAppointmentAction_StoppedActorAbandonsPromise`).
- Event payload: `MakeVetAppointment` is `xs.E` with `patientInfo` as `map[string]any`; `patientInfoOf` keeps only `name`, `pet`, `reason` (extra keys of
  the JS object are dropped).

## Trace coverage
`MakeAppointmentAction` is stubbed (resolves after 100 ms, appointmentId `1234:<patient name>` so the trace shows `input.patientInfo` reached the
actor, fixed date); the same stub is ported in `machine_test.go`. Scripts live in `scripts/trace/workflow-event-based-service/`.
- `testdata/appointment.golden.json` (`appointment.ts`): initial `Idle` (context all null); unknown event ignored in `Idle`; `MakeVetAppointment` ->
  `MakeVetAppointmentState` (patientInfo assigned, child `0.VetAppointmentWorkflow.MakeVetAppointmentState`); still pending at 30 ms; a second
  `MakeVetAppointment` and an unknown event ignored while invoking (patientInfo unchanged); `onDone` -> `Idle` (appointmentInfo assigned, no
  children); second round replaces patientInfo and, on its `onDone`, appointmentInfo.
- `testdata/workflow-event-based-service.stdout.txt` (`workflow-event-based-service.stdout.ts`): the real `main.ts` run with real timers
  (about 2 s) and `Date` pinned; two console.log blocks, no `workflow completed` line.
- Every state and both transitions are visited. The machine has no guards, delays or `onError`.
