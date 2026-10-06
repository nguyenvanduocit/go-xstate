package creditcheck_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	api "github.com/nguyenvanduocit/go-xstate/examples/server/creditcheck"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

func newTestServer(t *testing.T, store api.Store, newID func() string) *httptest.Server {
	t.Helper()
	srv := api.NewServer(api.Machine(stubServices(nil)), store, newID)
	srv.Logf = func(string, ...any) {}
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(func() {
		ts.Close()
		srv.Close()
	})
	return ts
}

func do(t *testing.T, ts *httptest.Server, method, path, body string) (int, string) {
	t.Helper()
	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, ts.URL+path, r)
	require.NoError(t, err)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := ts.Client().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	got, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp.StatusCode, string(got)
}

// TestServerTrace replays testdata/server.golden.json, recorded by running the real
// handlers of index.ts (with actorService.ts and machine.ts) behind a fake router and an
// in-memory mongodb (scripts/trace/mongodb-credit-check-api/server.ts). Workflow ids
// are random in JS; the ids the JS run returned are fed to the Go server in order.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/mongodb-credit-check-api/index.ts#L19
// JS trace: scripts/trace/mongodb-credit-check-api/server.ts.
func TestServerTrace(t *testing.T) {
	raw, err := os.ReadFile("testdata/server.golden.json")
	require.NoError(t, err)
	var g struct {
		Steps []struct {
			Wait    *float64 `json:"wait"`
			Request struct {
				Method string          `json:"method"`
				Path   string          `json:"path"`
				Body   json.RawMessage `json:"body"`
			} `json:"request"`
			Response struct {
				Status int            `json:"status"`
				JSON   map[string]any `json:"json"`
				Text   *string        `json:"text"`
			} `json:"response"`
		} `json:"steps"`
	}
	require.NoError(t, json.Unmarshal(raw, &g))
	require.NotEmpty(t, g.Steps)

	var ids []string
	for _, s := range g.Steps {
		if s.Request.Method == "POST" && s.Request.Path == "/workflows" {
			ids = append(ids, s.Response.JSON["workflowId"].(string))
		}
	}
	next := 0
	ts := newTestServer(t, api.NewMemoryStore(), func() string {
		id := ids[next]
		next++
		return id
	})

	for i, s := range g.Steps {
		if s.Wait != nil {
			time.Sleep(time.Duration(*s.Wait * float64(time.Millisecond)))
			continue
		}
		status, got := do(t, ts, s.Request.Method, s.Request.Path, string(s.Request.Body))
		require.Equal(t, s.Response.Status, status, "step %d %s %s", i, s.Request.Method, s.Request.Path)
		if s.Response.Text != nil {
			require.Equal(t, *s.Response.Text, got, "step %d %s %s", i, s.Request.Method, s.Request.Path)
			continue
		}
		var gotJSON map[string]any
		require.NoError(t, json.Unmarshal([]byte(got), &gotJSON), "step %d body %q", i, got)
		require.Equal(t, s.Response.JSON, gotJSON, "step %d %s %s", i, s.Request.Method, s.Request.Path)
	}
}

// failingStore fails every machine state write, like an unacknowledged replaceOne.
type failingStore struct{ api.Store }

func (failingStore) SaveMachineState(context.Context, api.MachineState) error {
	return errors.New("db down")
}

// Go regression; expected values are Go assertions, not a recorded JS test.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/mongodb-credit-check-api/services/actorService.ts#L33
// Related JS trace: scripts/trace/mongodb-credit-check-api/server.ts.
func TestStartFailsWhenStoreFindFails(t *testing.T) {
	ts := newTestServer(t, findFailingStore{api.NewMemoryStore()}, nil)
	status, body := do(t, ts, "POST", "/workflows/abc", `{"type":"Submit"}`)
	require.Equal(t, 500, status)
	require.Equal(t, "Error sending event. Details: Error: find failed", body)
	status, body = do(t, ts, "GET", "/workflows/abc", "")
	require.Equal(t, 500, status)
	require.Equal(t, "Internal Server Error", body)
}

type findFailingStore struct{ api.Store }

func (findFailingStore) FindMachineState(context.Context, string) (*api.MachineState, error) {
	return nil, errors.New("find failed")
}

// A failing write is logged by the subscription; the request itself still succeeds, as in JS
// where the rejected persist is an unhandled rejection of the detached next handler.
// Go regression; expected values are Go assertions, not a recorded JS test.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/mongodb-credit-check-api/services/actorService.ts#L33
// Related JS trace: scripts/trace/mongodb-credit-check-api/server.ts.
func TestPersistFailureIsLoggedNotReturned(t *testing.T) {
	var logged []string
	srv := api.NewServer(api.Machine(stubServices(nil)), failingStore{api.NewMemoryStore()}, func() string { return "w1" })
	srv.Logf = func(format string, args ...any) { logged = append(logged, format) }
	ts := httptest.NewServer(srv.Handler())
	defer func() { ts.Close(); srv.Close() }()

	status, _ := do(t, ts, "POST", "/workflows", "")
	require.Equal(t, 201, status)
	require.Contains(t, logged, "%v")
}

// Go regression; expected values are Go assertions, not a recorded JS test.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/mongodb-credit-check-api/index.ts#L19
// Related JS trace: scripts/trace/mongodb-credit-check-api/server.ts.
func TestBadEvents(t *testing.T) {
	ts := newTestServer(t, api.NewMemoryStore(), func() string { return "w1" })
	do(t, ts, "POST", "/workflows", "")
	for _, body := range []string{``, `{`, `{}`, `{"type":1}`, `[]`} {
		status, text := do(t, ts, "POST", "/workflows/w1", body)
		require.Equal(t, 400, status, "body %q", body)
		require.Equal(t, "Bad Request", text)
	}
}

// TestRun checks the listening message of app.listen; the port is the listener's.
// Go regression; expected values are Go assertions, not a recorded JS test.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/mongodb-credit-check-api/index.ts#L19
// Related JS trace: scripts/trace/mongodb-credit-check-api/server.ts.
func TestRun(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	var out strings.Builder
	done := make(chan error, 1)
	go func() { done <- api.Run(&out, ln, http.NewServeMux()) }()
	require.NoError(t, ln.Close())
	require.Error(t, <-done)
	_, port, _ := net.SplitHostPort(ln.Addr().String())
	require.Equal(t, "Server listening on port "+port+"\n", out.String())
}

// The server persists a restored workflow with the machine's own snapshot shape.
// Go regression; expected values are Go assertions, not a recorded JS test.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/mongodb-credit-check-api/services/actorService.ts#L33
// Related JS trace: scripts/trace/mongodb-credit-check-api/server.ts.
func TestPersistedStateRestoresIntoMachine(t *testing.T) {
	store := api.NewMemoryStore()
	ts := newTestServer(t, store, func() string { return "w1" })
	do(t, ts, "POST", "/workflows", "")
	state, err := store.FindMachineState(context.Background(), "w1")
	require.NoError(t, err)
	actor := xs.CreateActor(api.Machine(stubServices(nil)), xs.WithSnapshot(state.PersistedState)).Start()
	defer actor.Stop()
	require.True(t, actor.GetSnapshot().Matches("creditCheck.Entering Information"))
}
