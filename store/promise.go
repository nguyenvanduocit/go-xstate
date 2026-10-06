// Package store is the Go port of @xstate/store (references/xstate/packages/xstate-store).
//
// It mirrors the JS entry points `@xstate/store`, `@xstate/store/persist`,
// `@xstate/store/undo`, `@xstate/store/reset` and `@xstate/store/validate` in
// one package. Go names are the JS names in PascalCase (createStore →
// CreateStore, StoreSnapshot → StoreSnapshot). Events are xstate.Event values;
// a JS throw is a Go panic (root package convention).
package store

import (
	"fmt"
	"sync"
)

// Promise is the Go stand-in for a JS Promise returned by storage adapters,
// schema validators and persist helpers.
//
// JS `T | Promise<T>` maps to the Go pair `(T, *Promise[T])`: a non-nil
// promise means the operation is asynchronous and the T value is ignored.
// JS `void | Promise<void>` maps to `*Promise[struct{}]`: nil means the
// operation completed synchronously. Wait and Done are nil-safe: a nil
// promise is already settled with the zero value.
type Promise[T any] struct {
	once  sync.Once
	done  chan struct{}
	value T
	err   error
}

// NewPromise returns a pending promise and its settle functions (JS
// Promise.withResolvers / new Promise((resolve, reject) => ...)).
func NewPromise[T any]() (p *Promise[T], resolve func(T), reject func(err error)) {
	p = &Promise[T]{done: make(chan struct{})}
	resolve = func(v T) {
		p.once.Do(func() {
			p.value = v
			close(p.done)
		})
	}
	reject = func(err error) {
		p.once.Do(func() {
			p.err = err
			close(p.done)
		})
	}
	return p, resolve, reject
}

// Wait blocks until the promise settles and returns the value or the
// rejection (JS `await p`).
func (p *Promise[T]) Wait() (T, error) {
	if p == nil {
		var zero T
		return zero, nil
	}
	<-p.done
	return p.value, p.err
}

// Done is closed once the promise settles.
func (p *Promise[T]) Done() <-chan struct{} {
	if p == nil {
		return closedChan
	}
	return p.done
}

var closedChan = func() chan struct{} {
	c := make(chan struct{})
	close(c)
	return c
}()

// promiseGo runs fn on a new goroutine and settles the returned promise with
// its result; a panic inside fn rejects the promise (JS: a throw inside an
// async function).
func promiseGo[T any](fn func() (T, error)) *Promise[T] {
	p, resolve, reject := NewPromise[T]()
	go func() {
		defer func() {
			if r := recover(); r != nil {
				reject(panicError(r))
			}
		}()
		v, err := fn()
		if err != nil {
			reject(err)
			return
		}
		resolve(v)
	}()
	return p
}

// panicError converts a recovered panic value into an error, keeping errors
// as they are.
func panicError(r any) error {
	if err, ok := r.(error); ok {
		return err
	}
	return thrownValue{r}
}

// thrownValue wraps a non-error panic value (JS `throw 'x'`).
type thrownValue struct{ v any }

func (t thrownValue) Error() string { return fmt.Sprint(t.v) }

// thrown returns the original panic value of an error created by
// panicError.
func thrown(err error) any {
	if t, ok := err.(thrownValue); ok {
		return t.v
	}
	return err
}
