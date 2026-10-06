package booklending_test

import (
	"bytes"
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nguyenvanduocit/go-xstate/examples/internal/tracetest"
	wbl "github.com/nguyenvanduocit/go-xstate/examples/workflow/booklending"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// settle resolves like the JS stubs `fromPromise(async () => { await sleep(80) })`: after 80 ms,
// so a snapshot taken right after `send` still shows the invoking state.
var settle = xs.FromPromise(func(context.Context, xs.PromiseArgs) (any, error) {
	time.Sleep(80 * time.Millisecond)
	return nil, nil
})

// statusSequence mirrors the JS 'Get status for book' stub: the n-th invocation answers statuses[n].
func statusSequence(statuses ...string) xs.ActorLogic {
	var mu sync.Mutex
	n := 0
	return xs.FromPromise(func(context.Context, xs.PromiseArgs) (wbl.StatusOutput, error) {
		time.Sleep(80 * time.Millisecond)
		mu.Lock()
		defer mu.Unlock()
		out := wbl.StatusOutput{Status: statuses[n]}
		n++
		return out, nil
	})
}

// replay replays testdata/<name>.golden.json (recorded by scripts/trace/workflow-book-lending/<name>.ts)
// with the same deterministic stubs as the JS script.
func replay(t *testing.T, name string, impl xs.Implementations, statuses ...string) {
	t.Helper()
	impl.Actors = map[string]xs.ActorLogic{
		"Get status for book":            statusSequence(statuses...),
		"Send status to lender":          settle,
		"Request hold for lender":        settle,
		"Cancel hold request for lender": settle,
		"Check out book with id":         settle,
		"Notify Lender for checkout":     settle,
	}
	tracetest.Run(t, "testdata/"+name+".golden.json", func(clock xs.Clock, _ any) *xs.Actor[*xs.MachineSnapshot[wbl.Context]] {
		machine := wbl.NewMachine(func(string, any) {}, 0).Provide(impl)
		if clock == nil {
			return xs.CreateActor(machine)
		}
		return xs.CreateActor(machine, xs.WithClock(clock))
	})
}

// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-book-lending/main.ts#L22
// JS trace: scripts/trace/workflow-book-lending/onloan-hold-clock.ts.
func TestTraceOnloanHoldClock(t *testing.T) {
	replay(t, "onloan-hold-clock", xs.Implementations{Delays: map[string]any{"PT2W": 1000 * time.Millisecond}}, "onloan", "available")
}

// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-book-lending/main.ts#L22
// JS trace: scripts/trace/workflow-book-lending/decline.ts.
func TestTraceDecline(t *testing.T) { replay(t, "decline", xs.Implementations{}, "onloan") }

// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-book-lending/main.ts#L22
// JS trace: scripts/trace/workflow-book-lending/unknown-status.ts.
func TestTraceUnknownStatus(t *testing.T) { replay(t, "unknown-status", xs.Implementations{}, "lost") }

// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-book-lending/main.ts#L22
// JS trace: scripts/trace/workflow-book-lending/sleep-unresolved-delay.ts.
func TestTraceSleepUnresolvedDelay(t *testing.T) {
	replay(t, "sleep-unresolved-delay", xs.Implementations{}, "onloan", "available")
}

// TestStdout compares Run with testdata/workflow-book-lending.stdout.txt, recorded from main.ts.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-book-lending/main.ts#L258
// JS trace: scripts/trace/workflow-book-lending/workflow-book-lending.stdout.ts.
func TestStdout(t *testing.T) {
	want, err := os.ReadFile("testdata/workflow-book-lending.stdout.txt")
	require.NoError(t, err)
	var out bytes.Buffer
	require.NoError(t, wbl.RunWith(context.Background(), &out, 10*time.Millisecond))
	assert.Equal(t, string(want), out.String())
}

// Go regression; expected values are Go assertions, not a recorded JS test.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-book-lending/main.ts#L258
// Related JS trace: scripts/trace/workflow-book-lending/workflow-book-lending.stdout.ts.
func TestRunContextCancel(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	err := wbl.RunWith(ctx, &bytes.Buffer{}, time.Minute)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}
