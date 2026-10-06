package xstate

import (
	"encoding/json"
	"strings"
)

// Event is anything that can be sent to an actor. The event type is the
// equivalent of `event.type` in XState JS.
type Event interface {
	EventType() string
}

// E is the dynamic event shape, mirroring `{ type: 'FOO', ...payload }` in JS.
// The "type" key must hold a string.
type E map[string]any

func (e E) EventType() string {
	t, _ := e["type"].(string)
	return t
}

// Ev returns an E that carries only a type.
func Ev(eventType string) E { return E{"type": eventType} }

// InitEvent is the event that starts every machine ("xstate.init").
type InitEvent struct {
	Input any
}

func (InitEvent) EventType() string { return "xstate.init" }

// StopEvent is sent to an actor when it is stopped ("xstate.stop").
type StopEvent struct{}

func (StopEvent) EventType() string { return "xstate.stop" }

// DoneActorEvent is raised when an invoked/spawned child completes
// ("xstate.done.actor.<ActorID>").
type DoneActorEvent struct {
	ActorID string
	Output  any
}

func (e DoneActorEvent) EventType() string { return "xstate.done.actor." + e.ActorID }

// ErrorActorEvent is raised when a child actor errors
// ("xstate.error.actor.<ActorID>").
type ErrorActorEvent struct {
	ActorID string
	Error   any
}

func (e ErrorActorEvent) EventType() string { return "xstate.error.actor." + e.ActorID }

// DoneStateEvent is raised when a compound/parallel state reaches its final
// state ("xstate.done.state.<StateID>").
type DoneStateEvent struct {
	StateID string
	Output  any
}

func (e DoneStateEvent) EventType() string { return "xstate.done.state." + e.StateID }

// SnapshotEvent is raised to a parent when an invoked child emits a new
// snapshot ("xstate.snapshot.<ActorID>").
type SnapshotEvent struct {
	ActorID  string
	Snapshot Snapshot
}

func (e SnapshotEvent) EventType() string { return "xstate.snapshot." + e.ActorID }

// ---- internal events ----

const (
	xstateInit  = "xstate.init"
	xstateStop  = "xstate.stop"
	xstateError = "xstate.error"
	wildcard    = "*"
)

// createAfterEvent mirrors eventUtils.ts createAfterEvent.
func createAfterEvent(delayRef string, id string) E {
	return E{"type": "xstate.after." + delayRef + "." + id}
}

// promiseRejectEvent mirrors `{ type: 'xstate.promise.reject', data }`.
type promiseRejectEvent struct {
	Data any
}

func (promiseRejectEvent) EventType() string { return "xstate.promise.reject" }

// observableNextEvent mirrors `{ type: 'xstate.observable.next', data }`.
type observableNextEvent struct {
	Data any
}

func (observableNextEvent) EventType() string { return "xstate.observable.next" }

// observableErrorEvent mirrors `{ type: 'xstate.observable.error', data }`.
type observableErrorEvent struct {
	Data any
}

func (observableErrorEvent) EventType() string { return "xstate.observable.error" }

// observableCompleteEvent mirrors `{ type: 'xstate.observable.complete' }`.
type observableCompleteEvent struct{}

func (observableCompleteEvent) EventType() string { return "xstate.observable.complete" }

func isErrorActorEvent(e Event) bool {
	return strings.HasPrefix(e.EventType(), "xstate.error.actor")
}

// eventErrorValue returns `event.error` of an error event.
func eventErrorValue(e Event) any {
	switch ev := e.(type) {
	case ErrorActorEvent:
		return ev.Error
	case *ErrorActorEvent:
		return ev.Error
	case E:
		return ev["error"]
	}
	return nil
}

// jsonStringify mirrors JSON.stringify for warning/error messages.
func jsonStringify(v any) (string, bool) {
	b, err := json.Marshal(toJSONValue(v))
	if err != nil {
		return "", false
	}
	return string(b), true
}

// toJSONValue converts known engine values to the shape JSON.stringify would
// produce in JS.
func toJSONValue(v any) any {
	switch ev := v.(type) {
	case E:
		return map[string]any(ev)
	case InitEvent:
		return map[string]any{"type": xstateInit, "input": ev.Input}
	case StopEvent:
		return map[string]any{"type": xstateStop}
	case DoneActorEvent:
		return map[string]any{"type": ev.EventType(), "output": ev.Output, "actorId": ev.ActorID}
	case ErrorActorEvent:
		return map[string]any{"type": ev.EventType(), "error": ev.Error, "actorId": ev.ActorID}
	case DoneStateEvent:
		return map[string]any{"type": ev.EventType(), "output": ev.Output}
	}
	return v
}

// PromiseResolveEvent mirrors the internal event a promise actor sends to
// itself when its promise resolves: `{ type: 'xstate.promise.resolve', data }`.
type PromiseResolveEvent struct {
	Data any
}

func (PromiseResolveEvent) EventType() string { return "xstate.promise.resolve" }
