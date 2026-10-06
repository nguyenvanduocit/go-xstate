# mongodb-persisted-state

Ported from `references/xstate/examples/mongodb-persisted-state/` (`donutMachine.ts`, `TaskQueue.ts`, `main.ts`).

| JS | Go |
| --- | --- |
| `donutMachine.ts` | `machine.go` (`DonutMachine`) |
| `TaskQueue.ts` | `taskqueue.go` (`TaskQueue`) |
| `main.ts` (connect, restore, persist after every snapshot, print state and next events, stdin events) | `run.go` (`Run`) over the `Store` interface in `store.go` |
| `MongoClient` / `donut-maker.donuts` | `mongo.go` (`MongoStore`, build tag `mongodb`); in-memory fake `memstore_test.go` |
| `yarn ts-node ./main.ts` | `cmd/donut/main.go` (build tag `mongodb`) |

## Not ported
- `README.md`, `pnpm-lock.yaml`, `package.json`, `tsconfig.json`, `.gitignore`: documentation and JS toolchain.
- `process.stdin.on('data')` chunk semantics: JS sends one event per data chunk; Go sends one event per line (`bufio.Scanner`). Same for interactive use.
- JS keeps the process alive until killed; `Run` also returns at stdin EOF. After the actor completes, `Run` returns; the donut machine never completes (`serve` goes back to `ingredients`), so the `workflow completed` branch of `main.ts` (`run.go` `Complete` callback) has no trace and is untested.
- `console.log(object)` uses Node's `util.inspect` layout; Go prints objects as JSON with sorted keys (`jsonText`). The JS recording script prints the same way (see below), so the two stay comparable.
- Async task execution: JS tasks are promises on the event loop; `TaskQueue.AddTask` runs tasks on the calling goroutine (nested `AddTask` only enqueues, as in JS). A failed save is printed as `error details:` and the run continues; in JS a rejected task is an unhandled rejection.
- The real MongoDB adapter (`mongo.go`, `cmd/donut/main.go`) compiles only with `-tags mongodb` and was NOT exercised against a database (no mongod/Docker daemon available). It was compiled and vetted in a scratch copy of the module after `go mod tidy`. The frozen `examples/go.mod`/`go.sum` list `go.mongodb.org/mongo-driver/v2` as indirect and carry no go.sum entries for its transitive dependencies (klauspost/compress, xdg-go/scram, xdg-go/stringprep, youmark/pkcs8, golang.org/x/crypto, golang.org/x/sync, golang.org/x/text), so building without the tag fails: run `cd examples && go mod tidy` (needs network) before using the tag.

## Trace coverage
`testdata/donut.golden.json` (`scripts/trace/mongodb-persisted-state/donut.ts`): every state (`ingredients`, `directions.makeDough`, `directions.mix` with both regions in `mixing` and `mixed`, `directions.allMixed` -> `directions` onDone -> `fry`, `flip`, `dry`, `glaze`, `serve`), every transition (`NEXT`, `MIXED_DRY`, `MIXED_WET`, `ANOTHER_DONUT`), both `onDone` handlers (`mix`, `directions`), ignored events (`ANOTHER_DONUT` in `ingredients`, `NEXT`/`MIXED_*` in wrong states, an unknown event) and the loop `serve` -> `ingredients`. The machine has no guards, delays or invokes.

`testdata/donut-restored.golden.json` (`donut-restored.ts`): an actor restored from a persisted snapshot taken inside `mix` (`mixDry` done, `mixWet` mixing) continues to `fry`, `flip`, `dry`, `glaze`, `serve`, `ingredients`. The persisted snapshot is the golden's `input`; `TestPersistedSnapshotMatchesJS` checks the Go actor persists the same snapshot at that point.

`testdata/mongodb-persisted-state.stdout.txt` (`mongodb-persisted-state.stdout.ts`): the real, unmodified `main.ts` run against a fake `mongodb` module (one document, `modifiedCount` 0 when unchanged, `undefined` stored as null as the Node driver does), in three sessions on one database: (1) empty database, upsert, events including an unknown one and an empty line (re-print without a save); (2) restart restoring the persisted state and walking to `ingredients`; (3) failing `connect` (`error details:`). `TestRunMatchesJS` compares the Go output byte for byte. The script replaces `console.log` with a string-joining formatter that prints objects as sorted-key JSON and Errors as `Name: message`.

Go-only tests (no JS counterpart): `FindOne` error, failing save, restoring from a store written by a previous `Run`, `TaskQueue` ordering.

## Library observation
Persisted snapshots carry `"output":null,"error":null` where JS omits the keys; see `docs/porting/notes/examples-lib-findings.md` (mongodb-persisted-state). Round trip is unaffected.
