package graph

import (
	"errors"
	"fmt"
	"slices"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// GetStateNodes mirrors getStateNodes(stateNode): all descendant state nodes
// of stateNode (pass machine.Root for a machine), depth-first in document
// order.
func GetStateNodes(stateNode *xs.StateNode) []*xs.StateNode {
	var nodes []*xs.StateNode
	for _, child := range stateNode.ChildStates() {
		nodes = append(nodes, child)
		nodes = append(nodes, GetStateNodes(child)...)
	}
	return nodes
}

// SerializeSnapshot mirrors serializeSnapshot(snapshot):
// JSON.stringify({ value, context }) with context omitted when empty.
func SerializeSnapshot[C any](snapshot *xs.MachineSnapshot[C]) string {
	return serializeValueContext(stateValueJSON(snapshot.Value, rootNodeOf(snapshot.StateNodes())), snapshot.Context)
}

// serializeValueContext builds JSON.stringify({ value, context }) from the
// value's JSON; context is omitted when Object.keys(context) is empty.
func serializeValueContext(valueJSON string, context any) string {
	out := `{"value":` + valueJSON
	if jsonHasKeys(context) {
		out += `,"context":` + jsonStringify(context)
	}
	return out + "}"
}

// machineSnapshotView is the part of *xs.MachineSnapshot[C] the graph
// utilities read without knowing C.
type machineSnapshotView interface {
	xs.Snapshot
	StateNodes() []*xs.StateNode
	GetMeta() map[string]any
	ToJSON() map[string]any
}

// serializeMachineSnapshot is serializeSnapshot for a snapshot of unknown
// context type.
func serializeMachineSnapshot(snapshot any) string {
	view, ok := snapshot.(machineSnapshotView)
	if !ok {
		return jsonStringify(snapshot)
	}
	j := view.ToJSON()
	return serializeValueContext(stateValueJSON(j["value"], rootNodeOf(view.StateNodes())), j["context"])
}

// getAllOwnEventDescriptors mirrors __unsafe_getAllOwnEventDescriptors:
// the own events of every active state node, deduplicated in first-seen
// order.
func getAllOwnEventDescriptors(snapshot any) []string {
	view, ok := snapshot.(machineSnapshotView)
	if !ok {
		return nil
	}
	seen := map[string]bool{}
	var out []string
	for _, sn := range view.StateNodes() {
		for _, e := range sn.OwnEvents() {
			if !seen[e] {
				seen[e] = true
				out = append(out, e)
			}
		}
	}
	return out
}

// eventsWithOwnDescriptors mirrors the `events` resolver shared by
// createDefaultMachineOptions and createTestModel: for every own event
// descriptor of state, the provided events of that type, or `{ type }`.
func eventsWithOwnDescriptors[S xs.Snapshot](state S, provided []xs.Event) []xs.Event {
	var out []xs.Event
	for _, eventType := range getAllOwnEventDescriptors(state) {
		matched := false
		for _, ev := range provided {
			if ev.EventType() == eventType {
				out = append(out, ev)
				matched = true
			}
		}
		if !matched {
			out = append(out, xs.Ev(eventType))
		}
	}
	return out
}

// createDefaultMachineOptions mirrors createDefaultMachineOptions(machine,
// options).
func createDefaultMachineOptions[S xs.Snapshot](logic xs.TypedActorLogic[S], options *TraversalOptions[S]) TraversalOptions[S] {
	var opts TraversalOptions[S]
	if options != nil {
		opts = *options
	}
	return TraversalOptions[S]{
		SerializeState: func(state S, _ xs.Event, _ S) string { return serializeMachineSnapshot(state) },
		SerializeEvent: jsonStringifyEvent,
		EventsFn: func(state S) []xs.Event {
			return eventsWithOwnDescriptors(state, opts.events(state))
		},
		FromState: xs.GetInitialSnapshot(logic, opts.Input),
	}.overlay(withoutEvents(opts))
}

// withoutEvents mirrors `const { events, ...otherOptions } = options`.
func withoutEvents[S xs.Snapshot](o TraversalOptions[S]) TraversalOptions[S] {
	o.Events, o.EventsFn = nil, nil
	return o
}

// createDefaultLogicOptions mirrors createDefaultLogicOptions().
func createDefaultLogicOptions[S xs.Snapshot]() TraversalOptions[S] {
	return TraversalOptions[S]{
		SerializeState: func(state S, _ xs.Event, _ S) string { return jsonStringify(state) },
		SerializeEvent: jsonStringifyEvent,
	}
}

// ToDirectedGraph mirrors toDirectedGraph(stateMachine) (pass machine.Root
// for a machine).
func ToDirectedGraph(stateNode *xs.StateNode) *DirectedGraphNode {
	var edges []*DirectedGraphEdge
	for transitionIndex, t := range stateNode.TransitionList() {
		targets := t.Target
		if targets == nil {
			targets = []*xs.StateNode{stateNode}
		}
		for targetIndex, target := range targets {
			edges = append(edges, &DirectedGraphEdge{
				ID:         fmt.Sprintf("%s:%d:%d", stateNode.ID, transitionIndex, targetIndex),
				Source:     stateNode,
				Target:     target,
				Transition: t,
				Label:      DirectedGraphLabel{Text: t.EventType},
			})
		}
	}

	children := []*DirectedGraphNode{}
	for _, child := range stateNode.ChildStates() {
		children = append(children, ToDirectedGraph(child))
	}

	return &DirectedGraphNode{
		ID:        stateNode.ID,
		StateNode: stateNode,
		Children:  children,
		Edges:     edges,
	}
}

func isMachineLogic(logic xs.ActorLogic) bool {
	_, ok := logic.(xs.AnyStateMachine)
	return ok
}

// resolveTraversalOptions mirrors resolveTraversalOptions(logic,
// traversalOptions, defaultOptions). A nil pointer is JS `undefined`.
func resolveTraversalOptions[S xs.Snapshot](logic xs.TypedActorLogic[S], traversalOptions, defaultOptions *TraversalOptions[S]) TraversalOptions[S] {
	var resolvedDefaultOptions *TraversalOptions[S]
	if defaultOptions != nil {
		resolvedDefaultOptions = defaultOptions
	} else if isMachineLogic(logic) {
		d := createDefaultMachineOptions(logic, traversalOptions)
		resolvedDefaultOptions = &d
	}

	config := TraversalOptions[S]{
		SerializeState: func(state S, _ xs.Event, _ S) string { return jsonStringify(state) },
		SerializeEvent: jsonStringifyEvent,
		Events:         []xs.Event{},
	}
	if traversalOptions != nil {
		// Traversal should not continue past the `toState` predicate
		// since the target state has already been reached at that point
		config.StopWhen = traversalOptions.ToState
	}
	if resolvedDefaultOptions != nil {
		config = config.overlay(*resolvedDefaultOptions)
	}
	if traversalOptions != nil {
		config = config.overlay(*traversalOptions)
	}
	return config
}

// overlay mirrors the object spread `{ ...o, ...other }`: every field set in
// other (non-zero) replaces the one in o. `events` is a single JS key, so
// setting either Events or EventsFn replaces both.
func (o TraversalOptions[S]) overlay(other TraversalOptions[S]) TraversalOptions[S] {
	if other.Input != nil {
		o.Input = other.Input
	}
	if other.SerializeState != nil {
		o.SerializeState = other.SerializeState
	}
	if other.SerializeEvent != nil {
		o.SerializeEvent = other.SerializeEvent
	}
	if other.EventsFn != nil {
		o.Events, o.EventsFn = nil, other.EventsFn
	} else if other.Events != nil {
		o.Events, o.EventsFn = other.Events, nil
	}
	if other.FilterEvents != nil {
		o.FilterEvents = other.FilterEvents
	}
	if other.Limit != 0 {
		o.Limit = other.Limit
	}
	if !isZero(other.FromState) {
		o.FromState = other.FromState
	}
	if other.StopWhen != nil {
		o.StopWhen = other.StopWhen
	}
	if other.ToState != nil {
		o.ToState = other.ToState
	}
	return o
}

// events mirrors `typeof getEvents === 'function' ? getEvents(state) : getEvents`.
func (o TraversalOptions[S]) events(state S) []xs.Event {
	if o.EventsFn != nil {
		return o.EventsFn(state)
	}
	return o.Events
}

// initialState mirrors `fromState ?? logic.getInitialSnapshot(scope, input)`.
func (o TraversalOptions[S]) initialState(logic xs.TypedActorLogic[S], input any) S {
	if !isZero(o.FromState) {
		return o.FromState
	}
	return xs.GetInitialSnapshot(logic, input)
}

func inputOf[S xs.Snapshot](options []TraversalOptions[S]) any {
	if len(options) > 0 {
		return options[0].Input
	}
	return nil
}

func firstOptions[S xs.Snapshot](options []TraversalOptions[S]) *TraversalOptions[S] {
	if len(options) > 0 {
		o := options[0]
		return &o
	}
	return nil
}

// JoinPaths mirrors joinPaths(headPath, tailPath). It panics with
// errors.New("Paths cannot be joined") when tailPath does not start at
// headPath.State.
func JoinPaths[S xs.Snapshot](headPath, tailPath StatePath[S]) StatePath[S] {
	secondPathSource := tailPath.Steps[0].State

	if !sameState(secondPathSource, headPath.State) {
		panic(errors.New("Paths cannot be joined"))
	}

	// e.g. [A, B, C] + [C, D, E] = [A, B, C, D, E]
	steps := append(slices.Clone(headPath.Steps), tailPath.Steps[1:]...)
	return StatePath[S]{
		State:  tailPath.State,
		Steps:  steps,
		Weight: headPath.Weight + tailPath.Weight,
	}
}
