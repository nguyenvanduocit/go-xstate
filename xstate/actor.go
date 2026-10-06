package xstate

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync/atomic"
)

type processingStatus int

const (
	statusNotStarted processingStatus = iota
	statusRunning
	statusStopped
)

type observerEntry struct {
	next     func(Snapshot)
	err      func(any)
	complete func()
	removed  bool
}

type listenerEntry struct {
	fn      func(Event)
	removed bool
}

// actorCore is the type-erased implementation of Actor (createActor.ts).
type actorCore struct {
	self      ActorRef
	logic     logicImpl
	origLogic ActorLogic
	withError func(prev Snapshot, err any) Snapshot

	snapshot         Snapshot
	clock            Clock
	id               string
	sessionID        string
	systemID         string
	mailbox          *mailbox
	observers        []*observerEntry
	listeners        map[string][]*listenerEntry
	logger           func(args ...any)
	processingStatus processingStatus
	parent           ActorRef
	syncSnapshot     bool
	scope            *ActorScope
	system           *ActorSystem
	deferred         []func()
	src              any
	input            any
	logicState       any
}

// Actor mirrors Actor in JS, typed by its snapshot.
type Actor[S Snapshot] struct {
	actorCore
}

type coreHolder interface {
	core() *actorCore
}

func (a *Actor[S]) core() *actorCore { return &a.actorCore }

func coreOf(ref ActorRef) *actorCore {
	if h, ok := ref.(coreHolder); ok && !isNilRef(ref) {
		return h.core()
	}
	return nil
}

func actorParent(ref ActorRef) ActorRef {
	if c := coreOf(ref); c != nil {
		return c.parent
	}
	if ref == nil {
		return nil
	}
	return ref.Parent()
}

func startRef(ref ActorRef) {
	if c := coreOf(ref); c != nil {
		c.start()
	}
}

var sessionCounter atomic.Int64

// CreateActor mirrors createActor(logic, options).
func CreateActor[S Snapshot](logic TypedActorLogic[S], opts ...ActorOption) *Actor[S] {
	var o actorOptions
	for _, opt := range opts {
		opt(&o)
	}
	li, ok := any(logic).(logicImpl)
	if !ok {
		panic(fmt.Errorf("xstate: unsupported actor logic %T", logic))
	}
	return newActor[S](li, logic, o)
}

func newActor[S Snapshot](li logicImpl, orig ActorLogic, o actorOptions) *Actor[S] {
	a := &Actor[S]{}
	c := &a.actorCore
	c.self = a
	c.logic = li
	c.origLogic = orig
	c.withError = func(prev Snapshot, err any) Snapshot {
		p, _ := prev.(S)
		return snapshotWithError[S](p, err)
	}
	if o.parent != nil {
		c.system = o.parent.System()
	} else {
		clock := o.clock
		if clock == nil {
			clock = defaultClock{}
		}
		logger := o.logger
		if logger == nil {
			logger = defaultLogger
		}
		c.system = newActorSystem(c, clock, logger)
		c.system.onWarn = o.onWarn
		c.system.onUnhandledError = o.onUnhandledError
	}
	sys := c.system
	sys.lock()
	defer sys.unlock()

	if o.parent == nil {
		if o.inspect != nil {
			sys.Inspect(o.inspect)
		}
		if o.inspectObserver != nil {
			sys.InspectObserver(*o.inspectObserver)
		}
	}
	c.sessionID = sys.bookID()
	c.id = o.id
	if c.id == "" {
		c.id = c.sessionID
	}
	c.logger = o.logger
	if c.logger == nil {
		c.logger = sys.logger
	}
	c.clock = o.clock
	if c.clock == nil {
		c.clock = sys.clock
	}
	c.parent = o.parent
	c.syncSnapshot = o.syncSnapshot
	c.src = o.src
	if c.src == nil {
		c.src = orig
	}
	c.input = o.input
	c.mailbox = newMailbox(c.process)
	c.scope = &ActorScope{
		Self:      a,
		ID:        c.id,
		SessionID: c.sessionID,
		System:    sys,
		Logger:    func(args ...any) { c.logger(args...) },
		Defer:     func(fn func()) { c.deferred = append(c.deferred, fn) },
		Emit:      c.emit,
		StopChild: c.stopChild,
	}
	c.scope.actionExecutor = c.executeAction
	sys.sendInspectionEvent(InspectionEvent{Type: InspectActor, ActorRef: a})
	if o.systemID != "" {
		c.systemID = o.systemID
		sys.set(o.systemID, a)
	}
	c.initState(o.snapshot)
	if o.systemID != "" && c.snapshot.GetStatus() != StatusActive {
		sys.unregister(a)
	}
	return a
}

