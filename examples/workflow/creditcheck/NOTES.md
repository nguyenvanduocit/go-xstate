# workflow-credit-check

Ported from `references/xstate/examples/workflow-credit-check/main.ts` (serverless workflow "perform customer credit check").

- `machine.go`: the machine (`customercreditcheck`: `CheckCredit` invokes `callCreditCheckMicroservice`, `after PT15M` -> `Timeout`, `onDone` assigns `creditCheck` -> `EvaluateDecision`, `always` Approved -> `StartApplication`, Denied -> `RejectApplication`, otherwise -> `RejectApplication`; both invoke states `onDone` -> `End`, final), the three promise actors (`DefaultActors`) and the entry (`Run`, `RunWith`).
- `inspect.go`: renders `console.log(label, object)` the way Bun prints it, for the `Received event` lines and the actor log lines of the entry.
- `main.ts:137-165` builds a demo actor at import time; the Go `Run(w)` is that entry. `RunWith` injects the 1000 ms delay of the two "fake 1s" actors for tests.

## Not ported
- `package.json`, `pnpm-lock.yaml`, `tsconfig.json`, `.gitignore`: Node/pnpm/vite-node setup (`package.json` scripts `start`/`build`/`preview` use vite).
- `await new Promise(setTimeout(..., 1000))` (`main.ts:41`, `main.ts:54`): replaced by a `time.Timer` honouring `ctx.Done()` (the Go promise actor's cancellation signal); the JS promise has no cancellation.
- `console.log` (`main.ts:28`, `:39`, `:52`, `:150`, `:157`): written to the `io.Writer` passed to `Run`. Objects are rendered by `logInspect` in Bun's `console.log` layout (one property per line, two-space indent, trailing commas, double-quoted strings); it covers nested objects of strings and numbers, not arbitrary JS values. `workflow completed undefined` is printed literally because the machine has no output (`snapshot.output` is `undefined` in JS).
- The `inspect` callback (`main.ts:148-152`) is ported as `xs.WithInspect` filtered to `xs.InspectEvent`; event objects are rebuilt as ordered views (`initView`, `resolveView`, `doneActorView`) because Go events are typed values, not JS objects with insertion order.
- Typing of `context.creditCheck` as `{ decision } | null` (`main.ts:16-18`): at runtime the whole actor output is stored (`main.ts:84`), so Go types it as `*CreditCheck` with the four output fields and the golden traces show all four.
- A credit check that resolves with `undefined` (the `?.` in the guards, `main.ts:98`, `:102`): the Go output type is a struct, so `creditCheck` is never `undefined`. The `Review` trace covers the same fall-through branch with a decision that is neither `Approved` nor `Denied`.
- Rejection of any of the three actors: no `onError` exists, so JS reports an unhandled error asynchronously and cannot be recorded as a golden trace.

## Trace coverage (recorded by `scripts/trace/workflow-credit-check/*.ts` through `./gen.sh workflow-credit-check`)
All three actors are replaced by deterministic stubs (`lib/load.ts`, Go twins in `machine_test.go`): each resolves after 100 ms; the credit check of `timeout` never resolves. The traces use `SimulatedClock` for `after` and real waits for the promises.
- `testdata/approved.golden.json` (`approved.ts`): input -> context via `context: ({input})`, initial `CheckCredit` with the invoked child, an unknown event ignored, credit check pending, `onDone` + assign + `always` first guard -> `StartApplication`, `onDone` -> `End` (status `done`), then `advance` 15 min: the `PT15M` timer was cancelled on exit, still `End`.
- `testdata/denied.golden.json` (`denied.ts`): second guard (`Denied`) -> `RejectApplication` -> `End`.
- `testdata/review.golden.json` (`review.ts`): decision `Review`, both guards fail, unguarded third transition -> `RejectApplication` -> `End`.
- `testdata/timeout.golden.json` (`timeout.ts`): never-resolving credit check, `advance` 899999 ms stays in `CheckCredit`, `advance` 1 ms fires `PT15M` -> `Timeout` with the child stopped (no children), unknown event ignored.
- `testdata/workflow-credit-check.stdout.txt` (`workflow-credit-check.stdout.ts`): the real `main.ts` run with real 1 s delays: seven `Received event` blocks (init of the root and of each child, `xstate.promise.resolve`, `xstate.done.actor.0.customercreditcheck.*`), the two actor log lines of the approved path (`calling credit check microservice`, `starting application workflow`) and `workflow completed undefined`. `TestStdout` compares it byte for byte.
- Every state (`CheckCredit`, `EvaluateDecision` transient, `StartApplication`, `RejectApplication`, `End`, `Timeout`), every `always` branch and both `after` outcomes (fires, cancelled) are visited. `Run`'s inspect order (init of the child before the actor's log line) is covered only by the stdout test.
