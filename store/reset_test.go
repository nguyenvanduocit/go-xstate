package store_test

import (
	"testing"

	xstore "github.com/nguyenvanduocit/go-xstate/store"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/assert"
)

// JS: reset extension > should reset to initial context
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/reset.test.ts#L6
func TestStoreReset_ShouldResetToInitialContext(t *testing.T) {
	type ctx struct{ Count int }
	s := xstore.CreateStore(xstore.StoreConfig[ctx]{
		Context: ctx{Count: 0},
		On: map[string]xstore.StoreAssigner[ctx]{
			"inc": func(c ctx, ev xs.Event, enq *xstore.EnqueueObject[ctx]) (ctx, bool) {
				return ctx{Count: c.Count + 1}, true
			},
		},
	}).With(xstore.Reset[ctx]())

	s.Trigger("inc")
	s.Trigger("inc")
	assert.Equal(t, 2, s.GetSnapshot().Context.Count)

	s.Trigger("reset")
	assert.Equal(t, 0, s.GetSnapshot().Context.Count)
}

// JS: reset extension > should reset multiple fields to initial context
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/reset.test.ts#L22
func TestStoreReset_ShouldResetMultipleFieldsToInitialContext(t *testing.T) {
	type ctx struct {
		Count int
		Name  string
	}
	s := xstore.CreateStore(xstore.StoreConfig[ctx]{
		Context: ctx{Count: 0, Name: "Ada"},
		On: map[string]xstore.StoreAssigner[ctx]{
			"inc": func(c ctx, ev xs.Event, enq *xstore.EnqueueObject[ctx]) (ctx, bool) {
				c.Count++
				return c, true
			},
			"setName": func(c ctx, ev xs.Event, enq *xstore.EnqueueObject[ctx]) (ctx, bool) {
				c.Name = ev.(xs.E)["name"].(string)
				return c, true
			},
		},
	}).With(xstore.Reset[ctx]())

	s.Trigger("inc")
	s.Trigger("setName", xs.E{"name": "Bob"})
	assert.Equal(t, ctx{Count: 1, Name: "Bob"}, s.GetSnapshot().Context)

	s.Trigger("reset")
	assert.Equal(t, ctx{Count: 0, Name: "Ada"}, s.GetSnapshot().Context)
}

// JS: reset extension > should support partial reset via `to` option
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/reset.test.ts#L39
func TestStoreReset_ShouldSupportPartialResetViaToOption(t *testing.T) {
	type ctx struct {
		Count int
		User  *string // JS: string | null
	}
	s := xstore.CreateStore(xstore.StoreConfig[ctx]{
		Context: ctx{Count: 0, User: nil},
		On: map[string]xstore.StoreAssigner[ctx]{
			"inc": func(c ctx, ev xs.Event, enq *xstore.EnqueueObject[ctx]) (ctx, bool) {
				c.Count++
				return c, true
			},
			"login": func(c ctx, ev xs.Event, enq *xstore.EnqueueObject[ctx]) (ctx, bool) {
				c.User = strPtr(ev.(xs.E)["user"].(string))
				return c, true
			},
		},
	}).With(xstore.Reset(xstore.ResetOptions[ctx]{
		To: func(initial, current ctx) ctx {
			initial.User = current.User
			return initial
		},
	}))

	s.Trigger("inc")
	s.Trigger("inc")
	s.Trigger("login", xs.E{"user": "Alice"})
	assert.Equal(t, ctx{Count: 2, User: strPtr("Alice")}, s.GetSnapshot().Context)

	s.Trigger("reset")
	assert.Equal(t, ctx{Count: 0, User: strPtr("Alice")}, s.GetSnapshot().Context)
}

// JS: reset extension > should be idempotent when no changes have been made
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/reset.test.ts#L61
func TestStoreReset_ShouldBeIdempotentWhenNoChangesHaveBeenMade(t *testing.T) {
	type ctx struct{ Count int }
	s := xstore.CreateStore(xstore.StoreConfig[ctx]{
		Context: ctx{Count: 0},
		On: map[string]xstore.StoreAssigner[ctx]{
			"inc": func(c ctx, ev xs.Event, enq *xstore.EnqueueObject[ctx]) (ctx, bool) {
				return ctx{Count: c.Count + 1}, true
			},
		},
	}).With(xstore.Reset[ctx]())

	before := s.GetSnapshot()
	s.Trigger("reset")
	assert.Equal(t, before.Context, s.GetSnapshot().Context)
}

// JS: reset extension > should preserve snapshot status
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/reset.test.ts#L74
func TestStoreReset_ShouldPreserveSnapshotStatus(t *testing.T) {
	type ctx struct{ Count int }
	s := xstore.CreateStore(xstore.StoreConfig[ctx]{
		Context: ctx{Count: 0},
		On: map[string]xstore.StoreAssigner[ctx]{
			"inc": func(c ctx, ev xs.Event, enq *xstore.EnqueueObject[ctx]) (ctx, bool) {
				return ctx{Count: c.Count + 1}, true
			},
		},
	}).With(xstore.Reset[ctx]())

	s.Trigger("inc")
	s.Trigger("reset")
	assert.Equal(t, xs.StatusActive, s.GetSnapshot().Status)
}

