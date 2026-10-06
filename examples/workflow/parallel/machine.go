// Package parallel ports references/xstate/examples/workflow-parallel/main.ts
// (serverless workflow "parallel execution").
package parallel

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// Context is the machine's extended state. The JS machine declares none, so a
// snapshot carries `{}`.
type Context struct{}

// Delay mirrors the shortDelay / longDelay actors of main.ts: it waits d, logs
// "Resolved <name>" to w and resolves. Cancelling ctx stands in for abandoning
// the promise when the invoking state exits.
func Delay(w io.Writer, name string, d time.Duration) xs.ActorLogic {
	return xs.FromPromise(func(ctx context.Context, _ xs.PromiseArgs) (struct{}, error) {
		timer := time.NewTimer(d)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return struct{}{}, context.Cause(ctx)
		case <-timer.C:
		}
		fmt.Fprintln(w, "Resolved", name)
		return struct{}{}, nil
	})
}

// branch is one region of ParallelExec: `active` invokes src and moves to the
// final state `done`.
func branch(key, src string) xs.StateConfig {
	return xs.StateConfig{
		Key:     key,
		Initial: "active",
		States: xs.States{
			{
				Key: "active",
				Invoke: []xs.InvokeConfig{{
					Src:    src,
					OnDone: xs.Transitions{{Target: "done"}},
				}},
			},
			{Key: "done", Type: xs.Final},
		},
	}
}

// NewMachine mirrors `workflow` with the shortDelay and longDelay actors given.
func NewMachine(shortDelay, longDelay xs.ActorLogic) *xs.StateMachine[Context] {
	return xs.NewSetup[Context](xs.Implementations{
		Actors: map[string]xs.ActorLogic{
			"shortDelay": shortDelay,
			"longDelay":  longDelay,
		},
	}).CreateMachine(xs.MachineConfig[Context]{
		ID:      "parallel-execution",
		Initial: "ParallelExec",
		States: xs.States{
			{
				Key:    "ParallelExec",
				Type:   xs.Parallel,
				OnDone: xs.Transitions{{Target: "Success"}},
				States: xs.States{
					branch("ShortDelayBranch", "shortDelay"),
					branch("LongDelayBranch", "longDelay"),
				},
			},
			{Key: "Success", Type: xs.Final},
		},
	})
}

// Run mirrors the entry of main.ts with the real 1000 ms and 3000 ms delays, printing to w.
func Run(w io.Writer) {
	RunWith(w, time.Second, 3*time.Second)
}

// RunWith is Run with the two delays injected. It returns once "workflow completed"
// has been printed.
func RunWith(w io.Writer, short, long time.Duration) {
	locked := &lockedWriter{w: w}
	done := make(chan struct{})
	actor := xs.CreateActor(NewMachine(
		Delay(locked, "shortDelay", short),
		Delay(locked, "longDelay", long),
	))
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

// lockedWriter serialises the writes of the two promise goroutines and the subscriber.
type lockedWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func (l *lockedWriter) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.w.Write(p)
}
