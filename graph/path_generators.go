package graph

import xs "github.com/nguyenvanduocit/go-xstate/xstate"

// CreateShortestPathsGen mirrors createShortestPathsGen().
func CreateShortestPathsGen[S xs.Snapshot]() PathGenerator[S] {
	return func(logic xs.TypedActorLogic[S], defaultOptions TraversalOptions[S]) []StatePath[S] {
		return GetShortestPaths(logic, defaultOptions)
	}
}

// CreateSimplePathsGen mirrors createSimplePathsGen().
func CreateSimplePathsGen[S xs.Snapshot]() PathGenerator[S] {
	return func(logic xs.TypedActorLogic[S], defaultOptions TraversalOptions[S]) []StatePath[S] {
		return GetSimplePaths(logic, defaultOptions)
	}
}
