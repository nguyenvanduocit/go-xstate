package store_test

import (
	"testing"

	xstore "github.com/nguyenvanduocit/go-xstate/store"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/assert"
)

// storeUndo1Ctx is the `{ count: 0 }` context shared by most undo tests.
type storeUndo1Ctx struct{ Count int }

type storeUndo1Handlers = map[string]xstore.StoreAssigner[storeUndo1Ctx]

// JS: inc: (ctx) => ({ count: ctx.count + 1 })
func storeUndo1Inc(c storeUndo1Ctx, _ xs.Event, _ *xstore.EnqueueObject[storeUndo1Ctx]) (storeUndo1Ctx, bool) {
	return storeUndo1Ctx{Count: c.Count + 1}, true
}

// JS: dec: (ctx) => ({ count: ctx.count - 1 })
func storeUndo1Dec(c storeUndo1Ctx, _ xs.Event, _ *xstore.EnqueueObject[storeUndo1Ctx]) (storeUndo1Ctx, bool) {
	return storeUndo1Ctx{Count: c.Count - 1}, true
}

// JS: log / noop: (ctx) => ctx
func storeUndo1Identity(c storeUndo1Ctx, _ xs.Event, _ *xstore.EnqueueObject[storeUndo1Ctx]) (storeUndo1Ctx, bool) {
	return c, true
}

func storeUndo1New(on storeUndo1Handlers) *xstore.Store[storeUndo1Ctx] {
	return xstore.CreateStore(xstore.StoreConfig[storeUndo1Ctx]{
		Context: storeUndo1Ctx{Count: 0},
		On:      on,
	})
}

func storeUndo1Count(s *xstore.Store[storeUndo1Ctx]) int { return s.GetSnapshot().Context.Count }

// storeUndo1WithExt returns a copy of m with key set (ExtensionState is copy-on-write).
func storeUndo1WithExt(m map[string]any, key string, value any) map[string]any {
	out := make(map[string]any, len(m)+1)
	for k, v := range m {
		out[k] = v
	}
	out[key] = value
	return out
}

// JS: preserves persistence metadata through snapshot undo and redo (%s)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/undo.test.ts#L6
func TestStoreUndo_PreservesPersistenceMetadataThroughSnapshotUndoAndRedo(t *testing.T) {
	type ctx struct {
		Count int `json:"count"`
	}
	for _, order := range []string{"persist-first", "undo-first"} {
		// JS test (shared case definition): https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/undo.test.ts#L6
		t.Run(order, func(t *testing.T) {
			setItem := newSpy()
			removeItem := newSpy()
			storage := xstore.StorageFuncs{
				GetItemFunc: func(name string) (*string, *xstore.Promise[*string]) { return nil, nil },
				SetItemFunc: func(name, value string) *xstore.Promise[struct{}] {
					setItem.Call(name, value)
					return nil
				},
				RemoveItemFunc: func(name string) *xstore.Promise[struct{}] {
					removeItem.Call(name)
					return nil
				},
			}
			base := xstore.CreateStore(xstore.StoreConfig[ctx]{
				Context: ctx{Count: 0},
				On: map[string]xstore.StoreAssigner[ctx]{
					"inc": func(c ctx, _ xs.Event, _ *xstore.EnqueueObject[ctx]) (ctx, bool) {
						return ctx{Count: c.Count + 1}, true
					},
				},
			})
			persistExt := xstore.Persist(xstore.PersistOptions[ctx]{Name: "counter", Storage: storage})
			undoExt := xstore.UndoRedo(xstore.UndoRedoOptions[ctx]{Strategy: xstore.UndoRedoSnapshot})
			var s *xstore.Store[ctx]
			if order == "persist-first" {
				s = base.With(persistExt).With(undoExt)
			} else {
				s = base.With(undoExt).With(persistExt)
			}

			s.Trigger("inc")
			s.Trigger("undo")
			assert.Equal(t, 0, s.GetSnapshot().Context.Count)
			assert.True(t, xstore.IsHydrated(s))
			assert.NotPanics(t, func() { xstore.FlushStorage(s) })
			s.Trigger("redo")
			assert.Equal(t, 1, s.GetSnapshot().Context.Count)
			assert.True(t, xstore.IsHydrated(s))
		})
	}
}

// JS: preserves live extension metadata through custom restore triggers
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/undo.test.ts#L38
func TestStoreUndo_PreservesLiveExtensionMetadataThroughCustomRestoreTriggers(t *testing.T) {
	type ctx struct {
		Count int `json:"count"`
	}
	writes := newSpy()
	s := xstore.CreateStore(xstore.StoreConfig[ctx]{
		Context: ctx{Count: 0},
		On: map[string]xstore.StoreAssigner[ctx]{
			"inc": func(c ctx, _ xs.Event, _ *xstore.EnqueueObject[ctx]) (ctx, bool) {
				return ctx{Count: c.Count + 1}, true
			},
		},
	}).With(
		xstore.Persist(xstore.PersistOptions[ctx]{
			Name: "counter",
			Storage: xstore.StorageFuncs{
				GetItemFunc: func(name string) (*string, *xstore.Promise[*string]) { return nil, nil },
				SetItemFunc: func(name, value string) *xstore.Promise[struct{}] {
					writes.Call(name, value)
					return nil
				},
				RemoveItemFunc: func(name string) *xstore.Promise[struct{}] { return nil },
			},
		}),
	).With(
		xstore.UndoRedo(xstore.UndoRedoOptions[ctx]{
			Strategy: xstore.UndoRedoSnapshot,
			Restore: func(a xstore.UndoRedoRestoreArgs[ctx], enq *xstore.EnqueueObject[ctx]) ctx {
				enq.Trigger("inc")
				return a.Next
			},
		}),
	)

	s.Trigger("inc")
	writes.Reset()
	s.Trigger("undo")
	assert.True(t, xstore.IsHydrated(s))
	assert.Equal(t, 1, s.GetSnapshot().Context.Count)
	if assert.Equal(t, 1, writes.Count()) {
		assert.Equal(t, []any{
			"counter",
			toJSON(t, map[string]any{"context": map[string]any{"count": 1}, "version": 0}),
		}, writes.Calls()[0])
	}
}

// JS: keeps metadata updates produced by custom restore triggers
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/undo.test.ts#L71
func TestStoreUndo_KeepsMetadataUpdatesProducedByCustomRestoreTriggers(t *testing.T) {
	// JS: a Symbol('revision') key spread onto the snapshot; Go: ExtensionState["revision"].
	const revision = "revision"
	s := storeUndo1New(storeUndo1Handlers{"inc": storeUndo1Inc}).
		With(func(logic xstore.StoreLogic[storeUndo1Ctx]) xstore.StoreLogic[storeUndo1Ctx] {
			next := logic
			next.GetInitialSnapshot = func() *xstore.StoreSnapshot[storeUndo1Ctx] {
				snap := *logic.GetInitialSnapshot()
				snap.ExtensionState = storeUndo1WithExt(snap.ExtensionState, revision, 0)
				return &snap
			}
			next.Transition = func(snapshot *xstore.StoreSnapshot[storeUndo1Ctx], event xs.Event) xstore.StoreTransitionResult[storeUndo1Ctx] {
				res := logic.Transition(snapshot, event)
				n := *res.Snapshot
				n.ExtensionState = storeUndo1WithExt(n.ExtensionState, revision, snapshot.ExtensionState[revision].(int)+1)
				res.Snapshot = &n
				return res
			}
			return next
		}).
		With(xstore.UndoRedo(xstore.UndoRedoOptions[storeUndo1Ctx]{
			Strategy: xstore.UndoRedoSnapshot,
			Restore: func(a xstore.UndoRedoRestoreArgs[storeUndo1Ctx], enq *xstore.EnqueueObject[storeUndo1Ctx]) storeUndo1Ctx {
				enq.Trigger("inc")
				return a.Next
			},
		}))

	s.Trigger("inc")
	assert.Equal(t, 1, s.GetSnapshot().ExtensionState[revision])
	s.Trigger("undo")
	assert.Equal(t, 2, s.GetSnapshot().ExtensionState[revision])
	s.Trigger("redo")
	assert.Equal(t, 3, s.GetSnapshot().ExtensionState[revision])
}

