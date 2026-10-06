package xstate

import (
	"fmt"
	"log"
	"runtime"
	"sort"
	"sync"
	"time"
)

// ---- system (system.ts) ----

// ActorSystem mirrors the JS actor system.
type ActorSystem struct {
	lk               sysLock
	root             *actorCore
	children         map[string]ActorRef
	keyed            map[string]ActorRef
	reverseKeyed     map[ActorRef]string
	inspectors       []*Observer[InspectionEvent]
	scheduler        *Scheduler
	scheduledKeys    []string
	scheduled        map[string]*scheduledEvent
	timers           map[string]*timerEntry
	clock            Clock
	logger           func(args ...any)
	onWarn           func(args ...any)
	onUnhandledError func(err any)
	async            asyncTracker
}

// asyncTracker orders promise settlements. JS settles promises that resolve
// in the same tick in FIFO order (microtask queue); Go promise bodies finish
// on independent goroutines in arbitrary order. Settlements are queued and
// delivered sorted by promise start order, and a settling promise briefly
// lets earlier promises that were started within asyncGrace settle first.
type asyncTracker struct {
	mu       sync.Mutex
	next     uint64
	inflight map[uint64]time.Time
	pending  []asyncItem
}

type asyncItem struct {
	seq     uint64
	deliver func()
}

const asyncGrace = time.Millisecond

func (s *ActorSystem) asyncBegin() uint64 {
	a := &s.async
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.inflight == nil {
		a.inflight = map[uint64]time.Time{}
	}
	a.next++
	a.inflight[a.next] = time.Now()
	return a.next
}

func (a *asyncTracker) hasRecentEarlier(seq uint64) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	now := time.Now()
	for k, started := range a.inflight {
		if k < seq && now.Sub(started) < asyncGrace {
			return true
		}
	}
	return false
}

// asyncSettle delivers a settled promise result (called on the promise
// goroutine, without the system lock).
func (s *ActorSystem) asyncSettle(seq uint64, deliver func()) {
	a := &s.async
	a.mu.Lock()
	delete(a.inflight, seq)
	a.pending = append(a.pending, asyncItem{seq: seq, deliver: deliver})
	a.mu.Unlock()
	for a.hasRecentEarlier(seq) {
		runtime.Gosched()
	}
	s.lock()
	defer s.unlock()
	a.mu.Lock()
	items := a.pending
	a.pending = nil
	a.mu.Unlock()
	sort.Slice(items, func(i, j int) bool { return items[i].seq < items[j].seq })
	for _, it := range items {
		it.deliver()
	}
}

type scheduledEvent struct {
	id        string
	event     Event
	startedAt time.Time
	delay     time.Duration
	source    ActorRef
	target    ActorRef
}

// timerEntry is one scheduled timeout. live is cleared when the timer fires
// or is cancelled; a timer whose key was re-scheduled stays live and still
// fires, as in JS (system.ts keeps only the latest timeout per key).
type timerEntry struct {
	id   TimerID
	live bool
}

func newActorSystem(root *actorCore, clock Clock, logger func(args ...any)) *ActorSystem {
	s := &ActorSystem{
		root:         root,
		children:     map[string]ActorRef{},
		keyed:        map[string]ActorRef{},
		reverseKeyed: map[ActorRef]string{},
		scheduled:    map[string]*scheduledEvent{},
		timers:       map[string]*timerEntry{},
		clock:        clock,
		logger:       logger,
	}
	s.scheduler = &Scheduler{sys: s}
	return s
}

func (s *ActorSystem) lock() { s.lk.lock(s) }

func (s *ActorSystem) unlock() { s.lk.unlock(s) }

func (s *ActorSystem) bookID() string {
	return fmt.Sprintf("x:%d", sessionCounter.Add(1)-1)
}

func (s *ActorSystem) register(sessionID string, ref ActorRef) { s.children[sessionID] = ref }

func (s *ActorSystem) unregister(ref ActorRef) {
	delete(s.children, ref.SessionID())
	if id, ok := s.reverseKeyed[ref]; ok {
		delete(s.keyed, id)
		delete(s.reverseKeyed, ref)
	}
}

func (s *ActorSystem) set(systemID string, ref ActorRef) {
	if existing, ok := s.keyed[systemID]; ok && existing != ref {
		panic(fmt.Errorf("Actor with system ID '%s' already exists.", systemID))
	}
	s.keyed[systemID] = ref
	s.reverseKeyed[ref] = systemID
}

