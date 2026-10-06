package workflow_test

import (
	"testing"

	"github.com/nguyenvanduocit/go-xstate/examples/internal/tracetest"
	expressworkflow "github.com/nguyenvanduocit/go-xstate/examples/server/workflow"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// TestMachineTrace replays testdata/machine.golden.json, recorded from the JS
// machine by scripts/trace/express-workflow/machine.ts.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/express-workflow/machine.ts#L3
// JS trace: scripts/trace/express-workflow/machine.ts.
func TestMachineTrace(t *testing.T) {
	tracetest.Run(t, "testdata/machine.golden.json", func(clock xs.Clock, input any) *xs.Actor[*xs.MachineSnapshot[expressworkflow.Context]] {
		return xs.CreateActor(expressworkflow.Machine())
	})
}
