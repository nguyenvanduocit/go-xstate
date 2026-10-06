package sendcloudevent

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/nguyenvanduocit/go-xstate/examples/internal/tracetest"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/require"
)

// JS reference (upstream has no separate test): https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-send-cloudevent/main.ts#L10
// Expected behavior recorder: scripts/trace/workflow-send-cloudevent/lib/record.ts.
func TestTraces(t *testing.T) {
	for _, name := range []string{"orders", "empty", "error"} {
		t.Run(name, func(t *testing.T) {
			var provision xs.ActorLogic = ProvisionOrders(io.Discard, 100*time.Millisecond)
			if name == "empty" {
				provision = xs.FromPromise(func(context.Context, xs.PromiseArgs) ([]Result, error) {
					time.Sleep(100 * time.Millisecond)
					return []Result{}, nil
				})
			}
			if name == "error" {
				provision = xs.FromPromise(func(context.Context, xs.PromiseArgs) (any, error) {
					time.Sleep(100 * time.Millisecond)
					return nil, errors.New("provision failed")
				})
			}
			tracetest.Run(t, "testdata/"+name+".golden.json", func(_ xs.Clock, input any) *xs.Actor[*xs.MachineSnapshot[Context]] {
				b, err := json.Marshal(input)
				require.NoError(t, err)
				var in Input
				require.NoError(t, json.Unmarshal(b, &in))
				actor := xs.CreateActor(NewMachine(provision), xs.WithInput(in))
				actor.Subscribe(xs.Observer[*xs.MachineSnapshot[Context]]{Error: func(any) {}})
				return actor
			})
		})
	}
}

// JS reference (upstream has no separate test): https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-send-cloudevent/main.ts#L75
// Expected behavior recorder: scripts/trace/workflow-send-cloudevent/workflow-send-cloudevent.stdout.ts.
func TestStdout(t *testing.T) {
	want, err := os.ReadFile("testdata/workflow-send-cloudevent.stdout.txt")
	require.NoError(t, err)
	var out strings.Builder
	require.NoError(t, RunWith(context.Background(), &out, time.Millisecond))
	require.Equal(t, string(want), out.String())
}

// JS reference (upstream has no separate test): https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-send-cloudevent/main.ts#L26
// Expected behavior recorder: scripts/trace/workflow-send-cloudevent/empty-service.stdout.ts.
func TestEmptyService(t *testing.T) {
	want, err := os.ReadFile("testdata/empty-service.stdout.txt")
	require.NoError(t, err)
	actor := xs.CreateActor(ProvisionOrders(io.Discard, time.Second), xs.WithInput(Input{Orders: []Order{}}))
	done := make(chan struct{})
	actor.Subscribe(xs.Observer[*xs.PromiseSnapshot[[]Result]]{Complete: func() { close(done) }})
	actor.Start()
	defer actor.Stop()
	<-done
	output, err := json.Marshal(actor.GetSnapshot().GetOutput())
	require.NoError(t, err)
	require.Equal(t, string(want), string(output)+"\n")
}
