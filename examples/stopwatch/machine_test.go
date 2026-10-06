package stopwatch_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nguyenvanduocit/go-xstate/examples/internal/tracetest"
	"github.com/nguyenvanduocit/go-xstate/examples/stopwatch"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// idleTicks is the no-op `ticks` actor the JS trace swaps in via machine.provide.
func idleTicks() xs.ActorLogic {
	return xs.FromCallback(func(xs.CallbackArgs) func() { return func() {} })
}

// TestTrace replays testdata/stopwatch.golden.json, recorded from the JS example by
// scripts/trace/stopwatch/stopwatch.ts.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/stopwatch/src/stopwatchMachine.ts#L3
// JS trace: scripts/trace/stopwatch/stopwatch.ts.
func TestTrace(t *testing.T) {
	tracetest.Run(t, "testdata/stopwatch.golden.json", func(clock xs.Clock, input any) *xs.Actor[*xs.MachineSnapshot[stopwatch.Context]] {
		return xs.CreateActor(stopwatch.NewMachine(idleTicks()))
	})
}

// TestRealTicker drives the production machine (wall-clock 10 ms ticker): elapsed
// grows while running, freezes after stop, and reset returns to 0.
// Go regression; expected values are Go assertions, not a recorded JS test.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/stopwatch/src/stopwatchMachine.ts#L3
// Related JS trace: scripts/trace/stopwatch/stopwatch.ts.
func TestRealTicker(t *testing.T) {
	actor := xs.CreateActor(stopwatch.Machine())
	actor.Start()
	defer actor.Stop()

	actor.Send(xs.Ev("start"))
	require.Eventually(t, func() bool { return actor.GetSnapshot().Context.Elapsed >= 3 }, 2*time.Second, 5*time.Millisecond)

	actor.Send(xs.Ev("stop"))
	frozen := actor.GetSnapshot().Context.Elapsed
	assert.Equal(t, "stopped", actor.GetSnapshot().Value)
	time.Sleep(10 * stopwatch.TickInterval)
	assert.Equal(t, frozen, actor.GetSnapshot().Context.Elapsed, "ticker must stop with the running state")

	actor.Send(xs.Ev("reset"))
	assert.Equal(t, 0, actor.GetSnapshot().Context.Elapsed)
	assert.Equal(t, "stopped", actor.GetSnapshot().Value)
}
