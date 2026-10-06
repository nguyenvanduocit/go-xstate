package purchaseorderdeadline

import (
	"context"
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

// JS reference (upstream has no separate test): https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-purchase-order-deadline/main.ts#L29
// Expected behavior recorder: scripts/trace/workflow-purchase-order-deadline/lib/record.ts.
func TestTraces(t *testing.T) {
	for _, name := range []string{"success", "deadline-start", "deadline-confirmation", "deadline-shipping", "cancel-error"} {
		t.Run(name, func(t *testing.T) {
			cancel := xs.FromPromise(func(ctx context.Context, _ xs.PromiseArgs) (any, error) {
				if err := sleep(ctx, 100*time.Millisecond); err != nil {
					return nil, err
				}
				if name == "cancel-error" {
					return nil, errors.New("cancel failed")
				}
				return nil, nil
			})
			tracetest.Run(t, "testdata/"+name+".golden.json", func(clock xs.Clock, _ any) *xs.Actor[*xs.MachineSnapshot[Context]] {
				actor := xs.CreateActor(NewMachine(io.Discard, cancel), xs.WithClock(clock))
				actor.Subscribe(xs.Observer[*xs.MachineSnapshot[Context]]{Error: func(any) {}})
				return actor
			})
		})
	}
}

// JS reference (upstream has no separate test): https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-purchase-order-deadline/main.ts#L126
// Expected behavior recorder: scripts/trace/workflow-purchase-order-deadline/workflow-purchase-order-deadline.stdout.ts.
func TestStdout(t *testing.T) {
	want, err := os.ReadFile("testdata/workflow-purchase-order-deadline.stdout.txt")
	require.NoError(t, err)
	var out strings.Builder
	require.NoError(t, RunWith(context.Background(), &out, 0.01))
	require.Equal(t, string(want), out.String())
}
