# express-workflow

Ported from `references/xstate/examples/express-workflow/` (`machine.ts`, `index.ts`).

- `machine.go`: `machine.ts` (traffic light: green -> yellow -> red -> green, `cycles` counted on red -> green; the JS machine id is `'counter'`).
- `server.go`: `index.ts` as `net/http` handlers (`POST /workflows`, `POST /workflows/{id}`, `GET /workflows/{id}`, `GET /`), plus `Run`, the `app.listen` message.
- `cmd/server/main.go`: the program that listens on :4242.

## Not ported
- `README.md`, `package.json`, `pnpm-lock.yaml`, `tsconfig.json`: docs and Node/pnpm/ts-node setup.
- `express`, `body-parser`: replaced by `net/http`. Express-specific behaviour not reproduced: `Cannot GET /x` 404 page for unknown routes (Go's mux answers `404 page not found`), `X-Powered-By`/`ETag` headers, body-parser's exact error pages.
- Events without a string `type` (`{}`, no body): in JS `actor.send` throws `event.type.startsWith` inside `setTimeout`, which crashes the Node process. Go answers `400`. Covered by `TestBadEvents`, not by a golden trace (a crash cannot be recorded).
- `res.json` drops `undefined` properties; the Go handler drops nil `output` and `error` of the persisted snapshot to match.
- `Math.random` ids: random in both; the golden trace replays the ids the JS run produced.
- A real network listener is not exercised by tests (`TestRun` uses a fake `net.Listener`); `cmd/server` is compiled only.

## Trace coverage
- `testdata/machine.golden.json` (`scripts/trace/express-workflow/machine.ts`): start, TIMER x3 (green -> yellow -> red -> green, cycles 1), an unknown event, TIMER x3 again (cycles 2).
- `testdata/server.golden.json` (`scripts/trace/express-workflow/server.ts`): runs the real handlers of `index.ts` behind a fake router (`lib/fake-express.ts`, because `express` is not installed) with a fixed `Math.random` sequence. Covers `GET /`, `POST /workflows` twice, `GET` and `POST` on existing workflows (persist, restore, send, persist round trip through every state, unknown event, event with extra payload), two independent workflows, `404` on `GET` and `POST` for an unknown id.
- `testdata/express-workflow.stdout.txt` (`express-workflow.stdout.ts`): `Server listening on port 4242`.
