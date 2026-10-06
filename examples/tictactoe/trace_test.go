package tictactoe_test

import (
	"path/filepath"
	"testing"

	"github.com/nguyenvanduocit/go-xstate/examples/internal/tracetest"
	tictactoereact "github.com/nguyenvanduocit/go-xstate/examples/tictactoe"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// TestTraces replays every testdata/*.golden.json, recorded from the JS example by
// scripts/trace/tic-tac-toe-react/<name>.ts.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/tic-tac-toe-react/src/ticTacToeMachine.ts#L21
// JS trace: scripts/trace/tic-tac-toe-react/draw.ts.
// JS trace: scripts/trace/tic-tac-toe-react/o-wins.ts.
// JS trace: scripts/trace/tic-tac-toe-react/x-wins.ts.
// JS trace: scripts/trace/tic-tac-toe-react/win-on-last-move.ts.
func TestTraces(t *testing.T) {
	files, err := filepath.Glob("testdata/*.golden.json")
	if err != nil || len(files) != 4 {
		t.Fatalf("want 4 golden files, got %v (err %v)", files, err)
	}
	for _, f := range files {
		t.Run(filepath.Base(f), func(t *testing.T) {
			tracetest.Run(t, f, func(clock xs.Clock, input any) *xs.Actor[*xs.MachineSnapshot[tictactoereact.Context]] {
				return xs.CreateActor(tictactoereact.Machine())
			})
		})
	}
}

// TestGoEvents drives the machine with Play(int) events (the goldens deliver
// float64 values) and checks that an earlier snapshot's board is never mutated.
// Go regression; expected values are Go assertions, not a recorded JS test.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/tic-tac-toe-react/src/ticTacToeMachine.ts#L21
// Related JS trace: scripts/trace/tic-tac-toe-react/draw.ts.
func TestGoEvents(t *testing.T) {
	actor := xs.CreateActor(tictactoereact.Machine())
	actor.Start()
	defer actor.Stop()

	actor.Send(tictactoereact.Play(4))
	first := actor.GetSnapshot()
	actor.Send(tictactoereact.Play(4)) // occupied
	actor.Send(tictactoereact.Play(0))
	if got := actor.GetSnapshot().Context; got.Moves != 2 || got.Board[4] != tictactoereact.X || got.Board[0] != tictactoereact.O || got.Player != tictactoereact.X {
		t.Fatalf("unexpected context %+v", got)
	}
	if first.Context.Moves != 1 || first.Context.Board[0] != "" {
		t.Fatalf("earlier snapshot was mutated: %+v", first.Context)
	}
	actor.Send(tictactoereact.Reset()) // ignored while playing
	if actor.GetSnapshot().Context.Moves != 2 {
		t.Fatal("RESET must be ignored in playing")
	}
}
