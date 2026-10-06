package store_test

import (
	"testing"

	xstore "github.com/nguyenvanduocit/go-xstate/store"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/assert"
)

// storeSelect1User mirrors TestContext.user.
type storeSelect1User struct {
	Name string
	Age  int
}

// storeSelect1Settings mirrors TestContext.settings.
type storeSelect1Settings struct {
	Theme         string
	Notifications bool
}

// storeSelect1Ctx mirrors the JS TestContext interface.
type storeSelect1Ctx struct {
	User     storeSelect1User
	Settings storeSelect1Settings
}

// storeSelect1NewStore mirrors the createStore({...}) config shared by the
// first five JS tests (UPDATE_NAME and UPDATE_THEME handlers).
func storeSelect1NewStore() *xstore.Store[storeSelect1Ctx] {
	return xstore.CreateStore(xstore.StoreConfig[storeSelect1Ctx]{
		Context: storeSelect1Ctx{
			User:     storeSelect1User{Name: "John", Age: 30},
			Settings: storeSelect1Settings{Theme: "dark", Notifications: true},
		},
		On: map[string]xstore.StoreAssigner[storeSelect1Ctx]{
			"UPDATE_NAME": func(c storeSelect1Ctx, ev xs.Event, _ *xstore.EnqueueObject[storeSelect1Ctx]) (storeSelect1Ctx, bool) {
				next := c
				next.User.Name = ev.(xs.E)["name"].(string)
				return next, true
			},
			"UPDATE_THEME": func(c storeSelect1Ctx, ev xs.Event, _ *xstore.EnqueueObject[storeSelect1Ctx]) (storeSelect1Ctx, bool) {
				next := c
				next.Settings.Theme = ev.(xs.E)["theme"].(string)
				return next, true
			},
		},
	})
}

// JS: select > should get current value
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/src/select.test.ts#L15
func TestStoreSelect_ShouldGetCurrentValue(t *testing.T) {
	store := storeSelect1NewStore()

	name := xstore.Select(store, func(state storeSelect1Ctx) string { return state.User.Name }).Get()
	assert.Equal(t, "John", name)
}

// JS: select > should subscribe to changes
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/src/select.test.ts#L37
func TestStoreSelect_ShouldSubscribeToChanges(t *testing.T) {
	store := storeSelect1NewStore()

	callback := newSpy()
	xstore.Select(store, func(state storeSelect1Ctx) string { return state.User.Name }).
		SubscribeNext(func(v string) { callback.Call(v) })
	store.Send(xs.E{"type": "UPDATE_NAME", "name": "Jane"})

	assert.Equal(t, 1, callback.Count())
	assert.Contains(t, callback.Calls(), []any{"Jane"})
}

// JS: select > should not notify if selected value has not changed
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/src/select.test.ts#L63
func TestStoreSelect_ShouldNotNotifyIfSelectedValueHasNotChanged(t *testing.T) {
	store := storeSelect1NewStore()

	callback := newSpy()
	xstore.Select(store, func(state storeSelect1Ctx) string { return state.User.Name }).
		SubscribeNext(func(v string) { callback.Call(v) })
	store.Send(xs.E{"type": "UPDATE_THEME", "theme": "light"})

	assert.Equal(t, 0, callback.Count())
}

// JS: select > should support custom equality function
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/src/select.test.ts#L88
func TestStoreSelect_ShouldSupportCustomEqualityFunction(t *testing.T) {
	store := storeSelect1NewStore()

	type selected struct {
		Name  string
		Theme string
	}

	callback := newSpy()
	selector := func(c storeSelect1Ctx) selected {
		return selected{Name: c.User.Name, Theme: c.Settings.Theme}
	}
	equalityFn := func(a, b selected) bool {
		return a.Name == b.Name // Only compare names
	}

	xstore.Select(store, selector, equalityFn).SubscribeNext(func(v selected) { callback.Call(v) })

	store.Send(xs.E{"type": "UPDATE_THEME", "theme": "light"})
	assert.Equal(t, 0, callback.Count())

	store.Send(xs.E{"type": "UPDATE_NAME", "name": "Jane"})
	assert.Equal(t, 1, callback.Count())
}

