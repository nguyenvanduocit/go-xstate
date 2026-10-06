// Package accumulateroomreadings ports references/xstate/examples/workflow-accumulate-room-readings/main.ts:
// the `roomreadings` machine, its `produceReport` actor and the console entry.
package accumulateroomreadings

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"sync"
	"time"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// Context is the machine's extended state: the latest temperature and humidity (null until read).
type Context struct {
	Temperature *float64 `json:"temperature"`
	Humidity    *float64 `json:"humidity"`
}

// ReportInput is the input of produceReport: `{ temperature, humidity }`.
type ReportInput struct {
	Temperature *float64 `json:"temperature"`
	Humidity    *float64 `json:"humidity"`
}

// Timing holds the two durations main.ts hardcodes.
type Timing struct {
	// PT1H is the `PT1H` delay of the machine (10_000 ms in main.ts).
	PT1H time.Duration
	// ReportDelay is the delay(1_000) inside produceReport.
	ReportDelay time.Duration
}

// RealTiming is the timing of main.ts.
var RealTiming = Timing{PT1H: 10 * time.Second, ReportDelay: time.Second}

// scaledTiming is RealTiming with one JS millisecond lasting unit.
func scaledTiming(unit time.Duration) Timing {
	return Timing{PT1H: 10_000 * unit, ReportDelay: 1_000 * unit}
}

// SyncWriter serialises writes so the actor goroutine and the caller can share one io.Writer.
type SyncWriter struct {
	mu sync.Mutex
	W  io.Writer
}

func (s *SyncWriter) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.W.Write(p)
}

func number(v any) *float64 {
	var f float64
	switch n := v.(type) {
	case float64:
		f = n
	case float32:
		f = float64(n)
	case int:
		f = float64(n)
	case int64:
		f = float64(n)
	default:
		panic(fmt.Sprintf("workflowaccumulateroomreadings: reading is not a number: %T", v))
	}
	return &f
}

// NewMachine mirrors `workflow` with the given timing; produceReport prints its two lines to log.
func NewMachine(tm Timing, log io.Writer) *xs.StateMachine[Context] {
	produceReport := xs.FromPromise(func(ctx context.Context, a xs.PromiseArgs) (any, error) {
		fmt.Fprintf(log, "Starting ProduceReport %s\n", formatInput(a.Input.(ReportInput)))
		timer := time.NewTimer(tm.ReportDelay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return nil, context.Cause(ctx)
		case <-timer.C:
		}
		fmt.Fprintln(log, "ProduceReport done")
		return nil, nil
	})
	return xs.NewSetup[Context](xs.Implementations{
		Delays: map[string]any{"PT1H": tm.PT1H},
		Actors: map[string]xs.ActorLogic{"produceReport": produceReport},
	}).CreateMachine(xs.MachineConfig[Context]{
		ID:      "roomreadings",
		Initial: "ConsumeReading",
		Context: Context{},
		States: xs.States{
			{
				Key:   "ConsumeReading",
				Entry: xs.Actions{xs.Assign(func(xs.AssignArgs[Context]) Context { return Context{} })},
				On: map[string]xs.Transitions{
					"TemperatureEvent": {{Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[Context]) Context {
						c := a.Context
						c.Temperature = number(a.Event.(xs.E)["temperature"])
						return c
					})}}},
					"HumidityEvent": {{Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[Context]) Context {
						c := a.Context
						c.Humidity = number(a.Event.(xs.E)["humidity"])
						return c
					})}}},
				},
				After: map[string]xs.Transitions{
					"PT1H": {{
						Guard: xs.GuardFunc(func(a xs.GuardArgs[Context]) bool {
							return a.Context.Temperature != nil && a.Context.Humidity != nil
						}),
						Target: "GenerateReport",
					}},
				},
			},
			{
				Key: "GenerateReport",
				Invoke: []xs.InvokeConfig{{
					Src: "produceReport",
					Input: xs.NewExpr(func(a xs.ExprArgs[Context]) any {
						return ReportInput{Temperature: a.Context.Temperature, Humidity: a.Context.Humidity}
					}),
					OnDone: xs.Transitions{{Target: "ConsumeReading"}},
				}},
			},
		},
	})
}

// Machine mirrors `workflow` with main.ts's timing, logging to stdout.
func Machine() *xs.StateMachine[Context] {
	return NewMachine(RealTiming, os.Stdout)
}

// Run mirrors the entry of main.ts (about 22 s): it prints what the JS process prints.
func Run(w io.Writer) error {
	return RunWith(w, time.Millisecond)
}

// RunWith is Run with one JS millisecond lasting unit.
//
// main.ts sends four readings over 14 s and returns, but its process lives until the PT1H timer of the
// last ConsumeReading fires and the second report completes. Run waits for that second report.
func RunWith(w io.Writer, unit time.Duration) error {
	log := &SyncWriter{W: w}
	actor := xs.CreateActor(NewMachine(scaledTiming(unit), log))
	const reports = 2
	done := make(chan struct{})
	var mu sync.Mutex
	generating, completed := false, 0
	actor.SubscribeNext(func(s *xs.MachineSnapshot[Context]) {
		mu.Lock()
		defer mu.Unlock()
		switch {
		case s.Matches("GenerateReport"):
			generating = true
		case generating:
			generating = false
			if completed++; completed == reports {
				close(done)
			}
		}
	})
	actor.Start()
	defer actor.Stop()

	sleep := func(ms int) { time.Sleep(time.Duration(ms) * unit) }
	actor.Send(xs.E{"type": "TemperatureEvent", "roomId": "kitchen", "temperature": 20})
	sleep(1_000)
	actor.Send(xs.E{"type": "HumidityEvent", "roomId": "kitchen", "humidity": 50})
	sleep(11_000)
	actor.Send(xs.E{"type": "TemperatureEvent", "roomId": "kitchen", "temperature": 10})
	sleep(1_000)
	actor.Send(xs.E{"type": "HumidityEvent", "roomId": "kitchen", "humidity": 30})
	sleep(1_000)

	select {
	case <-done:
		return nil
	case <-time.After(10_000 * unit):
		return fmt.Errorf("second report not completed 10000 JS-ms after the last reading")
	}
}

// formatInput renders the input the way console.log prints it in the JS runtime (bun):
// multi-line object, trailing commas.
func formatInput(in ReportInput) string {
	return fmt.Sprintf("{\n  temperature: %s,\n  humidity: %s,\n}", formatNumber(in.Temperature), formatNumber(in.Humidity))
}

func formatNumber(f *float64) string {
	if f == nil {
		return "null"
	}
	return strconv.FormatFloat(*f, 'g', -1, 64)
}
