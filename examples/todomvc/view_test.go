package todomvc_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	todomvc "github.com/nguyenvanduocit/go-xstate/examples/todomvc"
)

// TestDerived replays testdata/derived.golden.json, recorded by
// scripts/trace/todomvc-react/derived.ts: filterTodos and the values <Todos> derives
// from the context, plus the URL hash handling.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/todomvc-react/src/Todos.tsx#L8
// JS trace: scripts/trace/todomvc-react/derived.ts.
// JS trace: scripts/trace/todomvc-react/todos.ts.
func TestDerived(t *testing.T) {
	raw, err := os.ReadFile("testdata/derived.golden.json")
	require.NoError(t, err)
	var g struct {
		Cases []struct {
			Filter  string             `json:"filter"`
			Todos   []todomvc.TodoItem `json:"todos"`
			Derived map[string]any     `json:"derived"`
		} `json:"cases"`
		HashCases []struct {
			Hash         string         `json:"hash"`
			Filter       string         `json:"filter"`
			InitialEvent map[string]any `json:"initialEvent"`
		} `json:"hashCases"`
	}
	require.NoError(t, json.Unmarshal(raw, &g))
	require.NotEmpty(t, g.Cases)
	require.NotEmpty(t, g.HashCases)

	for _, c := range g.Cases {
		require.Equal(t, c.Derived, roundTrip(t, todomvc.Derive(c.Filter, c.Todos)), "filter %q todos %v", c.Filter, c.Todos)
	}
	for _, c := range g.HashCases {
		require.Equal(t, c.Filter, todomvc.FilterFromHash(c.Hash), "hash %q", c.Hash)
		event, ok := todomvc.InitialHashEvent(c.Hash)
		if c.InitialEvent == nil {
			require.False(t, ok, "hash %q", c.Hash)
			continue
		}
		require.True(t, ok, "hash %q", c.Hash)
		require.Equal(t, c.InitialEvent, map[string]any(event), "hash %q", c.Hash)
	}
}
