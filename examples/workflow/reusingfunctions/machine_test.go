package reusingfunctions

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

	"github.com/nguyenvanduocit/go-xstate/examples/internal/tracetest"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/require"
)

// JS reference (upstream has no separate test): https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-reusing-functions/main.ts#L34
// Expected behavior recorder: scripts/trace/workflow-reusing-functions/lib/record.ts.
func TestTraces(t *testing.T) {
	for name, failure := range map[string]string{"success": "", "insufficient": "", "funds-error": "checkfunds", "success-email-error": "sendSuccessEmail", "insufficient-email-error": "sendInsufficientFundsEmail"} {
		t.Run(name, func(t *testing.T) {
			actors := Services(io.Discard, 200*time.Millisecond)
			if failure != "" {
				actors[failure] = xs.FromPromise(func(ctx context.Context, _ xs.PromiseArgs) (any, error) {
					if err := wait(ctx, 200*time.Millisecond); err != nil {
						return nil, err
					}
					return nil, errors.New("service failed")
				})
			}
			var mu sync.Mutex
			events := []any{}
			parent := xs.CreateActor(NewParent(NewMachine(actors), nil, func(e xs.Event) {
				mu.Lock()
				defer mu.Unlock()
				switch event := e.(type) {
				case xs.DoneActorEvent:
					events = append(events, map[string]any{"type": event.EventType(), "actorId": event.ActorID})
				case xs.ErrorActorEvent:
					events = append(events, map[string]any{"type": event.EventType(), "actorId": event.ActorID, "error": map[string]any{}})
				default:
					events = append(events, e)
				}
			}))
			parent.Subscribe(xs.Observer[*xs.MachineSnapshot[ParentContext]]{Error: func(any) {}})
			parent.Start()
			defer parent.Stop()
			child := parent.GetSnapshot().Children["paymentconfirmation"]
			raw, err := os.ReadFile("testdata/" + name + ".golden.json")
			require.NoError(t, err)
			var golden struct {
				Steps []struct {
					Step     json.RawMessage `json:"step"`
					Snapshot map[string]any  `json:"snapshot"`
				} `json:"steps"`
			}
			require.NoError(t, json.Unmarshal(raw, &golden))
			for i, step := range golden.Steps {
				if i > 0 {
					var action struct {
						Send map[string]any `json:"send"`
						Wait int            `json:"wait"`
					}
					require.NoError(t, json.Unmarshal(step.Step, &action))
					if action.Send != nil {
						parent.Send(xs.E(action.Send))
					} else {
						time.Sleep(time.Duration(action.Wait) * time.Millisecond)
					}
				}
				mu.Lock()
				recorded := append([]any{}, events...)
				mu.Unlock()
				got := map[string]any{"parent": tracetest.View(parent.GetSnapshot()), "child": tracetest.View(child.AnySnapshot()), "events": recorded}
				b, err := json.Marshal(got)
				require.NoError(t, err)
				var normalized map[string]any
				require.NoError(t, json.Unmarshal(b, &normalized))
				require.Equal(t, step.Snapshot, normalized, "step %d", i)
			}
		})
	}
}

// JS reference (upstream has no separate test): https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-reusing-functions/main.ts#L194
// Expected behavior recorder: scripts/trace/workflow-reusing-functions/workflow-reusing-functions.stdout.ts.
func TestStdout(t *testing.T) {
	want, err := os.ReadFile("testdata/workflow-reusing-functions.stdout.txt")
	require.NoError(t, err)
	var out strings.Builder
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	require.NoError(t, RunWith(ctx, &out, time.Millisecond))
	require.Equal(t, string(want), out.String())
}
