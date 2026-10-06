package store

// Port of alien.ts (adapted from Alien Signals): the push-pull reactive
// graph behind atoms, computed atoms, store selections and subscriptions,
// wired with the callbacks atom.ts passes to createReactiveSystem. All state
// is guarded by jsThread.

type reactiveFlags uint8

const (
	flagNone          reactiveFlags = 0
	flagMutable       reactiveFlags = 1
	flagWatching      reactiveFlags = 2
	flagRecursedCheck reactiveFlags = 4
	flagRecursed      reactiveFlags = 8
	flagDirty         reactiveFlags = 16
	flagPending       reactiveFlags = 32
)

// reactiveNode mirrors ReactiveNode. impl is the atom or effect that owns it.
type reactiveNode struct {
	deps, depsTail *reactiveLink
	subs, subsTail *reactiveLink
	flags          reactiveFlags
	impl           reactiveImpl
}

// reactiveImpl is implemented by atoms (update) and effects (notify).
type reactiveImpl interface {
	// update mirrors atom._update(): recompute and report a change.
	update() bool
	// notify mirrors Effect.notify(): run the effect if a dependency changed.
	notify()
}

// reactiveLink mirrors Link.
type reactiveLink struct {
	version          int
	dep, sub         *reactiveNode
	prevSub, nextSub *reactiveLink
	prevDep, nextDep *reactiveLink
}

type linkStack struct {
	value *reactiveLink
	prev  *linkStack
}

// Global reactive state (atom.ts module scope).
var (
	queuedEffects []*reactiveNode
	notifyIndex   int
	cycle         int
	activeSub     *reactiveNode
)

// reactive-system callbacks from atom.ts

func nodeUpdate(n *reactiveNode) bool { return n.impl.update() }

func nodeNotify(n *reactiveNode) {
	queuedEffects = append(queuedEffects, n)
	n.flags &^= flagWatching
}

func nodeUnwatched(n *reactiveNode) {
	if n.depsTail != nil {
		n.depsTail = nil
		n.flags = flagMutable | flagDirty
		purgeDeps(n)
	}
}

func purgeDeps(sub *reactiveNode) {
	depsTail := sub.depsTail
	var dep *reactiveLink
	if depsTail != nil {
		dep = depsTail.nextDep
	} else {
		dep = sub.deps
	}
	for dep != nil {
		dep = unlink(dep, sub)
	}
}

// flush mirrors atom.ts flush(): runs queued effects, keeps draining after a
// subscriber panics and re-panics with the first panic value at the end.
func flush() {
	didThrow := false
	var firstErr any
	func() {
		defer func() {
			notifyIndex = 0
			queuedEffects = queuedEffects[:0]
		}()
		for notifyIndex < len(queuedEffects) {
			effect := queuedEffects[notifyIndex]
			queuedEffects[notifyIndex] = nil
			notifyIndex++
			func() {
				defer func() {
					if r := recover(); r != nil {
						effect.flags |= flagWatching | flagRecursed
						if !didThrow {
							didThrow = true
							firstErr = r
						}
					}
				}()
				effect.impl.notify()
			}()
		}
	}()
	if didThrow {
		panic(firstErr)
	}
}

func link(dep, sub *reactiveNode, version int) {
	prevDep := sub.depsTail
	if prevDep != nil && prevDep.dep == dep {
		return
	}
	var nextDep *reactiveLink
	if prevDep != nil {
		nextDep = prevDep.nextDep
	} else {
		nextDep = sub.deps
	}
	if nextDep != nil && nextDep.dep == dep {
		nextDep.version = version
		sub.depsTail = nextDep
		return
	}
	prevSub := dep.subsTail
	if prevSub != nil && prevSub.version == version && prevSub.sub == sub {
		return
	}
	newLink := &reactiveLink{
		version: version,
		dep:     dep,
		sub:     sub,
		prevDep: prevDep,
		nextDep: nextDep,
		prevSub: prevSub,
	}
	sub.depsTail = newLink
	dep.subsTail = newLink
	if nextDep != nil {
		nextDep.prevDep = newLink
	}
	if prevDep != nil {
		prevDep.nextDep = newLink
	} else {
		sub.deps = newLink
	}
	if prevSub != nil {
		prevSub.nextSub = newLink
	} else {
		dep.subs = newLink
	}
}

