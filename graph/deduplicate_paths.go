package graph

import (
	"sort"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// deduplicatePaths mirrors deduplicatePaths(paths, serializeEvent): drops
// paths whose event sequence is a prefix of a longer path's, so A -> B is
// not executed separately to A -> B -> C.
func deduplicatePaths[S xs.Snapshot](paths []StatePath[S], serializeEvent func(xs.Event) string) []StatePath[S] {
	if serializeEvent == nil {
		serializeEvent = jsonStringifyEvent
	}
	type pathWithEventSequence struct {
		path          StatePath[S]
		eventSequence []string
	}

	// Put all paths on the same level so we can dedup them
	all := make([]pathWithEventSequence, 0, len(paths))
	for _, path := range paths {
		seq := make([]string, 0, len(path.Steps))
		for _, step := range path.Steps {
			seq = append(seq, serializeEvent(step.Event))
		}
		all = append(all, pathWithEventSequence{path: path, eventSequence: seq})
	}

	// Sort by path length, descending (Array.prototype.sort is stable)
	sort.SliceStable(all, func(i, j int) bool {
		return len(all[i].path.Steps) > len(all[j].path.Steps)
	})

	var superpaths []pathWithEventSequence

	// Filter out the paths that are subpaths of superpaths
pathLoop:
	for _, p := range all {
		// Check each existing superpath to see if the path is a subpath of it
	superpathLoop:
		for _, superpath := range superpaths {
			for i := range p.eventSequence {
				// Superpaths are never shorter (sorted by length), so
				// superpath.eventSequence[i] exists.
				if p.eventSequence[i] != superpath.eventSequence[i] {
					// If the path is different from the superpath,
					// continue to the next superpath
					continue superpathLoop
				}
			}

			// If we reached here, path is subpath of superpath
			// Continue & do not add path to superpaths
			continue pathLoop
		}

		// If we reached here, path is not a subpath of any existing superpaths
		// So add it to the superpaths
		superpaths = append(superpaths, p)
	}

	out := make([]StatePath[S], 0, len(superpaths))
	for _, p := range superpaths {
		out = append(out, p.path)
	}
	return out
}
