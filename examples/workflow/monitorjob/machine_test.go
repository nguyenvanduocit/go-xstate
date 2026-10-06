package monitorjob_test

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nguyenvanduocit/go-xstate/examples/internal/tracetest"
	wf "github.com/nguyenvanduocit/go-xstate/examples/workflow/monitorjob"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// afterStub is the Go twin of after100 in scripts/trace/workflow-monitor-job/lib/load.ts:
// it resolves with the value produced by next after 100 ms, or is abandoned when the invoking
// state exits.
func afterStub[T any](next func() T) xs.ActorLogic {
	return xs.FromPromise(func(ctx context.Context, _ xs.PromiseArgs) (T, error) {
		select {
		case <-time.After(100 * time.Millisecond):
			return next(), nil
		case <-ctx.Done():
			var zero T
			return zero, context.Cause(ctx)
		}
	})
}

// stubs is the Go twin of stubbedWorkflow in lib/load.ts: checkJobStatus answers with the given
// statuses in order (the last one repeats); "" is `undefined`.
func stubs(statuses ...string) wf.Actors {
	var mu sync.Mutex
	call := 0
	return wf.Actors{
		SubmitJob: afterStub(func() wf.SubmitOutput { return wf.SubmitOutput{JobUID: "123"} }),
		CheckJobStatus: afterStub(func() wf.StatusOutput {
			mu.Lock()
			defer mu.Unlock()
			s := statuses[min(call, len(statuses)-1)]
			call++
			return wf.StatusOutput{JobStatus: s}
		}),
		ReportJobSucceeded: afterStub(func() struct{} { return struct{}{} }),
		ReportJobFailed:    afterStub(func() struct{} { return struct{}{} }),
	}
}

// replay runs a golden file recorded with a SimulatedClock and the golden's own input.
func replay(t *testing.T, file string, actors wf.Actors) {
	tracetest.Run(t, "testdata/"+file, func(clock xs.Clock, input any) *xs.Actor[*xs.MachineSnapshot[wf.Context]] {
		raw, err := json.Marshal(input)
		require.NoError(t, err)
		var in wf.Input
		require.NoError(t, json.Unmarshal(raw, &in))
		return xs.CreateActor(wf.NewMachine(actors), xs.WithInput(in), xs.WithClock(clock))
	})
}

// TestSucceededTrace replays testdata/succeeded.golden.json (succeeded.ts): an undefined status
// loops back to WaitForCompletion, then SUCCEEDED -> JobSucceeded -> End.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-monitor-job/main.ts#L8
// JS trace: scripts/trace/workflow-monitor-job/succeeded.ts.
func TestSucceededTrace(t *testing.T) {
	replay(t, "succeeded.golden.json", stubs("", wf.Succeeded))
}

// TestFailedTrace replays testdata/failed.golden.json (failed.ts): FAILED -> JobFailed -> End.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-monitor-job/main.ts#L8
// JS trace: scripts/trace/workflow-monitor-job/failed.ts.
func TestFailedTrace(t *testing.T) {
	replay(t, "failed.golden.json", stubs(wf.Failed))
}

// TestStdout compares the printing entry with testdata/workflow-monitor-job.stdout.txt,
// recorded from main.ts. A SimulatedClock replaces the 5000 ms wait: a goroutine advances it
// until the workflow completes, so the test does not depend on when the timer is scheduled.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-monitor-job/main.ts#L115
// JS trace: scripts/trace/workflow-monitor-job/workflow-monitor-job.stdout.ts.
func TestStdout(t *testing.T) {
	want, err := os.ReadFile("testdata/workflow-monitor-job.stdout.txt")
	require.NoError(t, err)

	clock := xs.NewSimulatedClock()
	stop := make(chan struct{})
	ticked := make(chan struct{})
	go func() {
		defer close(ticked)
		for {
			select {
			case <-stop:
				return
			case <-time.After(5 * time.Millisecond):
				clock.Increment(wf.WaitDelay)
			}
		}
	}()

	var out strings.Builder
	wf.RunWith(&out, clock)
	close(stop)
	<-ticked
	assert.Equal(t, string(want), out.String())
}
