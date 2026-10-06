# go-xstate

[![CI](https://github.com/nguyenvanduocit/go-xstate/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/nguyenvanduocit/go-xstate/actions/workflows/ci.yml)
[![Go 1.26+](https://img.shields.io/badge/Go-1.26%2B-00ADD8?logo=go&logoColor=white)](go.mod)

State machines and actors for Go, ported from **XState v5**. Define states,
events, and typed context, then run the machine as an actor with observable
snapshots. Use it to express workflow transitions, coordinate asynchronous
work, or model application state.

The repository also includes stores and atoms, graph-based test models, an
SCXML loader, **49 example ports**, and original
[chess](examples/chessmatch) and [Go / Baduk](examples/badukmatch) matches
between decision models. It follows the
[XState 5.33.2 source](https://github.com/statelyai/xstate/tree/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core).
Compatibility and Go-specific behavior are described [below](#compatibility-and-concurrency).

[Quick start](#quick-start) · [Examples](#examples) · [Packages](#packages) ·
[Go APIs](docs/GO_API.md) · [Architecture](docs/ARCHITECTURE.md) · [Porting guide](docs/porting/core.md)

## Install

Requires **Go 1.26 or later**. From a Go module:

```sh
go get github.com/nguyenvanduocit/go-xstate/xstate@latest
```

The engine lives in the `xstate` subpackage. Import it as
`github.com/nguyenvanduocit/go-xstate/xstate`, not the module root.

## Quick start

Save this as `main.go` and run `go run .`:

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

`CreateMachine` defines the behavior; `CreateActor` creates a running instance
once `Start` is called. `Send` processes an event synchronously, and
`GetSnapshot` reads its result. For changing data, replace `struct{}` with your
context type and update it through `Assign`; see the
[counter machine](examples/counter/machine.go).

For error-returning configuration and typed asynchronous tasks, use `Compile`,
`NewTask`, `InvokeTask`, and `Await`; see the [Go API guide](docs/GO_API.md).
The existing XState-style APIs remain available.

## What is included

- Hierarchical and parallel states, history states, guards, entry/exit actions,
  delayed transitions, and final states.
- Typed context and snapshots through Go generics, with `Assign` for context
  updates and `Subscribe` for observing actors.
- Child actors and actor logic created with `FromPromise`, `FromCallback`,
  `FromObservable`, or `FromTransition`.
- Snapshot persistence and restoration, actor inspection, and a simulated clock
  for controlling timers in tests.
- Stores, atoms, selectors, persistence extensions, validation, and undo/redo.
- State graph traversal, path generation, model-based testing, and SCXML loading
  with an ECMAScript evaluator.

## Packages

All four packages belong to the same Go module:

| Package | Import path | Purpose |
| --- | --- | --- |
| [`xstate`](xstate) | `github.com/nguyenvanduocit/go-xstate/xstate` | State machines, actors, actions, guards, timers, and persistence |
| [`store`](store) | `github.com/nguyenvanduocit/go-xstate/store` | Stores, atoms, selectors, and extensions |
| [`graph`](graph) | `github.com/nguyenvanduocit/go-xstate/graph` | State traversal, paths, and test models |
| [`scxml`](scxml) | `github.com/nguyenvanduocit/go-xstate/scxml` | SCXML conversion and ECMAScript datamodel execution |

## Examples

The [`examples/`](examples) directory is a separate Go module. Its local
`replace` directive uses the library in the same checkout. Most examples are
packages exercised through tests; HTTP examples also provide runnable commands.

| Start here | What it demonstrates |
| --- | --- |
| [Clef vs Jev chess](examples/chessmatch) | Two decision models playing legal chess through a state machine |
| [Clef vs Jev Go / Baduk](examples/badukmatch) | Legal placements, passes, captures, superko, and area scoring on a 9×9 board |
| [Counter](examples/counter/machine.go) | Typed context and increment/decrement actions |
| [Toggle](examples/toggle/machine.go) | Switching between states |
| [Fetch](examples/fetch/machine.go) | Asynchronous work and actor completion |
| [Stopwatch](examples/stopwatch/machine.go) | Callback actors, timers, and cleanup |
| [Store counter](examples/store/counter) | Store events and state updates |
| [HTTP workflow](examples/server/workflow) | Persisting actor state between HTTP requests |
| [Parallel workflow](examples/workflow/parallel) | Concurrent branches and completion |
| [Async subflow](examples/workflow/asyncsubflow) | Invoking a child workflow |

Clone the repository and run an example's tests:

```sh
git clone https://github.com/nguyenvanduocit/go-xstate.git
cd go-xstate/examples
go test -v ./counter
go test -v ./workflow/parallel
```

To start the HTTP workflow server from `examples/`:

```sh
go run ./server/workflow/cmd/server
```

It listens on port `4242`. Stop it with Ctrl+C. See the
[server notes](examples/server/workflow/NOTES.md) for its routes, and the
[example manifest](examples/manifest.tsv) for all 49 ports. Each example has a
`NOTES.md` recording its upstream source, test coverage, and omissions.

## Testing and CI

From the repository root:

```sh
./scripts/test.sh
```

This runs `go vet ./...` and `go test -race -count=1 ./...` in both modules.
The race detector requires CGO and a C compiler. Running `go test ./...` from
the root alone does not include the nested examples module.

For narrower checks:

```sh
go test -race -count=1 ./xstate
(cd examples && go test -race -count=1 ./counter)
(cd examples && go build -tags mongodb ./...)
```

[GitHub Actions](https://github.com/nguyenvanduocit/go-xstate/actions/workflows/ci.yml)
runs on pushes to `main`, pull requests, and manual dispatch. Separate library
and example jobs run vet, build, and race tests on Ubuntu using each module's
Go version. The example job also builds the MongoDB adapter. Test logs are
retained as artifacts for seven days, including failed test runs.

Example tests compare recorded JavaScript traces and stdout with the Go
implementation. External services use local fakes; CI does not verify a live
MongoDB deployment or a real ffprobe installation. The committed fixtures are
sufficient to run the Go suites. Regenerating them requires the upstream source
and JavaScript tooling described in the [example porting guide](docs/porting/examples.md).

## Compatibility and concurrency

This is a Go port of XState semantics. Passing the translated tests does not
establish complete compatibility with every JavaScript API. React/Vue rendering,
browser bindings, and TypeScript-only checks are outside the port's scope.
The [porting records](docs/porting/manifest) document coverage and explicit skips.

Go-specific behavior matters when integrating actors:

- Actions and observers run synchronously under the actor system lock. They
  must not wait for another goroutine that needs the same lock.
- Promise bodies run on goroutines. Protect mutable data shared with async
  functions, and copy maps or slices before changing a context value.
- Cross-root `SendTo` delivery runs after the outermost source lock is released
  and completes before that outermost actor call returns. Direct reciprocal
  calls to foreign actors can still deadlock.
- `States` preserves declaration order. Go maps do not, so APIs that enumerate
  map-backed event names return sorted values.

Read [architecture and concurrency](docs/ARCHITECTURE.md) for ordering details
and the [core porting guide](docs/porting/core.md) for API translations.

## Contributing

Add a regression test for behavior changes and run `./scripts/test.sh` before
opening a pull request. Tests live beside the implementation in `_test.go`
files. Tests translated from upstream should link the corresponding JavaScript
case; label Go-specific regressions separately. Changes to example fixtures
should follow the [fixture recording guide](docs/porting/examples.md).

## Upstream

The implementation and conformance tests follow
[Stately's XState](https://github.com/statelyai/xstate), pinned to commit
[`38dcaff`](https://github.com/statelyai/xstate/tree/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701).
The local `references/` directory used during porting is excluded from Git;
upstream links in tests identify the source used for each case.
