package hello_test

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nguyenvanduocit/go-xstate/examples/internal/tracetest"
	wf "github.com/nguyenvanduocit/go-xstate/examples/workflow/hello"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// TestTrace replays testdata/workflow-hello.golden.json, recorded from the JS example by
// scripts/trace/workflow-hello/workflow-hello.ts.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-hello/main.ts#L4
// JS trace: scripts/trace/workflow-hello/workflow-hello.ts.
func TestTrace(t *testing.T) {
	tracetest.Run(t, "testdata/workflow-hello.golden.json", func(clock xs.Clock, input any) *xs.Actor[*xs.MachineSnapshot[wf.Context]] {
		return xs.CreateActor(wf.Machine())
	})
}

// TestStdout compares the printing entry with testdata/workflow-hello.stdout.txt,
// recorded from main.ts.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-hello/main.ts#L17
// JS trace: scripts/trace/workflow-hello/workflow-hello.stdout.ts.
func TestStdout(t *testing.T) {
	want, err := os.ReadFile("testdata/workflow-hello.stdout.txt")
	require.NoError(t, err)
	var out strings.Builder
	wf.Run(&out)
	assert.Equal(t, string(want), out.String())
}