// JS: should undo a single event
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/undo.test.ts#L112
func TestStoreUndo_ShouldUndoASingleEvent(t *testing.T) {
	s := storeUndo1New(storeUndo1Handlers{"inc": storeUndo1Inc}).With(xstore.UndoRedo[storeUndo1Ctx]())

	s.Trigger("inc")
	assert.Equal(t, 1, storeUndo1Count(s))

	s.Trigger("undo")
	assert.Equal(t, 0, storeUndo1Count(s))
}

// JS: should redo a previously undone event
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/undo.test.ts#L127
func TestStoreUndo_ShouldRedoAPreviouslyUndoneEvent(t *testing.T) {
	s := storeUndo1New(storeUndo1Handlers{"inc": storeUndo1Inc}).With(xstore.UndoRedo[storeUndo1Ctx]())

	s.Trigger("inc")
	s.Trigger("undo")
	s.Trigger("redo")
	assert.Equal(t, 1, storeUndo1Count(s))
}

// JS: should undo/redo multiple events, non-transactional
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/undo.test.ts#L141
func TestStoreUndo_ShouldUndoRedoMultipleEventsNonTransactional(t *testing.T) {
	s := storeUndo1New(storeUndo1Handlers{"inc": storeUndo1Inc}).With(xstore.UndoRedo[storeUndo1Ctx]())

	s.Trigger("inc")
	s.Trigger("inc")
	s.Trigger("inc")
	assert.Equal(t, 3, storeUndo1Count(s))
	s.Trigger("undo")
	assert.Equal(t, 2, storeUndo1Count(s))
	s.Trigger("undo")
	assert.Equal(t, 1, storeUndo1Count(s))
	s.Trigger("redo")
	assert.Equal(t, 2, storeUndo1Count(s))
	s.Trigger("redo")
	assert.Equal(t, 3, storeUndo1Count(s))
}

// JS: should group events by transaction ID
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/undo.test.ts#L163
func TestStoreUndo_ShouldGroupEventsByTransactionID(t *testing.T) {
	s := storeUndo1New(storeUndo1Handlers{"inc": storeUndo1Inc, "dec": storeUndo1Dec}).
		With(xstore.UndoRedo(xstore.UndoRedoOptions[storeUndo1Ctx]{
			GetTransactionID: func(ev xs.Event, _ *xstore.StoreSnapshot[storeUndo1Ctx]) string { return ev.EventType() },
		}))

	// First transaction
	s.Trigger("inc")
	s.Trigger("inc")
	assert.Equal(t, 2, storeUndo1Count(s))

	// Second transaction
	s.Trigger("dec")
	s.Trigger("dec")
	assert.Equal(t, 0, storeUndo1Count(s))

	// Undo second transaction (both decrements)
	s.Trigger("undo")
	assert.Equal(t, 2, storeUndo1Count(s))

	// Undo first transaction (both increments)
	s.Trigger("undo")
	assert.Equal(t, 0, storeUndo1Count(s))
}

// JS: should maintain correct state when interleaving undo/redo with new events
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/undo.test.ts#L191
func TestStoreUndo_ShouldMaintainCorrectStateWhenInterleavingUndoRedoWithNewEvents(t *testing.T) {
	s := storeUndo1New(storeUndo1Handlers{"inc": storeUndo1Inc, "dec": storeUndo1Dec}).With(xstore.UndoRedo[storeUndo1Ctx]())

	s.Trigger("inc") // 1
	assert.Equal(t, 1, storeUndo1Count(s))
	s.Trigger("inc") // 2
	assert.Equal(t, 2, storeUndo1Count(s))
	s.Trigger("undo") // 1
	assert.Equal(t, 1, storeUndo1Count(s))
	s.Trigger("dec") // 0
	assert.Equal(t, 0, storeUndo1Count(s))
	s.Trigger("undo") // 1
	assert.Equal(t, 1, storeUndo1Count(s))
	s.Trigger("redo") // 0
	assert.Equal(t, 0, storeUndo1Count(s))

	assert.Equal(t, 0, storeUndo1Count(s))
}

// JS: should do nothing when undoing with empty history
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/undo.test.ts#L216
func TestStoreUndo_ShouldDoNothingWhenUndoingWithEmptyHistory(t *testing.T) {
	s := storeUndo1New(storeUndo1Handlers{"inc": storeUndo1Inc}).With(xstore.UndoRedo[storeUndo1Ctx]())

	initialSnapshot := s.GetSnapshot()
	s.Trigger("undo")
	assert.Equal(t, initialSnapshot, s.GetSnapshot())
}

// JS: should do nothing when redoing with empty undo stack
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/undo.test.ts#L229
func TestStoreUndo_ShouldDoNothingWhenRedoingWithEmptyUndoStack(t *testing.T) {
	s := storeUndo1New(storeUndo1Handlers{"inc": storeUndo1Inc}).With(xstore.UndoRedo[storeUndo1Ctx]())

	initialSnapshot := s.GetSnapshot()
	s.Trigger("redo")
	assert.Equal(t, initialSnapshot, s.GetSnapshot())
}

// JS: should clear redo stack when new events occur after undo
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/undo.test.ts#L242
func TestStoreUndo_ShouldClearRedoStackWhenNewEventsOccurAfterUndo(t *testing.T) {
	s := storeUndo1New(storeUndo1Handlers{"inc": storeUndo1Inc, "dec": storeUndo1Dec}).With(xstore.UndoRedo[storeUndo1Ctx]())

	s.Trigger("inc") // 1
	assert.Equal(t, 1, storeUndo1Count(s))
	s.Trigger("inc") // 2
	assert.Equal(t, 2, storeUndo1Count(s))
	s.Trigger("undo") // 1
	assert.Equal(t, 1, storeUndo1Count(s))
	s.Trigger("dec") // 0

	// Redo should not work as we added a new event after undo
	s.Trigger("redo")
	assert.Equal(t, 0, storeUndo1Count(s))
}

