package store

import xs "github.com/nguyenvanduocit/go-xstate/xstate"

// ResetOptions mirrors ResetOptions<TContext>.
type ResetOptions[C any] struct {
	// To mirrors `to(initialContext, currentContext)`; nil resets to the
	// initial context.
	To func(initial, current C) C
}

// Reset mirrors reset(options?) from @xstate/store/reset: adds the "reset"
// event type.
func Reset[C any](opts ...ResetOptions[C]) StoreExtension[C] {
	var options ResetOptions[C]
	if len(opts) > 0 {
		options = opts[0]
	}
	return func(logic StoreLogic[C]) StoreLogic[C] {
		enhanced := logic
		enhanced.EventTypes = appendInternalEventTypes(logic.EventTypes, []string{"reset"}, "reset")
		enhanced.GetInitialSnapshot = func() *StoreSnapshot[C] { return logic.GetInitialSnapshot() }
		enhanced.Transition = func(snapshot *StoreSnapshot[C], event xs.Event) StoreTransitionResult[C] {
			if event.EventType() == "reset" {
				initial := logic.GetInitialSnapshot()
				resetContext := initial.Context
				if options.To != nil {
					resetContext = options.To(initial.Context, snapshot.Context)
				}
				return StoreTransitionResult[C]{Snapshot: snapshot.withContext(resetContext), Effects: []StoreEffect[C]{}}
			}
			return logic.Transition(snapshot, event)
		}
		return enhanced
	}
}
