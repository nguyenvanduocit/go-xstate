package chessmatch

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	chess "github.com/corentings/chess/v2"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/require"
)

// Original Go example tests; there is no corresponding upstream XState app.
func TestFoolsMate(t *testing.T) {
	moves := []string{"f2f3", "e7e5", "g2g4", "d8h4"}
	var turns []Turn
	var observed []Ply
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	match, err := Play(ctx, Config{}, func(_ context.Context, turn Turn) (Decision, error) {
		turns = append(turns, turn)
		if len(turns) > len(moves) {
			return Decision{}, errors.New("requested a move after checkmate")
		}
		return Decision{Move: moves[len(turns)-1], Confidence: .7, CostUSD: .001}, nil
	}, func(move Ply) { observed = append(observed, move) })
	require.NoError(t, err)
	require.Equal(t, "0-1", match.Result)
	require.Equal(t, "Checkmate", match.Reason)
	require.Len(t, match.Moves, 4)
	require.Equal(t, match.Moves, observed)
	require.Equal(t, "Qh4#", match.Moves[3].SAN)
	require.InDelta(t, .004, match.CostUSD, 1e-9)
	require.Contains(t, match.PGN, "1. f3 e5 2. g4 Qh4# 0-1")
	for i, turn := range turns {
		if i%2 == 0 {
			require.Equal(t, Clef, turn.Model)
			require.Equal(t, "white", turn.Side)
		} else {
			require.Equal(t, Jev, turn.Model)
			require.Equal(t, "black", turn.Side)
		}
	}
	require.NotEqual(t, turns[0].FEN, turns[1].FEN)
}

func TestRulesAndLimits(t *testing.T) {
	cases := []struct {
		name, fen, result, reason, lastSAN string
		moves                              []string
		limit                              int
	}{
		{name: "limit is not a draw", moves: []string{"e2e4", "e7e5"}, limit: 2, result: "*", reason: "ply limit", lastSAN: "e5"},
		{name: "castling", fen: "r3k2r/8/8/8/8/8/8/R3K2R w KQkq - 0 1", moves: []string{"e1g1"}, limit: 1, result: "*", reason: "ply limit", lastSAN: "O-O"},
		{name: "promotion mate takes precedence over limit", fen: "7k/P7/6K1/8/8/8/8/8 w - - 0 1", moves: []string{"a7a8q"}, limit: 1, result: "1-0", reason: "Checkmate", lastSAN: "a8=Q#"},
		{name: "en passant", moves: []string{"e2e4", "a7a6", "e4e5", "d7d5", "e5d6"}, limit: 5, result: "*", reason: "ply limit", lastSAN: "exd6"},
		{name: "black to move", fen: "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1", moves: []string{"e7e5"}, limit: 1, result: "*", reason: "ply limit", lastSAN: "e5"},
		{name: "stalemate", fen: "7k/5Q2/6K1/8/8/8/8/8 b - - 0 1", result: "1/2-1/2", reason: "Stalemate"},
		{name: "checkmate", fen: "7k/6Q1/6K1/8/8/8/8/8 b - - 0 1", result: "1-0", reason: "Checkmate"},
		{name: "insufficient material", fen: "7k/8/6K1/8/8/8/8/8 w - - 0 1", result: "1/2-1/2", reason: "InsufficientMaterial"},
		{name: "repetition keeps history", moves: strings.Fields(strings.Repeat("g1f3 g8f6 f3g1 f6g8 ", 4)), limit: 16, result: "1/2-1/2", reason: "FivefoldRepetition", lastSAN: "Ng8"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			calls := 0
			match, err := Play(ctx, Config{FEN: tc.fen, MaxPlies: tc.limit}, func(context.Context, Turn) (Decision, error) {
				if calls >= len(tc.moves) {
					return Decision{}, errors.New("unexpected extra model call")
				}
				move := tc.moves[calls]
				calls++
				return Decision{Move: move}, nil
			}, nil)
			require.NoError(t, err)
			require.Equal(t, tc.result, match.Result)
			require.Equal(t, tc.reason, match.Reason)
			require.Len(t, match.Moves, len(tc.moves))
			if tc.lastSAN != "" {
				require.Equal(t, tc.lastSAN, match.Moves[len(match.Moves)-1].SAN)
			}
			pgn, err := chess.PGN(strings.NewReader(match.PGN))
			require.NoError(t, err, "exported PGN must be replayable")
			replayed := chess.NewGame(pgn)
			require.Equal(t, match.FEN, replayed.FEN())
			// v2.6.0's PGN lexer misreads 1/2-1/2; check the exported result directly.
			fields := strings.Fields(match.PGN)
			require.Equal(t, match.Result, fields[len(fields)-1])
			require.Equal(t, match.Result, replayed.GetTagPair("Result"))
		})
	}
}

func TestInvalidDecisionAndProviderFailure(t *testing.T) {
	for _, fail := range []bool{false, true} {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		match, err := Play(ctx, Config{}, func(context.Context, Turn) (Decision, error) {
			if fail {
				return Decision{}, errors.New("provider unavailable")
			}
			return Decision{Move: "e2e5"}, nil
		}, nil)
		cancel()
		require.Error(t, err)
		require.Equal(t, "error", match.Reason)
		require.Equal(t, "*", match.Result)
		require.Empty(t, match.Moves)
		require.Equal(t, match.InitialFEN, match.FEN)
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
	machine, err := NewMachine(Config{MaxPlies: 1}, func(context.Context, Turn) (Decision, error) {
		<-proceed
		return Decision{Move: "e2e4"}, nil
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
	require.Equal(t, before.InitialFEN, before.FEN)
	require.Len(t, before.LegalMoves, 20)
	require.Len(t, actor.GetSnapshot().Context.Moves, 1)
}

func TestInvalidConfiguration(t *testing.T) {
	choose := func(context.Context, Turn) (Decision, error) { return Decision{}, nil }
	_, err := NewMachine(Config{FEN: "not a FEN"}, choose)
	require.Error(t, err)
	_, err = NewMachine(Config{MaxPlies: -1}, choose)
	require.Error(t, err)
	_, err = NewMachine(Config{}, nil)
	require.Error(t, err)
}
