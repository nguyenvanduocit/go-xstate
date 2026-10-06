package xstate

import (
	"context"
	"errors"
)

// Await waits for a matching snapshot or returns the actor/cancellation error.
// It is the blocking Go equivalent of WaitFor(ctx, actor, predicate).Wait().
// Cancellation removes the subscription but does not stop the actor. The
// predicate runs synchronously under the actor system lock and must not block.
// Do not call Await from an action or observer of that same actor system.
func Await[S Snapshot](ctx context.Context, actor *Actor[S], predicate func(S) bool) (S, error) {
	var zero S
	if ctx == nil {
		return zero, errors.New("xstate: context is required")
	}
	if actor == nil {
		return zero, errors.New("xstate: actor is required")
	}
	if predicate == nil {
		return zero, errors.New("xstate: predicate is required")
	}
	return WaitFor(ctx, actor, predicate).Wait()
}
