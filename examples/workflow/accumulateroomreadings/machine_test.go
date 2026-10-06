package accumulateroomreadings_test

import (
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nguyenvanduocit/go-xstate/examples/internal/tracetest"
	workflow "github.com/nguyenvanduocit/go-xstate/examples/workflow/accumulateroomreadings"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// TestTrace replays testdata/workflow-accumulate-room-readings.golden.json, recorded from main.ts by
// scripts/trace/workflow-accumulate-room-readings/workflow-accumulate-room-readings.ts.
// The 1000 ms promise of produceReport is 10 ms here; the trace waits 1100 ms for it.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-accumulate-room-readings/main.ts#L16
// JS trace: scripts/trace/workflow-accumulate-room-readings/workflow-accumulate-room-readings.ts.
func TestTrace(t *testing.T) {
	tracetest.Run(t, "testdata/workflow-accumulate-room-readings.golden.json", func(clock xs.Clock, input any) *xs.Actor[*xs.MachineSnapshot[workflow.Context]] {
		tm := workflow.Timing{PT1H: workflow.RealTiming.PT1H, ReportDelay: 10 * time.Millisecond}
		return xs.CreateActor(workflow.NewMachine(tm, io.Discard), xs.WithClock(clock))
	})
}

// TestStdout compares the printing entry with testdata/workflow-accumulate-room-readings.stdout.txt,
// recorded from main.ts. One JS millisecond lasts 100 us here (about 2.2 s per run).
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-accumulate-room-readings/main.ts#L105
// JS trace: scripts/trace/workflow-accumulate-room-readings/workflow-accumulate-room-readings.stdout.ts.
func TestStdout(t *testing.T) {
	want, err := os.ReadFile("testdata/workflow-accumulate-room-readings.stdout.txt")
	require.NoError(t, err)
	var out strings.Builder
	require.NoError(t, workflow.RunWith(&out, 100*time.Microsecond))
	assert.Equal(t, string(want), out.String())
}