// childOptions are the options used when the engine creates child actors.
type childOptions struct {
	id           string
	src          any
	parent       ActorRef
	syncSnapshot bool
	systemID     string
	input        any
	snapshot     any
}

func createChildActor(logic ActorLogic, co childOptions) ActorRef {
	li, ok := logic.(logicImpl)
	if !ok {
		panic(fmt.Errorf("xstate: unsupported actor logic %T", logic))
	}
	return li.newActorRef(actorOptions{
		id:           co.id,
		src:          co.src,
		parent:       co.parent,
		syncSnapshot: co.syncSnapshot,
		systemID:     co.systemID,
		input:        co.input,
		snapshot:     co.snapshot,
	})
}

func (c *actorCore) initState(persisted any) {
	defer func() {
		if r := recover(); r != nil {
			c.snapshot = c.withError(nil, r)
		}
	}()
	if persisted != nil {
		if c.logic.hasRestore() {
			c.snapshot = c.logic.restoreAny(persisted, c.scope)
		} else {
			c.snapshot = persisted.(Snapshot)
		}
		return
	}
	c.snapshot = c.logic.initialSnapshot(c.scope, c.input)
}

func (c *actorCore) executeAction(action ExecutableAction) {
	exec := func() {
		c.system.sendInspectionEvent(InspectionEvent{
			Type:     InspectAction,
			ActorRef: c.self,
			Action:   &InspectedAction{Type: action.Type, Params: action.Params},
		})
		if action.Exec == nil {
			return
		}
		prev := setExecutingCustomAction(true)
		defer setExecutingCustomAction(prev)
		action.Exec()
	}
	if c.processingStatus == statusRunning {
		exec()
	} else {
		c.deferred = append(c.deferred, exec)
	}
}

func (c *actorCore) emit(ev Event) {
	if ev == nil {
		return
	}
	listeners := c.listeners[ev.EventType()]
	wild := c.listeners[wildcard]
	if len(listeners) == 0 && len(wild) == 0 {
		return
	}
	all := append(append([]*listenerEntry{}, listeners...), wild...)
	for _, l := range all {
		if l.removed {
			continue
		}
		if r := safeCall(func() { l.fn(ev) }); r != nil {
			c.system.reportUnhandledError(r.value)
		}
	}
}

func (c *actorCore) stopChild(child ActorRef) {
	cc := coreOf(child)
	if cc == nil || cc.parent != c.self {
		panic(fmt.Errorf("Cannot stop child actor %s of %s because it is not a child", child.ID(), c.id))
	}
	cc.stop()
}

type panicBox struct{ value any }

func safeCall(fn func()) (caught *panicBox) {
	defer func() {
		if r := recover(); r != nil {
			caught = &panicBox{r}
		}
	}()
	fn()
	return nil
}

func (c *actorCore) update(snapshot Snapshot, event Event) {
	c.snapshot = snapshot
	for len(c.deferred) > 0 {
		fn := c.deferred[0]
		c.deferred = c.deferred[1:]
		if r := safeCall(fn); r != nil {
			c.deferred = nil
			c.snapshot = c.withError(snapshot, r.value)
		}
	}
	switch c.snapshot.GetStatus() {
	case StatusActive:
		c.notifyNext(snapshot)
	case StatusDone:
		c.notifyNext(snapshot)
		c.stopProcedure()
		c.complete()
		done := DoneActorEvent{ActorID: c.id, Output: c.snapshot.GetOutput()}
		if c.parent != nil {
			c.system.relay(c.self, c.parent, done)
		}
	case StatusError:
		c.errorFn(c.snapshot.GetError())
	}
	c.system.sendInspectionEvent(InspectionEvent{Type: InspectSnapshot, ActorRef: c.self, Event: event, Snapshot: snapshot})
}

func (c *actorCore) notifyNext(snapshot Snapshot) {
	for _, o := range append([]*observerEntry{}, c.observers...) {
		if o.removed || o.next == nil {
			continue
		}
		if r := safeCall(func() { o.next(snapshot) }); r != nil {
			c.system.reportUnhandledError(r.value)
		}
	}
}

func (c *actorCore) subscribe(o *observerEntry) Subscription {
	if c.processingStatus != statusStopped {
		c.observers = append(c.observers, o)
	} else {
		switch c.snapshot.GetStatus() {
		case StatusDone:
			if o.complete != nil {
				if r := safeCall(o.complete); r != nil {
					c.system.reportUnhandledError(r.value)
				}
			}
		case StatusError:
			err := c.snapshot.GetError()
			if o.err == nil {
				c.system.reportUnhandledError(err)
			} else if r := safeCall(func() { o.err(err) }); r != nil {
				c.system.reportUnhandledError(r.value)
			}
		}
	}
	sys := c.system
	return SubscriptionFunc(func() {
		sys.lock()
		defer sys.unlock()
		o.removed = true
		for i, e := range c.observers {
			if e == o {
				c.observers = append(c.observers[:i:i], c.observers[i+1:]...)
				break
			}
		}
	})
}

