package xstate

func getHistoryNodes(n *StateNode) []*StateNode {
	var out []*StateNode
	for _, c := range n.ChildStates() {
		if c.Type == History {
			out = append(out, c)
		}
	}
	return out
}

// isDescendant mirrors stateUtils.ts isDescendant; like JS, a nil parent
// (an undefined transition domain) matches every node.
func isDescendant(child, parent *StateNode) bool {
	marker := child
	for marker.Parent != nil && marker.Parent != parent {
		marker = marker.Parent
	}
	return marker.Parent == parent
}

func hasIntersection(a, b []*StateNode) bool {
	set := map[*StateNode]bool{}
	for _, n := range a {
		set[n] = true
	}
	for _, n := range b {
		if set[n] {
			return true
		}
	}
	return false
}

func removeConflictingTransitions(enabled []*TransitionDefinition, set *nodeSet, history map[string][]*StateNode) []*TransitionDefinition {
	var filtered []*TransitionDefinition
	contains := func(t *TransitionDefinition) bool {
		for _, f := range filtered {
			if f == t {
				return true
			}
		}
		return false
	}
	for _, t1 := range enabled {
		preempted := false
		var toRemove []*TransitionDefinition
		for _, t2 := range filtered {
			if hasIntersection(computeExitSet([]*TransitionDefinition{t1}, set, history), computeExitSet([]*TransitionDefinition{t2}, set, history)) {
				if isDescendant(t1.Source, t2.Source) {
					toRemove = append(toRemove, t2)
				} else {
					preempted = true
					break
				}
			}
		}
		if !preempted {
			for _, t3 := range toRemove {
				for i, f := range filtered {
					if f == t3 {
						filtered = append(filtered[:i:i], filtered[i+1:]...)
						break
					}
				}
			}
			if !contains(t1) {
				filtered = append(filtered, t1)
			}
		}
	}
	return filtered
}

func findLeastCommonAncestor(nodes []*StateNode) *StateNode {
	head, tail := nodes[0], nodes[1:]
	for _, anc := range getProperAncestors(head, nil) {
		all := true
		for _, sn := range tail {
			if !isDescendant(sn, anc) {
				all = false
				break
			}
		}
		if all {
			return anc
		}
	}
	return nil
}

func getEffectiveTargetStates(target []*StateNode, history map[string][]*StateNode) []*StateNode {
	if target == nil {
		return nil
	}
	targets := newNodeSet()
	for _, tn := range target {
		if tn.Type == History {
			if hv, ok := history[tn.ID]; ok && hv != nil {
				for _, n := range hv {
					targets.add(n)
				}
			} else {
				for _, n := range getEffectiveTargetStates(resolveHistoryDefaultTransition(tn).Target, history) {
					targets.add(n)
				}
			}
		} else {
			targets.add(tn)
		}
	}
	return targets.items
}

func resolveHistoryDefaultTransition(n *StateNode) *TransitionDefinition {
	if n.Config.Target == "" {
		if n.Parent.Type == Parallel {
			return &TransitionDefinition{Target: []*StateNode{n.Parent}}
		}
		return n.Parent.Initial()
	}
	return &TransitionDefinition{Target: []*StateNode{getStateNodeByPath(n.Parent, n.Config.Target)}}
}

func getTransitionDomain(t *TransitionDefinition, history map[string][]*StateNode) *StateNode {
	targets := getEffectiveTargetStates(t.Target, history)
	if targets == nil {
		return nil
	}
	if !t.Reenter {
		all := true
		for _, target := range targets {
			if !(target == t.Source || isDescendant(target, t.Source)) {
				all = false
				break
			}
		}
		if all {
			return t.Source
		}
	}
	if lca := findLeastCommonAncestor(append(append([]*StateNode{}, targets...), t.Source)); lca != nil {
		return lca
	}
	if t.Reenter {
		return nil
	}
	return t.Source.machine.RootNode()
}

func computeExitSet(transitions []*TransitionDefinition, set *nodeSet, history map[string][]*StateNode) []*StateNode {
	toExit := newNodeSet()
	for _, t := range transitions {
		if len(t.Target) > 0 {
			domain := getTransitionDomain(t, history)
			if t.Reenter && t.Source == domain {
				toExit.add(domain)
			}
			for _, sn := range set.items {
				if isDescendant(sn, domain) {
					toExit.add(sn)
				}
			}
		}
	}
	return toExit.items
}

func areStateNodeCollectionsEqual(prev []*StateNode, next *nodeSet) bool {
	if len(prev) != next.size() {
		return false
	}
	for _, n := range prev {
		if !next.has(n) {
			return false
		}
	}
	return true
}
