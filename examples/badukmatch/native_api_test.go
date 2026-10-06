package badukmatch

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestPlayPreservesCancellationIdentityWithCustomCause(t *testing.T) {
	cause := errors.New("user cancelled match")
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	started := make(chan struct{})
	finished := make(chan error, 1)
	go func() {
		_, err := Play(ctx, Config{}, func(ctx context.Context, _ Turn) (Decision, error) {
			close(started)
			<-ctx.Done()
			return Decision{}, ctx.Err()
		}, nil)
		finished <- err
	}()
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("task did not start")
	}
	cancel(cause)
	select {
	case err := <-finished:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("want context.Canceled, got %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("match did not cancel")
	}
	_, err := Play(ctx, Config{}, func(context.Context, Turn) (Decision, error) {
		t.Error("cancelled match started")
		return Decision{}, nil
	}, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-cancelled match: %v", err)
	}
}
