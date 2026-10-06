package toggle_test

import (
	"testing"

	"github.com/nguyenvanduocit/go-xstate/examples/internal/tracetest"
	"github.com/nguyenvanduocit/go-xstate/examples/toggle"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// TestTrace replays testdata/toggle.golden.json, recorded from the JS example by
// scripts/trace/toggle/toggle.ts.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/toggle/src/toggleMachine.ts#L3
// JS trace: scripts/trace/toggle/toggle.ts.
func TestTrace(t *testing.T) {
	tracetest.Run(t, "testdata/toggle.golden.json", func(clock xs.Clock, input any) *xs.Actor[*xs.MachineSnapshot[toggle.Context]] {
		return xs.CreateActor(toggle.Machine())
	})
}
