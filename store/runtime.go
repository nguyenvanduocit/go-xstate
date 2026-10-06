package store

import (
	"bytes"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
)

// The JS implementation runs on a single thread: the reactive graph keeps
// global state (active subscriber, effect queue, version counter) and stores
// mutate their snapshot without synchronisation. The Go port keeps that model
// by guarding every entry into the package with one process-wide lock.
//
// The lock is re-entrant per goroutine because JS code re-enters freely:
// a subscriber may call atom.Set, an effect may call store.Send, a computed
// getter may read a store. Another goroutine (an async effect, an async atom
// getter, a settled storage promise) waits until the current holder leaves,
// which mirrors a JS continuation running once the call stack is empty.
var jsThread reentrantLock

type reentrantLock struct {
	mu    sync.Mutex
	owner atomic.Int64
	depth int
}

// enter acquires the lock for the calling goroutine and returns the release
// function: `defer jsThread.enter()()`.
func (l *reentrantLock) enter() func() {
	id := goroutineID()
	if l.owner.Load() == id {
		l.depth++
		return l.leave
	}
	l.mu.Lock()
	l.owner.Store(id)
	l.depth = 1
	return l.leave
}

func (l *reentrantLock) leave() {
	l.depth--
	if l.depth == 0 {
		l.owner.Store(0)
		l.mu.Unlock()
	}
}

// goroutineID returns the runtime id of the calling goroutine (parsed from
// the "goroutine N [" header of runtime.Stack). Go has no public API for it;
// it is needed to make the lock re-entrant and to attribute reads made by an
// async atom getter to that atom.
func goroutineID() int64 {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	b := bytes.TrimPrefix(buf[:n], []byte("goroutine "))
	if i := bytes.IndexByte(b, ' '); i >= 0 {
		b = b[:i]
	}
	id, err := strconv.ParseInt(string(b), 10, 64)
	if err != nil {
		panic("store: cannot parse goroutine id: " + err.Error())
	}
	return id
}
