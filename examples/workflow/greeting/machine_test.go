package greeting_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nguyenvanduocit/go-xstate/examples/internal/tracetest"
	wf "github.com/nguyenvanduocit/go-xstate/examples/workflow/greeting"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// TestTrace replays testdata/workflow-greeting.golden.json, recorded from the JS
// example by scripts/trace/workflow-greeting/workflow-greeting.ts.
// greetingFunction is replaced by the same stub as in the JS script: resolves
// `Hello, <name>!` after 100 ms.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-greeting/main.ts#L4
// JS trace: scripts/trace/workflow-greeting/workflow-greeting.ts.
func TestTrace(t *testing.T) {
	tracetest.Run(t, "testdata/workflow-greeting.golden.json", func(clock xs.Clock, input any) *xs.Actor[*xs.MachineSnapshot[wf.Context]] {
		stub := xs.FromPromise(func(ctx context.Context, a xs.PromiseArgs) (wf.GreetingOutput, error) {
			select {
			case <-time.After(100 * time.Millisecond):
				return wf.GreetingOutput{Greeting: "Hello, " + a.Input.(wf.GreetingInput).Name + "!"}, nil
			case <-ctx.Done():
				return wf.GreetingOutput{}, context.Cause(ctx)
			}
		})
		name := input.(map[string]any)["person"].(map[string]any)["name"].(string)
		return xs.CreateActor(wf.NewMachine(stub), xs.WithInput(wf.Input{Person: wf.Person{Name: name}}))
	})
}

// TestStdout compares the printing entry with testdata/workflow-greeting.stdout.txt,
// recorded from main.ts. The delay is shortened; the text does not depend on it.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-greeting/main.ts#L55
// JS trace: scripts/trace/workflow-greeting/workflow-greeting.stdout.ts.
func TestStdout(t *testing.T) {
	want, err := os.ReadFile("testdata/workflow-greeting.stdout.txt")
	require.NoError(t, err)
	var out strings.Builder
	wf.RunWith(&out, 10*time.Millisecond)
	assert.Equal(t, string(want), out.String())
}

// The final state's output is computed from the context but is not the root
// output (the machine declares no root `output`), as in the JS snapshot.
// Go regression; expected values are Go assertions, not a recorded JS test.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-greeting/main.ts#L4
// Related JS trace: scripts/trace/workflow-greeting/workflow-greeting.ts.
func TestFinalOutputFromContext(t *testing.T) {
	actor := xs.CreateActor(wf.NewMachine(wf.GreetingFunction(time.Millisecond)), xs.WithInput(wf.Input{Person: wf.Person{Name: "Ann"}}))
	done := make(chan struct{})
	actor.Subscribe(xs.Observer[*xs.MachineSnapshot[wf.Context]]{Complete: func() { close(done) }})
	actor.Start()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("workflow did not complete")
	}
	snap := actor.GetSnapshot()
	assert.Equal(t, "Hello, Ann!", snap.Context.Greeting)
	assert.Nil(t, snap.Output)
}
