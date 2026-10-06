package workflow_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	workflow "github.com/nguyenvanduocit/go-xstate/examples/server/workflow"
	"github.com/stretchr/testify/require"
)

type blockedBody struct {
	entered chan struct{}
	release chan struct{}
}

func (b *blockedBody) Read([]byte) (int, error) {
	close(b.entered)
	<-b.release
	return 0, io.EOF
}

func newWorkflowHandler() http.Handler {
	next := 0
	return workflow.NewServer(func() string {
		next++
		return fmt.Sprintf("w%d", next)
	}).Handler()
}

func awaitRequest(t *testing.T, done <-chan struct{}, failure string) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal(failure)
	}
}

// Go regression: network reads must not hold the workflow state mutex.
func TestSlowBodyDoesNotBlockOtherWorkflow(t *testing.T) {
	h := newWorkflowHandler()
	for range 2 {
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", "/workflows", nil))
	}
	body := &blockedBody{make(chan struct{}), make(chan struct{})}
	postDone := make(chan struct{})
	go func() {
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", "/workflows/w1", body))
		close(postDone)
	}()
	defer func() {
		close(body.release)
		awaitRequest(t, postDone, "POST did not finish after body was released")
	}()
	awaitRequest(t, body.entered, "POST did not attempt to read its body")
	getDone := make(chan struct{})
	response := httptest.NewRecorder()
	go func() {
		h.ServeHTTP(response, httptest.NewRequest("GET", "/workflows/w2", nil))
		close(getDone)
	}()
	select {
	case <-getDone:
		require.Equal(t, http.StatusOK, response.Code)
	case <-time.After(time.Second):
		t.Fatal("unrelated workflow GET is blocked by an unfinished POST body")
	}
}

type blockedWriter struct {
	*httptest.ResponseRecorder
	entered chan struct{}
	release chan struct{}
}

func (w *blockedWriter) Write(p []byte) (int, error) {
	close(w.entered)
	<-w.release
	return w.ResponseRecorder.Write(p)
}

func (w *blockedWriter) WriteString(s string) (int, error) { return w.Write([]byte(s)) }

// Go regression: a stalled response must not block other requests either.
func TestSlowResponseDoesNotBlockOtherWorkflow(t *testing.T) {
	for _, request := range []struct{ method, path, body string }{
		{"POST", "/workflows", ""},
		{"POST", "/workflows/w1", `{"type":"TIMER"}`},
		{"GET", "/workflows/w1", ""},
	} {
		t.Run(request.method+request.path, func(t *testing.T) {
			h := newWorkflowHandler()
			h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", "/workflows", nil))
			writer := &blockedWriter{httptest.NewRecorder(), make(chan struct{}), make(chan struct{})}
			done := make(chan struct{})
			go func() {
				h.ServeHTTP(writer, httptest.NewRequest(request.method, request.path, strings.NewReader(request.body)))
				close(done)
			}()
			defer func() {
				close(writer.release)
				awaitRequest(t, done, "request did not finish after response writer was released")
			}()
			awaitRequest(t, writer.entered, "request did not attempt to write its response")
			otherDone := make(chan struct{})
			go func() {
				h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/workflows/w1", nil))
				close(otherDone)
			}()
			select {
			case <-otherDone:
			case <-time.After(time.Second):
				t.Fatal("workflow request blocked by a slow response writer")
			}
		})
	}
}

// Go regression: reading request bodies outside the mutex must preserve atomic updates.
func TestConcurrentWorkflowEvents(t *testing.T) {
	h := newWorkflowHandler()
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", "/workflows", nil))
	var sends sync.WaitGroup
	for range 30 {
		sends.Go(func() {
			h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", "/workflows/w1", strings.NewReader(`{"type":"TIMER"}`)))
		})
	}
	sends.Wait()
	response := httptest.NewRecorder()
	h.ServeHTTP(response, httptest.NewRequest("GET", "/workflows/w1", nil))
	require.Contains(t, response.Body.String(), `"cycles":10`)
}

// Go regression: decoding event bodies has a fixed size bound.
func TestOversizedEventBody(t *testing.T) {
	h := newWorkflowHandler()
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", "/workflows", nil))
	body := `{"type":"TIMER","padding":"` + strings.Repeat("x", 1<<20) + `"}`
	response := httptest.NewRecorder()
	h.ServeHTTP(response, httptest.NewRequest("POST", "/workflows/w1", strings.NewReader(body)))
	require.Equal(t, http.StatusRequestEntityTooLarge, response.Code)
}
