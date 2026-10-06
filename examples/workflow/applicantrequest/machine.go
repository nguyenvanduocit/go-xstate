// Package applicantrequest ports references/xstate/examples/workflow-applicant-request/main.ts
// (the Serverless Workflow "applicant request decision" example).
package applicantrequest

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// Applicant mirrors the JS `Applicant` interface.
type Applicant struct {
	Fname string `json:"fname"`
	Lname string `json:"lname"`
	Age   int    `json:"age"`
	Email string `json:"email"`
}

// Input is the machine input: `{ applicant }`.
type Input struct {
	Applicant Applicant `json:"applicant"`
}

// Context is the machine's extended state: `{ applicant }`.
type Context struct {
	Applicant Applicant `json:"applicant"`
}

// DefaultApplicant is the applicant hardcoded in the JS entry's createActor call.
var DefaultApplicant = Applicant{Fname: "John", Lname: "Stockton", Age: 22, Email: "js@something.com"}

// serviceCall mirrors the two fromPromise actors: print "<name> workflow started",
// wait delay (1000 ms in JS), print "<name> workflow completed". Cancelling ctx
// stands in for abandoning the promise when the invoking state exits.
func serviceCall(w io.Writer, name string, delay time.Duration) xs.ActorLogic {
	return xs.FromPromise(func(ctx context.Context, _ xs.PromiseArgs) (any, error) {
		fmt.Fprintf(w, "%s workflow started\n", name)
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return nil, context.Cause(ctx)
		case <-timer.C:
		}
		fmt.Fprintf(w, "%s workflow completed\n", name)
		return nil, nil
	})
}

// NewMachine mirrors `workflow`, with the actors' console.log output going to w
// and their delay (1000 ms in JS) injected.
func NewMachine(w io.Writer, delay time.Duration) *xs.StateMachine[Context] {
	return xs.NewSetup[Context](xs.Implementations{
		Actors: map[string]xs.ActorLogic{
			"startApplicationWorkflowId": serviceCall(w, "startApplicationWorkflowId", delay),
			"sendRejectionEmailFunction": serviceCall(w, "sendRejectionEmailFunction", delay),
		},
		Guards: map[string]xs.Guard{
			"isOver18": xs.GuardFunc(func(a xs.GuardArgs[Context]) bool {
				return a.Context.Applicant.Age >= 18
			}),
		},
	}).CreateMachine(xs.MachineConfig[Context]{
		ID:      "applicantrequest",
		Initial: "CheckApplication",
		ContextFn: func(a xs.ContextArgs) Context {
			return Context{Applicant: a.Input.(Input).Applicant}
		},
		States: xs.States{
			{
				Key: "CheckApplication",
				On: map[string]xs.Transitions{
					"Submit": {
						{Target: "StartApplication", Guard: xs.GuardRef{Type: "isOver18"}},
						{Target: "RejectApplication"},
					},
				},
			},
			{
				Key: "StartApplication",
				Invoke: []xs.InvokeConfig{{
					Src:     "startApplicationWorkflowId",
					OnDone:  xs.Transitions{{Target: "End"}},
					OnError: xs.Transitions{{Target: "RejectApplication"}},
				}},
			},
			{
				Key: "RejectApplication",
				Invoke: []xs.InvokeConfig{{
					Src: "sendRejectionEmailFunction",
					Input: xs.NewExpr(func(a xs.ExprArgs[Context]) any {
						return Input{Applicant: a.Context.Applicant}
					}),
					OnDone: xs.Transitions{{Target: "End"}},
				}},
			},
			{Key: "End", Type: xs.Final},
		},
	})
}

// Machine mirrors `workflow` with the real 1 s actors printing to stdout.
func Machine() *xs.StateMachine[Context] {
	return NewMachine(os.Stdout, time.Second)
}

// lockedWriter serialises writes from the promise goroutines and the subscriber.
type lockedWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func (l *lockedWriter) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.w.Write(p)
}

// Run mirrors the entry of main.ts: the default applicant, 1 s actors, events
// read from stdin, output to w.
func Run(w io.Writer) error {
	return RunWith(context.Background(), w, os.Stdin, DefaultApplicant, time.Second)
}

// RunWith is Run with the stdin source, applicant and actor delay injected.
// Every line read from in is sent as an event of that type (JS: each stdin
// 'data' chunk, trimmed). On completion it prints "workflow completed <output>".
// Like the Node process, it returns once the workflow is done, or once in has
// ended and no invoked actor is pending; ctx cancellation also returns.
func RunWith(ctx context.Context, w io.Writer, in io.Reader, applicant Applicant, delay time.Duration) error {
	out := &lockedWriter{w: w}
	actor := xs.CreateActor(NewMachine(out, delay), xs.WithInput(Input{Applicant: applicant}))
	completed := make(chan struct{})
	actor.Subscribe(xs.Observer[*xs.MachineSnapshot[Context]]{Complete: func() {
		// Stopping an unfinished actor (ctx cancelled, stdin ended while idle) also
		// completes observers; the JS process just exits, so only a done workflow prints.
		snap := actor.GetSnapshot()
		if snap.Status != xs.StatusDone {
			return
		}
		fmt.Fprintf(out, "workflow completed %s\n", formatOutput(snap.Output))
		close(completed)
	}})
	actor.Start()
	defer actor.Stop()

	lines := make(chan string)
	go func() {
		defer close(lines)
		sc := bufio.NewScanner(in)
		for sc.Scan() {
			select {
			case lines <- strings.TrimSpace(sc.Text()):
			case <-ctx.Done():
				return
			}
		}
	}()

	for {
		select {
		case <-completed:
			return nil
		case <-ctx.Done():
			return context.Cause(ctx)
		case line, ok := <-lines:
			if !ok {
				// stdin ended: wait for pending invoked work, as the Node event loop would.
				pending := xs.WaitFor(ctx, actor, func(s *xs.MachineSnapshot[Context]) bool {
					return s.Status != xs.StatusActive || len(s.Children) == 0
				}, xs.WaitForOptions{})
				if _, err := pending.Wait(); err != nil {
					return err
				}
				return nil
			}
			actor.Send(xs.Ev(line))
		}
	}
}

// formatOutput renders a snapshot output the way console.log prints it.
func formatOutput(v any) string {
	if v == nil {
		return "undefined"
	}
	return fmt.Sprint(v)
}
