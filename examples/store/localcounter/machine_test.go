package localcounter_test

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nguyenvanduocit/go-xstate/examples/internal/tracetest"
	counter "github.com/nguyenvanduocit/go-xstate/examples/store/localcounter"
	xstore "github.com/nguyenvanduocit/go-xstate/store"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// TestActorTrace replays testdata/counter-<n>.golden.json, recorded by
// scripts/trace/local-store-counter-react/counter-<n>.ts (fromStore actor).
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/local-store-counter-react/src/App.tsx#L4
// JS trace: scripts/trace/local-store-counter-react/counter-0.ts.
// JS trace: scripts/trace/local-store-counter-react/counter-10.ts.
// JS trace: scripts/trace/local-store-counter-react/counter-100.ts.
func TestActorTrace(t *testing.T) {
	for _, n := range counter.InitialCounts {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			tracetest.Run(t, fmt.Sprintf("testdata/counter-%d.golden.json", n), func(clock xs.Clock, input any) *xs.Actor[*xstore.StoreSnapshot[counter.Context]] {
				return xs.CreateActor(xstore.FromStore(counter.Config(n)))
			})
		})
	}
}

type storeView struct {
	Status  string          `json:"status"`
	Context counter.Context `json:"context"`
	Count   int             `json:"count"`
}

// TestStoreTrace replays testdata/store.golden.json, recorded by
// scripts/trace/local-store-counter-react/store.ts (createStore + select).
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/local-store-counter-react/src/App.tsx#L4
// JS trace: scripts/trace/local-store-counter-react/store.ts.
func TestStoreTrace(t *testing.T) {
	raw, err := os.ReadFile("testdata/store.golden.json")
	require.NoError(t, err)
	var g struct {
		Runs []struct {
			InitialCount int `json:"initialCount"`
			Steps        []struct {
				Step     json.RawMessage `json:"step"`
				Snapshot storeView       `json:"snapshot"`
			} `json:"steps"`
		} `json:"runs"`
	}
	require.NoError(t, json.Unmarshal(raw, &g))
	require.Len(t, g.Runs, len(counter.InitialCounts))

	for _, run := range g.Runs {
		t.Run(fmt.Sprint(run.InitialCount), func(t *testing.T) {
			c := counter.NewCounter(run.InitialCount)
			view := func() storeView {
				s := c.Store.GetSnapshot()
				return storeView{Status: string(s.Status), Context: s.Context, Count: c.Count.Get()}
			}
			for i, s := range run.Steps {
				var name string
				if json.Unmarshal(s.Step, &name) != nil {
					var step struct {
						Send map[string]any `json:"send"`
					}
					require.NoError(t, json.Unmarshal(s.Step, &step))
					c.Store.Send(xs.E(step.Send))
				}
				require.Equal(t, s.Snapshot, view(), "step %d (%s)", i, s.Step)
			}
		})
	}
}

// TestGoCallerPayload checks the int payload path used by Go callers (traces carry float64).
// Go regression; expected values are Go assertions, not a recorded JS test.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/local-store-counter-react/src/App.tsx#L4
// Related JS trace: scripts/trace/local-store-counter-react/counter-0.ts.
func TestGoCallerPayload(t *testing.T) {
	c := counter.NewCounter(10)
	var seen []int
	c.Count.SubscribeNext(func(n int) { seen = append(seen, n) })
	c.Store.Trigger("inc", xs.E{"by": 2})
	c.Store.Send(xs.E{"type": "inc", "by": 3})
	c.Store.Trigger("reset")
	require.Equal(t, []int{12, 15, 0}, seen)
}