// JS: should preserve emitted events during undo/redo
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/undo.test.ts#L264
func TestStoreUndo_ShouldPreserveEmittedEventsDuringUndoRedo(t *testing.T) {
	s := xstore.CreateStore(xstore.StoreConfig[storeUndo1Ctx]{
		Context: storeUndo1Ctx{Count: 0},
		Schemas: &xstore.StoreSchemas{
			Emitted: map[string]xstore.Schema{
				"changed": zObject(map[string]*zSchema{"value": zNumber()}),
			},
		},
		On: storeUndo1Handlers{
			"inc": func(c storeUndo1Ctx, _ xs.Event, enq *xstore.EnqueueObject[storeUndo1Ctx]) (storeUndo1Ctx, bool) {
				enq.Emit("changed", xs.E{"value": c.Count + 1})
				return storeUndo1Ctx{Count: c.Count + 1}, true
			},
		},
	}).With(xstore.UndoRedo[storeUndo1Ctx]())

	var emittedEvents []xs.Event
	s.On("changed", func(e xs.Event) { emittedEvents = append(emittedEvents, e) })

	s.Trigger("inc")
	s.Trigger("undo")
	s.Trigger("redo")

	assert.Equal(t, []xs.Event{
		xs.E{"type": "changed", "value": 1},
		xs.E{"type": "changed", "value": 1},
	}, emittedEvents)
}

// JS: should preserve context and event types
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/undo.test.ts#L297
func TestStoreUndo_ShouldPreserveContextAndEventTypes(t *testing.T) {
	t.Skip("N/A: type-level only — `satisfies { count: number }` and `@ts-expect-error` on unknown context key / unknown trigger; the only runtime calls (trigger inc/undo/redo) have no assertions")
}

// JS: should skip non-undoable events during undo
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/undo.test.ts#L319
func TestStoreUndo_ShouldSkipNonUndoableEventsDuringUndo(t *testing.T) {
	s := storeUndo1New(storeUndo1Handlers{"inc": storeUndo1Inc, "log": storeUndo1Identity}).
		With(xstore.UndoRedo(xstore.UndoRedoOptions[storeUndo1Ctx]{
			SkipEvent: func(ev xs.Event, _ *xstore.StoreSnapshot[storeUndo1Ctx]) bool { return ev.EventType() == "log" },
		}))

	s.Trigger("inc") // count = 1
	s.Trigger("log") // count = 1 (logged but not undoable)
	s.Trigger("inc") // count = 2
	assert.Equal(t, 2, storeUndo1Count(s))

	s.Trigger("undo") // count = 1 (skips log event)
	assert.Equal(t, 1, storeUndo1Count(s))
}

// JS: should skip non-redoable events during redo
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/undo.test.ts#L337
func TestStoreUndo_ShouldSkipNonRedoableEventsDuringRedo(t *testing.T) {
	s := storeUndo1New(storeUndo1Handlers{"inc": storeUndo1Inc, "log": storeUndo1Identity}).
		With(xstore.UndoRedo(xstore.UndoRedoOptions[storeUndo1Ctx]{
			SkipEvent: func(ev xs.Event, _ *xstore.StoreSnapshot[storeUndo1Ctx]) bool { return ev.EventType() == "log" },
		}))

	s.Trigger("inc")  // count = 1
	s.Trigger("log")  // count = 1 (logged but not redoable)
	s.Trigger("inc")  // count = 2
	s.Trigger("undo") // count = 1
	assert.Equal(t, 1, storeUndo1Count(s))

	s.Trigger("redo") // count = 2 (skips log event)
	assert.Equal(t, 2, storeUndo1Count(s))
}

// JS: should skip events with transaction grouping
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/undo.test.ts#L356
func TestStoreUndo_ShouldSkipEventsWithTransactionGrouping(t *testing.T) {
	s := storeUndo1New(storeUndo1Handlers{"inc": storeUndo1Inc, "log": storeUndo1Identity}).
		With(xstore.UndoRedo(xstore.UndoRedoOptions[storeUndo1Ctx]{
			GetTransactionID: func(ev xs.Event, _ *xstore.StoreSnapshot[storeUndo1Ctx]) string { return ev.EventType() },
			SkipEvent:        func(ev xs.Event, _ *xstore.StoreSnapshot[storeUndo1Ctx]) bool { return ev.EventType() == "log" },
		}))

	// First transaction: inc events
	s.Trigger("inc") // count = 1
	s.Trigger("inc") // count = 2
	assert.Equal(t, 2, storeUndo1Count(s))

	// Log events (not a transaction because they're skipped)
	s.Trigger("log") // count = 2 (logged but not undoable)
	s.Trigger("log") // count = 2 (logged but not undoable)
	assert.Equal(t, 2, storeUndo1Count(s))

	// Second transaction: inc events
	s.Trigger("inc") // count = 3
	s.Trigger("inc") // count = 4
	assert.Equal(t, 4, storeUndo1Count(s))

	// Undo second transaction (all inc events)
	s.Trigger("undo") // count = 0
	assert.Equal(t, 0, storeUndo1Count(s))
}

// JS: should handle mixed undoable and non-undoable events
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/undo.test.ts#L390
func TestStoreUndo_ShouldHandleMixedUndoableAndNonUndoableEvents(t *testing.T) {
	type ctx struct {
		Count int
		Logs  []string
	}
	s := xstore.CreateStore(xstore.StoreConfig[ctx]{
		Context: ctx{Count: 0, Logs: []string{}},
		On: map[string]xstore.StoreAssigner[ctx]{
			"inc": func(c ctx, _ xs.Event, _ *xstore.EnqueueObject[ctx]) (ctx, bool) {
				return ctx{Count: c.Count + 1, Logs: c.Logs}, true
			},
			"log": func(c ctx, ev xs.Event, _ *xstore.EnqueueObject[ctx]) (ctx, bool) {
				logs := append(append([]string{}, c.Logs...), ev.(xs.E)["message"].(string))
				return ctx{Logs: logs, Count: c.Count}, true
			},
		},
	}).With(xstore.UndoRedo(xstore.UndoRedoOptions[ctx]{
		SkipEvent: func(ev xs.Event, _ *xstore.StoreSnapshot[ctx]) bool { return ev.EventType() == "log" },
	}))

	s.Trigger("inc")                                // count = 1
	s.Trigger("log", xs.E{"message": "first log"})  // logs = ['first log'] (not stored in history)
	s.Trigger("inc")                                // count = 2
	s.Trigger("log", xs.E{"message": "second log"}) // logs = ['first log', 'second log'] (not stored in history)
	s.Trigger("inc")                                // count = 3

	assert.Equal(t, 3, s.GetSnapshot().Context.Count)
	assert.Equal(t, []string{"first log", "second log"}, s.GetSnapshot().Context.Logs)

	// Undo should skip log events (they're not in history) but still undo inc events
	// Since log events are skipped, they're not replayed during undo, so logs are lost
	s.Trigger("undo") // count = 2, logs = [] (logs lost because not replayed)
	assert.Equal(t, 2, s.GetSnapshot().Context.Count)
	assert.Equal(t, []string{}, s.GetSnapshot().Context.Logs)

	s.Trigger("undo") // count = 1, logs = [] (logs lost because not replayed)
	assert.Equal(t, 1, s.GetSnapshot().Context.Count)
	assert.Equal(t, []string{}, s.GetSnapshot().Context.Logs)

	s.Trigger("undo") // count = 0, logs = [] (logs lost because not replayed)
	assert.Equal(t, 0, s.GetSnapshot().Context.Count)
	assert.Equal(t, []string{}, s.GetSnapshot().Context.Logs)
}

