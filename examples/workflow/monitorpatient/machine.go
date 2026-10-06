// Package monitorpatient ports references/xstate/examples/workflow-monitor-patient/main.ts
// (serverless workflow "monitor patient vital signs"): a single-state machine
// that reacts to three vital-sign events by running an action for the patient.
package monitorpatient

import (
	"context"
	"fmt"
	"io"
	"math"
	"math/rand/v2"
	"time"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// The three event types of main.ts, in the order of the JS entry's random pick.
const (
	HighBodyTemp        = "org.monitor.highBodyTemp"
	HighBloodPressure   = "org.monitor.highBloodPressure"
	HighRespirationRate = "org.monitor.highRespirationRate"
)

// Constants of the JS entry (main.ts's setInterval callback and createActor input).
const (
	monitoringSource  = "monitoringSource"
	entryTickInterval = 3 * time.Second
	entryEventID      = "event1"
	entryValue        = "value1"
	entryPatientID    = "patient1"
	jsISOTimeLayout   = "2006-01-02T15:04:05.000Z" // Date.prototype.toISOString
)

var eventTypes = [...]string{HighBodyTemp, HighBloodPressure, HighRespirationRate}

// Input is the machine input: `{ patientId }`.
type Input struct {
	PatientID string `json:"patientId"`
}

// Context is the machine's extended state: `{ patientId }`.
type Context struct {
	PatientID string `json:"patientId"`
}

// VitalEvent builds one event of main.ts's `events` union:
// `{ type, source: 'monitoringSource', id, time, patientId, data: { value } }`.
func VitalEvent(eventType, id, time, patientID, value string) xs.E {
	return xs.E{
		"type":      eventType,
		"source":    monitoringSource,
		"id":        id,
		"time":      time,
		"patientId": patientID,
		"data":      map[string]any{"value": value},
	}
}

// order builds the action that mirrors one of sendTylenolOrder, callNurse and
// callPulmonologist: `console.log('Executing <name> for patient:', context.patientId)`.
func order(w io.Writer, name string) xs.Action {
	return xs.ActionFunc(func(a xs.ActionArgs[Context]) {
		fmt.Fprintln(w, "Executing", name, "for patient:", a.Context.PatientID)
	})
}

// NewMachine mirrors `workflow`, with the actions' console.log output going to w.
func NewMachine(w io.Writer) *xs.StateMachine[Context] {
	return xs.NewSetup[Context](xs.Implementations{
		Actions: map[string]xs.Action{
			"sendTylenolOrder":  order(w, "sendTylenolOrder"),
			"callNurse":         order(w, "callNurse"),
			"callPulmonologist": order(w, "callPulmonologist"),
		},
	}).CreateMachine(xs.MachineConfig[Context]{
		ID:      "patientVitalsWorkflow",
		Initial: "MonitorVitals",
		ContextFn: func(a xs.ContextArgs) Context {
			return Context{PatientID: a.Input.(Input).PatientID}
		},
		States: xs.States{
			{
				Key: "MonitorVitals",
				On: map[string]xs.Transitions{
					HighBodyTemp:        {{Actions: xs.Actions{xs.ActionRef{Type: "sendTylenolOrder"}}}},
					HighBloodPressure:   {{Actions: xs.Actions{xs.ActionRef{Type: "callNurse"}}}},
					HighRespirationRate: {{Actions: xs.Actions{xs.ActionRef{Type: "callPulmonologist"}}}},
				},
			},
		},
	})
}

// Run mirrors the entry of main.ts: actor for `patient1`, one random vital-sign
// event every 3 s, actions printing to w, until ctx is cancelled (the Node
// process never exits on its own).
func Run(ctx context.Context, w io.Writer) {
	ticker := time.NewTicker(entryTickInterval)
	defer ticker.Stop()
	RunWith(ctx, w, ticker.C, rand.Float64)
}

// RunWith is Run with the interval ticks and Math.random injected. It returns
// when ctx is cancelled or ticks is closed. Each tick sends the event type
// picked by `Math.floor(random() * 3)` with the current time.
func RunWith(ctx context.Context, w io.Writer, ticks <-chan time.Time, random func() float64) {
	actor := xs.CreateActor(NewMachine(w), xs.WithInput(Input{PatientID: entryPatientID}))
	actor.Start()
	defer actor.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now, ok := <-ticks:
			if !ok {
				return
			}
			eventType := eventTypes[int(math.Floor(random()*float64(len(eventTypes))))]
			actor.Send(VitalEvent(eventType, entryEventID, now.UTC().Format(jsISOTimeLayout), entryPatientID, entryValue))
		}
	}
}
