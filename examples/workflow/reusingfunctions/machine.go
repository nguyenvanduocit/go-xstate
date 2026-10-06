// Package reusingfunctions ports the payment confirmation child and its parent.
package reusingfunctions

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"sync"
	"time"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

type Payment struct {
	Amount float64 `json:"amount"`
}
type Customer struct {
	Name string `json:"name"`
}
type Funds struct {
	Available bool `json:"available"`
}
type Context struct {
	Customer  *Customer `json:"customer"`
	Payment   *Payment  `json:"payment"`
	Funds     *Funds    `json:"funds"`
	AccountID *string   `json:"accountId"`
}
type CheckInput struct {
	Account       *string `json:"account"`
	PaymentAmount float64 `json:"paymentamount"`
}
type EmailInput struct {
	Applicant *Customer `json:"applicant"`
}
type ParentContext struct{}

func wait(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return context.Cause(ctx)
	case <-timer.C:
		return nil
	}
}
func Services(w io.Writer, d time.Duration) map[string]xs.ActorLogic {
	return services(w, d, nil, nil)
}

func services(w io.Writer, d time.Duration, started, beforeDone func(string)) map[string]xs.ActorLogic {
	actors := map[string]xs.ActorLogic{"checkfunds": xs.FromPromise(func(ctx context.Context, a xs.PromiseArgs) (Funds, error) {
		fmt.Fprintln(w, "Running checkfunds")
		if started != nil {
			started("checkfunds")
		}
		if err := wait(ctx, d); err != nil {
			return Funds{}, err
		}
		if beforeDone != nil {
			beforeDone("checkfunds")
		}
		fmt.Fprintln(w, "checkfunds done")
		return Funds{Available: a.Input.(CheckInput).PaymentAmount < 1000}, nil
	})}
	for _, name := range []string{"sendSuccessEmail", "sendInsufficientFundsEmail"} {
		actors[name] = xs.FromPromise(func(ctx context.Context, a xs.PromiseArgs) (any, error) {
			fmt.Fprintf(w, "{\n  input: {\n    applicant: {\n      name: %q,\n    },\n  },\n}\n", a.Input.(EmailInput).Applicant.Name)
			fmt.Fprintln(w, "Running", name)
			if started != nil {
				started(name)
			}
			if err := wait(ctx, d); err != nil {
				return nil, err
			}
			if beforeDone != nil {
				beforeDone(name)
			}
			fmt.Fprintln(w, name, "done")
			return nil, nil
		})
	}
	return actors
}
func NewMachine(actors map[string]xs.ActorLogic) *xs.StateMachine[Context] {
	email := func(src string) []xs.InvokeConfig {
		return []xs.InvokeConfig{{Src: src, Input: xs.NewExpr(func(a xs.ExprArgs[Context]) any { return EmailInput{Applicant: a.Context.Customer} }), OnDone: xs.Transitions{{Target: "End"}}}}
	}
	return xs.NewSetup[Context](xs.Implementations{Actors: actors, Guards: map[string]xs.Guard{"fundsAvailable": xs.GuardFunc(func(a xs.GuardArgs[Context]) bool { return a.Context.Funds != nil && a.Context.Funds.Available })}}).CreateMachine(xs.MachineConfig[Context]{
		ID: "paymentconfirmation", Initial: "Pending", States: xs.States{
			{Key: "Pending", On: map[string]xs.Transitions{"PaymentReceivedEvent": {{Target: "PaymentReceived", Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[Context]) Context {
				e := a.Event.(xs.E)
				c := a.Context
				c.Payment = &Payment{Amount: e["payment"].(map[string]any)["amount"].(float64)}
				c.Customer = &Customer{Name: e["customer"].(map[string]any)["name"].(string)}
				c.Funds = &Funds{Available: e["funds"].(map[string]any)["available"].(bool)}
				return c
			})}}}}},
			{Key: "PaymentReceived", Invoke: []xs.InvokeConfig{{Src: "checkfunds", Input: xs.NewExpr(func(a xs.ExprArgs[Context]) any {
				return CheckInput{Account: a.Context.AccountID, PaymentAmount: a.Context.Payment.Amount}
			}), OnDone: xs.Transitions{{Target: "ConfirmBasedOnFunds", Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[Context]) Context {
				c := a.Context
				funds := a.Event.(xs.DoneActorEvent).Output.(Funds)
				c.Funds = &funds
				return c
			})}}}}}},
			{Key: "ConfirmBasedOnFunds", Always: xs.Transitions{{Target: "SendPaymentSuccess", Guard: xs.GuardRef{Type: "fundsAvailable"}}, {Target: "SendInsufficientResults"}}},
			{Key: "SendPaymentSuccess", Invoke: email("sendSuccessEmail")}, {Key: "SendInsufficientResults", Invoke: email("sendInsufficientFundsEmail")},
			{Key: "End", Type: xs.Final, Entry: xs.Actions{xs.SendParent(xs.NewExpr(func(a xs.ExprArgs[Context]) any {
				return xs.E{"type": "ConfirmationCompletedEvent", "payment": a.Context.Payment}
			}))}},
		},
	})
}