// JS: should not replay emitted events for skipped events during undo/redo
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/undo.test.ts#L430
func TestStoreUndo_ShouldNotReplayEmittedEventsForSkippedEventsDuringUndoRedo(t *testing.T) {
	s := xstore.CreateStore(xstore.StoreConfig[storeUndo1Ctx]{
		Context: storeUndo1Ctx{Count: 0},
		Schemas: &xstore.StoreSchemas{
			Emitted: map[string]xstore.Schema{
				"changed": zObject(map[string]*zSchema{"value": zNumber()}),
				"logged":  zObject(map[string]*zSchema{"message": zString()}),
			},
		},
		On: storeUndo1Handlers{
			"inc": func(c storeUndo1Ctx, _ xs.Event, enq *xstore.EnqueueObject[storeUndo1Ctx]) (storeUndo1Ctx, bool) {
				enq.Emit("changed", xs.E{"value": c.Count + 1})
				return storeUndo1Ctx{Count: c.Count + 1}, true
			},
			"log": func(c storeUndo1Ctx, ev xs.Event, enq *xstore.EnqueueObject[storeUndo1Ctx]) (storeUndo1Ctx, bool) {
				enq.Emit("logged", xs.E{"message": ev.(xs.E)["message"]})
				return c, true // No state change
			},
		},
	}).With(xstore.UndoRedo(xstore.UndoRedoOptions[storeUndo1Ctx]{
		SkipEvent: func(ev xs.Event, _ *xstore.StoreSnapshot[storeUndo1Ctx]) bool { return ev.EventType() == "log" },
	}))

	var emittedEvents []xs.Event
	s.On("changed", func(e xs.Event) { emittedEvents = append(emittedEvents, e) })
	s.On("logged", func(e xs.Event) { emittedEvents = append(emittedEvents, e) })

	s.Trigger("inc")                              // count = 1, emits changed(1)
	s.Trigger("log", xs.E{"message": "test log"}) // emits logged('test log') but not stored in history
	s.Trigger("inc")                              // count = 2, emits changed(2)

	assert.Equal(t, []xs.Event{
		xs.E{"type": "changed", "value": 1},
		xs.E{"type": "logged", "message": "test log"},
		xs.E{"type": "changed", "value": 2},
	}, emittedEvents)

	emittedEvents = nil
	s.Trigger("undo") // count = 1
	s.Trigger("undo") // count = 0
	s.Trigger("redo") // count = 1, emits changed(1)
	s.Trigger("redo") // count = 2, emits changed(2)

	// Only inc events should be emitted during undo/redo, log events are skipped from history
	assert.Equal(t, []xs.Event{
		xs.E{"type": "changed", "value": 1},
		xs.E{"type": "changed", "value": 2},
	}, emittedEvents)
}

// JS: should skip events with transaction grouping (second test of this name; getTransactionId reads the snapshot)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/undo.test.ts#L486
func TestStoreUndo_ShouldSkipEventsWithTransactionGrouping_2(t *testing.T) {
	type ctx struct {
		Count         int
		TransactionID *string // JS: string | null
	}
	s := xstore.CreateStore(xstore.StoreConfig[ctx]{
		Context: ctx{Count: 0, TransactionID: nil},
		On: map[string]xstore.StoreAssigner[ctx]{
			"inc": func(c ctx, _ xs.Event, _ *xstore.EnqueueObject[ctx]) (ctx, bool) {
				c.Count++
				return c, true
			},
			"transactionIdUpdated": func(c ctx, ev xs.Event, _ *xstore.EnqueueObject[ctx]) (ctx, bool) {
				c.TransactionID = strPtr(ev.(xs.E)["id"].(string))
				return c, true
			},
		},
	}).With(xstore.UndoRedo(xstore.UndoRedoOptions[ctx]{
		// JS: null/undefined transaction id is ""
		GetTransactionID: func(_ xs.Event, snapshot *xstore.StoreSnapshot[ctx]) string {
			if snapshot.Context.TransactionID == nil {
				return ""
			}
			return *snapshot.Context.TransactionID
		},
	}))

	s.Trigger("inc") // count = 1
	s.Trigger("transactionIdUpdated", xs.E{"id": "1"})
	s.Trigger("inc")
	s.Trigger("inc")
	s.Trigger("inc") // count = 4
	s.Trigger("transactionIdUpdated", xs.E{"id": "2"})
	s.Trigger("inc")
	s.Trigger("inc")
	s.Trigger("inc") // count = 7

	s.Trigger("undo")
	assert.Equal(t, 4, s.GetSnapshot().Context.Count)
	s.Trigger("undo")
	assert.Equal(t, 1, s.GetSnapshot().Context.Count)
	s.Trigger("redo")
	assert.Equal(t, 4, s.GetSnapshot().Context.Count)
	s.Trigger("redo")
	assert.Equal(t, 7, s.GetSnapshot().Context.Count)
}

// JS: should use the snapshot in the skipEvent function
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/undo.test.ts#L522
func TestStoreUndo_ShouldUseTheSnapshotInTheSkipEventFunction(t *testing.T) {
	s := storeUndo1New(storeUndo1Handlers{"inc": storeUndo1Inc}).
		With(xstore.UndoRedo(xstore.UndoRedoOptions[storeUndo1Ctx]{
			SkipEvent: func(_ xs.Event, snapshot *xstore.StoreSnapshot[storeUndo1Ctx]) bool {
				return snapshot.Context.Count >= 3
			},
		}))

	s.Trigger("inc") // count = 1
	s.Trigger("inc") // count = 2
	s.Trigger("inc") // count = 3
	s.Trigger("inc") // count = 4 (skipped)
	assert.Equal(t, 4, storeUndo1Count(s))
	s.Trigger("undo") // count = 2
	assert.Equal(t, 2, storeUndo1Count(s))
}

// JS: emit event types should be correct
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/undo.test.ts#L545
func TestStoreUndo_EmitEventTypesShouldBeCorrect(t *testing.T) {
	t.Skip("N/A: type-level only — `@ts-expect-error` on unknown emit/on event names and `satisfies` on emitted payloads; handlers are never invoked at runtime")
}

// JS: should detect undo/redo event collisions in development
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/undo.test.ts#L579
func TestStoreUndo_ShouldDetectUndoRedoEventCollisionsInDevelopment(t *testing.T) {
	msg := panicMessage(func() {
		storeUndo1New(storeUndo1Handlers{"undo": storeUndo1Identity}).With(xstore.UndoRedo[storeUndo1Ctx]())
	})
	// JS toThrow(string) is a substring match.
	assert.Contains(t, msg, `The "undoRedo" store extension uses reserved event type(s): "undo".`)
}

// JS: undoRedo with snapshot strategy > should undo a single event
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/undo.test.ts#L593
func TestStoreUndo_Snapshot_ShouldUndoASingleEvent(t *testing.T) {
	s := storeUndo1New(storeUndo1Handlers{"inc": storeUndo1Inc}).
		With(xstore.UndoRedo(xstore.UndoRedoOptions[storeUndo1Ctx]{Strategy: xstore.UndoRedoSnapshot}))

	s.Trigger("inc")
	assert.Equal(t, 1, storeUndo1Count(s))

	s.Trigger("undo")
	assert.Equal(t, 0, storeUndo1Count(s))
}

