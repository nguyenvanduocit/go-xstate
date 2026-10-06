package xstate

import (
	"sort"
)

func resolveOutput(mapper any, context any, event Event, self ActorRef) any {
	if e, ok := mapper.(*exprFunc); ok {
		return e.fn(exprArgs{context: context, event: event, self: self})
	}
	return mapper
}

func getMachineOutput(snap anyMachineSnapshot, event Event, scope *ActorScope, root, completion *StateNode) any {
	if root.Output == nil {
		return nil
	}
	var out any
	if completion.Output != nil && completion.Parent != nil {
		out = resolveOutput(completion.Output, snap.contextAny(), event, scope.Self)
	}
	done := DoneStateEvent{StateID: completion.ID, Output: out}
	return resolveOutput(root.Output, snap.contextAny(), done, scope.Self)
}

func enterStates(current anyMachineSnapshot, event Event, scope *ActorScope, filtered []*TransitionDefinition, mutSet *nodeSet, internalQueue *[]Event, history map[string][]*StateNode, isInitial bool) anyMachineSnapshot {
	next := current
	toEnter := newNodeSet()
	forDefaultEntry := newNodeSet()
	computeEntrySet(filtered, history, forDefaultEntry, toEnter)
	if isInitial {
		forDefaultEntry.add(current.machineAny().RootNode())
	}
	completed := map[*StateNode]bool{}
	sorted := toEnter.values()
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Order < sorted[j].Order })
	for _, n := range sorted {
		mutSet.add(n)
		var actions Actions
		actions = append(actions, n.Entry...)
		for _, def := range n.invokeDefs {
			actions = append(actions, spawnChildFromInvoke(def))
		}
		if forDefaultEntry.has(n) {
			actions = append(actions, n.Initial().Actions...)
		}
		ids := make([]string, len(n.invokeDefs))
		for i, def := range n.invokeDefs {
			ids[i] = def.ID
		}
		next = resolveActionsAndContext(next, event, scope, actions, internalQueue, ids, true)

		if n.Type == Final {
			parent := n.Parent
			var marker *StateNode
			if parent != nil {
				if parent.Type == Parallel {
					marker = parent
				} else {
					marker = parent.Parent
				}
			}
			completion := n
			if marker != nil {
				completion = marker
			}
			if parent != nil && parent.Type == Compound {
				var out any
				if n.Output != nil {
					out = resolveOutput(n.Output, next.contextAny(), event, scope.Self)
				}
				*internalQueue = append(*internalQueue, DoneStateEvent{StateID: parent.ID, Output: out})
			}
			for marker != nil && marker.Type == Parallel && !completed[marker] && isInFinalState(mutSet, marker) {
				completed[marker] = true
				*internalQueue = append(*internalQueue, DoneStateEvent{StateID: marker.ID})
				completion = marker
				marker = marker.Parent
			}
			if marker != nil {
				continue
			}
			root := next.machineAny().RootNode()
			output := getMachineOutput(next, event, scope, root, completion)
			next = next.clone(snapshotPatch{status: statusPtr(StatusDone), output: anyPtr(output)})
		}
	}
	return next
}

func computeEntrySet(transitions []*TransitionDefinition, history map[string][]*StateNode, forDefaultEntry, toEnter *nodeSet) {
	for _, t := range transitions {
		domain := getTransitionDomain(t, history)
		for _, s := range t.Target {
			if s.Type != History && (t.Source != s || t.Source != domain || t.Reenter) {
				toEnter.add(s)
				forDefaultEntry.add(s)
			}
			addDescendantStatesToEnter(s, history, forDefaultEntry, toEnter)
		}
		for _, s := range getEffectiveTargetStates(t.Target, history) {
			ancestors := getProperAncestors(s, domain)
			if domain != nil && domain.Type == Parallel {
				ancestors = append(ancestors, domain)
			}
			reentrancy := domain
			if t.Source.Parent == nil && t.Reenter {
				reentrancy = nil
			}
			addAncestorStatesToEnter(toEnter, history, forDefaultEntry, ancestors, reentrancy)
		}
	}
}

