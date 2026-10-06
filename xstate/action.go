package xstate

import (
	"errors"
	"fmt"
	"math/rand"
	"reflect"
	"strconv"
	"time"
)

// Action is an executable or referenced action. Construct with ActionFunc,
// ActionRef, Assign, Raise, SendTo, ... .
type Action interface {
	ActionType() string
}

// ActionArgs are passed to inline action functions.
type ActionArgs[C any] struct {
	Context C
	Event   Event
	Self    ActorRef
	System  *ActorSystem
	Params  any
}

// ExprArgs are passed to Expr functions (dynamic values: input, output,
// params, send targets, delays, ids, ...).
type ExprArgs[C any] struct {
	Context C
	Event   Event
	Self    ActorRef
	System  *ActorSystem
	Params  any
}

// Expr wraps a function that computes a dynamic value. It is accepted
// everywhere the JS API accepts "value or function".
type Expr interface {
	isExpr()
}

// exprArgs is the type-erased ExprArgs.
type exprArgs struct {
	context any
	event   Event
	self    ActorRef
	system  *ActorSystem
	params  any
}

type exprFunc struct {
	fn func(exprArgs) any
}

func (*exprFunc) isExpr() {}

// NewExpr creates an Expr. The context type C must match the machine.
func NewExpr[C any](fn func(ExprArgs[C]) any) Expr {
	return &exprFunc{fn: func(a exprArgs) any {
		return fn(ExprArgs[C]{Context: castTo[C](a.context), Event: a.event, Self: a.self, System: a.system, Params: a.params})
	}}
}

func evalExpr(v any, a exprArgs) any {
	if e, ok := v.(*exprFunc); ok {
		return e.fn(a)
	}
	return v
}

// castTo converts a type-erased value to T (nil becomes the zero T).
func castTo[T any](v any) T {
	if v == nil {
		var z T
		return z
	}
	if t, ok := v.(T); ok {
		return t
	}
	rv := reflect.ValueOf(v)
	tt := reflect.TypeFor[T]()
	if rv.Type().ConvertibleTo(tt) && rv.Kind() == tt.Kind() {
		return rv.Convert(tt).Interface().(T)
	}
	panic(fmt.Errorf("xstate: context of type %T is not assignable to %v", v, tt))
}

// actionArgs is the type-erased ActionArgs.
type actionArgs struct {
	context any
	event   Event
	self    ActorRef
	system  *ActorSystem
}

func (a actionArgs) public() ActionArgs[any] {
	return ActionArgs[any]{Context: a.context, Event: a.event, Self: a.self, System: a.system}
}

func (a actionArgs) expr(params any) exprArgs {
	return exprArgs{context: a.context, event: a.event, self: a.self, system: a.system, params: params}
}

type inlineAction struct {
	fn func(a actionArgs, params any)
}

func (*inlineAction) ActionType() string { return "" }

// ActionFunc creates an inline action, mirroring `entry: ({ context }) => ...`.
func ActionFunc[C any](fn func(ActionArgs[C])) Action {
	return &inlineAction{fn: func(a actionArgs, params any) {
		fn(ActionArgs[C]{Context: castTo[C](a.context), Event: a.event, Self: a.self, System: a.system, Params: params})
	}}
}

// ActionRef references an action by name, mirroring `'name'` or
// `{ type: 'name', params }`. Params may be a static value or an Expr.
type ActionRef struct {
	Type   string
	Params any
}

func (a ActionRef) ActionType() string { return a.Type }

// builtinAction mirrors the JS BuiltinAction (resolve / retryResolve /
// execute).
type builtinAction struct {
	typ          string
	resolve      func(scope *ActorScope, snap anyMachineSnapshot, args actionArgs, params any, extra *resolveExtra) (anyMachineSnapshot, any, Actions)
	retryResolve func(scope *ActorScope, snap anyMachineSnapshot, params any)
	execute      func(scope *ActorScope, params any)

	delay    any
	hasDelay bool
}

func (b *builtinAction) ActionType() string { return b.typ }

// ExecutableAction mirrors an action returned by transition()/initialTransition().
type ExecutableAction struct {
	Type   string
	Params any
	Info   ActionArgs[any]
	Exec   func()
}

