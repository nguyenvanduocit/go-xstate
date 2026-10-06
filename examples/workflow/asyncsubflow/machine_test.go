package asyncsubflow_test

import (
	"bufio"
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nguyenvanduocit/go-xstate/examples/internal/tracetest"
	wf "github.com/nguyenvanduocit/go-xstate/examples/workflow/asyncsubflow"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// promptStub is the Go twin of promptStub in scripts/trace/workflow-async-subflow/lib/load.ts:
// it answers from a table after 50 ms and fails on a question that is not in the table.
func promptStub(answers map[string]string) xs.ActorLogic {
	return xs.FromPromise(func(ctx context.Context, a xs.PromiseArgs) (wf.PromptOutput, error) {
		select {
		case <-time.After(50 * time.Millisecond):
		case <-ctx.Done():
			return wf.PromptOutput{}, context.Cause(ctx)
		}
		q := a.Input.(wf.PromptInput).Question
		resp, ok := answers[q]
		if !ok {
			return wf.PromptOutput{}, errors.New("unexpected question: " + q)
		}
		return wf.PromptOutput{Response: resp}, nil
	})
}

func answersFor(name string) map[string]string {
	return map[string]string{
		"What is your name?": name,
		"Welcome " + name + ", press enter to finish the onboarding process": "",
	}
}

// TestWorkflowTrace replays testdata/workflow-async-subflow.golden.json, recorded from the JS
// example by scripts/trace/workflow-async-subflow/workflow-async-subflow.ts.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-async-subflow/main.ts#L60
// JS trace: scripts/trace/workflow-async-subflow/workflow-async-subflow.ts.
func TestWorkflowTrace(t *testing.T) {
	tracetest.Run(t, "testdata/workflow-async-subflow.golden.json", func(_ xs.Clock, _ any) *xs.Actor[*xs.MachineSnapshot[wf.WorkflowContext]] {
		return xs.CreateActor(wf.NewWorkflow(wf.NewOnboarding(promptStub(answersFor("Ada")))))
	})
}

// TestOnboardingTrace replays testdata/onboarding.golden.json (onboarding.ts).
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-async-subflow/main.ts#L15
// JS trace: scripts/trace/workflow-async-subflow/onboarding.ts.
func TestOnboardingTrace(t *testing.T) {
	tracetest.Run(t, "testdata/onboarding.golden.json", func(_ xs.Clock, _ any) *xs.Actor[*xs.MachineSnapshot[wf.OnboardingContext]] {
		return xs.CreateActor(wf.NewOnboarding(promptStub(answersFor("Ada"))))
	})
}

// TestOnboardingEmptyNameTrace replays testdata/onboarding-empty-name.golden.json
// (onboarding-empty-name.ts): an empty first answer is assigned as "", not left undefined.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-async-subflow/main.ts#L15
// JS trace: scripts/trace/workflow-async-subflow/onboarding-empty-name.ts.
func TestOnboardingEmptyNameTrace(t *testing.T) {
	tracetest.Run(t, "testdata/onboarding-empty-name.golden.json", func(_ xs.Clock, _ any) *xs.Actor[*xs.MachineSnapshot[wf.OnboardingContext]] {
		return xs.CreateActor(wf.NewOnboarding(promptStub(answersFor(""))))
	})
}

// TestStdout compares the printing entry with testdata/workflow-async-subflow.stdout.txt,
// recorded by workflow-async-subflow.stdout.ts, which runs the real main.ts and answers
// "Ada" and an empty line.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-async-subflow/main.ts#L80
// JS trace: scripts/trace/workflow-async-subflow/workflow-async-subflow.stdout.ts.
func TestStdout(t *testing.T) {
	want, err := os.ReadFile("testdata/workflow-async-subflow.stdout.txt")
	require.NoError(t, err)
	var out strings.Builder
	require.NoError(t, wf.RunWith(strings.NewReader("Ada\n\n"), &out))
	assert.Equal(t, string(want), out.String())
}

// Answers are read on demand, one line per question; CRLF endings are stripped like readline does
// and a last line without a newline still counts as an answer.
// Go regression; expected values are Go assertions, not a recorded JS test.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-async-subflow/main.ts#L80
// Related JS trace: scripts/trace/workflow-async-subflow/workflow-async-subflow.stdout.ts.
func TestRunWith_CRLFAndNoTrailingNewline(t *testing.T) {
	var out strings.Builder
	require.NoError(t, wf.RunWith(strings.NewReader("Grace\r\nignored"), &out))
	assert.Equal(t, "What is your name?Welcome Grace, press enter to finish the onboarding processworkflow completed undefined\n", out.String())
}

// With no input at all the first prompt fails and, as the machine has no onError, the workflow errors.
// Go regression; expected values are Go assertions, not a recorded JS test.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-async-subflow/main.ts#L80
// Related JS trace: scripts/trace/workflow-async-subflow/workflow-async-subflow.stdout.ts.
func TestRunWith_EndOfInput(t *testing.T) {
	var out strings.Builder
	err := wf.RunWith(strings.NewReader(""), &out)
	require.Error(t, err)
	assert.Contains(t, err.Error(), io.EOF.Error())
	assert.Equal(t, "What is your name?", out.String())
}

type syncBuffer struct {
	mu sync.Mutex
	sb strings.Builder
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.sb.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.sb.String()
}

type promptReadObserver struct {
	io.Reader
	started chan struct{}
	once    sync.Once
}

func (r *promptReadObserver) Read(p []byte) (int, error) {
	r.once.Do(func() { close(r.started) })
	return r.Reader.Read(p)
}

// Stopping the actor while the prompt waits for input abandons it: the machine never advances.
// Go regression; expected values are Go assertions, not a recorded JS test.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-async-subflow/main.ts#L12
// Related JS trace: scripts/trace/workflow-async-subflow/onboarding-empty-name.ts.
func TestPrompt_StoppedActorAbandonsPrompt(t *testing.T) {
	pr, pw := io.Pipe()
	defer pr.Close()
	defer pw.Close()
	input := &promptReadObserver{Reader: pr, started: make(chan struct{})}
	var out syncBuffer
	actor := xs.CreateActor(wf.NewOnboarding(wf.Prompt(bufio.NewReader(input), &out)))
	defer actor.Stop()
	actor.Start()
	select {
	case <-input.started:
	case <-time.After(time.Second):
		t.Fatal("prompt did not begin waiting for input")
	}
	actor.Stop()
	assert.Equal(t, "What is your name?", out.String())
	assert.Equal(t, "Welcome", actor.GetSnapshot().Value)
}
