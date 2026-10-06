// Package eventbasedservice ports references/xstate/examples/workflow-event-basedservice/main.ts
// (serverless workflow "event-based service invocation").
package eventbasedservice

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// PatientInfo mirrors the JS interface of the same name.
type PatientInfo struct {
	Name   string `json:"name"`
	Pet    string `json:"pet"`
	Reason string `json:"reason"`
}

// AppointmentInfo is `{ appointmentId, appointmentDate }` built by MakeAppointmentAction.
type AppointmentInfo struct {
	AppointmentID   string `json:"appointmentId"`
	AppointmentDate string `json:"appointmentDate"`
}

// AppointmentResult is the output of MakeAppointmentAction: `{ appointmentInfo }`.
// The JS machine assigns this whole object to context.appointmentInfo, so the
// context holds `{appointmentInfo: {appointmentId, appointmentDate}}`.
type AppointmentResult struct {
	AppointmentInfo AppointmentInfo `json:"appointmentInfo"`
}

// Context mirrors the JS context; null fields are nil pointers.
type Context struct {
	PatientInfo     *PatientInfo       `json:"patientInfo"`
	AppointmentInfo *AppointmentResult `json:"appointmentInfo"`
}

// ActionInput is the input of MakeAppointmentAction: `{ patientInfo }`.
type ActionInput struct {
	PatientInfo *PatientInfo
}

// DateLayout is Date.prototype.toISOString.
const DateLayout = "2006-01-02T15:04:05.000Z"

// MakeAppointmentAction mirrors the actor of main.ts: it logs the request to w, waits
// delay (2000 ms in the JS), builds the appointment stamped with now() and logs it.
// Cancelling ctx stands in for abandoning the promise when the invoking state exits.
func MakeAppointmentAction(w io.Writer, delay time.Duration, now func() time.Time) xs.ActorLogic {
	return xs.FromPromise(func(ctx context.Context, a xs.PromiseArgs) (AppointmentResult, error) {
		in := a.Input.(ActionInput)
		fmt.Fprintf(w, "Making vet appointment for {\n  name: %q,\n  pet: %q,\n  reason: %q,\n}\n",
			in.PatientInfo.Name, in.PatientInfo.Pet, in.PatientInfo.Reason)
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return AppointmentResult{}, context.Cause(ctx)
		case <-timer.C:
		}
		info := AppointmentInfo{AppointmentID: "1234", AppointmentDate: now().UTC().Format(DateLayout)}
		fmt.Fprintf(w, "Vet appointment made {\n  appointmentId: %q,\n  appointmentDate: %q,\n}\n",
			info.AppointmentID, info.AppointmentDate)
		return AppointmentResult{AppointmentInfo: info}, nil
	})
}

// patientInfoOf reads `event.patientInfo` of a MakeVetAppointment event.
func patientInfoOf(event xs.Event) *PatientInfo {
	m := event.(xs.E)["patientInfo"].(map[string]any)
	name, _ := m["name"].(string)
	pet, _ := m["pet"].(string)
	reason, _ := m["reason"].(string)
	return &PatientInfo{Name: name, Pet: pet, Reason: reason}
}

// NewMachine mirrors `workflow` with the given MakeAppointmentAction implementation.
func NewMachine(makeAppointment xs.ActorLogic) *xs.StateMachine[Context] {
	return xs.NewSetup[Context](xs.Implementations{
		Actors: map[string]xs.ActorLogic{"MakeAppointmentAction": makeAppointment},
	}).CreateMachine(xs.MachineConfig[Context]{
		ID:      "VetAppointmentWorkflow",
		Initial: "Idle",
		Context: Context{},
		States: xs.States{
			{
				Key: "Idle",
				On: map[string]xs.Transitions{
					"MakeVetAppointment": {{
						Target: "MakeVetAppointmentState",
						Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[Context]) Context {
							c := a.Context
							c.PatientInfo = patientInfoOf(a.Event)
							return c
						})},
					}},
				},
			},
			{
				Key: "MakeVetAppointmentState",
				Invoke: []xs.InvokeConfig{{
					Src: "MakeAppointmentAction",
					Input: xs.NewExpr(func(a xs.ExprArgs[Context]) any {
						return ActionInput{PatientInfo: a.Context.PatientInfo}
					}),
					OnDone: xs.Transitions{{
						Target: "Idle",
						Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[Context]) Context {
							c := a.Context
							out := a.Event.(xs.DoneActorEvent).Output.(AppointmentResult)
							c.AppointmentInfo = &out
							return c
						})},
					}},
				}},
			},
		},
	})
}

// RequestEvent is the event main.ts sends: Jenny's annual checkup for Ato.
func RequestEvent() xs.E {
	return xs.E{
		"type": "MakeVetAppointment",
		"patientInfo": map[string]any{
			"name":   "Jenny",
			"pet":    "Ato",
			"reason": "Annual checkup",
		},
	}
}

// Run mirrors the entry of main.ts with the real 2000 ms action, printing to w.
func Run(w io.Writer) {
	RunWith(w, 2*time.Second, time.Now)
}

// RunWith is Run with the action's delay and clock injected. The JS process exits when
// no timer is left; RunWith returns once the machine is back in Idle with the appointment
// assigned, then stops the actor. The machine has no final state, so the JS
// `complete` observer never prints and the Go one is wired the same way.
func RunWith(w io.Writer, delay time.Duration, now func() time.Time) {
	locked := &lockedWriter{w: w}
	done := make(chan struct{})
	var once sync.Once
	actor := xs.CreateActor(NewMachine(MakeAppointmentAction(locked, delay, now)), xs.WithInput(map[string]any{
		"person": map[string]any{"name": "Jenny"},
	}))
	sub := actor.Subscribe(xs.Observer[*xs.MachineSnapshot[Context]]{
		Next: func(s *xs.MachineSnapshot[Context]) {
			if s.Context.AppointmentInfo != nil {
				once.Do(func() { close(done) })
			}
		},
		Complete: func() {
			fmt.Fprintln(locked, "workflow completed", "undefined")
		},
	})
	actor.Start()
	actor.Send(RequestEvent())
	<-done
	// Go-only teardown: Stop completes the actor (JS does too), which would print the
	// `complete` line the JS entry never reaches, so the observer is dropped first.
	sub.Unsubscribe()
	actor.Stop()
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
