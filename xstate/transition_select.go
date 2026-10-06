package xstate

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// ---- event descriptors ----

var wildcardNotLastRe = regexp.MustCompile(`.*\*.+`)

// matchesEventDescriptor mirrors utils.ts matchesEventDescriptor.
func matchesEventDescriptor(eventType, descriptor string) bool {
	if descriptor == eventType || descriptor == wildcard {
		return true
	}
	if !strings.HasSuffix(descriptor, ".*") {
		return false
	}
	if wildcardNotLastRe.MatchString(descriptor) {
		warn(fmt.Sprintf(`Wildcards can only be the last token of an event descriptor (e.g., "event.*") or the entire event descriptor ("*"). Check the "%s" event.`, descriptor))
	}
	partial := strings.Split(descriptor, ".")
	tokens := strings.Split(eventType, ".")
	for i, p := range partial {
		if p == "*" {
			isLast := i == len(partial)-1
			if !isLast {
				warn(fmt.Sprintf(`Infix wildcards in transition events are not allowed. Check the "%s" transition.`, descriptor))
			}
			return isLast
		}
		if i >= len(tokens) || p != tokens[i] {
			return false
		}
	}
	return true
}

func getCandidates(n *StateNode, eventType string) []*TransitionDefinition {
	exact, hasExact := n.transitions.get(eventType)
	var keys []string
	for _, k := range n.transitions.keys {
		if k != eventType && matchesEventDescriptor(eventType, k) {
			keys = append(keys, k)
		}
	}
	sort.SliceStable(keys, func(i, j int) bool { return len(keys[i]) > len(keys[j]) })
	var wild []*TransitionDefinition
	for _, k := range keys {
		wild = append(wild, n.transitions.m[k]...)
	}
	if hasExact {
		return append(append([]*TransitionDefinition{}, exact...), wild...)
	}
	return wild
}

// candidates memoizes getCandidates per event type (JS memo()).
func (n *StateNode) candidates(eventType string) []*TransitionDefinition {
	c := n.machine.nodeCache()
	c.mu.Lock()
	key := candKey{n, eventType}
	if v, ok := c.candidates[key]; ok {
		c.mu.Unlock()
		return v
	}
	c.mu.Unlock()
	v := getCandidates(n, eventType)
	c.mu.Lock()
	if existing, ok := c.candidates[key]; ok {
		v = existing
	} else {
		c.candidates[key] = v
	}
	c.mu.Unlock()
	return v
}

// next mirrors StateNode.next.
func (n *StateNode) next(snap anyMachineSnapshot, event Event) []*TransitionDefinition {
	eventType := event.EventType()
	for _, candidate := range n.candidates(eventType) {
		guard := candidate.Guard
		passed := true
		if guard != nil {
			func() {
				defer func() {
					if r := recover(); r != nil {
						guardType := ""
						switch g := guard.(type) {
						case GuardRef:
							guardType = g.Type
						case *GuardRef:
							guardType = g.Type
						case *inlineGuard, *builtinGuard:
						default:
							guardType = g.GuardType()
						}
						prefix := ""
						if guardType != "" {
							prefix = "'" + guardType + "' "
						}
						panic(fmt.Errorf("Unable to evaluate guard %sin transition for event '%s' in state node '%s':\n%s", prefix, eventType, n.ID, errorMessage(r)))
					}
				}()
				passed = evaluateGuard(guard, snap.contextAny(), event, snap)
			}()
		}
		if passed {
			return []*TransitionDefinition{candidate}
		}
	}
	return nil
}

// ---- transition selection ----

func transitionAtomicNode(n *StateNode, value string, snap anyMachineSnapshot, event Event) []*TransitionDefinition {
	child := getStateNodeChild(n, value)
	next := child.next(snap, event)
	if len(next) == 0 {
		return n.next(snap, event)
	}
	return next
}

func transitionCompoundNode(n *StateNode, value *ovalue, snap anyMachineSnapshot, event Event) []*TransitionDefinition {
	key := value.keys[0]
	child := getStateNodeChild(n, key)
	next := transitionNode(child, value.kids[key], snap, event)
	if len(next) == 0 {
		return n.next(snap, event)
	}
	return next
}

func transitionParallelNode(n *StateNode, value *ovalue, snap anyMachineSnapshot, event Event) []*TransitionDefinition {
	var all []*TransitionDefinition
	for _, key := range value.keys {
		sub := value.kids[key]
		if sub == nil {
			continue
		}
		child := getStateNodeChild(n, key)
		all = append(all, transitionNode(child, sub, snap, event)...)
	}
	if len(all) == 0 {
		return n.next(snap, event)
	}
	return all
}

func transitionNode(n *StateNode, value *ovalue, snap anyMachineSnapshot, event Event) []*TransitionDefinition {
	if value.isLeaf {
		return transitionAtomicNode(n, value.leaf, snap, event)
	}
	if len(value.keys) == 1 {
		return transitionCompoundNode(n, value, snap, event)
	}
	return transitionParallelNode(n, value, snap, event)
}

// snapshotOrderedValue returns the snapshot value with JS key order.
func snapshotOrderedValue(snap anyMachineSnapshot) *ovalue {
	root := snap.machineAny().RootNode()
	if nodes := snap.nodesAny(); len(nodes) > 0 {
		return getOrderedStateValue(root, nodes)
	}
	return toOrderedValue(root, snap.valueAny())
}

// toOrderedValue orders the keys of a plain state value by document order.
func toOrderedValue(n *StateNode, v StateValue) *ovalue {
	switch val := v.(type) {
	case string:
		return &ovalue{isLeaf: true, leaf: val}
	case map[string]any:
		out := &ovalue{kids: map[string]*ovalue{}}
		out.keys = orderKeysByNode(n, val)
		for _, k := range out.keys {
			child := n
			if c := n.States[k]; c != nil {
				child = c
			}
			out.kids[k] = toOrderedValue(child, val[k])
		}
		return out
	}
	return &ovalue{kids: map[string]*ovalue{}}
}

func orderKeysByNode(n *StateNode, m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.SliceStable(keys, func(i, j int) bool {
		ci, cj := n.States[keys[i]], n.States[keys[j]]
		switch {
		case ci != nil && cj != nil:
			return ci.Order < cj.Order
		case ci != nil:
			return true
		case cj != nil:
			return false
		}
		return keys[i] < keys[j]
	})
	return keys
}

// getTransitionData mirrors StateMachine.getTransitionData.
func getTransitionData(snap anyMachineSnapshot, event Event) []*TransitionDefinition {
	root := snap.machineAny().RootNode()
	return transitionNode(root, snapshotOrderedValue(snap), snap, event)
}