func warnCustomAction(name string) {
	if isExecutingCustomAction() {
		warn("Custom actions should not call `" + name + "()` directly, as it is not imperative. See https://stately.ai/docs/actions#built-in-actions for more details.")
	}
}

// AssignArgs are passed to Assign functions.
type AssignArgs[C any] struct {
	Context C
	Event   Event
	Self    ActorRef
	System  *ActorSystem
	Params  any
	Spawn   Spawner
}

// Assign mirrors assign(); fn returns the complete next context.
func Assign[C any](fn func(AssignArgs[C]) C) Action {
	warnCustomAction("assign")
	return &builtinAction{
		typ: "xstate.assign",
		resolve: func(scope *ActorScope, snap anyMachineSnapshot, args actionArgs, params any, _ *resolveExtra) (anyMachineSnapshot, any, Actions) {
			spawned := &childSet{m: map[string]ActorRef{}}
			spawner := createSpawner(scope, snap, args.event, spawned)
			next := fn(AssignArgs[C]{
				Context: castTo[C](snap.contextAny()),
				Event:   args.event,
				Self:    scope.Self,
				System:  scope.System,
				Params:  params,
				Spawn:   spawner,
			})
			patch := snapshotPatch{context: anyPtr(next)}
			if len(spawned.order) > 0 {
				cs := snap.children()
				for _, id := range spawned.order {
					cs = cs.with(id, spawned.m[id])
				}
				patch.children = cs
			}
			return snap.clone(patch), nil, nil
		},
	}
}

// resolveDelay mirrors the delay resolution of raise/sendTo: a delay name is
// looked up in the machine delays; ok is false when the delay is not a number.
func resolveDelay(delay any, snap anyMachineSnapshot, a exprArgs) (time.Duration, bool) {
	if name, isName := delay.(string); isName {
		cfg, found := snap.machineAny().impl().Delays[name]
		if !found {
			return 0, false
		}
		return toDuration(evalExpr(cfg, a))
	}
	return toDuration(evalExpr(delay, a))
}

func toDuration(v any) (time.Duration, bool) {
	switch d := v.(type) {
	case time.Duration:
		return d, true
	case int:
		return time.Duration(d) * time.Millisecond, true
	case int64:
		return time.Duration(d) * time.Millisecond, true
	case float64:
		return time.Duration(d * float64(time.Millisecond)), true
	}
	return 0, false
}

func resolveEventArg(event any, a exprArgs, kind string) Event {
	if s, ok := event.(string); ok {
		panic(fmt.Errorf(`Only event objects may be used with %s; use %s({ type: "%s" }) instead`, kind, kind, s))
	}
	resolved := evalExpr(event, a)
	if resolved == nil {
		return nil
	}
	if ev, ok := resolved.(Event); ok {
		return ev
	}
	if m, ok := resolved.(map[string]any); ok {
		return E(m)
	}
	panic(fmt.Errorf("xstate: %T is not an event", resolved))
}

func resolveSendID(id any, a exprArgs) any {
	v := evalExpr(id, a)
	if v == nil {
		return nil
	}
	return v
}

func optsOf(opts []SendOptions) SendOptions {
	if len(opts) > 0 {
		return opts[0]
	}
	return SendOptions{}
}

// Raise mirrors raise(event, options). event is an Event or an Expr.
func Raise(event any, opts ...SendOptions) Action {
	warnCustomAction("raise")
	o := optsOf(opts)
	return &builtinAction{
		typ:      "xstate.raise",
		delay:    o.Delay,
		hasDelay: o.Delay != nil,
		resolve: func(scope *ActorScope, snap anyMachineSnapshot, args actionArgs, params any, extra *resolveExtra) (anyMachineSnapshot, any, Actions) {
			ea := args.expr(params)
			resolved := resolveEventArg(event, ea, "raise")
			delay, hasDelay := resolveDelay(o.Delay, snap, ea)
			if !hasDelay {
				*extra.internalQueue = append(*extra.internalQueue, resolved)
			}
			p := map[string]any{"event": resolved, "id": resolveSendID(o.ID, ea), "delay": nil}
			if hasDelay {
				p["delay"] = delay
			}
			return snap, p, nil
		},
		execute: func(scope *ActorScope, params any) {
			p := params.(map[string]any)
			delay, ok := p["delay"].(time.Duration)
			if !ok {
				return
			}
			event := p["event"].(Event)
			id, _ := p["id"].(string)
			scope.Defer(func() {
				self := scope.Self
				scope.System.Scheduler().Schedule(self, self, event, delay, id)
			})
		},
	}
}

