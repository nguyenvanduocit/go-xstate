// Package asyncsubflow ports references/xstate/examples/workflow-async-subflow/main.ts
// (serverless workflow "async subflow invocation"): a workflow whose `Onboard`
// state invokes an `onboarding` machine, which asks two questions on the console.
package asyncsubflow

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// PromptInput is the input of the prompt actor: `{ question: string }`.
type PromptInput struct {
	Question string `json:"question"`
}

// PromptOutput is the resolved value of the prompt actor: `{ response: string }`.
type PromptOutput struct {
	Response string `json:"response"`
}

// OnboardingContext is the onboarding machine's extended state. `name` is
// `undefined` until the first answer is assigned, so Name is a pointer that is
// omitted from JSON while nil.
type OnboardingContext struct {
	Name *string `json:"name,omitempty"`
}

// WorkflowContext is the workflow's extended state: the JS machine has no
// context, which is `{}` in a snapshot.
type WorkflowContext struct{}

// Prompt mirrors the `prompt` actor of main.ts: it writes the question to w
// (readline prints it without a newline), then resolves with the next line read
// from r, without its line terminator. Lines are read on demand, one per
// question. Cancelling ctx stands in for abandoning the promise when the
// invoking state exits.
func Prompt(r *bufio.Reader, w io.Writer) xs.ActorLogic {
	return xs.FromPromise(func(ctx context.Context, a xs.PromiseArgs) (PromptOutput, error) {
		fmt.Fprint(w, a.Input.(PromptInput).Question)
		type result struct {
			line string
			err  error
		}
		ch := make(chan result, 1)
		go func() {
			line, err := r.ReadString('\n')
			if err == io.EOF && line != "" {
				err = nil
			}
			ch <- result{strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r"), err}
		}()
		select {
		case <-ctx.Done():
			return PromptOutput{}, context.Cause(ctx)
		case res := <-ch:
			return PromptOutput{Response: res.line}, res.err
		}
	})
}

// NewOnboarding mirrors `onboardingWorkflow` with the prompt actor given.
func NewOnboarding(prompt xs.ActorLogic) *xs.StateMachine[OnboardingContext] {
	return xs.NewSetup[OnboardingContext](xs.Implementations{
		Actors: map[string]xs.ActorLogic{"prompt": prompt},
	}).CreateMachine(xs.MachineConfig[OnboardingContext]{
		ID:      "onboarding",
		Initial: "Welcome",
		Context: OnboardingContext{},
		States: xs.States{
			{
				Key: "Welcome",
				Invoke: []xs.InvokeConfig{{
					Src:   "prompt",
					Input: PromptInput{Question: "What is your name?"},
					OnDone: xs.Transitions{{
						Target: "Personalize",
						Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[OnboardingContext]) OnboardingContext {
							response := a.Event.(xs.DoneActorEvent).Output.(PromptOutput).Response
							return OnboardingContext{Name: &response}
						})},
					}},
				}},
			},
			{
				Key: "Personalize",
				Invoke: []xs.InvokeConfig{{
					Src: "prompt",
					Input: xs.NewExpr(func(a xs.ExprArgs[OnboardingContext]) any {
						return PromptInput{Question: fmt.Sprintf("Welcome %s, press enter to finish the onboarding process", jsString(a.Context.Name))}
					}),
					OnDone: xs.Transitions{{Target: "Completed"}},
				}},
			},
			{Key: "Completed", Type: xs.Final},
		},
	})
}

// NewWorkflow mirrors `workflow` with the onboarding actor given.
func NewWorkflow(onboarding xs.ActorLogic) *xs.StateMachine[WorkflowContext] {
	return xs.NewSetup[WorkflowContext](xs.Implementations{
		Actors: map[string]xs.ActorLogic{"onboarding": onboarding},
	}).CreateMachine(xs.MachineConfig[WorkflowContext]{
		ID:      "async-function-invocation",
		Initial: "Onboard",
		States: xs.States{
			{
				Key: "Onboard",
				Invoke: []xs.InvokeConfig{{
					Src:    "onboarding",
					OnDone: xs.Transitions{{Target: "Onboarded"}},
				}},
			},
			{Key: "Onboarded", Type: xs.Final},
		},
	})
}

// Run mirrors the entry of main.ts: it reads the answers from os.Stdin
// and prints the questions and the completion line to w.
func Run(w io.Writer) error {
	return RunWith(os.Stdin, w)
}

// RunWith is Run with the answer source injected.
func RunWith(in io.Reader, w io.Writer) error {
	var mu sync.Mutex
	locked := &lockedWriter{mu: &mu, w: w}
	done := make(chan error, 1)
	actor := xs.CreateActor(NewWorkflow(NewOnboarding(Prompt(bufio.NewReader(in), locked))))
	actor.Subscribe(xs.Observer[*xs.MachineSnapshot[WorkflowContext]]{
		Error: func(err any) { done <- fmt.Errorf("workflow failed: %v", err) },
		Complete: func() {
			fmt.Fprintln(locked, "workflow completed", formatOutput(actor.GetSnapshot().Output))
			done <- nil
		},
	})
	actor.Start()
	err := <-done
	actor.Stop()
	return err
}

// jsString renders an optional string the way a JS template literal does:
// `undefined` when unset.
func jsString(s *string) string {
	if s == nil {
		return "undefined"
	}
	return *s
}

// formatOutput renders the machine output the way console.log prints it in the
// JS runtime: this machine has no output, so `undefined`.
func formatOutput(out any) string {
	if out == nil {
		return "undefined"
	}
	return fmt.Sprint(out)
}

type lockedWriter struct {
	mu *sync.Mutex
	w  io.Writer
}

func (l *lockedWriter) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.w.Write(p)
}
