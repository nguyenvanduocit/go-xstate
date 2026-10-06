# Architecture

The engine is package `xstate`, imported as
`github.com/nguyenvanduocit/go-xstate/xstate`. It follows the local XState
5.33.2 source in `references/xstate/packages/core/src`.

## Engine file map

All Go files in this table are under `xstate/`.

| Go files | Responsibility and upstream source |
|---|---|
| `config.go`, `event.go` | Public configuration and event types |
| `machine.go`, `statenode.go`, `definition.go`, `setup.go` | Machine construction, state nodes, definitions, and setup (`StateMachine.ts`, `StateNode.ts`, `setup.ts`) |
| `transition_config.go`, `route.go` | Transition configuration and route targets |
| `machine_logic.go` | Machine actor logic: initial snapshot, transition, persist, and restore |
| `macrostep.go`, `microstep.go` | Transition execution (`stateUtils.ts`) |
| `transition_select.go`, `transition_domain.go`, `state_entry.go` | Transition selection, conflicts, entry/exit sets, and history (`stateUtils.ts`) |
| `statevalue.go`, `snapshot.go` | State values and snapshots (`stateUtils.ts`, `State.ts`) |
| `action.go`, `action_exec.go`, `spawn.go`, `guard.go` | Action definitions and execution, child spawning, and guards |
| `actor.go`, `actor_options.go`, `mailbox.go`, `observer.go` | Actor lifecycle, options, event queues, and subscriptions (`createActor.ts`, `Mailbox.ts`) |
| `system.go`, `scheduler.go`, `sync.go` | Actor registry, promise settlement, delayed events, and locking |
| `logic.go`, `promise_actor.go`, `callback_actor.go`, `observable_actor.go`, `transition_actor.go` | Common actor logic and concrete actor kinds (`actors/*.ts`) |
| `transition.go`, `promise.go`, `wait.go` | Pure transitions, `ToPromise`, and `WaitFor` |
| `clock.go`, `inspect.go`, `select.go`, `assert.go`, `map.go` | Clocks, inspection, selection, assertions, and mapping helpers |
| `persist.go`, `json.go`, `machine.schema.json` | Context persistence and JSON machine validation/revival; the schema is embedded by `json.go` |

`store/`, `graph/`, and `scxml/` are separate packages. Tests live beside the
implementation in each package. SCXML fixtures are in `scxml/testdata/`.
`examples/` is a nested module; its tests must run separately from the root
module. `scripts/test.sh` checks both modules.

## Typed API and engine internals

`StateNode` is not generic. The transition engine works through the internal
`anyMachineSnapshot` interface; `StateMachine[C].newSnapshot` constructs typed
`MachineSnapshot[C]` values. `ActionFunc[C]`, `GuardFunc[C]`, `NewExpr[C]`, and
`Assign[C]` adapt typed callbacks to the engine's internal arguments.

An `Actor[S]` embeds the non-generic `actorCore`. Actor kinds implement
`logicImpl`, including `newActorRef`, so the engine can create typed child actors
through the common interface. Public extension points include `Logic[S]` and
the `Implementations` maps.

`Compile` adds static validation and returns tagged `ConfigError` values; the
legacy constructor keeps its panic API. `NewTask` / `InvokeTask` lower typed
input/output callbacks to existing promise invocations. `Await` is the blocking,
context-based form of `WaitFor(...).Wait()`. These additions share the existing
execution model; see the [Go API guide](GO_API.md).

## Concurrency and ordering

`Send` processes transitions synchronously: when it returns, the actor snapshot
reflects that event and synchronous child activity. Each actor system has a
reentrant lock. A per-actor mailbox queues reentrant sends and drains them on the
calling goroutine. Actions and observers run while the system lock is held.
They may reenter the API on that goroutine, but must not wait for another
goroutine that needs the same system lock.

`SendTo` between built-in actors in separate root systems delivers under the
destination lock after the calling goroutine releases its outermost system
lock. Delivery and synchronous replies finish before that outermost actor call
returns. Foreign snapshot reads inside actions or observers do not yet see the
send; source snapshot notifications precede delivery. Concurrent calls may
interleave after unlocking. Same-system sends keep their reentrant behavior.
Direct calls to a foreign actor's `Send` or `GetSnapshot` still acquire nested
locks and can deadlock when reciprocal callbacks make those calls.

Promise bodies run on goroutines. Callbacks, observable sources, and timers take
the system lock when delivering events. `SimulatedClock` fires timers on the
goroutine calling `Increment`. The lock protects engine operations; callers
must still protect mutable data they share with async functions.

`asyncTracker` in `system.go` sorts pending promise results by start order and
briefly waits for earlier promises started within a one-millisecond window.
This approximates JavaScript microtask ordering; it is not a JavaScript event
loop. Tests that need a particular async boundary use explicit signals.

`sync.go` tracks the current system and custom-action flag by goroutine ID,
parsed from the runtime stack header. This supports reentrancy and routes
warning and unhandled-error callbacks to the current system.

`States` preserves declaration order. Go maps do not preserve insertion order,
so APIs that enumerate map-backed event names return sorted values. Context
updates should return a new value and copy mutable maps or slices before edits.

## Compatibility records

The [porting guides](porting/core.md) document API translations. Files under
`porting/manifest/` and `porting/notes/` retain test provenance and historical
findings, including obsolete chunk tags and line numbers. Their historical run
counts are not a current test report. Use `scripts/test.sh` for the current tree.
