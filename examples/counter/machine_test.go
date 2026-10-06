package counter_test

import (
	"testing"

	"github.com/nguyenvanduocit/go-xstate/examples/counter"
	"github.com/nguyenvanduocit/go-xstate/examples/internal/tracetest"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// TestTrace replays testdata/counter.golden.json, recorded from the JS example by
// scripts/trace/counter/counter.ts.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/counter/src/counterMachine.ts#L3
// JS trace: scripts/trace/counter/counter.ts.
func TestTrace(t *testing.T) {
	tracetest.Run(t, "testdata/counter.golden.json", func(clock xs.Clock, input any) *xs.Actor[*xs.MachineSnapshot[counter.Context]] {
		return xs.CreateActor(counter.Machine())
	})
}
