package finalizecollegeapp_test

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
	wf "github.com/nguyenvanduocit/go-xstate/examples/workflow/finalizecollegeapp"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// TestTrace replays testdata/workflow-finalize-college-app.golden.json, recorded from the JS
// example by scripts/trace/workflow-finalize-college-app/workflow-finalize-college-app.ts.
// finalizeApplicationFunction is replaced by the same stub as in the JS script:
// resolves with `{ applicantId }` after 100 ms.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-finalize-college-app/main.ts#L4
// JS trace: scripts/trace/workflow-finalize-college-app/workflow-finalize-college-app.ts.
func TestTrace(t *testing.T) {
	tracetest.Run(t, "testdata/workflow-finalize-college-app.golden.json", func(_ xs.Clock, input any) *xs.Actor[*xs.MachineSnapshot[wf.Context]] {
		raw, err := json.Marshal(input)
		require.NoError(t, err)
		var in wf.Input
		require.NoError(t, json.Unmarshal(raw, &in))
		stub := xs.FromPromise(func(ctx context.Context, a xs.PromiseArgs) (wf.FinalizeOutput, error) {
			select {
			case <-time.After(100 * time.Millisecond):
				return wf.FinalizeOutput{ApplicantID: a.Input.(wf.Input).ApplicantID}, nil
			case <-ctx.Done():
				return wf.FinalizeOutput{}, context.Cause(ctx)
			}
		})
		return xs.CreateActor(wf.NewMachine(stub), xs.WithInput(in))
	})
}

// TestStdout compares the printing entry with testdata/workflow-finalize-college-app.stdout.txt,
// recorded from main.ts. The delay is shortened; the text does not depend on it.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-finalize-college-app/main.ts#L83
// JS trace: scripts/trace/workflow-finalize-college-app/workflow-finalize-college-app.stdout.ts.
func TestStdout(t *testing.T) {
	want, err := os.ReadFile("testdata/workflow-finalize-college-app.stdout.txt")
	require.NoError(t, err)
	var out strings.Builder
	wf.RunWith(&out, 10*time.Millisecond)
	assert.Equal(t, string(want), out.String())
}

// syncBuffer is a strings.Builder safe for the promise goroutine and the test.
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

// Stopping the actor while finalizeApplicationFunction is pending abandons it:
// "Finalized application" is never printed.
// Go regression; expected values are Go assertions, not a recorded JS test.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-finalize-college-app/main.ts#L17
// Related JS trace: scripts/trace/workflow-finalize-college-app/workflow-finalize-college-app.ts.
func TestFinalize_StoppedActorAbandonsPromise(t *testing.T) {
	var out syncBuffer
	actor := xs.CreateActor(wf.NewMachine(wf.FinalizeApplicationFunction(&out, 50*time.Millisecond)), xs.WithInput(wf.Input{ApplicantID: "x"}))
	actor.Start()
	for _, e := range []string{"ApplicationSubmitted", "SATScoresReceived", "RecommendationLetterReceived"} {
		actor.Send(xs.Ev(e))
	}
	actor.Stop()
	time.Sleep(150 * time.Millisecond)
	assert.Equal(t, "Starting to finalize application for x\n", out.String())
}
