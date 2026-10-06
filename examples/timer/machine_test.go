package timer_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nguyenvanduocit/go-xstate/examples/internal/tracetest"
	"github.com/nguyenvanduocit/go-xstate/examples/timer"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// idleTicks is the no-op `ticks` actor the JS trace swaps in via machine.provide.
func idleTicks() xs.ActorLogic {
	return xs.FromCallback(func(xs.CallbackArgs) func() { return func() {} })
}

// TestTrace replays testdata/timer.golden.json, recorded from the JS example by
// scripts/trace/timer/timer.ts.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/timer/src/timerMachine.ts#L3
// JS trace: scripts/trace/timer/timer.ts.
func TestTrace(t *testing.T) {
	tracetest.Run(t, "testdata/timer.golden.json", func(clock xs.Clock, input any) *xs.Actor[*xs.MachineSnapshot[timer.Context]] {
		return xs.CreateActor(timer.NewMachine(idleTicks()))
	})
}

// TestRealTicker drives the production machine with a short wall-clock ticker:
// seconds count down while running, the machine stops itself at 0, and the
// ticker stops with the running state.
// Go regression; expected values are Go assertions, not a recorded JS test.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/timer/src/timerMachine.ts#L3
// Related JS trace: scripts/trace/timer/timer.ts.
func TestRealTicker(t *testing.T) {
	const interval = 5 * time.Millisecond
	actor := xs.CreateActor(timer.NewMachine(timer.Ticks(interval)))
	actor.Start()
	defer actor.Stop()

	actor.Send(xs.Ev("second"))
	actor.Send(xs.Ev("second"))
	actor.Send(xs.Ev("second"))
	actor.Send(xs.Ev("start"))
	require.Equal(t, "running", actor.GetSnapshot().Value)

	require.Eventually(t, func() bool {
		s := actor.GetSnapshot()
		return s.Value == "stopped" && s.Context.Seconds == 0
	}, 2*time.Second, interval)

	time.Sleep(10 * interval)
	assert.Equal(t, 0, actor.GetSnapshot().Context.Seconds, "ticker must stop with the running state")
}

// Go regression; expected values are Go assertions, not a recorded JS test.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/timer/src/timerMachine.ts#L3
// Related JS trace: scripts/trace/timer/timer.ts.
func TestProductionTickInterval(t *testing.T) {
	assert.Equal(t, time.Second, timer.TickInterval)
}
