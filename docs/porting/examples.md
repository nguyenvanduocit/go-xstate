# Porting examples

The source is `references/xstate/examples/`, containing 49 apps. The Go ports
live in the nested module `github.com/nguyenvanduocit/go-xstate/examples`.
[`examples/manifest.tsv`](../../examples/manifest.tsv) maps each upstream name
to its Go package path and status. For example, `workflow-hello` maps to
`examples/workflow/hello/`; `7guis-counter-react` maps to
`examples/sevenguis/counterreact/`.

Read `examples/counter/` and `scripts/trace/counter/counter.ts` for a small port.
The [core](core.md) and [store](store.md) guides describe API translations.

## Package contents

Each example package contains its machine, actors, and non-UI helpers. Preserve
state IDs, event names, guards, actions, and delays. Context structs use JSON
tags matching the JavaScript keys. Use an empty struct for an empty context so
it serializes to `{}`.

Tests replay recorded JavaScript traces with `examples/internal/tracetest/`.
Fixtures in `testdata/*.golden.json` come from JavaScript runs; never write or
edit their expected values by hand. When the upstream app has a printing entry
point, provide `Run(io.Writer)` and compare its output with a recorded
`*.stdout.txt` fixture.

Each package's `NOTES.md` lists source files, covered scenarios, and omissions.
React/Vue components, hooks, HTML, CSS, and bundler configuration are outside the
Go port. Port pure reducers, selectors, and other application logic used by the
machines.

Server examples use `net/http` handlers with `httptest` coverage. Database access
uses a small interface and an in-memory test implementation. The MongoDB driver
is already required in the examples module; a compiled adapter is not evidence
of a successful live-database test. Record that limitation in `NOTES.md`.

## Record JavaScript fixtures

1. Add `scripts/trace/<upstream-name>/<trace>.ts`, importing the upstream app
   from `../../../references/xstate/examples/<upstream-name>/...` and
   `runTrace` from `../trace.ts`.
2. Use `{send: event}`, `{advance: ms}`, and `{wait: ms}` steps. `advance`
   requires `useClock: true`; `wait` uses real time. Record the input and clock
   metadata expected by the Go replay helper.
3. Replace network calls, randomness, and wall-clock reads with deterministic
   dependencies on both sides. Exercise every state and transition, both guard
   outcomes, timeouts, completion, and failure paths; record any coverage gaps
   in `NOTES.md`.
4. Add `<trace>.stdout.ts` for a printing entry point.
5. Run `./scripts/trace/gen.sh <upstream-name>` from the repository root. It uses
   the manifest to select `examples/<package>/testdata/`. A failed recorder
   leaves that recorder's existing fixture intact.

Bun is required for recording. The trace directory contains its resolver setup
in `bunfig.toml` and `preload.ts`. Check the actual resolved source when changing
the harness; [historical findings](notes/examples-lib-findings.md) include
resolver and trace-metadata observations.

## Verify a port

For `workflow-hello`, run from the repository root:

```sh
./scripts/trace/gen.sh workflow-hello
cd examples
gofmt -l workflow/hello
go vet ./workflow/hello
go test -race -count=3 ./workflow/hello
```

Replace both names using the manifest when checking another example. Set its
status to `ported` only after the package and fixtures pass these checks. The
root `scripts/test.sh` checks both Go modules.

## Report a library difference

If a faithful port differs from the JavaScript trace, keep the failing assertion
and add a minimal reproduction to `notes/examples-lib-findings.md`: upstream
evidence, Go reproduction, expected output, and observed output. Resolve the
cause before changing fixtures or claiming the example passes.

During parallel work, edit only your assigned example and trace directories.
Coordinate shared harness, manifest, and dependency changes with the owner of
those files. Do not change `examples/go.mod` or `go.sum` while another worker
owns dependency changes.
