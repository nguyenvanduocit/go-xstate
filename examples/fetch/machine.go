// Package fetch ports references/xstate/examples/fetch/src/fetchMachine.ts and the non-UI helpers of
// examples/fetch/src/index.ts (getGreeting and the console entry).
package fetch

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"sync"
	"time"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// Greeting is the resolved value of getGreeting: `{ greeting: string }`.
type Greeting struct {
	Greeting string `json:"greeting"`
}

// Context is the fetch machine's extended state.
type Context struct {
	Name string    `json:"name"`
	Data *Greeting `json:"data"`
}

// FetchInput is the input of the fetchUser actor: `{ name }`.
type FetchInput struct {
	Name string `json:"name"`
}

// ErrRejected is what getGreeting returns where the JS promise calls `rej()`.
var ErrRejected = errors.New("rejected")

// GetGreeting mirrors getGreeting: after delay it fails with ErrRejected when
// random() < 0.5, otherwise resolves to `Hello, <name>!`. The JS hardcodes
// 1000 ms and Math.random; both are parameters so tests are deterministic.
// Cancelling ctx stands in for abandoning the promise when the invoking state exits.
func GetGreeting(ctx context.Context, name string, delay time.Duration, random func() float64) (Greeting, error) {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return Greeting{}, context.Cause(ctx)
	case <-timer.C:
	}
	if random() < 0.5 {
		return Greeting{}, ErrRejected
	}
	return Greeting{Greeting: fmt.Sprintf("Hello, %s!", name)}, nil
}

// NewMachine mirrors fetchMachine with getGreeting configured by delay and random.
func NewMachine(delay time.Duration, random func() float64) *xs.StateMachine[Context] {
	fetchUser := xs.FromPromise(func(ctx context.Context, a xs.PromiseArgs) (Greeting, error) {
		return GetGreeting(ctx, a.Input.(FetchInput).Name, delay, random)
	})
	return xs.NewSetup[Context](xs.Implementations{
		Actors: map[string]xs.ActorLogic{"fetchUser": fetchUser},
	}).CreateMachine(xs.MachineConfig[Context]{
		Initial: "idle",
		Context: Context{Name: "World", Data: nil},
		States: xs.States{
			{
				Key: "idle",
				On:  map[string]xs.Transitions{"FETCH": {{Target: "loading"}}},
			},
			{
				Key: "loading",
				Invoke: []xs.InvokeConfig{{
					Src: "fetchUser",
					Input: xs.NewExpr(func(a xs.ExprArgs[Context]) any {
						return FetchInput{Name: a.Context.Name}
					}),
					OnDone: xs.Transitions{{
						Target: "success",
						Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[Context]) Context {
							c := a.Context
							out := a.Event.(xs.DoneActorEvent).Output.(Greeting)
							c.Data = &out
							return c
						})},
					}},
					OnError: xs.Transitions{{Target: "failure"}},
				}},
			},
			{Key: "success"},
			{
				Key:   "failure",
				After: map[string]xs.Transitions{"1000": {{Target: "loading"}}},
				On:    map[string]xs.Transitions{"RETRY": {{Target: "loading"}}},
			},
		},
	})
}

// Machine mirrors fetchMachine with the real getGreeting (1 s delay, 50% failure).
func Machine() *xs.StateMachine[Context] {
	return NewMachine(time.Second, rand.Float64)
}

// Run mirrors the console entry of src/index.ts with the real getGreeting.
// It returns once the machine reaches `success`.
func Run(w io.Writer) error {
	return RunWith(w, time.Second, rand.Float64)
}

// RunWith is Run with getGreeting's delay and random source injected.
func RunWith(w io.Writer, delay time.Duration, random func() float64) error {
	var mu sync.Mutex
	actor := xs.CreateActor(NewMachine(delay, random))
	actor.SubscribeNext(func(s *xs.MachineSnapshot[Context]) {
		mu.Lock()
		defer mu.Unlock()
		fmt.Fprintf(w, "Value: %v\nContext: %s\n", s.Value, formatContext(s.Context))
	})
	pending := xs.WaitFor(context.Background(), actor, func(s *xs.MachineSnapshot[Context]) bool {
		return s.Matches("success")
	}, xs.WaitForOptions{})
	actor.Start()
	actor.Send(xs.Ev("FETCH"))
	_, err := pending.Wait()
	actor.Stop()
	mu.Lock()
	mu.Unlock() //nolint:staticcheck // barrier: every subscriber write has completed
	return err
}

// formatContext renders the context the way console.log prints it in the JS
// runtime (bun): multi-line object, double-quoted strings, trailing commas.
func formatContext(c Context) string {
	s := fmt.Sprintf("{\n  name: %q,\n", c.Name)
	if c.Data == nil {
		return s + "  data: null,\n}"
	}
	return s + fmt.Sprintf("  data: {\n    greeting: %q,\n  },\n}", c.Data.Greeting)
}
