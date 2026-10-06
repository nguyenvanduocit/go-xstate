package asyncfunction_test

import (
	"context"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nguyenvanduocit/go-xstate/examples/internal/tracetest"
	wf "github.com/nguyenvanduocit/go-xstate/examples/workflow/asyncfunction"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// TestTrace replays testdata/workflow-async-function.golden.json, recorded from the JS
// example by scripts/trace/workflow-async-function/workflow-async-function.ts.
// sendEmail is replaced by the same stub as in the JS script: resolves after 100 ms.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-async-function/main.ts#L4
// JS trace: scripts/trace/workflow-async-function/workflow-async-function.ts.
func TestTrace(t *testing.T) {
	tracetest.Run(t, "testdata/workflow-async-function.golden.json", func(clock xs.Clock, input any) *xs.Actor[*xs.MachineSnapshot[wf.Context]] {
		stub := xs.FromPromise(func(ctx context.Context, _ xs.PromiseArgs) (struct{}, error) {
			select {
			case <-time.After(100 * time.Millisecond):
				return struct{}{}, nil
			case <-ctx.Done():
				return struct{}{}, context.Cause(ctx)
			}
		})
		in := input.(map[string]any)
		return xs.CreateActor(wf.NewMachine(stub), xs.WithInput(wf.Input{Customer: in["customer"].(string)}))
	})
}

// TestStdout compares the printing entry with testdata/workflow-async-function.stdout.txt,
// recorded from main.ts. The delay is shortened; the text does not depend on it.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-async-function/main.ts#L46
// JS trace: scripts/trace/workflow-async-function/workflow-async-function.stdout.ts.
func TestStdout(t *testing.T) {
	want, err := os.ReadFile("testdata/workflow-async-function.stdout.txt")
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

// Stopping the actor while sendEmail is pending abandons it: "Email sent" is never printed.
// Go regression; expected values are Go assertions, not a recorded JS test.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-async-function/main.ts#L11
// Related JS trace: scripts/trace/workflow-async-function/workflow-async-function.ts.
func TestSendEmail_StoppedActorAbandonsPromise(t *testing.T) {
	var out syncBuffer
	actor := xs.CreateActor(wf.NewMachine(wf.SendEmail(&out, 50*time.Millisecond)), xs.WithInput(wf.Input{Customer: "x"}))
	actor.Start()
	actor.Stop()
	time.Sleep(150 * time.Millisecond)
	assert.Equal(t, "Sending email to x\n", out.String())
}
