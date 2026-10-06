package fillingwater_test

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nguyenvanduocit/go-xstate/examples/internal/tracetest"
	wfw "github.com/nguyenvanduocit/go-xstate/examples/workflow/fillingwater"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// replay replays testdata/<name>.golden.json, recorded from the JS example by
// scripts/trace/workflow-filling-water/<script>.ts, with the recorded input and SimulatedClock.
func replay(t *testing.T, name string) {
	t.Helper()
	tracetest.Run(t, "testdata/"+name+".golden.json", func(clock xs.Clock, input any) *xs.Actor[*xs.MachineSnapshot[wfw.Context]] {
		in := input.(map[string]any)
		return xs.CreateActor(wfw.Machine(),
			xs.WithInput(wfw.Input{Current: int(in["current"].(float64)), Max: int(in["max"].(float64))}),
			xs.WithClock(clock))
	})
}

// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-filling-water/main.ts#L4
// JS trace: scripts/trace/workflow-filling-water/fill.ts.
func TestTraceFill(t *testing.T) { replay(t, "fill") }

// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-filling-water/main.ts#L4
// JS trace: scripts/trace/workflow-filling-water/already-full.ts.
func TestTraceAlreadyFull(t *testing.T) { replay(t, "already-full") }

// TestStdout compares the printing entry with testdata/workflow-filling-water.stdout.txt,
// recorded from main.ts. The delay is shortened; the text does not depend on it.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-filling-water/main.ts#L56
// JS trace: scripts/trace/workflow-filling-water/workflow-filling-water.stdout.ts.
func TestStdout(t *testing.T) {
	want, err := os.ReadFile("testdata/workflow-filling-water.stdout.txt")
	require.NoError(t, err)
	var out strings.Builder
	wfw.RunWith(&out, 5*time.Millisecond)
	assert.Equal(t, string(want), out.String())
}
