package temperaturevue_test

import (
	"testing"

	"github.com/nguyenvanduocit/go-xstate/examples/internal/tracetest"
	temperaturevue "github.com/nguyenvanduocit/go-xstate/examples/sevenguis/temperaturevue"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// TestTrace replays testdata/temperature.golden.json, recorded from the JS example by
// scripts/trace/7guis-2-temperature-vue/temperature.ts.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/7guis-2-temperature-vue/src/tempMachine.ts#L20
// JS trace: scripts/trace/7guis-2-temperature-vue/temperature.ts.
func TestTrace(t *testing.T) {
	tracetest.Run(t, "testdata/temperature.golden.json", func(clock xs.Clock, input any) *xs.Actor[*xs.MachineSnapshot[temperaturevue.Context]] {
		return xs.CreateActor(temperaturevue.Machine())
	})
}
