package xstate_test

// Shared test helpers used by every translated test file. Translators must
// use these instead of defining their own equivalents.

import (
	"fmt"
	"sync"
	"testing"
	"time"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// ---- vi.fn() ----

// spy mirrors vi.fn(): it records every call's arguments.
type spy struct {
	mu    sync.Mutex
	calls [][]any
}

func newSpy() *spy { return &spy{} }

// Call records a call.
func (s *spy) Call(args ...any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls = append(s.calls, args)
}

// Calls returns a copy of recorded calls.
func (s *spy) Calls() [][]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([][]any, len(s.calls))
	copy(out, s.calls)
	return out
}

// Count returns the number of calls (toHaveBeenCalledTimes).
func (s *spy) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.calls)
}

// ---- Promise.withResolvers / await ----

// signal is a one-shot completion used where JS tests await
// Promise.withResolvers().promise.
type signal struct {
	once sync.Once
	ch   chan struct{}
}

func newSignal() *signal { return &signal{ch: make(chan struct{})} }

func (s *signal) Resolve() { s.once.Do(func() { close(s.ch) }) }

// Wait fails the test if the signal is not resolved within d (default 2s).
func (s *signal) Wait(t testing.TB, d ...time.Duration) {
	t.Helper()
	timeout := 2 * time.Second
	if len(d) > 0 {
		timeout = d[0]
	}
	select {
	case <-s.ch:
	case <-time.After(timeout):
		t.Fatalf("timed out after %v waiting for signal", timeout)
	}
}

// sleep mirrors `await sleep(ms)` from node:timers/promises.
func sleep(ms int) { time.Sleep(time.Duration(ms) * time.Millisecond) }

// ms converts JS milliseconds to time.Duration.
func ms(n int) time.Duration { return time.Duration(n) * time.Millisecond }

// ---- rxjs stand-ins ----

type observable[T any] struct {
	subscribe func(o xs.Observer[T]) func()
}

func (ob *observable[T]) Subscribe(o xs.Observer[T]) xs.Subscription {
	var mu sync.Mutex
	closed := false
	safe := xs.Observer[T]{
		Next: func(v T) {
			mu.Lock()
			c := closed
			mu.Unlock()
			if !c && o.Next != nil {
				o.Next(v)
			}
		},
		Error: func(err any) {
			mu.Lock()
			c := closed
			closed = true
			mu.Unlock()
			if !c && o.Error != nil {
				o.Error(err)
			}
		},
		Complete: func() {
			mu.Lock()
			c := closed
			closed = true
			mu.Unlock()
			if !c && o.Complete != nil {
				o.Complete()
			}
		},
	}
	teardown := ob.subscribe(safe)
	return xs.SubscriptionFunc(func() {
		mu.Lock()
		closed = true
		mu.Unlock()
		if teardown != nil {
			teardown()
		}
	})
}

// rxOf mirrors rxjs of(...values): emits synchronously then completes.
func rxOf[T any](values ...T) xs.Subscribable[T] {
	return &observable[T]{subscribe: func(o xs.Observer[T]) func() {
		for _, v := range values {
			o.Next(v)
		}
		o.Complete()
		return nil
	}}
}

// rxFrom mirrors rxjs from(array).
func rxFrom[T any](values []T) xs.Subscribable[T] { return rxOf(values...) }

// rxEmpty mirrors rxjs EMPTY.
func rxEmpty[T any]() xs.Subscribable[T] { return rxOf[T]() }

// rxThrowError mirrors rxjs throwError(() => err).
func rxThrowError[T any](err any) xs.Subscribable[T] {
	return &observable[T]{subscribe: func(o xs.Observer[T]) func() {
		o.Error(err)
		return nil
	}}
}

// rxInterval mirrors rxjs interval(ms): emits 0, 1, 2, ... every period.
func rxInterval(period int) xs.Subscribable[int] {
	return &observable[int]{subscribe: func(o xs.Observer[int]) func() {
		stop := make(chan struct{})
		go func() {
			tk := time.NewTicker(ms(period))
			defer tk.Stop()
			for i := 0; ; i++ {
				select {
				case <-stop:
					return
				case <-tk.C:
					o.Next(i)
				}
			}
		}()
		var once sync.Once
		return func() { once.Do(func() { close(stop) }) }
	}}
}