func addDescendantStatesToEnter(n *StateNode, history map[string][]*StateNode, forDefaultEntry, toEnter *nodeSet) {
	if n.Type == History {
		if hv, ok := history[n.ID]; ok && hv != nil {
			for _, s := range hv {
				toEnter.add(s)
				addDescendantStatesToEnter(s, history, forDefaultEntry, toEnter)
			}
			for _, s := range hv {
				addProperAncestorStatesToEnter(s, n.Parent, toEnter, history, forDefaultEntry)
			}
		} else {
			def := resolveHistoryDefaultTransition(n)
			for _, s := range def.Target {
				toEnter.add(s)
				if n.Parent != nil && def == n.Parent.initial {
					forDefaultEntry.add(n.Parent)
				}
				addDescendantStatesToEnter(s, history, forDefaultEntry, toEnter)
			}
			for _, s := range def.Target {
				addProperAncestorStatesToEnter(s, n.Parent, toEnter, history, forDefaultEntry)
			}
		}
		return
	}
	if n.Type == Compound {
		initial := n.Initial().Target[0]
		if initial.Type != History {
			toEnter.add(initial)
			forDefaultEntry.add(initial)
		}
		addDescendantStatesToEnter(initial, history, forDefaultEntry, toEnter)
		addProperAncestorStatesToEnter(initial, n, toEnter, history, forDefaultEntry)
		return
	}
	if n.Type == Parallel {
		for _, child := range getChildren(n) {
			if child.Type == History {
				continue
			}
			if !someDescendant(toEnter, child) {
				toEnter.add(child)
				forDefaultEntry.add(child)
				addDescendantStatesToEnter(child, history, forDefaultEntry, toEnter)
			}
		}
	}
}

func someDescendant(set *nodeSet, of *StateNode) bool {
	for _, s := range set.items {
		if isDescendant(s, of) {
			return true
		}
	}
	return false
}

func addAncestorStatesToEnter(toEnter *nodeSet, history map[string][]*StateNode, forDefaultEntry *nodeSet, ancestors []*StateNode, reentrancy *StateNode) {
	for _, anc := range ancestors {
		if reentrancy == nil || isDescendant(anc, reentrancy) {
			toEnter.add(anc)
		}
		if anc.Type == Parallel {
			for _, child := range getChildren(anc) {
				if child.Type == History {
					continue
				}
				if !someDescendant(toEnter, child) {
					toEnter.add(child)
					addDescendantStatesToEnter(child, history, forDefaultEntry, toEnter)
				}
			}
		}
	}
}

func addProperAncestorStatesToEnter(n, to *StateNode, toEnter *nodeSet, history map[string][]*StateNode, forDefaultEntry *nodeSet) {
	addAncestorStatesToEnter(toEnter, history, forDefaultEntry, getProperAncestors(n, to), nil)
}

func exitStates(current anyMachineSnapshot, event Event, scope *ActorScope, transitions []*TransitionDefinition, mutSet *nodeSet, history map[string][]*StateNode, internalQueue *[]Event) (anyMachineSnapshot, map[string][]*StateNode) {
	next := current
	toExit := computeExitSet(transitions, mutSet, history)
	sort.SliceStable(toExit, func(i, j int) bool { return toExit[i].Order > toExit[j].Order })

	var changed map[string][]*StateNode
	for _, exitNode := range toExit {
		for _, hn := range getHistoryNodes(exitNode) {
			var pred func(sn *StateNode) bool
			if hn.History == Deep {
				pred = func(sn *StateNode) bool { return isAtomicStateNode(sn) && isDescendant(sn, exitNode) }
			} else {
				pred = func(sn *StateNode) bool { return sn.Parent == exitNode }
			}
			if changed == nil {
				changed = make(map[string][]*StateNode, len(history)+1)
				for k, v := range history {
					changed[k] = v
				}
			}
			var list []*StateNode
			for _, sn := range mutSet.items {
				if pred(sn) {
					list = append(list, sn)
				}
			}
			if list == nil {
				list = []*StateNode{}
			}
			changed[hn.ID] = list
		}
	}
	for _, s := range toExit {
		actions := append(Actions{}, s.Exit...)
		for _, def := range s.invokeDefs {
			actions = append(actions, StopChild(def.ID))
		}
		next = resolveActionsAndContext(next, event, scope, actions, internalQueue, nil, false)
		mutSet.delete(s)
	}
	if changed != nil {
		return next, changed
	}
	return next, history
}