// JS: reset extension > should notify subscribers on reset
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/reset.test.ts#L87
func TestStoreReset_ShouldNotifySubscribersOnReset(t *testing.T) {
	type ctx struct{ Count int }
	s := xstore.CreateStore(xstore.StoreConfig[ctx]{
		Context: ctx{Count: 0},
		On: map[string]xstore.StoreAssigner[ctx]{
			"inc": func(c ctx, ev xs.Event, enq *xstore.EnqueueObject[ctx]) (ctx, bool) {
				return ctx{Count: c.Count + 1}, true
			},
		},
	}).With(xstore.Reset[ctx]())

	snapshots := []int{}
	s.SubscribeNext(func(snap *xstore.StoreSnapshot[ctx]) {
		snapshots = append(snapshots, snap.Context.Count)
	})

	s.Trigger("inc")   // 1
	s.Trigger("inc")   // 2
	s.Trigger("reset") // 0

	assert.Equal(t, []int{1, 2, 0}, snapshots)
}

// JS: reset extension > should work with undoRedo (reset is undoable)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/reset.test.ts#L105
func TestStoreReset_ShouldWorkWithUndoRedoResetIsUndoable(t *testing.T) {
	type ctx struct{ Count int }
	s := xstore.CreateStore(xstore.StoreConfig[ctx]{
		Context: ctx{Count: 0},
		On: map[string]xstore.StoreAssigner[ctx]{
			"inc": func(c ctx, ev xs.Event, enq *xstore.EnqueueObject[ctx]) (ctx, bool) {
				return ctx{Count: c.Count + 1}, true
			},
		},
	}).
		With(xstore.Reset[ctx]()).
		With(xstore.UndoRedo[ctx]())

	s.Trigger("inc")
	s.Trigger("inc")
	assert.Equal(t, 2, s.GetSnapshot().Context.Count)

	s.Trigger("reset")
	assert.Equal(t, 0, s.GetSnapshot().Context.Count)

	s.Trigger("undo")
	assert.Equal(t, 2, s.GetSnapshot().Context.Count)
}

// JS: reset extension > should allow resetting after multiple operations
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/reset.test.ts#L126
func TestStoreReset_ShouldAllowResettingAfterMultipleOperations(t *testing.T) {
	type ctx struct{ Count int }
	s := xstore.CreateStore(xstore.StoreConfig[ctx]{
		Context: ctx{Count: 0},
		On: map[string]xstore.StoreAssigner[ctx]{
			"inc": func(c ctx, ev xs.Event, enq *xstore.EnqueueObject[ctx]) (ctx, bool) {
				return ctx{Count: c.Count + 1}, true
			},
			"dec": func(c ctx, ev xs.Event, enq *xstore.EnqueueObject[ctx]) (ctx, bool) {
				return ctx{Count: c.Count - 1}, true
			},
		},
	}).With(xstore.Reset[ctx]())

	s.Trigger("inc")
	s.Trigger("inc")
	s.Trigger("dec")
	s.Trigger("inc")
	assert.Equal(t, 2, s.GetSnapshot().Context.Count)

	s.Trigger("reset")
	assert.Equal(t, 0, s.GetSnapshot().Context.Count)

	// Should still work after reset
	s.Trigger("inc")
	assert.Equal(t, 1, s.GetSnapshot().Context.Count)
}

// JS: reset extension > should detect reset event collisions in development
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/reset.test.ts#L149
func TestStoreReset_ShouldDetectResetEventCollisionsInDevelopment(t *testing.T) {
	type ctx struct{ Count int }
	msg := panicMessage(func() {
		xstore.CreateStore(xstore.StoreConfig[ctx]{
			Context: ctx{Count: 0},
			On: map[string]xstore.StoreAssigner[ctx]{
				// JS: reset: (ctx) => ctx (returns the same context object)
				"reset": func(c ctx, ev xs.Event, enq *xstore.EnqueueObject[ctx]) (ctx, bool) {
					return c, true
				},
			},
		}).With(xstore.Reset[ctx]())
	})
	// toThrow(string) is a substring match
	assert.Contains(t, msg, `The "reset" store extension uses reserved event type(s): "reset".`)
}

// JS: reset extension > should return initial snapshot from getInitialSnapshot
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/reset.test.ts#L162
func TestStoreReset_ShouldReturnInitialSnapshotFromGetInitialSnapshot(t *testing.T) {
	type ctx struct{ Count int }
	s := xstore.CreateStore(xstore.StoreConfig[ctx]{
		Context: ctx{Count: 0},
		On: map[string]xstore.StoreAssigner[ctx]{
			"inc": func(c ctx, ev xs.Event, enq *xstore.EnqueueObject[ctx]) (ctx, bool) {
				return ctx{Count: c.Count + 1}, true
			},
		},
	}).With(xstore.Reset[ctx]())

	s.Trigger("inc")
	s.Trigger("inc")

	assert.Equal(t, 0, s.GetInitialSnapshot().Context.Count)
}
