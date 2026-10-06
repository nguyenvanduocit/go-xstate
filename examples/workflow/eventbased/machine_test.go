package eventbased_test

import (
	"context"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nguyenvanduocit/go-xstate/examples/internal/tracetest"
	wf "github.com/nguyenvanduocit/go-xstate/examples/workflow/eventbased"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// stubActors are the same deterministic stubs as in the JS scripts
// (scripts/trace/workflow-event-based/lib/record.ts): each resolves after 100 ms.
func stubActors() map[string]xs.ActorLogic {
	stub := func() xs.ActorLogic {
		return xs.FromPromise(func(ctx context.Context, _ xs.PromiseArgs) (struct{}, error) {
			select {
			case <-time.After(100 * time.Millisecond):
				return struct{}{}, nil
			case <-ctx.Done():
				return struct{}{}, context.Cause(ctx)
			}
		})
	}
	return map[string]xs.ActorLogic{
		"handleApprovedVisaWorkflowID":   stub(),
		"handleRejectedVisaWorkflowID":   stub(),
		"handleNoVisaDecisionWorkflowId": stub(),
	}
}

// TestTrace replays the golden traces recorded from the JS example by
// scripts/trace/workflow-event-based/{approved,rejected,timeout}.ts.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-event-based/main.ts#L4
// JS trace: scripts/trace/workflow-event-based/approved.ts.
// JS trace: scripts/trace/workflow-event-based/rejected.ts.
// JS trace: scripts/trace/workflow-event-based/timeout.ts.
func TestTrace(t *testing.T) {
	for _, name := range []string{"approved", "rejected", "timeout"} {
		t.Run(name, func(t *testing.T) {
			tracetest.Run(t, "testdata/"+name+".golden.json", func(clock xs.Clock, _ any) *xs.Actor[*xs.MachineSnapshot[wf.Context]] {
				return xs.CreateActor(wf.NewMachine(stubActors()), xs.WithClock(clock))
			})
		})
	}
}

// TestStdout compares the printing entry with testdata/workflow-event-based.stdout.txt,
// recorded from main.ts. The delay is shortened; the text does not depend on it.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-event-based/main.ts#L65
// JS trace: scripts/trace/workflow-event-based/workflow-event-based.stdout.ts.
func TestStdout(t *testing.T) {
	want, err := os.ReadFile("testdata/workflow-event-based.stdout.txt")
	require.NoError(t, err)
	var out strings.Builder
	wf.RunWith(&out, 10*time.Millisecond)
	assert.Equal(t, string(want), out.String())
}

// syncBuffer is a strings.Builder safe for the promise goroutine and the test.
type syncBuffer struct {
	mu sync.Mutex
	sb strings.Builder
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.sb.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.sb.String()
}

// A decision event cancels visaDecisionTimeout: the timeout handler never starts.
// Go regression; expected values are Go assertions, not a recorded JS test.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-event-based/main.ts#L4
// Related JS trace: scripts/trace/workflow-event-based/approved.ts.
func TestDecisionCancelsTimeout(t *testing.T) {
	var out syncBuffer
	clock := xs.NewSimulatedClock()
	actor := xs.CreateActor(wf.NewMachine(wf.Actors(&out, 10*time.Millisecond)), xs.WithClock(clock))
	actor.Start()
	actor.Send(xs.Ev("visaRejectedEvent"))
	clock.Increment(5 * wf.VisaDecisionTimeout)
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, "handleRejectedVisaWorkflowID workflow started\nhandleRejectedVisaWorkflowID workflow completed\n", out.String())
	assert.Equal(t, xs.StatusDone, actor.GetSnapshot().Status)
}

// Stopping the actor while a handler is pending abandons it: "completed" is never printed.
// Go regression; expected values are Go assertions, not a recorded JS test.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-event-based/main.ts#L9
// Related JS trace: scripts/trace/workflow-event-based/approved.ts.
func TestHandler_StoppedActorAbandonsPromise(t *testing.T) {
	var out syncBuffer
	actor := xs.CreateActor(wf.NewMachine(wf.Actors(&out, 50*time.Millisecond)))
	actor.Start()
	actor.Send(xs.Ev("visaApprovedEvent"))
	actor.Stop()
	time.Sleep(150 * time.Millisecond)
	assert.Equal(t, "handleApprovedVisaWorkflowID workflow started\n", out.String())
}
