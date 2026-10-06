// Package counter ports the module-level store of
// examples/store/counter/src/App.tsx.
package counter

import (
	xstore "github.com/nguyenvanduocit/go-xstate/store"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// Context is the counter store's context.
type Context struct {
	Count int `json:"count"`
}

// by reads event.by; ints come from Go callers, float64 from JSON-decoded traces.
func by(ev xs.Event) int {
	switch v := ev.(xs.E)["by"].(type) {
	case int:
		return v
	case float64:
		return int(v)
	}
	return 0
}

// Config mirrors the object passed to createStore in App.tsx (lines 6-21).
func Config() xstore.StoreConfig[Context] {
	return xstore.StoreConfig[Context]{
		Context: Context{Count: 0},
		On: map[string]xstore.StoreAssigner[Context]{
			"inc": func(c Context, ev xs.Event, _ *xstore.EnqueueObject[Context]) (Context, bool) {
				return Context{Count: c.Count + by(ev)}, true
			},
			"reset": func(Context, xs.Event, *xstore.EnqueueObject[Context]) (Context, bool) {
				return Context{Count: 0}, true
			},
		},
	}
}

// Counter mirrors App.tsx: the store and the selector
// useSelector(store, (s) => s.context.count).
type Counter struct {
	Store *xstore.Store[Context]
	Count xstore.ReadonlyAtom[int]
}

// NewCounter creates the store and the count selector. Inspect the store with
// Counter.Store.Inspect (App.tsx passes createBrowserInspector().inspect there).
func NewCounter() *Counter {
	s := xstore.CreateStore(Config())
	return &Counter{
		Store: s,
		Count: xstore.Select(s, func(c Context) int { return c.Count }),
	}
}
