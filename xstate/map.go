package xstate

// MapState mirrors the v4 mapState(stateMap, stateId).
func MapState[T any](stateMap map[string]T, stateID string) (T, bool) {
	found := ""
	ok := false
	for _, id := range sortedKeys(stateMap) {
		if matchesState(id, stateID) && (!ok || len(stateID) > len(found)) {
			found, ok = id, true
		}
	}
	if !ok {
		var z T
		return z, false
	}
	return stateMap[found], true
}

// StateMapper mirrors the v5 `StateSchemaMapper` object passed to
// mapState(snapshot, mapper).
type StateMapper[C any, R any] struct {
	Map    func(s *MachineSnapshot[C]) R
	States map[string]StateMapper[C, R]
}

// StateMapResult mirrors one `{ stateNode, result }` entry.
type StateMapResult[R any] struct {
	StateNode *StateNode
	Result    R
}

// MapSnapshot mirrors the v5 mapState(snapshot, mapper) from mapState.ts.
func MapSnapshot[C any, R any](s *MachineSnapshot[C], mapper StateMapper[C, R]) []StateMapResult[R] {
	var results []StateMapResult[R]
	findMapper := func(path []string) (StateMapper[C, R], bool) {
		m := mapper
		for _, key := range path {
			if m.States == nil {
				return StateMapper[C, R]{}, false
			}
			next, ok := m.States[key]
			if !ok {
				return StateMapper[C, R]{}, false
			}
			m = next
		}
		return m, true
	}
	visited := map[*StateNode]bool{}
	for _, atomic := range s.nodes {
		if !isAtomicStateNode(atomic) {
			continue
		}
		for cur := atomic; cur != nil && !visited[cur]; cur = cur.Parent {
			visited[cur] = true
			if m, ok := findMapper(cur.Path); ok && m.Map != nil {
				results = append(results, StateMapResult[R]{StateNode: cur, Result: m.Map(s)})
			}
		}
	}
	return results
}
