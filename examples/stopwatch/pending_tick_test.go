package stopwatch_test

import (
	"testing"
	"time"

	"github.com/nguyenvanduocit/go-xstate/examples/stopwatch"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/require"
)

// Go regression: callback cleanup cannot join a ticker waiting for the system lock.
func TestStopWithPendingTick(t *testing.T) {
	for _, event := range []string{"stop", "reset"} {
		t.Run(event, func(t *testing.T) {
			actor := xs.CreateActor(stopwatch.Machine()).Start()
			actor.SubscribeNext(func(s *xs.MachineSnapshot[stopwatch.Context]) {
				if s.Matches("running") && s.Context.Elapsed == 0 {
					// Observers hold the system lock. Allow the ticker to attempt
					// SendBack before queuing the event that disposes it.
					time.Sleep(10 * stopwatch.TickInterval)
					actor.Send(xs.Ev(event))
				}
			})
			done := make(chan struct{})
			go func() {
				actor.Send(xs.Ev("start"))
				close(done)
			}()
			select {
			case <-done:
			case <-time.After(2 * time.Second):
				t.Fatal("cleanup deadlocked with a pending tick")
			}
			defer actor.Stop()
			require.True(t, actor.GetSnapshot().Matches("stopped"))
			time.Sleep(2 * stopwatch.TickInterval)
			require.Zero(t, actor.GetSnapshot().Context.Elapsed, "late tick must not reach the stopped machine")
		})
	}
}
