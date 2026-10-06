package xstate

import (
	"errors"
	"sort"
	"sync"
	"time"
)

// TimerID identifies a scheduled timeout.
type TimerID any

// Clock mirrors the JS Clock interface.
type Clock interface {
	SetTimeout(fn func(), d time.Duration) TimerID
	ClearTimeout(id TimerID)
}

// defaultClock mirrors the default `{ setTimeout, clearTimeout }` clock with
// real timers; callbacks run on timer goroutines.
type defaultClock struct{}

func (defaultClock) SetTimeout(fn func(), d time.Duration) TimerID { return time.AfterFunc(d, fn) }
func (defaultClock) ClearTimeout(id TimerID) {
	if t, ok := id.(*time.Timer); ok {
		t.Stop()
	}
}

type simulatedTimeout struct {
	start   time.Duration
	timeout time.Duration
	fn      func()
}

// SimulatedClock mirrors SimulatedClock: timers fire synchronously inside
// Increment / Set on the calling goroutine.
type SimulatedClock struct {
	mu                  sync.Mutex
	timeouts            map[int]*simulatedTimeout
	order               []int
	now                 time.Duration
	id                  int
	flushing            bool
	flushingInvalidated bool
}

func NewSimulatedClock() *SimulatedClock {
	return &SimulatedClock{timeouts: map[int]*simulatedTimeout{}}
}

func (c *SimulatedClock) SetTimeout(fn func(), d time.Duration) TimerID {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.flushingInvalidated = c.flushing
	id := c.id
	c.id++
	c.timeouts[id] = &simulatedTimeout{start: c.now, timeout: d, fn: fn}
	c.order = append(c.order, id)
	return id
}

func (c *SimulatedClock) ClearTimeout(id TimerID) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.flushingInvalidated = c.flushing
	if i, ok := id.(int); ok {
		c.deleteLocked(i)
	}
}

func (c *SimulatedClock) deleteLocked(id int) {
	if _, ok := c.timeouts[id]; !ok {
		return
	}
	delete(c.timeouts, id)
	for i, v := range c.order {
		if v == id {
			c.order = append(c.order[:i:i], c.order[i+1:]...)
			break
		}
	}
}

func (c *SimulatedClock) Now() time.Duration {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *SimulatedClock) Set(now time.Duration) {
	c.mu.Lock()
	if c.now > now {
		c.mu.Unlock()
		panic(errors.New("Unable to travel back in time"))
	}
	c.now = now
	c.mu.Unlock()
	c.flushTimeouts()
}

func (c *SimulatedClock) Increment(d time.Duration) {
	c.mu.Lock()
	c.now += d
	c.mu.Unlock()
	c.flushTimeouts()
}

// flushTimeouts mirrors SimulatedClock.flushTimeouts.
func (c *SimulatedClock) flushTimeouts() {
	c.mu.Lock()
	if c.flushing {
		c.flushingInvalidated = true
		c.mu.Unlock()
		return
	}
	c.flushing = true
	type entry struct {
		id int
		t  *simulatedTimeout
	}
	sorted := make([]entry, 0, len(c.order))
	for _, id := range c.order {
		sorted = append(sorted, entry{id, c.timeouts[id]})
	}
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].t.start+sorted[i].t.timeout < sorted[j].t.start+sorted[j].t.timeout
	})
	c.mu.Unlock()

	for _, e := range sorted {
		c.mu.Lock()
		if c.flushingInvalidated {
			c.flushingInvalidated = false
			c.flushing = false
			c.mu.Unlock()
			c.flushTimeouts()
			return
		}
		_, still := c.timeouts[e.id]
		due := still && c.now-e.t.start >= e.t.timeout
		if due {
			c.deleteLocked(e.id)
		}
		c.mu.Unlock()
		if due {
			e.t.fn()
		}
	}
	c.mu.Lock()
	c.flushing = false
	c.mu.Unlock()
}
