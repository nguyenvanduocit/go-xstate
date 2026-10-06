# workflow-event-greeting

Ported from `references/xstate/examples/workflow-event-greeting/main.ts` (serverless workflow "event-based greeting").

- `machine.go`: the machine (`event-greeting`: `Waiting` -`greet`-> `Greet`, which invokes `greetingFunction` with `event.greet.name`; `onDone` assigns `greeting` and goes to the final state `Greeted`), the `greetingFunction` actor (`GreetingFunction`), the `GreetEvent` constructor and the entry (`Run`, `RunWith`).
- `main.ts` builds the machine and runs a demo actor at import time; the Go `Run(w)` is that entry (sends `greet` with `Jenny`). `RunWith` injects the 1000 ms delay for tests.

## Not ported
- `package.json`, `pnpm-lock.yaml`, `tsconfig.json`, `.gitignore`: Node/pnpm/vite-node setup.
- `await new Promise(setTimeout(..., 1000))`: replaced by a `time.Timer` honouring `ctx.Done()` (the Go promise actor's cancellation signal); the JS promise has no cancellation.
- `console.log('workflow completed', output)`: written to the `io.Writer` passed to `Run`; `actor.getSnapshot().output` is `undefined` in JS (the root machine has no `output`; the `Greeted` state's `output` only feeds a parent's done event), so `formatOutput` prints `undefined` for a nil output.
- Rejection of `greetingFunction`: the machine has no `onError`; in JS an unhandled rejection is reported asynchronously and cannot be recorded in a golden trace (and the actor never rejects).
- TypeScript types (`setup({ types })`): Go uses the typed `Context`, `GreetingInput`, `GreetingOutput`, `Output` structs and an `xs.E` event built by `GreetEvent`.

## Trace coverage
- `testdata/workflow-event-greeting.golden.json` (`scripts/trace/workflow-event-greeting/workflow-event-greeting.ts`, the example's real 1000 ms `greetingFunction`): initial `Waiting` with context `{}`; unknown event ignored in `Waiting`; `greet` -> `Greet` with the invoked child `0.event-greeting.Greet`; a second `greet` ignored in `Greet`; still pending after 300 ms; `onDone` -> `Greeted` (status `done`, context `{greeting: "Hello, Jenny!"}`, no children) after 1000 ms; `greet` ignored once done. This is every state and the only transition; the machine has no guards or delays.
- `testdata/workflow-event-greeting.stdout.txt` (`workflow-event-greeting.stdout.ts`): the real entry, `workflow completed undefined`.
- `TestGreeted_ContextCarriesGreeting` (Go only): the greeting is in context and root output is nil, with a shortened delay.