// SendTo mirrors sendTo(target, event, options).
// target: actor id (string), ActorRef, or Expr returning ActorRef/string.
// event: Event or Expr returning Event.
func SendTo(target any, event any, opts ...SendOptions) Action {
	warnCustomAction("sendTo")
	return sendTo(target, event, optsOf(opts))
}

func sendTo(target any, event any, o SendOptions) *builtinAction {
	return &builtinAction{
		typ:      "xstate.sendTo",
		delay:    o.Delay,
		hasDelay: o.Delay != nil,
		resolve: func(scope *ActorScope, snap anyMachineSnapshot, args actionArgs, params any, extra *resolveExtra) (anyMachineSnapshot, any, Actions) {
			ea := args.expr(params)
			resolvedEvent := resolveEventArg(event, ea, "sendTo")
			delay, hasDelay := resolveDelay(o.Delay, snap, ea)
			resolvedTarget := evalExpr(target, ea)
			var to any
			var targetID any
			if s, ok := resolvedTarget.(string); ok {
				targetID = s
				children := snap.children().m
				switch {
				case s == "#_parent":
					if p := actorParent(scope.Self); p != nil {
						to = p
					}
				case s == "#_internal":
					to = scope.Self
				case len(s) > 2 && s[:2] == "#_":
					if c := children[s[2:]]; c != nil {
						to = c
					}
				default:
					if containsString(extra.deferredActorIDs, s) {
						to = s
					} else if c := children[s]; c != nil {
						to = c
					}
				}
				if to == nil {
					panic(fmt.Errorf("Unable to send event to actor '%s' from machine '%s'.", s, snap.machineAny().MachineID()))
				}
			} else if ref, ok := resolvedTarget.(ActorRef); ok && !isNilRef(ref) {
				to = ref
			} else {
				to = scope.Self
			}
			p := map[string]any{"to": to, "targetId": targetID, "event": resolvedEvent, "id": resolveSendID(o.ID, ea), "delay": nil}
			if hasDelay {
				p["delay"] = delay
			}
			return snap, p, nil
		},
		retryResolve: func(scope *ActorScope, snap anyMachineSnapshot, params any) {
			p := params.(map[string]any)
			if s, ok := p["to"].(string); ok {
				p["to"] = snap.children().m[s]
			}
		},
		execute: func(scope *ActorScope, params any) {
			p := params.(map[string]any)
			scope.Defer(func() {
				to, _ := p["to"].(ActorRef)
				event, _ := p["event"].(Event)
				id, _ := p["id"].(string)
				if delay, ok := p["delay"].(time.Duration); ok {
					scope.System.Scheduler().Schedule(scope.Self, to, event, delay, id)
					return
				}
				if event != nil && event.EventType() == xstateError {
					event = ErrorActorEvent{ActorID: scope.Self.ID(), Error: eventField(event, "data")}
				}
				scope.System.relay(scope.Self, to, event)
			})
		},
	}
}

