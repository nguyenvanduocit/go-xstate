// Package newpatientonboarding ports the new-patient onboarding workflow.
package newpatientonboarding

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"sync"
	"time"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

type Patient struct {
	Name      string `json:"name"`
	Condition string `json:"condition"`
}
type Context struct {
	Patient *Patient `json:"patient"`
}

// ServiceError carries the error type used by the reference retry predicate.
type ServiceError struct{ Type string }

func (e ServiceError) Error() string { return e.Type }

// Delay is the timer/random boundary shared by the three service implementations.
type Delay func(context.Context, time.Duration, float64) error

func RandomDelay(ctx context.Context, duration time.Duration, probability float64) error {
	if err := Sleep(ctx, duration); err != nil {
		return err
	}
	if rand.Float64() < probability {
		return ServiceError{Type: "ServiceNotAvailable"}
	}
	return nil
}
func Sleep(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return context.Cause(ctx)
	case <-timer.C:
		return nil
	}
}

// Retry performs the initial call and at most ten retries, matching Cockatiel's maxAttempts.
func Retry(ctx context.Context, call func() error, backoff func(context.Context, time.Duration) error, onRetry func(int, error)) error {
	for attempt := 0; ; attempt++ {
		err := call()
		if err == nil {
			return nil
		}
		var serviceError ServiceError
		if !errors.As(err, &serviceError) || serviceError.Type != "ServiceNotAvailable" || attempt == 10 {
			return err
		}
		if onRetry != nil {
			onRetry(attempt+1, err)
		}
		if err = backoff(ctx, 3*time.Second); err != nil {
			return err
		}
	}
}

func Services(w io.Writer, delay Delay, backoff func(context.Context, time.Duration) error) map[string]xs.ActorLogic {
	actors := map[string]xs.ActorLogic{}
	for _, name := range []string{"StoreNewPatientInfo", "AssignDoctor", "ScheduleAppt"} {
		actors[name] = xs.FromPromise(func(ctx context.Context, args xs.PromiseArgs) (any, error) {
			if name == "StoreNewPatientInfo" {
				p := args.Input.(*Patient)
				fmt.Fprintf(w, "Starting StoreNewPatientInfo {\n  name: %q,\n  condition: %q,\n}\n", p.Name, p.Condition)
			} else {
				fmt.Fprintln(w, "Starting", name)
			}
			err := Retry(ctx, func() error { return delay(ctx, time.Second, 0.5) }, backoff, func(attempt int, err error) {
				fmt.Fprintf(w, "Retrying... {\n  error: {\n    type: \"ServiceNotAvailable\",\n  },\n  delay: 3000,\n  attempt: %d,\n}\n", attempt)
			})
			if err != nil {
				return nil, err
			}
			fmt.Fprintln(w, "Completed", name)
			return nil, nil
		})
	}
	return actors
}

func NewMachine(actors map[string]xs.ActorLogic) *xs.StateMachine[Context] {
	invoke := func(src, target string) []xs.InvokeConfig {
		return []xs.InvokeConfig{{Src: src, OnDone: xs.Transitions{{Target: target}}, OnError: xs.Transitions{{Target: "#End"}}}}
	}
	store := invoke("StoreNewPatientInfo", "AssignDoctor")
	store[0].Input = xs.NewExpr(func(a xs.ExprArgs[Context]) any { return a.Context.Patient })
	return xs.NewSetup[Context](xs.Implementations{Actors: actors}).CreateMachine(xs.MachineConfig[Context]{
		ID: "patientonboarding", Initial: "Idle", Context: Context{}, States: xs.States{
			{Key: "Idle", On: map[string]xs.Transitions{"NewPatientEvent": {{Target: "Onboard", Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[Context]) Context {
				event := a.Event.(xs.E)
				return Context{Patient: &Patient{Name: event["name"].(string), Condition: event["condition"].(string)}}
			})}}}}},
			{Key: "Onboard", Initial: "StorePatient", States: xs.States{
				{Key: "StorePatient", Invoke: store}, {Key: "AssignDoctor", Invoke: invoke("AssignDoctor", "ScheduleAppt")},
				{Key: "ScheduleAppt", Invoke: invoke("ScheduleAppt", "Done")}, {Key: "Done", Type: xs.Final},
			}, OnDone: xs.Transitions{{Target: "End", Actions: xs.Actions{xs.Assign(func(xs.AssignArgs[Context]) Context { return Context{} })}}}},
			{Key: "End", ID: "End", Type: xs.Final},
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

func Run(w io.Writer) error { return RunWith(context.Background(), w, RandomDelay, Sleep) }
func RunWith(ctx context.Context, w io.Writer, delay Delay, backoff func(context.Context, time.Duration) error) error {
	out := &lockedWriter{w: w}
	actor := xs.CreateActor(NewMachine(Services(out, delay, backoff)))
	done := make(chan struct{})
	actor.Subscribe(xs.Observer[*xs.MachineSnapshot[Context]]{Complete: func() {
		if actor.GetSnapshot().Status == xs.StatusDone {
			fmt.Fprintln(out, "workflow completed undefined")
			close(done)
		}
	}})
	actor.Start()
	defer actor.Stop()
	actor.Send(xs.E{"type": "NewPatientEvent", "name": "John Doe", "condition": "Broken Arm"})
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return context.Cause(ctx)
	}
}
