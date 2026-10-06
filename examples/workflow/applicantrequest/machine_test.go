package applicantrequest_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nguyenvanduocit/go-xstate/examples/internal/tracetest"
	war "github.com/nguyenvanduocit/go-xstate/examples/workflow/applicantrequest"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// instant resolves like `fromPromise(async () => { await sleep(50) })` in the JS scripts.
// Both sides settle after 50 ms (JS: setTimeout) so a snapshot taken right after `send` still shows
// the invoking state; the `wait: 75` steps observe the result.
var instant = xs.FromPromise(func(context.Context, xs.PromiseArgs) (any, error) {
	time.Sleep(50 * time.Millisecond)
	return nil, nil
})

// failing rejects like `fromPromise(async () => { await sleep(50); throw new Error('boom') })`.
var failing = xs.FromPromise(func(context.Context, xs.PromiseArgs) (any, error) {
	time.Sleep(50 * time.Millisecond)
	return nil, errors.New("boom")
})

// replayTrace replays testdata/<name>.golden.json, recorded by scripts/trace/workflow-applicant-request/<name>.ts.
// The golden file carries the machine input; the two promise actors are replaced by the
// same deterministic stubs as in the JS script.
func replayTrace(t *testing.T, name string, start xs.ActorLogic) {
	tracetest.Run(t, "testdata/"+name+".golden.json", func(_ xs.Clock, input any) *xs.Actor[*xs.MachineSnapshot[war.Context]] {
		raw, err := json.Marshal(input)
		require.NoError(t, err)
		var in war.Input
		require.NoError(t, json.Unmarshal(raw, &in))
		machine := war.Machine().Provide(xs.Implementations{Actors: map[string]xs.ActorLogic{
			"startApplicationWorkflowId": start,
			"sendRejectionEmailFunction": instant,
		}})
		return xs.CreateActor(machine, xs.WithInput(in))
	})
}

// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-applicant-request/main.ts#L11
// JS trace: scripts/trace/workflow-applicant-request/adult.ts.
func TestTraceAdult(t *testing.T) { replayTrace(t, "adult", instant) }

// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-applicant-request/main.ts#L11
// JS trace: scripts/trace/workflow-applicant-request/minor.ts.
func TestTraceMinor(t *testing.T) { replayTrace(t, "minor", instant) }

// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-applicant-request/main.ts#L11
// JS trace: scripts/trace/workflow-applicant-request/start-error.ts.
func TestTraceStartError(t *testing.T) { replayTrace(t, "start-error", failing) }

// TestStdout compares the printing entry with testdata/workflow-applicant-request.stdout.txt,
// recorded from main.ts fed the stdin line `Submit`.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-applicant-request/main.ts#L82
// JS trace: scripts/trace/workflow-applicant-request/workflow-applicant-request.stdout.ts.
func TestStdout(t *testing.T) {
	want, err := os.ReadFile("testdata/workflow-applicant-request.stdout.txt")
	require.NoError(t, err)
	var out bytes.Buffer
	require.NoError(t, war.RunWith(context.Background(), &out, strings.NewReader("Submit\n"), war.DefaultApplicant, 10*time.Millisecond))
	assert.Equal(t, string(want), out.String())
}

// The JS entry hardcodes an adult applicant; the minor and failing paths are driven here without a recording.
// Go regression; expected values are Go assertions, not a recorded JS test.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-applicant-request/main.ts#L82
// Related JS trace: scripts/trace/workflow-applicant-request/workflow-applicant-request.stdout.ts.
func TestRunMinor(t *testing.T) {
	var out bytes.Buffer
	minor := war.Applicant{Fname: "Ann", Lname: "Lee", Age: 17, Email: "a@b.c"}
	require.NoError(t, war.RunWith(context.Background(), &out, strings.NewReader("Submit\n"), minor, 5*time.Millisecond))
	assert.Equal(t, "sendRejectionEmailFunction workflow started\nsendRejectionEmailFunction workflow completed\nworkflow completed undefined\n", out.String())
}

// stdin ends in CheckApplication with nothing pending: Run returns, as Node would exit.
// Go regression; expected values are Go assertions, not a recorded JS test.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-applicant-request/main.ts#L82
// Related JS trace: scripts/trace/workflow-applicant-request/workflow-applicant-request.stdout.ts.
func TestRunStdinEndsIdle(t *testing.T) {
	var out bytes.Buffer
	require.NoError(t, war.RunWith(context.Background(), &out, strings.NewReader("Other\n"), war.DefaultApplicant, time.Millisecond))
	assert.Empty(t, out.String())
}

// Go regression; expected values are Go assertions, not a recorded JS test.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-applicant-request/main.ts#L82
// Related JS trace: scripts/trace/workflow-applicant-request/workflow-applicant-request.stdout.ts.
func TestRunContextCancel(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	pr, pw := io.Pipe()
	defer pw.Close()
	err := war.RunWith(ctx, io.Discard, pr, war.DefaultApplicant, time.Millisecond)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}
