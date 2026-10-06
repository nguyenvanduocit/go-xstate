package xstate

import (
	"errors"
	"reflect"
	"sync"
)

// Port of the ActorLogic half of StateMachine.ts.

type candKey struct {
	n  *StateNode
	ev string
}

type nodeCacheT struct {
	mu         sync.Mutex
	candidates map[candKey][]*TransitionDefinition
}

// The cache shares the machine's lifetime, including when multiple actors use it.
func (m *StateMachine[C]) nodeCache() *nodeCacheT { return m.cache }

func (m *StateMachine[C]) logic() {}

func (m *StateMachine[C]) typedLogic() *MachineSnapshot[C] { return nil }

func (m *StateMachine[C]) initialSnapshot(scope *ActorScope, input any) Snapshot {
	return m.GetInitialSnapshot(scope, input)
}

func (m *StateMachine[C]) transitionAny(s Snapshot, e Event, scope *ActorScope) Snapshot {
	return m.Transition(s.(*MachineSnapshot[C]), e, scope)
}

func (m *StateMachine[C]) hasStart() bool { return true }

// startAny mirrors StateMachine.start: it starts the active children.
func (m *StateMachine[C]) startAny(s Snapshot, _ *ActorScope) {
	snap := s.(*MachineSnapshot[C])
	cs := snap.children()
	for _, k := range cs.order {
		child := cs.m[k]
		if isNilRef(child) {
			continue
		}
		if child.AnySnapshot().GetStatus() == StatusActive {
			startRef(child)
		}
	}
}

func (m *StateMachine[C]) persistAny(s Snapshot) any {
	return m.GetPersistedSnapshot(s.(*MachineSnapshot[C]))
}

func (m *StateMachine[C]) hasRestore() bool { return true }

func (m *StateMachine[C]) restoreAny(p any, scope *ActorScope) Snapshot {
	return m.RestoreSnapshot(p, scope)
}

func (m *StateMachine[C]) newActorRef(o actorOptions) ActorRef {
	return newActor[*MachineSnapshot[C]](m, m, o)
}

// defaultContext mirrors `typeof context !== 'function' && context ? context : {}`.
func (m *StateMachine[C]) defaultContext() any {
	ctx := any(m.Config.Context)
	rv := reflect.ValueOf(&m.Config.Context).Elem()
	if rv.Kind() == reflect.Map && rv.IsNil() {
		return reflect.MakeMap(rv.Type()).Interface()
	}
	return ctx
}

// ResolveStateConfig mirrors the argument of machine.resolveState.
type ResolveStateConfig[C any] struct {
	Value   StateValue
	Context C
}

// ResolveState mirrors machine.resolveState({ value, context }).
func (m *StateMachine[C]) ResolveState(cfg ResolveStateConfig[C]) *MachineSnapshot[C] {
	resolved := resolveStateValue(m.Root, cfg.Value)
	nodes := getAllStateNodes(getStateNodes(m.Root, resolved))
	status := StatusActive
	if isInFinalState(nodes, m.Root) {
		status = StatusDone
	}
	var ctx any = cfg.Context
	rv := reflect.ValueOf(&cfg.Context).Elem()
	if (rv.Kind() == reflect.Map || rv.Kind() == reflect.Interface) && rv.IsNil() {
		if rv.Kind() == reflect.Map {
			ctx = reflect.MakeMap(rv.Type()).Interface()
		} else {
			ctx = map[string]any{}
		}
	}
	return m.newSnapshot(snapshotConfig{
		nodes:   nodes.items,
		context: ctx,
		status:  status,
	})
}