// JS: undoRedo with snapshot strategy > should redo a previously undone event
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/undo.test.ts#L608
func TestStoreUndo_Snapshot_ShouldRedoAPreviouslyUndoneEvent(t *testing.T) {
	s := storeUndo1New(storeUndo1Handlers{"inc": storeUndo1Inc}).
		With(xstore.UndoRedo(xstore.UndoRedoOptions[storeUndo1Ctx]{Strategy: xstore.UndoRedoSnapshot}))

	s.Trigger("inc")
	s.Trigger("undo")
	s.Trigger("redo")
	assert.Equal(t, 1, storeUndo1Count(s))
}

// JS: undoRedo with snapshot strategy > should undo/redo multiple events, non-transactional
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/undo.test.ts#L622
func TestStoreUndo_Snapshot_ShouldUndoRedoMultipleEventsNonTransactional(t *testing.T) {
	s := storeUndo1New(storeUndo1Handlers{"inc": storeUndo1Inc}).
		With(xstore.UndoRedo(xstore.UndoRedoOptions[storeUndo1Ctx]{Strategy: xstore.UndoRedoSnapshot}))

	s.Trigger("inc")
	s.Trigger("inc")
	s.Trigger("inc")
	assert.Equal(t, 3, storeUndo1Count(s))
	s.Trigger("undo")
	assert.Equal(t, 2, storeUndo1Count(s))
	s.Trigger("undo")
	assert.Equal(t, 1, storeUndo1Count(s))
	s.Trigger("redo")
	assert.Equal(t, 2, storeUndo1Count(s))
	s.Trigger("redo")
	assert.Equal(t, 3, storeUndo1Count(s))
}

// JS: undoRedo with snapshot strategy > should undo back into history after a redo
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/undo.test.ts#L644
func TestStoreUndo_Snapshot_ShouldUndoBackIntoHistoryAfterARedo(t *testing.T) {
	s := storeUndo1New(storeUndo1Handlers{"inc": storeUndo1Inc}).
		With(xstore.UndoRedo(xstore.UndoRedoOptions[storeUndo1Ctx]{Strategy: xstore.UndoRedoSnapshot}))

	s.Trigger("inc")
	s.Trigger("inc")
	s.Trigger("inc")
	s.Trigger("undo")
	s.Trigger("undo")
	assert.Equal(t, 1, storeUndo1Count(s))
	s.Trigger("redo")
	assert.Equal(t, 2, storeUndo1Count(s))

	// Undoing the redo returns to the snapshot the redo moved away from
	s.Trigger("undo")
	assert.Equal(t, 1, storeUndo1Count(s))
	s.Trigger("undo")
	assert.Equal(t, 0, storeUndo1Count(s))
	s.Trigger("redo")
	assert.Equal(t, 1, storeUndo1Count(s))
}

// JS: undoRedo with snapshot strategy > should group events by transaction ID
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/undo.test.ts#L670
func TestStoreUndo_Snapshot_ShouldGroupEventsByTransactionID(t *testing.T) {
	s := storeUndo1New(storeUndo1Handlers{"inc": storeUndo1Inc, "dec": storeUndo1Dec}).
		With(xstore.UndoRedo(xstore.UndoRedoOptions[storeUndo1Ctx]{
			Strategy:         xstore.UndoRedoSnapshot,
			GetTransactionID: func(ev xs.Event, _ *xstore.StoreSnapshot[storeUndo1Ctx]) string { return ev.EventType() },
		}))

	// First transaction
	s.Trigger("inc")
	s.Trigger("inc")
	assert.Equal(t, 2, storeUndo1Count(s))

	// Second transaction
	s.Trigger("dec")
	s.Trigger("dec")
	assert.Equal(t, 0, storeUndo1Count(s))

	// Undo second transaction (both decrements)
	s.Trigger("undo")
	assert.Equal(t, 2, storeUndo1Count(s))

	// Undo first transaction (both increments)
	s.Trigger("undo")
	assert.Equal(t, 0, storeUndo1Count(s))
}

// JS: undoRedo with snapshot strategy > should undo back into history after redoing a transaction
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/undo.test.ts#L705
func TestStoreUndo_Snapshot_ShouldUndoBackIntoHistoryAfterRedoingATransaction(t *testing.T) {
	s := storeUndo1New(storeUndo1Handlers{"inc": storeUndo1Inc, "dec": storeUndo1Dec}).
		With(xstore.UndoRedo(xstore.UndoRedoOptions[storeUndo1Ctx]{
			Strategy:         xstore.UndoRedoSnapshot,
			GetTransactionID: func(ev xs.Event, _ *xstore.StoreSnapshot[storeUndo1Ctx]) string { return ev.EventType() },
		}))

	// First transaction
	s.Trigger("inc")
	s.Trigger("inc")

	// Second transaction
	s.Trigger("dec")
	s.Trigger("dec")

	s.Trigger("undo")
	s.Trigger("undo")
	assert.Equal(t, 0, storeUndo1Count(s))

	// Redo the first transaction, then undo it again
	s.Trigger("redo")
	assert.Equal(t, 2, storeUndo1Count(s))
	s.Trigger("undo")
	assert.Equal(t, 0, storeUndo1Count(s))
}

// JS: undoRedo with snapshot strategy > should maintain correct state when interleaving undo/redo with new events
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/undo.test.ts#L740
func TestStoreUndo_Snapshot_ShouldMaintainCorrectStateWhenInterleavingUndoRedoWithNewEvents(t *testing.T) {
	s := storeUndo1New(storeUndo1Handlers{"inc": storeUndo1Inc, "dec": storeUndo1Dec}).
		With(xstore.UndoRedo(xstore.UndoRedoOptions[storeUndo1Ctx]{Strategy: xstore.UndoRedoSnapshot}))

	s.Trigger("inc") // 1
	assert.Equal(t, 1, storeUndo1Count(s))
	s.Trigger("inc") // 2
	assert.Equal(t, 2, storeUndo1Count(s))
	s.Trigger("undo") // 1
	assert.Equal(t, 1, storeUndo1Count(s))
	s.Trigger("dec") // 0
	assert.Equal(t, 0, storeUndo1Count(s))
	s.Trigger("undo") // 1
	assert.Equal(t, 1, storeUndo1Count(s))
	s.Trigger("redo") // 0
	assert.Equal(t, 0, storeUndo1Count(s))

	assert.Equal(t, 0, storeUndo1Count(s))
}

// JS: undoRedo with snapshot strategy > should do nothing when undoing with empty history
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/undo.test.ts#L765
func TestStoreUndo_Snapshot_ShouldDoNothingWhenUndoingWithEmptyHistory(t *testing.T) {
	s := storeUndo1New(storeUndo1Handlers{"inc": storeUndo1Inc}).
		With(xstore.UndoRedo(xstore.UndoRedoOptions[storeUndo1Ctx]{Strategy: xstore.UndoRedoSnapshot}))

	initialSnapshot := s.GetSnapshot()
	s.Trigger("undo")
	assert.Equal(t, initialSnapshot.Context, s.GetSnapshot().Context)
}

