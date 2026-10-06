package flightbooker_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nguyenvanduocit/go-xstate/examples/internal/tracetest"
	flight "github.com/nguyenvanduocit/go-xstate/examples/sevenguis/flightbooker"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// The JS traces pin TODAY to 2024-03-10 (scripts/trace/7guis-flight-booker-react/harness.mjs).
const (
	today    = "2024-03-10"
	tomorrow = "2024-03-11"
)

func create(booker xs.ActorLogic) func(xs.Clock, any) *xs.Actor[*xs.MachineSnapshot[flight.Context]] {
	return func(xs.Clock, any) *xs.Actor[*xs.MachineSnapshot[flight.Context]] {
		m := flight.Machine(today, tomorrow).Provide(xs.Implementations{
			Actors: map[string]xs.ActorLogic{"Booker": booker},
		})
		return xs.CreateActor(m)
	}
}

// stubBooker settles after 100 ms of real time, like the Booker stubs of the JS traces, so the machine
// stays in `booking` while the next trace steps run.
func stubBooker(err error) xs.ActorLogic {
	return xs.FromPromise(func(ctx context.Context, _ xs.PromiseArgs) (any, error) {
		select {
		case <-time.After(100 * time.Millisecond):
			return nil, err
		case <-ctx.Done():
			return nil, context.Cause(ctx)
		}
	})
}

// TestTrace replays testdata/flight-booker.golden.json, recorded by
// scripts/trace/7guis-flight-booker-react/flight-booker.ts (Booker stub resolves).
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/7guis-flight-booker-react/src/machines/flightMachine.ts#L5
// JS trace: scripts/trace/7guis-flight-booker-react/flight-booker.ts.
func TestTrace(t *testing.T) {
	tracetest.Run(t, "testdata/flight-booker.golden.json", create(stubBooker(nil)))
}

// TestTraceOnewayError replays testdata/flight-booker-oneway-error.golden.json, recorded by
// scripts/trace/7guis-flight-booker-react/flight-booker-oneway-error.ts (Booker stub rejects).
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/7guis-flight-booker-react/src/machines/flightMachine.ts#L5
// JS trace: scripts/trace/7guis-flight-booker-react/flight-booker-oneway-error.ts.
func TestTraceOnewayError(t *testing.T) {
	tracetest.Run(t, "testdata/flight-booker-oneway-error.golden.json", create(stubBooker(errors.New("booking failed"))))
}

// Go regression; expected values are Go assertions, not a recorded JS test.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/7guis-flight-booker-react/src/utils/index.ts#L2
// Related JS trace: scripts/trace/7guis-flight-booker-react/flight-booker-oneway-error.ts.
func TestDates(t *testing.T) {
	assert.Regexp(t, `^\d{4}-\d{2}-\d{2}$`, flight.Today())
	assert.Regexp(t, `^\d{4}-\d{2}-\d{2}$`, flight.Tomorrow())
	assert.NotEqual(t, flight.Today(), flight.Tomorrow())
}

// The real Booker sleeps 2000 ms and stops promptly when the actor is stopped.
// Go regression; expected values are Go assertions, not a recorded JS test.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/7guis-flight-booker-react/src/machines/flightMachine.ts#L26
// Related JS trace: scripts/trace/7guis-flight-booker-react/flight-booker-oneway-error.ts.
func TestBookerStopsOnCancel(t *testing.T) {
	actor := xs.CreateActor(flight.Machine(today, tomorrow))
	actor.Start()
	actor.Send(xs.Ev("BOOK_DEPART"))
	snap := actor.GetSnapshot()
	require.True(t, snap.Matches("booking"))
	require.Len(t, snap.Children, 1)
	start := time.Now()
	actor.Stop()
	assert.Less(t, time.Since(start), time.Second)
}
