# workflow-hello

Ported from `references/xstate/examples/workflow-hello/main.ts` (serverless workflow "hello world").

- `machine.go`: the machine (`helloworld`: the initial state `Hello State` is final with output `{ result: 'Hello World!' }`, so the actor is done at start) and the entry `Run(w)`.
- `main.ts` builds the machine and runs a demo actor at import time; the Go `Run(w)` is that entry.

## Not ported
- `package.json`, `pnpm-lock.yaml`, `tsconfig.json`, `.gitignore`: Node/pnpm/vite-node setup.
- `console.log('workflow completed', output)`: written to the `io.Writer` passed to `Run`; `actor.getSnapshot().output` is `undefined` in JS (the root machine has no `output`; `getMachineOutput` in `packages/core/src/stateUtils.ts` returns early, and the final state's `output` only feeds a parent's done event), so `formatOutput` prints `undefined` for a nil output.

## Trace coverage
- `testdata/workflow-hello.golden.json` (`scripts/trace/workflow-hello/workflow-hello.ts`): the start snapshot (status `done`, value `Hello State`, context `{}`, output null) and an unknown event ignored once done. This is the only state; the machine has no transitions, guards, delays or invokes.
- `testdata/workflow-hello.stdout.txt` (`workflow-hello.stdout.ts`): the real entry, `workflow completed undefined`.
