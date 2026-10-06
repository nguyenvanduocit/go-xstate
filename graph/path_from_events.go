package graph

import (
	"fmt"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// GetPathsFromEvents mirrors getPathsFromEvents(logic, events, options). It
// panics with "Invalid transition from <state> with <event>" when an event
// has no entry in the adjacency map.
func GetPathsFromEvents[S xs.Snapshot](logic xs.TypedActorLogic[S], events []xs.Event, options ...TraversalOptions[S]) []StatePath[S] {
	// { events, ...options }: events is always an array here (non-nil), so it
	// replaces the default own-event resolver even when empty.
	traversalOptions := TraversalOptions[S]{Events: append([]xs.Event{}, events...)}
	if len(options) > 0 {
		traversalOptions = traversalOptions.overlay(options[0])
	}
	var defaultOptions TraversalOptions[S]
	if isMachineLogic(logic) {
		defaultOptions = createDefaultMachineOptions(logic, nil)
	} else {
		defaultOptions = createDefaultLogicOptions[S]()
	}
	resolvedOptions := resolveTraversalOptions(logic, &traversalOptions, &defaultOptions)
	fromState := resolvedOptions.initialState(logic, inputOf(options))

	serializeState, serializeEvent := resolvedOptions.SerializeState, resolvedOptions.SerializeEvent

	adjacency := GetAdjacencyMap(logic, resolvedOptions)

	stateMap := map[string]S{}
	steps := []Step[S]{}

	var none S
	serializedFromState := serializeState(fromState, nil, none)
	stateMap[serializedFromState] = fromState

	stateSerial := serializedFromState
	state := fromState
	for _, event := range events {
		steps = append(steps, Step[S]{
			State: stateMap[stateSerial],
			Event: event,
		})

		eventSerial := serializeEvent(event)
		var transition AdjacencyTransition[S]
		found := false
		if adjValue, ok := adjacency.Values[stateSerial]; ok {
			transition, found = adjValue.Transitions[eventSerial]
		}
		if !found || isZero(transition.State) {
			panic(fmt.Errorf("Invalid transition from %s with %s", stateSerial, eventSerial))
		}
		nextState := transition.State
		prevState := stateMap[stateSerial]
		nextStateSerial := serializeState(nextState, event, prevState)
		stateMap[nextStateSerial] = nextState

		stateSerial = nextStateSerial
		state = nextState
	}

	// If it is expected to reach a specific state (`toState`) and that state
	// isn't reached, there are no paths
	if resolvedOptions.ToState != nil && !resolvedOptions.ToState(state) {
		return []StatePath[S]{}
	}

	return []StatePath[S]{
		alterPath(StatePath[S]{
			State:  state,
			Steps:  steps,
			Weight: len(steps),
		}),
	}
}
