package xstate

import (
	"errors"
)

// Guard is an inline or referenced transition guard.
type Guard interface {
	GuardType() string
}

// GuardArgs are passed to inline guard functions.
type GuardArgs[C any] struct {
	Context C
	Event   Event
	Self    ActorRef
	System  *ActorSystem
	Params  any
}

type guardArgs struct {
	context any
	event   Event
}

type inlineGuard struct {
	fn func(a guardArgs, params any) bool
}

func (*inlineGuard) GuardType() string { return "" }

// GuardFunc creates an inline guard, mirroring `guard: ({ context }) => ...`.
func GuardFunc[C any](fn func(GuardArgs[C]) bool) Guard {
	return &inlineGuard{fn: func(a guardArgs, params any) bool {
		return fn(GuardArgs[C]{Context: castTo[C](a.context), Event: a.event, Params: params})
	}}
}

// GuardRef references a guard by name, mirroring `'name'` or
// `{ type: 'name', params }`. Params may be a static value or an Expr.
type GuardRef struct {
	Type   string
	Params any
}

func (g GuardRef) GuardType() string { return g.Type }

// builtinGuard mirrors the JS BuiltinGuard (and/or/not/stateIn).
type builtinGuard struct {
	typ        string
	guards     []Guard
	stateValue StateValue
	check      func(snap anyMachineSnapshot, a guardArgs, g *builtinGuard) bool
}

func (g *builtinGuard) GuardType() string { return g.typ }

// And mirrors and([...guards]).
func And(guards ...Guard) Guard {
	return &builtinGuard{typ: "xstate.and", guards: guards, check: func(snap anyMachineSnapshot, a guardArgs, g *builtinGuard) bool {
		for _, sub := range g.guards {
			if !evaluateGuard(sub, a.context, a.event, snap) {
				return false
			}
		}
		return true
	}}
}

// Or mirrors or([...guards]).
func Or(guards ...Guard) Guard {
	return &builtinGuard{typ: "xstate.or", guards: guards, check: func(snap anyMachineSnapshot, a guardArgs, g *builtinGuard) bool {
		for _, sub := range g.guards {
			if evaluateGuard(sub, a.context, a.event, snap) {
				return true
			}
		}
		return false
	}}
}

// Not mirrors not(guard).
func Not(guard Guard) Guard {
	return &builtinGuard{typ: "xstate.not", guards: []Guard{guard}, check: func(snap anyMachineSnapshot, a guardArgs, g *builtinGuard) bool {
		return !evaluateGuard(g.guards[0], a.context, a.event, snap)
	}}
}

// StateIn mirrors stateIn(stateValue). value: "#id", "a.b" or a state value map.
func StateIn(value StateValue) Guard {
	return &builtinGuard{typ: "xstate.stateIn", stateValue: value, check: func(snap anyMachineSnapshot, _ guardArgs, g *builtinGuard) bool {
		if s, ok := g.stateValue.(string); ok && isStateID(s) {
			target := snap.machineAny().getStateNodeByID(s)
			for _, n := range snap.nodesAny() {
				if n == target {
					return true
				}
			}
			return false
		}
		return matchesState(g.stateValue, snap.valueAny())
	}}
}

// evaluateGuard mirrors guards.ts evaluateGuard.
func evaluateGuard(guard Guard, context any, event Event, snap anyMachineSnapshot) bool {
	args := guardArgs{context: context, event: event}
	switch g := guard.(type) {
	case *inlineGuard:
		return g.fn(args, nil)
	case *builtinGuard:
		return g.check(snap, args, g)
	}
	var typ string
	var params any
	switch g := guard.(type) {
	case GuardRef:
		typ, params = g.Type, g.Params
	case *GuardRef:
		typ, params = g.Type, g.Params
	default:
		typ = guard.GuardType()
	}
	resolved := snap.machineAny().impl().Guards[typ]
	if resolved == nil {
		panic(errors.New("Guard '" + typ + "' is not implemented.'."))
	}
	switch r := resolved.(type) {
	case *inlineGuard:
		return r.fn(args, resolveParams(params, context, event, nil))
	case *builtinGuard:
		return r.check(snap, args, r)
	}
	return evaluateGuard(resolved, context, event, snap)
}
