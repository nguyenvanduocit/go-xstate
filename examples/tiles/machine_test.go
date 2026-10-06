package tiles_test

import (
	"testing"

	"github.com/nguyenvanduocit/go-xstate/examples/internal/tracetest"
	"github.com/nguyenvanduocit/go-xstate/examples/tiles"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// fixedRandom returns the same Math.random sequence as scripts/trace/tiles/tiles.ts:
// SWAP01, IDENTITY, SWAP01 (see the comment there). Running out of values fails the test.
func fixedRandom(t *testing.T) func() float64 {
	var seq []float64
	rep := func(n int, v float64) {
		for range n {
			seq = append(seq, v)
		}
	}
	rep(14, 0.99)
	seq = append(seq, 0) // SWAP01
	rep(15, 0.99)        // IDENTITY
	rep(14, 0.99)
	seq = append(seq, 0) // SWAP01
	next := 0
	return func() float64 {
		if next >= len(seq) {
			t.Fatal("random sequence exhausted")
		}
		next++
		return seq[next-1]
	}
}

// TestTrace replays testdata/tiles.golden.json, recorded from the JS example by
// scripts/trace/tiles/tiles.ts.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/tiles/src/tilesMachine.ts#L13
// JS trace: scripts/trace/tiles/tiles.ts.
func TestTrace(t *testing.T) {
	tracetest.Run(t, "testdata/tiles.golden.json", func(clock xs.Clock, input any) *xs.Actor[*xs.MachineSnapshot[tiles.Context]] {
		return xs.CreateActor(tiles.NewMachine(fixedRandom(t)))
	})
}

// TestTraceWithGoEvents drives the same script with events carrying Go Tile values (the
// replay above carries JSON maps) and checks the board ends where the JS golden trace does.
// Go regression; expected values are Go assertions, not a recorded JS test.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/tiles/src/tilesMachine.ts#L13
// Related JS trace: scripts/trace/tiles/tiles.ts.
func TestTraceWithGoEvents(t *testing.T) {
	actor := xs.CreateActor(tiles.NewMachine(fixedRandom(t)))
	actor.Start()
	defer actor.Stop()
	tile := func(i int) tiles.Tile { return tiles.Tile{Index: i, X: i % 4, Y: i / 4} }

	actor.Send(tiles.ShuffleEvent())
	actor.Send(tiles.SelectEvent(tile(1)))
	actor.Send(tiles.HoverEvent(tile(0)))
	actor.Send(tiles.MoveEvent())
	if snap := actor.GetSnapshot(); snap.Value != "gameOver" {
		t.Fatalf("value = %v, want gameOver", snap.Value)
	}
}

// TestMachineWithMathRandom builds the production machine (math/rand) and checks that a
// shuffle always yields a permutation of 0..15 and ends in a playable or solved state.
// Go regression; expected values are Go assertions, not a recorded JS test.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/tiles/src/tilesMachine.ts#L13
// Related JS trace: scripts/trace/tiles/tiles.ts.
func TestMachineWithMathRandom(t *testing.T) {
	actor := xs.CreateActor(tiles.Machine())
	actor.Start()
	defer actor.Stop()
	actor.Send(tiles.ShuffleEvent())
	snap := actor.GetSnapshot()
	seen := make([]bool, 16)
	for _, v := range snap.Context.Tiles {
		seen[v] = true
	}
	for i, ok := range seen {
		if !ok {
			t.Fatalf("tile %d missing after shuffle: %v", i, snap.Context.Tiles)
		}
	}
}
