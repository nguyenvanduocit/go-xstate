package monitorpatient_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nguyenvanduocit/go-xstate/examples/internal/tracetest"
	wf "github.com/nguyenvanduocit/go-xstate/examples/workflow/monitorpatient"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// TestTrace replays testdata/workflow-monitor-patient.golden.json, recorded from the JS
// example by scripts/trace/workflow-monitor-patient/workflow-monitor-patient.ts.
// The golden file carries the machine input.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-monitor-patient/main.ts#L4
// JS trace: scripts/trace/workflow-monitor-patient/workflow-monitor-patient.ts.
func TestTrace(t *testing.T) {
	tracetest.Run(t, "testdata/workflow-monitor-patient.golden.json", func(_ xs.Clock, input any) *xs.Actor[*xs.MachineSnapshot[wf.Context]] {
		raw, err := json.Marshal(input)
		require.NoError(t, err)
		var in wf.Input
		require.NoError(t, json.Unmarshal(raw, &in))
		return xs.CreateActor(wf.NewMachine(&bytes.Buffer{}), xs.WithInput(in))
	})
}

// TestActionsStdout drives the machine by hand and compares what the three actions
// print with testdata/actions.stdout.txt, recorded by scripts/trace/workflow-monitor-patient/actions.stdout.ts.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-monitor-patient/main.ts#L81
// JS trace: scripts/trace/workflow-monitor-patient/actions.stdout.ts.
func TestActionsStdout(t *testing.T) {
	want, err := os.ReadFile("testdata/actions.stdout.txt")
	require.NoError(t, err)
	var out bytes.Buffer
	actor := xs.CreateActor(wf.NewMachine(&out), xs.WithInput(wf.Input{PatientID: "patient42"}))
	actor.Start()
	defer actor.Stop()
	const at = "2024-01-01T00:00:00.000Z"
	actor.Send(wf.VitalEvent(wf.HighRespirationRate, "event1", at, "patient42", "v"))
	actor.Send(wf.VitalEvent(wf.HighBodyTemp, "event1", at, "patient42", "v"))
	actor.Send(xs.E{"type": "unknown"})
	actor.Send(wf.VitalEvent(wf.HighBloodPressure, "event1", at, "patient42", "v"))
	assert.Equal(t, string(want), out.String())
}

// TestStdout runs the entry with Math.random replaced by the sequence of
// scripts/trace/workflow-monitor-patient/workflow-monitor-patient.stdout.ts and
// the 3 s interval replaced by six immediate ticks; the text is recorded from main.ts.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-monitor-patient/main.ts#L81
// JS trace: scripts/trace/workflow-monitor-patient/workflow-monitor-patient.stdout.ts.
func TestStdout(t *testing.T) {
	want, err := os.ReadFile("testdata/workflow-monitor-patient.stdout.txt")
	require.NoError(t, err)
	picks := []float64{0.0, 0.5, 0.9, 0.4, 0.7, 0.1}
	ticks := make(chan time.Time, len(picks))
	for range picks {
		ticks <- time.Now()
	}
	close(ticks)
	i := 0
	random := func() float64 { i++; return picks[i-1] }
	var out bytes.Buffer
	wf.RunWith(context.Background(), &out, ticks, random)
	assert.Equal(t, string(want), out.String())
}

// Run stops when its context is cancelled (the JS entry runs until the process is killed).
// Go regression; expected values are Go assertions, not a recorded JS test.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-monitor-patient/main.ts#L81
// Related JS trace: scripts/trace/workflow-monitor-patient/actions.stdout.ts.
func TestRun_StopsOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		wf.Run(ctx, &bytes.Buffer{})
		close(done)
	}()
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run did not return after cancel")
	}
}
