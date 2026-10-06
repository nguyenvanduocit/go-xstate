package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	chess "github.com/corentings/chess/v2"
	"github.com/stretchr/testify/require"
)

func TestRejectsExistingPGNBeforeStartingMatch(t *testing.T) {
	t.Setenv("OPENROUTER_KEY", "unused-test-key")
	path := filepath.Join(t.TempDir(), "game.pgn")
	require.NoError(t, os.WriteFile(path, []byte("keep this game"), 0o644))
	// An invalid FEN would fail inside Play if the output check ran too late.
	err := run([]string{"-pgn", path, "-fen", "invalid"})
	require.ErrorContains(t, err, "save PGN")
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "keep this game", string(data))
}

func TestExportsTerminalFENWithoutNetwork(t *testing.T) {
	t.Setenv("OPENROUTER_KEY", "")
	t.Setenv("OPENROUTER_API_KEY", "unused-test-key")
	path := filepath.Join(t.TempDir(), "game.pgn")
	fen := "7k/6Q1/6K1/8/8/8/8/8 b - - 0 1"
	require.NoError(t, run([]string{"-fen", fen, "-pgn", path}))
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	option, err := chess.PGN(strings.NewReader(string(data)))
	require.NoError(t, err)
	game := chess.NewGame(option)
	require.Equal(t, fen, game.FEN())
	require.Equal(t, chess.WhiteWon, game.Outcome())
}

func TestRejectsInvalidFlagsAndMissingKey(t *testing.T) {
	t.Setenv("OPENROUTER_KEY", "")
	t.Setenv("OPENROUTER_API_KEY", "")
	for _, args := range [][]string{{"-max-plies", "0"}, {"-timeout", "0"}, {"-unknown"}, {"unexpected"}, {}} {
		require.Error(t, run(args))
	}
	require.NoError(t, run([]string{"-help"}))
}

func TestInvalidFENRemovesEmptyPGNReservation(t *testing.T) {
	t.Setenv("OPENROUTER_KEY", "unused-test-key")
	path := filepath.Join(t.TempDir(), "game.pgn")
	require.Error(t, run([]string{"-fen", "invalid", "-pgn", path}))
	_, err := os.Stat(path)
	require.True(t, os.IsNotExist(err))
}
