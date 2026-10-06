package badukmatch

import (
	"context"
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/require"
)

func TestSetupTurnAlternationAndScoringBeforeMoveLimit(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var turns []Turn
	match, err := Play(ctx, Config{Komi: 7.5, MaxMoves: 2}, func(_ context.Context, turn Turn) (Decision, error) {
		turns = append(turns, turn)
		return Decision{Move: "pass", CostUSD: .001}, nil
	}, nil)
	require.NoError(t, err)
	require.Len(t, turns, 2)
	require.Equal(t, "black", turns[0].Side)
	require.Equal(t, Clef, turns[0].Model)
	require.Equal(t, Jev, turns[1].Model)
	require.Equal(t, turns[0].Board, turns[1].Board)
	require.Equal(t, 1, turns[1].ConsecutivePasses)
	require.Len(t, turns[0].LegalMoves, 82)
	require.Equal(t, []string{"black pass"}, turns[1].RecentMoves)
	require.Equal(t, strings.Repeat(".", 81), match.Cells)
	require.True(t, match.Scored)
	require.Equal(t, "W+7.5", match.Result)
	require.Equal(t, "two consecutive passes", match.Reason)
	require.InDelta(t, .002, match.CostUSD, 1e-10)
	require.Contains(t, match.SGF, "RE[W+7.5]")
	require.Contains(t, match.SGF, ";B[];W[]")
}

func TestLimitKeepsUnfinishedResultAndSGFCoordinates(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	moves := []string{"A9", "J1", "D4"}
	match, err := Play(ctx, Config{MaxMoves: 3, BlackModel: "model]\\name"}, func(_ context.Context, turn Turn) (Decision, error) {
		return Decision{Move: moves[turn.MoveNumber-1]}, nil
	}, nil)
	require.NoError(t, err)
	require.Equal(t, "*", match.Result)
	require.False(t, match.Scored)
	require.Equal(t, "move limit", match.Reason)
	require.NotContains(t, match.SGF, "RE[")
	require.Contains(t, match.SGF, ";B[aa];W[ii];B[df]")
	require.Contains(t, match.SGF, `PB[model\]\\name]`)
}

func TestChooserErrorsAndInjectedChoicesDoNotAdvanceBoard(t *testing.T) {
	for _, fail := range []bool{false, true} {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		match, err := Play(ctx, Config{}, func(_ context.Context, turn Turn) (Decision, error) {
			if fail {
				return Decision{}, errors.New("provider unavailable")
			}
			turn.LegalMoves[0] = "Z99"
			return Decision{Move: "Z99"}, nil
		}, nil)
		cancel()
		require.Error(t, err)
		require.Empty(t, match.Moves)
		require.Equal(t, strings.Repeat(".", 81), match.Cells)
		require.Equal(t, "error", match.Reason)
		require.Equal(t, "*", match.Result)
	}
}

func TestCancellationStopsModelRequest(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	started, stopped := make(chan struct{}), make(chan struct{})
	go func() { <-started; cancel() }()
	match, err := Play(ctx, Config{}, func(ctx context.Context, _ Turn) (Decision, error) {
		close(started)
		<-ctx.Done()
		close(stopped)
		return Decision{}, ctx.Err()
	}, nil)
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, "cancelled", match.Reason)
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("model work did not receive cancellation")
	}
}

func TestSnapshotsDoNotChangeAfterMoves(t *testing.T) {
	proceed := make(chan struct{})
	machine, err := NewMachine(Config{MaxMoves: 1}, func(context.Context, Turn) (Decision, error) {
		<-proceed
		return Decision{Move: "D4"}, nil
	})
	require.NoError(t, err)
	actor := xs.CreateActor(machine).Start()
	defer actor.Stop()
	before := actor.GetSnapshot().Context
	done := make(chan struct{})
	actor.Subscribe(xs.Observer[*xs.MachineSnapshot[Match]]{Complete: func() { close(done) }})
	close(proceed)
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("match did not finish")
	}
	require.Empty(t, before.Moves)
	require.Equal(t, strings.Repeat(".", 81), before.Cells)
	require.Len(t, before.LegalMoves, 82)
	require.Len(t, before.position.history, 1)
	require.Len(t, actor.GetSnapshot().Context.Moves, 1)
}

func TestInvalidConfiguration(t *testing.T) {
	for _, cfg := range []Config{{Size: 1}, {Size: 20}, {MaxMoves: -1}, {Komi: math.NaN()}, {Komi: math.Inf(1)}, {Komi: .1}} {
		_, err := NewMachine(cfg, DemoChoose)
		require.Error(t, err)
	}
	_, err := NewMachine(Config{}, nil)
	require.Error(t, err)
}
