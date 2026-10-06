package xstate

// ---- fromCallback ----

// CallbackArgs are passed to FromCallback functions.
type CallbackArgs struct {
	Input    any
	Self     ActorRef
	System   *ActorSystem
	SendBack func(Event)
	Receive  func(func(Event))
	Emit     func(Event)
}

// CallbackSnapshot mirrors CallbackSnapshot.
type CallbackSnapshot struct {
	Status Status
	Error  any
	Input  any
}

func (s *CallbackSnapshot) GetStatus() Status { return s.Status }

func (s *CallbackSnapshot) GetOutput() any { return nil }

func (s *CallbackSnapshot) GetError() any { return s.Error }

// CallbackLogic mirrors CallbackActorLogic.
type CallbackLogic struct {
	fn func(args CallbackArgs) func()
}

func (l *CallbackLogic) logic() {}

func (l *CallbackLogic) typedLogic() *CallbackSnapshot { return nil }

// FromCallback mirrors fromCallback(fn). fn returns an optional cleanup.
func FromCallback(fn func(args CallbackArgs) func()) *CallbackLogic {
	return &CallbackLogic{fn: fn}
}

type callbackState struct {
	receivers []func(Event)
	dispose   func()
}

func (l *CallbackLogic) initialSnapshot(_ *ActorScope, input any) Snapshot {
	return &CallbackSnapshot{Status: StatusActive, Input: input}
}

func (l *CallbackLogic) hasStart() bool { return true }

func (l *CallbackLogic) startAny(s Snapshot, scope *ActorScope) {
	state := s.(*CallbackSnapshot)
	c := scopeCore(scope)
	cs := &callbackState{}
	if c != nil {
		c.logicState = cs
	}
	self, system := scope.Self, scope.System
	cs.dispose = l.fn(CallbackArgs{
		Input:  state.Input,
		Self:   self,
		System: system,
		SendBack: func(ev Event) {
			system.lock()
			defer system.unlock()
			if c != nil && c.snapshot.GetStatus() == StatusStopped {
				return
			}
			if p := actorParent(self); p != nil {
				system.relay(self, p, ev)
			}
		},
		Receive: func(listener func(Event)) {
			system.lock()
			defer system.unlock()
			cs.receivers = append(cs.receivers, listener)
		},
		Emit: func(ev Event) {
			system.lock()
			defer system.unlock()
			scope.Emit(ev)
		},
	})
}

func (l *CallbackLogic) transitionAny(s Snapshot, e Event, scope *ActorScope) Snapshot {
	state := s.(*CallbackSnapshot)
	c := scopeCore(scope)
	var cs *callbackState
	if c != nil {
		cs, _ = c.logicState.(*callbackState)
	}
	if e.EventType() == xstateStop {
		next := *state
		next.Status = StatusStopped
		next.Error = nil
		if c != nil {
			c.logicState = nil
		}
		if cs != nil {
			cs.receivers = nil
			if cs.dispose != nil {
				cs.dispose()
			}
		}
		return &next
	}
	if cs != nil {
		for _, r := range append([]func(Event){}, cs.receivers...) {
			r(e)
		}
	}
	return state
}

func (l *CallbackLogic) persistAny(s Snapshot) any { return s }

func (l *CallbackLogic) hasRestore() bool { return true }

func (l *CallbackLogic) restoreAny(p any, _ *ActorScope) Snapshot {
	return convertSnapshot[*CallbackSnapshot](p)
}

func (l *CallbackLogic) newActorRef(o actorOptions) ActorRef {
	return newActor[*CallbackSnapshot](l, l, o)
}
