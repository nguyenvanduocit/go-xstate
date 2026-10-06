package mathproblem_test

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nguyenvanduocit/go-xstate/examples/internal/tracetest"
	wf "github.com/nguyenvanduocit/go-xstate/examples/workflow/mathproblem"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// stub resolves like the JS scripts' batchMathFunction: after 50 ms, with the
// same `Solved <problem>` mapping (SolveBatch with no per-problem delay).
var stub = xs.FromPromise(func(ctx context.Context, a xs.PromiseArgs) ([]wf.Solved, error) {
	select {
	case <-time.After(50 * time.Millisecond):
	case <-ctx.Done():
		return nil, context.Cause(ctx)
	}
	return wf.SolveBatch(ctx, io.Discard, 0, a.Input.(wf.BatchInput).Problems)
})

// replayTrace replays testdata/<golden>.golden.json, recorded by
// scripts/trace/workflow-math-problem/<script>.ts.
func replayTrace(t *testing.T, golden string) {
	tracetest.Run(t, "testdata/"+golden+".golden.json", func(_ xs.Clock, input any) *xs.Actor[*xs.MachineSnapshot[wf.Context]] {
		raw, err := json.Marshal(input)
		require.NoError(t, err)
		var in wf.Input
		require.NoError(t, json.Unmarshal(raw, &in))
		return xs.CreateActor(wf.NewMachine(stub), xs.WithInput(in))
	})
}

// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-math-problem/main.ts#L4
// JS trace: scripts/trace/workflow-math-problem/workflow-math-problem.ts.
func TestTrace(t *testing.T) { replayTrace(t, "workflow-math-problem") }

// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-math-problem/main.ts#L4
// JS trace: scripts/trace/workflow-math-problem/empty.ts.
func TestTraceEmpty(t *testing.T) { replayTrace(t, "empty") }

// TestStdout compares the printing entry with testdata/workflow-math-problem.stdout.txt,
// recorded from main.ts. The delay is shortened; the text does not depend on it.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-math-problem/main.ts#L62
// JS trace: scripts/trace/workflow-math-problem/workflow-math-problem.stdout.ts.
func TestStdout(t *testing.T) {
	want, err := os.ReadFile("testdata/workflow-math-problem.stdout.txt")
	require.NoError(t, err)
	var out strings.Builder
	wf.RunWith(&out, 10*time.Millisecond)
	assert.Equal(t, string(want), out.String())
}

// The problems run concurrently: n problems take about one delay, not n delays.
// Go regression; expected values are Go assertions, not a recorded JS test.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-math-problem/main.ts#L11
// Related JS trace: scripts/trace/workflow-math-problem/empty.ts.
func TestSolveBatch_Concurrent(t *testing.T) {
	start := time.Now()
	got, err := wf.SolveBatch(context.Background(), io.Discard, 100*time.Millisecond, []string{"a", "b", "c", "d"})
	require.NoError(t, err)
	assert.Less(t, time.Since(start), 300*time.Millisecond)
	assert.Equal(t, []wf.Solved{
		{Problem: "a", Result: "Solved a"}, {Problem: "b", Result: "Solved b"},
		{Problem: "c", Result: "Solved c"}, {Problem: "d", Result: "Solved d"},
	}, got)
}

// Cancelling the context abandons the batch.
// Go regression; expected values are Go assertions, not a recorded JS test.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-math-problem/main.ts#L11
// Related JS trace: scripts/trace/workflow-math-problem/empty.ts.
func TestSolveBatch_Cancelled(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(context.Canceled)
	_, err := wf.SolveBatch(ctx, io.Discard, time.Hour, []string{"a"})
	assert.ErrorIs(t, err, context.Canceled)
}

// Context marshals nil Results as an absent key and empty Results as [].
// Go regression; expected values are Go assertions, not a recorded JS test.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-math-problem/main.ts#L4
// Related JS trace: scripts/trace/workflow-math-problem/empty.ts.
func TestContext_MarshalJSON(t *testing.T) {
	b, err := json.Marshal(wf.Context{})
	require.NoError(t, err)
	assert.JSONEq(t, `{}`, string(b))
	b, err = json.Marshal(wf.Context{Results: []string{}})
	require.NoError(t, err)
	assert.JSONEq(t, `{"results":[]}`, string(b))
}
