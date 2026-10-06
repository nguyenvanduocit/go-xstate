package newpatientonboarding

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/nguyenvanduocit/go-xstate/examples/internal/tracetest"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/require"
)

// JS reference (upstream has no separate test): https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-new-patient-onboarding/main.ts#L29
// Expected behavior recorder: scripts/trace/workflow-new-patient-onboarding/lib/record.ts.
func TestTraces(t *testing.T) {
	for _, failure := range []string{"success", "StoreNewPatientInfo", "AssignDoctor", "ScheduleAppt"} {
		t.Run(failure, func(t *testing.T) {
			actors := map[string]xs.ActorLogic{}
			for _, name := range []string{"StoreNewPatientInfo", "AssignDoctor", "ScheduleAppt"} {
				actors[name] = xs.FromPromise(func(ctx context.Context, _ xs.PromiseArgs) (any, error) {
					if err := Sleep(ctx, 200*time.Millisecond); err != nil {
						return nil, err
					}
					if name == failure {
						return nil, errors.New("service failed")
					}
					return nil, nil
				})
			}
			tracetest.Run(t, "testdata/"+failure+".golden.json", func(xs.Clock, any) *xs.Actor[*xs.MachineSnapshot[Context]] { return xs.CreateActor(NewMachine(actors)) })
		})
	}
}

// JS reference (upstream has no separate test): https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-new-patient-onboarding/main.ts#L4
// Expected behavior recorder: scripts/trace/workflow-new-patient-onboarding/retry.stdout.ts.
// Go-only retry helper regression, checking the upstream policy directly.
func TestRetry(t *testing.T) {
	for _, tc := range []struct {
		name     string
		failures int
		err      error
		calls    int
		retries  int
	}{
		{"success", 0, nil, 1, 0}, {"transient", 2, ServiceError{"ServiceNotAvailable"}, 3, 2},
		{"exhausted", 20, ServiceError{"ServiceNotAvailable"}, 11, 10}, {"unhandled", 1, errors.New("other"), 1, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls, retries, sleeps := 0, 0, 0
			err := Retry(context.Background(), func() error {
				calls++
				if calls <= tc.failures {
					return tc.err
				}
				return nil
			}, func(_ context.Context, d time.Duration) error {
				require.Equal(t, 3*time.Second, d)
				sleeps++
				return nil
			}, func(attempt int, _ error) { retries++; require.Equal(t, retries, attempt) })
			require.Equal(t, tc.calls, calls)
			require.Equal(t, tc.retries, retries)
			require.Equal(t, retries, sleeps)
			if tc.failures >= tc.calls {
				require.Equal(t, tc.err, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// JS reference (upstream has no separate test): https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-new-patient-onboarding/main.ts#L134
// Expected behavior recorder: scripts/trace/workflow-new-patient-onboarding/workflow-new-patient-onboarding.stdout.ts.
func TestStdout(t *testing.T) {
	want, err := os.ReadFile("testdata/workflow-new-patient-onboarding.stdout.txt")
	require.NoError(t, err)
	var out strings.Builder
	require.NoError(t, RunWith(context.Background(), &out, func(_ context.Context, d time.Duration, p float64) error {
		require.Equal(t, time.Second, d)
		require.Equal(t, 0.5, p)
		return nil
	}, func(context.Context, time.Duration) error { return nil }))
	require.Equal(t, string(want), out.String())
}

// JS reference (upstream has no separate test): https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-new-patient-onboarding/main.ts#L4
// Expected behavior recorder: scripts/trace/workflow-new-patient-onboarding/exhausted.stdout.ts.
func TestRetryStdout(t *testing.T) {
	for _, name := range []string{"retry", "exhausted"} {
		t.Run(name, func(t *testing.T) {
			want, err := os.ReadFile("testdata/" + name + ".stdout.txt")
			require.NoError(t, err)
			var out strings.Builder
			calls := 0
			require.NoError(t, RunWith(context.Background(), &out, func(context.Context, time.Duration, float64) error {
				calls++
				if name == "exhausted" || calls == 1 {
					return ServiceError{"ServiceNotAvailable"}
				}
				return nil
			}, func(context.Context, time.Duration) error { return nil }))
			require.Equal(t, string(want), out.String())
			if name == "exhausted" {
				require.Equal(t, 11, calls)
			} else {
				require.Equal(t, 4, calls)
			}
		})
	}
}