func containsString(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func isNilRef(ref ActorRef) bool {
	if ref == nil {
		return true
	}
	rv := reflect.ValueOf(ref)
	return rv.Kind() == reflect.Pointer && rv.IsNil()
}

// SendParent mirrors sendParent(event, options).
func SendParent(event any, opts ...SendOptions) Action {
	warnCustomAction("sendTo")
	return sendTo("#_parent", event, optsOf(opts))
}

// ForwardTo mirrors forwardTo(target, options).
func ForwardTo(target any, opts ...SendOptions) Action {
	resolvedTarget := target
	_, isExpr := target.(*exprFunc)
	if target == nil || isExpr {
		original := target
		resolvedTarget = &exprFunc{fn: func(a exprArgs) any {
			r := evalExpr(original, a)
			if r == nil {
				panic(errors.New("Attempted to forward event to undefined actor. This risks an infinite loop in the sender."))
			}
			if ref, ok := r.(ActorRef); ok && isNilRef(ref) {
				panic(errors.New("Attempted to forward event to undefined actor. This risks an infinite loop in the sender."))
			}
			return r
		}}
	}
	warnCustomAction("sendTo")
	return sendTo(resolvedTarget, &exprFunc{fn: func(a exprArgs) any { return a.event }}, optsOf(opts))
}

// Log mirrors log(expr, label). expr: string/value or Expr; nil logs the
// default `{ context, event }`.
func Log(expr any, label ...string) Action {
	var l string
	if len(label) > 0 {
		l = label[0]
	}
	if expr == nil {
		expr = &exprFunc{fn: func(a exprArgs) any {
			return map[string]any{"context": a.context, "event": a.event}
		}}
	}
	return &builtinAction{
		typ: "xstate.log",
		resolve: func(scope *ActorScope, snap anyMachineSnapshot, args actionArgs, params any, _ *resolveExtra) (anyMachineSnapshot, any, Actions) {
			var lv any
			if l != "" {
				lv = l
			}
			return snap, map[string]any{"value": evalExpr(expr, args.expr(params)), "label": lv}, nil
		},
		execute: func(scope *ActorScope, params any) {
			p := params.(map[string]any)
			if l, ok := p["label"].(string); ok && l != "" {
				scope.Logger(l, p["value"])
			} else {
				scope.Logger(p["value"])
			}
		},
	}
}

// Cancel mirrors cancel(sendId). sendID: string or Expr.
func Cancel(sendID any) Action {
	return &builtinAction{
		typ: "xstate.cancel",
		resolve: func(scope *ActorScope, snap anyMachineSnapshot, args actionArgs, params any, _ *resolveExtra) (anyMachineSnapshot, any, Actions) {
			return snap, map[string]any{"sendId": evalExpr(sendID, args.expr(params))}, nil
		},
		execute: func(scope *ActorScope, params any) {
			id, _ := params.(map[string]any)["sendId"].(string)
			scope.Defer(func() {
				scope.System.Scheduler().Cancel(scope.Self, id)
			})
		},
	}
}

// StopChild mirrors stopChild(actorRef). ref: id string, ActorRef or Expr.
func StopChild(ref any) Action {
	return &builtinAction{
		typ: "xstate.stopChild",
		resolve: func(scope *ActorScope, snap anyMachineSnapshot, args actionArgs, params any, _ *resolveExtra) (anyMachineSnapshot, any, Actions) {
			v := evalExpr(ref, args.expr(params))
			var resolved ActorRef
			switch r := v.(type) {
			case string:
				resolved = snap.children().m[r]
			case ActorRef:
				if !isNilRef(r) {
					resolved = r
				}
			}
			cs := snap.children()
			if resolved != nil {
				cs = cs.without(resolved.ID())
			}
			var p any
			if resolved != nil {
				p = resolved
			}
			return snap.clone(snapshotPatch{children: cs}), p, nil
		},
		execute: func(scope *ActorScope, params any) {
			ref, _ := params.(ActorRef)
			if ref == nil {
				return
			}
			unregisterRecursively(scope, ref)
			if c := coreOf(ref); c == nil || c.processingStatus != statusRunning {
				scope.StopChild(ref)
				return
			}
			scope.Defer(func() { scope.StopChild(ref) })
		},
	}
}

func unregisterRecursively(scope *ActorScope, ref ActorRef) {
	if snap, ok := ref.AnySnapshot().(anyMachineSnapshot); ok {
		cs := snap.children()
		for _, k := range cs.order {
			if child := cs.m[k]; child != nil {
				unregisterRecursively(scope, child)
			}
		}
	}
	scope.System.unregister(ref)
}

// SpawnChild mirrors spawnChild(src, options). src: actor name or ActorLogic.
func SpawnChild(src any, opts ...SpawnOptions) Action {
	var o SpawnOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	return spawnChildAction(src, o.ID, o.SystemID, o.Input, o.SyncSnapshot)
}

func spawnChildFromInvoke(def *InvokeDefinition) Action {
	return spawnChildAction(def.Src, def.ID, def.SystemID, def.Input, def.OnSnapshot != nil)
}

func spawnChildAction(src any, id any, systemID string, input any, syncSnapshot bool) *builtinAction {
	return &builtinAction{
		typ: "xstate.spawnChild",
		resolve: func(scope *ActorScope, snap anyMachineSnapshot, args actionArgs, _ any, _ *resolveExtra) (anyMachineSnapshot, any, Actions) {
			var logic ActorLogic
			switch s := src.(type) {
			case string:
				logic = snap.machineAny().resolveReferencedActor(s)
			case ActorLogic:
				logic = s
			}
			resolvedID := evalExpr(id, args.expr(nil))
			var actorRef ActorRef
			var resolvedInput any
			if logic != nil {
				resolvedInput = evalExpr(input, exprArgs{context: snap.contextAny(), event: args.event, self: scope.Self, system: scope.System})
				idStr, _ := resolvedID.(string)
				actorRef = createChildActor(logic, childOptions{
					id:           idStr,
					src:          src,
					parent:       scope.Self,
					syncSnapshot: syncSnapshot,
					systemID:     systemID,
					input:        resolvedInput,
				})
			} else {
				warn(fmt.Sprintf("Actor type '%v' not found in machine '%s'.", srcString(src), scope.ID))
			}
			key := "undefined"
			if s, ok := resolvedID.(string); ok {
				key = s
			} else if resolvedID != nil {
				key = fmt.Sprint(resolvedID)
			}
			p := map[string]any{"id": id, "systemId": nilIfEmpty(systemID), "actorRef": actorRef, "src": src, "input": resolvedInput}
			return snap.clone(snapshotPatch{children: snap.children().with(key, actorRef)}), p, nil
		},
		execute: func(scope *ActorScope, params any) {
			ref, _ := params.(map[string]any)["actorRef"].(ActorRef)
			if ref == nil {
				return
			}
			scope.Defer(func() {
				c := coreOf(ref)
				if c != nil && c.processingStatus == statusStopped {
					return
				}
				startRef(ref)
			})
		},
	}
}

func srcString(src any) string {
	if s, ok := src.(string); ok {
		return s
	}
	return fmt.Sprint(src)
}

func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// Emit mirrors emit(event). event: Event or Expr.
func Emit(event any) Action {
	warnCustomAction("emit")
	return &builtinAction{
		typ: "xstate.emit",
		resolve: func(scope *ActorScope, snap anyMachineSnapshot, args actionArgs, params any, _ *resolveExtra) (anyMachineSnapshot, any, Actions) {
			resolved := evalExpr(event, args.expr(params))
			ev, ok := resolved.(Event)
			if !ok {
				if m, isMap := resolved.(map[string]any); isMap {
					ev = E(m)
				}
			}
			return snap, map[string]any{"event": ev}, nil
		},
		execute: func(scope *ActorScope, params any) {
			ev, _ := params.(map[string]any)["event"].(Event)
			scope.Defer(func() { scope.Emit(ev) })
		},
	}
}

// EnqueueArgs are passed to EnqueueActions functions.
type EnqueueArgs[C any] struct {
	Context C
	Event   Event
	Self    ActorRef
	System  *ActorSystem
	Params  any
	// Enqueue adds an action (any Action, including ActionRef) to the queue.
	Enqueue func(Action)
	// Check evaluates a guard (GuardFunc, GuardRef, And/Or/Not, StateIn).
	Check func(Guard) bool
}

// EnqueueActions mirrors enqueueActions(fn).
func EnqueueActions[C any](fn func(EnqueueArgs[C])) Action {
	return &builtinAction{
		typ: "xstate.enqueueActions",
		resolve: func(scope *ActorScope, snap anyMachineSnapshot, args actionArgs, params any, _ *resolveExtra) (anyMachineSnapshot, any, Actions) {
			actions := Actions{}
			fn(EnqueueArgs[C]{
				Context: castTo[C](args.context),
				Event:   args.event,
				Self:    scope.Self,
				System:  scope.System,
				Params:  params,
				Enqueue: func(a Action) { actions = append(actions, a) },
				Check: func(g Guard) bool {
					return evaluateGuard(g, snap.contextAny(), args.event, snap)
				},
			})
			return snap, nil, actions
		},
	}
}

func randomID() string {
	return strconv.FormatInt(rand.Int63(), 36)
}

// ActionDelay reports the delay of a built-in action created with a delay
// option.
func ActionDelay(action Action) (delay any, ok bool) {
	if b, isBuiltin := action.(*builtinAction); isBuiltin && b.hasDelay {
		return b.delay, true
	}
	return nil, false
}
