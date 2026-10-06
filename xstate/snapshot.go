package xstate

import (
	"sort"
)

// Status mirrors snapshot.status.
type Status string

const (
	StatusActive  Status = "active"
	StatusDone    Status = "done"
	StatusError   Status = "error"
	StatusStopped Status = "stopped"
)

// Snapshot is implemented by every actor snapshot.
type Snapshot interface {
	GetStatus() Status
	GetOutput() any
	GetError() any
}

// StateValue is either a string (atomic child key) or a
// map[string]any whose values are StateValues (compound/parallel).
type StateValue = any

// MachineSnapshot mirrors MachineSnapshot in JS.
type MachineSnapshot[C any] struct {
	Status       Status
	Value        StateValue
	Context      C
	Output       any
	Error        any
	Children     map[string]ActorRef
	Tags         []string // sorted
	HistoryValue map[string][]*StateNode
	Machine      *StateMachine[C]

	nodes      []*StateNode
	childOrder []string
}

func (s *MachineSnapshot[C]) GetStatus() Status { return s.Status }
func (s *MachineSnapshot[C]) GetOutput() any    { return s.Output }
func (s *MachineSnapshot[C]) GetError() any     { return s.Error }

// Matches mirrors snapshot.matches(value). value: "a.b" or map form.
func (s *MachineSnapshot[C]) Matches(value StateValue) bool {
	return matchesState(value, s.Value)
}

