// Package eventbased ports references/xstate/examples/workflow-event-based/main.ts
// (serverless workflow "event-based transitions").
package eventbased

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// VisaDecisionTimeout is the `visaDecisionTimeout` delay of the JS machine (1000 ms).
const VisaDecisionTimeout = 1000 * time.Millisecond

// Context is empty: the JS machine declares none, and xstate snapshots then
// carry `context: {}`, which an empty struct serializes to.
type Context struct{}

// Handler mirrors one of the three handle*Workflow actors of main.ts: it logs
// "<name> workflow started" to w, waits delay (1000 ms in the JS), logs
// "<name> workflow completed" and resolves. Cancelling ctx stands in for
// abandoning the promise when the invoking state exits.
func Handler(w io.Writer, name string, delay time.Duration) xs.ActorLogic {
	return xs.FromPromise(func(ctx context.Context, _ xs.PromiseArgs) (struct{}, error) {
		fmt.Fprintln(w, name, "workflow started")
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return struct{}{}, context.Cause(ctx)
		case <-timer.C:
		}
		fmt.Fprintln(w, name, "workflow completed")
		return struct{}{}, nil
	})
}

// Actors returns the three printing actors of main.ts keyed by their setup names.
func Actors(w io.Writer, delay time.Duration) map[string]xs.ActorLogic {
	return map[string]xs.ActorLogic{
		"handleApprovedVisaWorkflowID":   Handler(w, "handleApprovedVisaWorkflowID", delay),
		"handleRejectedVisaWorkflowID":   Handler(w, "handleRejectedVisaWorkflowID", delay),
		"handleNoVisaDecisionWorkflowId": Handler(w, "handleNoVisaDecisionWorkflowId", delay),
	}
}

func invoke(src string) []xs.InvokeConfig {
	return []xs.InvokeConfig{{Src: src, OnDone: xs.Transitions{{Target: "End"}}}}
}

// NewMachine mirrors `workflow` with the given actor implementations.
func NewMachine(actors map[string]xs.ActorLogic) *xs.StateMachine[Context] {
	return xs.NewSetup[Context](xs.Implementations{
		Delays: map[string]any{"visaDecisionTimeout": VisaDecisionTimeout},
		Actors: actors,
	}).CreateMachine(xs.MachineConfig[Context]{
		ID:      "eventbasedswitchstate",
		Initial: "CheckVisaStatus",
		States: xs.States{
			{
				Key: "CheckVisaStatus",
				On: map[string]xs.Transitions{
					"visaApprovedEvent": {{Target: "HandleApprovedVisa"}},
					"visaRejectedEvent": {{Target: "HandleRejectedVisa"}},
				},
				After: map[string]xs.Transitions{
					"visaDecisionTimeout": {{Target: "HandleNoVisaDecision"}},
				},
			},
			{Key: "HandleApprovedVisa", Invoke: invoke("handleApprovedVisaWorkflowID")},
			{Key: "HandleRejectedVisa", Invoke: invoke("handleRejectedVisaWorkflowID")},
			{Key: "HandleNoVisaDecision", Invoke: invoke("handleNoVisaDecisionWorkflowId")},
			{Key: "End", Type: xs.Final},
		},
	})
}

// Run mirrors the entry of main.ts with the real 1000 ms handlers, printing to w.
func Run(w io.Writer) {
	RunWith(w, time.Second)
}

// RunWith is Run with the handlers' delay injected.
func RunWith(w io.Writer, delay time.Duration) {
	locked := &lockedWriter{w: w}
	done := make(chan struct{})
	actor := xs.CreateActor(NewMachine(Actors(locked, delay)))
	actor.Subscribe(xs.Observer[*xs.MachineSnapshot[Context]]{
		Complete: func() {
			fmt.Fprintln(locked, "workflow completed", formatOutput(actor.GetSnapshot().Output))
			close(done)
		},
	})
	actor.Start()
	actor.Send(xs.Ev("visaApprovedEvent"))
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
	mu sync.Mutex
	w  io.Writer
}

func (l *lockedWriter) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.w.Write(p)
}
