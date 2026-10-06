package graph

import (
	"slices"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// GetSimplePaths mirrors getSimplePaths(logic, options).
func GetSimplePaths[S xs.Snapshot](logic xs.TypedActorLogic[S], options ...TraversalOptions[S]) []StatePath[S] {
	resolvedOptions := resolveTraversalOptions(logic, firstOptions(options), nil)
	fromState := resolvedOptions.initialState(logic, inputOf(options))
	serializeState := resolvedOptions.SerializeState
	adjacency := GetAdjacencyMap(logic, resolvedOptions)
	stateMap := map[string]S{}
	vertices := map[string]bool{}
	var steps []Step[S]
	// pathMap is a JS object keyed by serialized state: keys keep JS object
	// enumeration order.
	var pathMapKeys []string
	pathMap := map[string][]StatePath[S]{}

	var util func(fromStateSerial, toStateSerial string)
	util = func(fromStateSerial, toStateSerial string) {
		fromState := stateMap[fromStateSerial]
		vertices[fromStateSerial] = true

		if fromStateSerial == toStateSerial {
			if _, ok := pathMap[toStateSerial]; !ok {
				pathMapKeys = appendJSKey(pathMapKeys, toStateSerial)
				pathMap[toStateSerial] = nil
			}

			pathMap[toStateSerial] = append(pathMap[toStateSerial], StatePath[S]{
				State:  fromState,
				Weight: len(steps),
				Steps:  slices.Clone(steps),
			})
		} else {
			adjValue := adjacency.Values[fromStateSerial]
			for _, serializedEvent := range adjValue.EventKeys {
				transition := adjValue.Transitions[serializedEvent]
				nextState, subEvent := transition.State, transition.Event

				prevState := stateMap[fromStateSerial]

				nextStateSerial := serializeState(nextState, subEvent, prevState)
				stateMap[nextStateSerial] = nextState

				if !vertices[nextStateSerial] {
					steps = append(steps, Step[S]{
						State: stateMap[fromStateSerial],
						Event: subEvent,
					})
					util(nextStateSerial, toStateSerial)
				}
			}
		}

		if len(steps) > 0 {
			steps = steps[:len(steps)-1]
		}
		delete(vertices, fromStateSerial)
	}

	var none S
	fromStateSerial := serializeState(fromState, nil, none)
	stateMap[fromStateSerial] = fromState

	for _, nextStateSerial := range adjacency.Keys {
		util(fromStateSerial, nextStateSerial)
	}

	var simplePaths []StatePath[S]
	for _, k := range pathMapKeys {
		simplePaths = append(simplePaths, pathMap[k]...)
	}

	return filterToStateAndAlter(simplePaths, resolvedOptions.ToState)
}
