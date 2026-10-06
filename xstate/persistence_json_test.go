package xstate_test

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"testing"
	"time"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/require"
)

// Go regression: Promise JSON encoding preserves lifecycle fields and resolved zero values; no direct upstream testcase.
// Related JS implementation: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/actors/promise.ts#L228
func TestPromiseSnapshotJSON(t *testing.T) {
	for _, tt := range []struct {
		name     string
		snapshot any
		want     string
	}{
		{"active", &xs.PromiseSnapshot[int]{Status: xs.StatusActive}, `{"status":"active"}`},
		{"active input", xs.PromiseSnapshot[int]{Status: xs.StatusActive, Input: false}, `{"status":"active","input":false}`},
		{"done zero", &xs.PromiseSnapshot[int]{Status: xs.StatusDone}, `{"status":"done","output":0}`},
		{"done false", xs.PromiseSnapshot[bool]{Status: xs.StatusDone}, `{"status":"done","output":false}`},
		{"done empty string", xs.PromiseSnapshot[string]{Status: xs.StatusDone}, `{"status":"done","output":""}`},
		{"done nil", xs.PromiseSnapshot[any]{Status: xs.StatusDone}, `{"status":"done","output":null}`},
		{"error", &xs.PromiseSnapshot[int]{Status: xs.StatusError, Error: "failed"}, `{"status":"error","error":"failed"}`},
		{"stopped", &xs.PromiseSnapshot[int]{Status: xs.StatusStopped}, `{"status":"stopped"}`},
	} {
		// Go regression case; related JS implementation: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/actors/promise.ts#L228
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.snapshot)
			require.NoError(t, err)
			require.JSONEq(t, tt.want, string(data))
		})
	}
}

// Go regression: Promise JSON persistence and restore round trip; no direct upstream testcase.
// Related JS implementation: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/actors/promise.ts#L228
func TestPromiseSnapshotJSONRestore(t *testing.T) {
	var calls atomic.Int32
	logic := xs.FromPromise(func(_ context.Context, args xs.PromiseArgs) (int, error) {
		calls.Add(1)
		return int(args.Input.(float64)) - 7, nil
	})
	actor := xs.CreateActor(logic, xs.WithInput(7))
	t.Cleanup(func() { actor.Stop() })

	roundTrip := func(value any, want string) any {
		t.Helper()
		data, err := json.Marshal(value)
		require.NoError(t, err)
		require.JSONEq(t, want, string(data))
		var restored any
		require.NoError(t, json.Unmarshal(data, &restored))
		return restored
	}
	active := roundTrip(actor.GetPersistedSnapshot(), `{"status":"active","input":7}`)
	restored := xs.CreateActor(logic, xs.WithSnapshot(active)).Start()
	t.Cleanup(func() { restored.Stop() })
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	snapshot, err := xs.WaitFor(ctx, restored, func(s *xs.PromiseSnapshot[int]) bool {
		return s.Status == xs.StatusDone
	}).Wait()
	require.NoError(t, err)
	require.Zero(t, snapshot.Output)
	require.Nil(t, snapshot.Input)

	done := roundTrip(restored.GetPersistedSnapshot(), `{"status":"done","output":0}`)
	completed := xs.CreateActor(logic, xs.WithSnapshot(done)).Start()
	t.Cleanup(func() { completed.Stop() })
	require.Equal(t, xs.StatusDone, completed.GetSnapshot().Status)
	require.Zero(t, completed.GetSnapshot().Output)
	require.Equal(t, int32(1), calls.Load())
}
