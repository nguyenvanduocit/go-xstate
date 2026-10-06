package provisionorders_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nguyenvanduocit/go-xstate/examples/internal/tracetest"
	wpo "github.com/nguyenvanduocit/go-xstate/examples/workflow/provisionorders"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// traceDelay is the actors' delay in the JS scripts (scripts/trace/workflow-provision-orders/lib/record.ts
// shortens setTimeout(…, 1000) to 50 ms). Snapshots are read at 25, 75 and 150 ms.
const traceDelay = 50 * time.Millisecond

// replayTrace replays testdata/<name>.golden.json with the example's own actors at traceDelay.
// The golden file carries the machine input.
func replayTrace(t *testing.T, name string, machine func() *xs.StateMachine[wpo.Context]) {
	var errs []any
	var mu sync.Mutex
	tracetest.Run(t, "testdata/"+name+".golden.json", func(_ xs.Clock, input any) *xs.Actor[*xs.MachineSnapshot[wpo.Context]] {
		raw, err := json.Marshal(input)
		require.NoError(t, err)
		var in wpo.Input
		require.NoError(t, json.Unmarshal(raw, &in))
		return xs.CreateActor(machine(), xs.WithInput(in), xs.WithUnhandledErrorHandler(func(e any) {
			mu.Lock()
			defer mu.Unlock()
			errs = append(errs, e)
		}))
	})
	_ = errs
}

func realActors() *xs.StateMachine[wpo.Context] { return wpo.NewMachine(io.Discard, traceDelay) }

// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-provision-orders/main.ts#L10
// JS trace: scripts/trace/workflow-provision-orders/success.ts.
func TestTraceSuccess(t *testing.T) { replayTrace(t, "success", realActors) }

// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-provision-orders/main.ts#L10
// JS trace: scripts/trace/workflow-provision-orders/missing-id.ts.
func TestTraceMissingID(t *testing.T) { replayTrace(t, "missing-id", realActors) }

// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-provision-orders/main.ts#L10
// JS trace: scripts/trace/workflow-provision-orders/missing-item.ts.
func TestTraceMissingItem(t *testing.T) { replayTrace(t, "missing-item", realActors) }

// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-provision-orders/main.ts#L10
// JS trace: scripts/trace/workflow-provision-orders/missing-quantity.ts.
func TestTraceMissingQuantity(t *testing.T) { replayTrace(t, "missing-quantity", realActors) }

// TestTraceOtherError uses the same stub as other-error.ts: provisionOrderFunction rejects with a
// message that matches none of the three onError guards, so the machine errors out.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-provision-orders/main.ts#L10
// JS trace: scripts/trace/workflow-provision-orders/other-error.ts.
func TestTraceOtherError(t *testing.T) {
	replayTrace(t, "other-error", func() *xs.StateMachine[wpo.Context] {
		return realActors().Provide(xs.Implementations{Actors: map[string]xs.ActorLogic{
			"provisionOrderFunction": xs.FromPromise(func(context.Context, xs.PromiseArgs) (any, error) {
				time.Sleep(traceDelay)
				return nil, errors.New("Order service unavailable")
			}),
		}})
	})
}

// TestStdout compares the printing entry with testdata/workflow-provision-orders.stdout.txt,
// recorded from main.ts. The delay is shortened; the text does not depend on it.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-provision-orders/main.ts#L135
// JS trace: scripts/trace/workflow-provision-orders/workflow-provision-orders.stdout.ts.
func TestStdout(t *testing.T) {
	want, err := os.ReadFile("testdata/workflow-provision-orders.stdout.txt")
	require.NoError(t, err)
	var out strings.Builder
	wpo.RunWith(&out, wpo.DefaultInput, 10*time.Millisecond)
	assert.Equal(t, string(want), out.String())
}

// The JS entry hardcodes an empty id; the other paths print as follows.
// Go regression; expected values are Go assertions, not a recorded JS test.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-provision-orders/main.ts#L10
// Related JS trace: scripts/trace/workflow-provision-orders/workflow-provision-orders.stdout.ts.
func TestRunOutput(t *testing.T) {
	cases := map[string]struct {
		order wpo.Order
		want  string
	}{
		"success": {
			wpo.Order{ID: "o-1", Item: "laptop", Quantity: "10"},
			"starting provisionOrderFunction\nfinished provisionOrderFunction\nstarting applyOrderWorkflowId\nfinished applyOrderWorkflowId\nworkflow completed undefined\n",
		},
		"missing item": {
			wpo.Order{ID: "o-1", Quantity: "10"},
			"starting provisionOrderFunction\nstarting handleMissingItemExceptionWorkflow\nfinished handleMissingItemExceptionWorkflow\nworkflow completed undefined\n",
		},
		"missing quantity": {
			wpo.Order{ID: "o-1", Item: "laptop"},
			"starting provisionOrderFunction\nstarting handleMissingQuantityExceptionWorkflow\nfinished handleMissingQuantityExceptionWorkflow\nworkflow completed undefined\n",
		},
		"id is checked before item": {
			wpo.Order{},
			"starting provisionOrderFunction\nstarting handleMissingIdExceptionWorkflow\nfinished handleMissingIdExceptionWorkflow\nworkflow completed undefined\n",
		},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			var out strings.Builder
			wpo.RunWith(&out, wpo.Input{Order: c.order}, 5*time.Millisecond)
			assert.Equal(t, c.want, out.String())
		})
	}
}