// JS: undoRedo with snapshot strategy > should do nothing when redoing with empty future stack
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/undo.test.ts#L778
func TestStoreUndo_Snapshot_ShouldDoNothingWhenRedoingWithEmptyFutureStack(t *testing.T) {
	s := storeUndo1New(storeUndo1Handlers{"inc": storeUndo1Inc}).
		With(xstore.UndoRedo(xstore.UndoRedoOptions[storeUndo1Ctx]{Strategy: xstore.UndoRedoSnapshot}))

	initialSnapshot := s.GetSnapshot()
	s.Trigger("redo")
	assert.Equal(t, initialSnapshot.Context, s.GetSnapshot().Context)
}

// JS: undoRedo with snapshot strategy > should clear redo stack when new events occur after undo
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/undo.test.ts#L791
func TestStoreUndo_Snapshot_ShouldClearRedoStackWhenNewEventsOccurAfterUndo(t *testing.T) {
	s := storeUndo1New(storeUndo1Handlers{"inc": storeUndo1Inc, "dec": storeUndo1Dec}).
		With(xstore.UndoRedo(xstore.UndoRedoOptions[storeUndo1Ctx]{Strategy: xstore.UndoRedoSnapshot}))

	s.Trigger("inc") // 1
	assert.Equal(t, 1, storeUndo1Count(s))
	s.Trigger("inc") // 2
	assert.Equal(t, 2, storeUndo1Count(s))
	s.Trigger("undo") // 1
	assert.Equal(t, 1, storeUndo1Count(s))
	s.Trigger("dec") // 0

	// Redo should not work as we added a new event after undo
	s.Trigger("redo")
	assert.Equal(t, 0, storeUndo1Count(s))
}

// JS: undoRedo with snapshot strategy > should skip non-undoable events
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/undo.test.ts#L813
func TestStoreUndo_Snapshot_ShouldSkipNonUndoableEvents(t *testing.T) {
	s := storeUndo1New(storeUndo1Handlers{"inc": storeUndo1Inc, "log": storeUndo1Identity}).
		With(xstore.UndoRedo(xstore.UndoRedoOptions[storeUndo1Ctx]{
			Strategy:  xstore.UndoRedoSnapshot,
			SkipEvent: func(ev xs.Event, _ *xstore.StoreSnapshot[storeUndo1Ctx]) bool { return ev.EventType() == "log" },
		}))

	s.Trigger("inc") // count = 1
	s.Trigger("log") // count = 1 (logged but not tracked)
	s.Trigger("inc") // count = 2
	assert.Equal(t, 2, storeUndo1Count(s))

	s.Trigger("undo") // count = 1 (skips log event)
	assert.Equal(t, 1, storeUndo1Count(s))
}

// JS: undoRedo with snapshot strategy > should respect historyLimit
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/undo.test.ts#L836
func TestStoreUndo_Snapshot_ShouldRespectHistoryLimit(t *testing.T) {
	s := storeUndo1New(storeUndo1Handlers{"inc": storeUndo1Inc}).
		With(xstore.UndoRedo(xstore.UndoRedoOptions[storeUndo1Ctx]{Strategy: xstore.UndoRedoSnapshot, HistoryLimit: 2}))

	s.Trigger("inc") // 1
	s.Trigger("inc") // 2
	s.Trigger("inc") // 3
	s.Trigger("inc") // 4

	// Can only undo 2 times because of history limit
	s.Trigger("undo") // 3
	assert.Equal(t, 3, storeUndo1Count(s))
	s.Trigger("undo") // 2
	assert.Equal(t, 2, storeUndo1Count(s))
	s.Trigger("undo") // Should stay at 2 (limit reached)
	assert.Equal(t, 2, storeUndo1Count(s))
}

// JS: undoRedo with snapshot strategy > should apply historyLimit during redo
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/undo.test.ts#L858
func TestStoreUndo_Snapshot_ShouldApplyHistoryLimitDuringRedo(t *testing.T) {
	s := storeUndo1New(storeUndo1Handlers{"inc": storeUndo1Inc}).
		With(xstore.UndoRedo(xstore.UndoRedoOptions[storeUndo1Ctx]{Strategy: xstore.UndoRedoSnapshot, HistoryLimit: 2}))

	s.Trigger("inc")  // 1
	s.Trigger("inc")  // 2
	s.Trigger("undo") // 1
	s.Trigger("undo") // 0
	s.Trigger("redo") // 1
	s.Trigger("redo") // 2
	s.Trigger("inc")  // 3
	s.Trigger("inc")  // 4

	// History should be trimmed to last 2 snapshots
	s.Trigger("undo") // 3
	s.Trigger("undo") // 2
	s.Trigger("undo") // Should stay at 2
	assert.Equal(t, 2, storeUndo1Count(s))
}

// JS: undoRedo with snapshot strategy > should preserve context with skipped events
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/undo.test.ts#L882
func TestStoreUndo_Snapshot_ShouldPreserveContextWithSkippedEvents(t *testing.T) {
	type ctx struct {
		Count int
		Logs  []string
	}
	s := xstore.CreateStore(xstore.StoreConfig[ctx]{
		Context: ctx{Count: 0, Logs: []string{}},
		On: map[string]xstore.StoreAssigner[ctx]{
			"inc": func(c ctx, _ xs.Event, _ *xstore.EnqueueObject[ctx]) (ctx, bool) {
				return ctx{Count: c.Count + 1, Logs: c.Logs}, true
			},
			"log": func(c ctx, ev xs.Event, _ *xstore.EnqueueObject[ctx]) (ctx, bool) {
				logs := append(append([]string{}, c.Logs...), ev.(xs.E)["message"].(string))
				return ctx{Logs: logs, Count: c.Count}, true
			},
		},
	}).With(xstore.UndoRedo(xstore.UndoRedoOptions[ctx]{
		Strategy:  xstore.UndoRedoSnapshot,
		SkipEvent: func(ev xs.Event, _ *xstore.StoreSnapshot[ctx]) bool { return ev.EventType() == "log" },
	}))

	s.Trigger("inc")                               // count = 1
	s.Trigger("log", xs.E{"message": "first log"}) // logs = ['first log'] (not tracked)
	s.Trigger("inc")                               // count = 2

	assert.Equal(t, 2, s.GetSnapshot().Context.Count)
	assert.Equal(t, []string{"first log"}, s.GetSnapshot().Context.Logs)

	// Undo should restore snapshot before second inc, which includes the log
	s.Trigger("undo") // count = 1, logs = ['first log']
	assert.Equal(t, 1, s.GetSnapshot().Context.Count)
	assert.Equal(t, []string{"first log"}, s.GetSnapshot().Context.Logs)
}