func (c *actorCore) on(eventType string, handler func(Event)) Subscription {
	if c.listeners == nil {
		c.listeners = map[string][]*listenerEntry{}
	}
	l := &listenerEntry{fn: handler}
	c.listeners[eventType] = append(c.listeners[eventType], l)
	sys := c.system
	return SubscriptionFunc(func() {
		sys.lock()
		defer sys.unlock()
		l.removed = true
		list := c.listeners[eventType]
		for i, e := range list {
			if e == l {
				c.listeners[eventType] = append(list[:i:i], list[i+1:]...)
				break
			}
		}
	})
}

func (c *actorCore) start() {
	if c.processingStatus == statusRunning {
		return
	}
	if c.syncSnapshot {
		self := c.self
		c.subscribe(&observerEntry{
			next: func(s Snapshot) {
				if s.GetStatus() == StatusActive {
					c.system.relay(self, c.parent, SnapshotEvent{ActorID: c.id, Snapshot: s})
				}
			},
			err: func(any) {},
		})
	}
	c.system.register(c.sessionID, c.self)
	if c.systemID != "" {
		c.system.set(c.systemID, c.self)
	}
	c.processingStatus = statusRunning
	initEvent := InitEvent{Input: c.input}
	c.system.sendInspectionEvent(InspectionEvent{Type: InspectEvent, SourceRef: c.parent, ActorRef: c.self, Event: initEvent})

	switch c.snapshot.GetStatus() {
	case StatusDone:
		c.update(c.snapshot, initEvent)
		return
	case StatusError:
		c.errorFn(c.snapshot.GetError())
		return
	}
	if c.parent == nil {
		c.system.start()
	}
	if c.logic.hasStart() {
		if r := safeCall(func() { c.logic.startAny(c.snapshot, c.scope) }); r != nil {
			c.snapshot = c.withError(c.snapshot, r.value)
			c.errorFn(r.value)
			return
		}
	}
	c.update(c.snapshot, initEvent)
	c.mailbox.start()
}

func (c *actorCore) process(event Event) {
	var next Snapshot
	r := safeCall(func() { next = c.logic.transitionAny(c.snapshot, event, c.scope) })
	if r != nil {
		c.snapshot = c.withError(c.snapshot, r.value)
		c.errorFn(r.value)
		return
	}
	c.update(next, event)
	if event.EventType() == xstateStop {
		c.stopProcedure()
		c.complete()
	}
}

func (c *actorCore) stop() {
	if c.processingStatus == statusStopped {
		return
	}
	c.mailbox.clear()
	if c.processingStatus == statusNotStarted {
		c.processingStatus = statusStopped
		return
	}
	c.mailbox.enqueue(StopEvent{})
}

func (c *actorCore) complete() {
	for _, o := range append([]*observerEntry{}, c.observers...) {
		if o.removed || o.complete == nil {
			continue
		}
		if r := safeCall(o.complete); r != nil {
			c.system.reportUnhandledError(r.value)
		}
	}
	c.observers = nil
	c.listeners = nil
}

func (c *actorCore) reportError(err any) {
	if len(c.observers) == 0 {
		if c.parent == nil {
			c.system.reportUnhandledError(err)
		}
		c.listeners = nil
		return
	}
	report := false
	for _, o := range append([]*observerEntry{}, c.observers...) {
		if o.removed {
			continue
		}
		if o.err == nil {
			report = true
			continue
		}
		if r := safeCall(func() { o.err(err) }); r != nil {
			c.system.reportUnhandledError(r.value)
		}
	}
	c.observers = nil
	c.listeners = nil
	if report {
		c.system.reportUnhandledError(err)
	}
}

func (c *actorCore) errorFn(err any) {
	c.stopProcedure()
	c.reportError(err)
	if c.parent != nil {
		c.system.relay(c.self, c.parent, ErrorActorEvent{ActorID: c.id, Error: err})
	}
}

func (c *actorCore) stopProcedure() {
	if c.processingStatus != statusRunning {
		return
	}
	c.system.scheduler.cancelAll(c.self)
	c.mailbox.clear()
	c.mailbox = newMailbox(c.process)
	c.processingStatus = statusStopped
	c.system.unregister(c.self)
}

