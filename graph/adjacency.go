package graph

import (
	"errors"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// GetAdjacencyMap mirrors getAdjacencyMap(logic, options). It panics with
// errors.New("Traversal limit exceeded") when options.Limit is exceeded.
func GetAdjacencyMap[S xs.Snapshot](logic xs.TypedActorLogic[S], options TraversalOptions[S]) AdjacencyMap[S] {
	resolved := resolveTraversalOptions(logic, &options, nil)
	fromState := resolved.initialState(logic, options.Input)
	adj := AdjacencyMap[S]{Values: map[string]*AdjacencyValue[S]{}}

	type queued struct {
		nextState S
		event     xs.Event
		prevState S
	}
	var none S
	iterations := 0
	queue := []queued{{nextState: fromState, event: nil, prevState: none}}

	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]
		state := item.nextState

		if resolved.Limit > 0 && iterations > resolved.Limit {
			panic(errors.New("Traversal limit exceeded"))
		}
		iterations++

		serializedState := resolved.SerializeState(state, item.event, item.prevState)
		if _, ok := adj.Values[serializedState]; ok {
			continue
		}

		value := &AdjacencyValue[S]{State: state, Transitions: map[string]AdjacencyTransition[S]{}}
		adj.Keys = appendJSKey(adj.Keys, serializedState)
		adj.Values[serializedState] = value

		if resolved.StopWhen != nil && resolved.StopWhen(state) {
			continue
		}

		for _, nextEvent := range resolved.events(state) {
			if resolved.FilterEvents != nil && !resolved.FilterEvents(state, nextEvent) {
				continue
			}

			nextSnapshot := xs.GetNextSnapshot(logic, state, nextEvent)

			value.setTransition(resolved.SerializeEvent(nextEvent), AdjacencyTransition[S]{
				Event: nextEvent,
				State: nextSnapshot,
			})
			queue = append(queue, queued{nextState: nextSnapshot, event: nextEvent, prevState: state})
		}
	}

	return adj
}

// setTransition mirrors `transitions[key] = value` on a JS object: a new key
// takes its JS enumeration position, an existing key keeps its position.
func (v *AdjacencyValue[S]) setTransition(key string, t AdjacencyTransition[S]) {
	if _, ok := v.Transitions[key]; !ok {
		v.EventKeys = appendJSKey(v.EventKeys, key)
	}
	v.Transitions[key] = t
}

// AdjacencyMapToArray mirrors adjacencyMapToArray(adjMap).
func AdjacencyMapToArray[S xs.Snapshot](adjMap AdjacencyMap[S]) []AdjacencyEntry[S] {
	var adjList []AdjacencyEntry[S]
	for _, key := range adjMap.Keys {
		adjValue := adjMap.Values[key]
		for _, eventKey := range adjValue.EventKeys {
			transition := adjValue.Transitions[eventKey]
			adjList = append(adjList, AdjacencyEntry[S]{
				State:     adjValue.State,
				Event:     transition.Event,
				NextState: transition.State,
			})
		}
	}
	return adjList
}
