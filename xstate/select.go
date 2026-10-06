package xstate

import "reflect"

// Readable mirrors the JS `Readable<T>` interface returned by actor.select():
// a Subscribable[T] whose current value can be read with Get.
type Readable[T any] interface {
	Subscribable[T]
	// Get mirrors readable.get(): selector applied to the actor's current snapshot.
	Get() T
}

type readable[S Snapshot, T any] struct {
	actor    *Actor[S]
	selector func(S) T
	equal    func(a, b T) bool
}

// Select mirrors actor.select(selector, equalityFn). Subscribers are notified
// only when equal(previous, next) reports false. The default mirrors JS
// Object.is: == for comparable values, reflect.DeepEqual otherwise.
func Select[S Snapshot, T any](actor *Actor[S], selector func(S) T, equal ...func(a, b T) bool) Readable[T] {
	r := &readable[S, T]{actor: actor, selector: selector, equal: defaultEqual[T]}
	if len(equal) > 0 && equal[0] != nil {
		r.equal = equal[0]
	}
	return r
}

func defaultEqual[T any](a, b T) bool {
	if reflect.TypeFor[T]().Comparable() {
		defer func() { _ = recover() }()
		return any(a) == any(b)
	}
	return reflect.DeepEqual(a, b)
}

func (r *readable[S, T]) Get() T { return r.selector(r.actor.GetSnapshot()) }

func (r *readable[S, T]) Subscribe(observer Observer[T]) Subscription {
	a := r.actor
	a.system.lock()
	defer a.system.unlock()
	cur, _ := a.snapshot.(S)
	previous := r.selector(cur)
	return a.subscribe(&observerEntry{next: func(s Snapshot) {
		v, _ := s.(S)
		next := r.selector(v)
		if !r.equal(previous, next) {
			previous = next
			if observer.Next != nil {
				observer.Next(next)
			}
		}
	}})
}
