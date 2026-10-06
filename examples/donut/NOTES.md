# persisted-donut-maker

Ported from `references/xstate/examples/persisted-donut-maker/donutMachine.ts` (machine, `machine.go`) and `main.ts`
(console entry, `Run` in `machine.go`).

## Not ported
- `package.json`, `tsconfig.json`, `.gitignore`, `pnpm-lock.yaml`: bundler / vite-node setup.
- `main.ts` literal I/O wiring: `./persisted-state.json` becomes the `stateFile` argument, `process.stdin` becomes the
  `io.Reader` argument, `console.log` becomes the `io.Writer` argument of `Run(w, stdin, stateFile)`.
- `main.ts` treats every stdin chunk as one event type (`data.toString().trim()`); `Run` treats every line as one
  event, which is what an interactive terminal delivers. Piped multi-line chunks would be one event in JS.
- JS `fs.writeFile` is not awaited (async); Go writes synchronously before `Send` returns.
- `complete()` prints `workflow completed <output>`: ported, but the machine has no top-level final state, so neither
  runtime reaches it. `Run` unsubscribes before stopping the actor because the JS process exits without stopping it.
- `Run` returns write/marshal errors; JS ignores a failing `writeFile`.

## Differences kept visible
- Persisted snapshot JSON: Go emits `"output": null, "error": null` for an active machine where JS `JSON.stringify` drops
  the `undefined` keys. `TestSessions` removes top-level `output`/`error` keys that are null before comparing; see
  `docs/porting/notes/examples-lib-findings.md` (persisted-donut-maker). A JS-written file restores fine in Go (TestSessions
  seeds each session from the JS-written file).
- `NextEvents` uses `StateNode.OwnEvents()`, which returns each node's events sorted; JS keeps transition insertion order.
  They agree here (no state node has two own events whose insertion order is not alphabetical, and the `xstate.done.state.*`
  keys sit on distinct nodes).

## Trace coverage
`testdata/persisted-donut-maker.golden.json`, recorded by `scripts/trace/persisted-donut-maker/persisted-donut-maker.ts`:
- every state: `ingredients`, `directions.makeDough`, `directions.mix` (both regions `mixing`, `mixed`), `allMixed` (via
  `directions` onDone), `fry`, `flip`, `dry`, `glaze`, `serve`.
- every transition: `NEXT` x6 (`ingredients`, `makeDough`, `fry`, `flip`, `dry`, `glaze`), `MIXED_DRY`, `MIXED_WET`
  (both completion orders over two donuts), `mix` onDone -> `allMixed`, `directions` onDone -> `fry`, `ANOTHER_DONUT`.
- unhandled events: `ANOTHER_DONUT` in `directions`, `NEXT` in `mix`, repeated `MIXED_DRY` in a `mixed` region, `unknown`.
- No guards, delays or invokes exist in this machine.

`testdata/sessions.golden.json` and `testdata/persisted-donut-maker.stdout.txt`, recorded by running the real `main.ts`
(`sessions.ts` and `persisted-donut-maker.stdout.ts`) three times in one temp directory:
- session 1: no persisted file (`No persisted state found.`), stops mid-mix after an unknown event `BOGUS`.
- session 2: restores the mid-mix snapshot (parallel state + `mixDry` final), finishes the donut, starts another.
- session 3: restores `directions.makeDough` and moves into `mix`.
- `TestSessions` (each session seeded from the JS-written file) and `TestStdout` (sessions chained through the Go-written file)
  compare the printed text byte for byte; `TestRunIgnoresUnparsableState` covers the unparsable-file branch of the restore.
