package creditcheck_test

import (
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/nguyenvanduocit/go-xstate/examples/internal/tracetest"
	api "github.com/nguyenvanduocit/go-xstate/examples/server/creditcheck"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// TestTrace replays testdata/<name>.golden.json, recorded from the JS example by
// scripts/trace/mongodb-credit-check-api/<name>.ts using explicit virtual service advances.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/mongodb-credit-check-api/machine.ts#L14
// JS trace: scripts/trace/mongodb-credit-check-api/bureau-failure.ts.
// JS trace: scripts/trace/mongodb-credit-check-api/existing-reports.ts.
// JS trace: scripts/trace/mongodb-credit-check-api/happy.ts.
// JS trace: scripts/trace/mongodb-credit-check-api/invalid-credentials.ts.
// JS trace: scripts/trace/mongodb-credit-check-api/rate-error.ts.
// JS trace: scripts/trace/mongodb-credit-check-api/reports-table-error.ts.
// JS trace: scripts/trace/mongodb-credit-check-api/score-error.ts.
func TestTrace(t *testing.T) {
	for _, name := range []string{
		"happy",
		"invalid-credentials",
		"bureau-failure",
		"existing-reports",
		"reports-table-error",
		"rate-error",
		"score-error",
	} {
		t.Run(name, func(t *testing.T) {
			tracetest.Run(t, "testdata/"+name+".golden.json", func(clock xs.Clock, input any) *xs.Actor[*xs.MachineSnapshot[api.CreditProfile]] {
				return xs.CreateActor(traceMachine(clock), xs.WithClock(clock), xs.WithUnhandledErrorHandler(func(any) {}))
			})
		})
	}
}

// TestLogs compares the console.log lines of the actions and guards with
// testdata/logs.stdout.txt, recorded by scripts/trace/mongodb-credit-check-api/logs.stdout.ts.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/mongodb-credit-check-api/machine.ts#L14
// JS trace: scripts/trace/mongodb-credit-check-api/logs.stdout.ts.
func TestLogs(t *testing.T) {
	want, err := os.ReadFile("testdata/logs.stdout.txt")
	require.NoError(t, err)

	var mu sync.Mutex
	var lines []string
	actor := xs.CreateActor(api.Machine(stubServices(func(line string) {
		mu.Lock()
		defer mu.Unlock()
		lines = append(lines, line)
	}))).Start()
	defer actor.Stop()
	actor.Send(xs.E{"type": "Submit", "SSN": "123456789", "firstName": "Gavin", "lastName": "Bauman"})

	require.Eventually(t, func() bool {
		return actor.GetSnapshot().Matches("creditCheck.DeterminingInterestRateOptions.RatesProvided")
	}, 5*time.Second, 10*time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, strings.TrimSuffix(string(want), "\n"), strings.Join(lines, "\n"))
}
