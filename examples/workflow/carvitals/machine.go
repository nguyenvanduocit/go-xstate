// Package carvitals ports references/xstate/examples/workflow-car-vitals/main.ts
// (serverless workflow "car vitals checks"): a `checkcarvitals` workflow that
// invokes the `vitalscheck` sub-workflow, which runs four checks in parallel.
package carvitals

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

// Reading is the result of one check: `{ value: 100 }` in main.ts.
type Reading struct {
	Value int `json:"value"`
}

// VitalsContext is the vitalscheck machine's extended state. A reading is
// `null` until its check completes, so each field is a pointer without omitempty.
type VitalsContext struct {
	TirePressure *Reading `json:"tirePressure"`
	OilPressure  *Reading `json:"oilPressure"`
	CoolantLevel *Reading `json:"coolantLevel"`
	Battery      *Reading `json:"battery"`
}

// WorkflowContext is the checkcarvitals machine's extended state: the JS machine
// has no context, which is `{}` in a snapshot.
type WorkflowContext struct{}

// ErrServiceNotAvailable mirrors the `{ type: 'ServiceNotAvailable' }` rejection of delay().
var ErrServiceNotAvailable = errors.New("ServiceNotAvailable")

// Delay mirrors delay(ms, errorProbability) of main.ts: it waits d, then fails
// with ErrServiceNotAvailable with the given probability. Like the JS promise it
// is not cancellable: an actor stopped meanwhile still runs the rest of its body.
func Delay(d time.Duration, errorProbability float64) error {
	time.Sleep(d)
	if rand.Float64() < errorProbability {
		return ErrServiceNotAvailable
	}
	return nil
}

// The four checks of main.ts, in invoke order.
const (
	CheckTirePressure = "checkTirePressure"
	CheckOilPressure  = "checkOilPressure"
	CheckCoolantLevel = "checkCoolantLevel"
	CheckBattery      = "checkBattery"
)

// Checks holds the four check actors of the vitalscheck machine.
type Checks struct {
	TirePressure, OilPressure, CoolantLevel, Battery xs.ActorLogic
}

// NewVitals mirrors `vitalsWorkflow` with the four check actors given.
func NewVitals(c Checks) *xs.StateMachine[VitalsContext] {
	// check mirrors one invoke entry: onDone assigns the output to a context key.
	check := func(src string, set func(*VitalsContext, *Reading)) xs.InvokeConfig {
		return xs.InvokeConfig{
			Src: src,
			OnDone: xs.Transitions{{
				Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[VitalsContext]) VitalsContext {
					r := a.Event.(xs.DoneActorEvent).Output.(Reading)
					ctx := a.Context
					set(&ctx, &r)
					return ctx
				})},
			}},
		}
	}
	return xs.NewSetup[VitalsContext](xs.Implementations{
		Actors: map[string]xs.ActorLogic{
			CheckTirePressure: c.TirePressure,
			CheckOilPressure:  c.OilPressure,
			CheckCoolantLevel: c.CoolantLevel,
			CheckBattery:      c.Battery,
		},
	}).CreateMachine(xs.MachineConfig[VitalsContext]{
		ID:      "vitalscheck",
		Initial: "CheckVitals",
		Context: VitalsContext{},
		States: xs.States{
			{
				Key: "CheckVitals",
				Invoke: []xs.InvokeConfig{
					check(CheckTirePressure, func(c *VitalsContext, r *Reading) { c.TirePressure = r }),
					check(CheckOilPressure, func(c *VitalsContext, r *Reading) { c.OilPressure = r }),
					check(CheckCoolantLevel, func(c *VitalsContext, r *Reading) { c.CoolantLevel = r }),
					check(CheckBattery, func(c *VitalsContext, r *Reading) { c.Battery = r }),
				},
				Always: xs.Transitions{{
					Guard: xs.GuardFunc(func(a xs.GuardArgs[VitalsContext]) bool {
						c := a.Context
						return c.TirePressure != nil && c.OilPressure != nil && c.CoolantLevel != nil && c.Battery != nil
					}),
					Target: "VitalsChecked",
				}},
			},
			{
				Key:  "VitalsChecked",
				Type: xs.Final,
				// JS: `output: ({ context }) => context`. On a state node of the root
				// machine it does not become the machine output, so the done event
				// carries `undefined`; the engine reproduces that.
				Output: xs.NewExpr(func(a xs.ExprArgs[VitalsContext]) any { return a.Context }),
			},
		},
	})
}

// NewWorkflow mirrors `workflow` with the vitalscheck actor given; the
// console.log of the "Done with vitals check" action goes to w.
func NewWorkflow(vitals xs.ActorLogic, w io.Writer) *xs.StateMachine[WorkflowContext] {
	return xs.NewSetup[WorkflowContext](xs.Implementations{
		Actors: map[string]xs.ActorLogic{"vitalscheck": vitals},
	}).CreateMachine(xs.MachineConfig[WorkflowContext]{
		ID:      "checkcarvitals",
		Initial: "WhenCarIsOn",
		States: xs.States{
			{
				Key: "WhenCarIsOn",
				On:  map[string]xs.Transitions{"CarTurnedOnEvent": {{Target: "DoCarVitalChecks"}}},
			},
			{
				Key: "DoCarVitalChecks",
				Invoke: []xs.InvokeConfig{{
					Src: "vitalscheck",
					OnDone: xs.Transitions{{
						Actions: xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[WorkflowContext]) {
							fmt.Fprintln(w, "Done with vitals check", formatOutput(a.Event.(xs.DoneActorEvent).Output))
						})},
						Target: "CheckContinueVitalChecks",
					}},
				}},
			},
			{
				Key:   "CheckContinueVitalChecks",
				After: map[string]xs.Transitions{"1000": {{Target: "DoCarVitalChecks"}}},
			},
		},
		On: map[string]xs.Transitions{
			"CarTurnedOffEvent": {{
				Actions: xs.Actions{xs.Log("Car turned off")},
				Target:  ".WhenCarIsOn",
			}},
		},
	})
}