// GetInitialSnapshot mirrors machine.getInitialSnapshot(actorScope, input).
func (m *StateMachine[C]) GetInitialSnapshot(scope *ActorScope, input any) (result *MachineSnapshot[C]) {
	initEvent := InitEvent{Input: input}
	internalQueue := &[]Event{}
	var snap anyMachineSnapshot = m.newSnapshot(snapshotConfig{
		context: m.defaultContext(),
		nodes:   []*StateNode{m.Root},
		status:  StatusActive,
	})
	defer func() {
		if r := recover(); r != nil {
			result = snap.clone(snapshotPatch{status: statusPtr(StatusError), err: anyPtr(r)}).(*MachineSnapshot[C])
		}
	}()
	snap = m.getPreInitialState(scope, initEvent, internalQueue)
	step := initialMicrostep(m.Root, snap, scope, initEvent, internalQueue)
	macro := macrostep(step.snapshot, initEvent, scope, internalQueue)
	return macro.snapshot.(*MachineSnapshot[C])
}

func (m *StateMachine[C]) getPreInitialState(scope *ActorScope, initEvent InitEvent, internalQueue *[]Event) anyMachineSnapshot {
	pre := m.newSnapshot(snapshotConfig{
		context: m.defaultContext(),
		nodes:   []*StateNode{m.Root},
		status:  StatusActive,
	})
	if m.Config.ContextFn != nil {
		fn := m.Config.ContextFn
		assignment := Assign(func(a AssignArgs[C]) C {
			var input any
			if ie, ok := a.Event.(InitEvent); ok {
				input = ie.Input
			}
			return fn(ContextArgs{Input: input, Self: a.Self, Spawn: a.Spawn})
		})
		return resolveActionsAndContext(pre, initEvent, scope, Actions{assignment}, internalQueue, nil, false)
	}
	return pre
}

// Transition mirrors machine.transition(snapshot, event, actorScope).
func (m *StateMachine[C]) Transition(s *MachineSnapshot[C], e Event, scope *ActorScope) *MachineSnapshot[C] {
	return macrostep(s, e, scope, &[]Event{}).snapshot.(*MachineSnapshot[C])
}

// Microstep mirrors machine.microstep(snapshot, event, actorScope).
func (m *StateMachine[C]) Microstep(s *MachineSnapshot[C], e Event, scope *ActorScope) []*MachineSnapshot[C] {
	res := macrostep(s, e, scope, &[]Event{})
	out := make([]*MachineSnapshot[C], len(res.microsteps))
	for i, step := range res.microsteps {
		out[i] = step.snapshot.(*MachineSnapshot[C])
	}
	return out
}

// GetPersistedSnapshot mirrors machine.getPersistedSnapshot(snapshot).
func (m *StateMachine[C]) GetPersistedSnapshot(s *MachineSnapshot[C]) any {
	children := map[string]any{}
	cs := s.children()
	for _, id := range cs.order {
		child := cs.m[id]
		if isNilRef(child) {
			continue
		}
		cc := coreOf(child)
		if _, isString := child.Src().(string); !isString {
			panic(errors.New("An inline child actor cannot be persisted."))
		}
		entry := map[string]any{
			"src": child.Src(),
		}
		if cc != nil {
			entry["snapshot"] = cc.logic.persistAny(cc.snapshot)
			entry["systemId"] = nilIfEmpty(cc.systemID)
			entry["syncSnapshot"] = cc.syncSnapshot
		}
		children[id] = entry
	}
	history := map[string]any{}
	for k, v := range s.HistoryValue {
		list := make([]any, len(v))
		for i, n := range v {
			list[i] = map[string]any{"id": n.ID}
		}
		history[k] = list
	}
	return map[string]any{
		"status":       s.Status,
		"value":        s.Value,
		"output":       s.Output,
		"error":        s.Error,
		"context":      persistContext(s.Context),
		"children":     children,
		"historyValue": history,
	}
}

