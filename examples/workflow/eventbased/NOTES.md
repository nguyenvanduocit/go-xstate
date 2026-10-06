# workflow-event-based

Ported from `references/xstate/examples/workflow-event-based/main.ts` (serverless workflow "event-based transitions").

- `machine.go`: the machine (`eventbasedswitchstate`: `CheckVisaStatus` waits for `visaApprovedEvent` / `visaRejectedEvent` or the `visaDecisionTimeout` delay of 1000 ms, then one of three invoking states, each `onDone` -> `End`, final), the three printing promise actors (`Handler`, `Actors`) and the entry (`Run`, `RunWith`).
- `main.ts` builds the machine and runs a demo actor at import time (sends `visaApprovedEvent` right after start); the Go `Run(w)` is that entry. `RunWith` injects the 1000 ms handler delay for tests.

## Not ported
- `package.json`, `pnpm-lock.yaml`, `tsconfig.json`, `.gitignore`: Node/pnpm/vite-node setup.
- `await new Promise(setTimeout(..., 1000))`: replaced by a `time.Timer` honouring `ctx.Done()` (the Go promise actor's cancellation signal); the JS promise has no cancellation.
- `console.log`: written to the `io.Writer` passed to `Run`; `actor.getSnapshot().output` is `undefined` in JS, so `workflow completed undefined` is printed by `formatOutput` for a nil output.
- Rejection of the handler actors: the machine has no `onError`; in JS an unhandled promise rejection is reported asynchronously and crashes the process, which cannot be recorded in a golden trace.

## Trace coverage
Recorded by `scripts/trace/workflow-event-based/{approved,rejected,timeout}.ts` (shared `lib/record.ts`: SimulatedClock for the `after` delay, deterministic 100 ms stubs for the three promise actors, real `wait` for the promises).
- `testdata/approved.golden.json`: unknown event ignored, 999 ms elapsed (still `CheckVisaStatus`), `visaApprovedEvent` -> `HandleApprovedVisa` (child invoked, `after` timer cancelled: +5000 ms does nothing), `visaRejectedEvent` ignored there, `onDone` -> `End` (status `done`), event after done ignored.
- `testdata/rejected.golden.json`: `visaRejectedEvent` -> `HandleRejectedVisa`, cancelled timer, `visaApprovedEvent` ignored there, `onDone` -> `End`.
- `testdata/timeout.golden.json`: 500 + 499 ms still `CheckVisaStatus`, +1 ms (1000) `after visaDecisionTimeout` -> `HandleNoVisaDecision`, decision event ignored there, `onDone` -> `End`.
- Together: all 5 states and all 6 transitions (2 events, the `after`, 3 `onDone`); both the event and the timeout side of the race.
- `testdata/workflow-event-based.stdout.txt` (`workflow-event-based.stdout.ts`): the real entry's console output.
- Go only: `TestDecisionCancelsTimeout` (a decision event cancels the delay: the timeout handler never prints), `TestHandler_StoppedActorAbandonsPromise` (stopping the actor prevents `workflow completed` of a pending handler).
