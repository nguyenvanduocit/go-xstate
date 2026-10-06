# Go APIs

`Compile`, `NewTask`, `InvokeTask`, and `Await` provide typed task boundaries and
ordinary Go error returns on the existing XState runtime. They use the same
statechart execution, synchronous `Send`, and promise actor lifecycle as the
compatibility APIs. They do not introduce a different scheduler.

## Compile configuration with errors

```go
machine, err := xs.Compile(xs.MachineConfig[Counter]{
    Initial: "ready",
    States: xs.States{{Key: "ready"}},
})
if err != nil {
    var configErr *xs.ConfigError
    if errors.As(err, &configErr) {
        return fmt.Errorf("state %s: %w", configErr.StateID, err)
    }
    return err
}
```

`Compile` returns static construction errors, including invalid targets,
missing or invalid initial states, duplicate sibling keys or state IDs, and
unknown state/history types. Explicit history defaults are resolved during
compilation. It checks deferred initial-state errors in every
state, including currently inactive states. `ConfigError.StateID` identifies the
node being configured or resolved; `Unwrap` preserves its underlying error.

Compilation does not execute context factories, guards, actions, or invocation
inputs. It does not validate dynamic expressions, late-bound implementations,
or arbitrary runtime values. Unexpected programmer panics propagate.

`CreateMachine` retains its panic-based API and existing error messages for
XState compatibility. Config validation is performed using tagged construction
errors, so `Compile` does not catch every panic and label it a config failure.

## Describe tasks with concrete input and output

Keep I/O in ordinary Go functions:

```go
func chooseMove(ctx context.Context, turn Turn) (Decision, error) {
    return client.Choose(ctx, turn)
}
```

Bind a function to a machine using a typed task and invocation:

```go
choose, err := xs.NewTask(chooseMove)
if err != nil {
    return nil, err
}
invocation, err := xs.InvokeTask(choose, xs.Invocation[Match, Turn, Decision]{
    ID:          "choose",
    DoneTarget:  "applying",
    ErrorTarget: "failed",
    Input:       makeTurn,       // func(Match) Turn
    Done:        acceptDecision, // func(Match, Decision) Match
    Failed:      recordFailure, // func(Match, error) Match
})
if err != nil {
    return nil, err
}
```

Put the returned value in a state's `Invoke: []xs.InvokeConfig{invocation}`.
The compiler checks that the task and callbacks agree on input/output types.
Each task can be reused in several invocations. `NewTask` does not start work.
It rejects a nil function; `InvokeTask` rejects a zero task, missing callbacks,
empty ID, or empty completion/error target.

`Input`, `Done`, and `Failed` run synchronously under the actor system lock.
They must not block. The task body runs on a goroutine and receives cancellation
when its actor stops or its invoking state exits. Copy mutable input that the
machine may still use. Context updates still need to copy modified maps/slices;
the adapter does not provide automatic deep copies.

The `Failed` callback receives the original Go error, preserving `errors.Is`
and `errors.As`. Non-error rejection values are wrapped in `RejectionError`.
Nil interface input/output values are supported. The adapter checks erased
payloads internally; application callbacks do not need type assertions.

Task context currently follows `FromPromise`: it starts from a background
context and is cancelled by actor lifecycle. `Await` does not supply its context
to the task. Root context propagation and waiting for all cancelled workers to
exit remain separate runtime work.

For a runnable example, see
[`ExampleInvokeTask`](../xstate/go_api_example_test.go). Both original
[chess](../examples/chessmatch/machine.go) and
[Go / Baduk](../examples/badukmatch/machine.go) matches use these APIs.

## Wait using a context

```go
actor := xs.CreateActor(machine).Start()
defer actor.Stop()

ctx, cancel := context.WithTimeout(context.Background(), time.Second)
defer cancel()
snapshot, err := xs.Await(ctx, actor, func(s *xs.MachineSnapshot[Counter]) bool {
    return s.Status == xs.StatusDone
})
```

`Await` is the blocking equivalent of `WaitFor(ctx, actor, predicate).Wait()`.
It removes the subscription on completion or cancellation. Cancelling the wait
returns the context's cancellation cause and does not stop the actor. Stop the
actor separately when the caller owns it and no longer needs it. Actor errors
retain their identity. If the actor terminates before the predicate matches,
the wait returns an error.

Do not call `Await` from an action or observer of the same actor system; the
blocked callback holds the lock needed to make progress. The predicate must
also return promptly. Use a deadline if another goroutine may never produce the
required state. Nil context, actor, or predicate arguments return errors.

## Persisting a typed invocation

The typed adapter stores input/output payloads as JSON strings inside envelopes.
This keeps large `int64` and `uint64` values exact when intermediate snapshot
objects use `map[string]any`. Zero and nil values are preserved. In-memory
snapshots retain their Go types; restoration decodes the saved payload into its
declared type. Malformed envelopes and trailing payload data produce errors.

Input and output must support the JSON round-trip you need. Numbers inside
interface fields restore as `json.Number`; JSON cannot restore arbitrary
concrete Go types behind an interface. Marshal errors are returned, and task
snapshot decode failures reject on startup and reach the invoking machine's
`Failed` callback without running the task function.

Restoring an active task starts its function again. Supply idempotency keys for
external operations that must tolerate a restart. Functions, HTTP clients,
credentials, channels, and mutexes belong in injected dependencies, not saved
input. This is not a durable exactly-once job system.

The tests verify an active typed invocation restored through JSON, cancellation,
error identity, nil interface values, and compile-time rejection of incompatible
output types. This does not make every application context automatically
restorable; for example, the Go match still needs a complete codec for its
private board history before it can resume a saved game.
