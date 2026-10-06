package xstate

import (
	"context"
	"encoding/json"
)

// ---- fromPromise ----

// PromiseArgs are passed to FromPromise functions. ctx is cancelled when the
// actor stops (JS `signal`).
type PromiseArgs struct {
	Input  any
	Self   ActorRef
	System *ActorSystem
	Emit   func(Event)
}

// PromiseSnapshot mirrors PromiseSnapshot.
type PromiseSnapshot[O any] struct {
	Status Status `json:"status"`
	Output O      `json:"output"`
	Error  any    `json:"error,omitempty"`
	Input  any    `json:"input,omitempty"`
}

// MarshalJSON omits unresolved output while preserving resolved zero values.
func (s PromiseSnapshot[O]) MarshalJSON() ([]byte, error) {
	var output *O
	if s.Status == StatusDone {
		output = &s.Output
	}
	return json.Marshal(struct {
		Status Status `json:"status"`
		Output *O     `json:"output,omitempty"`
		Error  any    `json:"error,omitempty"`
		Input  any    `json:"input,omitempty"`
	}{s.Status, output, s.Error, s.Input})
}

func (s *PromiseSnapshot[O]) GetStatus() Status { return s.Status }

func (s *PromiseSnapshot[O]) GetOutput() any { return s.Output }

func (s *PromiseSnapshot[O]) GetError() any { return s.Error }

// PromiseLogic mirrors PromiseActorLogic.
type PromiseLogic[O any] struct {
	fn func(ctx context.Context, args PromiseArgs) (O, error)
}

func (l *PromiseLogic[O]) logic() {}

func (l *PromiseLogic[O]) typedLogic() *PromiseSnapshot[O] {
	return nil
}

// FromPromise mirrors fromPromise(fn). fn runs on its own goroutine.
func FromPromise[O any](fn func(ctx context.Context, args PromiseArgs) (O, error)) *PromiseLogic[O] {
	return &PromiseLogic[O]{fn: fn}
}

type promiseState struct {
	cancel context.CancelCauseFunc
}

func (l *PromiseLogic[O]) initialSnapshot(_ *ActorScope, input any) Snapshot {
	return &PromiseSnapshot[O]{Status: StatusActive, Input: input}
}

func (l *PromiseLogic[O]) transitionAny(s Snapshot, e Event, scope *ActorScope) Snapshot {
	state := s.(*PromiseSnapshot[O])
	if state.Status != StatusActive {
		return state
	}
	switch ev := e.(type) {
	case PromiseResolveEvent:
		next := *state
		next.Status = StatusDone
		next.Output = castTo[O](ev.Data)
		next.Input = nil
		return &next
	case promiseRejectEvent:
		next := *state
		next.Status = StatusError
		next.Error = ev.Data
		next.Input = nil
		return &next
	}
	if e.EventType() == xstateStop {
		if c := scopeCore(scope); c != nil {
			if ps, ok := c.logicState.(*promiseState); ok && ps.cancel != nil {
				ps.cancel(context.Canceled)
			}
			c.logicState = nil
		}
		next := *state
		next.Status = StatusStopped
		next.Input = nil
		return &next
	}
	return state
}

func (l *PromiseLogic[O]) hasStart() bool { return true }

func (l *PromiseLogic[O]) startAny(s Snapshot, scope *ActorScope) {
	state := s.(*PromiseSnapshot[O])
	if state.Status != StatusActive {
		return
	}
	c := scopeCore(scope)
	ctx, cancel := context.WithCancelCause(context.Background())
	ps := &promiseState{cancel: cancel}
	if c != nil {
		c.logicState = ps
	}
	self, system := scope.Self, scope.System
	emit := func(ev Event) {
		system.lock()
		defer system.unlock()
		scope.Emit(ev)
	}
	args := PromiseArgs{Input: state.Input, Self: self, System: system, Emit: emit}
	seq := system.asyncBegin()
	go func() {
		var ev Event
		var out O
		var err error
		if r := safeCall(func() { out, err = l.fn(ctx, args) }); r != nil {
			ev = promiseRejectEvent{Data: r.value}
		} else if err != nil {
			ev = promiseRejectEvent{Data: err}
		} else {
			ev = PromiseResolveEvent{Data: out}
		}
		system.asyncSettle(seq, func() {
			if c == nil || c.snapshot.GetStatus() != StatusActive {
				return
			}
			if c.logicState == ps {
				c.logicState = nil
			}
			system.relay(self, self, ev)
		})
	}()
}

func (l *PromiseLogic[O]) persistAny(s Snapshot) any { return s }

func (l *PromiseLogic[O]) hasRestore() bool { return true }

func (l *PromiseLogic[O]) restoreAny(p any, _ *ActorScope) Snapshot {
	return convertSnapshot[*PromiseSnapshot[O]](p)
}

func (l *PromiseLogic[O]) newActorRef(o actorOptions) ActorRef {
	return newActor[*PromiseSnapshot[O]](l, l, o)
}
