package tictactoe_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	game "github.com/nguyenvanduocit/go-xstate/examples/store/tictactoe"
	xstore "github.com/nguyenvanduocit/go-xstate/store"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// asJSON round-trips v through JSON, as JS JSON.stringify does for the golden
// files (nil/empty marks become null, structs become objects, ints float64).
func asJSON(t *testing.T, v any) any {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	var out any
	require.NoError(t, json.Unmarshal(b, &out))
	return out
}

// TestStoreTrace replays testdata/store.golden.json, recorded by
// scripts/trace/store-tic-tac-toe/store.ts (gameStore + inspect + the
// three selectors of App.tsx).
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/store-tic-tac-toe/src/store.ts#L51
// JS trace: scripts/trace/store-tic-tac-toe/outcome.ts.
// JS trace: scripts/trace/store-tic-tac-toe/store.ts.
func TestStoreTrace(t *testing.T) {
	raw, err := os.ReadFile("testdata/store.golden.json")
	require.NoError(t, err)
	var g struct {
		Steps []struct {
			Step     json.RawMessage `json:"step"`
			Snapshot any             `json:"snapshot"`
		} `json:"steps"`
	}
	require.NoError(t, json.Unmarshal(raw, &g))
	require.NotEmpty(t, g.Steps)

	gm := game.NewGame()
	inspected := []map[string]any{}
	notified := map[string][]any{"board": {}, "currentPlayer": {}, "status": {}}
	gm.Store.Inspect(func(e xstore.StoreInspectionEvent) {
		inspected = append(inspected, map[string]any{
			"type":    e.Type,
			"event":   e.Event,
			"context": e.Snapshot.(*xstore.StoreSnapshot[game.Context]).Context,
		})
	})
	gm.Board.SubscribeNext(func(v [9]game.Mark) { notified["board"] = append(notified["board"], v) })
	gm.CurrentPlayer.SubscribeNext(func(v game.Mark) { notified["currentPlayer"] = append(notified["currentPlayer"], v) })
	gm.Status.SubscribeNext(func(v game.Status) { notified["status"] = append(notified["status"], v) })

	view := func() any {
		s := gm.Store.GetSnapshot()
		out := asJSON(t, map[string]any{
			"status":    s.Status,
			"context":   s.Context,
			"outcome":   game.GetGameOutcome(s.Context.Board),
			"inspected": inspected,
			"notified":  notified,
		})
		inspected = []map[string]any{}
		notified = map[string][]any{"board": {}, "currentPlayer": {}, "status": {}}
		return out
	}

	for i, s := range g.Steps {
		var name string
		if json.Unmarshal(s.Step, &name) != nil {
			var step struct {
				Send map[string]any `json:"send"`
			}
			require.NoError(t, json.Unmarshal(s.Step, &step))
			gm.Store.Send(xs.E(step.Send))
		}
		require.Equal(t, s.Snapshot, view(), "step %d (%s)", i, s.Step)
	}
}

// TestOutcomeTable replays testdata/outcome.golden.json, recorded by
// scripts/trace/store-tic-tac-toe/outcome.ts.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/store-tic-tac-toe/src/store.ts#L25
// JS trace: scripts/trace/store-tic-tac-toe/outcome.ts.
func TestOutcomeTable(t *testing.T) {
	raw, err := os.ReadFile("testdata/outcome.golden.json")
	require.NoError(t, err)
	var g struct {
		Cases []struct {
			Board   [9]game.Mark `json:"board"`
			Outcome any          `json:"outcome"`
		} `json:"cases"`
	}
	require.NoError(t, json.Unmarshal(raw, &g))
	require.NotEmpty(t, g.Cases)
	for i, c := range g.Cases {
		require.Equal(t, c.Outcome, asJSON(t, game.GetGameOutcome(c.Board)), "case %d %v", i, c.Board)
	}
}

// TestGoCallerPayload checks the int position path used by Go callers (traces
// carry float64) and the positions JS treats as "not null": fractional and missing.
// Go regression; expected values are Go assertions, not a recorded JS test.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/store-tic-tac-toe/src/store.ts#L51
// Related JS trace: scripts/trace/store-tic-tac-toe/outcome.ts.
func TestGoCallerPayload(t *testing.T) {
	gm := game.NewGame()
	gm.Store.Trigger("played", xs.E{"position": 4})
	gm.Store.Send(xs.E{"type": "played", "position": 4})   // taken
	gm.Store.Send(xs.E{"type": "played", "position": 1.5}) // JS board[1.5] is undefined
	gm.Store.Send(xs.E{"type": "played"})                  // JS board[undefined] is undefined
	c := gm.Store.GetSnapshot().Context
	require.Equal(t, game.Mark(game.X), c.Board[4])
	require.Equal(t, game.O, c.CurrentPlayer)
	require.Equal(t, game.Playing, c.Status)
	gm.Store.Trigger("reset")
	require.Equal(t, game.InitialState, gm.Store.GetSnapshot().Context)
}
