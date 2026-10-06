package graph

import xs "github.com/nguyenvanduocit/go-xstate/xstate"

// initEvent mirrors the `{ type: 'xstate.init' }` event alterPath inserts.
func initEvent() xs.Event { return xs.Ev("xstate.init") }

// alterPath mirrors alterPath(path): every step carries the event that led
// to its state (the first one `xstate.init`), and the ending state becomes
// the last step.
func alterPath[S xs.Snapshot](path StatePath[S]) StatePath[S] {
	var steps []Step[S]

	if len(path.Steps) == 0 {
		steps = []Step[S]{{State: path.State, Event: initEvent()}}
	} else {
		for i, step := range path.Steps {
			event := initEvent()
			if i > 0 {
				event = path.Steps[i-1].Event
			}
			steps = append(steps, Step[S]{State: step.State, Event: event})
		}
		steps = append(steps, Step[S]{
			State: path.State,
			Event: path.Steps[len(path.Steps)-1].Event,
		})
	}
	path.Steps = steps
	return path
}
