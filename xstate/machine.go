package xstate

import (
	"fmt"
	"regexp"
	"strconv"
)

// ---- StateMachine ----

// AnyStateMachine is implemented by every *StateMachine[C].
type AnyStateMachine interface {
	ActorLogic
	MachineID() string
	RootNode() *StateNode
	GetStateNodeByID(id string) *StateNode
}

// machineInternal is the type-erased machine used by the engine.
type machineInternal interface {
	AnyStateMachine
	registerNode(n *StateNode) int
	getStateNodeByID(id string) *StateNode
	impl() *Implementations
	machineOptions() MachineOptions
	machineVersion() string
	createSnapshot(cfg snapshotConfig) anyMachineSnapshot
	resolveReferencedActor(src string) ActorLogic
	nodeCache() *nodeCacheT
}

// StateMachine mirrors StateMachine in JS.
type StateMachine[C any] struct {
	ID              string
	Version         string
	Root            *StateNode
	States          map[string]*StateNode
	Config          MachineConfig[C]
	Implementations Implementations

	idMap   map[string]*StateNode
	idCount int
	options MachineOptions
	cache   *nodeCacheT
}

// CreateMachine mirrors createMachine(config, implementations). It panics on
// an invalid config with the same message XState JS throws.
func CreateMachine[C any](config MachineConfig[C], impl ...Implementations) *StateMachine[C] {
	var im Implementations
	if len(impl) > 0 {
		im = impl[0]
	}
	return newStateMachine(config, im)
}

func newStateMachine[C any](config MachineConfig[C], im Implementations) *StateMachine[C] {
	m := &StateMachine[C]{Config: config, idMap: map[string]*StateNode{}, cache: &nodeCacheT{candidates: map[candKey][]*TransitionDefinition{}}}
	m.ID = config.ID
	if m.ID == "" {
		m.ID = "(machine)"
	}
	m.Implementations = Implementations{
		Actions: copyMap(im.Actions),
		Guards:  copyMap(im.Guards),
		Actors:  copyMap(im.Actors),
		Delays:  copyMap(im.Delays),
	}
	m.Version = config.Version
	if config.options != nil {
		m.options = *config.options
	}
	rootCfg := StateConfig{
		Key:         m.ID,
		ID:          config.ID,
		Type:        config.Type,
		Initial:     config.Initial,
		States:      config.States,
		Invoke:      config.Invoke,
		On:          config.On,
		Entry:       config.Entry,
		Exit:        config.Exit,
		OnDone:      config.OnDone,
		After:       config.After,
		Always:      config.Always,
		Meta:        config.Meta,
		Output:      config.Output,
		Tags:        config.Tags,
		Description: config.Description,
		initial:     config.initial,
	}
	m.Root = newStateNode(rootCfg, nil, m.ID, m)
	m.Root.initialize()
	formatRouteTransitions(m.Root)
	m.States = m.Root.States
	return m
}

func copyMap[V any](in map[string]V) map[string]V {
	out := make(map[string]V, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func (m *StateMachine[C]) MachineID() string { return m.ID }

func (m *StateMachine[C]) RootNode() *StateNode { return m.Root }

func (m *StateMachine[C]) registerNode(n *StateNode) int {
	order := m.idCount
	m.idCount++
	m.idMap[n.ID] = n
	return order
}

func (m *StateMachine[C]) impl() *Implementations { return &m.Implementations }

func (m *StateMachine[C]) machineOptions() MachineOptions { return m.options }

func (m *StateMachine[C]) machineVersion() string { return m.Version }

// Provide mirrors machine.provide(implementations).
func (m *StateMachine[C]) Provide(impl Implementations) *StateMachine[C] {
	merged := Implementations{
		Actions: mergeMaps(m.Implementations.Actions, impl.Actions),
		Guards:  mergeMaps(m.Implementations.Guards, impl.Guards),
		Actors:  mergeMaps(m.Implementations.Actors, impl.Actors),
		Delays:  mergeMaps(m.Implementations.Delays, impl.Delays),
	}
	return newStateMachine(m.Config, merged)
}

func mergeMaps[V any](a, b map[string]V) map[string]V {
	out := make(map[string]V, len(a)+len(b))
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		out[k] = v
	}
	return out
}

// Events mirrors machine.events, sorted.
func (m *StateMachine[C]) Events() []string { return m.Root.Events() }

// GetStateNodeByID mirrors machine.getStateNodeById(id).
func (m *StateMachine[C]) GetStateNodeByID(id string) *StateNode { return m.getStateNodeByID(id) }

func (m *StateMachine[C]) getStateNodeByID(stateID string) *StateNode {
	fullPath := toStatePath(stateID)
	relative := fullPath[1:]
	resolvedID := fullPath[0]
	if isStateID(resolvedID) {
		resolvedID = resolvedID[1:]
	}
	n := m.idMap[resolvedID]
	if n == nil {
		panic(fmt.Errorf("Child state node '#%s' does not exist on machine '%s'", resolvedID, m.ID))
	}
	return getStateNodeByPathArray(n, relative)
}

// Definition mirrors machine.definition.
func (m *StateMachine[C]) Definition() StateNodeDefinition { return m.Root.Definition() }

// ToJSON mirrors machine.toJSON().
func (m *StateMachine[C]) ToJSON() map[string]any { return m.Root.Definition() }

// MarshalJSON serializes the machine definition.
func (m *StateMachine[C]) MarshalJSON() ([]byte, error) { return jsonMarshal(m.ToJSON()) }

var invokeSrcRe = regexp.MustCompile(`^xstate\.invoke\.(\d+)\.(.*)`)

func (m *StateMachine[C]) resolveReferencedActor(src string) ActorLogic {
	match := invokeSrcRe.FindStringSubmatch(src)
	if match == nil {
		return m.Implementations.Actors[src]
	}
	idx, _ := strconv.Atoi(match[1])
	node := m.getStateNodeByID(match[2])
	if idx >= len(node.Config.Invoke) {
		return nil
	}
	cfg := node.Config.Invoke[idx]
	if cfg.Logic != nil {
		return cfg.Logic
	}
	if cfg.Src != "" {
		return m.Implementations.Actors[cfg.Src]
	}
	return nil
}

// ResolveReferencedActor mirrors resolveReferencedActor(machine, src).
func ResolveReferencedActor[C any](m *StateMachine[C], src string) ActorLogic {
	return m.resolveReferencedActor(src)
}
