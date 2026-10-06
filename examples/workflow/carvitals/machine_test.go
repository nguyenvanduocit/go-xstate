package carvitals_test

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nguyenvanduocit/go-xstate/examples/internal/tracetest"
	wf "github.com/nguyenvanduocit/go-xstate/examples/workflow/carvitals"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// stub mirrors the `check` helper of the JS trace scripts: resolves
// `{ value }` after d of real time.
func stub(d time.Duration, value int) xs.ActorLogic {
	return xs.FromPromise(func(ctx context.Context, _ xs.PromiseArgs) (wf.Reading, error) {
		select {
		case <-time.After(d):
			return wf.Reading{Value: value}, nil
		case <-ctx.Done():
			return wf.Reading{}, context.Cause(ctx)
		}
	})
}

func ms(n int) time.Duration { return time.Duration(n) * time.Millisecond }

// TestTrace_Workflow replays testdata/workflow-car-vitals.golden.json, recorded from the
// JS example by scripts/trace/workflow-car-vitals/workflow-car-vitals.ts. The checks are
// the same stubs as in the JS script: coolant 20 ms, tire 40 ms, battery 60 ms, oil 80 ms.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-car-vitals/main.ts#L110
// JS trace: scripts/trace/workflow-car-vitals/workflow-car-vitals.ts.
func TestTrace_Workflow(t *testing.T) {
	tracetest.Run(t, "testdata/workflow-car-vitals.golden.json", func(clock xs.Clock, _ any) *xs.Actor[*xs.MachineSnapshot[wf.WorkflowContext]] {
		vitals := wf.NewVitals(wf.Checks{
			CoolantLevel: stub(ms(20), 90),
			TirePressure: stub(ms(40), 32),
			Battery:      stub(ms(60), 12),
			OilPressure:  stub(ms(80), 45),
		})
		return xs.CreateActor(wf.NewWorkflow(vitals, io.Discard), xs.WithClock(clock), xs.WithLogger(func(...any) {}))
	})
}

// TestTrace_Vitals replays testdata/vitals.golden.json, recorded by
// scripts/trace/workflow-car-vitals/vitals.ts: coolant 100 ms, tire 200 ms,
// battery 300 ms, oil 400 ms.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-car-vitals/main.ts#L15
// JS trace: scripts/trace/workflow-car-vitals/vitals.ts.
func TestTrace_Vitals(t *testing.T) {
	tracetest.Run(t, "testdata/vitals.golden.json", func(_ xs.Clock, _ any) *xs.Actor[*xs.MachineSnapshot[wf.VitalsContext]] {
		return xs.CreateActor(wf.NewVitals(wf.Checks{
			CoolantLevel: stub(ms(100), 90),
			TirePressure: stub(ms(200), 32),
			Battery:      stub(ms(300), 12),
			OilPressure:  stub(ms(400), 45),
		}))
	})
}

type roundConsole struct{ strings.Builder }

func (w *roundConsole) Write(p []byte) (int, error) {
	n, err := w.Builder.Write(p)
	if strings.HasPrefix(string(p), "Done with vitals check ") {
		// Model nonzero console I/O so the next round's tire check falls after
		// car-off, as in the recording, rather than tying its exact deadline.
		time.Sleep(time.Nanosecond)
	}
	return n, err
}

// TestStdout compares the printing entry with testdata/workflow-car-vitals.stdout.txt,
// recorded from main.ts. Virtual time preserves timer order under CPU load.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-car-vitals/main.ts#L151
// JS trace: scripts/trace/workflow-car-vitals/workflow-car-vitals.stdout.ts.
func TestStdout(t *testing.T) {
	want, err := os.ReadFile("testdata/workflow-car-vitals.stdout.txt")
	require.NoError(t, err)
	synctest.Test(t, func(t *testing.T) {
		var out roundConsole
		wf.RunWith(&out, 10)
		assert.Equal(t, string(want), out.String())
	})
}

// delay rejects with ServiceNotAvailable when the error probability is reached.
// Go regression; expected values are Go assertions, not a recorded JS test.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-car-vitals/main.ts#L3
// Related JS trace: scripts/trace/workflow-car-vitals/vitals.ts.
func TestDelay_ErrorProbability(t *testing.T) {
	assert.NoError(t, wf.Delay(0, 0))
	assert.ErrorIs(t, wf.Delay(0, 1), wf.ErrServiceNotAvailable)
}
