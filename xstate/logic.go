package xstate

import (
	"encoding/json"
	"reflect"
)

// ActorLogic is the type-erased actor logic (machines, promise/callback/
// observable/transition logic, custom Logic[S]). It is sealed: custom logic
// is built with the Logic[S] struct.
type ActorLogic interface {
	logic()
}

// TypedActorLogic is actor logic whose snapshot type is S. CreateActor infers
// S from it.
type TypedActorLogic[S Snapshot] interface {
	ActorLogic
	typedLogic() S
}

// logicImpl is the engine-facing interface every logic implements.
type logicImpl interface {
	ActorLogic
	initialSnapshot(scope *ActorScope, input any) Snapshot
	transitionAny(s Snapshot, e Event, scope *ActorScope) Snapshot
	hasStart() bool
	startAny(s Snapshot, scope *ActorScope)
	persistAny(s Snapshot) any
	hasRestore() bool
	restoreAny(p any, scope *ActorScope) Snapshot
	newActorRef(o actorOptions) ActorRef
}

// ActorScope mirrors actorScope in JS.
type ActorScope struct {
	Self      ActorRef
	ID        string
	SessionID string
	System    *ActorSystem
	Logger    func(args ...any)
	// Defer runs fn after the current transition is committed.
	Defer func(fn func())
	// Emit mirrors actorScope.emit(event).
	Emit func(event Event)
	// StopChild mirrors actorScope.stopChild(ref).
	StopChild func(ref ActorRef)

	actionExecutor func(ExecutableAction)
}

func scopeCore(scope *ActorScope) *actorCore {
	if scope == nil {
		return nil
	}
	return coreOf(scope.Self)
}

// Logic is custom actor logic built from functions, mirroring a JS object
// literal `{ transition, getInitialSnapshot, start?, getPersistedSnapshot?,
// restoreSnapshot? }`.
type Logic[S Snapshot] struct {
	Transition           func(snapshot S, event Event, scope *ActorScope) S
	GetInitialSnapshot   func(scope *ActorScope, input any) S
	Start                func(snapshot S, scope *ActorScope)
	GetPersistedSnapshot func(snapshot S) any
	RestoreSnapshot      func(persisted any, scope *ActorScope) S
}

func (l *Logic[S]) logic() {}

func (l *Logic[S]) typedLogic() S {
	var z S
	return z
}

func (l *Logic[S]) initialSnapshot(scope *ActorScope, input any) Snapshot {
	if l.GetInitialSnapshot == nil {
		var z S
		return z
	}
	return l.GetInitialSnapshot(scope, input)
}

func (l *Logic[S]) transitionAny(s Snapshot, e Event, scope *ActorScope) Snapshot {
	v, _ := s.(S)
	if l.Transition == nil {
		return v
	}
	return l.Transition(v, e, scope)
}

func (l *Logic[S]) hasStart() bool { return l.Start != nil }

func (l *Logic[S]) startAny(s Snapshot, scope *ActorScope) {
	v, _ := s.(S)
	l.Start(v, scope)
}

func (l *Logic[S]) persistAny(s Snapshot) any {
	v, _ := s.(S)
	if l.GetPersistedSnapshot == nil {
		return v
	}
	return l.GetPersistedSnapshot(v)
}

func (l *Logic[S]) hasRestore() bool { return true }

func (l *Logic[S]) restoreAny(p any, scope *ActorScope) Snapshot {
	if l.RestoreSnapshot == nil {
		return convertSnapshot[S](p)
	}
	return l.RestoreSnapshot(p, scope)
}

func (l *Logic[S]) newActorRef(o actorOptions) ActorRef { return newActor[S](l, l, o) }

// convertSnapshot converts a persisted value to S (identity when it already
// is an S; JSON round-trip otherwise).
func convertSnapshot[S any](p any) S {
	if s, ok := p.(S); ok {
		return s
	}
	var out S
	t := reflect.TypeFor[S]()
	if t.Kind() == reflect.Pointer {
		v := reflect.New(t.Elem())
		b, err := json.Marshal(p)
		if err == nil && json.Unmarshal(b, v.Interface()) == nil {
			return v.Interface().(S)
		}
		return out
	}
	b, err := json.Marshal(p)
	if err == nil {
		_ = json.Unmarshal(b, &out)
	}
	return out
}

// snapshotWithError mirrors `{ ...snapshot, status: 'error', error: err }`.
func snapshotWithError[S Snapshot](prev S, err any) S {
	if ms, ok := any(prev).(anyMachineSnapshot); ok && !isNilSnapshot(prev) {
		return ms.clone(snapshotPatch{status: statusPtr(StatusError), err: anyPtr(err)}).(S)
	}
	t := reflect.TypeFor[S]()
	if t.Kind() != reflect.Pointer || t.Elem().Kind() != reflect.Struct {
		return prev
	}
	v := reflect.New(t.Elem())
	if !isNilSnapshot(prev) {
		v.Elem().Set(reflect.ValueOf(prev).Elem())
	}
	if f := v.Elem().FieldByName("Status"); f.IsValid() && f.CanSet() && f.Type() == reflect.TypeFor[Status]() {
		f.Set(reflect.ValueOf(StatusError))
	}
	if f := v.Elem().FieldByName("Error"); f.IsValid() && f.CanSet() {
		if err == nil {
			f.Set(reflect.Zero(f.Type()))
		} else if reflect.TypeOf(err).AssignableTo(f.Type()) {
			f.Set(reflect.ValueOf(err))
		}
	}
	return v.Interface().(S)
}

func isNilSnapshot(s any) bool {
	if s == nil {
		return true
	}
	rv := reflect.ValueOf(s)
	return rv.Kind() == reflect.Pointer && rv.IsNil()
}

// BasicSnapshot is a minimal snapshot for custom logic.
type BasicSnapshot[T any] struct {
	Status  Status
	Context T
	Output  any
	Error   any
}

func (s *BasicSnapshot[T]) GetStatus() Status { return s.Status }

func (s *BasicSnapshot[T]) GetOutput() any { return s.Output }

func (s *BasicSnapshot[T]) GetError() any { return s.Error }
