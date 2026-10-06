package xstate

import (
	"errors"
	"fmt"
)

type macrostepResult struct {
	snapshot   anyMachineSnapshot
	microsteps []microstepResult
}

func macrostep(snap anyMachineSnapshot, event Event, scope *ActorScope, internalQueue *[]Event) macrostepResult {
	if event == nil {
		panic(errors.New("Cannot read properties of undefined (reading 'type')"))
	}
	if event.EventType() == wildcard {
		panic(fmt.Errorf("An event cannot have the wildcard type ('%s')", wildcard))
	}
	next := snap
	var microsteps []microstepResult
	addMicrostep := func(step microstepResult, ev Event, transitions []*TransitionDefinition) {
		scope.System.sendInspectionEvent(InspectionEvent{
			Type:        InspectMicrostep,
			ActorRef:    scope.Self,
			Event:       ev,
			Snapshot:    step.snapshot,
			Transitions: transitions,
		})
		microsteps = append(microsteps, step)
	}

	if event.EventType() == xstateStop {
		next = stopChildren(next, event, scope).clone(snapshotPatch{status: statusPtr(StatusStopped)})
		addMicrostep(microstepResult{next, nil}, event, nil)
		return macrostepResult{next, microsteps}
	}

	nextEvent := event
	if nextEvent.EventType() != xstateInit {
		current := nextEvent
		isErr := isErrorActorEvent(current)
		transitions := getTransitionData(next, current)
		if isErr && len(transitions) == 0 {
			next = snap.clone(snapshotPatch{status: statusPtr(StatusError), err: anyPtr(eventErrorValue(current))})
			addMicrostep(microstepResult{next, nil}, current, nil)
			return macrostepResult{next, microsteps}
		}
		step := microstep(transitions, snap, scope, nextEvent, false, internalQueue)
		next = step.snapshot
		addMicrostep(step, current, transitions)
	}

	shouldSelectEventless := true
	maxIterations := snap.machineAny().machineOptions().MaxIterations
	iteration := 0
	for next.GetStatus() == StatusActive {
		iteration++
		if maxIterations > 0 && iteration > maxIterations {
			panic(fmt.Errorf("Infinite loop detected: the machine has processed more than %d microsteps without reaching a stable state. This usually happens when there's a cycle of transitions (e.g., eventless transitions or raised events causing state A -> B -> C -> A).", maxIterations))
		}
		var enabled []*TransitionDefinition
		if shouldSelectEventless {
			enabled = selectEventlessTransitions(next, nextEvent)
		}
		var previous anyMachineSnapshot
		if len(enabled) > 0 {
			previous = next
		}
		if len(enabled) == 0 {
			if len(*internalQueue) == 0 {
				break
			}
			nextEvent = (*internalQueue)[0]
			*internalQueue = (*internalQueue)[1:]
			enabled = getTransitionData(next, nextEvent)
		}
		step := microstep(enabled, next, scope, nextEvent, false, internalQueue)
		next = step.snapshot
		shouldSelectEventless = next != previous
		addMicrostep(step, nextEvent, enabled)
	}
	if next.GetStatus() != StatusActive {
		stopChildren(next, nextEvent, scope)
	}
	return macrostepResult{next, microsteps}
}

func stopChildren(snap anyMachineSnapshot, event Event, scope *ActorScope) anyMachineSnapshot {
	cs := snap.children()
	var actions Actions
	for _, k := range cs.order {
		actions = append(actions, StopChild(cs.m[k]))
	}
	return resolveActionsAndContext(snap, event, scope, actions, &[]Event{}, nil, false)
}

func selectEventlessTransitions(snap anyMachineSnapshot, event Event) []*TransitionDefinition {
	enabled := []*TransitionDefinition{}
	seen := map[*TransitionDefinition]bool{}
	for _, n := range snap.nodesAny() {
		if !isAtomicStateNode(n) {
			continue
		}
	loop:
		for _, s := range append([]*StateNode{n}, getProperAncestors(n, nil)...) {
			if s.always == nil {
				continue
			}
			for _, t := range s.always {
				if t.Guard == nil || evaluateGuard(t.Guard, snap.contextAny(), event, snap) {
					if !seen[t] {
						seen[t] = true
						enabled = append(enabled, t)
					}
					break loop
				}
			}
		}
	}
	return removeConflictingTransitions(enabled, newNodeSet(snap.nodesAny()...), snap.historyAny())
}