// JS: undoRedo with snapshot strategy > should handle transaction grouping with historyLimit
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/undo.test.ts#L912
func TestStoreUndo_Snapshot_ShouldHandleTransactionGroupingWithHistoryLimit(t *testing.T) {
	s := storeUndo1New(storeUndo1Handlers{"inc": storeUndo1Inc, "dec": storeUndo1Dec}).
		With(xstore.UndoRedo(xstore.UndoRedoOptions[storeUndo1Ctx]{
			Strategy:         xstore.UndoRedoSnapshot,
			GetTransactionID: func(ev xs.Event, _ *xstore.StoreSnapshot[storeUndo1Ctx]) string { return ev.EventType() },
			HistoryLimit:     3,
		}))

	// First transaction
	s.Trigger("inc") // 1
	s.Trigger("inc") // 2

	// Second transaction
	s.Trigger("dec") // 1
	s.Trigger("dec") // 0

	// Third transaction
	s.Trigger("inc") // 1
	s.Trigger("inc") // 2

	// Undo third transaction
	s.Trigger("undo") // 0
	assert.Equal(t, 0, storeUndo1Count(s))

	// Undo second transaction - only partial history available due to limit
	// The {2,dec} snapshot was trimmed, so we can only restore to {1,dec}
	s.Trigger("undo") // 1
	assert.Equal(t, 1, storeUndo1Count(s))

	// Can't undo further due to limit (first transaction's snapshots were trimmed)
	s.Trigger("undo") // Should stay at 1
	assert.Equal(t, 1, storeUndo1Count(s))
}

// JS: undoRedo with snapshot strategy > should use compare function to skip duplicate snapshots
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/undo.test.ts#L953
func TestStoreUndo_Snapshot_ShouldUseCompareFunctionToSkipDuplicateSnapshots(t *testing.T) {
	s := storeUndo1New(storeUndo1Handlers{"inc": storeUndo1Inc, "noop": storeUndo1Identity}).
		With(xstore.UndoRedo(xstore.UndoRedoOptions[storeUndo1Ctx]{
			Strategy: xstore.UndoRedoSnapshot,
			Compare: func(past, current *xstore.StoreSnapshot[storeUndo1Ctx]) bool {
				return past.Context.Count == current.Context.Count
			},
		}))

	s.Trigger("inc")  // count = 1
	s.Trigger("noop") // count = 1 (duplicate, not saved)
	s.Trigger("noop") // count = 1 (duplicate, not saved)
	s.Trigger("inc")  // count = 2

	// Should only have 2 snapshots in history (0 and 1), not 4
	s.Trigger("undo") // count = 1
	assert.Equal(t, 1, storeUndo1Count(s))
	s.Trigger("undo") // count = 0
	assert.Equal(t, 0, storeUndo1Count(s))
	s.Trigger("undo") // Should stay at 0
	assert.Equal(t, 0, storeUndo1Count(s))
}

// JS: undoRedo with snapshot strategy > should save all snapshots when no compare function is provided
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/undo.test.ts#L981
func TestStoreUndo_Snapshot_ShouldSaveAllSnapshotsWhenNoCompareFunctionIsProvided(t *testing.T) {
	s := storeUndo1New(storeUndo1Handlers{"inc": storeUndo1Inc, "noop": storeUndo1Identity}).
		With(xstore.UndoRedo(xstore.UndoRedoOptions[storeUndo1Ctx]{Strategy: xstore.UndoRedoSnapshot}))

	s.Trigger("inc")  // count = 1
	s.Trigger("noop") // count = 1 (saved even though duplicate)
	s.Trigger("noop") // count = 1 (saved even though duplicate)
	s.Trigger("inc")  // count = 2

	// Should have 4 snapshots in history (0, 1, 1, 1)
	s.Trigger("undo") // count = 1
	assert.Equal(t, 1, storeUndo1Count(s))
	s.Trigger("undo") // count = 1
	assert.Equal(t, 1, storeUndo1Count(s))
	s.Trigger("undo") // count = 1
	assert.Equal(t, 1, storeUndo1Count(s))
	s.Trigger("undo") // count = 0
	assert.Equal(t, 0, storeUndo1Count(s))
}

// JS: undoRedo with snapshot strategy > should preserve orthogonal context during undo and redo
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/undo.test.ts#L1006
func TestStoreUndo_Snapshot_ShouldPreserveOrthogonalContextDuringUndoAndRedo(t *testing.T) {
	type ctx struct {
		Document string
		Viewport int
	}
	s := xstore.CreateStore(xstore.StoreConfig[ctx]{
		Context: ctx{Document: "a", Viewport: 0},
		On: map[string]xstore.StoreAssigner[ctx]{
			"updateDocument": func(c ctx, ev xs.Event, _ *xstore.EnqueueObject[ctx]) (ctx, bool) {
				c.Document = ev.(xs.E)["document"].(string)
				return c, true
			},
			"updateViewport": func(c ctx, ev xs.Event, _ *xstore.EnqueueObject[ctx]) (ctx, bool) {
				c.Viewport = ev.(xs.E)["viewport"].(int)
				return c, true
			},
		},
	}).With(xstore.UndoRedo(xstore.UndoRedoOptions[ctx]{
		Strategy:  xstore.UndoRedoSnapshot,
		SkipEvent: func(ev xs.Event, _ *xstore.StoreSnapshot[ctx]) bool { return ev.EventType() == "updateViewport" },
		Restore: func(a xstore.UndoRedoRestoreArgs[ctx], _ *xstore.EnqueueObject[ctx]) ctx {
			next := a.Next
			next.Viewport = a.Current.Viewport
			return next
		},
	}))

	s.Trigger("updateDocument", xs.E{"document": "b"})
	s.Trigger("updateViewport", xs.E{"viewport": 10})
	s.Trigger("undo")
	assert.Equal(t, ctx{Document: "a", Viewport: 10}, s.GetSnapshot().Context)

	s.Trigger("redo")
	assert.Equal(t, ctx{Document: "b", Viewport: 10}, s.GetSnapshot().Context)

	s.Trigger("updateViewport", xs.E{"viewport": 20})
	s.Trigger("undo")
	assert.Equal(t, ctx{Document: "a", Viewport: 20}, s.GetSnapshot().Context)
}

// JS: undoRedo with snapshot strategy > should pass current, next, and direction to restore
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/undo.test.ts#L1052
func TestStoreUndo_Snapshot_ShouldPassCurrentNextAndDirectionToRestore(t *testing.T) {
	restore := newSpy()
	s := storeUndo1New(storeUndo1Handlers{"inc": storeUndo1Inc}).
		With(xstore.UndoRedo(xstore.UndoRedoOptions[storeUndo1Ctx]{
			Strategy: xstore.UndoRedoSnapshot,
			Restore: func(a xstore.UndoRedoRestoreArgs[storeUndo1Ctx], _ *xstore.EnqueueObject[storeUndo1Ctx]) storeUndo1Ctx {
				restore.Call(a)
				return a.Next
			},
		}))

	s.Trigger("inc")
	s.Trigger("undo")
	s.Trigger("redo")

	var args []any
	for _, call := range restore.Calls() {
		args = append(args, call[0])
	}
	assert.Equal(t, []any{
		xstore.UndoRedoRestoreArgs[storeUndo1Ctx]{Current: storeUndo1Ctx{Count: 1}, Next: storeUndo1Ctx{Count: 0}, Direction: xstore.DirectionUndo},
		xstore.UndoRedoRestoreArgs[storeUndo1Ctx]{Current: storeUndo1Ctx{Count: 0}, Next: storeUndo1Ctx{Count: 1}, Direction: xstore.DirectionRedo},
	}, args)
}

