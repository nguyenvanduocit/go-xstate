package xstate

import (
	"fmt"
)

// ---- pure transition API (transition.ts / getNextSnapshot.ts) ----

// CreateInertActorScope mirrors createInertActorScope(actorLogic).
func CreateInertActorScope(logic ActorLogic) *ActorScope {
	li, ok := logic.(logicImpl)
	if !ok {
		panic(fmt.Errorf("xstate: unsupported actor logic %T", logic))
	}
	self := li.newActorRef(actorOptions{})
	c := coreOf(self)
	fresh := newActorSystem(c, c.system.clock, c.system.logger)
	c.system = fresh
	return &ActorScope{
		Self:           self,
		Defer:          func(func()) {},
		Logger:         func(...any) {},
		StopChild:      func(ActorRef) {},
		System:         fresh,
		Emit:           func(Event) {},
		actionExecutor: func(ExecutableAction) {},
	}
}

func withInertScope[T any](logic ActorLogic, fn func(scope *ActorScope, actions *[]ExecutableAction) T) (T, []ExecutableAction) {
	scope := CreateInertActorScope(logic)
	var actions []ExecutableAction
	scope.actionExecutor = func(a ExecutableAction) { actions = append(actions, a) }
	scope.System.lock()
	defer scope.System.unlock()
	out := fn(scope, &actions)
	return out, actions
}

// Transition mirrors transition(machine, snapshot, event).
func Transition[C any](m *StateMachine[C], s *MachineSnapshot[C], e Event) (*MachineSnapshot[C], []ExecutableAction) {
	return withInertScope(m, func(scope *ActorScope, _ *[]ExecutableAction) *MachineSnapshot[C] {
		return m.Transition(s, e, scope)
	})
}

// InitialTransition mirrors initialTransition(machine, input).
func InitialTransition[C any](m *StateMachine[C], input ...any) (*MachineSnapshot[C], []ExecutableAction) {
	return withInertScope(m, func(scope *ActorScope, _ *[]ExecutableAction) *MachineSnapshot[C] {
		return m.GetInitialSnapshot(scope, firstInput(input))
	})
}

// TransitionActorLogic mirrors transition(logic, snapshot, event) for any
// actor logic.
func TransitionActorLogic[S Snapshot](logic TypedActorLogic[S], s S, e Event) (S, []ExecutableAction) {
	li := any(logic).(logicImpl)
	return withInertScope(logic, func(scope *ActorScope, _ *[]ExecutableAction) S {
		return li.transitionAny(s, e, scope).(S)
	})
}

// InitialTransitionActorLogic mirrors initialTransition(logic, input) for any
// actor logic.
func InitialTransitionActorLogic[S Snapshot](logic TypedActorLogic[S], input ...any) (S, []ExecutableAction) {
	li := any(logic).(logicImpl)
	return withInertScope(logic, func(scope *ActorScope, _ *[]ExecutableAction) S {
		return li.initialSnapshot(scope, firstInput(input)).(S)
	})
}

func firstInput(input []any) any {
	if len(input) > 0 {
		return input[0]
	}
	return nil
}

// GetNextSnapshot mirrors getNextSnapshot(logic, snapshot, event).
func GetNextSnapshot[S Snapshot](logic TypedActorLogic[S], s S, e Event) S {
	li := any(logic).(logicImpl)
	scope := CreateInertActorScope(logic)
	if c := coreOf(scope.Self); c != nil {
		c.snapshot = s
	}
	scope.System.lock()
	defer scope.System.unlock()
	return li.transitionAny(s, e, scope).(S)
}

// GetInitialSnapshot mirrors getInitialSnapshot(logic, input).
func GetInitialSnapshot[S Snapshot](logic TypedActorLogic[S], input ...any) S {
	li := any(logic).(logicImpl)
	scope := CreateInertActorScope(logic)
	scope.System.lock()
	defer scope.System.unlock()
	return li.initialSnapshot(scope, firstInput(input)).(S)
}

// Microstep is one [snapshot, actions] pair.
type Microstep[C any] struct {
	Snapshot *MachineSnapshot[C]
	Actions  []ExecutableAction
}

func toMicrosteps[C any](steps []microstepResult) []Microstep[C] {
	out := make([]Microstep[C], len(steps))
	for i, s := range steps {
		out[i] = Microstep[C]{Snapshot: s.snapshot.(*MachineSnapshot[C]), Actions: s.actions}
	}
	return out
}

// GetMicrosteps mirrors getMicrosteps(machine, snapshot, event).
func GetMicrosteps[C any](m *StateMachine[C], s *MachineSnapshot[C], e Event) []Microstep[C] {
	scope := CreateInertActorScope(m)
	scope.System.lock()
	defer scope.System.unlock()
	return toMicrosteps[C](macrostep(s, e, scope, &[]Event{}).microsteps)
}

// GetInitialMicrosteps mirrors getInitialMicrosteps(machine, input).
func GetInitialMicrosteps[C any](m *StateMachine[C], input ...any) []Microstep[C] {
	scope := CreateInertActorScope(m)
	scope.System.lock()
	defer scope.System.unlock()
	initEvent := InitEvent{Input: firstInput(input)}
	iq := &[]Event{}
	pre := m.getPreInitialState(scope, initEvent, iq)
	first := initialMicrostep(m.Root, pre, scope, initEvent, iq)
	macro := macrostep(first.snapshot, initEvent, scope, iq)
	return toMicrosteps[C](append([]microstepResult{first}, macro.microsteps...))
}

// GetNextTransitions mirrors getNextTransitions(snapshot).
func GetNextTransitions[C any](s *MachineSnapshot[C]) []*TransitionDefinition {
	var out []*TransitionDefinition
	visited := map[string]bool{}
	for _, n := range s.nodes {
		if !isAtomicStateNode(n) {
			continue
		}
		for _, sn := range append([]*StateNode{n}, getProperAncestors(n, nil)...) {
			if visited[sn.ID] {
				continue
			}
			visited[sn.ID] = true
			for _, k := range sn.transitions.keys {
				out = append(out, sn.transitions.m[k]...)
			}
			out = append(out, sn.always...)
		}
	}
	return out
}
