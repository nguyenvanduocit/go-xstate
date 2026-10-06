// Package purchaseorderdeadline ports the purchase-order deadline workflow.
package purchaseorderdeadline

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

type Context struct{}

const Deadline = 15 * time.Second

func CancelOrder(w io.Writer, delay time.Duration) xs.ActorLogic {
	return xs.FromPromise(func(ctx context.Context, _ xs.PromiseArgs) (any, error) {
		fmt.Fprintln(w, "Starting CancelOrder")
		if err := sleep(ctx, delay); err != nil {
			return nil, err
		}
		fmt.Fprintln(w, "Completed CancelOrder")
		return nil, nil
	})
}
func sleep(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return context.Cause(ctx)
	case <-timer.C:
		return nil
	}
}
func NewMachine(w io.Writer, cancel xs.ActorLogic) *xs.StateMachine[Context] {
	actions := map[string]xs.Action{}
	for _, name := range []string{"logNewOrderCreated", "logOrderConfirmed", "logOrderShipped", "logOrderFinished", "logOrderCancelled"} {
		actions[name] = xs.ActionFunc(func(xs.ActionArgs[Context]) { fmt.Fprintln(w, name) })
	}
	return xs.NewSetup[Context](xs.Implementations{Actors: map[string]xs.ActorLogic{"CancelOrder": cancel}, Actions: actions, Delays: map[string]any{"PT30D": Deadline}}).CreateMachine(xs.MachineConfig[Context]{
		ID: "order", Initial: "StartNewOrder", After: map[string]xs.Transitions{"PT30D": {{Target: ".CancelOrder"}}}, States: xs.States{
			{Key: "StartNewOrder", On: map[string]xs.Transitions{"OrderCreatedEvent": {{Target: "WaitForOrderConfirmation", Actions: xs.Actions{xs.ActionRef{Type: "logNewOrderCreated"}}}}}},
			{Key: "WaitForOrderConfirmation", On: map[string]xs.Transitions{"OrderConfirmedEvent": {{Target: "WaitOrderShipped", Actions: xs.Actions{xs.ActionRef{Type: "logOrderConfirmed"}}}}}},
			{Key: "WaitOrderShipped", On: map[string]xs.Transitions{"ShipmentSentEvent": {{Target: "OrderFinished", Actions: xs.Actions{xs.ActionRef{Type: "logOrderShipped"}}}}}},
			{Key: "OrderFinished", Type: xs.Final, Entry: xs.Actions{xs.ActionRef{Type: "logOrderFinished"}}},
			{Key: "CancelOrder", Invoke: []xs.InvokeConfig{{Src: "CancelOrder", OnDone: xs.Transitions{{Target: "OrderCancelled"}}}}},
			{Key: "OrderCancelled", Type: xs.Final, Entry: xs.Actions{xs.ActionRef{Type: "logOrderCancelled"}}},
		},
	})
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
func Run(w io.Writer) error { return RunWith(context.Background(), w, 1) }

// RunWith scales the entry's timers; the machine deadline remains 15 seconds by default.
func RunWith(ctx context.Context, w io.Writer, scale float64) error {
	out := &lockedWriter{w: w}
	machine := NewMachine(out, CancelOrder(out, time.Duration(float64(time.Second)*scale)))
	machine = machine.Provide(xs.Implementations{Delays: map[string]any{"PT30D": time.Duration(float64(Deadline) * scale)}})
	actor := xs.CreateActor(machine)
	done := make(chan struct{})
	actor.Subscribe(xs.Observer[*xs.MachineSnapshot[Context]]{Complete: func() {
		if actor.GetSnapshot().Status == xs.StatusDone {
			fmt.Fprintln(out, "workflow completed undefined")
			close(done)
		}
	}})
	actor.Start()
	defer actor.Stop()
	actor.Send(xs.Ev("OrderCreatedEvent"))
	if err := sleep(ctx, time.Duration(float64(10*time.Second)*scale)); err != nil {
		return err
	}
	actor.Send(xs.Ev("OrderConfirmedEvent"))
	if err := sleep(ctx, time.Duration(float64(10*time.Second)*scale)); err != nil {
		return err
	}
	actor.Send(xs.Ev("ShipmentSentEvent"))
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return context.Cause(ctx)
	}
}
