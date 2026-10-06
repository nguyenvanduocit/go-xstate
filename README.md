# go-xstate

A Go port of XState's state machine and actor runtime, based on the local
[XState 5.33.2 reference](references/xstate/packages/core). Requires Go 1.26.

The library includes hierarchical and parallel states, guards, actions, actor
invocation, timers, inspection, and snapshot persistence. Companion packages
provide stores and atoms, graph traversal and test models, and SCXML loading.

## Usage

Import the engine as `github.com/nguyenvanduocit/go-xstate/xstate`. The following
program runs against this checkout; the examples module uses a local `replace`
directive for that purpose.

```go
package main

import (
	"fmt"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

func main() {
	machine := xs.CreateMachine(xs.MachineConfig[struct{}]{
		ID:      "toggle",
		Initial: "off",
		States: xs.States{
			{Key: "off", On: map[string]xs.Transitions{
				"toggle": {{Target: "on"}},
			}},
			{Key: "on", On: map[string]xs.Transitions{
				"toggle": {{Target: "off"}},
			}},
		},
	})
	actor := xs.CreateActor(machine).Start()
	defer actor.Stop()
	actor.Send(xs.Ev("toggle"))
	fmt.Println(actor.GetSnapshot().Value) // on
}
```

Use a context struct with `Assign` for data that changes with transitions. The
[counter example](examples/counter/machine.go) shows this pattern.

## Packages and layout

| Directory | Contents |
|---|---|
| [`xstate/`](xstate) | State machine engine and actors; import `github.com/nguyenvanduocit/go-xstate/xstate` |
| [`store/`](store) | Stores, atoms, persistence, validation, and undo/redo |
| [`graph/`](graph) | State traversal, paths, and model-based tests |
| [`scxml/`](scxml) | SCXML parsing and execution with an ECMAScript evaluator |
| [`examples/`](examples) | Example ports in a separate Go module |
| [`scripts/trace/`](scripts/trace) | JavaScript scripts used to record example fixtures |
| [`docs/porting/`](docs/porting) | Translation guides, manifests, and historical findings |
| [`references/xstate/`](references/xstate) | Upstream source used for comparison |

Tests live beside each package’s implementation and use `_test.go` filenames.

## Checks

From the repository root:

```sh
./scripts/test.sh
```

This runs `go vet ./...` and `go test -race -count=1 ./...` in both the root and
`examples/` modules. Running `go test ./...` from the root alone does not include
the nested examples module. To run one suite:

```sh
go test -race -count=1 ./xstate
(cd examples && go test -race -count=1 ./counter)
```

## GitHub Actions

The [CI workflow](.github/workflows/ci.yml) runs on pushes to `main`, pull
requests, and manual dispatch. Separate library and example jobs run vet,
build, and race tests on Ubuntu with the Go version declared in each module.
The example job also builds with `-tags mongodb`.

Example tests execute the workflows and compare recorded traces and stdout.
They use local fakes for external services; CI does not start a live MongoDB
server or interactive example commands. Each job keeps its test log as an
artifact for seven days, including failed test runs. To run it manually, open
[Actions → CI](https://github.com/nguyenvanduocit/go-xstate/actions/workflows/ci.yml)
and choose **Run workflow**.

## Compatibility

The tests translate upstream runtime assertions and record explicit skips for
TypeScript-only checks, browser bindings, and upstream skipped tests. Passing
tests do not imply complete compatibility with every JavaScript API. Example
ports cover machine and application logic; React/Vue rendering and browser UI
are outside their scope. See the [example manifest](examples/manifest.tsv) for
each port's status and its `NOTES.md` for coverage and omissions.

Conformance tests link to the corresponding JavaScript testcase at the pinned
upstream commit. Generated tests also link their case data where available.
Go-only regressions are labeled separately. Example tests link the upstream
implementation and the local recorder when the upstream app has no testcase.

Go-specific choices include ordered `States` slices, sorted names where a Go map
cannot preserve JavaScript object order, and goroutines for promise bodies.
Read [architecture and concurrency](docs/ARCHITECTURE.md) before relying on
callback ordering or sharing mutable context across goroutines. The
[porting guides](docs/porting/core.md) explain API translations; historical
[findings](docs/porting/notes/examples-lib-findings.md) record observed differences.
