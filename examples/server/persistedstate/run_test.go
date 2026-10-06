package persistedstate_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	donut "github.com/nguyenvanduocit/go-xstate/examples/server/persistedstate"
)

func runSession(store donut.Store, events ...string) string {
	var out bytes.Buffer
	donut.Run(context.Background(), &out, strings.NewReader(strings.Join(events, "\n")+"\n"), store)
	return out.String()
}

// TestRunMatchesJS replays the three sessions of
// scripts/trace/mongodb-persisted-state/mongodb-persisted-state.stdout.ts (the real
// main.ts running against a fake `mongodb` module) and compares the whole output.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/mongodb-persisted-state/main.ts#L26
// JS trace: scripts/trace/mongodb-persisted-state/mongodb-persisted-state.stdout.ts.
func TestRunMatchesJS(t *testing.T) {
	want, err := os.ReadFile("testdata/mongodb-persisted-state.stdout.txt")
	require.NoError(t, err)

	store := &memStore{}
	var got strings.Builder
	session := func(n int, store donut.Store, events ...string) {
		fmt.Fprintf(&got, "--- session %d ---\n", n)
		got.WriteString(runSession(store, events...))
	}
	session(1, store, "NEXT", "BOGUS", "NEXT", "MIXED_DRY", "")
	session(2, store, "MIXED_WET", "NEXT", "NEXT", "NEXT", "NEXT", "ANOTHER_DONUT")
	session(3, &memStore{connectErr: errConnect})

	require.Equal(t, string(want), got.String())
}

// The cases below have no JS counterpart (the fake mongodb of the JS script never fails
// after connecting); they pin the Go-side error handling.

// Go regression; expected values are Go assertions, not a recorded JS test.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/mongodb-persisted-state/main.ts#L26
// Related JS trace: scripts/trace/mongodb-persisted-state/mongodb-persisted-state.stdout.ts.
func TestRunFindOneError(t *testing.T) {
	out := runSession(&memStore{findErr: errConnect})
	require.Equal(t, "error details:  Error: connect ECONNREFUSED\n", out)
}

// Go regression; expected values are Go assertions, not a recorded JS test.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/mongodb-persisted-state/main.ts#L26
// Related JS trace: scripts/trace/mongodb-persisted-state/mongodb-persisted-state.stdout.ts.
func TestRunSaveErrorIsPrintedAndRunContinues(t *testing.T) {
	store := &memStore{updateErr: errConnect}
	out := runSession(store, "NEXT")
	require.Equal(t, 2, store.updateCalls) // initial snapshot + NEXT
	require.Equal(t, 2, strings.Count(out, "error details:  Error: connect ECONNREFUSED\n"))
	require.NotContains(t, out, "Current state")
}

// Go regression; expected values are Go assertions, not a recorded JS test.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/mongodb-persisted-state/main.ts#L26
// Related JS trace: scripts/trace/mongodb-persisted-state/mongodb-persisted-state.stdout.ts.
func TestRunRestoresPersistedStateFromStore(t *testing.T) {
	store := &memStore{}
	runSession(store, "NEXT", "NEXT")
	out := runSession(store)
	require.Contains(t, out, `"restored state:  {"_id":"doc-1"`[1:])
	require.Contains(t, out, "\x1b[1m{\"directions\":{\"mix\":")
	require.NotContains(t, out, "persisted state saved") // identical snapshot: modifiedCount 0
}

// Go regression; expected values are Go assertions, not a recorded JS test.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/mongodb-persisted-state/TaskQueue.ts#L1
// Related JS trace: scripts/trace/mongodb-persisted-state/mongodb-persisted-state.stdout.ts.
func TestTaskQueueRunsTasksInOrderWithoutReentry(t *testing.T) {
	q := &donut.TaskQueue{}
	var order []string
	q.AddTask(func() {
		order = append(order, "a:start")
		q.AddTask(func() { order = append(order, "b") })
		order = append(order, "a:end")
	})
	q.AddTask(func() { order = append(order, "c") })
	require.Equal(t, []string{"a:start", "a:end", "b", "c"}, order)
}
