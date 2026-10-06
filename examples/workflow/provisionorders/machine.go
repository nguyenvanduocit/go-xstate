// Package provisionorders ports references/xstate/examples/workflow-provision-orders/main.ts
// (the Serverless Workflow "provision orders" example).
package provisionorders

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// Order mirrors the JS `Order` interface (all fields are strings).
type Order struct {
	ID       string `json:"id"`
	Item     string `json:"item"`
	Quantity string `json:"quantity"`
}

// Input is the machine input: `{ order }`.
type Input struct {
	Order Order `json:"order"`
}

// Context is the machine's extended state: `{ order }`.
type Context struct {
	Order Order `json:"order"`
}

// DefaultInput is the order hardcoded in the JS entry's createActor call
// (empty id, so the MissingId exception path runs).
var DefaultInput = Input{Order: Order{ID: "", Item: "laptop", Quantity: "10"}}

// Error messages thrown by provisionOrderFunction and matched by the onError guards.
const (
	missingIDMessage       = "Missing order id"
	missingItemMessage     = "Missing order item"
	missingQuantityMessage = "Missing order quantity"
)

// wait blocks for delay or until ctx is cancelled. Cancelling ctx stands in for
// abandoning the promise when the invoking state exits.
func wait(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return context.Cause(ctx)
	case <-timer.C:
		return nil
	}
}

// provisionOrderFunction mirrors the actor of the same name: print, wait delay
// (1000 ms in JS), validate id, item and quantity in that order, print, resolve
// with `{ order }`.
func provisionOrderFunction(w io.Writer, delay time.Duration) xs.ActorLogic {
	return xs.FromPromise(func(ctx context.Context, a xs.PromiseArgs) (Input, error) {
		in := a.Input.(Input)
		fmt.Fprintln(w, "starting provisionOrderFunction")
		if err := wait(ctx, delay); err != nil {
			return Input{}, err
		}
		switch {
		case in.Order.ID == "":
			return Input{}, errors.New(missingIDMessage)
		case in.Order.Item == "":
			return Input{}, errors.New(missingItemMessage)
		case in.Order.Quantity == "":
			return Input{}, errors.New(missingQuantityMessage)
		}
		fmt.Fprintln(w, "finished provisionOrderFunction")
		return Input{Order: in.Order}, nil
	})
}

// serviceCall mirrors applyOrderWorkflowId and the three handleMissing*ExceptionWorkflow
// actors: print "starting <name>", wait delay, print "finished <name>".
func serviceCall(w io.Writer, name string, delay time.Duration) xs.ActorLogic {
	return xs.FromPromise(func(ctx context.Context, _ xs.PromiseArgs) (struct{}, error) {
		fmt.Fprintf(w, "starting %s\n", name)
		if err := wait(ctx, delay); err != nil {
			return struct{}{}, err
		}
		fmt.Fprintf(w, "finished %s\n", name)
		return struct{}{}, nil
	})
}

// errorIs is the shape of the three onError guards:
// `(event.error as any).message === message`.
func errorIs(message string) xs.Guard {
	return xs.GuardFunc(func(a xs.GuardArgs[Context]) bool {
		err, ok := a.Event.(xs.ErrorActorEvent).Error.(error)
		return ok && err.Error() == message
	})
}

// exceptionHandler is one Exception.<Missing*> state: invoke the handler, then
// go to the sibling final state `End`.
func exceptionHandler(key, src string) xs.StateConfig {
	return xs.StateConfig{
		Key:    key,
		Invoke: []xs.InvokeConfig{{Src: src, OnDone: xs.Transitions{{Target: "End"}}}},
	}
}

// NewMachine mirrors `workflow`, with the actors' console.log output going to w
// and their delay (1000 ms in JS) injected.
func NewMachine(w io.Writer, delay time.Duration) *xs.StateMachine[Context] {
	return xs.NewSetup[Context](xs.Implementations{
		Actors: map[string]xs.ActorLogic{
			"provisionOrderFunction":                 provisionOrderFunction(w, delay),
			"applyOrderWorkflowId":                   serviceCall(w, "applyOrderWorkflowId", delay),
			"handleMissingIdExceptionWorkflow":       serviceCall(w, "handleMissingIdExceptionWorkflow", delay),
			"handleMissingItemExceptionWorkflow":     serviceCall(w, "handleMissingItemExceptionWorkflow", delay),
			"handleMissingQuantityExceptionWorkflow": serviceCall(w, "handleMissingQuantityExceptionWorkflow", delay),
		},
	}).CreateMachine(xs.MachineConfig[Context]{
		ID:      "provisionorders",
		Initial: "ProvisionOrder",
		ContextFn: func(a xs.ContextArgs) Context {
			return Context{Order: a.Input.(Input).Order}
		},
		States: xs.States{
			{
				Key: "ProvisionOrder",
				Invoke: []xs.InvokeConfig{{
					Src: "provisionOrderFunction",
					Input: xs.NewExpr(func(a xs.ExprArgs[Context]) any {
						return Input{Order: a.Context.Order}
					}),
					OnDone: xs.Transitions{{Target: "ApplyOrder"}},
					OnError: xs.Transitions{
						{Guard: errorIs(missingIDMessage), Target: "Exception.MissingId"},
						{Guard: errorIs(missingItemMessage), Target: "Exception.MissingItem"},
						{Guard: errorIs(missingQuantityMessage), Target: "Exception.MissingQuantity"},
					},
				}},
			},
			{
				Key:    "ApplyOrder",
				Invoke: []xs.InvokeConfig{{Src: "applyOrderWorkflowId", OnDone: xs.Transitions{{Target: "End"}}}},
			},
			{Key: "End", Type: xs.Final},
			{
				Key:     "Exception",
				Initial: "MissingId",
				States: xs.States{
					exceptionHandler("MissingId", "handleMissingIdExceptionWorkflow"),
					exceptionHandler("MissingItem", "handleMissingItemExceptionWorkflow"),
					exceptionHandler("MissingQuantity", "handleMissingQuantityExceptionWorkflow"),
					{Key: "End", Type: xs.Final},
				},
				OnDone: xs.Transitions{{Target: "End"}},
			},
		},
	})
}

// Run mirrors the entry of main.ts: the hardcoded order with the real 1 s
// actors, printing to w until the workflow completes.
func Run(w io.Writer) {
	RunWith(w, DefaultInput, time.Second)
}

// RunWith is Run with the order and the actors' delay injected.
func RunWith(w io.Writer, in Input, delay time.Duration) {
	out := &lockedWriter{w: w}
	done := make(chan struct{})
	actor := xs.CreateActor(NewMachine(out, delay), xs.WithInput(in))
	actor.Subscribe(xs.Observer[*xs.MachineSnapshot[Context]]{
		Complete: func() {
			fmt.Fprintln(out, "workflow completed", formatOutput(actor.GetSnapshot().Output))
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
