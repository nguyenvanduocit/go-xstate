package donut_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	donut "github.com/nguyenvanduocit/go-xstate/examples/donut"
	"github.com/nguyenvanduocit/go-xstate/examples/internal/tracetest"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// TestTrace replays testdata/persisted-donut-maker.golden.json, recorded from the JS
// example by scripts/trace/persisted-donut-maker/persisted-donut-maker.ts.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/persisted-donut-maker/donutMachine.ts#L3
// JS trace: scripts/trace/persisted-donut-maker/persisted-donut-maker.ts.
func TestTrace(t *testing.T) {
	tracetest.Run(t, "testdata/persisted-donut-maker.golden.json", func(clock xs.Clock, input any) *xs.Actor[*xs.MachineSnapshot[donut.Context]] {
		return xs.CreateActor(donut.Machine())
	})
}

type session struct {
	Input     []string `json:"input"`
	Stdout    string   `json:"stdout"`
	Persisted any      `json:"persisted"`
}

func loadSessions(t *testing.T) []session {
	t.Helper()
	raw, err := os.ReadFile("testdata/sessions.golden.json")
	require.NoError(t, err)
	var g struct {
		Sessions []session `json:"sessions"`
	}
	require.NoError(t, json.Unmarshal(raw, &g))
	require.NotEmpty(t, g.Sessions)
	return g.Sessions
}

// withoutNilOutputAndError mirrors JSON.stringify dropping `undefined`: the JS persisted
// snapshot of an active machine has no `output` / `error` key, the Go one carries them as
// null (docs/porting/notes/examples-lib-findings.md, persisted-donut-maker).
func withoutNilOutputAndError(t *testing.T, v any) any {
	t.Helper()
	m, ok := v.(map[string]any)
	require.True(t, ok, "persisted snapshot is not an object: %v", v)
	out := map[string]any{}
	for k, val := range m {
		if (k == "output" || k == "error") && val == nil {
			continue
		}
		out[k] = val
	}
	return out
}

// TestSessions runs the console entry once per recorded JS session. Each session starts from
// the persisted-state.json the JS run left behind after the previous session (none for the
// first), so the restore of a JS-written snapshot is exercised in isolation. The printed text
// and the persisted file written by Go are compared with the JS run
// (scripts/trace/persisted-donut-maker/sessions.ts).
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/persisted-donut-maker/main.ts#L15
// JS trace: scripts/trace/persisted-donut-maker/sessions.ts.
func TestSessions(t *testing.T) {
	var previous any
	for i, s := range loadSessions(t) {
		stateFile := filepath.Join(t.TempDir(), "persisted-state.json")
		if previous != nil {
			seed, err := json.Marshal(previous)
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(stateFile, seed, 0o644))
		}
		var out bytes.Buffer
		require.NoError(t, donut.Run(&out, strings.NewReader(strings.Join(s.Input, "\n")+"\n"), stateFile), "session %d", i)
		require.Equal(t, s.Stdout, out.String(), "session %d stdout", i)

		raw, err := os.ReadFile(stateFile)
		require.NoError(t, err)
		var persisted any
		require.NoError(t, json.Unmarshal(raw, &persisted))
		require.Equal(t, s.Persisted, withoutNilOutputAndError(t, persisted), "session %d persisted-state.json", i)
		previous = s.Persisted
	}
}

// TestStdout chains the sessions through the Go-written persisted-state.json and compares the
// concatenated console output with testdata/persisted-donut-maker.stdout.txt,
// recorded by persisted-donut-maker.stdout.ts.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/persisted-donut-maker/main.ts#L15
// JS trace: scripts/trace/persisted-donut-maker/persisted-donut-maker.stdout.ts.
func TestStdout(t *testing.T) {
	want, err := os.ReadFile("testdata/persisted-donut-maker.stdout.txt")
	require.NoError(t, err)
	stateFile := filepath.Join(t.TempDir(), "persisted-state.json")
	parts := []string{}
	for i, s := range loadSessions(t) {
		var out bytes.Buffer
		require.NoError(t, donut.Run(&out, strings.NewReader(strings.Join(s.Input, "\n")+"\n"), stateFile), "session %d", i)
		parts = append(parts, out.String())
	}
	require.Equal(t, string(want), strings.Join(parts, "=== next session ===\n"))
}

// TestRunIgnoresUnparsableState: a persisted file that does not parse behaves like a missing one.
// Go regression; expected values are Go assertions, not a recorded JS test.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/persisted-donut-maker/main.ts#L15
// Related JS trace: scripts/trace/persisted-donut-maker/persisted-donut-maker.stdout.ts.
func TestRunIgnoresUnparsableState(t *testing.T) {
	stateFile := filepath.Join(t.TempDir(), "persisted-state.json")
	require.NoError(t, os.WriteFile(stateFile, []byte("{not json"), 0o644))
	var out bytes.Buffer
	require.NoError(t, donut.Run(&out, strings.NewReader(""), stateFile))
	require.True(t, strings.HasPrefix(out.String(), "No persisted state found.\nCurrent state: "), out.String())
}
