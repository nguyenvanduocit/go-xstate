# workflow-finalize-college-app

Ported from `references/xstate/examples/workflow-finalize-college-app/main.ts` (Serverless Workflow "finalize college application").

- `machine.go`: the machine (`finalizeCollegeApplication`: `FinalizeApplication` collects `ApplicationSubmitted`, `SATScoresReceived` and `RecommendationLetterReceived` into context flags; an `always` guard moves to `FinalizingApplication` once all three are set; that state invokes `finalizeApplicationFunction`, `onDone` -> `Finalized`, final), the actor (`FinalizeApplicationFunction`) and the entry (`Run`, `RunWith`).
- `main.ts` builds the machine and runs a demo actor at import time; the Go `Run(w)` is that entry. `RunWith` injects the 1000 ms delay (between the events and inside the actor) for tests.

## Not ported
- `package.json`, `pnpm-lock.yaml`, `tsconfig.json`, `.gitignore`: Node/vite-node/pnpm setup.
- `await new Promise(setTimeout(..., 1000))` inside the actor: replaced by a `time.Timer` honouring `ctx.Done()` (the Go promise actor's cancellation signal); the JS promise has no cancellation.
- `console.log`: written to the `io.Writer` passed to `Run`; `actor.getSnapshot().output` is `undefined` in JS, so `workflow completed undefined` is printed by `formatOutput` for a nil output.
- Synchronous start of the promise executor: JS runs it up to its first `await` inside the transition, so `Starting to finalize application for 123` is printed before the subscriber prints `FinalizingApplication`. The Go promise runs on its own goroutine; `RunWith` makes the subscriber wait for that first line (an unexported `started` channel) to keep the recorded order. The machine itself is unaffected.
- Rejection of `finalizeApplicationFunction`: the machine has no `onError`; in JS an unhandled promise rejection is reported asynchronously and crashes the process, which cannot be recorded in a golden trace.

## Trace coverage
- `testdata/workflow-finalize-college-app.golden.json` (`scripts/trace/workflow-finalize-college-app/workflow-finalize-college-app.ts`, input `{applicantId: "123"}`, `finalizeApplicationFunction` replaced by a 100 ms stub on both sides, `main.ts` imported with its console output muted): initial context via `context: ({input})`; unknown event ignored; `SATScoresReceived` (guard false, 1 of 3), the same event repeated (still false), `ApplicationSubmitted` (guard false, 2 of 3), `RecommendationLetterReceived` (guard true, `always` -> `FinalizingApplication` with the invoked child); unknown event ignored while finalizing; still pending after 30 ms; `onDone` -> `Finalized` (status done, no children, no output) after the stub's 100 ms. This is every state, the only `always` guard on both sides, and the only invoke / `onDone`.
- `testdata/workflow-finalize-college-app.stdout.txt` (`workflow-finalize-college-app.stdout.ts`): the real `main.ts` with its 1000 ms timers: state values printed on subscribe and after each event, actor log lines, `workflow completed undefined`.
- `TestFinalize_StoppedActorAbandonsPromise` (Go only): stopping the actor while the actor is pending prevents `Finalized application`.