// HasTag mirrors snapshot.hasTag(tag).
func (s *MachineSnapshot[C]) HasTag(tag string) bool {
	for _, t := range s.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

// Can mirrors snapshot.can(event).
func (s *MachineSnapshot[C]) Can(event Event) bool {
	if s.Machine == nil {
		warn("state.can(...) used outside of a machine-created State object; this will always return false.")
		return false
	}
	data := getTransitionData(s, event)
	for _, t := range data {
		if t.Target != nil || len(t.Actions) > 0 {
			return true
		}
	}
	return false
}

// GetMeta mirrors snapshot.getMeta(): state id -> meta.
func (s *MachineSnapshot[C]) GetMeta() map[string]any {
	out := map[string]any{}
	for _, n := range s.nodes {
		if n.Meta != nil {
			out[n.ID] = n.Meta
		}
	}
	return out
}

// ToJSON mirrors snapshot.toJSON().
func (s *MachineSnapshot[C]) ToJSON() map[string]any {
	return map[string]any{
		"status":       s.Status,
		"value":        s.Value,
		"context":      s.Context,
		"output":       s.Output,
		"error":        s.Error,
		"children":     s.Children,
		"historyValue": s.HistoryValue,
		"tags":         append([]string{}, s.Tags...),
	}
}

// StateNodes returns the active state nodes (JS `snapshot._nodes`).
func (s *MachineSnapshot[C]) StateNodes() []*StateNode { return append([]*StateNode{}, s.nodes...) }

// IsMachineSnapshot mirrors isMachineSnapshot(value).
func IsMachineSnapshot(v any) bool {
	_, ok := v.(anyMachineSnapshot)
	return ok
}

// ---- type-erased machine snapshot used by the engine ----

// snapshotConfig mirrors the JS StateConfig passed to createMachineSnapshot.
type snapshotConfig struct {
	status       Status
	output       any
	err          any
	context      any
	nodes        []*StateNode
	children     map[string]ActorRef
	childOrder   []string
	historyValue map[string][]*StateNode
}

// snapshotPatch mirrors the partial config of cloneMachineSnapshot. nil
// pointer fields are left unchanged.
type snapshotPatch struct {
	status       *Status
	output       *any
	err          *any
	context      *any
	nodes        []*StateNode
	setNodes     bool
	children     *childSet
	historyValue map[string][]*StateNode
	setHistory   bool
}

// childSet is an insertion-ordered children map (JS object semantics).
type childSet struct {
	order []string
	m     map[string]ActorRef
}

func (c *childSet) with(id string, ref ActorRef) *childSet {
	out := &childSet{m: make(map[string]ActorRef, len(c.m)+1)}
	for k, v := range c.m {
		out.m[k] = v
	}
	out.order = append(out.order, c.order...)
	if _, ok := out.m[id]; !ok {
		out.order = append(out.order, id)
	}
	out.m[id] = ref
	return out
}

func (c *childSet) without(id string) *childSet {
	out := &childSet{m: make(map[string]ActorRef, len(c.m))}
	for k, v := range c.m {
		if k != id {
			out.m[k] = v
		}
	}
	for _, k := range c.order {
		if k != id {
			out.order = append(out.order, k)
		}
	}
	return out
}

type anyMachineSnapshot interface {
	Snapshot
	contextAny() any
	children() *childSet
	nodesAny() []*StateNode
	machineAny() machineInternal
	valueAny() StateValue
	historyAny() map[string][]*StateNode
	clone(p snapshotPatch) anyMachineSnapshot
}

func (s *MachineSnapshot[C]) contextAny() any                     { return s.Context }
func (s *MachineSnapshot[C]) nodesAny() []*StateNode              { return s.nodes }
func (s *MachineSnapshot[C]) machineAny() machineInternal         { return s.Machine }
func (s *MachineSnapshot[C]) valueAny() StateValue                { return s.Value }
func (s *MachineSnapshot[C]) historyAny() map[string][]*StateNode { return s.HistoryValue }

// children returns the children in insertion order.
func (s *MachineSnapshot[C]) children() *childSet {
	cs := &childSet{m: s.Children}
	if cs.m == nil {
		cs.m = map[string]ActorRef{}
	}
	seen := map[string]bool{}
	for _, k := range s.childOrder {
		if _, ok := cs.m[k]; ok && !seen[k] {
			cs.order = append(cs.order, k)
			seen[k] = true
		}
	}
	if len(cs.order) != len(cs.m) {
		var rest []string
		for k := range cs.m {
			if !seen[k] {
				rest = append(rest, k)
			}
		}
		sort.Strings(rest)
		cs.order = append(cs.order, rest...)
	}
	return cs
}

func (s *MachineSnapshot[C]) clone(p snapshotPatch) anyMachineSnapshot {
	cs := s.children()
	cfg := snapshotConfig{
		status:       s.Status,
		output:       s.Output,
		err:          s.Error,
		context:      s.Context,
		nodes:        s.nodes,
		children:     cs.m,
		childOrder:   cs.order,
		historyValue: s.HistoryValue,
	}
	if p.status != nil {
		cfg.status = *p.status
	}
	if p.output != nil {
		cfg.output = *p.output
	}
	if p.err != nil {
		cfg.err = *p.err
	}
	if p.context != nil {
		cfg.context = *p.context
	}
	if p.setNodes {
		cfg.nodes = p.nodes
	}
	if p.children != nil {
		cfg.children = p.children.m
		cfg.childOrder = p.children.order
	}
	if p.setHistory {
		cfg.historyValue = p.historyValue
	}
	return s.Machine.createSnapshot(cfg)
}

func statusPtr(s Status) *Status { return &s }
func anyPtr(v any) *any          { return &v }

// createSnapshot mirrors State.ts createMachineSnapshot.
func (m *StateMachine[C]) createSnapshot(cfg snapshotConfig) anyMachineSnapshot {
	return m.newSnapshot(cfg)
}

func (m *StateMachine[C]) newSnapshot(cfg snapshotConfig) *MachineSnapshot[C] {
	children := cfg.children
	if children == nil {
		children = map[string]ActorRef{}
	}
	history := cfg.historyValue
	if history == nil {
		history = map[string][]*StateNode{}
	}
	tagSet := map[string]bool{}
	for _, n := range cfg.nodes {
		for _, t := range n.Tags {
			tagSet[t] = true
		}
	}
	return &MachineSnapshot[C]{
		Status:       cfg.status,
		Output:       cfg.output,
		Error:        cfg.err,
		Machine:      m,
		Context:      castTo[C](cfg.context),
		nodes:        cfg.nodes,
		Value:        getStateValue(m.Root, cfg.nodes),
		Tags:         sortedKeys(tagSet),
		Children:     children,
		childOrder:   cfg.childOrder,
		HistoryValue: history,
	}
}
