package store

import (
	"fmt"
	"maps"
	"math/rand/v2"
	"slices"
	"strconv"
	"strings"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// StoreSnapshot mirrors StoreSnapshot<TContext>. Snapshots are immutable:
// every committed change produces a new *StoreSnapshot.
type StoreSnapshot[C any] struct {
	Status  xs.Status
	Context C
	Output  any
	Error   any
	// ExtensionState holds state that extensions spread onto the JS snapshot
	// (undoRedo history, persist metadata, custom `{...snapshot, [key]: v}`),
	// keyed by name. Copy-on-write: never mutate a previous snapshot's map.
	ExtensionState map[string]any
}

func (s *StoreSnapshot[C]) GetStatus() xs.Status { return s.Status }
func (s *StoreSnapshot[C]) GetOutput() any       { return s.Output }
func (s *StoreSnapshot[C]) GetError() any        { return s.Error }

// spread mirrors `{ ...snapshot }`: a new snapshot sharing the extension map
// (copy-on-write).
func (s *StoreSnapshot[C]) spread() *StoreSnapshot[C] {
	next := *s
	return &next
}

// withContext mirrors `{ ...snapshot, context }`.
func (s *StoreSnapshot[C]) withContext(ctx C) *StoreSnapshot[C] {
	next := s.spread()
	next.Context = ctx
	return next
}

// withExt mirrors `{ ...snapshot, [key]: value, ... }` for extension keys;
// kv alternates key, value.
func (s *StoreSnapshot[C]) withExt(kv ...any) *StoreSnapshot[C] {
	next := s.spread()
	next.ExtensionState = make(map[string]any, len(s.ExtensionState)+len(kv)/2)
	maps.Copy(next.ExtensionState, s.ExtensionState)
	for i := 0; i+1 < len(kv); i += 2 {
		next.ExtensionState[kv[i].(string)] = kv[i+1]
	}
	return next
}

// ext reads an extension key (nil when absent).
func (s *StoreSnapshot[C]) ext(key string) any {
	if s == nil || s.ExtensionState == nil {
		return nil
	}
	return s.ExtensionState[key]
}

// StoreAssigner mirrors StoreAssigner: an `on` handler. The bool result is
// false when the JS handler returns undefined (no assignment); then the
// context is kept and the event counts as "not allowed" unless it enqueued
// effects, emits or triggers.
type StoreAssigner[C any] func(ctx C, event xs.Event, enq *EnqueueObject[C]) (C, bool)

// EnqueueObject mirrors the `enq` argument of an assigner.
type EnqueueObject[C any] struct {
	// Emit mirrors enq.emit[eventType](payload).
	Emit func(eventType string, payload ...xs.E)
	// Trigger mirrors enq.trigger[eventType](payload): the event is processed
	// in the same transition after the current handler.
	Trigger func(eventType string, payload ...xs.E)
	// Effect mirrors enq.effect(fn): fn runs after the transition commits.
	Effect func(fn func(enq *StoreEffectEnqueue[C]))
}

// StoreEffectEnqueue mirrors StoreEffectEnqueue: the argument of an effect.
type StoreEffectEnqueue[C any] struct {
	Trigger     func(eventType string, payload ...xs.E)
	Send        func(event xs.Event)
	GetSnapshot func() *StoreSnapshot[C]
}

// StoreEffect mirrors StoreEffect: exactly one of Emitted or Run is set.
type StoreEffect[C any] struct {
	// Emitted is an emitted event (JS: an event object in the effects array).
	Emitted xs.Event
	// Run is a side effect (JS: a function in the effects array). JS tests
	// that call `effect()` with no argument call Run(nil).
	Run func(enq *StoreEffectEnqueue[C])
}

// StoreSchemas mirrors StoreSchemas: Standard Schema validators for context,
// event payloads and emitted payloads (keyed by event type).
type StoreSchemas struct {
	Context Schema
	Events  map[string]Schema
	Emitted map[string]Schema
}

// StoreConfig mirrors the config of createStore / createStoreConfig /
// createStoreLogic / fromStore. Every constructor reads every field.
type StoreConfig[C any] struct {
	Context C
	// ContextFn mirrors `context: (input) => ...`; used instead of Context
	// when set. CreateStore calls it with nil input.
	ContextFn func(input any) C
	Schemas   *StoreSchemas
	On        map[string]StoreAssigner[C]
	// Selectors mirrors createStoreLogic `selectors`; resolved selections are
	// exposed by Store.Selectors.
	Selectors map[string]func(ctx C) any
}

// StoreTransitionResult mirrors StoreTransitionResult: [snapshot, effects]
// plus the `_allowed` marker.
type StoreTransitionResult[C any] struct {
	Snapshot *StoreSnapshot[C]
	Effects  []StoreEffect[C]
	// Allowed mirrors `_allowed`; nil means "infer": allowed when the snapshot
	// changed or effects is non-empty.
	Allowed *bool
}

// StoreTransition mirrors StoreTransition (the pure transition function).
type StoreTransition[C any] func(snapshot *StoreSnapshot[C], event xs.Event) StoreTransitionResult[C]

// StoreLogic mirrors StoreLogic: the input of createStore(logic) and the
// value extensions wrap. It is a value type so extensions can copy and
// override fields like the JS `{...logic, transition}` spread.
type StoreLogic[C any] struct {
	// EventTypes mirrors `eventTypes`; nil means any event type is accepted
	// by Trigger / Can.
	EventTypes         []string
	Schemas            *StoreSchemas
	GetInitialSnapshot func() *StoreSnapshot[C]
	Transition         StoreTransition[C]
}

// StoreExtension mirrors StoreExtension, the argument of store.with(ext).
type StoreExtension[C any] func(logic StoreLogic[C]) StoreLogic[C]

// ActorRefLike mirrors ActorRefLike in inspection events.
type ActorRefLike interface {
	SessionID() string
	Send(event xs.Event)
	AnySnapshot() xs.Snapshot
}

// StoreInspectionEvent mirrors StoreInspectionEvent. Type is always
// xs.InspectTransition ("@xstate.transition"); the first event delivered to a
// new inspector carries Event = xs.Ev("@xstate.init").
type StoreInspectionEvent struct {
	Type     string
	RootID   string
	ActorRef ActorRefLike
	Event    xs.Event
	Snapshot xs.Snapshot
}

// toEvent mirrors toEvent(eventType, payload): `{ ...payload, type }`.
func toEvent(eventType string, payload []xs.E) xs.E {
	ev := xs.E{}
	if len(payload) > 0 {
		maps.Copy(ev, payload[0])
	}
	ev["type"] = eventType
	return ev
}

// newEnqueueObject mirrors createEnqueueObject(effects, trigger).
func newEnqueueObject[C any](effects *[]StoreEffect[C], trigger func(event xs.Event)) *EnqueueObject[C] {
	return &EnqueueObject[C]{
		Emit: func(eventType string, payload ...xs.E) {
			*effects = append(*effects, StoreEffect[C]{Emitted: toEvent(eventType, payload)})
		},
		Trigger: func(eventType string, payload ...xs.E) {
			if trigger != nil {
				trigger(toEvent(eventType, payload))
			}
		},
		Effect: func(fn func(enq *StoreEffectEnqueue[C])) {
			*effects = append(*effects, StoreEffect[C]{Run: fn})
		},
	}
}

// assertNoInternalEventTypeCollisions mirrors the development check run by
// extensions that reserve event types.
func assertNoInternalEventTypeCollisions(existing, internal []string, extensionName string) {
	if len(existing) == 0 {
		return
	}
	var collisions []string
	for _, t := range internal {
		if slices.Contains(existing, t) && !slices.Contains(collisions, t) {
			collisions = append(collisions, t)
		}
	}
	if len(collisions) == 0 {
		return
	}
	quoted := make([]string, len(collisions))
	for i, c := range collisions {
		quoted[i] = strconv.Quote(c)
	}
	panic(fmt.Errorf("The %q store extension uses reserved event type(s): %s. Rename the conflicting store event(s) before applying the extension.",
		extensionName, strings.Join(quoted, ", ")))
}

// appendInternalEventTypes mirrors appendInternalEventTypes.
func appendInternalEventTypes(existing, internal []string, extensionName string) []string {
	assertNoInternalEventTypeCollisions(existing, internal, extensionName)
	out := make([]string, 0, len(existing)+len(internal))
	for _, t := range append(slices.Clone(existing), internal...) {
		if !slices.Contains(out, t) {
			out = append(out, t)
		}
	}
	return out
}

// listenerSet mirrors a JS Set of handlers iterated with forEach: entries
// added during an iteration are visited, entries deleted before being
// reached are skipped. Go funcs are not comparable, so every add is a new
// entry. Caller holds jsThread.
type listenerSet[F any] struct {
	entries   []*listenerEntry[F]
	iterating int
}

type listenerEntry[F any] struct {
	fn      F
	removed bool
}

func (l *listenerSet[F]) add(fn F) xs.Subscription {
	entry := &listenerEntry[F]{fn: fn}
	l.entries = append(l.entries, entry)
	return xs.SubscriptionFunc(func() {
		defer jsThread.enter()()
		entry.removed = true
		l.compact()
	})
}

func (l *listenerSet[F]) forEach(call func(F)) {
	l.iterating++
	defer func() {
		l.iterating--
		l.compact()
	}()
	for i := 0; i < len(l.entries); i++ {
		if e := l.entries[i]; !e.removed {
			call(e.fn)
		}
	}
}

// compact drops removed entries once no iteration is running.
func (l *listenerSet[F]) compact() {
	if l.iterating == 0 {
		l.entries = slices.DeleteFunc(l.entries, func(e *listenerEntry[F]) bool { return e.removed })
	}
}

// Store mirrors Store<TContext, TEventPayloadMap, TEmitted> (and
// StoreWithSelectors). It implements ReadonlyAtom[*StoreSnapshot[C]] and
// ActorRefLike.
type Store[C any] struct {
	logic           StoreLogic[C]
	initialSnapshot *StoreSnapshot[C]
	currentSnapshot *StoreSnapshot[C]
	atom            *atomCore[*StoreSnapshot[C]]
	sessionID       string

	listeners       map[string]*listenerSet[func(xs.Event)]
	inspectionObs   listenerSet[func(StoreInspectionEvent)]
	selectorsConfig map[string]func(ctx C) any
	selectors       map[string]ReadonlyAtom[any]
}

// CreateStore mirrors createStore(config).
func CreateStore[C any](config StoreConfig[C]) *Store[C] {
	var ctx C
	if config.ContextFn != nil {
		ctx = config.ContextFn(nil)
	} else {
		ctx = config.Context
	}
	return createStoreFromConfig(config, ctx)
}

func createStoreFromConfig[C any](config StoreConfig[C], ctx C) *Store[C] {
	var eventTypes []string
	if config.Schemas != nil && config.Schemas.Events != nil {
		eventTypes = slices.Sorted(maps.Keys(config.Schemas.Events))
	} else {
		eventTypes = slices.Sorted(maps.Keys(config.On))
	}
	if eventTypes == nil {
		eventTypes = []string{}
	}
	logic := StoreLogic[C]{
		EventTypes: eventTypes,
		Schemas:    config.Schemas,
		GetInitialSnapshot: func() *StoreSnapshot[C] {
			return &StoreSnapshot[C]{Status: xs.StatusActive, Context: ctx}
		},
		Transition: CreateStoreTransition(config.On),
	}
	return createStoreCore(logic)
}

// CreateStoreFromLogic mirrors createStore(logic) with a logic object
// `{ getInitialSnapshot, transition }`.
func CreateStoreFromLogic[C any](logic StoreLogic[C]) *Store[C] {
	return createStoreCore(logic)
}

// CreateStoreConfig mirrors createStoreConfig(config): returns config as is.
func CreateStoreConfig[C any](config StoreConfig[C]) StoreConfig[C] { return config }

// StoreLogicCreator mirrors the object returned by createStoreLogic.
type StoreLogicCreator[C any] struct{ config StoreConfig[C] }

// CreateStoreLogic mirrors createStoreLogic(config).
func CreateStoreLogic[C any](config StoreConfig[C]) *StoreLogicCreator[C] {
	return &StoreLogicCreator[C]{config: config}
}

// CreateStore mirrors logic.createStore(input?).
func (l *StoreLogicCreator[C]) CreateStore(input ...any) *Store[C] {
	var in any
	if len(input) > 0 {
		in = input[0]
	}
	var ctx C
	if l.config.ContextFn != nil {
		ctx = l.config.ContextFn(in)
	} else {
		ctx = l.config.Context
	}
	s := createStoreFromConfig(l.config, ctx)
	if l.config.Selectors != nil {
		s.attachSelectors(l.config.Selectors)
	}
	return s
}

// attachSelectors mirrors attachSelectors(store, selectorsConfig).
func (s *Store[C]) attachSelectors(config map[string]func(ctx C) any) {
	s.selectorsConfig = config
	s.selectors = make(map[string]ReadonlyAtom[any], len(config))
	for key, sel := range config {
		s.selectors[key] = Select(s, sel)
	}
}

// CreateStoreTransition mirrors createStoreTransition(transitions). The
// Immer `producer` argument has no Go equivalent.
func CreateStoreTransition[C any](on map[string]StoreAssigner[C]) StoreTransition[C] {
	return func(snapshot *StoreSnapshot[C], event xs.Event) StoreTransitionResult[C] {
		current := snapshot
		var effects []StoreEffect[C]
		pending := []xs.Event{event}
		allowed := false

		for index := 0; index < len(pending); index++ {
			currentEvent := pending[index]
			currentContext := current.Context
			assigner := on[currentEvent.EventType()]
			if assigner == nil {
				continue
			}
			effectsLength := len(effects)
			enq := newEnqueueObject(&effects, func(triggered xs.Event) {
				allowed = true
				pending = append(pending, triggered)
			})
			next, assigned := assigner(currentContext, currentEvent, enq)
			if !assigned {
				next = currentContext
			}
			allowed = allowed || assigned || len(effects) > effectsLength
			if !objectIs(next, currentContext) {
				current = current.withContext(next)
			}
		}

		if effects == nil {
			effects = []StoreEffect[C]{}
		}
		return StoreTransitionResult[C]{Snapshot: current, Effects: effects, Allowed: &allowed}
	}
}

// uniqueID mirrors uniqueId(): a short random base-36 identifier.
func uniqueID() string {
	const digits = "0123456789abcdefghijklmnopqrstuvwxyz"
	b := make([]byte, 6)
	for i := range b {
		b[i] = digits[rand.IntN(len(digits))]
	}
	return string(b)
}

func createStoreCore[C any](logic StoreLogic[C]) *Store[C] {
	defer jsThread.enter()()
	s := &Store[C]{logic: logic, sessionID: uniqueID()}
	s.initialSnapshot = logic.GetInitialSnapshot()
	s.currentSnapshot = s.initialSnapshot
	s.atom = newWritableCore(s.currentSnapshot, nil)
	return s
}

func (s *Store[C]) emit(ev xs.Event) {
	call := func(fn func(xs.Event)) { fn(ev) }
	if set := s.listeners[ev.EventType()]; set != nil {
		set.forEach(call)
	}
	if set := s.listeners["*"]; set != nil {
		set.forEach(call)
	}
}

func (s *Store[C]) notifyInspection(event xs.Event, snapshot *StoreSnapshot[C]) {
	s.inspectionObs.forEach(func(fn func(StoreInspectionEvent)) {
		fn(StoreInspectionEvent{
			Type:     xs.InspectTransition,
			Event:    event,
			Snapshot: snapshot,
			ActorRef: s,
			RootID:   s.sessionID,
		})
	})
}

// receive mirrors receive(event). Caller holds jsThread.
func (s *Store[C]) receive(event xs.Event) {
	result := s.logic.Transition(s.currentSnapshot, event)
	next := result.Snapshot
	s.currentSnapshot = next

	s.atom.set(nil, &next)
	s.notifyInspection(event, next)

	committed := false
	effectEnqueue := &StoreEffectEnqueue[C]{
		Trigger: s.Trigger,
		Send:    s.Send,
		GetSnapshot: func() *StoreSnapshot[C] {
			defer jsThread.enter()()
			if committed {
				return s.currentSnapshot
			}
			return next
		},
	}

	for _, effect := range result.Effects {
		if effect.Run != nil {
			effect.Run(effectEnqueue)
		} else if effect.Emitted != nil {
			s.emit(effect.Emitted)
		}
	}

	committed = true
}

// unknownTrigger mirrors calling a missing member of the concrete trigger /
// can object (JS TypeError: store.trigger.x is not a function).
func unknownTrigger(object, eventType string) {
	panic(fmt.Errorf("TypeError: store.%s.%s is not a function", object, eventType))
}

func (s *Store[C]) hasEventType(eventType string) bool {
	return len(s.logic.EventTypes) == 0 || slices.Contains(s.logic.EventTypes, eventType)
}

// Send mirrors store.send(event).
func (s *Store[C]) Send(event xs.Event) {
	defer jsThread.enter()()
	s.receive(event)
}

// Trigger mirrors store.trigger[eventType](payload).
func (s *Store[C]) Trigger(eventType string, payload ...xs.E) {
	defer jsThread.enter()()
	if !s.hasEventType(eventType) {
		unknownTrigger("trigger", eventType)
	}
	s.receive(toEvent(eventType, payload))
}

// Can mirrors store.can[eventType](payload): whether the event would be
// allowed, without committing anything. Validation errors yield false.
func (s *Store[C]) Can(eventType string, payload ...xs.E) bool {
	defer jsThread.enter()()
	if !s.hasEventType(eventType) {
		unknownTrigger("can", eventType)
	}
	return s.canTransition(toEvent(eventType, payload))
}

func (s *Store[C]) canTransition(event xs.Event) (ok bool) {
	snapshot := s.currentSnapshot
	defer func() {
		if r := recover(); r != nil {
			if IsStoreValidationError(r) {
				ok = false
				return
			}
			panic(r)
		}
	}()
	result := s.logic.Transition(snapshot, event)
	if result.Allowed != nil {
		return *result.Allowed
	}
	return result.Snapshot != snapshot || len(result.Effects) > 0
}

// GetSnapshot mirrors store.getSnapshot().
func (s *Store[C]) GetSnapshot() *StoreSnapshot[C] {
	defer jsThread.enter()()
	return s.currentSnapshot
}

// Get mirrors store.get() (reactive read; tracked inside computed atoms).
func (s *Store[C]) Get() *StoreSnapshot[C] {
	defer jsThread.enter()()
	return s.atom.get()
}

// GetInitialSnapshot mirrors store.getInitialSnapshot().
func (s *Store[C]) GetInitialSnapshot() *StoreSnapshot[C] { return s.initialSnapshot }

// AnySnapshot is the type-erased GetSnapshot (ActorRefLike).
func (s *Store[C]) AnySnapshot() xs.Snapshot { return s.GetSnapshot() }

// SessionID mirrors store.sessionId.
func (s *Store[C]) SessionID() string { return s.sessionID }

// Schemas mirrors store.schemas (nil when the config had none).
func (s *Store[C]) Schemas() *StoreSchemas { return s.logic.Schemas }

// Selectors mirrors store.selectors of a store created by createStoreLogic
// with `selectors` (nil otherwise).
func (s *Store[C]) Selectors() map[string]ReadonlyAtom[any] { return s.selectors }

// EventTypes mirrors `Object.keys(store.trigger)`: the event types accepted by
// Trigger and Can, sorted alphabetically. For a config-based store these are the
// keys of `on`, the keys of `schemas.events` and the event types added by
// extensions (for example "reset"); for a store created from a logic object they
// are StoreLogic.EventTypes.
func (s *Store[C]) EventTypes() []string {
	return slices.Sorted(slices.Values(s.logic.EventTypes))
}

// Subscribe mirrors store.subscribe(observer).
func (s *Store[C]) Subscribe(observer xs.Observer[*StoreSnapshot[C]]) xs.Subscription {
	defer jsThread.enter()()
	return s.atom.subscribe(observer)
}

// SubscribeNext mirrors store.subscribe(fn).
func (s *Store[C]) SubscribeNext(next func(*StoreSnapshot[C])) xs.Subscription {
	return s.Subscribe(xs.Observer[*StoreSnapshot[C]]{Next: next})
}

// On mirrors store.on(emittedType, handler). "*" matches every emitted event.
func (s *Store[C]) On(eventType string, handler func(xs.Event)) xs.Subscription {
	defer jsThread.enter()()
	if s.listeners == nil {
		s.listeners = map[string]*listenerSet[func(xs.Event)]{}
	}
	set := s.listeners[eventType]
	if set == nil {
		set = &listenerSet[func(xs.Event)]{}
		s.listeners[eventType] = set
	}
	return set.add(handler)
}

// Inspect mirrors store.inspect(fn). The current snapshot is delivered
// immediately with Event = xs.Ev("@xstate.init").
func (s *Store[C]) Inspect(fn func(StoreInspectionEvent)) xs.Subscription {
	defer jsThread.enter()()
	sub := s.inspectionObs.add(fn)
	fn(StoreInspectionEvent{
		Type:     xs.InspectTransition,
		Event:    xs.Ev("@xstate.init"),
		Snapshot: s.currentSnapshot,
		ActorRef: s,
		RootID:   s.sessionID,
	})
	return sub
}

// Transition mirrors store.transition(snapshot, event) → [next, effects].
// It is pure: nothing is committed, emitted or executed.
func (s *Store[C]) Transition(snapshot *StoreSnapshot[C], event xs.Event) (*StoreSnapshot[C], []StoreEffect[C]) {
	defer jsThread.enter()()
	result := s.logic.Transition(snapshot, event)
	return result.Snapshot, result.Effects
}

// With mirrors store.with(extension): a new store over the extended logic.
// Selectors are preserved.
func (s *Store[C]) With(extension StoreExtension[C]) *Store[C] {
	defer jsThread.enter()()
	next := createStoreCore(extension(s.logic))
	if s.selectorsConfig != nil {
		next.attachSelectors(s.selectorsConfig)
	}
	return next
}

// Select mirrors store.select(selector, equalityFn?). Default equality
// mirrors Object.is (see AtomOptions.Compare).
func Select[C, T any](s *Store[C], selector func(ctx C) T, equal ...func(a, b T) bool) ReadonlyAtom[T] {
	var opts []AtomOptions[T]
	if len(equal) > 0 && equal[0] != nil {
		opts = append(opts, AtomOptions[T]{Compare: equal[0]})
	}
	return CreateComputedAtom(func(*T) T { return selector(s.atom.get().Context) }, opts...)
}
