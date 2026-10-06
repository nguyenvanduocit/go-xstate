// Package localcounter ports the store config of <Counter /> in
// examples/store/localcounter/src/App.tsx.
package localcounter

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

// Config mirrors the object passed to useStore in <Counter initialCount={initialCount} />.
func Config(initialCount int) xstore.StoreConfig[Context] {
	return xstore.StoreConfig[Context]{
		Context: Context{Count: initialCount},
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

// Counter mirrors <Counter />: useStore(config) plus
// useSelector(store, (s) => s.context.count).
type Counter struct {
	Store *xstore.Store[Context]
	Count xstore.ReadonlyAtom[int]
}

// NewCounter creates the store and the count selector of one <Counter />.
func NewCounter(initialCount int) *Counter {
	s := xstore.CreateStore(Config(initialCount))
	return &Counter{
		Store: s,
		Count: xstore.Select(s, func(c Context) int { return c.Count }),
	}
}

// InitialCounts are the initialCount props App renders.
var InitialCounts = []int{0, 10, 100}