// rxMap mirrors pipe(map(fn)).
func rxMap[T, U any](src xs.Subscribable[T], fn func(T) U) xs.Subscribable[U] {
	return &observable[U]{subscribe: func(o xs.Observer[U]) func() {
		sub := src.Subscribe(xs.Observer[T]{
			Next:     func(v T) { o.Next(fn(v)) },
			Error:    o.Error,
			Complete: o.Complete,
		})
		return sub.Unsubscribe
	}}
}

// rxFilter mirrors pipe(filter(fn)).
func rxFilter[T any](src xs.Subscribable[T], fn func(T) bool) xs.Subscribable[T] {
	return &observable[T]{subscribe: func(o xs.Observer[T]) func() {
		sub := src.Subscribe(xs.Observer[T]{
			Next: func(v T) {
				if fn(v) {
					o.Next(v)
				}
			},
			Error:    o.Error,
			Complete: o.Complete,
		})
		return sub.Unsubscribe
	}}
}

// rxTake mirrors pipe(take(n)).
func rxTake[T any](src xs.Subscribable[T], n int) xs.Subscribable[T] {
	return &observable[T]{subscribe: func(o xs.Observer[T]) func() {
		var mu sync.Mutex
		count := 0
		var sub xs.Subscription
		done := false
		sub = src.Subscribe(xs.Observer[T]{
			Next: func(v T) {
				mu.Lock()
				if done {
					mu.Unlock()
					return
				}
				count++
				last := count >= n
				if last {
					done = true
				}
				mu.Unlock()
				o.Next(v)
				if last {
					o.Complete()
					if sub != nil {
						sub.Unsubscribe()
					}
				}
			},
			Error:    o.Error,
			Complete: o.Complete,
		})
		return func() { sub.Unsubscribe() }
	}}
}

// behaviorSubject mirrors rxjs BehaviorSubject.
type behaviorSubject[T any] struct {
	mu        sync.Mutex
	value     T
	observers map[int]xs.Observer[T]
	nextID    int
}

func newBehaviorSubject[T any](initial T) *behaviorSubject[T] {
	return &behaviorSubject[T]{value: initial, observers: map[int]xs.Observer[T]{}}
}

func (b *behaviorSubject[T]) Subscribe(o xs.Observer[T]) xs.Subscription {
	b.mu.Lock()
	id := b.nextID
	b.nextID++
	b.observers[id] = o
	v := b.value
	b.mu.Unlock()
	if o.Next != nil {
		o.Next(v)
	}
	return xs.SubscriptionFunc(func() {
		b.mu.Lock()
		delete(b.observers, id)
		b.mu.Unlock()
	})
}

func (b *behaviorSubject[T]) Next(v T) {
	b.mu.Lock()
	b.value = v
	obs := make([]xs.Observer[T], 0, len(b.observers))
	for _, o := range b.observers {
		obs = append(obs, o)
	}
	b.mu.Unlock()
	for _, o := range obs {
		if o.Next != nil {
			o.Next(v)
		}
	}
}

func (b *behaviorSubject[T]) GetValue() T {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.value
}

// ---- snapshot access ----

// machineSnap returns the typed machine snapshot of a child ActorRef.
func machineSnap[C any](ref xs.ActorRef) *xs.MachineSnapshot[C] {
	return ref.AnySnapshot().(*xs.MachineSnapshot[C])
}

// panicMessage runs fn and returns the recovered panic message ("" if fn
// did not panic). error values yield err.Error().
func panicMessage(fn func()) (msg string) {
	defer func() {
		r := recover()
		switch v := r.(type) {
		case nil:
			msg = ""
		case error:
			msg = v.Error()
		default:
			msg = fmt.Sprint(v)
		}
	}()
	fn()
	return ""
}
