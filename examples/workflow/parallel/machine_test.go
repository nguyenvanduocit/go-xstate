package parallel_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nguyenvanduocit/go-xstate/examples/internal/tracetest"
	wf "github.com/nguyenvanduocit/go-xstate/examples/workflow/parallel"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// stub resolves after d, like the stubs in the JS script.
func stub(d time.Duration) xs.ActorLogic {
	return xs.FromPromise(func(ctx context.Context, _ xs.PromiseArgs) (struct{}, error) {
		select {
		case <-time.After(d):
			return struct{}{}, nil
		case <-ctx.Done():
			return struct{}{}, context.Cause(ctx)
		}
	})
}

// TestTrace replays testdata/workflow-parallel.golden.json, recorded from the JS example by
// scripts/trace/workflow-parallel/workflow-parallel.ts. shortDelay and longDelay are
// replaced by the same stubs as in the JS script: resolve after 100 ms and 300 ms.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-parallel/main.ts#L4
// JS trace: scripts/trace/workflow-parallel/workflow-parallel.ts.
func TestTrace(t *testing.T) {
	tracetest.Run(t, "testdata/workflow-parallel.golden.json", func(clock xs.Clock, input any) *xs.Actor[*xs.MachineSnapshot[wf.Context]] {
		return xs.CreateActor(wf.NewMachine(stub(100*time.Millisecond), stub(300*time.Millisecond)))
	})
}

// TestStdout compares the printing entry with testdata/workflow-parallel.stdout.txt,
// recorded from main.ts. The delays are shortened; the text does not depend on them.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-parallel/main.ts#L67
// JS trace: scripts/trace/workflow-parallel/workflow-parallel.stdout.ts.
func TestStdout(t *testing.T) {
	want, err := os.ReadFile("testdata/workflow-parallel.stdout.txt")
	require.NoError(t, err)
	var out strings.Builder
	wf.RunWith(&out, 10*time.Millisecond, 60*time.Millisecond)
	assert.Equal(t, string(want), out.String())
}
