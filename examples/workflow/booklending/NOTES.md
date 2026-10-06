# workflow-book-lending

Ported from `references/xstate/examples/workflow-book-lending/main.ts` (serverless workflow "book lending").

- `machine.go`: the `workflow` machine (`NewMachine(log, d)`), its six promise actors (`Actors`), context/input/output types with `json` tags equal to the JS keys, and the `BookLendingRequest` event builder.
- `run.go`: the entry of `main.ts` (`Run`, `RunWith`): one actor, a subscriber printing every context, `workflow completed ...` on completion, then the `bookLendingRequest` event. Includes the Bun-style `console.log` formatter.

## Not ported
- `package.json`, `pnpm-lock.yaml`, `tsconfig.json`, `.gitignore`: Node/vite-node/pnpm setup (`cockatiel` is declared in package.json but never imported by main.ts).
- `delay(ms, errorProbability)` (`main.ts:3-14`): every call passes 1000 and the default probability 0, so the `Math.random()` rejection with `{ type: 'ServiceNotAvailable' }` never fires; Go keeps only the wait, which honours `ctx.Done()` (the Go promise actor's cancellation signal; the JS promise has no cancellation).
- `console.log` of actor inputs and contexts: Go prints through `LogFunc` / the `io.Writer` of `Run`, rendered by `inspect` in Bun's `console.log` format (multi-line objects, two-space indent, trailing commas, double-quoted strings, `null`). `inspect` handles only what this example prints (structs, maps, strings, scalars, nil), not Node's line-width folding or escaping rules.
- Ordering of the printed lines: JS runs a promise actor's body synchronously when its state is entered, so `Starting ...` precedes the subscriber's print of that snapshot. Go promise bodies run on a goroutine, so `console.snapshot` (`run.go`) waits until every actor newly invoked in the snapshot has logged before printing. A snapshot containing a child with the same key as the previous snapshot is not counted as new (not reachable on the default path).
- `Run` returns when nothing is pending (no running child, not in `Sleep two weeks`), mirroring the Node process exiting when the event loop is empty; Node keeps running while a timer is pending. With the default actors the workflow never reaches `workflow completed` (see below), so the real stdout recording ends without that line.
- `event.book` is copied as `{title, id}`; the JS `{ ...event.book, status: 'unknown' }` would also copy any extra keys of the event's book object. `event.lender` is ignored by the machine in JS as well (`main.ts:59-65` assigns only `book`), so `context.lender` stays `null` in every trace.
- A rejected actor: `Get Book Status` and the other invokes have no `onError`; in JS the rejection is an unhandled error that crashes the process, which cannot be recorded in a golden trace.
- Type-level declarations (`types: {} as {...}`, `interface Lender`): carried by the Go structs.

## Behaviours worth knowing (all verified by the JS recording)
- `after: { PT2W: ... }` has no matching entry in `delays`. xstate resolves an unknown delay name to `undefined` and queues the after event immediately (`references/xstate/packages/core/src/actions/raise.ts:66-79`), so `Sleep two weeks` is left in the same macrostep and the "two weeks" never elapse. `sleep-unresolved-delay.golden.json` records that; `onloan-hold-clock.golden.json` provides `delays: { PT2W: 1000 }` to exercise the real timer. The Go engine matches both (`action.go:196-205`).
- `Check Out Book` has no `onDone`, so after its final child `End` the machine stays `active` with value `{ "Check Out Book": "End" }` and never completes; only the top-level `End` (unknown status, `Cancel Request`) is a final state of the machine.

## Trace coverage
Scripts in `scripts/trace/workflow-book-lending/` import `main.ts` with its console output muted (it starts a demo actor at import) and drive `workflow` with deterministic stubs: every promise actor settles after 80 ms (a snapshot taken right after `send` still shows the invoking state), `Get status for book` answers from a fixed list. `wait` steps are 120 ms after an event and 80 ms between hops, so each lands 40 ms after exactly one actor hop.
- `testdata/onloan-hold-clock.golden.json` (SimulatedClock for `after`): `holdBook` ignored in `Book Lending Request` and `Get Book Status`; request -> `Get Book Status` (`status: unknown`); `onloan` -> `Book Status Decision` always[0] -> `Report Status To Lender` -> `Wait for Lender response` (request ignored there); `holdBook` -> `Request Hold` -> `Sleep two weeks`; `advance 999` stays, `advance 1` fires `PT2W` -> `Get Book Status` again; `available` -> always[1] -> `Check Out Book.Checking out book` -> `Notifying Lender` -> `Check Out Book.End`; `holdBook` ignored.
- `testdata/decline.golden.json`: `onloan` ... `Wait for Lender response` -> `declineBookhold` -> `Cancel Request` -> top-level `End` (status done); request ignored once done.
- `testdata/unknown-status.golden.json`: status `lost` -> always[2] (no guard) -> `End` (done).
- `testdata/sleep-unresolved-delay.golden.json` (no clock): `holdBook` -> `Request Hold` -> `Sleep two weeks` is left at once (unresolved delay name) -> `Get Book Status` -> `available` -> `Check Out Book` chain.
- `testdata/workflow-book-lending.stdout.txt` (`workflow-book-lending.stdout.ts`): the real `main.ts` with its 1 s actors; `TestStdout` compares it with `RunWith` using 10 ms actors.
- All 12 states (including the three of `Check Out Book`), all three `always` branches, every `on` event incl. ignored ones, every `onDone`, and `after` with and without a delay implementation are covered. `TestRunContextCancel` (Go only) checks that `RunWith` returns on cancellation while an actor is pending.
