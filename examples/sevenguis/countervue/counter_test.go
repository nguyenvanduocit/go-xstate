package countervue_test

import (
	"testing"

	"github.com/nguyenvanduocit/go-xstate/examples/internal/tracetest"
	countervue "github.com/nguyenvanduocit/go-xstate/examples/sevenguis/countervue"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// TestTrace replays testdata/counter.golden.json, recorded from the JS example by
// scripts/trace/7guis-1-counter-vue/counter.ts.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/7guis-1-counter-vue/src/counterMachine.ts#L3
// JS trace: scripts/trace/7guis-1-counter-vue/counter.ts.
func TestTrace(t *testing.T) {
	tracetest.Run(t, "testdata/counter.golden.json", func(clock xs.Clock, input any) *xs.Actor[*xs.MachineSnapshot[countervue.Context]] {
		return xs.CreateActor(countervue.Machine())
	})
}
