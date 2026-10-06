package persistedstate_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nguyenvanduocit/go-xstate/examples/internal/tracetest"
	donut "github.com/nguyenvanduocit/go-xstate/examples/server/persistedstate"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// TestTrace replays testdata/donut.golden.json, recorded from the JS example by
// scripts/trace/mongodb-persisted-state/donut.ts.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/mongodb-persisted-state/donutMachine.ts#L3
// JS trace: scripts/trace/mongodb-persisted-state/donut.ts.
func TestTrace(t *testing.T) {
	tracetest.Run(t, "testdata/donut.golden.json", func(clock xs.Clock, input any) *xs.Actor[donut.Snapshot] {
		return xs.CreateActor(donut.DonutMachine())
	})
}

// TestTraceRestored replays testdata/donut-restored.golden.json, recorded by
// scripts/trace/mongodb-persisted-state/donut-restored.ts: the golden's `input` is the
// persisted snapshot the actor is restored from.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/mongodb-persisted-state/donutMachine.ts#L3
// JS trace: scripts/trace/mongodb-persisted-state/donut-restored.ts.
func TestTraceRestored(t *testing.T) {
	tracetest.Run(t, "testdata/donut-restored.golden.json", func(clock xs.Clock, input any) *xs.Actor[donut.Snapshot] {
		return xs.CreateActor(donut.DonutMachine(), xs.WithSnapshot(input))
	})
}

// TestPersistedSnapshotMatchesJS compares the snapshot persisted by a Go actor with the
// one the JS actor persisted at the same point (the `input` of the restored golden).
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/mongodb-persisted-state/donutMachine.ts#L3
// JS trace: scripts/trace/mongodb-persisted-state/donut-restored.ts.
func TestPersistedSnapshotMatchesJS(t *testing.T) {
	raw, err := os.ReadFile("testdata/donut-restored.golden.json")
	require.NoError(t, err)
	var golden struct {
		Input map[string]any `json:"input"`
	}
	require.NoError(t, json.Unmarshal(raw, &golden))

	actor := xs.CreateActor(donut.DonutMachine()).Start()
	defer actor.Stop()
	for _, e := range []string{"NEXT", "NEXT", "MIXED_DRY"} {
		actor.Send(xs.Ev(e))
	}
	persisted, err := json.Marshal(actor.GetPersistedSnapshot())
	require.NoError(t, err)
	var got map[string]any
	require.NoError(t, json.Unmarshal(persisted, &got))
	// JS drops the undefined `output` and `error` when serializing; Go's nil is the same value.
	for k, v := range got {
		if v == nil {
			delete(got, k)
		}
	}
	require.Equal(t, golden.Input, got)
}
