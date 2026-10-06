// Package greeting ports references/xstate/examples/workflow-greeting/main.ts
// (serverless workflow "greeting").
package greeting

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// GreetingDelay is the delay of the greetingFunction actor in main.ts (1000 ms).
const GreetingDelay = time.Second

// Person is `input.person`: `{ name: string }`.
type Person struct {
	Name string `json:"name"`
}

// Input is the machine input: `{ person: { name: string } }`.
type Input struct {
	Person Person `json:"person"`
}

// Context is the machine's extended state. `greeting` is `undefined` until the
// `Greet` invoke resolves; a greeting is never empty ("Hello, <name>!"), so
// omitempty reproduces the absent key in snapshots.
type Context struct {
	Greeting string `json:"greeting,omitempty"`
}

// GreetingInput is the input of the greetingFunction actor: `{ name: string }`.
type GreetingInput struct {
	Name string `json:"name"`
}

// GreetingOutput is the resolved value of greetingFunction: `{ greeting: string }`.
type GreetingOutput struct {
	Greeting string `json:"greeting"`
}

// Output is the output of the `Greeted` final state: `{ greeting: string }`.
type Output struct {
	Greeting string `json:"greeting"`
}

// GreetingFunction mirrors the greetingFunction actor of main.ts: it waits delay
// (1000 ms in the JS) and resolves with `Hello, <name>!`. Cancelling ctx stands
// in for abandoning the promise when the invoking state exits.
func GreetingFunction(delay time.Duration) xs.ActorLogic {
	return xs.FromPromise(func(ctx context.Context, a xs.PromiseArgs) (GreetingOutput, error) {
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return GreetingOutput{}, context.Cause(ctx)
		case <-timer.C:
		}
		return GreetingOutput{Greeting: fmt.Sprintf("Hello, %s!", a.Input.(GreetingInput).Name)}, nil
	})
}

// NewMachine mirrors `workflow` with the greetingFunction actor given.
func NewMachine(greetingFunction xs.ActorLogic) *xs.StateMachine[Context] {
	return xs.NewSetup[Context](xs.Implementations{
		Actors: map[string]xs.ActorLogic{"greetingFunction": greetingFunction},
	}).CreateMachine(xs.MachineConfig[Context]{
		ID:      "greeting",
		Context: Context{},
		Initial: "Greet",
		States: xs.States{
			{
				Key: "Greet",
				Invoke: []xs.InvokeConfig{{
					Src: "greetingFunction",
					// `event.input.person.name`: the event is xstate.init carrying the actor input.
					Input: xs.NewExpr(func(a xs.ExprArgs[Context]) any {
						return GreetingInput{Name: a.Event.(xs.InitEvent).Input.(Input).Person.Name}
					}),
					OnDone: xs.Transitions{{
						Target: "Greeted",
						Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[Context]) Context {
							return Context{Greeting: a.Event.(xs.DoneActorEvent).Output.(GreetingOutput).Greeting}
						})},
					}},
				}},
			},
			{
				Key:  "Greeted",
				Type: xs.Final,
				Output: xs.NewExpr(func(a xs.ExprArgs[Context]) any {
					return Output{Greeting: a.Context.Greeting}
				}),
			},
		},
	})
}

// Run mirrors the entry of main.ts with the real 1000 ms greetingFunction, printing to w.
func Run(w io.Writer) {
	RunWith(w, GreetingDelay)
}

// RunWith is Run with greetingFunction's delay injected.
func RunWith(w io.Writer, delay time.Duration) {
	var mu sync.Mutex
	locked := &lockedWriter{mu: &mu, w: w}
	done := make(chan struct{})
	actor := xs.CreateActor(NewMachine(GreetingFunction(delay)), xs.WithInput(Input{Person: Person{Name: "Jenny"}}))
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
// JS runtime: the root snapshot's output is `undefined` (the `Greeted` state's
// `output` only feeds the parent's done event, and this machine has no root
// `output`), so a nil output prints `undefined`.
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
