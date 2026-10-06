package xstate

import (
	"fmt"
)

// Port of packages/core/src/stateUtils.ts. JS Sets keep insertion order;
// nodeSet does the same.

type nodeSet struct {
	idx   map[*StateNode]bool
	items []*StateNode
}

func newNodeSet(nodes ...*StateNode) *nodeSet {
	s := &nodeSet{idx: map[*StateNode]bool{}}
	for _, n := range nodes {
		s.add(n)
	}
	return s
}

func (s *nodeSet) add(n *StateNode) {
	if !s.idx[n] {
		s.idx[n] = true
		s.items = append(s.items, n)
	}
}

func (s *nodeSet) has(n *StateNode) bool { return s.idx[n] }

func (s *nodeSet) delete(n *StateNode) {
	if !s.idx[n] {
		return
	}
	delete(s.idx, n)
	for i, it := range s.items {
		if it == n {
			s.items = append(s.items[:i:i], s.items[i+1:]...)
			break
		}
	}
}

func (s *nodeSet) size() int { return len(s.items) }

func (s *nodeSet) values() []*StateNode { return append([]*StateNode{}, s.items...) }

func isAtomicStateNode(n *StateNode) bool { return n.Type == Atomic || n.Type == Final }

func getChildren(n *StateNode) []*StateNode {
	var out []*StateNode
	for _, c := range n.ChildStates() {
		if c.Type != History {
			out = append(out, c)
		}
	}
	return out
}

func getProperAncestors(n *StateNode, to *StateNode) []*StateNode {
	var ancestors []*StateNode
	if to == n {
		return ancestors
	}
	for m := n.Parent; m != nil && m != to; m = m.Parent {
		ancestors = append(ancestors, m)
	}
	return ancestors
}

func getAllStateNodes(nodes []*StateNode) *nodeSet {
	set := newNodeSet(nodes...)
	adj := getAdjList(set.items)
	for i := 0; i < len(set.items); i++ {
		s := set.items[i]
		if s.Type == Compound && len(adj.get(s)) == 0 {
			for _, sn := range getInitialStateNodesWithTheirAncestors(s).items {
				set.add(sn)
			}
		} else if s.Type == Parallel {
			for _, child := range getChildren(s) {
				if child.Type == History {
					continue
				}
				if !set.has(child) {
					for _, sn := range getInitialStateNodesWithTheirAncestors(child).items {
						set.add(sn)
					}
				}
			}
		}
	}
	for i := 0; i < len(set.items); i++ {
		for m := set.items[i].Parent; m != nil; m = m.Parent {
			set.add(m)
		}
	}
	return set
}

type adjList struct {
	m     map[*StateNode][]*StateNode
	order []*StateNode
}

func (a *adjList) get(n *StateNode) []*StateNode { return a.m[n] }

func getAdjList(nodes []*StateNode) *adjList {
	a := &adjList{m: map[*StateNode][]*StateNode{}}
	ensure := func(n *StateNode) {
		if _, ok := a.m[n]; !ok {
			a.m[n] = []*StateNode{}
			a.order = append(a.order, n)
		}
	}
	for _, s := range nodes {
		ensure(s)
		if s.Parent != nil {
			ensure(s.Parent)
			a.m[s.Parent] = append(a.m[s.Parent], s)
		}
	}
	return a
}

// ovalue is a state value that keeps JS object-key order.
type ovalue struct {
	isLeaf bool
	leaf   string
	keys   []string
	kids   map[string]*ovalue
}

func (v *ovalue) toStateValue() StateValue {
	if v.isLeaf {
		return v.leaf
	}
	m := make(map[string]any, len(v.keys))
	for _, k := range v.keys {
		m[k] = v.kids[k].toStateValue()
	}
	return m
}

func getValueFromAdj(base *StateNode, adj *adjList) *ovalue {
	children, ok := adj.m[base]
	if !ok {
		return &ovalue{kids: map[string]*ovalue{}}
	}
	if base.Type == Compound {
		if len(children) > 0 {
			if isAtomicStateNode(children[0]) {
				return &ovalue{isLeaf: true, leaf: children[0].Key}
			}
		} else {
			return &ovalue{kids: map[string]*ovalue{}}
		}
	}
	v := &ovalue{kids: map[string]*ovalue{}}
	for _, c := range children {
		if _, dup := v.kids[c.Key]; !dup {
			v.keys = append(v.keys, c.Key)
		}
		v.kids[c.Key] = getValueFromAdj(c, adj)
	}
	return v
}