// JS: undoRedo with snapshot strategy > should restore once for a transaction group
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/undo.test.ts#L1069
func TestStoreUndo_Snapshot_ShouldRestoreOnceForATransactionGroup(t *testing.T) {
	restore := newSpy()
	s := storeUndo1New(storeUndo1Handlers{"inc": storeUndo1Inc}).
		With(xstore.UndoRedo(xstore.UndoRedoOptions[storeUndo1Ctx]{
			Strategy:         xstore.UndoRedoSnapshot,
			GetTransactionID: func(xs.Event, *xstore.StoreSnapshot[storeUndo1Ctx]) string { return "transaction" },
			Restore: func(a xstore.UndoRedoRestoreArgs[storeUndo1Ctx], _ *xstore.EnqueueObject[storeUndo1Ctx]) storeUndo1Ctx {
				restore.Call(a)
				return a.Next
			},
		}))

	s.Trigger("inc")
	s.Trigger("inc")
	s.Trigger("undo")
	assert.Equal(t, 0, storeUndo1Count(s))
	s.Trigger("redo")
	assert.Equal(t, 2, storeUndo1Count(s))
	s.Trigger("undo")
	assert.Equal(t, 0, storeUndo1Count(s))
	assert.Equal(t, 3, restore.Count())
}

// JS: undoRedo with snapshot strategy > should run emitted events after restored context commits
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/undo.test.ts#L1093
func TestStoreUndo_Snapshot_ShouldRunEmittedEventsAfterRestoredContextCommits(t *testing.T) {
	var observed []int
	s := xstore.CreateStore(xstore.StoreConfig[storeUndo1Ctx]{
		Schemas: &xstore.StoreSchemas{
			Emitted: map[string]xstore.Schema{
				"restored": zObject(map[string]*zSchema{"count": zNumber()}),
			},
		},
		Context: storeUndo1Ctx{Count: 0},
		On:      storeUndo1Handlers{"inc": storeUndo1Inc},
	}).With(xstore.UndoRedo(xstore.UndoRedoOptions[storeUndo1Ctx]{
		Strategy: xstore.UndoRedoSnapshot,
		Restore: func(a xstore.UndoRedoRestoreArgs[storeUndo1Ctx], enq *xstore.EnqueueObject[storeUndo1Ctx]) storeUndo1Ctx {
			enq.Emit("restored", xs.E{"count": a.Next.Count})
			return a.Next
		},
	}))
	s.On("restored", func(xs.Event) { observed = append(observed, storeUndo1Count(s)) })

	s.Trigger("inc")
	s.Trigger("undo")

	assert.Equal(t, []int{0}, observed)
}

// JS: undoRedo with snapshot strategy > should run effects after restored context commits
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/undo.test.ts#L1120
func TestStoreUndo_Snapshot_ShouldRunEffectsAfterRestoredContextCommits(t *testing.T) {
	var observed []int
	s := storeUndo1New(storeUndo1Handlers{"inc": storeUndo1Inc}).
		With(xstore.UndoRedo(xstore.UndoRedoOptions[storeUndo1Ctx]{
			Strategy: xstore.UndoRedoSnapshot,
			Restore: func(a xstore.UndoRedoRestoreArgs[storeUndo1Ctx], enq *xstore.EnqueueObject[storeUndo1Ctx]) storeUndo1Ctx {
				enq.Effect(func(e *xstore.StoreEffectEnqueue[storeUndo1Ctx]) {
					observed = append(observed, e.GetSnapshot().Context.Count)
				})
				return a.Next
			},
		}))

	s.Trigger("inc")
	s.Trigger("undo")

	assert.Equal(t, []int{0}, observed)
}

// JS: undoRedo with snapshot strategy > should apply triggered transitions and effects once without history
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/undo.test.ts#L1143
func TestStoreUndo_Snapshot_ShouldApplyTriggeredTransitionsAndEffectsOnceWithoutHistory(t *testing.T) {
	effect := newSpy()
	s := storeUndo1New(storeUndo1Handlers{
		"inc": storeUndo1Inc,
		"add": func(c storeUndo1Ctx, ev xs.Event, enq *xstore.EnqueueObject[storeUndo1Ctx]) (storeUndo1Ctx, bool) {
			enq.Effect(func(e *xstore.StoreEffectEnqueue[storeUndo1Ctx]) { effect.Call(e) })
			return storeUndo1Ctx{Count: c.Count + ev.(xs.E)["by"].(int)}, true
		},
	}).With(xstore.UndoRedo(xstore.UndoRedoOptions[storeUndo1Ctx]{
		Strategy: xstore.UndoRedoSnapshot,
		Restore: func(a xstore.UndoRedoRestoreArgs[storeUndo1Ctx], enq *xstore.EnqueueObject[storeUndo1Ctx]) storeUndo1Ctx {
			enq.Trigger("add", xs.E{"by": 10})
			return a.Next
		},
	}))

	s.Trigger("inc")
	assert.True(t, s.Can("undo"))
	assert.Equal(t, 0, effect.Count())
	s.Trigger("undo")

	assert.Equal(t, 10, storeUndo1Count(s))
	assert.Equal(t, 1, effect.Count())
	assert.False(t, s.Can("undo"))
}

// JS: undoRedo with snapshot strategy > should not execute enqueued effects when checking can
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/undo.test.ts#L1174
func TestStoreUndo_Snapshot_ShouldNotExecuteEnqueuedEffectsWhenCheckingCan(t *testing.T) {
	effect := newSpy()
	emitted := newSpy()
	s := xstore.CreateStore(xstore.StoreConfig[storeUndo1Ctx]{
		Schemas: &xstore.StoreSchemas{
			Emitted: map[string]xstore.Schema{"restored": zObject(map[string]*zSchema{})},
		},
		Context: storeUndo1Ctx{Count: 0},
		On:      storeUndo1Handlers{"inc": storeUndo1Inc},
	}).With(xstore.UndoRedo(xstore.UndoRedoOptions[storeUndo1Ctx]{
		Strategy: xstore.UndoRedoSnapshot,
		Restore: func(a xstore.UndoRedoRestoreArgs[storeUndo1Ctx], enq *xstore.EnqueueObject[storeUndo1Ctx]) storeUndo1Ctx {
			enq.Effect(func(e *xstore.StoreEffectEnqueue[storeUndo1Ctx]) { effect.Call(e) })
			enq.Emit("restored", xs.E{})
			return a.Next
		},
	}))
	s.On("restored", func(e xs.Event) { emitted.Call(e) })

	s.Trigger("inc")
	assert.True(t, s.Can("undo"))
	assert.Equal(t, 0, effect.Count())
	assert.Equal(t, 0, emitted.Count())
	s.Trigger("undo")
	assert.True(t, s.Can("redo"))
	assert.Equal(t, 1, effect.Count())
	assert.Equal(t, 1, emitted.Count())
}

// JS: undoRedo with snapshot strategy > should infer restore context, emitted events, and triggers
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/undo.test.ts#L1203
func TestStoreUndo_Snapshot_ShouldInferRestoreContextEmittedEventsAndTriggers(t *testing.T) {
	t.Skip("N/A: type-level only — `satisfies` and `@ts-expect-error` checks on restore args, emit and trigger payloads; the restore callback is never invoked at runtime")
}
