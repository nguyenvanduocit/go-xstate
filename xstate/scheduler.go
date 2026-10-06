package xstate

import (
	"time"
)

// Scheduler mirrors the delayed-event scheduler of the system.
type Scheduler struct {
	sys *ActorSystem
}

// Schedule mirrors scheduler.schedule(source, target, event, delay, id).
func (s *Scheduler) Schedule(source, target ActorRef, event Event, delay time.Duration, id string) {
	s.sys.lock()
	defer s.sys.unlock()
	s.schedule(source, target, event, delay, id)
}

// Cancel mirrors scheduler.cancel(source, id).
func (s *Scheduler) Cancel(source ActorRef, id string) {
	s.sys.lock()
	defer s.sys.unlock()
	s.cancel(source, id)
}

// CancelAll mirrors scheduler.cancelAll(actorRef).
func (s *Scheduler) CancelAll(actor ActorRef) {
	s.sys.lock()
	defer s.sys.unlock()
	s.cancelAll(actor)
}

func (s *Scheduler) schedule(source, target ActorRef, event Event, delay time.Duration, id string) {
	sys := s.sys
	if id == "" {
		id = randomID()
	}
	key := source.SessionID() + "." + id
	se := &scheduledEvent{id: id, event: event, startedAt: time.Now(), delay: delay, source: source, target: target}
	if _, exists := sys.scheduled[key]; !exists {
		sys.scheduledKeys = append(sys.scheduledKeys, key)
	}
	sys.scheduled[key] = se
	entry := &timerEntry{live: true}
	sys.timers[key] = entry
	entry.id = sys.clock.SetTimeout(func() {
		sys.lock()
		defer sys.unlock()
		if !entry.live {
			return
		}
		entry.live = false
		if sys.timers[key] == entry {
			delete(sys.timers, key)
		}
		sys.removeScheduled(key)
		sys.relay(source, target, event)
	}, delay)
}

func (s *ActorSystem) removeScheduled(key string) {
	if _, ok := s.scheduled[key]; !ok {
		return
	}
	delete(s.scheduled, key)
	for i, k := range s.scheduledKeys {
		if k == key {
			s.scheduledKeys = append(s.scheduledKeys[:i:i], s.scheduledKeys[i+1:]...)
			break
		}
	}
}

func (s *Scheduler) cancel(source ActorRef, id string) {
	sys := s.sys
	key := source.SessionID() + "." + id
	entry := sys.timers[key]
	delete(sys.timers, key)
	sys.removeScheduled(key)
	if entry != nil {
		entry.live = false
		sys.clock.ClearTimeout(entry.id)
	}
}

func (s *Scheduler) cancelAll(actor ActorRef) {
	sys := s.sys
	for _, k := range append([]string{}, sys.scheduledKeys...) {
		se := sys.scheduled[k]
		if se != nil && se.source == actor {
			s.cancel(actor, se.id)
		}
	}
}
