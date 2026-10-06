# workflow-async-subflow

Ported from `references/xstate/examples/workflow-async-subflow/main.ts` (serverless workflow "async subflow invocation").

- `machine.go`: `NewOnboarding` (machine `onboarding`: `Welcome` -> `Personalize` -> `Completed`, final), `NewWorkflow` (machine
  `async-function-invocation`: `Onboard` invokes `onboarding`, `onDone` -> `Onboarded`, final), the real `prompt` actor (`Prompt`) and the
  entry (`Run`, `RunWith`).
- `main.ts` builds both machines and runs a demo actor at import time; Go `Run(w)` is that entry, reading answers from stdin.
  `RunWith(in, w)` injects the answer source for tests.

## Not ported
- `package.json`, `pnpm-lock.yaml`, `tsconfig.json`, `.gitignore`: Node/pnpm/vite-node setup.
- `readline.createInterface` + `rl.question`: replaced by `Prompt`, which prints the question to the writer without a newline (as readline
  does on a non-TTY) and reads one line per question from a `bufio.Reader`. readline drops lines that arrive while no question is pending;
  Go reads on demand, so all-at-once input also works. `rl.close()` has no Go counterpart.
- The JS `prompt` promise cannot be cancelled; Go selects on `ctx.Done()` so stopping the actor abandons the wait (the reading goroutine
  stays blocked until its next line arrives).
- `onboardingWorkflow` is not exported by `main.ts`; the JS trace scripts take it from `workflow.implementations.actors.onboarding`.
- `main.ts` has no `onError`: a failing `prompt` (end of input) is an unhandled error in JS (crashes the process, not recordable in a
  golden trace). `RunWith` returns it as an error; `TestRunWith_EndOfInput` covers it (Go only).
- Context: `name: undefined` is `Name *string` with `omitempty` (absent from JSON until assigned; assigning `""` is observable).
  The workflow has no context (`{}` in the snapshot): `WorkflowContext struct{}`.

## Trace coverage
Prompts are stubbed (answer from a table after 50 ms, throw on an unexpected question, so the exact question text is part of the trace);
the same stub is ported in `machine_test.go`. The scripts live in `scripts/trace/workflow-async-subflow/`.
- `testdata/workflow-async-subflow.golden.json` (`workflow-async-subflow.ts`): the whole workflow: `Onboard` with the invoked child
  `0.async-function-invocation.Onboard` (unknown event ignored, still pending at 20 ms and 70 ms), `onDone` -> `Onboarded` (status `done`,
  no children), unknown event ignored once done.
- `testdata/onboarding.golden.json` (`onboarding.ts`): the child alone: `Welcome` (child `0.onboarding.Welcome`), `onDone` with
  `assign` -> `Personalize` (context `{name: "Ada"}`, child `0.onboarding.Personalize`), `onDone` -> `Completed` (done). The `Personalize`
  question `Welcome Ada, press enter ...` is checked through the stub table.
- `testdata/onboarding-empty-name.golden.json` (`onboarding-empty-name.ts`): empty first answer: `name` is `""` (present), question
  `Welcome , press enter ...`.
- `testdata/workflow-async-subflow.stdout.txt` (`workflow-async-subflow.stdout.ts`): the real `main.ts` run as a child process with answers
  `Ada` and an empty line, each written after its question is printed: `What is your name?Welcome Ada, press enter to finish the onboarding processworkflow completed undefined\n`.
- Every state and every transition of both machines is visited; neither machine has guards, delays or `onError`.
