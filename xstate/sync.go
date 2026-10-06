package xstate

import (
	"bytes"
	"runtime"
	"strconv"
	"sync"
)

// goid returns the id of the calling goroutine. The runtime does not expose
// it directly; it is parsed from the first line of the goroutine stack
// ("goroutine 123 [running]:").
func goid() int64 {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	b := bytes.TrimPrefix(buf[:n], []byte("goroutine "))
	if i := bytes.IndexByte(b, ' '); i >= 0 {
		b = b[:i]
	}
	id, _ := strconv.ParseInt(string(b), 10, 64)
	return id
}

// gstate is the goroutine-local engine state: the actor systems whose lock the
// goroutine holds (innermost last) and whether a custom action is executing.
// JS is single threaded and keeps these as module-level variables
// (createActor.ts `executingCustomAction`, global console / error reporting).
type gstate struct {
	systems         []*ActorSystem
	executingCustom bool
}

var (
	gstatesMu sync.Mutex
	gstates   = map[int64]*gstate{}
)

func withGState(id int64, fn func(g *gstate)) {
	gstatesMu.Lock()
	defer gstatesMu.Unlock()
	g := gstates[id]
	if g == nil {
		g = &gstate{}
		gstates[id] = g
	}
	fn(g)
	if len(g.systems) == 0 && !g.executingCustom {
		delete(gstates, id)
	}
}

// currentSystem returns the innermost actor system whose lock the calling
// goroutine holds, or nil.
func currentSystem() *ActorSystem {
	id := goid()
	gstatesMu.Lock()
	defer gstatesMu.Unlock()
	if g := gstates[id]; g != nil && len(g.systems) > 0 {
		return g.systems[len(g.systems)-1]
	}
	return nil
}

// outermostSystem identifies the last lock this goroutine will release.
func outermostSystem() *ActorSystem {
	id := goid()
	gstatesMu.Lock()
	defer gstatesMu.Unlock()
	if g := gstates[id]; g != nil && len(g.systems) > 0 {
		return g.systems[0]
	}
	return nil
}

func isExecutingCustomAction() bool {
	id := goid()
	gstatesMu.Lock()
	defer gstatesMu.Unlock()
	g := gstates[id]
	return g != nil && g.executingCustom
}

// setExecutingCustomAction sets the flag and returns the previous value.
func setExecutingCustomAction(v bool) (prev bool) {
	withGState(goid(), func(g *gstate) {
		prev = g.executingCustom
		g.executingCustom = v
	})
	return prev
}

// sysLock is the reentrant lock of an actor system. JS runs every actor of a
// system on one thread; Go serializes them with this lock. Reentrancy lets
// engine callbacks (observers, actions, ...) call back into the public API on
// the goroutine that already holds the lock.
type sysLock struct {
	mu    sync.Mutex
	owner int64 // goroutine id; guarded by mu (read without mu only by owner check below)
	depth int
	ownMu sync.Mutex // guards owner reads from other goroutines
	// microtasks run after the outermost unlock (JS promise continuations
	// run after the current synchronous chain).
	microtasks []func()
}

func (l *sysLock) lock(sys *ActorSystem) {
	id := goid()
	l.ownMu.Lock()
	if l.owner == id {
		l.depth++
		l.ownMu.Unlock()
		return
	}
	l.ownMu.Unlock()
	l.mu.Lock()
	l.ownMu.Lock()
	l.owner = id
	l.depth = 1
	l.ownMu.Unlock()
	withGState(id, func(g *gstate) { g.systems = append(g.systems, sys) })
}

func (l *sysLock) unlock(sys *ActorSystem) {
	l.ownMu.Lock()
	l.depth--
	if l.depth > 0 {
		l.ownMu.Unlock()
		return
	}
	id := l.owner
	l.owner = 0
	tasks := l.microtasks
	l.microtasks = nil
	l.ownMu.Unlock()
	defer func() {
		for _, fn := range tasks {
			fn()
		}
	}()
	withGState(id, func(g *gstate) {
		for i := len(g.systems) - 1; i >= 0; i-- {
			if g.systems[i] == sys {
				g.systems = append(g.systems[:i], g.systems[i+1:]...)
				break
			}
		}
	})
	l.mu.Unlock()
}

// microtask runs fn after the calling goroutine releases the lock, or now
// when it does not hold it.
func (l *sysLock) microtask(fn func()) {
	id := goid()
	l.ownMu.Lock()
	if l.owner == id {
		l.microtasks = append(l.microtasks, fn)
		l.ownMu.Unlock()
		return
	}
	l.ownMu.Unlock()
	fn()
}
