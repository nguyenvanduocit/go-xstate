// Package monitorjob ports references/xstate/examples/workflow-monitor-job/main.ts
// (serverless workflow "monitor job": submit a job, poll its status every
// 5 seconds, report success or failure).
package monitorjob

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"sync"
	"time"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// Job mirrors the `Job` interface of main.ts.
type Job struct {
	Name string `json:"name"`
}

// Input is the machine input: `{ job }`.
type Input struct {
	Job Job `json:"job"`
}

// Context is the machine's extended state. JobUID and JobStatus are
// `undefined` in JS until assigned; the empty string stands for `undefined`
// and is omitted from the JSON snapshot, like JSON.stringify drops it.
type Context struct {
	Job       Job    `json:"job"`
	JobUID    string `json:"jobuid,omitempty"`
	JobStatus string `json:"jobStatus,omitempty"`
}

// Job statuses of the `jobStatus` context key.
const (
	Succeeded = "SUCCEEDED"
	Failed    = "FAILED"
)

// WaitDelay is the `5000` delay of WaitForCompletion.
const WaitDelay = 5 * time.Second

// NameInput is the input of submitJob and checkJobStatus: `{ name }`.
type NameInput struct {
	Name string `json:"name"`
}

// SubmitOutput is the output of submitJob: `{ jobuid }`.
type SubmitOutput struct {
	JobUID string `json:"jobuid"`
}

// StatusOutput is the output of checkJobStatus: `{ jobStatus }`. An empty
// JobStatus is `undefined`.
type StatusOutput struct {
	JobStatus string `json:"jobStatus"`
}

// Actors are the four promise actors of main.ts.
type Actors struct {
	SubmitJob          xs.ActorLogic
	CheckJobStatus     xs.ActorLogic
	ReportJobSucceeded xs.ActorLogic
	ReportJobFailed    xs.ActorLogic
}

// DefaultActors mirrors the actors of main.ts, logging to w.
func DefaultActors(w io.Writer) Actors {
	return Actors{
		SubmitJob: xs.FromPromise(func(_ context.Context, a xs.PromiseArgs) (SubmitOutput, error) {
			logName(w, "Starting submitJob", a.Input)
			return SubmitOutput{JobUID: "123"}, nil
		}),
		CheckJobStatus: xs.FromPromise(func(_ context.Context, a xs.PromiseArgs) (StatusOutput, error) {
			logName(w, "Starting checkJobStatus", a.Input)
			return StatusOutput{JobStatus: Succeeded}, nil
		}),
		ReportJobSucceeded: xs.FromPromise(func(_ context.Context, a xs.PromiseArgs) (struct{}, error) {
			logName(w, "Starting reportJobSucceeded", a.Input)
			return struct{}{}, nil
		}),
		ReportJobFailed: xs.FromPromise(func(_ context.Context, a xs.PromiseArgs) (struct{}, error) {
			logName(w, "Starting reportJobFailed", a.Input)
			return struct{}{}, nil
		}),
	}
}

func statusIs(want string) xs.Guard {
	return xs.GuardFunc(func(a xs.GuardArgs[Context]) bool {
		return a.Context.JobStatus == want
	})
}

// NewMachine mirrors `workflow` with the given actors.
func NewMachine(actors Actors) *xs.StateMachine[Context] {
	return xs.NewSetup[Context](xs.Implementations{
		Actors: map[string]xs.ActorLogic{
			"submitJob":          actors.SubmitJob,
			"checkJobStatus":     actors.CheckJobStatus,
			"reportJobSucceeded": actors.ReportJobSucceeded,
			"reportJobFailed":    actors.ReportJobFailed,
		},
	}).CreateMachine(xs.MachineConfig[Context]{
		ID:      "jobmonitoring",
		Initial: "SubmitJob",
		ContextFn: func(a xs.ContextArgs) Context {
			return Context{Job: a.Input.(Input).Job}
		},
		States: xs.States{
			{
				Key: "SubmitJob",
				Invoke: []xs.InvokeConfig{{
					Src: "submitJob",
					Input: xs.NewExpr(func(a xs.ExprArgs[Context]) any {
						return NameInput{Name: a.Context.Job.Name}
					}),
					OnDone: xs.Transitions{{
						Target: "WaitForCompletion",
						Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[Context]) Context {
							c := a.Context
							c.JobUID = a.Event.(xs.DoneActorEvent).Output.(SubmitOutput).JobUID
							return c
						})},
					}},
				}},
			},
			{
				Key: "WaitForCompletion",
				After: map[string]xs.Transitions{
					strconv.FormatInt(WaitDelay.Milliseconds(), 10): {{Target: "GetJobStatus"}},
				},
			},
			{
				Key: "GetJobStatus",
				Invoke: []xs.InvokeConfig{{
					Src: "checkJobStatus",
					Input: xs.NewExpr(func(a xs.ExprArgs[Context]) any {
						return NameInput{Name: a.Context.JobUID}
					}),
					OnDone: xs.Transitions{{
						Target: "DetermineCompletion",
						Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[Context]) Context {
							c := a.Context
							c.JobStatus = a.Event.(xs.DoneActorEvent).Output.(StatusOutput).JobStatus
							return c
						})},
					}},
				}},
			},
			{
				Key: "DetermineCompletion",
				Always: xs.Transitions{
					{Guard: statusIs(Succeeded), Target: "JobSucceeded"},
					{Guard: statusIs(Failed), Target: "JobFailed"},
					{Target: "WaitForCompletion"},
				},
			},
			{
				Key: "JobSucceeded",
				Invoke: []xs.InvokeConfig{{
					Src:    "reportJobSucceeded",
					OnDone: xs.Transitions{{Target: "End"}},
				}},
			},
			{
				Key: "JobFailed",
				Invoke: []xs.InvokeConfig{{
					Src:    "reportJobFailed",
					OnDone: xs.Transitions{{Target: "End"}},
				}},
			},
			{Key: "End", Type: xs.Final},
		},
	})
}

// DemoInput is the input of the demo actor at the bottom of main.ts.
var DemoInput = Input{Job: Job{Name: "job1"}}

// Run mirrors the entry of main.ts with the real 5000 ms delay, printing to w.
func Run(w io.Writer) {
	RunWith(w, nil)
}

// RunWith is Run with the actor's clock injected (nil is the real clock).
func RunWith(w io.Writer, clock xs.Clock) {
	locked := &lockedWriter{w: w}
	done := make(chan struct{})
	actor := xs.CreateActor(
		NewMachine(DefaultActors(locked)),
		xs.WithInput(DemoInput),
		xs.WithClock(clock),
	)
	actor.Subscribe(xs.Observer[*xs.MachineSnapshot[Context]]{
		Complete: func() {
			// The machine has no output: `undefined` in the JS console.
			fmt.Fprintln(locked, "workflow completed undefined")
			close(done)
		},
	})
	actor.Start()
	<-done
	actor.Stop()
}

// logName renders console.log(label, input) the way Bun prints it for the
// inputs of this example: `undefined`, or an object `{ name: "<string>" }`
// (one property per line, two-space indent, trailing comma, double-quoted
// string; strings with characters JS escapes differently from Go are not
// handled).
func logName(w io.Writer, label string, input any) {
	in, ok := input.(NameInput)
	if !ok {
		fmt.Fprintf(w, "%s undefined\n", label)
		return
	}
	fmt.Fprintf(w, "%s {\n  name: %s,\n}\n", label, strconv.Quote(in.Name))
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
