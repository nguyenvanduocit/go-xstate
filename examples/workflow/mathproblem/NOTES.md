# workflow-math-problem

Ported from `references/xstate/examples/workflow-math-problem/main.ts` (serverless workflow "solving math problems").

- `machine.go`: the machine (`math-problem`: `Solve` invokes `batchMathFunction` with `event.input.expressions`, `onDone` assigns `results`, `Solved` final with `output`), the actor (`BatchMathFunction`, body in `SolveBatch`) and the entry (`Run`, `RunWith`).
- `main.ts` builds the machine and runs a demo actor at import time; the Go `Run(w)` is that entry. `RunWith` injects the 1000 ms delay for tests.
- `Context` has a custom `MarshalJSON`: nil `Results` is an absent key (JS `undefined`), an empty slice is `[]`.

## Not ported
- `package.json`, `pnpm-lock.yaml`, `tsconfig.json`, `.gitignore`: Node/pnpm/vite-node setup (the `build`/`preview` scripts reference vite but there is no HTML entry).
- `await new Promise(setTimeout(..., 1000))`: replaced by a `time.Timer` per problem honouring `ctx.Done()` (the Go promise actor's cancellation signal); the JS promise has no cancellation.
- `console.log`: written to the `io.Writer` passed to `Run`. `actor.getSnapshot().output` is `undefined` in JS (the machine has no root `output`, so the `Solved` state's `output` is never surfaced: `getMachineOutput` returns early, `references/xstate/packages/core/src/stateUtils.ts:1146`), so `workflow completed undefined` is printed by `formatOutput` for a nil output. The Go engine matches (snapshot `output` is null in the golden trace and the replay passes).
- The Solved state's `output` (`{ results }`): ported as `Output`, never observable in a snapshot.
- Rejection of `batchMathFunction`: the machine has no `onError`; the real actor cannot reject, and in JS an unhandled rejection would crash the process, which cannot be recorded in a golden trace.

## Trace coverage
- `testdata/workflow-math-problem.golden.json` (`scripts/trace/workflow-math-problem/workflow-math-problem.ts`): input `{expressions: [4 items]}`, initial `Solve` with the invoked child and `results` absent, still pending after 20 ms, `onDone` -> `Solved` (status `done`, `results` assigned, no children) after the stubbed 50 ms batch. This is every state and the only transition; the machine has no guards or delays.
- `testdata/empty.golden.json` (`.../empty.ts`): input `{expressions: []}`, same path, `results` becomes `[]` (not absent).
- `testdata/workflow-math-problem.stdout.txt` (`workflow-math-problem.stdout.ts`): the real entry, four `solving ...` lines then `workflow completed undefined`.
- Go only: `TestSolveBatch_Concurrent` (problems wait concurrently), `TestSolveBatch_Cancelled` (ctx abandons the batch), `TestContext_MarshalJSON`.
- The JS trace scripts replace `batchMathFunction` with a 50 ms stub having the same `Solved <problem>` mapping; the Go trace tests use a 50 ms stub around `SolveBatch`, so the real actor's body is exercised by `TestStdout` and the Go-only tests.
