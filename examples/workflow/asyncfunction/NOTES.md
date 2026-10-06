# workflow-async-function

Ported from `references/xstate/examples/workflow-async-function/main.ts` (serverless workflow "async function invocation").

- `machine.go`: the machine (`async-function-invocation`: `Send email` invokes `sendEmail`, `onDone` -> `Email sent`, final), the `sendEmail` actor (`SendEmail`) and the entry (`Run`, `RunWith`).
- `main.ts` builds the machine and runs a demo actor at import time; the Go `Run(w)` is that entry. `RunWith` injects the 1000 ms delay for tests.

## Not ported
- `package.json`, `pnpm-lock.yaml`, `tsconfig.json`, `.gitignore`: Node/pnpm/vite-node setup.
- `await new Promise(setTimeout(..., 1000))`: replaced by a `time.Timer` honouring `ctx.Done()` (the Go promise actor's cancellation signal); the JS promise has no cancellation.
- `console.log`: written to the `io.Writer` passed to `Run`; `actor.getSnapshot().output` is `undefined` in JS, so `workflow completed undefined` is printed by `formatOutput` for a nil output.
- Rejection of `sendEmail`: the machine has no `onError`; in JS an unhandled promise rejection is reported asynchronously and crashes the process, which cannot be recorded in a golden trace.

## Trace coverage
- `testdata/workflow-async-function.golden.json` (`scripts/trace/workflow-async-function/workflow-async-function.ts`): input `{customer}` -> context via `context: ({input})`, initial `Send email` with the invoked child, still pending after 30 ms, `onDone` -> `Email sent` (status `done`, no children) after the stubbed 100 ms `sendEmail`. This is every state and the only transition; the machine has no guards or delays.
- `testdata/workflow-async-function.stdout.txt` (`workflow-async-function.stdout.ts`): the real entry, `Sending email to ...`, `Email sent to ...`, `workflow completed undefined`.
- `TestSendEmail_StoppedActorAbandonsPromise` (Go only): stopping the actor while `sendEmail` is pending prevents `Email sent`.
