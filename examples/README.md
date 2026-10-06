# Examples

This nested Go module ports the logic from the 49 apps in
[XState examples](https://github.com/statelyai/xstate/tree/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples). The
[`manifest.tsv`](manifest.tsv) maps upstream names to Go package directories and
records whether each port is ready or pending.

| Group | Go directories |
|---|---|
| Small machines and games | `counter/`, `toggle/`, `fetch/`, `timer/`, `snake/`, and other top-level examples |
| 7GUIs | `sevenguis/` |
| Store examples | `store/` |
| HTTP and persistence examples | `server/` |
| Workflows | `workflow/` |

Each port has a `NOTES.md` describing its source, fixture coverage, and omitted
UI or external-service behavior. React/Vue components, HTML, and CSS are not
ported. Server tests use local fakes; compiling a database adapter does not
verify it against a live database.

Run from this directory:

```sh
go vet ./...
go test -race -count=1 ./...
go test -race -count=1 ./workflow/hello
```

The module replaces `github.com/nguyenvanduocit/go-xstate` with `../`, so tests
use the library in this checkout. Library actors are imported from
`github.com/nguyenvanduocit/go-xstate/xstate`.

Fixtures come from JavaScript runs. To regenerate one example from the repository
root, with Bun installed:

```sh
./scripts/trace/gen.sh workflow-hello
```

The argument is the upstream name; the manifest selects the destination
`workflow/hello/testdata/`. See the [porting guide](../docs/porting/examples.md)
before adding or changing a fixture. `internal/tracetest/` replays JSON traces
against the Go implementation.