func (c *actorCore) send(event Event) {
	if c.processingStatus == statusStopped {
		eventString, ok := jsonStringify(event)
		if !ok {
			eventString = "[object Object]"
		}
		typ := ""
		if event != nil {
			typ = event.EventType()
		}
		warn(fmt.Sprintf("Event \"%s\" was sent to stopped actor \"%s (%s)\". This actor has already reached its final state, and will not transition.\nEvent: %s", typ, c.id, c.sessionID, eventString))
		return
	}
	c.mailbox.enqueue(event)
}

// Start mirrors actor.start().
func (a *Actor[S]) Start() *Actor[S] {
	a.system.lock()
	defer a.system.unlock()
	a.start()
	return a
}

// Stop mirrors actor.stop().
func (a *Actor[S]) Stop() *Actor[S] {
	if a.parent != nil {
		panic(errors.New("A non-root actor cannot be stopped directly."))
	}
	a.system.lock()
	defer a.system.unlock()
	a.stop()
	return a
}

// Send mirrors actor.send(event).
func (a *Actor[S]) Send(event Event) {
	a.system.lock()
	defer a.system.unlock()
	a.system.relay(nil, a, event)
}

// GetSnapshot mirrors actor.getSnapshot().
func (a *Actor[S]) GetSnapshot() S {
	a.system.lock()
	defer a.system.unlock()
	s, _ := a.snapshot.(S)
	return s
}

// GetPersistedSnapshot mirrors actor.getPersistedSnapshot().
func (a *Actor[S]) GetPersistedSnapshot() any {
	a.system.lock()
	defer a.system.unlock()
	return a.logic.persistAny(a.snapshot)
}

func (a *Actor[S]) ID() string { return a.id }

func (a *Actor[S]) SessionID() string { return a.sessionID }

func (a *Actor[S]) System() *ActorSystem { return a.system }

func (a *Actor[S]) Src() any { return a.src }

func (a *Actor[S]) Parent() ActorRef { return a.parent }

func (a *Actor[S]) Clock() Clock { return a.clock }

// SystemID mirrors actor.systemId.
func (a *Actor[S]) SystemID() string { return a.systemID }

// Logic returns the actor logic.
func (a *Actor[S]) Logic() TypedActorLogic[S] {
	l, _ := a.origLogic.(TypedActorLogic[S])
	return l
}

// AnySnapshot is the type-erased GetSnapshot.
func (a *Actor[S]) AnySnapshot() Snapshot {
	a.system.lock()
	defer a.system.unlock()
	return a.snapshot
}

// Subscribe mirrors actor.subscribe(observer).
func (a *Actor[S]) Subscribe(observer Observer[S]) Subscription {
	o := &observerEntry{err: observer.Error, complete: observer.Complete}
	if observer.Next != nil {
		next := observer.Next
		o.next = func(s Snapshot) { v, _ := s.(S); next(v) }
	}
	a.system.lock()
	defer a.system.unlock()
	return a.subscribe(o)
}

// SubscribeNext mirrors actor.subscribe(fn).
func (a *Actor[S]) SubscribeNext(next func(S)) Subscription {
	return a.Subscribe(Observer[S]{Next: next})
}

// SubscribeAny is the type-erased Subscribe.
func (a *Actor[S]) SubscribeAny(observer Observer[Snapshot]) Subscription {
	a.system.lock()
	defer a.system.unlock()
	return a.subscribe(&observerEntry{next: observer.Next, err: observer.Error, complete: observer.Complete})
}

// On mirrors actor.on(eventType, handler) for emitted events. "*" matches all.
func (a *Actor[S]) On(eventType string, handler func(Event)) Subscription {
	a.system.lock()
	defer a.system.unlock()
	return a.on(eventType, handler)
}

// ToJSON mirrors actor.toJSON().
func (a *Actor[S]) ToJSON() map[string]any {
	return map[string]any{"xstate$$type": 1, "id": a.id}
}

// MarshalJSON serializes the actor like JS toJSON().
func (a *Actor[S]) MarshalJSON() ([]byte, error) { return json.Marshal(a.ToJSON()) }

// As recovers a typed actor from an ActorRef. It panics on a type mismatch.
func As[S Snapshot](ref ActorRef) *Actor[S] {
	a, ok := ref.(*Actor[S])
	if !ok {
		panic(fmt.Errorf("xstate: actor %v is %T, not *Actor[%v]", refID(ref), ref, typeName[S]()))
	}
	return a
}

func refID(ref ActorRef) string {
	if isNilRef(ref) {
		return "<nil>"
	}
	return ref.ID()
}

func typeName[T any]() string {
	var z T
	return fmt.Sprintf("%T", z)
}
