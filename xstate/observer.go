package xstate

// ToObserver mirrors toObserver(nextOrObserver, error, complete).
func ToObserver[T any](next func(T), errFn func(any), complete func()) Observer[T] {
	return Observer[T]{Next: next, Error: errFn, Complete: complete}
}

// ActorRef is the type-erased reference to any actor (children, parents,
// system lookups). Use As to recover a typed *Actor[S].
type ActorRef interface {
	ID() string
	SessionID() string
	Send(event Event)
	AnySnapshot() Snapshot
	SubscribeAny(observer Observer[Snapshot]) Subscription
	System() *ActorSystem
	Src() any // string name or ActorLogic
	Parent() ActorRef
}

// Observer mirrors the JS observer object; nil callbacks are ignored.
type Observer[T any] struct {
	Next     func(T)
	Error    func(err any)
	Complete func()
}

// Subscription mirrors { unsubscribe() }.
type Subscription interface {
	Unsubscribe()
}

// SubscriptionFunc adapts a func to Subscription.
type SubscriptionFunc func()

func (f SubscriptionFunc) Unsubscribe() { f() }

// Subscribable mirrors the JS Subscribable / Observable interface.
type Subscribable[T any] interface {
	Subscribe(observer Observer[T]) Subscription
}
