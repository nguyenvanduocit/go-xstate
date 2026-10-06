package fetch_test

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nguyenvanduocit/go-xstate/examples/fetch"
	"github.com/nguyenvanduocit/go-xstate/examples/internal/tracetest"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"os"
)

// TestTrace replays testdata/fetch.golden.json, recorded from the JS example by
// scripts/trace/fetch/fetch.ts. fetchUser is replaced by the same deterministic
// stub as in the JS script: attempts 1 and 2 reject, attempt 3 resolves.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/fetch/src/fetchMachine.ts#L4
// JS trace: scripts/trace/fetch/fetch.ts.
func TestTrace(t *testing.T) {
	tracetest.Run(t, "testdata/fetch.golden.json", func(clock xs.Clock, input any) *xs.Actor[*xs.MachineSnapshot[fetch.Context]] {
		var attempts atomic.Int32
		stub := xs.FromPromise(func(_ context.Context, a xs.PromiseArgs) (fetch.Greeting, error) {
			if attempts.Add(1) < 3 {
				return fetch.Greeting{}, errors.New("boom")
			}
			return fetch.Greeting{Greeting: "Hello, " + a.Input.(fetch.FetchInput).Name + "!"}, nil
		})
		machine := fetch.Machine().Provide(xs.Implementations{Actors: map[string]xs.ActorLogic{"fetchUser": stub}})
		return xs.CreateActor(machine, xs.WithClock(clock))
	})
}

// TestStdout compares the printing entry with testdata/fetch.stdout.txt, recorded from
// src/index.ts with Math.random pinned to 0.9 (getGreeting resolves).
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/fetch/src/index.ts#L18
// JS trace: scripts/trace/fetch/fetch.stdout.ts.
func TestStdout(t *testing.T) {
	want, err := os.ReadFile("testdata/fetch.stdout.txt")
	require.NoError(t, err)
	var out strings.Builder
	require.NoError(t, fetch.RunWith(&out, 10*time.Millisecond, func() float64 { return 0.9 }))
	assert.Equal(t, string(want), out.String())
}

// Go regression; expected values are Go assertions, not a recorded JS test.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/fetch/src/index.ts#L4
// Related JS trace: scripts/trace/fetch/fetch.ts.
func TestGetGreeting(t *testing.T) {
	g, err := fetch.GetGreeting(context.Background(), "Ada", time.Millisecond, func() float64 { return 0.5 })
	require.NoError(t, err)
	assert.Equal(t, fetch.Greeting{Greeting: "Hello, Ada!"}, g)

	_, err = fetch.GetGreeting(context.Background(), "Ada", time.Millisecond, func() float64 { return 0.49 })
	assert.ErrorIs(t, err, fetch.ErrRejected)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = fetch.GetGreeting(ctx, "Ada", time.Hour, func() float64 { return 0.9 })
	assert.ErrorIs(t, err, context.Canceled)
}
