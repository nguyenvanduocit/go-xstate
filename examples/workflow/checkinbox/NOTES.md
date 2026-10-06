# workflow-check-inbox

Ported from `references/xstate/examples/workflow-check-inbox/main.ts` (serverless workflow "check inbox periodically").

- `machine.go`: the `checkInbox` machine (`NewMachine`), the three actors (`Schedule`, `CheckInboxFunction`, `SendTextsFunction`, bundled as `Actors` by `NewActors`), `Timings` (the hardcoded 2000 / 1000 / 100 / 500 ms) and the console entry (`Run`, `RunScaled`).
- The root `invoke` of `schedule` has input `{interval: 2000}`; `Schedule(override)` uses a non-zero `override` instead of `input.interval` so tests can shorten it.

## Not ported
- `package.json`, `pnpm-lock.yaml`, `tsconfig.json`, `.gitignore`: Node/pnpm/vite-node setup.
- `actor.subscribe({ complete() { console.log('workflow completed', ...) } })`: the machine has no final state and the schedule never stops, so `complete` is never called; the Go entry has no equivalent line.
- `main.ts` never ends (the schedule reminds every 2000 ms forever). `Run(w, window)` runs the workflow for `window` and then stops the actor; `RunScaled` does the same with one JS millisecond lasting `unit`.
- `setInterval` / `clearInterval`: `time.Ticker` plus a goroutine, stopped by the callback's cleanup function.
- `await delay(ms)`: a `time.Timer` honouring `ctx.Done()` (the Go promise actor's cancellation signal); the JS promises have no cancellation.
- `Promise.all(messages.map(async ...))`: one goroutine per message plus a `WaitGroup`; "sending text" is printed in message order before the goroutines start, as `Array.map` runs the first half of each callback synchronously.
- `console.log` goes to the `io.Writer` given to the entry / `NewActors`.

## Trace coverage
All three traces replay JS goldens recorded by `scripts/trace/workflow-check-inbox/`. The traces replace the 1000 / 100 / 500 ms delays by 100 / 20 / 100 ms (`lib/stubs.ts`); the Go test builds the same stubs.
- `testdata/manual.golden.json` (`manual.ts`, schedule actor that never fires): initial `Idle` with the `schedule` child; `reminder` -> `CheckInbox` (invoked child); `reminder` ignored in `CheckInbox`; `onDone` -> `SendTextForHighPriority` with `messages` assigned from the event output; `reminder` ignored there; `onDone` -> `Idle` with `messages` kept; unknown event ignored in `Idle`; `reminder` -> `CheckInbox` again. Every state and transition.
- `testdata/scheduled.golden.json` (`scheduled.ts`, schedule actor reminding every 300 ms): the timer-driven path, `Idle` -> `CheckInbox` -> `SendTextForHighPriority` -> `Idle` -> `CheckInbox`, with no event sent by the script.
- `testdata/workflow-check-inbox.stdout.txt` (`workflow-check-inbox.stdout.ts`): the real `main.ts` with real timers, ended at 4500 ms: `sending text Hello`, `sending text Hi`, `text sent Hello`, `text sent Hi`. The Go `TestStdout` runs the entry with every duration divided by 10.
- `TestSendTexts_StoppedActorAbandonsPromise` (Go only): stopping the actor while `sendTextsFunction` is pending prevents `text sent`.
- The machine has no guards, `after` delays or `onError`, so there is nothing else to cover. The traces use real time (`wait`), with at least 50 ms margin to each timer.