// JS: select > should unsubscribe correctly
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/src/select.test.ts#L123
func TestStoreSelect_ShouldUnsubscribeCorrectly(t *testing.T) {
	store := storeSelect1NewStore()

	callback := newSpy()
	subscription := xstore.Select(store, func(state storeSelect1Ctx) string { return state.User.Name }).
		SubscribeNext(func(v string) { callback.Call(v) })
	subscription.Unsubscribe()
	store.Send(xs.E{"type": "UPDATE_NAME", "name": "Jane"})

	assert.Equal(t, 0, callback.Count())
}

// JS: select > should handle updates with multiple subscribers
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/src/select.test.ts#L151
func TestStoreSelect_ShouldHandleUpdatesWithMultipleSubscribers(t *testing.T) {
	type position struct {
		X int
		Y int
	}
	type positionCtx struct {
		Position position
		User     storeSelect1User
	}

	store := xstore.CreateStore(xstore.StoreConfig[positionCtx]{
		Context: positionCtx{
			Position: position{X: 0, Y: 0},
			User:     storeSelect1User{Name: "John", Age: 30},
		},
		On: map[string]xstore.StoreAssigner[positionCtx]{
			"positionUpdated": func(c positionCtx, ev xs.Event, _ *xstore.EnqueueObject[positionCtx]) (positionCtx, bool) {
				next := c
				next.Position = ev.(xs.E)["position"].(position)
				return next, true
			},
			"userUpdated": func(c positionCtx, ev xs.Event, _ *xstore.EnqueueObject[positionCtx]) (positionCtx, bool) {
				next := c
				next.User = ev.(xs.E)["user"].(storeSelect1User)
				return next, true
			},
		},
	})

	// Mock DOM manipulation callback
	renderCallback := newSpy()
	xstore.Select(store, func(state positionCtx) position { return state.Position }).
		SubscribeNext(func(p position) {
			renderCallback.Call(p)
		})

	// Mock logger callback for x position only
	loggerCallback := newSpy()
	xstore.Select(store, func(state positionCtx) int { return state.Position.X }).
		SubscribeNext(func(x int) {
			loggerCallback.Call(x)
		})

	lastCall := func(s *spy) []any {
		calls := s.Calls()
		if len(calls) == 0 {
			return nil
		}
		return calls[len(calls)-1]
	}

	// Simulate position update
	store.Trigger("positionUpdated", xs.E{"position": position{X: 100, Y: 200}})

	// Verify render callback received full position update
	assert.Equal(t, 1, renderCallback.Count())
	assert.Contains(t, renderCallback.Calls(), []any{position{X: 100, Y: 200}})

	// Verify logger callback received only x position
	assert.Equal(t, 1, loggerCallback.Count())
	assert.Contains(t, loggerCallback.Calls(), []any{100})

	// Simulate another update
	store.Trigger("positionUpdated", xs.E{"position": position{X: 150, Y: 300}})

	assert.Equal(t, 2, renderCallback.Count())
	assert.Equal(t, []any{position{X: 150, Y: 300}}, lastCall(renderCallback))
	assert.Equal(t, 2, loggerCallback.Count())
	assert.Equal(t, []any{150}, lastCall(loggerCallback))

	// Simulate changing only the y position
	store.Trigger("positionUpdated", xs.E{"position": position{X: 150, Y: 400}})

	assert.Equal(t, 3, renderCallback.Count())
	assert.Equal(t, []any{position{X: 150, Y: 400}}, lastCall(renderCallback))

	// loggerCallback should not have been called
	assert.Equal(t, 2, loggerCallback.Count())

	// Simulate changing only the user
	store.Trigger("userUpdated", xs.E{"user": storeSelect1User{Name: "Jane", Age: 25}})

	// renderCallback should not have been called
	assert.Equal(t, 3, renderCallback.Count())

	// loggerCallback should not have been called
	assert.Equal(t, 2, loggerCallback.Count())
}
