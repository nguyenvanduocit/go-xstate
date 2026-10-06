package creditcheck_test

import (
	"testing"
	"time"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/require"
)

// Go-only regression: observer pauses must not advance the JS score-error trace.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/mongodb-credit-check-api/machine.ts#L14
// JS trace: scripts/trace/mongodb-credit-check-api/score-error.ts.
func TestTraceObserverPause(t *testing.T) {
	clock := xs.NewSimulatedClock()
	actor := xs.CreateActor(traceMachine(clock), xs.WithClock(clock), xs.WithUnhandledErrorHandler(func(any) {})).Start()
	defer actor.Stop()
	actor.Send(xs.E{"type": "Submit", "SSN": "555555555", "firstName": "Gavin", "lastName": "Bauman"})
	for _, ms := range []int{20, 40, 100, 100, 100} {
		clock.Increment(time.Duration(ms) * time.Millisecond)
	}
	require.True(t, actor.GetSnapshot().Matches("creditCheck.DeterminingInterestRateOptions.DeterminingMiddleScore"))
	// Simulate a scheduler pause between finishing a wait and reading its snapshot.
	time.Sleep(80 * time.Millisecond)
	require.Equal(t, xs.StatusActive, actor.GetSnapshot().Status)
	clock.Increment(40 * time.Millisecond)
	require.Equal(t, xs.StatusError, actor.GetSnapshot().Status)
	require.EqualError(t, actor.GetSnapshot().Error.(error), "score service unavailable")
}