// NewParent forwards payment events, observes child snapshots, and receives completion events.
func NewParent(child xs.ActorLogic, onSnapshot func(*xs.MachineSnapshot[Context]), onEvent func(xs.Event)) *xs.StateMachine[ParentContext] {
	return xs.NewSetup[ParentContext](xs.Implementations{Actors: map[string]xs.ActorLogic{"workflow": child}}).CreateMachine(xs.MachineConfig[ParentContext]{
		ID: "parent",
		Invoke: []xs.InvokeConfig{{
			ID:  "paymentconfirmation",
			Src: "workflow",
			OnSnapshot: xs.Transitions{{Actions: xs.Actions{
				xs.ActionFunc(func(a xs.ActionArgs[ParentContext]) {
					if onSnapshot != nil {
						onSnapshot(a.Event.(xs.SnapshotEvent).Snapshot.(*xs.MachineSnapshot[Context]))
					}
				}),
			}}},
		}},
		On: map[string]xs.Transitions{
			"PaymentReceivedEvent": {{Actions: xs.Actions{xs.ForwardTo("paymentconfirmation")}}},
			"*": {{Actions: xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[ParentContext]) {
				if onEvent != nil {
					onEvent(a.Event)
				}
			})}}},
		},
	})
}

func snapshotJSON(s *xs.MachineSnapshot[Context]) []byte {
	children := make([]string, 0, len(s.Children))
	for id := range s.Children {
		children = append(children, id)
	}
	sort.Strings(children)
	tags := append([]string{}, s.Tags...)
	sort.Strings(tags)
	b, _ := json.Marshal(struct {
		Status   xs.Status `json:"status"`
		Value    any       `json:"value"`
		Context  Context   `json:"context"`
		Output   any       `json:"output"`
		Tags     []string  `json:"tags"`
		Children []string  `json:"children"`
	}{s.Status, s.Value, s.Context, s.Output, tags, children})
	return b
}

type lockedWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func (w *lockedWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.w.Write(p)
}
func Run(w io.Writer) error { return RunWith(context.Background(), w, time.Second) }
func RunWith(ctx context.Context, w io.Writer, d time.Duration) error {
	out := &lockedWriter{w: w}
	done := make(chan struct{})
	// JS executes pre-await logs before publishing the invoked state's snapshot.
	// Coordinate the Go goroutines at that same observable boundary.
	started := map[string]chan struct{}{}
	observed := map[string]chan struct{}{}
	for _, name := range []string{"checkfunds", "sendSuccessEmail", "sendInsufficientFundsEmail"} {
		started[name] = make(chan struct{})
		observed[name] = make(chan struct{})
	}
	actors := services(out, d, func(name string) { close(started[name]) }, func(name string) { <-observed[name] })
	parent := NewParent(NewMachine(actors), func(s *xs.MachineSnapshot[Context]) {
		service := ""
		switch s.Value {
		case "PaymentReceived":
			service = "checkfunds"
		case "SendPaymentSuccess":
			service = "sendSuccessEmail"
		case "SendInsufficientResults":
			service = "sendInsufficientFundsEmail"
		}
		if service != "" {
			<-started[service]
		}
		fmt.Fprintln(out, string(snapshotJSON(s)))
		if service != "" {
			close(observed[service])
		}
	}, func(event xs.Event) {
		switch e := event.(type) {
		case xs.E:
			p := e["payment"].(*Payment)
			fmt.Fprintf(out, "Received event {\n  type: %q,\n  payment: {\n    amount: %v,\n  },\n}\n", e.EventType(), p.Amount)
		case xs.DoneActorEvent:
			fmt.Fprintf(out, "Received event {\n  type: %q,\n  output: undefined,\n  actorId: %q,\n}\n", e.EventType(), e.ActorID)
			close(done)
		}
	})
	actor := xs.CreateActor(parent)
	actor.Start()
	defer actor.Stop()
	actor.Send(xs.E{"type": "PaymentReceivedEvent", "accountId": "1234", "payment": map[string]any{"amount": float64(100)}, "customer": map[string]any{"name": "John Doe"}, "funds": map[string]any{"available": true}})
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return context.Cause(ctx)
	}
}
