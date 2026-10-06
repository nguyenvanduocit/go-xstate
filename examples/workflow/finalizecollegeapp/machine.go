// Package finalizecollegeapp ports references/xstate/examples/workflow-finalize-college-app/main.ts
// (the Serverless Workflow "finalize college application" example).
package finalizecollegeapp

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// Input is the machine input: `{ applicantId }`. It is also the input of
// finalizeApplicationFunction.
type Input struct {
	ApplicantID string `json:"applicantId"`
}

// Context is the machine's extended state.
type Context struct {
	ApplicantID                  string `json:"applicantId"`
	ApplicationSubmitted         bool   `json:"applicationSubmitted"`
	SATScoresReceived            bool   `json:"satScoresReceived"`
	RecommendationLetterReceived bool   `json:"recommendationLetterReceived"`
}

// FinalizeOutput is the output of finalizeApplicationFunction: `{ applicantId }`.
type FinalizeOutput struct {
	ApplicantID string `json:"applicantId"`
}

// FinalizeApplicationFunction mirrors the actor of the same name: it logs to w,
// waits delay (1000 ms in the JS), logs again and resolves with the applicant
// id. Cancelling ctx stands in for abandoning the promise when the invoking
// state exits.
func FinalizeApplicationFunction(w io.Writer, delay time.Duration) *xs.PromiseLogic[FinalizeOutput] {
	return finalizeApplicationFunction(w, delay, nil)
}

// finalizeApplicationFunction is FinalizeApplicationFunction that also closes
// started (when non-nil) right after the first log line, see RunWith.
func finalizeApplicationFunction(w io.Writer, delay time.Duration, started chan<- struct{}) *xs.PromiseLogic[FinalizeOutput] {
	return xs.FromPromise(func(ctx context.Context, a xs.PromiseArgs) (FinalizeOutput, error) {
		in := a.Input.(Input)
		fmt.Fprintf(w, "Starting to finalize application for %s\n", in.ApplicantID)
		if started != nil {
			close(started)
		}
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return FinalizeOutput{}, context.Cause(ctx)
		case <-timer.C:
		}
		fmt.Fprintf(w, "Finalized application for %s\n", in.ApplicantID)
		return FinalizeOutput{ApplicantID: in.ApplicantID}, nil
	})
}

func setFlag(set func(*Context)) xs.Action {
	return xs.Assign(func(a xs.AssignArgs[Context]) Context {
		c := a.Context
		set(&c)
		return c
	})
}

// NewMachine mirrors `workflow` with the finalizeApplicationFunction actor given.
func NewMachine(finalize xs.ActorLogic) *xs.StateMachine[Context] {
	return xs.NewSetup[Context](xs.Implementations{
		Actors: map[string]xs.ActorLogic{"finalizeApplicationFunction": finalize},
	}).CreateMachine(xs.MachineConfig[Context]{
		ID:      "finalizeCollegeApplication",
		Initial: "FinalizeApplication",
		ContextFn: func(a xs.ContextArgs) Context {
			return Context{ApplicantID: a.Input.(Input).ApplicantID}
		},
		States: xs.States{
			{
				Key: "FinalizeApplication",
				On: map[string]xs.Transitions{
					"ApplicationSubmitted": {{Actions: xs.Actions{setFlag(func(c *Context) { c.ApplicationSubmitted = true })}}},
					"SATScoresReceived":    {{Actions: xs.Actions{setFlag(func(c *Context) { c.SATScoresReceived = true })}}},
					"RecommendationLetterReceived": {{Actions: xs.Actions{
						setFlag(func(c *Context) { c.RecommendationLetterReceived = true }),
					}}},
				},
				Always: xs.Transitions{{
					Guard: xs.GuardFunc(func(a xs.GuardArgs[Context]) bool {
						return a.Context.ApplicationSubmitted && a.Context.SATScoresReceived && a.Context.RecommendationLetterReceived
					}),
					Target: "FinalizingApplication",
				}},
			},
			{
				Key: "FinalizingApplication",
				Invoke: []xs.InvokeConfig{{
					Src: "finalizeApplicationFunction",
					Input: xs.NewExpr(func(a xs.ExprArgs[Context]) any {
						return Input{ApplicantID: a.Context.ApplicantID}
					}),
					OnDone: xs.Transitions{{Target: "Finalized"}},
				}},
			},
			{Key: "Finalized", Type: xs.Final},
		},
	})
}

// Run mirrors the entry of main.ts: applicant "123", the real 1000 ms
// finalizeApplicationFunction, one event per second, printing to w.
func Run(w io.Writer) {
	RunWith(w, time.Second)
}

// RunWith is Run with the delay injected: it is both the pause before each of
// the three events and the duration of finalizeApplicationFunction (1000 ms in
// the JS).
func RunWith(w io.Writer, delay time.Duration) {
	out := &lockedWriter{w: w}
	// JS runs a promise executor synchronously until its first await, so
	// "Starting to finalize ..." is printed before the subscriber sees
	// FinalizingApplication. The Go promise runs on its own goroutine; the
	// subscriber waits for the first log line to keep the same order.
	started := make(chan struct{})
	done := make(chan struct{})
	actor := xs.CreateActor(
		NewMachine(finalizeApplicationFunction(out, delay, started)),
		xs.WithInput(Input{ApplicantID: "123"}),
	)
	actor.Subscribe(xs.Observer[*xs.MachineSnapshot[Context]]{
		Next: func(s *xs.MachineSnapshot[Context]) {
			if s.Value == "FinalizingApplication" {
				<-started
			}
			fmt.Fprintln(out, s.Value)
		},
		Complete: func() {
			fmt.Fprintln(out, "workflow completed", formatOutput(actor.GetSnapshot().Output))
			close(done)
		},
	})
	actor.Start()
	for _, e := range []string{"ApplicationSubmitted", "SATScoresReceived", "RecommendationLetterReceived"} {
		time.Sleep(delay)
		actor.Send(xs.Ev(e))
	}
	<-done
}

// formatOutput renders the machine output the way console.log prints it in the
// JS runtime: this machine has no output, so `undefined`.
func formatOutput(out any) string {
	if out == nil {
		return "undefined"
	}
	return fmt.Sprint(out)
}

// lockedWriter serialises writes from the promise goroutine and the subscriber.
type lockedWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func (l *lockedWriter) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.w.Write(p)
}
