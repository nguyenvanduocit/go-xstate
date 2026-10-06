// Package asyncfunction ports references/xstate/examples/workflow-async-function/main.ts
// (serverless workflow "async function invocation").
package asyncfunction

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// Input is the machine input: `{ customer: string }`.
type Input struct {
	Customer string `json:"customer"`
}

// Context is the machine's extended state.
type Context struct {
	Customer string `json:"customer"`
}

// SendEmail mirrors the sendEmail actor of main.ts: it logs to w, waits delay
// (1000 ms in the JS), logs again and resolves. Cancelling ctx stands in for
// abandoning the promise when the invoking state exits.
func SendEmail(w io.Writer, delay time.Duration) xs.ActorLogic {
	return xs.FromPromise(func(ctx context.Context, a xs.PromiseArgs) (struct{}, error) {
		in := a.Input.(Input)
		fmt.Fprintln(w, "Sending email to", in.Customer)
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return struct{}{}, context.Cause(ctx)
		case <-timer.C:
		}
		fmt.Fprintln(w, "Email sent to", in.Customer)
		return struct{}{}, nil
	})
}

// NewMachine mirrors `workflow` with the sendEmail actor given.
func NewMachine(sendEmail xs.ActorLogic) *xs.StateMachine[Context] {
	return xs.NewSetup[Context](xs.Implementations{
		Actors: map[string]xs.ActorLogic{"sendEmail": sendEmail},
	}).CreateMachine(xs.MachineConfig[Context]{
		ID:      "async-function-invocation",
		Initial: "Send email",
		ContextFn: func(a xs.ContextArgs) Context {
			return Context{Customer: a.Input.(Input).Customer}
		},
		States: xs.States{
			{
				Key: "Send email",
				Invoke: []xs.InvokeConfig{{
					Src: "sendEmail",
					Input: xs.NewExpr(func(a xs.ExprArgs[Context]) any {
						return Input{Customer: a.Context.Customer}
					}),
					OnDone: xs.Transitions{{Target: "Email sent"}},
				}},
			},
			{Key: "Email sent", Type: xs.Final},
		},
	})
}

// Run mirrors the entry of main.ts with the real 1000 ms sendEmail, printing to w.
func Run(w io.Writer) {
	RunWith(w, time.Second)
}

// RunWith is Run with sendEmail's delay injected.
func RunWith(w io.Writer, delay time.Duration) {
	var mu sync.Mutex
	locked := &lockedWriter{mu: &mu, w: w}
	done := make(chan struct{})
	actor := xs.CreateActor(NewMachine(SendEmail(locked, delay)), xs.WithInput(Input{Customer: "david@example.com"}))
	actor.Subscribe(xs.Observer[*xs.MachineSnapshot[Context]]{
		Complete: func() {
			fmt.Fprintln(locked, "workflow completed", formatOutput(actor.GetSnapshot().Output))
			close(done)
		},
	})
	actor.Start()
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

type lockedWriter struct {
	mu *sync.Mutex
	w  io.Writer
}

func (l *lockedWriter) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.w.Write(p)
}
