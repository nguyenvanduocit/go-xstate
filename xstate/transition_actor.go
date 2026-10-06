package xstate

import (
	"fmt"
)

// ---- fromTransition ----

// TransitionSnapshot mirrors TransitionSnapshot.
type TransitionSnapshot[T any] struct {
	Status  Status
	Context T
	Output  any
	Error   any
}

func (s *TransitionSnapshot[T]) GetStatus() Status { return s.Status }

func (s *TransitionSnapshot[T]) GetOutput() any { return s.Output }

func (s *TransitionSnapshot[T]) GetError() any { return s.Error }

// TransitionLogic mirrors TransitionActorLogic.
type TransitionLogic[T any] struct {
	reducer func(state T, event Event, scope *ActorScope) T
	initial func(args TransitionInitArgs) T
}

func (l *TransitionLogic[T]) logic() {}

func (l *TransitionLogic[T]) typedLogic() *TransitionSnapshot[T] { return nil }

// TransitionInitArgs are passed to the initial-state function of FromTransition.
type TransitionInitArgs struct {
	Input  any
	Self   ActorRef
	System *ActorSystem
}

// FromTransition mirrors fromTransition(reducer, initialState).
func FromTransition[T any](
	reducer func(state T, event Event, scope *ActorScope) T,
	initial func(args TransitionInitArgs) T,
) *TransitionLogic[T] {
	return &TransitionLogic[T]{reducer: reducer, initial: initial}
}

func (l *TransitionLogic[T]) initialSnapshot(scope *ActorScope, input any) Snapshot {
	var ctx T
	if l.initial != nil {
		args := TransitionInitArgs{Input: input}
		if scope != nil {
			args.Self, args.System = scope.Self, scope.System
		}
		ctx = l.initial(args)
	}
	return &TransitionSnapshot[T]{Status: StatusActive, Context: ctx}
}

func (l *TransitionLogic[T]) transitionAny(s Snapshot, e Event, scope *ActorScope) Snapshot {
	state := s.(*TransitionSnapshot[T])
	next := *state
	next.Context = l.reducer(state.Context, e, scope)
	return &next
}

func (l *TransitionLogic[T]) hasStart() bool { return false }

func (l *TransitionLogic[T]) startAny(Snapshot, *ActorScope) {}

func (l *TransitionLogic[T]) persistAny(s Snapshot) any { return s }

func (l *TransitionLogic[T]) hasRestore() bool { return true }

func (l *TransitionLogic[T]) restoreAny(p any, _ *ActorScope) Snapshot {
	return convertSnapshot[*TransitionSnapshot[T]](p)
}

func (l *TransitionLogic[T]) newActorRef(o actorOptions) ActorRef {
	return newActor[*TransitionSnapshot[T]](l, l, o)
}

// ---- LogicOf ----

// LogicOf mirrors the JS object spread `{ ...actorLogic }` used to compose
// actor logic. It returns a Logic[S] whose function fields delegate to logic.
func LogicOf[S Snapshot](logic TypedActorLogic[S]) *Logic[S] {
	li, ok := any(logic).(logicImpl)
	if !ok {
		panic(fmt.Errorf("xstate: unsupported actor logic %T", logic))
	}
	l := &Logic[S]{
		Transition: func(s S, e Event, scope *ActorScope) S {
			return li.transitionAny(s, e, scope).(S)
		},
		GetInitialSnapshot: func(scope *ActorScope, input any) S {
			return li.initialSnapshot(scope, input).(S)
		},
		GetPersistedSnapshot: func(s S) any { return li.persistAny(s) },
		RestoreSnapshot: func(p any, scope *ActorScope) S {
			return li.restoreAny(p, scope).(S)
		},
	}
	if li.hasStart() {
		l.Start = func(s S, scope *ActorScope) { li.startAny(s, scope) }
	}
	return l
}
