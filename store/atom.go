package store

import (
	"context"
	"reflect"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// ReadonlyAtom mirrors Readable<T> / BaseAtom<T> / ReadonlyAtom<T>: computed
// atoms, async atoms, selections and stores implement it.
type ReadonlyAtom[T any] interface {
	// Get mirrors atom.get(); reads inside a computed getter are tracked.
	Get() T
	// Subscribe mirrors atom.subscribe(observer).
	Subscribe(observer xs.Observer[T]) xs.Subscription
	// SubscribeNext mirrors atom.subscribe(fn).
	SubscribeNext(next func(T)) xs.Subscription
}

// AtomOptions mirrors AtomOptions<T>. A nil Compare mirrors Object.is: `==`
// for comparable dynamic values, identity for maps, slices and pointers.
// Non-nil functions compare unequal; provide Compare to customize this.
type AtomOptions[T any] struct {
	Compare func(prev, next T) bool
}

// atomCore mirrors InternalAtom<T>: the reactive node plus the value.
type atomCore[T any] struct {
	node       reactiveNode
	snapshot   T
	hasValue   bool // false until a computed atom ran its getter (JS undefined)
	isComputed bool
	getter     func(prev *T) T
	compare    func(prev, next T) bool
}

func newAtomCore[T any](opts []AtomOptions[T]) *atomCore[T] {
	a := &atomCore[T]{compare: func(prev, next T) bool { return objectIs(prev, next) }}
	if len(opts) > 0 && opts[0].Compare != nil {
		a.compare = opts[0].Compare
	}
	a.node.impl = a
	return a
}

func newWritableCore[T any](initial T, opts []AtomOptions[T]) *atomCore[T] {
	a := newAtomCore(opts)
	a.snapshot = initial
	a.hasValue = true
	a.node.flags = flagMutable
	return a
}

func newComputedCore[T any](getter func(prev *T) T, opts []AtomOptions[T]) *atomCore[T] {
	a := newAtomCore(opts)
	a.isComputed = true
	a.getter = getter
	a.node.flags = flagMutable | flagDirty
	return a
}

// isUndefined mirrors `oldValue === undefined`: a computed atom that never
// ran, or a nil interface value.
func (a *atomCore[T]) isUndefined() bool {
	if !a.hasValue {
		return true
	}
	return !reflect.ValueOf(any(a.snapshot)).IsValid()
}

// update mirrors atom._update() (recompute a computed atom).
func (a *atomCore[T]) update() bool { return a.updateWith(nil, nil) }

func (a *atomCore[T]) notify() {}

// updateWith mirrors atom._update(getValue): fn is the `(prev) => next`
// form, value the plain-value form; both nil recomputes a computed atom.
func (a *atomCore[T]) updateWith(fn func(T) T, value *T) bool {
	prevSub := activeSub
	if a.isComputed {
		activeSub = &a.node
	} else {
		activeSub = nil
	}
	cycle++
	a.node.depsTail = nil
	if a.isComputed {
		a.node.flags = flagMutable | flagRecursedCheck
	}
	defer func() {
		activeSub = prevSub
		if a.isComputed {
			a.node.flags &^= flagRecursedCheck
		}
		purgeDeps(&a.node)
	}()
	oldUndefined := a.isUndefined()
	oldValue := a.snapshot
	var newValue T
	switch {
	case fn != nil:
		newValue = fn(oldValue)
	case value != nil:
		newValue = *value
	case a.isComputed:
		if a.hasValue {
			prev := oldValue
			newValue = a.getter(&prev)
		} else {
			newValue = a.getter(nil)
		}
	}
	if oldUndefined || !a.compare(oldValue, newValue) {
		a.snapshot = newValue
		a.hasValue = true
		return true
	}
	return false
}

// trackRead links the atom to the active subscriber (atom.get()).
func (a *atomCore[T]) trackRead() {
	if activeSub != nil {
		link(&a.node, activeSub, cycle)
		return
	}
	if sub := asyncRunSub(); sub != nil {
		link(&a.node, sub, cycle)
	}
}

// get mirrors atom.get() for both writable and computed atoms. Caller holds
// jsThread.
func (a *atomCore[T]) get() T {
	if a.isComputed {
		flags := a.node.flags
		if flags&flagDirty != 0 || (flags&flagPending != 0 && a.node.deps != nil && checkDirty(a.node.deps, &a.node)) {
			if a.update() {
				if subs := a.node.subs; subs != nil {
					shallowPropagate(subs)
				}
			}
		} else if flags&flagPending != 0 {
			a.node.flags = flags &^ flagPending
		}
	}
	a.trackRead()
	return a.snapshot
}

// set mirrors atom.set(valueOrFn) for writable atoms.
func (a *atomCore[T]) set(fn func(T) T, value *T) {
	if a.updateWith(fn, value) {
		if subs := a.node.subs; subs != nil {
			propagate(subs)
			shallowPropagate(subs)
			flush()
		}
	}
}

// subscribe mirrors atom.subscribe(observerOrFn).
func (a *atomCore[T]) subscribe(observer xs.Observer[T]) xs.Subscription {
	observed := false
	e := newEffect(func() {
		a.get()
		if !observed {
			observed = true
			return
		}
		func() {
			prevSub := activeSub
			activeSub = nil
			defer func() { activeSub = prevSub }()
			if observer.Next != nil {
				observer.Next(a.snapshot)
			}
		}()
		// If the observer synchronously updates any of our deps we'll be
		// marked as dirty preventing this effect from re-running. Request
		// the value again to reconcile any dirty deps.
		a.get()
	})
	return xs.SubscriptionFunc(func() {
		defer jsThread.enter()()
		e.stop()
	})
}

// effectNode mirrors the Effect of atom.ts.
type effectNode struct {
	node reactiveNode
	fn   func()
}

func newEffect(fn func()) *effectNode {
	e := &effectNode{fn: fn}
	e.node.impl = e
	e.node.flags = flagWatching | flagRecursedCheck
	e.run()
	return e
}

func (e *effectNode) run() {
	prevSub := activeSub
	activeSub = &e.node
	cycle++
	e.node.depsTail = nil
	e.node.flags = flagWatching | flagRecursedCheck
	defer func() {
		activeSub = prevSub
		e.node.flags &^= flagRecursedCheck
		purgeDeps(&e.node)
	}()
	e.fn()
}

func (e *effectNode) update() bool { return false }

func (e *effectNode) notify() {
	flags := e.node.flags
	if flags&flagDirty != 0 || (flags&flagPending != 0 && e.node.deps != nil && checkDirty(e.node.deps, &e.node)) {
		e.run()
	} else {
		e.node.flags = flagWatching
	}
}

func (e *effectNode) stop() {
	e.node.flags = flagNone
	e.node.depsTail = nil
	purgeDeps(&e.node)
}

// Atom mirrors Atom<T>, the writable atom from createAtom(value).
type Atom[T any] struct{ core *atomCore[T] }

// CreateAtom mirrors createAtom(initialValue, options?).
func CreateAtom[T any](initial T, opts ...AtomOptions[T]) *Atom[T] {
	return &Atom[T]{core: newWritableCore(initial, opts)}
}

// computedAtom is the read-only atom returned by CreateComputedAtom,
// CreateAsyncAtom and Select.
type computedAtom[T any] struct{ core *atomCore[T] }

// CreateComputedAtom mirrors createAtom(getValue, options?) with a getter:
// a read-only atom recomputed when the atoms it read change. prev is nil on
// the first computation (JS `prev?: T`).
func CreateComputedAtom[T any](get func(prev *T) T, opts ...AtomOptions[T]) ReadonlyAtom[T] {
	return &computedAtom[T]{core: newComputedCore(get, opts)}
}

func (a *computedAtom[T]) Get() T {
	defer jsThread.enter()()
	return a.core.get()
}

func (a *computedAtom[T]) Subscribe(observer xs.Observer[T]) xs.Subscription {
	defer jsThread.enter()()
	return a.core.subscribe(observer)
}

func (a *computedAtom[T]) SubscribeNext(next func(T)) xs.Subscription {
	return a.Subscribe(xs.Observer[T]{Next: next})
}

func (a *Atom[T]) Get() T {
	defer jsThread.enter()()
	return a.core.get()
}

func (a *Atom[T]) Subscribe(observer xs.Observer[T]) xs.Subscription {
	defer jsThread.enter()()
	return a.core.subscribe(observer)
}

func (a *Atom[T]) SubscribeNext(next func(T)) xs.Subscription {
	return a.Subscribe(xs.Observer[T]{Next: next})
}

// Set mirrors atom.set(value).
func (a *Atom[T]) Set(value T) {
	defer jsThread.enter()()
	a.core.set(nil, &value)
}

// Update mirrors atom.set((prev) => next).
func (a *Atom[T]) Update(fn func(prev T) T) {
	defer jsThread.enter()()
	a.core.set(fn, nil)
}

// AtomConfig mirrors AtomConfig<TValue, TInput>.
type AtomConfig[T, I any] struct {
	initial func(input I) T
	opts    []AtomOptions[T]
}

// CreateAtomConfig mirrors createAtomConfig(initialValue, options?).
func CreateAtomConfig[T any](initial T, opts ...AtomOptions[T]) *AtomConfig[T, any] {
	return &AtomConfig[T, any]{initial: func(any) T { return initial }, opts: opts}
}

// CreateAtomConfigFunc mirrors createAtomConfig((input) => value, options?).
func CreateAtomConfigFunc[T, I any](initial func(input I) T, opts ...AtomOptions[T]) *AtomConfig[T, I] {
	return &AtomConfig[T, I]{initial: initial, opts: opts}
}

// CreateAtom mirrors config.createAtom(input?).
func (c *AtomConfig[T, I]) CreateAtom(input ...I) *Atom[T] {
	var in I
	if len(input) > 0 {
		in = input[0]
	}
	return CreateAtom(c.initial(in), c.opts...)
}

// ReducerAtom mirrors ReducerAtom<TState, TEvent>.
type ReducerAtom[S, E any] struct {
	atom    *Atom[S]
	reducer func(state S, event E) S
}

// CreateReducerAtom mirrors createReducerAtom(initialValue, reducer, options?).
func CreateReducerAtom[S, E any](initial S, reducer func(state S, event E) S, opts ...AtomOptions[S]) *ReducerAtom[S, E] {
	return &ReducerAtom[S, E]{atom: CreateAtom(initial, opts...), reducer: reducer}
}

func (a *ReducerAtom[S, E]) Get() S { return a.atom.Get() }
func (a *ReducerAtom[S, E]) Subscribe(observer xs.Observer[S]) xs.Subscription {
	return a.atom.Subscribe(observer)
}
func (a *ReducerAtom[S, E]) SubscribeNext(next func(S)) xs.Subscription {
	return a.atom.SubscribeNext(next)
}

// Send mirrors reducerAtom.send(event).
func (a *ReducerAtom[S, E]) Send(event E) {
	defer jsThread.enter()()
	var next S
	func() {
		prevSub := activeSub
		activeSub = nil
		defer func() { activeSub = prevSub }()
		next = a.reducer(a.atom.core.get(), event)
	}()
	a.atom.core.set(nil, &next)
}

// AsyncStatus mirrors AsyncAtomState['status'].
type AsyncStatus string

const (
	AsyncPending AsyncStatus = "pending"
	AsyncDone    AsyncStatus = "done"
	AsyncError   AsyncStatus = "error"
)

// AsyncAtomState mirrors AsyncAtomState<Data>: Data is set when Status is
// AsyncDone, Error when Status is AsyncError.
type AsyncAtomState[T any] struct {
	Status AsyncStatus
	Data   T
	Error  error
}

// asyncRun is one in-flight getValue call of an async atom. JS reads the
// getter's dependencies synchronously before its first await; in Go the
// getter runs on its own goroutine, so reads made on that goroutine while the
// run is current are linked to the async atom. Two differences follow: a
// dependency changed before the goroutine reads it does not restart the run
// (the run then reads the new value), and reads made after a blocking call
// are tracked too.
type asyncRun struct {
	node    *reactiveNode
	current func() bool
}

var asyncRuns = map[int64]*asyncRun{}

func asyncRunSub() *reactiveNode {
	if len(asyncRuns) == 0 {
		return nil
	}
	run := asyncRuns[goroutineID()]
	if run == nil || !run.current() {
		return nil
	}
	return run.node
}

// CreateAsyncAtom mirrors createAsyncAtom(getValue, options?). getValue runs
// on its own goroutine; ctx is cancelled when the atom recomputes before the
// run settles (JS `signal`). A returned error mirrors a rejection.
func CreateAsyncAtom[T any](
	getValue func(ctx context.Context) (T, error),
	opts ...AtomOptions[AsyncAtomState[T]],
) ReadonlyAtom[AsyncAtomState[T]] {
	var core *atomCore[AsyncAtomState[T]]
	var cancelCurrent context.CancelFunc
	currentRunID := 0

	compare := func(prev, next AsyncAtomState[T]) bool { return objectIs(prev, next) }
	if len(opts) > 0 && opts[0].Compare != nil {
		compare = opts[0].Compare
	}

	settle := func(runID int, ctx context.Context, next AsyncAtomState[T]) {
		if runID != currentRunID || ctx.Err() != nil {
			return
		}
		// Settling changes the value without recollecting the getter's
		// dependencies.
		if !compare(core.snapshot, next) {
			core.snapshot = next
			if subs := core.node.subs; subs != nil {
				propagate(subs)
				shallowPropagate(subs)
				flush()
			}
		}
	}

	core = newComputedCore(func(*AsyncAtomState[T]) AsyncAtomState[T] {
		if cancelCurrent != nil {
			cancelCurrent()
		}
		ctx, cancel := context.WithCancel(context.Background())
		currentRunID++
		runID := currentRunID
		cancelCurrent = cancel
		run := &asyncRun{
			node:    &core.node,
			current: func() bool { return runID == currentRunID && ctx.Err() == nil },
		}

		go func() {
			id := goroutineID()
			release := jsThread.enter()
			asyncRuns[id] = run
			release()

			data, err := callAsyncGetter(getValue, ctx)

			defer jsThread.enter()()
			delete(asyncRuns, id)
			// A panic in a subscriber reached from here has no caller to
			// propagate to (JS: an unhandled rejection).
			defer func() { _ = recover() }()
			if err != nil {
				settle(runID, ctx, AsyncAtomState[T]{Status: AsyncError, Error: err})
			} else {
				settle(runID, ctx, AsyncAtomState[T]{Status: AsyncDone, Data: data})
			}
		}()

		return AsyncAtomState[T]{Status: AsyncPending}
	}, opts)

	return &computedAtom[AsyncAtomState[T]]{core: core}
}

// callAsyncGetter runs getValue, turning a panic into a rejection.
func callAsyncGetter[T any](getValue func(ctx context.Context) (T, error), ctx context.Context) (data T, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = panicError(r)
		}
	}()
	return getValue(ctx)
}
