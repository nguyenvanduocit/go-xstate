// Package sendcloudevent ports the provisioning workflow and its entry.
package sendcloudevent

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

type Order struct {
	ID       string `json:"id"`
	Item     string `json:"item"`
	Quantity string `json:"quantity"`
}
type Result struct {
	ID      string `json:"id"`
	Outcome string `json:"outcome"`
}
type Input struct {
	Orders []Order `json:"orders"`
}
type Context struct {
	Orders            []Order   `json:"orders"`
	ProvisionedOrders *[]Result `json:"provisionedOrders,omitempty"`
}

// ProvisionOrders starts one timer per order and preserves the input order of results.
func ProvisionOrders(w io.Writer, delay time.Duration) *xs.PromiseLogic[[]Result] {
	return xs.FromPromise(func(ctx context.Context, args xs.PromiseArgs) ([]Result, error) {
		orders := args.Input.(Input).Orders
		results := make([]Result, len(orders))
		var wg sync.WaitGroup
		for i, order := range orders {
			fmt.Fprintf(w, "provisioning order {\n  id: %q,\n  item: %q,\n  quantity: %q,\n}\n", order.ID, order.Item, order.Quantity)
			wg.Add(1)
			go func() {
				defer wg.Done()
				timer := time.NewTimer(delay)
				defer timer.Stop()
				select {
				case <-ctx.Done():
					return
				case <-timer.C:
					results[i] = Result{ID: order.ID, Outcome: "SUCCESS"}
				}
			}()
		}
		wg.Wait()
		if err := context.Cause(ctx); err != nil {
			return nil, err
		}
		return results, nil
	})
}
func NewMachine(provision xs.ActorLogic) *xs.StateMachine[Context] {
	return xs.NewSetup[Context](xs.Implementations{Actors: map[string]xs.ActorLogic{"provisionOrdersFunction": provision}}).CreateMachine(xs.MachineConfig[Context]{
		ID: "sendcloudeventonprovision", Initial: "ProvisionOrdersState", ContextFn: func(a xs.ContextArgs) Context { return Context{Orders: a.Input.(Input).Orders} }, States: xs.States{
			{Key: "ProvisionOrdersState", Invoke: []xs.InvokeConfig{{Src: "provisionOrdersFunction", Input: xs.NewExpr(func(a xs.ExprArgs[Context]) any { return Input{Orders: a.Context.Orders} }), OnDone: xs.Transitions{{Target: "End", Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[Context]) Context {
				c := a.Context
				results := a.Event.(xs.DoneActorEvent).Output.([]Result)
				c.ProvisionedOrders = &results
				return c
			})}}}}}},
			{Key: "End", Type: xs.Final, Output: xs.NewExpr(func(a xs.ExprArgs[Context]) any {
				return struct {
					ProvisionedOrders *[]Result `json:"provisionedOrders"`
				}{a.Context.ProvisionedOrders}
			})},
		},
	})
}
func Run(w io.Writer) error { return RunWith(context.Background(), w, time.Second) }
func RunWith(ctx context.Context, w io.Writer, delay time.Duration) error {
	actor := xs.CreateActor(NewMachine(ProvisionOrders(w, delay)), xs.WithInput(Input{Orders: []Order{{"123", "laptop", "10"}, {"456", "desktop", "4"}}}))
	done := make(chan struct{})
	actor.Subscribe(xs.Observer[*xs.MachineSnapshot[Context]]{Complete: func() {
		if actor.GetSnapshot().Status == xs.StatusDone {
			fmt.Fprintln(w, "workflow completed undefined")
			close(done)
		}
	}})
	actor.Start()
	defer actor.Stop()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return context.Cause(ctx)
	}
}
