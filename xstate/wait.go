package xstate

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// WaitForOptions mirrors waitFor options. Zero Timeout means no timeout.
type WaitForOptions struct {
	Timeout time.Duration
}

// WaitFor mirrors waitFor(actor, predicate, options). ctx cancellation
// mirrors the JS `signal` option.
func WaitFor[S Snapshot](ctx context.Context, actor *Actor[S], predicate func(S) bool, opts ...WaitForOptions) *Promise[S] {
	p := newPromise[S]()
	if ctx == nil {
		ctx = context.Background()
	}
	if ctx.Err() != nil {
		p.reject(context.Cause(ctx))
		return p
	}
	var timeout time.Duration
	hasTimeout := false
	if len(opts) > 0 && opts[0].Timeout != 0 {
		timeout = opts[0].Timeout
		hasTimeout = true
		if timeout < 0 {
			currentSystem().warn("`timeout` passed to `waitFor` is negative and it will reject its internal promise immediately.")
			timeout = 0
		}
	}

	sys := actor.system
	sys.lock()
	defer sys.unlock()

	var mu sync.Mutex
	done := false
	var sub Subscription
	var timer *time.Timer
	var stopAfter func() bool
	// finish settles the waiter once: the first of predicate match, timeout,
	// abort, error or completion wins.
	finish := func() bool {
		mu.Lock()
		if done {
			mu.Unlock()
			return false
		}
		done = true
		t, s, sa := timer, sub, stopAfter
		mu.Unlock()
		if t != nil {
			t.Stop()
		}
		if s != nil {
			s.Unsubscribe()
		}
		if sa != nil {
			sa()
		}
		return true
	}
	isDone := func() bool {
		mu.Lock()
		defer mu.Unlock()
		return done
	}
	check := func(s S) {
		if isDone() {
			return
		}
		if predicate(s) && finish() {
			sys.lk.microtask(func() { p.resolve(s) })
		}
	}
	if hasTimeout {
		mu.Lock()
		timer = time.AfterFunc(timeout, func() {
			if finish() {
				p.reject(fmt.Errorf("Timeout of %d ms exceeded", timeout.Milliseconds()))
			}
		})
		mu.Unlock()
	}
	cur, _ := actor.snapshot.(S)
	check(cur)
	if isDone() {
		return p
	}
	if ctx.Done() != nil {
		sa := context.AfterFunc(ctx, func() {
			if finish() {
				p.reject(context.Cause(ctx))
			}
		})
		mu.Lock()
		stopAfter = sa
		mu.Unlock()
	}
	s := actor.subscribe(&observerEntry{
		next: func(snap Snapshot) {
			v, _ := snap.(S)
			check(v)
		},
		err: func(err any) {
			if finish() {
				sys.lk.microtask(func() { p.reject(err) })
			}
		},
		complete: func() {
			if finish() {
				sys.lk.microtask(func() { p.reject(errors.New("Actor terminated without satisfying predicate")) })
			}
		},
	})
	mu.Lock()
	sub = s
	alreadyDone := done
	mu.Unlock()
	if alreadyDone {
		s.Unsubscribe()
	}
	return p
}
