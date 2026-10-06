// Package checkinbox ports references/xstate/examples/workflow-check-inbox/main.ts (serverless workflow
// "check inbox periodically"): the `checkInbox` machine, its three actors and the console entry.
package checkinbox

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// Message mirrors `interface Message`.
type Message struct {
	Subject  string `json:"subject"`
	Priority string `json:"priority"` // "high" | "low"
}

// Context is the machine's extended state.
type Context struct {
	Messages []Message `json:"messages"`
}

// ScheduleInput is the input of the schedule actor: `{ interval: number }` (milliseconds).
type ScheduleInput struct {
	Interval int `json:"interval"`
}

// SendTextsInput is the input of sendTextsFunction: `{ messages: Message[] }`.
type SendTextsInput struct {
	Messages []Message `json:"messages"`
}

// SendTextsOutput is the output of sendTextsFunction: `{ status: 'success' }`.
type SendTextsOutput struct {
	Status string `json:"status"`
}

// Timings holds the durations main.ts hardcodes.
type Timings struct {
	// Interval overrides the schedule's input.interval (2000 ms in main.ts); 0 keeps the input.
	Interval time.Duration
	// CheckInbox is the delay(1000) of checkInboxFunction.
	CheckInbox time.Duration
	// SendHigh and SendLow are the delay(100) / delay(500) of sendTextsFunction per priority.
	SendHigh, SendLow time.Duration
}

// RealTimings is the timing of main.ts.
var RealTimings = Timings{CheckInbox: time.Second, SendHigh: 100 * time.Millisecond, SendLow: 500 * time.Millisecond}

// scaledTimings is RealTimings with one JS millisecond lasting unit; the schedule keeps input.interval,
// which NewActors scales through Interval.
func scaledTimings(unit time.Duration) Timings {
	return Timings{Interval: 2000 * unit, CheckInbox: 1000 * unit, SendHigh: 100 * unit, SendLow: 500 * unit}
}

// Actors are the three actor implementations of the machine.
type Actors struct {
	Schedule, CheckInboxFunction, SendTextsFunction xs.ActorLogic
}

// NewActors mirrors the `actors` of setup(): schedule, checkInboxFunction, sendTextsFunction.
// sendTextsFunction prints its two lines per message to log.
func NewActors(t Timings, log io.Writer) Actors {
	return Actors{
		Schedule:           Schedule(t.Interval),
		CheckInboxFunction: CheckInboxFunction(t.CheckInbox),
		SendTextsFunction:  SendTextsFunction(t.SendHigh, t.SendLow, log),
	}
}

// Schedule mirrors the `schedule` callback actor: it sends `reminder` to its parent every
// input.interval ms until stopped. A non-zero override replaces input.interval.
func Schedule(override time.Duration) *xs.CallbackLogic {
	return xs.FromCallback(func(a xs.CallbackArgs) func() {
		interval := override
		if interval == 0 {
			interval = time.Duration(a.Input.(ScheduleInput).Interval) * time.Millisecond
		}
		ticker := time.NewTicker(interval)
		stop := make(chan struct{})
		go func() {
			for {
				select {
				case <-ticker.C:
					a.SendBack(xs.E{"type": "reminder"})
				case <-stop:
					return
				}
			}
		}()
		var once sync.Once
		return func() {
			once.Do(func() {
				ticker.Stop()
				close(stop)
			})
		}
	})
}

// CheckInboxFunction mirrors `checkInboxFunction`: after delay it resolves with the two messages.
func CheckInboxFunction(delay time.Duration) *xs.PromiseLogic[[]Message] {
	return xs.FromPromise(func(ctx context.Context, _ xs.PromiseArgs) ([]Message, error) {
		if err := sleep(ctx, delay); err != nil {
			return nil, err
		}
		return []Message{
			{Subject: "Hello", Priority: "high"},
			{Subject: "Hi", Priority: "low"},
		}, nil
	})
}

// SendTextsFunction mirrors `sendTextsFunction`: for every message concurrently it logs
// "sending text", waits high or low, logs "text sent"; resolves with `{status: 'success'}`.
func SendTextsFunction(high, low time.Duration, log io.Writer) *xs.PromiseLogic[SendTextsOutput] {
	log = &syncWriter{w: log}
	return xs.FromPromise(func(ctx context.Context, a xs.PromiseArgs) (SendTextsOutput, error) {
		in := a.Input.(SendTextsInput)
		var wg sync.WaitGroup
		errs := make([]error, len(in.Messages))
		for i, m := range in.Messages {
			// like Array.map, the first half of every callback runs in message order
			fmt.Fprintln(log, "sending text", m.Subject)
			wg.Add(1)
			go func() {
				defer wg.Done()
				d := low
				if m.Priority == "high" {
					d = high
				}
				if errs[i] = sleep(ctx, d); errs[i] == nil {
					fmt.Fprintln(log, "text sent", m.Subject)
				}
			}()
		}
		wg.Wait()
		for _, err := range errs {
			if err != nil {
				return SendTextsOutput{}, err
			}
		}
		return SendTextsOutput{Status: "success"}, nil
	})
}

// sleep is delay(ms); cancelling ctx stands in for abandoning the promise when the invoking state exits.
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

// NewMachine mirrors `workflow` with the given actors.
func NewMachine(actors Actors) *xs.StateMachine[Context] {
	return xs.NewSetup[Context](xs.Implementations{
		Actors: map[string]xs.ActorLogic{
			"schedule":           actors.Schedule,
			"checkInboxFunction": actors.CheckInboxFunction,
			"sendTextsFunction":  actors.SendTextsFunction,
		},
	}).CreateMachine(xs.MachineConfig[Context]{
		ID:      "checkInbox",
		Initial: "Idle",
		Context: Context{Messages: []Message{}},
		Invoke: []xs.InvokeConfig{{
			Src:   "schedule",
			Input: ScheduleInput{Interval: 2000},
		}},
		States: xs.States{
			{
				Key: "Idle",
				On:  map[string]xs.Transitions{"reminder": {{Target: "CheckInbox"}}},
			},
			{
				Key: "CheckInbox",
				Invoke: []xs.InvokeConfig{{
					Src: "checkInboxFunction",
					OnDone: xs.Transitions{{
						Target: "SendTextForHighPriority",
						Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[Context]) Context {
							c := a.Context
							c.Messages = a.Event.(xs.DoneActorEvent).Output.([]Message)
							return c
						})},
					}},
				}},
			},
			{
				Key: "SendTextForHighPriority",
				Invoke: []xs.InvokeConfig{{
					Src: "sendTextsFunction",
					Input: xs.NewExpr(func(a xs.ExprArgs[Context]) any {
						return SendTextsInput{Messages: a.Context.Messages}
					}),
					OnDone: xs.Transitions{{Target: "Idle"}},
				}},
			},
		},
	})
}

// Run mirrors the entry of main.ts for window: the workflow never completes (the schedule reminds
// forever), so Run returns after window instead of at process exit. It prints what the JS prints.
func Run(w io.Writer, window time.Duration) {
	runWith(w, RealTimings, window)
}

// RunScaled is Run with one JS millisecond lasting unit; window is in JS milliseconds.
func RunScaled(w io.Writer, unit time.Duration, windowMs int) {
	runWith(w, scaledTimings(unit), time.Duration(windowMs)*unit)
}

func runWith(w io.Writer, t Timings, window time.Duration) {
	actor := xs.CreateActor(NewMachine(NewActors(t, w)))
	actor.Start()
	time.Sleep(window)
	actor.Stop()
}

type syncWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func (s *syncWriter) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.w.Write(p)
}
