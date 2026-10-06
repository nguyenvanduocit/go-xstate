package friends_test

import (
	"encoding/json"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	friends "github.com/nguyenvanduocit/go-xstate/examples/friends"
	"github.com/nguyenvanduocit/go-xstate/examples/internal/tracetest"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

type childrenGolden struct {
	Name  string `json:"name"`
	Steps []struct {
		Step     json.RawMessage           `json:"step"`
		Snapshot map[string]any            `json:"snapshot"`
		Actors   map[string]map[string]any `json:"actors"`
	} `json:"steps"`
}

// roundTrip normalises a value the way JSON.stringify does for the golden file.
func roundTrip(t *testing.T, v any) map[string]any {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	var out map[string]any
	require.NoError(t, json.Unmarshal(b, &out))
	return out
}

// TestFriendsChildrenTrace replays testdata/friends-children.golden.json, recorded by
// scripts/trace/friends-list-react/friends-children.ts. It drives the real
// saveUser (1 s) of the friend actors spawned by friendsMachine, and checks the
// snapshot of every friend actor seen so far after each step.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/friends-list-react/src/friendsMachine.ts#L6
// JS trace: scripts/trace/friends-list-react/friends-children.ts.
func TestFriendsChildrenTrace(t *testing.T) {
	raw, err := os.ReadFile("testdata/friends-children.golden.json")
	require.NoError(t, err)
	var g childrenGolden
	require.NoError(t, json.Unmarshal(raw, &g))
	require.NotEmpty(t, g.Steps)

	actor := xs.CreateActor(friends.FriendsMachine(seqIDs("xkxayr", "bta69r", "a4v5h4")))
	defer actor.Stop()
	actor.Start()

	seen := map[string]xs.ActorRef{}
	for i, s := range g.Steps {
		var name string
		if json.Unmarshal(s.Step, &name) == nil {
			require.Equal(t, "start", name)
		} else {
			var step struct {
				Send map[string]any `json:"send"`
				To   *int           `json:"to"`
				Wait *float64       `json:"wait"`
			}
			require.NoError(t, json.Unmarshal(s.Step, &step))
			switch {
			case step.Wait != nil:
				time.Sleep(time.Duration(*step.Wait * float64(time.Millisecond)))
			case step.To != nil:
				actor.GetSnapshot().Context.Friends[*step.To].Send(xs.E(step.Send))
			default:
				actor.Send(xs.E(step.Send))
			}
		}
		snap := actor.GetSnapshot()
		for _, ref := range snap.Context.Friends {
			seen[ref.ID()] = ref
		}
		require.Equal(t, roundTrip(t, s.Snapshot), roundTrip(t, tracetest.View(snap)), "step %d (%s): root", i, s.Step)

		ids := make([]string, 0, len(seen))
		for id := range seen {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		got := map[string]any{}
		for _, id := range ids {
			got[id] = tracetest.View(seen[id].AnySnapshot())
		}
		want := map[string]any{}
		for id, v := range s.Actors {
			want[id] = v
		}
		require.Equal(t, roundTrip(t, want), roundTrip(t, got), "step %d (%s): friend actors", i, s.Step)
	}
}
