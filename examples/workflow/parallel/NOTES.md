# workflow-parallel

Ported from `references/xstate/examples/workflow-parallel/main.ts` (serverless workflow "parallel execution").

- `machine.go`: the `parallel-execution` machine (`NewMachine(shortDelay, longDelay)`): parallel state `ParallelExec` with regions `ShortDelayBranch` and `LongDelayBranch` (each `active` invokes its actor, `onDone` -> final `done`), `ParallelExec.onDone` -> final `Success`. `Delay` is the `shortDelay`/`longDelay` actor, `Run`/`RunWith` the entry of `main.ts`.
- `main.ts` builds the machine and runs a demo actor at import time; the Go `Run(w)` is that entry. `RunWith` injects the 1000 ms / 3000 ms delays for tests.

## Not ported
- `package.json`, `pnpm-lock.yaml`, `tsconfig.json`, `.gitignore`: Node/pnpm/vite-node setup.
- `setTimeout` inside `new Promise`: replaced by a `time.Timer` honouring `ctx.Done()` (the Go promise actor's cancellation signal); the JS promise has no cancellation.
- `console.log`: written to the `io.Writer` passed to `Run`; `actor.getSnapshot().output` is `undefined` in JS, so `workflow completed undefined` is printed by `formatOutput` for a nil output.
- Rejection of `shortDelay`/`longDelay`: the machine has no `onError`; the JS promises never reject.

## Trace coverage
- `testdata/workflow-parallel.golden.json` (`scripts/trace/workflow-parallel/workflow-parallel.ts`): shortDelay/longDelay are stubbed to 100 ms / 300 ms (`workflow.provide`), the Go test uses the same stubs. Covers: initial parallel value with both invoked children; an unknown event (no change) while pending; shortDelay resolved -> `ShortDelayBranch.done` while `LongDelayBranch` is still `active` (one child left); longDelay resolved -> both regions final -> `ParallelExec.onDone` -> `Success` (status `done`, no children); an unknown event after completion. All 5 leaf states (`active` x2, `done` x2, `Success`) and the parallel state and every transition (2 invoke `onDone`, 1 state `onDone`) are covered. The machine has no guards, delays (`after`) or events.
- `testdata/workflow-parallel.stdout.txt` (`workflow-parallel.stdout.ts`): the real `main.ts` with its 1000/3000 ms delays: `Resolved shortDelay`, `Resolved longDelay`, `workflow completed undefined`; `TestStdout` compares it with `RunWith` using 10/60 ms delays.
