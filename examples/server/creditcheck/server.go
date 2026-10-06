package creditcheck

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand/v2"
	"net"
	"net/http"
	"sync"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// Port is the port index.ts listens on.
const Port = 4242

const indexHTML = `
    <html>
      <body style="font-family: sans-serif;">
        <h1>Express Workflow</h1>
        <p>Start a new workflow instance:</p>
        <pre>curl -X POST http://localhost:4242/workflows</pre>
        <p>Send an event to a workflow instance:</p>
        <pre>curl -X POST http://localhost:4242/workflows/:workflowId -d '{"type":"TIMER"}'</pre>
        <p>Get the current state of a workflow instance:</p>
        <pre>curl -X GET http://localhost:4242/workflows/:workflowId</pre>
      </body>
    </html>
  `

// Actor is a running credit check workflow.
type Actor = xs.Actor[*xs.MachineSnapshot[CreditProfile]]

// Server mirrors index.ts and services/actorService.ts: every request builds an actor
// (hydrated from the stored snapshot when a workflow id is given) that persists each
// snapshot to the store. Actors stay alive after the response, like in JS, so in-flight
// bureau calls finish and persist; Close stops them.
type Server struct {
	// Logf receives what the JS code prints with console.log. Defaults to log.Printf.
	Logf func(format string, args ...any)

	machine *xs.StateMachine[CreditProfile]
	store   Store
	newID   func() string

	mu     sync.Mutex
	actors map[*Actor]struct{}
}

// NewServer returns a server for machine over store. newID names new workflows; nil
// generates random 6-character base-36 ids like generateActorId().
func NewServer(machine *xs.StateMachine[CreditProfile], store Store, newID func() string) *Server {
	if newID == nil {
		newID = generateActorID
	}
	return &Server{Logf: log.Printf, machine: machine, store: store, newID: newID, actors: map[*Actor]struct{}{}}
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

// Close stops every actor the server started.
func (s *Server) Close() {
	s.mu.Lock()
	actors := make([]*Actor, 0, len(s.actors))
	for a := range s.actors {
		actors = append(actors, a)
	}
	s.mu.Unlock()
	for _, a := range actors {
		a.Stop()
	}
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
	sendText(w, http.StatusOK, indexHTML)
}

// start mirrors POST /workflows.
func (s *Server) start(w http.ResponseWriter, r *http.Request) {
	s.Logf("starting new workflow...")
	_, workflowID, err := s.durableActor(r.Context(), "")
	if err != nil {
		s.Logf("%v", err)
		sendText(w, http.StatusInternalServerError, "Error starting workflow. Details: "+jsString(err))
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{
		"message":    "New workflow created successfully",
		"workflowId": workflowID,
	})
}

// send mirrors POST /workflows/:workflowId. After a 500 JS goes on to send a second
// response, which Express rejects; here the handler returns.
func (s *Server) send(w http.ResponseWriter, r *http.Request) {
	// body-parser rejects malformed JSON with 400; an event without a string type
	// crashes the Node process in JS (see NOTES.md), so it is a 400 here too.
	var event xs.E
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		sendText(w, http.StatusBadRequest, "Bad Request")
		return
	}
	if _, ok := event["type"].(string); !ok {
		sendText(w, http.StatusBadRequest, "Bad Request")
		return
	}
	actor, _, err := s.durableActor(r.Context(), r.PathValue("workflowId"))
	if err != nil {
		s.Logf("%v", err)
		sendText(w, http.StatusInternalServerError, "Error sending event. Details: "+jsString(err))
		return
	}
	actor.Send(event)
	sendText(w, http.StatusOK, "Event received. Issue a GET request to see the current workflow state")
}

// get mirrors GET /workflows/:workflowId.
func (s *Server) get(w http.ResponseWriter, r *http.Request) {
	state, err := s.store.FindMachineState(r.Context(), r.PathValue("workflowId"))
	if err != nil {
		s.Logf("%v", err)
		sendText(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	if state == nil {
		sendText(w, http.StatusNotFound, "Workflow state not found")
		return
	}
	writeJSON(w, http.StatusOK, state)
}

// durableActor mirrors getDurableActor: it creates an actor for the machine, restored from
// the stored snapshot when workflowID is set (a new id is generated otherwise), persists
// every snapshot, and starts it.
func (s *Server) durableActor(ctx context.Context, workflowID string) (*Actor, string, error) {
	var opts []xs.ActorOption
	if workflowID != "" {
		state, err := s.store.FindMachineState(ctx, workflowID)
		if err != nil {
			return nil, "", err
		}
		if state == nil {
			return nil, "", errors.New("Actor not found with the provided workflowId")
		}
		s.Logf("restored state %v", state)
		opts = append(opts, xs.WithSnapshot(state.PersistedState))
	} else {
		workflowID = s.newID()
	}

	actor := xs.CreateActor(s.machine, opts...)
	actor.Subscribe(xs.Observer[*xs.MachineSnapshot[CreditProfile]]{
		Next: func(*xs.MachineSnapshot[CreditProfile]) {
			if err := s.persist(workflowID, actor); err != nil {
				s.Logf("%v", err)
			}
		},
		Error: func(err any) { s.Logf("Error in actor subscription: %v", err) },
		Complete: func() {
			s.Logf("Actor is finished!")
			actor.Stop()
		},
	})
	s.mu.Lock()
	s.actors[actor] = struct{}{}
	s.mu.Unlock()
	actor.Start()
	return actor, workflowID, nil
}

// persist mirrors the subscription's next handler of getDurableActor.
func (s *Server) persist(workflowID string, actor *Actor) error {
	persisted, err := plainJSON(actor.GetPersistedSnapshot())
	if err != nil {
		return err
	}
	s.Logf("persisted state %v", persisted)
	if err := s.store.SaveMachineState(context.Background(), MachineState{WorkflowID: workflowID, PersistedState: persisted}); err != nil {
		return fmt.Errorf("Error persisting actor state. Verify db connection is configured correctly. %w", err)
	}
	return nil
}

// plainJSON converts a persisted snapshot to the generic JSON form a document holds.
// JS drops `undefined` properties (output and error of an active snapshot); nil entries
// are this port's undefined.
func plainJSON(v any) (map[string]any, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	dropNulls(out)
	return out, nil
}

func dropNulls(m map[string]any) {
	for k, v := range m {
		switch x := v.(type) {
		case nil:
			delete(m, k)
		case map[string]any:
			dropNulls(x)
		}
	}
}

// sendText mirrors res.status(code).send(string): no trailing newline.
func sendText(w http.ResponseWriter, status int, text string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	io.WriteString(w, text)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// Run mirrors app.listen(4242, ...): it prints the listening message and serves h on ln.
func Run(w io.Writer, ln net.Listener, h http.Handler) error {
	_, port, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "Server listening on port %s\n", port)
	return http.Serve(ln, h)
}
