package workflow_test

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	expressworkflow "github.com/nguyenvanduocit/go-xstate/examples/server/workflow"
)

type goldenStep struct {
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
}

// TestServerTrace replays testdata/server.golden.json, recorded by running the
// real handlers of index.ts (scripts/trace/express-workflow/server.ts).
// Workflow ids are random in JS; the ids the JS run returned are fed to the
// Go server in order.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/express-workflow/index.ts#L22
// JS trace: scripts/trace/express-workflow/server.ts.
func TestServerTrace(t *testing.T) {
	raw, err := os.ReadFile("testdata/server.golden.json")
	require.NoError(t, err)
	var g struct {
		Steps []goldenStep `json:"steps"`
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
	srv := httptest.NewServer(expressworkflow.NewServer(func() string {
		id := ids[next]
		next++
		return id
	}).Handler())
	defer srv.Close()

	for i, s := range g.Steps {
		var body io.Reader
		if len(s.Request.Body) > 0 {
			body = strings.NewReader(string(s.Request.Body))
		}
		req, err := http.NewRequest(s.Request.Method, srv.URL+s.Request.Path, body)
		require.NoError(t, err)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		resp, err := srv.Client().Do(req)
		require.NoError(t, err)
		got, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		require.NoError(t, err)

		require.Equal(t, s.Response.Status, resp.StatusCode, "step %d %s %s", i, s.Request.Method, s.Request.Path)
		if s.Response.Text != nil {
			require.Equal(t, *s.Response.Text, string(got), "step %d", i)
		} else {
			var gotJSON map[string]any
			require.NoError(t, json.Unmarshal(got, &gotJSON), "step %d body %q", i, got)
			require.Equal(t, s.Response.JSON, gotJSON, "step %d", i)
		}
	}
	require.Equal(t, len(ids), next, "every recorded workflow id was used")
}

// TestBadEvents covers inputs the JS server cannot answer (a typeless event
// crashes the Node process), so they have no golden entry.
// Go regression; expected values are Go assertions, not a recorded JS test.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/express-workflow/index.ts#L22
// Related JS trace: scripts/trace/express-workflow/server.ts.
func TestBadEvents(t *testing.T) {
	h := expressworkflow.NewServer(func() string { return "w1" }).Handler()
	do := func(method, path, body string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(method, path, strings.NewReader(body)))
		return rec
	}
	require.Equal(t, 200, do("POST", "/workflows", "").Code)
	for _, body := range []string{"", "not json", "{}", `{"type":1}`, "[]"} {
		require.Equal(t, 400, do("POST", "/workflows/w1", body).Code, "body %q", body)
	}
	require.Equal(t, 404, do("POST", "/workflows/nope", "not json").Code)
	require.Contains(t, do("GET", "/workflows/w1", "").Body.String(), `"value":"green"`)
}

// TestGeneratedIDs checks the default id generator: 6 base-36 characters.
// Go regression; expected values are Go assertions, not a recorded JS test.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/express-workflow/index.ts#L4
// Related JS trace: scripts/trace/express-workflow/server.ts.
func TestGeneratedIDs(t *testing.T) {
	h := expressworkflow.NewServer(nil).Handler()
	seen := map[string]bool{}
	for range 20 {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest("POST", "/workflows", nil))
		var out map[string]string
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
		require.Regexp(t, `^[0-9a-z]{6}$`, out["workflowId"])
		seen[out["workflowId"]] = true
	}
	require.Greater(t, len(seen), 1)
}

type fakeListener struct{ port int }

func (l fakeListener) Accept() (net.Conn, error) { return nil, net.ErrClosed }
func (l fakeListener) Close() error              { return nil }
func (l fakeListener) Addr() net.Addr            { return &net.TCPAddr{IP: net.IPv4zero, Port: l.port} }

// TestRun compares Run's output with testdata/express-workflow.stdout.txt,
// recorded from index.ts by scripts/trace/express-workflow/express-workflow.stdout.ts.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/express-workflow/index.ts#L22
// JS trace: scripts/trace/express-workflow/express-workflow.stdout.ts.
func TestRun(t *testing.T) {
	want, err := os.ReadFile("testdata/express-workflow.stdout.txt")
	require.NoError(t, err)
	var out strings.Builder
	err = expressworkflow.Run(&out, fakeListener{port: expressworkflow.Port}, http.NotFoundHandler())
	require.ErrorIs(t, err, net.ErrClosed)
	require.Equal(t, string(want), out.String())
}
