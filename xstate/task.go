package xstate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// Task preserves the input/output relationship before the engine erases types.
type Task[I, O any] struct {
	run func(context.Context, I) (O, error)
}

// NewTask describes a cancellable task without starting it. InvokeTask binds it
// to a machine context. A nil function is rejected.
func NewTask[I, O any](run func(context.Context, I) (O, error)) (Task[I, O], error) {
	if run == nil {
		return Task[I, O]{}, errors.New("task function is required")
	}
	return Task[I, O]{run: run}, nil
}

// Invocation binds a task to a machine context and its completion transitions.
// Input, Done, and Failed run synchronously and must not block.
// Input must copy mutable data shared with the asynchronous task.
type Invocation[C, I, O any] struct {
	ID          string
	Input       func(C) I
	Done        func(C, O) C
	Failed      func(C, error) C
	DoneTarget  string
	ErrorTarget string
}

// Envelopes preserve a typed nil when I or O is itself an interface.
type taskInput[I any] struct {
	Value I `json:"value"`
}
type taskOutput[O any] struct {
	Value O `json:"value"`
}

// InvokeTask builds an invocation whose input and output types are checked at
// compile time. It uses the existing promise actor lifecycle and cancellation.
// JSON restoration requires JSON-serializable input/output and restarts an
// active task; callers are responsible for idempotent external effects.
func InvokeTask[C, I, O any](task Task[I, O], cfg Invocation[C, I, O]) (InvokeConfig, error) {
	if task.run == nil || cfg.Input == nil || cfg.Done == nil || cfg.Failed == nil {
		return InvokeConfig{}, errors.New("task and invocation callbacks are required")
	}
	if cfg.ID == "" || cfg.DoneTarget == "" || cfg.ErrorTarget == "" {
		return InvokeConfig{}, errors.New("invocation id and targets are required")
	}
	logic := &taskPromise[O]{PromiseLogic: FromPromise(func(ctx context.Context, args PromiseArgs) (taskOutput[O], error) {
		input, err := decodeTaskInput[I](args.Input)
		if err != nil {
			return taskOutput[O]{}, fmt.Errorf("invocation %q: %w", cfg.ID, err)
		}
		out, err := task.run(ctx, input.Value)
		return taskOutput[O]{Value: out}, err
	})}
	decode := func(e Event) (O, bool) {
		var zero O
		done, ok := e.(DoneActorEvent)
		if !ok || done.ActorID != cfg.ID {
			return zero, false
		}
		output, ok := done.Output.(taskOutput[O])
		return output.Value, ok
	}
	return InvokeConfig{
		ID:    cfg.ID,
		Logic: logic,
		Input: NewExpr(func(a ExprArgs[C]) any { return taskInput[I]{Value: cfg.Input(a.Context)} }),
		OnDone: Transitions{
			{Target: cfg.DoneTarget,
				Guard:   GuardFunc(func(a GuardArgs[C]) bool { _, ok := decode(a.Event); return ok }),
				Actions: Actions{Assign(func(a AssignArgs[C]) C { out, _ := decode(a.Event); return cfg.Done(a.Context, out) })}},
			{Target: cfg.ErrorTarget, Actions: Actions{Assign(func(a AssignArgs[C]) C {
				return cfg.Failed(a.Context, fmt.Errorf("invocation %q: unexpected completion payload", cfg.ID))
			})}},
		},
		OnError: Transitions{{Target: cfg.ErrorTarget, Actions: Actions{Assign(func(a AssignArgs[C]) C {
			rejected, ok := a.Event.(ErrorActorEvent)
			if !ok {
				return cfg.Failed(a.Context, fmt.Errorf("invocation %q: unexpected error event", cfg.ID))
			}
			err, ok := rejected.Error.(error)
			if !ok {
				err = &RejectionError{Value: rejected.Error}
			}
			return cfg.Failed(a.Context, err)
		})}}},
	}, nil
}

// Promise Input is type-erased after JSON restoration. In-memory invocations
// retain the envelope type; only restored JSON objects need decoding.
func decodeTaskInput[I any](value any) (taskInput[I], error) {
	if input, ok := value.(taskInput[I]); ok {
		return input, nil
	}
	var input taskInput[I]
	object, ok := value.(map[string]any)
	if !ok {
		return input, fmt.Errorf("unexpected task input %T", value)
	}
	if _, exists := object["json"]; !exists || len(object) != 1 {
		return input, fmt.Errorf("invalid task input envelope")
	}
	encoded, err := json.Marshal(object)
	if err != nil {
		return input, fmt.Errorf("encode restored task input: %w", err)
	}
	if err := json.Unmarshal(encoded, &input); err != nil {
		return input, fmt.Errorf("decode restored task input: %w", err)
	}
	return input, nil
}