// Get mirrors system.get(systemId); nil when absent.
func (s *ActorSystem) Get(systemID string) ActorRef {
	s.lock()
	defer s.unlock()
	return s.keyed[systemID]
}

// GetAll mirrors system.getAll().
func (s *ActorSystem) GetAll() map[string]ActorRef {
	s.lock()
	defer s.unlock()
	out := make(map[string]ActorRef, len(s.keyed))
	for k, v := range s.keyed {
		out[k] = v
	}
	return out
}

// Inspect mirrors system.inspect(observer).
func (s *ActorSystem) Inspect(fn func(InspectionEvent)) Subscription {
	return s.InspectObserver(Observer[InspectionEvent]{Next: fn})
}

// InspectObserver mirrors system.inspect({ next }).
func (s *ActorSystem) InspectObserver(observer Observer[InspectionEvent]) Subscription {
	s.lock()
	defer s.unlock()
	o := &observer
	s.inspectors = append(s.inspectors, o)
	return SubscriptionFunc(func() {
		s.lock()
		defer s.unlock()
		for i, e := range s.inspectors {
			if e == o {
				s.inspectors = append(s.inspectors[:i:i], s.inspectors[i+1:]...)
				break
			}
		}
	})
}

func (s *ActorSystem) sendInspectionEvent(ev InspectionEvent) {
	if s == nil || len(s.inspectors) == 0 {
		return
	}
	ev.RootID = s.root.sessionID
	for _, o := range append([]*Observer[InspectionEvent]{}, s.inspectors...) {
		if o.Next != nil {
			o.Next(ev)
		}
	}
}

func (s *ActorSystem) relay(source ActorRef, target ActorRef, event Event) {
	ev := InspectionEvent{Type: InspectEvent, ActorRef: target, Event: event}
	if !isNilRef(source) {
		ev.SourceRef = source
	}
	s.sendInspectionEvent(ev)
	if c := coreOf(target); c != nil {
		if c.system == s {
			c.send(event)
			return
		}
		deliver := func() {
			c.system.lock()
			defer c.system.unlock()
			c.send(event)
		}
		// Acquiring a foreign lock while holding the source lock can deadlock
		// reciprocal sends. Drain under the destination lock after all of this
		// goroutine's system locks are released, before the outer call returns.
		if outer := outermostSystem(); outer != nil {
			outer.lk.microtask(deliver)
		} else {
			deliver()
		}
		return
	}
	target.Send(event)
}

// GetSnapshot mirrors system.getSnapshot().
func (s *ActorSystem) GetSnapshot() map[string]any {
	s.lock()
	defer s.unlock()
	events := map[string]any{}
	for _, k := range s.scheduledKeys {
		se := s.scheduled[k]
		events[k] = map[string]any{
			"id":        se.id,
			"event":     se.event,
			"startedAt": se.startedAt,
			"delay":     se.delay,
			"source":    se.source,
			"target":    se.target,
		}
	}
	return map[string]any{"_scheduledEvents": events}
}

// Scheduler mirrors system.scheduler.
func (s *ActorSystem) Scheduler() *Scheduler { return s.scheduler }

func (s *ActorSystem) start() {
	keys := s.scheduledKeys
	events := s.scheduled
	s.scheduledKeys = nil
	s.scheduled = map[string]*scheduledEvent{}
	for _, k := range keys {
		se := events[k]
		s.scheduler.schedule(se.source, se.target, se.event, se.delay, se.id)
	}
}

func (s *ActorSystem) warn(args ...any) {
	if s != nil && s.onWarn != nil {
		s.onWarn(args...)
		return
	}
	log.Println(append([]any{"[xstate warn]"}, args...)...)
}

func (s *ActorSystem) reportUnhandledError(err any) {
	if s != nil && s.onUnhandledError != nil {
		s.onUnhandledError(err)
		return
	}
	log.Println("[xstate] unhandled error:", err)
}

// warn mirrors console.warn: it goes to the warn handler of the actor system
// the calling goroutine is processing, or to the default logger.
func warn(args ...any) {
	currentSystem().warn(args...)
}

func defaultLogger(args ...any) { log.Println(args...) }