func unlink(l *reactiveLink, sub *reactiveNode) *reactiveLink {
	dep := l.dep
	prevDep := l.prevDep
	nextDep := l.nextDep
	nextSub := l.nextSub
	prevSub := l.prevSub
	if nextDep != nil {
		nextDep.prevDep = prevDep
	} else {
		sub.depsTail = prevDep
	}
	if prevDep != nil {
		prevDep.nextDep = nextDep
	} else {
		sub.deps = nextDep
	}
	if nextSub != nil {
		nextSub.prevSub = prevSub
	} else {
		dep.subsTail = prevSub
	}
	if prevSub != nil {
		prevSub.nextSub = nextSub
	} else {
		dep.subs = nextSub
		if nextSub == nil {
			nodeUnwatched(dep)
		}
	}
	return nextDep
}

func propagate(l *reactiveLink) {
	next := l.nextSub
	var stack *linkStack

top:
	for {
		sub := l.sub
		flags := sub.flags

		if flags&(flagRecursedCheck|flagRecursed|flagDirty|flagPending) == 0 {
			sub.flags = flags | flagPending
		} else if flags&(flagRecursedCheck|flagRecursed) == 0 {
			flags = flagNone
		} else if flags&flagRecursedCheck == 0 {
			sub.flags = (flags &^ flagRecursed) | flagPending
		} else if flags&(flagDirty|flagPending) == 0 && isValidLink(l, sub) {
			sub.flags = flags | flagRecursed | flagPending
			flags &= flagMutable
		} else {
			flags = flagNone
		}

		if flags&flagWatching != 0 {
			nodeNotify(sub)
		}

		if flags&flagMutable != 0 {
			subSubs := sub.subs
			if subSubs != nil {
				l = subSubs
				nextSub := l.nextSub
				if nextSub != nil {
					stack = &linkStack{value: next, prev: stack}
					next = nextSub
				}
				continue
			}
		}

		if l = next; l != nil {
			next = l.nextSub
			continue
		}

		for stack != nil {
			l = stack.value
			stack = stack.prev
			if l != nil {
				next = l.nextSub
				continue top
			}
		}

		break
	}
}

func checkDirty(l *reactiveLink, sub *reactiveNode) bool {
	var stack *linkStack
	checkDepth := 0
	dirty := false

top:
	for {
		dep := l.dep
		flags := dep.flags

		if sub.flags&flagDirty != 0 {
			dirty = true
		} else if flags&(flagMutable|flagDirty) == flagMutable|flagDirty {
			if nodeUpdate(dep) {
				subs := dep.subs
				if subs.nextSub != nil {
					shallowPropagate(subs)
				}
				dirty = true
			}
		} else if flags&(flagMutable|flagPending) == flagMutable|flagPending {
			if l.nextSub != nil || l.prevSub != nil {
				stack = &linkStack{value: l, prev: stack}
			}
			l = dep.deps
			sub = dep
			checkDepth++
			continue
		}

		if !dirty {
			if nextDep := l.nextDep; nextDep != nil {
				l = nextDep
				continue
			}
		}

		for checkDepth > 0 {
			checkDepth--
			firstSub := sub.subs
			hasMultipleSubs := firstSub.nextSub != nil
			if hasMultipleSubs {
				l = stack.value
				stack = stack.prev
			} else {
				l = firstSub
			}
			if dirty {
				if nodeUpdate(sub) {
					if hasMultipleSubs {
						shallowPropagate(firstSub)
					}
					sub = l.sub
					continue
				}
				dirty = false
			} else {
				sub.flags &^= flagPending
			}
			sub = l.sub
			if nextDep := l.nextDep; nextDep != nil {
				l = nextDep
				continue top
			}
		}

		return dirty
	}
}

func shallowPropagate(l *reactiveLink) {
	for l != nil {
		sub := l.sub
		flags := sub.flags
		if flags&(flagPending|flagDirty) == flagPending {
			sub.flags = flags | flagDirty
			if flags&(flagWatching|flagRecursedCheck) == flagWatching {
				nodeNotify(sub)
			}
		}
		l = l.nextSub
	}
}

func isValidLink(checkLink *reactiveLink, sub *reactiveNode) bool {
	for l := sub.depsTail; l != nil; l = l.prevDep {
		if l == checkLink {
			return true
		}
	}
	return false
}
