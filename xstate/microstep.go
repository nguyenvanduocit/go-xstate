package xstate

import (
	"reflect"
	"sort"
)

type microstepResult struct {
	snapshot anyMachineSnapshot
	actions  []ExecutableAction
}

func getInitialStateNodesWithTheirAncestors(n *StateNode) *nodeSet {
	states := getInitialStateNodes(n)
	for _, initial := range states.values() {
		for _, anc := range getProperAncestors(initial, n) {
			states.add(anc)
		}
	}
	return states
}

func getInitialStateNodes(n *StateNode) *nodeSet {
	set := newNodeSet()
	var iter func(d *StateNode)
	iter = func(d *StateNode) {
		if set.has(d) {
			return
		}
		set.add(d)
		if d.Type == Compound {
			iter(d.Initial().Target[0])
		} else if d.Type == Parallel {
			for _, c := range getChildren(d) {
				iter(c)
			}
		}
	}
	iter(n)
	return set
}

func initialMicrostep(root *StateNode, preInitial anyMachineSnapshot, scope *ActorScope, initEvent Event, internalQueue *[]Event) microstepResult {
	return microstep([]*TransitionDefinition{{
		Target:  getInitialStateNodes(root).values(),
		Source:  root,
		Reenter: true,
	}}, preInitial, scope, initEvent, true, internalQueue)
}

// microstep mirrors https://www.w3.org/TR/scxml/#microstepProcedure.
func microstep(transitions []*TransitionDefinition, current anyMachineSnapshot, scope *ActorScope, event Event, isInitial bool, internalQueue *[]Event) microstepResult {
	var actions []ExecutableAction
	if len(transitions) == 0 {
		return microstepResult{current, actions}
	}
	original := scope.actionExecutor
	scope.actionExecutor = func(a ExecutableAction) {
		actions = append(actions, a)
		original(a)
	}
	defer func() { scope.actionExecutor = original }()

	mutSet := newNodeSet(current.nodesAny()...)
	history := current.historyAny()
	filtered := removeConflictingTransitions(transitions, mutSet, history)
	next := current
	if !isInitial {
		next, history = exitStates(next, event, scope, filtered, mutSet, history, internalQueue)
	}
	var transitionActions Actions
	for _, t := range filtered {
		transitionActions = append(transitionActions, t.Actions...)
	}
	next = resolveActionsAndContext(next, event, scope, transitionActions, internalQueue, nil, false)
	next = enterStates(next, event, scope, filtered, mutSet, internalQueue, history, isInitial)

	nextNodes := mutSet.values()
	if next.GetStatus() == StatusDone {
		sorted := append([]*StateNode{}, nextNodes...)
		sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Order > sorted[j].Order })
		var exitActions Actions
		for _, n := range sorted {
			exitActions = append(exitActions, n.Exit...)
		}
		next = resolveActionsAndContext(next, event, scope, exitActions, internalQueue, nil, false)
	}
	if sameHistory(history, current.historyAny()) && areStateNodeCollectionsEqual(current.nodesAny(), mutSet) {
		return microstepResult{next, actions}
	}
	return microstepResult{next.clone(snapshotPatch{nodes: nextNodes, setNodes: true, historyValue: history, setHistory: true}), actions}
}

// sameHistory mirrors the JS identity check `historyValue === currentSnapshot.historyValue`.
func sameHistory(a, b map[string][]*StateNode) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return reflect.ValueOf(a).UnsafePointer() == reflect.ValueOf(b).UnsafePointer()
}
