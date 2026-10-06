package store

import (
	"slices"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// UndoRedoStrategy mirrors the undoRedo `strategy` option.
type UndoRedoStrategy string

const (
	UndoRedoEvent    UndoRedoStrategy = "event" // default ("" behaves as "event")
	UndoRedoSnapshot UndoRedoStrategy = "snapshot"
)

// UndoRedoDirection mirrors the restore `direction` argument.
type UndoRedoDirection string

const (
	DirectionUndo UndoRedoDirection = "undo"
	DirectionRedo UndoRedoDirection = "redo"
)

// UndoRedoRestoreArgs mirrors the first argument of the `restore` option.
type UndoRedoRestoreArgs[C any] struct {
	Current   C
	Next      C
	Direction UndoRedoDirection
}

// UndoRedoOptions mirrors UndoRedoStrategyOptions (both strategies in one
// struct; fields marked "snapshot only" are ignored by the event strategy).
type UndoRedoOptions[C any] struct {
	Strategy UndoRedoStrategy
	// GetTransactionID mirrors getTransactionId; "" mirrors null/undefined
	// (no transaction).
	GetTransactionID func(event xs.Event, snapshot *StoreSnapshot[C]) string
	SkipEvent        func(event xs.Event, snapshot *StoreSnapshot[C]) bool
	// HistoryLimit mirrors historyLimit (snapshot only); 0 means Infinity.
	HistoryLimit int
	// Compare mirrors compare (snapshot only).
	Compare func(past, current *StoreSnapshot[C]) bool
	// Restore mirrors restore (snapshot only).
	Restore func(args UndoRedoRestoreArgs[C], enq *EnqueueObject[C]) C
}

// Extension-state keys of undoRedo (the JS snapshot properties).
const (
	undoKeyEvents    = "events"
	undoKeyUndoStack = "undoStack"
	undoKeyPast      = "past"
	undoKeyFuture    = "future"
)

// undoEventItem mirrors `{ event, transactionId }` of the event strategy;
// transactionID "" mirrors undefined.
type undoEventItem struct {
	event         xs.Event
	transactionID string
}

// undoSnapshotItem mirrors `{ snapshot, transactionId }` of the snapshot
// strategy; the snapshot holds only status, context, output and error.
type undoSnapshotItem[C any] struct {
	snapshot      *StoreSnapshot[C]
	transactionID string
}

// UndoRedo mirrors undoRedo(options?) from @xstate/store/undo: adds the
// "undo" and "redo" event types.
func UndoRedo[C any](opts ...UndoRedoOptions[C]) StoreExtension[C] {
	var options UndoRedoOptions[C]
	if len(opts) > 0 {
		options = opts[0]
	}
	return func(logic StoreLogic[C]) StoreLogic[C] {
		if options.Strategy == UndoRedoSnapshot {
			return undoRedoSnapshot(logic, options)
		}
		return undoRedoEvents(logic, options)
	}
}

func (o UndoRedoOptions[C]) transactionID(event xs.Event, snapshot *StoreSnapshot[C]) string {
	if o.GetTransactionID == nil {
		return ""
	}
	return o.GetTransactionID(event, snapshot)
}

func undoEvents[C any](s *StoreSnapshot[C], key string) []undoEventItem {
	v, _ := s.ext(key).([]undoEventItem)
	return v
}

func undoSnapshots[C any](s *StoreSnapshot[C], key string) []undoSnapshotItem[C] {
	v, _ := s.ext(key).([]undoSnapshotItem[C])
	return v
}

// baseSnapshot mirrors `{ status, context, output, error }`.
func baseSnapshot[C any](s *StoreSnapshot[C]) *StoreSnapshot[C] {
	return &StoreSnapshot[C]{Status: s.Status, Context: s.Context, Output: s.Output, Error: s.Error}
}

func undoRedoEvents[C any](logic StoreLogic[C], options UndoRedoOptions[C]) StoreLogic[C] {
	enhanced := logic
	enhanced.EventTypes = appendInternalEventTypes(logic.EventTypes, []string{"undo", "redo"}, "undoRedo")
	enhanced.GetInitialSnapshot = func() *StoreSnapshot[C] {
		return logic.GetInitialSnapshot().withExt(undoKeyEvents, []undoEventItem{}, undoKeyUndoStack, []undoEventItem{})
	}
	enhanced.Transition = func(snapshot *StoreSnapshot[C], event xs.Event) StoreTransitionResult[C] {
		switch event.EventType() {
		case "undo":
			events := slices.Clone(undoEvents(snapshot, undoKeyEvents))
			undoStack := slices.Clone(undoEvents(snapshot, undoKeyUndoStack))
			if len(events) == 0 {
				return StoreTransitionResult[C]{Snapshot: snapshot, Effects: []StoreEffect[C]{}}
			}
			lastTransactionID := events[len(events)-1].transactionID
			if lastTransactionID == "" {
				undoStack = append(undoStack, events[len(events)-1])
				events = events[:len(events)-1]
			} else {
				for {
					ev := events[len(events)-1]
					events = events[:len(events)-1]
					undoStack = append(undoStack, ev)
					if len(events) == 0 || events[len(events)-1].transactionID != lastTransactionID {
						break
					}
				}
			}
			state := logic.GetInitialSnapshot().withExt(undoKeyEvents, events, undoKeyUndoStack, undoStack)
			for _, item := range events {
				next := logic.Transition(state, item.event).Snapshot
				state = next.withExt(undoKeyEvents, events, undoKeyUndoStack, undoStack)
			}
			return StoreTransitionResult[C]{Snapshot: state, Effects: []StoreEffect[C]{}}

		case "redo":
			events := slices.Clone(undoEvents(snapshot, undoKeyEvents))
			undoStack := slices.Clone(undoEvents(snapshot, undoKeyUndoStack))
			if len(undoStack) == 0 {
				return StoreTransitionResult[C]{
					Snapshot: snapshot.withExt(undoKeyEvents, events, undoKeyUndoStack, undoStack),
					Effects:  []StoreEffect[C]{},
				}
			}
			lastTransactionID := undoStack[len(undoStack)-1].transactionID
			state := snapshot.withExt(undoKeyEvents, events, undoKeyUndoStack, undoStack)
			allEffects := []StoreEffect[C]{}
			replay := func() {
				item := undoStack[len(undoStack)-1]
				undoStack = undoStack[:len(undoStack)-1]
				events = append(events, item)
				result := logic.Transition(state, item.event)
				state = result.Snapshot.withExt(undoKeyEvents, events, undoKeyUndoStack, undoStack)
				allEffects = append(allEffects, result.Effects...)
			}
			if lastTransactionID == "" {
				replay()
			} else {
				for len(undoStack) > 0 && undoStack[len(undoStack)-1].transactionID == lastTransactionID {
					replay()
				}
			}
			return StoreTransitionResult[C]{Snapshot: state, Effects: allEffects}
		}

		result := logic.Transition(snapshot, event)
		skipped := options.SkipEvent != nil && options.SkipEvent(event, snapshot)
		events := undoEvents(snapshot, undoKeyEvents)
		if !skipped {
			events = append(slices.Clone(events), undoEventItem{
				event:         event,
				transactionID: options.transactionID(event, snapshot),
			})
		}
		return StoreTransitionResult[C]{
			Snapshot: result.Snapshot.withExt(undoKeyEvents, events, undoKeyUndoStack, []undoEventItem{}),
			Effects:  result.Effects,
		}
	}
	return enhanced
}

func undoRedoSnapshot[C any](logic StoreLogic[C], options UndoRedoOptions[C]) StoreLogic[C] {
	historyLimit := options.HistoryLimit

	trimPast := func(past []undoSnapshotItem[C]) []undoSnapshotItem[C] {
		if historyLimit > 0 {
			if excess := len(past) - historyLimit; excess > 0 {
				past = past[excess:]
			}
		}
		return past
	}

	// restore mirrors the inner restore(snapshot, historical, direction,
	// past, future) of undo.ts.
	restore := func(
		snapshot, historical *StoreSnapshot[C],
		direction UndoRedoDirection,
		past, future []undoSnapshotItem[C],
	) StoreTransitionResult[C] {
		merged := snapshot.spread()
		merged.Status, merged.Context, merged.Output, merged.Error =
			historical.Status, historical.Context, historical.Output, historical.Error
		if options.Restore == nil {
			return StoreTransitionResult[C]{
				Snapshot: merged.withExt(undoKeyPast, past, undoKeyFuture, future),
				Effects:  []StoreEffect[C]{},
			}
		}

		effects := []StoreEffect[C]{}
		var triggered []xs.Event
		enq := newEnqueueObject(&effects, func(ev xs.Event) { triggered = append(triggered, ev) })
		merged.Context = options.Restore(UndoRedoRestoreArgs[C]{
			Current:   snapshot.Context,
			Next:      historical.Context,
			Direction: direction,
		}, enq)
		current := merged
		for _, ev := range triggered {
			result := logic.Transition(current, ev)
			current = result.Snapshot
			effects = append(effects, result.Effects...)
		}
		return StoreTransitionResult[C]{
			Snapshot: current.withExt(undoKeyPast, past, undoKeyFuture, future),
			Effects:  effects,
		}
	}

	enhanced := logic
	enhanced.EventTypes = appendInternalEventTypes(logic.EventTypes, []string{"undo", "redo"}, "undoRedo")
	enhanced.GetInitialSnapshot = func() *StoreSnapshot[C] {
		return logic.GetInitialSnapshot().withExt(undoKeyPast, []undoSnapshotItem[C]{}, undoKeyFuture, []undoSnapshotItem[C]{})
	}
	enhanced.Transition = func(snapshot *StoreSnapshot[C], event xs.Event) StoreTransitionResult[C] {
		switch event.EventType() {
		case "undo":
			past := slices.Clone(undoSnapshots[C](snapshot, undoKeyPast))
			future := slices.Clone(undoSnapshots[C](snapshot, undoKeyFuture))
			if len(past) == 0 {
				return StoreTransitionResult[C]{Snapshot: snapshot, Effects: []StoreEffect[C]{}}
			}
			current := baseSnapshot(snapshot)
			lastTransactionID := past[len(past)-1].transactionID
			var historical *StoreSnapshot[C]
			if lastTransactionID == "" {
				historical = past[len(past)-1].snapshot
				past = past[:len(past)-1]
			} else {
				var transaction []undoSnapshotItem[C]
				for len(past) > 0 && past[len(past)-1].transactionID == lastTransactionID {
					transaction = append([]undoSnapshotItem[C]{past[len(past)-1]}, transaction...)
					past = past[:len(past)-1]
				}
				historical = transaction[0].snapshot
			}
			future = append([]undoSnapshotItem[C]{{snapshot: current, transactionID: lastTransactionID}}, future...)
			return restore(snapshot, historical, DirectionUndo, past, future)

		case "redo":
			past := slices.Clone(undoSnapshots[C](snapshot, undoKeyPast))
			future := slices.Clone(undoSnapshots[C](snapshot, undoKeyFuture))
			if len(future) == 0 {
				return StoreTransitionResult[C]{Snapshot: snapshot, Effects: []StoreEffect[C]{}}
			}
			firstTransactionID := future[0].transactionID
			current := baseSnapshot(snapshot)
			var historical *StoreSnapshot[C]
			if firstTransactionID == "" {
				historical = future[0].snapshot
				future = future[1:]
			} else {
				for len(future) > 0 && future[0].transactionID == firstTransactionID {
					historical = future[0].snapshot
					future = future[1:]
				}
			}
			// Both stacks hold the snapshot that preceded the change, so a redo
			// pushes the snapshot it is moving away from, not the one it restores.
			past = trimPast(append(past, undoSnapshotItem[C]{snapshot: current, transactionID: firstTransactionID}))
			return restore(snapshot, historical, DirectionRedo, past, future)
		}

		result := logic.Transition(snapshot, event)
		prevPast := undoSnapshots[C](snapshot, undoKeyPast)
		prevFuture := undoSnapshots[C](snapshot, undoKeyFuture)
		if options.SkipEvent != nil && options.SkipEvent(event, snapshot) {
			return StoreTransitionResult[C]{
				Snapshot: result.Snapshot.withExt(undoKeyPast, prevPast, undoKeyFuture, prevFuture),
				Effects:  result.Effects,
			}
		}

		current := baseSnapshot(snapshot)
		if len(prevPast) > 0 && options.Compare != nil && options.Compare(prevPast[len(prevPast)-1].snapshot, current) {
			return StoreTransitionResult[C]{
				Snapshot: result.Snapshot.withExt(undoKeyPast, prevPast, undoKeyFuture, []undoSnapshotItem[C]{}),
				Effects:  result.Effects,
			}
		}

		past := trimPast(append(slices.Clone(prevPast), undoSnapshotItem[C]{
			snapshot:      current,
			transactionID: options.transactionID(event, snapshot),
		}))
		return StoreTransitionResult[C]{
			Snapshot: result.Snapshot.withExt(undoKeyPast, past, undoKeyFuture, []undoSnapshotItem[C]{}),
			Effects:  result.Effects,
		}
	}
	return enhanced
}
