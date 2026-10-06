package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDemoSavesSGFWithoutCredentials(t *testing.T) {
	t.Setenv("OPENROUTER_KEY", "")
	t.Setenv("OPENROUTER_API_KEY", "")
	path := filepath.Join(t.TempDir(), "demo.sgf")
	require.NoError(t, run([]string{"-demo", "-sgf", path}))
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Contains(t, string(raw), "SZ[9]KM[7.5]RU[Tromp-Taylor]")
	require.Contains(t, string(raw), "PB[scripted-demo-black]")
	require.Contains(t, string(raw), "RE[")
	require.Contains(t, string(raw), ";B[];W[])")
	require.NotContains(t, string(raw), "cloudflare/clef")
}

func TestExistingSGFIsPreserved(t *testing.T) {
	path := filepath.Join(t.TempDir(), "existing.sgf")
	require.NoError(t, os.WriteFile(path, []byte("preserve me"), 0o600))
	require.ErrorContains(t, run([]string{"-demo", "-sgf", path}), "save SGF")
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "preserve me", string(raw))
}

func TestMoveLimitSavesUnscoredGame(t *testing.T) {
	path := filepath.Join(t.TempDir(), "partial.sgf")
	require.NoError(t, run([]string{"-demo", "-max-moves", "1", "-sgf", path}))
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	require.NotContains(t, string(raw), "RE[")
}

func TestHelpAndInvalidArguments(t *testing.T) {
	t.Setenv("OPENROUTER_KEY", "")
	t.Setenv("OPENROUTER_API_KEY", "")
	require.NoError(t, run([]string{"-help"}))
	require.ErrorContains(t, run(nil), "OPENROUTER_KEY")
	for _, args := range [][]string{
		{"-demo", "-size", "0"}, {"-demo", "-size", "20"},
		{"-demo", "-komi", "NaN"}, {"-demo", "-max-moves", "0"},
		{"-demo", "-timeout", "0s"}, {"-demo", "-request-timeout", "0s"},
		{"unexpected"},
	} {
		require.Error(t, run(args), args)
	}
}
