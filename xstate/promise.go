package xstate

import (
	"fmt"
	"sync"
)

// Promise is the Go stand-in for a JS promise returned by ToPromise / WaitFor.
// It subscribes eagerly when created.
type Promise[T any] struct {
	once sync.Once
	done chan struct{}
	val  T
	err  error
}

func newPromise[T any]() *Promise[T] { return &Promise[T]{done: make(chan struct{})} }

func (p *Promise[T]) resolve(v T) {
	p.once.Do(func() {
		p.val = v
		close(p.done)
	})
}

func (p *Promise[T]) reject(err any) {
	p.once.Do(func() {
		p.err = toError(err)
		close(p.done)
	})
}

// RejectionError wraps a non-error rejection value.
type RejectionError struct{ Value any }

func (e *RejectionError) Error() string { return fmt.Sprint(e.Value) }

func toError(err any) error {
	if err == nil {
		return &RejectionError{Value: nil}
	}
	if e, ok := err.(error); ok {
		return e
	}
	return &RejectionError{Value: err}
}

// Wait blocks until settled and returns the value or the rejection.
func (p *Promise[T]) Wait() (T, error) {
	<-p.done
	return p.val, p.err
}

// Done is closed once the promise settles.
func (p *Promise[T]) Done() <-chan struct{} { return p.done }

// ToPromise mirrors toPromise(actor): resolves with the output when done,
// rejects with the error when errored.
func ToPromise[S Snapshot](actor *Actor[S]) *Promise[any] {
	p := newPromise[any]()
	actor.system.lock()
	defer actor.system.unlock()
	sys := actor.system
	actor.subscribe(&observerEntry{
		complete: func() {
			out := actor.snapshot.GetOutput()
			sys.lk.microtask(func() { p.resolve(out) })
		},
		err: func(err any) { sys.lk.microtask(func() { p.reject(err) }) },
	})
	return p
}
