package checkinbox_test

import (
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nguyenvanduocit/go-xstate/examples/internal/tracetest"
	wf "github.com/nguyenvanduocit/go-xstate/examples/workflow/checkinbox"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// traceActors are the stubs of scripts/trace/workflow-check-inbox/lib/stubs.ts: inbox read
// 100 ms, texts sent 20 ms (high) / 100 ms (low), schedule reminding every interval (0: never).
func traceActors(interval time.Duration) wf.Actors {
	a := wf.NewActors(wf.Timings{
		CheckInbox: 100 * time.Millisecond,
		SendHigh:   20 * time.Millisecond,
		SendLow:    100 * time.Millisecond,
	}, io.Discard)
	if interval == 0 {
		a.Schedule = xs.FromCallback(func(xs.CallbackArgs) func() { return func() {} })
	} else {
		a.Schedule = wf.Schedule(interval)
	}
	return a
}

func create(interval time.Duration) func(xs.Clock, any) *xs.Actor[*xs.MachineSnapshot[wf.Context]] {
	return func(_ xs.Clock, _ any) *xs.Actor[*xs.MachineSnapshot[wf.Context]] {
		return xs.CreateActor(wf.NewMachine(traceActors(interval)))
	}
}

// TestTraceManual replays testdata/manual.golden.json (reminders sent by hand).
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-check-inbox/main.ts#L17
// JS trace: scripts/trace/workflow-check-inbox/manual.ts.
func TestTraceManual(t *testing.T) {
	tracetest.Run(t, "testdata/manual.golden.json", create(0))
}

// TestTraceScheduled replays testdata/scheduled.golden.json (the schedule actor reminds every 300 ms).
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-check-inbox/main.ts#L17
// JS trace: scripts/trace/workflow-check-inbox/scheduled.ts.
func TestTraceScheduled(t *testing.T) {
	tracetest.Run(t, "testdata/scheduled.golden.json", create(300*time.Millisecond))
}

// syncBuffer is a strings.Builder safe for the promise goroutines and the test.
type syncBuffer struct {
	mu sync.Mutex
	sb strings.Builder
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.sb.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.sb.String()
}

// TestStdout compares the printing entry with testdata/workflow-check-inbox.stdout.txt, recorded
// from main.ts run for 4500 ms. One JS millisecond lasts 100 us here (450 ms per run).
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-check-inbox/main.ts#L110
// JS trace: scripts/trace/workflow-check-inbox/workflow-check-inbox.stdout.ts.
func TestStdout(t *testing.T) {
	want, err := os.ReadFile("testdata/workflow-check-inbox.stdout.txt")
	require.NoError(t, err)
	var out syncBuffer
	wf.RunScaled(&out, 100*time.Microsecond, 4500)
	assert.Equal(t, string(want), out.String())
}

// Stopping the actor while sendTextsFunction is pending abandons it: "text sent" is never printed.
// Go regression; expected values are Go assertions, not a recorded JS test.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-check-inbox/main.ts#L47
// Related JS trace: scripts/trace/workflow-check-inbox/manual.ts.
func TestSendTexts_StoppedActorAbandonsPromise(t *testing.T) {
	var out syncBuffer
	logic := wf.SendTextsFunction(200*time.Millisecond, 200*time.Millisecond, &out)
	input := wf.SendTextsInput{Messages: []wf.Message{{Subject: "x", Priority: "high"}}}
	actor := xs.CreateActor(logic, xs.WithInput(input))
	actor.Start()
	time.Sleep(20 * time.Millisecond)
	actor.Stop()
	time.Sleep(300 * time.Millisecond)
	assert.Equal(t, "sending text x\n", out.String())
}
