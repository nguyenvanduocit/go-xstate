package snake_test

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/nguyenvanduocit/go-xstate/examples/internal/tracetest"
	snakereact "github.com/nguyenvanduocit/go-xstate/examples/snake"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// TestTrace replays testdata/snake.golden.json, recorded from the JS example by
// scripts/trace/snake-react/snake.ts. The JS script pins Math.random to a fixed
// sequence and replaces the `ticks` actor with a no-op; both are mirrored here.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/snake-react/src/snakeMachine.ts#L128
// JS trace: scripts/trace/snake-react/snake.ts.
func TestTrace(t *testing.T) {
	tracetest.Run(t, "testdata/snake.golden.json", func(clock xs.Clock, input any) *xs.Actor[*xs.MachineSnapshot[snakereact.Context]] {
		randoms := []float64{0.73, 0.48, 0.77, 0.48, 0.13, 0.21, 0.22, 0.7}
		next := 0
		random := func() float64 {
			require.Less(t, next, len(randoms), "random sequence exhausted")
			next++
			return randoms[next-1]
		}
		noTicks := xs.FromCallback(func(xs.CallbackArgs) func() { return nil })
		machine := snakereact.NewMachine(random, snakereact.TickInterval).
			Provide(xs.Implementations{Actors: map[string]xs.ActorLogic{"ticks": noTicks}})
		return xs.CreateActor(machine, xs.WithClock(clock))
	})
}

// TestGameObjectAtPos compares GameObjectAtPos with testdata/gameobjects.golden.json, recorded
// by scripts/trace/snake-react/gameobjects.ts.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/snake-react/src/snakeMachine.ts#L21
// JS trace: scripts/trace/snake-react/gameobjects.ts.
func TestGameObjectAtPos(t *testing.T) {
	raw, err := os.ReadFile("testdata/gameobjects.golden.json")
	require.NoError(t, err)
	var g struct {
		Contexts map[string]snakereact.Context `json:"contexts"`
		Cases    []struct {
			Context string           `json:"context"`
			P       snakereact.Point `json:"p"`
			Result  *struct {
				Type string         `json:"type"`
				Dir  snakereact.Dir `json:"dir"`
			} `json:"result"`
		} `json:"cases"`
	}
	require.NoError(t, json.Unmarshal(raw, &g))
	require.NotEmpty(t, g.Cases)
	seen := map[string]int{}
	for _, c := range g.Cases {
		got, ok := snakereact.GameObjectAtPos(g.Contexts[c.Context], c.P)
		if c.Result == nil {
			require.False(t, ok, "%s %v", c.Context, c.P)
			seen["none"]++
			continue
		}
		require.True(t, ok, "%s %v", c.Context, c.P)
		require.Equal(t, c.Result.Type, got.Type, "%s %v", c.Context, c.P)
		require.Equal(t, c.Result.Dir, got.Dir, "%s %v", c.Context, c.P)
		seen[c.Result.Type]++
	}
	for _, k := range []string{"head", "body", "apple", "none"} {
		require.Positive(t, seen[k], "golden covers %s", k)
	}
}

// TestRealTicks drives the machine with the real `ticks` actor (short interval): the snake
// keeps moving without any TICK sent by the test, and stops once the game is over.
// Go regression; expected values are Go assertions, not a recorded JS test.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/snake-react/src/snakeMachine.ts#L128
// Related JS trace: scripts/trace/snake-react/gameobjects.ts.
func TestRealTicks(t *testing.T) {
	actor := xs.CreateActor(snakereact.NewMachine(func() float64 { return 0.5 }, 5*time.Millisecond))
	defer actor.Stop()
	actor.Start()
	actor.Send(xs.E{"type": "ARROW_KEY", "dir": "Up"})
	// Up from (12,7): the head leaves the grid after 8 moves (1 on entry + 7 ticks).
	require.Eventually(t, func() bool {
		return actor.GetSnapshot().Matches("Game Over")
	}, 5*time.Second, time.Millisecond)
	s := actor.GetSnapshot()
	require.Equal(t, -1, s.Context.Snake[0].Y)
	require.Empty(t, s.Children)
	time.Sleep(30 * time.Millisecond)
	require.Equal(t, -1, actor.GetSnapshot().Context.Snake[0].Y)
}
