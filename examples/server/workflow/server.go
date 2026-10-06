package workflow

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net"
	"net/http"
	"sync"
	"time"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// Port is the port index.ts listens on.
const Port = 4242

// indexHTML is the body of GET / in index.ts.
const indexHTML = `
    <html>
      <body style="font-family: sans-serif;">
        <h1>Express Workflow</h1>
        <p>Start a new workflow instance:</p>
        <pre>curl -X POST http://localhost:4242/workflows</pre>
        <p>Send an event to a workflow instance:</p>
        <pre>curl -X POST http://localhost:4242/workflows/:workflowId -d '{"type":"TIMER"}' -H "Content-Type: application/json" </pre>
        <p>Get the current state of a workflow instance:</p>
        <pre>curl -X GET http://localhost:4242/workflows/:workflowId</pre>
      </body>
    </html>
  `

// Server mirrors the express app of index.ts: workflow instances live as
// persisted snapshots, an actor is created per request and stopped after it.
type Server struct {
	machine *xs.StateMachine[Context]
	newID   func() string

	// mu keeps read-restore-send-persist atomic; network I/O runs outside it.
	mu              sync.Mutex
	persistedStates map[string]any
}

// NewServer returns a server that names workflows with newID; a nil newID
// generates random 6-character base-36 ids like generateActorId().
func NewServer(newID func() string) *Server {
	if newID == nil {
		newID = generateActorID
	}
	return &Server{machine: Machine(), newID: newID, persistedStates: map[string]any{}}
}

// generateActorID mirrors Math.random().toString(36).substring(2, 8).
func generateActorID() string {
	const alphabet = "0123456789abcdefghijklmnopqrstuvwxyz"
	id := make([]byte, 6)
	for i := range id {
		id[i] = alphabet[rand.IntN(len(alphabet))]
	}
	return string(id)
}

// Handler returns the routes of index.ts.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /workflows", s.start)
	mux.HandleFunc("POST /workflows/{workflowId}", s.send)
	mux.HandleFunc("GET /workflows/{workflowId}", s.get)
	mux.HandleFunc("GET /{$}", index)
	return mux
}

func index(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	io.WriteString(w, indexHTML)
}

// start mirrors POST /workflows.
func (s *Server) start(w http.ResponseWriter, _ *http.Request) {
	workflowID := func() string {
		s.mu.Lock()
		defer s.mu.Unlock()
		id := s.newID()
		actor := xs.CreateActor(s.machine).Start()
		s.persistedStates[id] = actor.GetPersistedSnapshot()
		actor.Stop()
		return id
	}()
	writeJSON(w, map[string]string{"workflowId": workflowID})
}

// send mirrors POST /workflows/:workflowId.
func (s *Server) send(w http.ResponseWriter, r *http.Request) {
	workflowID := r.PathValue("workflowId")
	s.mu.Lock()
	_, ok := s.persistedStates[workflowID]
	s.mu.Unlock()
	if !ok {
		sendText(w, http.StatusNotFound, "Actor not found")
		return
	}
	// A malformed body is a 400 in Express (body-parser). An event without a
	// string type crashes the JS process (event.type.startsWith); here it is
	// a client error too.
	var event xs.E
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 100<<10)).Decode(&event); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			sendText(w, http.StatusRequestEntityTooLarge, "Request Entity Too Large")
			return
		}
		sendText(w, http.StatusBadRequest, "Bad Request")
		return
	}
	if _, ok := event["type"].(string); !ok {
		sendText(w, http.StatusBadRequest, "Bad Request")
		return
	}
	func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		// Read the latest snapshot after decoding; another event may have
		// completed while this request was waiting for its body.
		snapshot := s.persistedStates[workflowID]
		actor := xs.CreateActor(s.machine, xs.WithSnapshot(snapshot)).Start()
		actor.Send(event)
		s.persistedStates[workflowID] = actor.GetPersistedSnapshot()
		actor.Stop()
	}()
	io.WriteString(w, "OK")
}

// get mirrors GET /workflows/:workflowId.
func (s *Server) get(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	persisted, ok := s.persistedStates[r.PathValue("workflowId")].(map[string]any)
	s.mu.Unlock()
	if !ok {
		sendText(w, http.StatusNotFound, "Actor not found")
		return
	}
	// res.json drops undefined properties (output and error of an active
	// snapshot); a nil map entry is this port's undefined.
	out := make(map[string]any, len(persisted))
	for k, v := range persisted {
		if v != nil {
			out[k] = v
		}
	}
	writeJSON(w, out)
}

// sendText mirrors res.status(code).send(string): no trailing newline.
func sendText(w http.ResponseWriter, status int, text string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	io.WriteString(w, text)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(v)
}

// Run mirrors app.listen(4242, ...): it prints the listening message and
// serves h on ln until the listener fails.
func Run(w io.Writer, ln net.Listener, h http.Handler) error {
	_, port, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "Server listening on port %s\n", port)
	server := &http.Server{
		Handler:           h,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       time.Minute,
	}
	return server.Serve(ln)
}
