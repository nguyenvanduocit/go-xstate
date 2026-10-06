# workflow-applicant-request

Ported from `references/xstate/examples/workflow-applicant-request/main.ts` (Serverless Workflow "applicant request decision").

- `machine.go`: the `workflow` machine (`NewMachine(w, delay)` / `Machine()`; the real actors print to `w` and wait `delay`, 1 s in JS), and `Run` / `RunWith`, the entry of `main.ts` (actor with the default applicant, `complete` subscriber, stdin lines sent as events).

## Not ported
- `package.json`, `pnpm-lock.yaml`, `tsconfig.json`, `.gitignore`: Node/vite-node/pnpm setup.
- Raw `process.stdin` `'data'` chunks: Go reads lines (`bufio.Scanner`) and trims each, so a chunk holding several lines becomes several events; an interactive terminal delivers one line per chunk, so behaviour is the same there.
- `RunWith` returns when the workflow completes, when stdin has ended and no invoked actor is pending (the point where Node would exit), or on context cancellation. Node keeps running while stdin stays open; `Run` does the same until the workflow completes.
- `console.log('workflow completed', undefined)` prints `undefined`; Go prints the same text for a nil output (`formatOutput`).
- The JS entry hardcodes an adult applicant (age 22), so the real stdout recording covers only the `StartApplication` path. The rejection path of `Run` is exercised by `TestRunMinor` without a JS recording (expected text derived from the actors' fixed messages).

## Trace coverage
All three scripts live in `scripts/trace/workflow-applicant-request/` and drive `workflow` with deterministic stubs (`provide`, each settles after 50 ms so a snapshot taken right after `send` still shows the invoking state; `wait` steps are 75 ms) for the two 1 s promise actors, importing `main.ts` with its console output muted (it starts a demo actor and a stdin listener at import).
- `testdata/adult.golden.json` (`adult.ts`, age 18): unknown event ignored in `CheckApplication`; `Submit` with `isOver18` true (boundary age 18) -> `StartApplication` (child invoked); `Submit` ignored there; `onDone` -> `End` (status done, no children); `Submit` ignored once done.
- `testdata/minor.golden.json` (`minor.ts`, age 17): `Submit` with `isOver18` false -> `RejectApplication`; `Submit` ignored; `onDone` -> `End`.
- `testdata/start-error.golden.json` (`start-error.ts`, age 30): `StartApplication` actor rejects -> `onError` -> `RejectApplication` -> `onDone` -> `End`.
- `testdata/workflow-applicant-request.stdout.txt` (`workflow-applicant-request.stdout.ts`): the real `main.ts` with the stdin event `Submit` emitted; real actors print started/completed, then `workflow completed undefined`.
- All 4 states, both guard branches, `onDone` of both actors and `onError` of `StartApplication` are covered. `RejectApplication` has no `onError`, so there is no error branch there.
