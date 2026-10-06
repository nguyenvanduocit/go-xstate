package eventgreeting_test

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nguyenvanduocit/go-xstate/examples/internal/tracetest"
	wf "github.com/nguyenvanduocit/go-xstate/examples/workflow/eventgreeting"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// TestTrace replays testdata/workflow-event-greeting.golden.json, recorded from the JS
// example by scripts/trace/workflow-event-greeting/workflow-event-greeting.ts
// with the real 1000 ms greetingFunction.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-event-greeting/main.ts#L4
// JS trace: scripts/trace/workflow-event-greeting/workflow-event-greeting.ts.
func TestTrace(t *testing.T) {
	tracetest.Run(t, "testdata/workflow-event-greeting.golden.json", func(clock xs.Clock, input any) *xs.Actor[*xs.MachineSnapshot[wf.Context]] {
		return xs.CreateActor(wf.NewMachine(wf.GreetingFunction(wf.GreetingDelay)))
	})
}

// TestStdout compares the printing entry with testdata/workflow-event-greeting.stdout.txt,
// recorded from main.ts. The delay is shortened; the text does not depend on it.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-event-greeting/main.ts#L51
// JS trace: scripts/trace/workflow-event-greeting/workflow-event-greeting.stdout.ts.
func TestStdout(t *testing.T) {
	want, err := os.ReadFile("testdata/workflow-event-greeting.stdout.txt")
	require.NoError(t, err)
	var out strings.Builder
	wf.RunWith(&out, 10*time.Millisecond)
	assert.Equal(t, string(want), out.String())
}

// After onDone the context carries the greeting; the root snapshot output stays
// nil, as in the JS trace (the final state's `output` only feeds a parent).
// Go regression; expected values are Go assertions, not a recorded JS test.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-event-greeting/main.ts#L4
// Related JS trace: scripts/trace/workflow-event-greeting/workflow-event-greeting.ts.
func TestGreeted_ContextCarriesGreeting(t *testing.T) {
	actor := xs.CreateActor(wf.NewMachine(wf.GreetingFunction(10 * time.Millisecond)))
	actor.Start()
	defer actor.Stop()
	actor.Send(wf.GreetEvent("Ada"))
	require.Eventually(t, func() bool { return actor.GetSnapshot().Status == xs.StatusDone }, time.Second, 5*time.Millisecond)
	assert.Equal(t, wf.Context{Greeting: "Hello, Ada!"}, actor.GetSnapshot().Context)
	assert.Nil(t, actor.GetSnapshot().Output)
}
