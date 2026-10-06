# workflow-greeting

Ported from `references/xstate/examples/workflow-greeting/main.ts` (serverless workflow "greeting").

- `machine.go`: the machine (`greeting`: `Greet` invokes `greetingFunction` with `event.input.person.name`, `onDone` assigns `greeting` and goes to `Greeted`, final with `output` from the context), the `greetingFunction` actor (`GreetingFunction`) and the entry (`Run`, `RunWith`).
- `main.ts` builds the machine and runs a demo actor at import time (input `{person: {name: 'Jenny'}}`); the Go `Run(w)` is that entry. `RunWith` injects the 1000 ms delay for tests.
- `event.input` of the invoke `input` function is the `xstate.init` event: Go reads it as `xs.InitEvent.Input`.

## Not ported
- `package.json`, `pnpm-lock.yaml`, `tsconfig.json`, `.gitignore`: Node/pnpm/vite-node setup.
- `await new Promise(setTimeout(..., 1000))`: replaced by a `time.Timer` honouring `ctx.Done()` (the Go promise actor's cancellation signal); the JS promise has no cancellation.
- `console.log`: written to the `io.Writer` passed to `Run`; the root snapshot `output` is `undefined` in JS (the machine has no root `output`; the final state's `output` only feeds a parent's done event), so `workflow completed undefined` is printed by `formatOutput` for a nil output.
- Rejection of `greetingFunction`: the machine has no `onError`; in JS an unhandled promise rejection is reported asynchronously and crashes the process, which cannot be recorded in a golden trace.

## Trace coverage
- `testdata/workflow-greeting.golden.json` (`scripts/trace/workflow-greeting/workflow-greeting.ts`): input `{person: {name}}`, initial `Greet` with context `{}` (`greeting: undefined`) and the invoked child, still pending after 30 ms, `onDone` -> assign `greeting` -> `Greeted` (status `done`, no children, root output undefined) after the stubbed 100 ms `greetingFunction`. This is every state and the only transition; the machine has no guards or delays.
- `testdata/workflow-greeting.stdout.txt` (`workflow-greeting.stdout.ts`): the real entry, `workflow completed undefined`.
- `TestFinalOutputFromContext` (Go only): final context carries the greeting; root output stays nil.