// Printer is the console of the real checks: it serialises writes from the
// promise goroutines and tracks them so Run can wait for them like the Node
// process waits for pending timers.
type Printer struct {
	mu       sync.Mutex
	w        io.Writer
	inflight sync.WaitGroup

	// startMu/startCond/started order the "Starting" lines: the JS runs the
	// synchronous head of each promise body in invoke order, while a Go promise
	// body runs on its own goroutine, so goroutine scheduling alone does not
	// give that order.
	startMu   sync.Mutex
	startCond *sync.Cond
	started   int
}

// NewPrinter returns a Printer writing to w.
func NewPrinter(w io.Writer) *Printer {
	p := &Printer{w: w}
	p.startCond = sync.NewCond(&p.startMu)
	return p
}

// printStarting prints "Starting <name>" once it is the turn of check number
// index (0-based invoke order) among perRound checks, which are all started in
// every round of the vitalscheck machine.
func (p *Printer) printStarting(index, perRound int, name string) {
	p.startMu.Lock()
	defer p.startMu.Unlock()
	for p.started%perRound != index {
		p.startCond.Wait()
	}
	fmt.Fprintln(p, "Starting", name)
	p.started++
	p.startCond.Broadcast()
}

// Write implements io.Writer.
func (p *Printer) Write(b []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.w.Write(b)
}

// RealCheck mirrors one `checkX: fromPromise(async () => {...})` actor of
// main.ts: print "Starting <name>", wait delay, print "Completed <name>", resolve
// `{ value: 100 }`. The wait is not cancellable, as in JS. index is the position
// of the check in the machine's invoke list.
func (p *Printer) RealCheck(index int, name string, delay time.Duration) xs.ActorLogic {
	return xs.FromPromise(func(_ context.Context, _ xs.PromiseArgs) (Reading, error) {
		p.inflight.Add(1)
		defer p.inflight.Done()
		p.printStarting(index, realChecksPerRound, name)
		if err := Delay(delay, 0); err != nil {
			return Reading{}, err
		}
		fmt.Fprintln(p, "Completed", name)
		return Reading{Value: 100}, nil
	})
}

const realChecksPerRound = 4

// RealChecks returns the four actors of main.ts with delays 1000/1500/500/1200 ms
// (each divided by speedup).
func (p *Printer) RealChecks(speedup int) Checks {
	d := func(ms int) time.Duration { return time.Duration(ms) * time.Millisecond / time.Duration(speedup) }
	return Checks{
		TirePressure: p.RealCheck(0, CheckTirePressure, d(1000)),
		OilPressure:  p.RealCheck(1, CheckOilPressure, d(1500)),
		CoolantLevel: p.RealCheck(2, CheckCoolantLevel, d(500)),
		Battery:      p.RealCheck(3, CheckBattery, d(1200)),
	}
}

// scaledClock runs machine timers (`after`) speedup times faster than real time.
type scaledClock struct{ speedup time.Duration }

func (c scaledClock) SetTimeout(fn func(), d time.Duration) xs.TimerID {
	return time.AfterFunc(d/c.speedup, fn)
}

func (scaledClock) ClearTimeout(id xs.TimerID) { id.(*time.Timer).Stop() }

// Run mirrors the entry of main.ts in real time (about 7.5 s), printing to w.
func Run(w io.Writer) { RunWith(w, 1) }

// RunWith is Run with every duration divided by speedup; the printed text does
// not depend on it.
func RunWith(w io.Writer, speedup int) {
	out := NewPrinter(w)
	ms := func(n int) time.Duration { return time.Duration(n) * time.Millisecond / time.Duration(speedup) }
	actor := xs.CreateActor(
		NewWorkflow(NewVitals(out.RealChecks(speedup)), out),
		xs.WithClock(scaledClock{speedup: time.Duration(speedup)}),
		xs.WithLogger(func(args ...any) { fmt.Fprintln(out, args...) }),
	)
	// JS: the `complete` subscriber logs the output; this machine never reaches a final state.
	actor.Subscribe(xs.Observer[*xs.MachineSnapshot[WorkflowContext]]{
		Complete: func() {
			if snap := actor.GetSnapshot(); snap.Status == xs.StatusDone {
				fmt.Fprintln(out, "workflow completed", formatOutput(snap.Output))
			}
		},
	})
	actor.Start()
	time.Sleep(ms(1000))
	actor.Send(xs.Ev("CarTurnedOnEvent"))
	time.Sleep(ms(6000))
	actor.Send(xs.Ev("CarTurnedOffEvent"))
	// Node exits once no timer is pending; abandoned checks still print "Completed".
	out.inflight.Wait()
	actor.Stop()
}

// formatOutput renders an event output the way console.log prints it in the JS
// runtime for the only value this app produces: `undefined`.
func formatOutput(out any) string {
	if out == nil {
		return "undefined"
	}
	return fmt.Sprint(out)
}
