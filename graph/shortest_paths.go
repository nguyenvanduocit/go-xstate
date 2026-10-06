package graph

import (
	"slices"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// GetShortestPaths mirrors getShortestPaths(logic, options).
func GetShortestPaths[S xs.Snapshot](logic xs.TypedActorLogic[S], options ...TraversalOptions[S]) []StatePath[S] {
	resolvedOptions := resolveTraversalOptions(logic, firstOptions(options), nil)
	serializeState := resolvedOptions.SerializeState
	fromState := resolvedOptions.initialState(logic, inputOf(options))
	adjacency := GetAdjacencyMap(logic, resolvedOptions)

	type weightEntry struct {
		weight int
		state  string // meaningful only when has; !has mirrors `undefined`
		has    bool
		event  xs.Event
	}
	// weightMap is a JS Map: keys keep insertion order.
	weightKeys := []string{}
	weightMap := map[string]weightEntry{}
	setWeight := func(k string, w weightEntry) {
		if _, ok := weightMap[k]; !ok {
			weightKeys = append(weightKeys, k)
		}
		weightMap[k] = w
	}
	stateMap := map[string]S{}
	var none S
	serializedFromState := serializeState(fromState, nil, none)
	stateMap[serializedFromState] = fromState

	setWeight(serializedFromState, weightEntry{weight: 0})

	// `unvisited` is a JS Set iterated while it grows; a state is added only
	// when it is neither visited nor queued, so this is a FIFO queue.
	unvisited := []string{serializedFromState}
	queued := map[string]bool{serializedFromState: true}
	visited := map[string]bool{}

	for i := 0; i < len(unvisited); i++ {
		serializedState := unvisited[i]
		prevState := stateMap[serializedState]
		weight := weightMap[serializedState].weight
		adjValue := adjacency.Values[serializedState]
		for _, event := range adjValue.EventKeys {
			transition := adjValue.Transitions[event]
			nextState, eventObject := transition.State, transition.Event
			nextSerializedState := serializeState(nextState, eventObject, prevState)
			stateMap[nextSerializedState] = nextState
			if next, ok := weightMap[nextSerializedState]; !ok {
				setWeight(nextSerializedState, weightEntry{weight: weight + 1, state: serializedState, has: true, event: eventObject})
			} else if next.weight > weight+1 {
				setWeight(nextSerializedState, weightEntry{weight: weight + 1, state: serializedState, has: true, event: eventObject})
			}
			if !visited[nextSerializedState] && !queued[nextSerializedState] {
				queued[nextSerializedState] = true
				unvisited = append(unvisited, nextSerializedState)
			}
		}
		visited[serializedState] = true
		delete(queued, serializedState)
	}

	// statePlanMap[serial].paths[0].steps
	statePlanSteps := map[string][]Step[S]{}
	paths := make([]StatePath[S], 0, len(weightKeys))

	for _, stateSerial := range weightKeys {
		w := weightMap[stateSerial]
		state := stateMap[stateSerial]
		var steps []Step[S]
		if w.has {
			steps = append(slices.Clone(statePlanSteps[w.state]), Step[S]{
				State: stateMap[w.state],
				Event: w.event,
			})
		} else {
			steps = []Step[S]{}
		}

		paths = append(paths, StatePath[S]{State: state, Steps: steps, Weight: w.weight})
		statePlanSteps[stateSerial] = steps
	}

	return filterToStateAndAlter(paths, resolvedOptions.ToState)
}

// filterToStateAndAlter mirrors the shared tail of the path generators:
// keep paths reaching toState (when set), then alterPath each.
func filterToStateAndAlter[S xs.Snapshot](paths []StatePath[S], toState func(S) bool) []StatePath[S] {
	out := make([]StatePath[S], 0, len(paths))
	for _, path := range paths {
		if toState != nil && !toState(path.State) {
			continue
		}
		out = append(out, alterPath(path))
	}
	return out
}
