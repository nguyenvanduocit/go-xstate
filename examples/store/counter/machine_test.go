package counter_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nguyenvanduocit/go-xstate/examples/internal/tracetest"
	counter "github.com/nguyenvanduocit/go-xstate/examples/store/counter"
	xstore "github.com/nguyenvanduocit/go-xstate/store"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// TestActorTrace replays testdata/counter.golden.json, recorded by
// scripts/trace/store-counter-react/counter.ts (fromStore actor).
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/store-counter-react/src/App.tsx#L8
// JS trace: scripts/trace/store-counter-react/counter.ts.
func TestActorTrace(t *testing.T) {
	tracetest.Run(t, "testdata/counter.golden.json", func(clock xs.Clock, input any) *xs.Actor[*xstore.StoreSnapshot[counter.Context]] {
		return xs.CreateActor(xstore.FromStore(counter.Config()))
	})
}

type inspected struct {
	Type    string          `json:"type"`
	Event   map[string]any  `json:"event"`
	Context counter.Context `json:"context"`
}

type storeView struct {
	Status    string          `json:"status"`
	Context   counter.Context `json:"context"`
	Count     int             `json:"count"`
	Inspected []inspected     `json:"inspected"`
	Notified  []int           `json:"notified"`
}

// TestStoreTrace replays testdata/store.golden.json, recorded by
// scripts/trace/store-counter-react/store.ts (createStore + inspect + select).
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/store-counter-react/src/App.tsx#L8
// JS trace: scripts/trace/store-counter-react/store.ts.
func TestStoreTrace(t *testing.T) {
	raw, err := os.ReadFile("testdata/store.golden.json")
	require.NoError(t, err)
	var g struct {
		Steps []struct {
			Step     json.RawMessage `json:"step"`
			Snapshot storeView       `json:"snapshot"`
		} `json:"steps"`
	}
	require.NoError(t, json.Unmarshal(raw, &g))
	require.NotEmpty(t, g.Steps)

	c := counter.NewCounter()
	rec := storeView{Inspected: []inspected{}, Notified: []int{}}
	c.Store.Inspect(func(e xstore.StoreInspectionEvent) {
		ev, ok := e.Event.(xs.E)
		require.True(t, ok, "inspection event is %T", e.Event)
		rec.Inspected = append(rec.Inspected, inspected{
			Type:    e.Type,
			Event:   normalizeEvent(t, ev),
			Context: e.Snapshot.(*xstore.StoreSnapshot[counter.Context]).Context,
		})
	})
	c.Count.SubscribeNext(func(n int) { rec.Notified = append(rec.Notified, n) })

	view := func() storeView {
		s := c.Store.GetSnapshot()
		out := storeView{
			Status:    string(s.Status),
			Context:   s.Context,
			Count:     c.Count.Get(),
			Inspected: rec.Inspected,
			Notified:  rec.Notified,
		}
		rec.Inspected = []inspected{}
		rec.Notified = []int{}
		return out
	}

	for i, s := range g.Steps {
		var name string
		if json.Unmarshal(s.Step, &name) != nil {
			var step struct {
				Send map[string]any `json:"send"`
			}
			require.NoError(t, json.Unmarshal(s.Step, &step))
			c.Store.Send(xs.E(step.Send))
		}
		want := s.Snapshot
		if want.Inspected == nil {
			want.Inspected = []inspected{}
		}
		if want.Notified == nil {
			want.Notified = []int{}
		}
		require.Equal(t, want, view(), "step %d (%s)", i, s.Step)
	}
}

// normalizeEvent round-trips an event through JSON so numbers compare like the golden file's.
func normalizeEvent(t *testing.T, ev xs.E) map[string]any {
	t.Helper()
	b, err := json.Marshal(ev)
	require.NoError(t, err)
	var out map[string]any
	require.NoError(t, json.Unmarshal(b, &out))
	return out
}

// TestGoCallerPayload checks the int payload path used by Go callers (traces carry float64).
// Go regression; expected values are Go assertions, not a recorded JS test.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/store-counter-react/src/App.tsx#L8
// Related JS trace: scripts/trace/store-counter-react/counter.ts.
func TestGoCallerPayload(t *testing.T) {
	c := counter.NewCounter()
	var seen []int
	c.Count.SubscribeNext(func(n int) { seen = append(seen, n) })
	c.Store.Trigger("inc", xs.E{"by": 2})
	c.Store.Send(xs.E{"type": "inc", "by": 3})
	c.Store.Trigger("reset")
	require.Equal(t, []int{2, 5, 0}, seen)
}
