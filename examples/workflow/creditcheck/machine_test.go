package creditcheck_test

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nguyenvanduocit/go-xstate/examples/internal/tracetest"
	wf "github.com/nguyenvanduocit/go-xstate/examples/workflow/creditcheck"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// afterStub is the Go twin of after100 in scripts/trace/workflow-credit-check/lib/load.ts:
// it resolves with value after 100 ms, or is abandoned when the invoking state exits.
func afterStub[T any](value T) xs.ActorLogic {
	return xs.FromPromise(func(ctx context.Context, _ xs.PromiseArgs) (T, error) {
		select {
		case <-time.After(100 * time.Millisecond):
			return value, nil
		case <-ctx.Done():
			var zero T
			return zero, context.Cause(ctx)
		}
	})
}

func stubs(check xs.ActorLogic) wf.Actors {
	return wf.Actors{
		CallCreditCheckMicroservice: check,
		StartApplicationWorkflowID:  afterStub(wf.ApplicationOutput{Application: wf.Application{ID: "application123", Status: "Approved"}}),
		SendRejectionEmailFunction:  afterStub(wf.EmailOutput{Email: wf.Email{ID: "email123", Status: "Sent"}}),
	}
}

// hangingCheck never answers, like hangingCreditCheckStub in lib/load.ts.
func hangingCheck() xs.ActorLogic {
	return xs.FromPromise(func(ctx context.Context, _ xs.PromiseArgs) (wf.CreditCheck, error) {
		<-ctx.Done()
		return wf.CreditCheck{}, context.Cause(ctx)
	})
}

// replay runs a golden file recorded with a SimulatedClock and the golden's own input.
func replay(t *testing.T, file string, actors wf.Actors) {
	tracetest.Run(t, "testdata/"+file, func(clock xs.Clock, input any) *xs.Actor[*xs.MachineSnapshot[wf.Context]] {
		raw, err := json.Marshal(input)
		require.NoError(t, err)
		var in wf.Input
		require.NoError(t, json.Unmarshal(raw, &in))
		return xs.CreateActor(wf.NewMachine(actors), xs.WithInput(in), xs.WithClock(clock))
	})
}

// TestApprovedTrace replays testdata/approved.golden.json (approved.ts).
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-credit-check/main.ts#L13
// JS trace: scripts/trace/workflow-credit-check/approved.ts.
func TestApprovedTrace(t *testing.T) {
	replay(t, "approved.golden.json", stubs(afterStub(wf.CreditCheck{ID: "customer123", Score: 700, Decision: "Approved", Reason: "Good credit score"})))
}

// TestDeniedTrace replays testdata/denied.golden.json (denied.ts).
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-credit-check/main.ts#L13
// JS trace: scripts/trace/workflow-credit-check/denied.ts.
func TestDeniedTrace(t *testing.T) {
	replay(t, "denied.golden.json", stubs(afterStub(wf.CreditCheck{ID: "customer123", Score: 400, Decision: "Denied", Reason: "Low credit score"})))
}

// TestReviewTrace replays testdata/review.golden.json (review.ts): neither guard matches.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-credit-check/main.ts#L13
// JS trace: scripts/trace/workflow-credit-check/review.ts.
func TestReviewTrace(t *testing.T) {
	replay(t, "review.golden.json", stubs(afterStub(wf.CreditCheck{ID: "customer123", Score: 600, Decision: "Review", Reason: "Needs manual review"})))
}

// TestTimeoutTrace replays testdata/timeout.golden.json (timeout.ts).
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-credit-check/main.ts#L13
// JS trace: scripts/trace/workflow-credit-check/timeout.ts.
func TestTimeoutTrace(t *testing.T) {
	replay(t, "timeout.golden.json", stubs(hangingCheck()))
}

// TestStdout compares the printing entry with testdata/workflow-credit-check.stdout.txt,
// recorded from main.ts. The delay is shortened; the text does not depend on it.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-credit-check/main.ts#L137
// JS trace: scripts/trace/workflow-credit-check/workflow-credit-check.stdout.ts.
func TestStdout(t *testing.T) {
	want, err := os.ReadFile("testdata/workflow-credit-check.stdout.txt")
	require.NoError(t, err)
	var out strings.Builder
	wf.RunWith(&out, 10*time.Millisecond)
	assert.Equal(t, string(want), out.String())
}
