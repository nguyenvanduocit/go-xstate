package eventbasedservice_test

import (
	"context"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nguyenvanduocit/go-xstate/examples/internal/tracetest"
	wf "github.com/nguyenvanduocit/go-xstate/examples/workflow/eventbasedservice"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// fixedDate is the instant the JS scripts pin Date to
// (scripts/trace/workflow-event-based-service/lib/load.ts and the .stdout.ts script).
var fixedDate = time.Date(2026, 1, 2, 3, 4, 5, 678_000_000, time.UTC)

func fixedNow() time.Time { return fixedDate }

// appointmentStub is the same deterministic stub as in the JS script: resolves after 100 ms
// with an appointmentId that echoes the patient name.
func appointmentStub() xs.ActorLogic {
	return xs.FromPromise(func(ctx context.Context, a xs.PromiseArgs) (wf.AppointmentResult, error) {
		in := a.Input.(wf.ActionInput)
		select {
		case <-time.After(100 * time.Millisecond):
			return wf.AppointmentResult{AppointmentInfo: wf.AppointmentInfo{
				AppointmentID:   "1234:" + in.PatientInfo.Name,
				AppointmentDate: fixedDate.Format(wf.DateLayout),
			}}, nil
		case <-ctx.Done():
			return wf.AppointmentResult{}, context.Cause(ctx)
		}
	})
}

// TestTrace replays testdata/appointment.golden.json, recorded from the JS example by
// scripts/trace/workflow-event-based-service/appointment.ts.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-event-based-service/main.ts#L18
// JS trace: scripts/trace/workflow-event-based-service/appointment.ts.
func TestTrace(t *testing.T) {
	tracetest.Run(t, "testdata/appointment.golden.json", func(clock xs.Clock, _ any) *xs.Actor[*xs.MachineSnapshot[wf.Context]] {
		return xs.CreateActor(wf.NewMachine(appointmentStub()), xs.WithClock(clock))
	})
}

// TestStdout compares the printing entry with testdata/workflow-event-based-service.stdout.txt,
// recorded from main.ts. The delay is shortened; the text does not depend on it.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-event-based-service/main.ts#L89
// JS trace: scripts/trace/workflow-event-based-service/workflow-event-based-service.stdout.ts.
func TestStdout(t *testing.T) {
	want, err := os.ReadFile("testdata/workflow-event-based-service.stdout.txt")
	require.NoError(t, err)
	var out strings.Builder
	wf.RunWith(&out, 10*time.Millisecond, fixedNow)
	assert.Equal(t, string(want), out.String())
}

// syncBuffer is a strings.Builder safe for the promise goroutine and the test.
type syncBuffer struct {
	mu sync.Mutex
	sb strings.Builder
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.sb.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.sb.String()
}

// The real action takes its date from the injected clock and formats it like toISOString.
// Go regression; expected values are Go assertions, not a recorded JS test.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-event-based-service/main.ts#L18
// Related JS trace: scripts/trace/workflow-event-based-service/appointment.ts.
func TestMakeAppointmentAction_DateFormat(t *testing.T) {
	var out syncBuffer
	wf.RunWith(&out, time.Millisecond, func() time.Time {
		return time.Date(2030, 12, 31, 23, 59, 58, 7_000_000, time.FixedZone("x", 3600))
	})
	assert.Contains(t, out.String(), `appointmentDate: "2030-12-31T22:59:58.007Z"`)
}

// Stopping the actor while the action is pending abandons it: "Vet appointment made" is never printed.
// Go regression; expected values are Go assertions, not a recorded JS test.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-event-based-service/main.ts#L18
// Related JS trace: scripts/trace/workflow-event-based-service/appointment.ts.
func TestMakeAppointmentAction_StoppedActorAbandonsPromise(t *testing.T) {
	var out syncBuffer
	actor := xs.CreateActor(wf.NewMachine(wf.MakeAppointmentAction(&out, 50*time.Millisecond, fixedNow)))
	actor.Start()
	actor.Send(wf.RequestEvent())
	actor.Stop()
	time.Sleep(150 * time.Millisecond)
	assert.Equal(t, "Making vet appointment for {\n  name: \"Jenny\",\n  pet: \"Ato\",\n  reason: \"Annual checkup\",\n}\n", out.String())
}