func getOrderedStateValue(root *StateNode, nodes []*StateNode) *ovalue {
	config := getAllStateNodes(nodes)
	return getValueFromAdj(root, getAdjList(config.items))
}

func getStateValue(root *StateNode, nodes []*StateNode) StateValue {
	return getOrderedStateValue(root, nodes).toStateValue()
}

func isInFinalState(set *nodeSet, n *StateNode) bool {
	switch n.Type {
	case Compound:
		for _, s := range getChildren(n) {
			if s.Type == Final && set.has(s) {
				return true
			}
		}
		return false
	case Parallel:
		for _, s := range getChildren(n) {
			if !isInFinalState(set, s) {
				return false
			}
		}
		return true
	}
	return n.Type == Final
}

// getStateNodes mirrors stateUtils.ts getStateNodes.
func getStateNodes(n *StateNode, value StateValue) []*StateNode {
	switch v := value.(type) {
	case string:
		child := n.States[v]
		if child == nil {
			panic(fmt.Errorf("State '%s' does not exist on '%s'", v, n.ID))
		}
		return []*StateNode{n, child}
	case map[string]any:
		keys := orderKeysByNode(n, v)
		out := []*StateNode{n.machine.RootNode(), n}
		for _, k := range keys {
			out = append(out, getStateNodeChild(n, k))
		}
		for _, k := range keys {
			sub := getStateNodeChild(n, k)
			out = append(out, getStateNodes(sub, v[k])...)
		}
		return out
	case nil:
		panic(fmt.Errorf("Cannot convert undefined or null to object"))
	}
	panic(fmt.Errorf("invalid state value %v", value))
}

// resolveStateValue mirrors stateUtils.ts resolveStateValue.
func resolveStateValue(root *StateNode, value StateValue) StateValue {
	all := getAllStateNodes(getStateNodes(root, value))
	return getStateValue(root, all.items)
}

// ---- state values ----

func toStateValue(v any) StateValue {
	if ms, ok := v.(anyMachineSnapshot); ok {
		return ms.valueAny()
	}
	if s, ok := v.(string); ok {
		return pathToStateValue(toStatePath(s))
	}
	return v
}

func asValueMap(v any) (map[string]any, bool) {
	switch m := v.(type) {
	case map[string]any:
		return m, true
	case map[string]string:
		out := make(map[string]any, len(m))
		for k, s := range m {
			out[k] = s
		}
		return out, true
	}
	return nil, false
}

func matchesState(parentStateID, childStateID StateValue) bool {
	parent := toStateValue(parentStateID)
	child := toStateValue(childStateID)
	if cs, ok := child.(string); ok {
		if ps, ok := parent.(string); ok {
			return cs == ps
		}
		return false
	}
	cm, ok := asValueMap(child)
	if !ok {
		return false
	}
	if ps, ok := parent.(string); ok {
		_, has := cm[ps]
		return has
	}
	pm, ok := asValueMap(parent)
	if !ok {
		return false
	}
	for k, pv := range pm {
		cv, has := cm[k]
		if !has {
			return false
		}
		if !matchesState(pv, cv) {
			return false
		}
	}
	return true
}

// MatchesState mirrors matchesState(parentValue, childValue).
func MatchesState(parent, child StateValue) bool { return matchesState(parent, child) }

func pathToStateValue(path []string) StateValue {
	if len(path) == 1 {
		return path[0]
	}
	value := map[string]any{}
	marker := value
	for i := 0; i < len(path)-1; i++ {
		if i == len(path)-2 {
			marker[path[i]] = path[i+1]
		} else {
			prev := marker
			marker = map[string]any{}
			prev[path[i]] = marker
		}
	}
	return value
}

// PathToStateValue mirrors pathToStateValue(path).
func PathToStateValue(path []string) StateValue { return pathToStateValue(path) }

// GetStateNodes mirrors getStateNodes(root, stateValue).
func GetStateNodes(root *StateNode, value StateValue) []*StateNode { return getStateNodes(root, value) }

// ResolveStateValue mirrors resolveStateValue(rootNode, stateValue).
func ResolveStateValue(root *StateNode, value StateValue) StateValue {
	return resolveStateValue(root, value)
}