// RestoreSnapshot mirrors machine.restoreSnapshot(persisted, actorScope).
func (m *StateMachine[C]) RestoreSnapshot(persisted any, scope *ActorScope) *MachineSnapshot[C] {
	var p map[string]any
	switch v := persisted.(type) {
	case map[string]any:
		p = v
	case *MachineSnapshot[C]:
		p = liveSnapshotAsPersisted(v)
	case anyMachineSnapshot:
		p = map[string]any{"value": v.valueAny(), "status": v.GetStatus(), "context": v.contextAny(), "output": v.GetOutput(), "error": v.GetError()}
	default:
		panic(errors.New("xstate: unsupported persisted snapshot"))
	}
	cs := &childSet{m: map[string]ActorRef{}}
	if pc, ok := p["children"].(map[string]any); ok {
		for _, id := range sortedKeys(pc) {
			data, _ := pc[id].(map[string]any)
			if data == nil {
				continue
			}
			src := data["src"]
			var logic ActorLogic
			switch s := src.(type) {
			case string:
				logic = m.resolveReferencedActor(s)
			case ActorLogic:
				logic = s
			}
			if logic == nil {
				continue
			}
			systemID, _ := data["systemId"].(string)
			syncSnapshot, _ := data["syncSnapshot"].(bool)
			var self ActorRef
			if scope != nil {
				self = scope.Self
			}
			ref := createChildActor(logic, childOptions{
				id:           id,
				parent:       self,
				syncSnapshot: syncSnapshot,
				snapshot:     data["snapshot"],
				src:          src,
				systemID:     systemID,
			})
			cs = cs.with(id, ref)
		}
	}
	history := reviveHistoryValue(m, p["historyValue"])
	value := p["value"]
	nodes := getAllStateNodes(getStateNodes(m.Root, value))
	status := StatusActive
	if st, ok := p["status"].(Status); ok {
		status = st
	} else if st, ok := p["status"].(string); ok {
		status = Status(st)
	}
	ctx := restoreContext[C](p["context"], cs.m)
	return m.newSnapshot(snapshotConfig{
		status:       status,
		output:       p["output"],
		err:          p["error"],
		context:      ctx,
		nodes:        nodes.items,
		children:     cs.m,
		childOrder:   cs.order,
		historyValue: history,
	})
}

// liveSnapshotAsPersisted mirrors restoreSnapshot called with a live
// snapshot object: its children are actor refs, whose `snapshot` property
// is undefined (fresh actors are created from `src`).
func liveSnapshotAsPersisted[C any](s *MachineSnapshot[C]) map[string]any {
	children := map[string]any{}
	cs := s.children()
	for _, id := range cs.order {
		child := cs.m[id]
		if isNilRef(child) {
			continue
		}
		entry := map[string]any{"src": child.Src()}
		if cc := coreOf(child); cc != nil {
			entry["systemId"] = nilIfEmpty(cc.systemID)
		}
		children[id] = entry
	}
	history := map[string]any{}
	for k, v := range s.HistoryValue {
		list := make([]any, len(v))
		for i, n := range v {
			list[i] = n
		}
		history[k] = list
	}
	return map[string]any{
		"status":       s.Status,
		"value":        s.Value,
		"output":       s.Output,
		"error":        s.Error,
		"context":      s.Context,
		"children":     children,
		"historyValue": history,
	}
}

func reviveHistoryValue(m machineInternal, hv any) map[string][]*StateNode {
	revived := map[string][]*StateNode{}
	obj, ok := hv.(map[string]any)
	if !ok {
		return revived
	}
	for key, arr := range obj {
		items, _ := arr.([]any)
		for _, item := range items {
			var resolved *StateNode
			switch it := item.(type) {
			case *StateNode:
				resolved = it
			case map[string]any:
				id, _ := it["id"].(string)
				func() {
					defer func() {
						if r := recover(); r != nil {
							warn("Could not resolve StateNode for id: " + id)
						}
					}()
					resolved = m.getStateNodeByID(id)
				}()
			}
			if resolved == nil {
				continue
			}
			revived[key] = append(revived[key], resolved)
		}
	}
	return revived
}
