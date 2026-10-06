package xstate

import (
	"fmt"
	"sort"
	"strings"
)

// StateNode mirrors StateNode in JS. It is not generic: actions and guards
// are type-erased and checked against the machine context at runtime.
type StateNode struct {
	Key         string
	ID          string
	Type        StateType
	Path        []string
	States      map[string]*StateNode
	History     HistoryType // "" when not a history node
	Entry       Actions
	Exit        Actions
	Parent      *StateNode
	Order       int
	Tags        []string
	Meta        any
	Description string
	Output      any
	Config      StateConfig
	Machine     AnyStateMachine

	machine     machineInternal
	childOrder  []string
	transitions *transitionMap
	always      []*TransitionDefinition
	initial     *TransitionDefinition
	initialErr  any
	invokeDefs  []*InvokeDefinition
	after       []*DelayedTransitionDefinition
}

// transitionMap mirrors the JS `Map<descriptor, TransitionDefinition[]>`
// keeping key insertion order.
type transitionMap struct {
	keys []string
	m    map[string][]*TransitionDefinition
}

func newTransitionMap() *transitionMap {
	return &transitionMap{m: map[string][]*TransitionDefinition{}}
}

func (tm *transitionMap) get(k string) ([]*TransitionDefinition, bool) {
	v, ok := tm.m[k]
	return v, ok
}

func (tm *transitionMap) set(k string, v []*TransitionDefinition) {
	if _, ok := tm.m[k]; !ok {
		tm.keys = append(tm.keys, k)
	}
	tm.m[k] = v
}

// StateNodeDefinition mirrors stateNode.definition (JSON-like).
type StateNodeDefinition = map[string]any

func isStateID(s string) bool { return len(s) > 0 && s[0] == '#' }

func newStateNode(config StateConfig, parent *StateNode, key string, m machineInternal) *StateNode {
	n := &StateNode{Config: config, Parent: parent, Key: key, Machine: m, machine: m}
	if parent != nil {
		n.Path = append(append([]string{}, parent.Path...), key)
	} else {
		n.Path = []string{}
	}
	if config.ID != "" {
		n.ID = config.ID
	} else {
		n.ID = strings.Join(append([]string{m.MachineID()}, n.Path...), ".")
	}
	switch {
	case config.Type != "":
		n.Type = config.Type
	case len(config.States) > 0:
		n.Type = Compound
	case config.History != "":
		n.Type = History
	default:
		n.Type = Atomic
	}
	n.Description = config.Description
	n.Order = m.registerNode(n)

	n.States = map[string]*StateNode{}
	for _, child := range config.States {
		cn := newStateNode(child, n, child.Key, m)
		if _, dup := n.States[child.Key]; !dup {
			n.childOrder = append(n.childOrder, child.Key)
		}
		n.States[child.Key] = cn
	}

	if n.Type == Compound && config.Initial == "" {
		first := ""
		if len(n.childOrder) > 0 {
			first = n.childOrder[0]
		}
		panic(fmt.Errorf("No initial state specified for compound state node \"#%s\". Try adding { initial: \"%s\" } to the state config.", n.ID, first))
	}

	if n.Type == History {
		n.History = config.History
		if n.History == "" {
			n.History = Shallow
		}
	}
	n.Entry = append(Actions{}, config.Entry...)
	n.Exit = append(Actions{}, config.Exit...)
	n.Meta = config.Meta
	if n.Type == Final || parent == nil {
		n.Output = config.Output
	}
	n.Tags = append([]string{}, config.Tags...)
	return n
}

// initialize mirrors StateNode._initialize.
func (n *StateNode) initialize() {
	n.invokeDefs = n.computeInvoke()
	n.after = getDelayedTransitions(n)
	n.transitions = formatTransitions(n)
	if n.Config.Always != nil {
		for _, t := range toTransitionConfigArray(n.Config.Always) {
			n.always = append(n.always, formatTransition(n, "", t))
		}
	}
	func() {
		defer func() {
			if r := recover(); r != nil {
				n.initialErr = r
			}
		}()
		n.initial = formatInitialTransition(n, n.Config.Initial, n.Config.initial)
	}()
	for _, k := range n.childOrder {
		n.States[k].initialize()
	}
}

// ChildStates returns child nodes in document order.
func (n *StateNode) ChildStates() []*StateNode {
	out := make([]*StateNode, 0, len(n.childOrder))
	for _, k := range n.childOrder {
		out = append(out, n.States[k])
	}
	return out
}

// On mirrors stateNode.on: event descriptor -> transition definitions.
func (n *StateNode) On() map[string][]*TransitionDefinition {
	out := map[string][]*TransitionDefinition{}
	for _, k := range n.transitions.keys {
		ts := n.transitions.m[k]
		if len(ts) == 0 {
			continue
		}
		out[k] = append([]*TransitionDefinition{}, ts...)
	}
	return out
}

// Transitions mirrors `stateNode.transitions`: every descriptor, including
// descriptors without transitions.
func (n *StateNode) Transitions() map[string][]*TransitionDefinition {
	out := map[string][]*TransitionDefinition{}
	for _, k := range n.transitions.keys {
		out[k] = append([]*TransitionDefinition{}, n.transitions.m[k]...)
	}
	return out
}

// TransitionList mirrors `[...stateNode.transitions.values()].flat()`: all
// transitions in descriptor insertion order.
func (n *StateNode) TransitionList() []*TransitionDefinition {
	var out []*TransitionDefinition
	for _, k := range n.transitions.keys {
		out = append(out, n.transitions.m[k]...)
	}
	return out
}

// After mirrors stateNode.after.
func (n *StateNode) After() []*DelayedTransitionDefinition {
	return append([]*DelayedTransitionDefinition{}, n.after...)
}

// Always mirrors stateNode.always.
func (n *StateNode) Always() []*TransitionDefinition {
	if n.always == nil {
		return nil
	}
	return append([]*TransitionDefinition{}, n.always...)
}

// Initial mirrors stateNode.initial.
func (n *StateNode) Initial() *TransitionDefinition {
	if n.initialErr != nil {
		panic(n.initialErr)
	}
	return n.initial
}

// Invoke mirrors stateNode.invoke.
func (n *StateNode) Invoke() []*InvokeDefinition {
	return append([]*InvokeDefinition{}, n.invokeDefs...)
}

// Events mirrors stateNode.events (own + descendants), sorted.
func (n *StateNode) Events() []string {
	set := map[string]bool{}
	for _, e := range n.OwnEvents() {
		set[e] = true
	}
	for _, c := range n.ChildStates() {
		for _, e := range c.Events() {
			set[e] = true
		}
	}
	return sortedKeys(set)
}

// OwnEvents mirrors stateNode.ownEvents, sorted.
func (n *StateNode) OwnEvents() []string {
	set := map[string]bool{}
	for _, k := range n.transitions.keys {
		for _, t := range n.transitions.m[k] {
			if !(t.Target == nil && len(t.Actions) == 0 && !t.Reenter) {
				set[k] = true
				break
			}
		}
	}
	return sortedKeys(set)
}

func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
