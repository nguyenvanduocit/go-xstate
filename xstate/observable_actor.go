package xstate

// ---- fromObservable / fromEventObservable ----

// ObservableArgs are passed to FromObservable functions.
type ObservableArgs struct {
	Input  any
	Self   ActorRef
	System *ActorSystem
	Emit   func(Event)
}

// ObservableSnapshot mirrors ObservableSnapshot; Context is the last value.
type ObservableSnapshot[T any] struct {
	Status  Status
	Context T
	Output  any
	Error   any
	Input   any
}

func (s *ObservableSnapshot[T]) GetStatus() Status { return s.Status }

func (s *ObservableSnapshot[T]) GetOutput() any { return s.Output }

func (s *ObservableSnapshot[T]) GetError() any { return s.Error }

// ObservableLogic mirrors ObservableActorLogic.
type ObservableLogic[T any] struct {
	fn        func(args ObservableArgs) Subscribable[T]
	eventMode bool
}

func (l *ObservableLogic[T]) logic() {}

func (l *ObservableLogic[T]) typedLogic() *ObservableSnapshot[T] { return nil }

// FromObservable mirrors fromObservable(fn).
func FromObservable[T any](fn func(args ObservableArgs) Subscribable[T]) *ObservableLogic[T] {
	return &ObservableLogic[T]{fn: fn}
}

// FromEventObservable mirrors fromEventObservable(fn): every emitted value is
// sent to the parent as an event.
func FromEventObservable(fn func(args ObservableArgs) Subscribable[Event]) *ObservableLogic[Event] {
	return &ObservableLogic[Event]{fn: fn, eventMode: true}
}

type observableState struct {
	sub Subscription
}

func (l *ObservableLogic[T]) initialSnapshot(_ *ActorScope, input any) Snapshot {
	return &ObservableSnapshot[T]{Status: StatusActive, Input: input}
}

func (l *ObservableLogic[T]) transitionAny(s Snapshot, e Event, scope *ActorScope) Snapshot {
	state := s.(*ObservableSnapshot[T])
	if state.Status != StatusActive {
		return state
	}
	switch ev := e.(type) {
	case observableNextEvent:
		if l.eventMode {
			return state
		}
		next := *state
		next.Context = castTo[T](ev.Data)
		return &next
	case observableErrorEvent:
		next := *state
		next.Status = StatusError
		next.Error = ev.Data
		next.Input = nil
		l.clearSub(scope, false)
		return &next
	case observableCompleteEvent:
		next := *state
		next.Status = StatusDone
		next.Input = nil
		l.clearSub(scope, false)
		return &next
	}
	if e.EventType() == xstateStop {
		next := *state
		next.Status = StatusStopped
		next.Input = nil
		l.clearSub(scope, true)
		return &next
	}
	return state
}

func (l *ObservableLogic[T]) clearSub(scope *ActorScope, unsubscribe bool) {
	c := scopeCore(scope)
	if c == nil {
		return
	}
	if os, ok := c.logicState.(*observableState); ok && unsubscribe && os.sub != nil {
		os.sub.Unsubscribe()
	}
	c.logicState = nil
}

func (l *ObservableLogic[T]) hasStart() bool { return true }

func (l *ObservableLogic[T]) startAny(s Snapshot, scope *ActorScope) {
	state := s.(*ObservableSnapshot[T])
	if state.Status == StatusDone {
		return
	}
	c := scopeCore(scope)
	os := &observableState{}
	if c != nil {
		c.logicState = os
	}
	self, system := scope.Self, scope.System
	relay := func(target ActorRef, ev Event) {
		system.lock()
		defer system.unlock()
		// An emission that was already in flight when the subscription was
		// cancelled must not reach the stopped actor (JS unsubscribe is
		// synchronous, so no emission can follow it).
		if c != nil && c.processingStatus == statusStopped {
			return
		}
		system.relay(self, target, ev)
	}
	src := l.fn(ObservableArgs{
		Input:  state.Input,
		Self:   self,
		System: system,
		Emit: func(ev Event) {
			system.lock()
			defer system.unlock()
			scope.Emit(ev)
		},
	})
	os.sub = src.Subscribe(Observer[T]{
		Next: func(v T) {
			if l.eventMode {
				if ev, ok := any(v).(Event); ok {
					if p := actorParent(self); p != nil {
						relay(p, ev)
					}
				}
				return
			}
			relay(self, observableNextEvent{Data: v})
		},
		Error: func(err any) { relay(self, observableErrorEvent{Data: err}) },
		Complete: func() {
			relay(self, observableCompleteEvent{})
		},
	})
}

func (l *ObservableLogic[T]) persistAny(s Snapshot) any { return s }

func (l *ObservableLogic[T]) hasRestore() bool { return true }

func (l *ObservableLogic[T]) restoreAny(p any, _ *ActorScope) Snapshot {
	return convertSnapshot[*ObservableSnapshot[T]](p)
}

func (l *ObservableLogic[T]) newActorRef(o actorOptions) ActorRef {
	return newActor[*ObservableSnapshot[T]](l, l, o)
}
