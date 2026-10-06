package store_test

import (
	"testing"

	xstore "github.com/nguyenvanduocit/go-xstate/store"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// storeStore1Counter is the `{ count: number }` context used by most tests.
type storeStore1Counter struct {
	Count int `json:"count"`
}

// storeStore1Payload returns the payload map of an event sent to a handler.
func storeStore1Payload(ev xs.Event) xs.E { return ev.(xs.E) }

// storeStore1Inc mirrors `(ctx) => ({ count: ctx.count + 1 })`.
func storeStore1Inc(c storeStore1Counter, _ xs.Event, _ *xstore.EnqueueObject[storeStore1Counter]) (storeStore1Counter, bool) {
	return storeStore1Counter{Count: c.Count + 1}, true
}

// JS: processes triggered events breadth-first when handlers append more events
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/store.test.ts#L17
func TestStoreStore_ProcessesTriggeredEventsBreadthFirstWhenHandlersAppendMoreEvents(t *testing.T) {
	var processed []int
	var effects []int
	store := xstore.CreateStore(xstore.StoreConfig[storeStore1Counter]{
		Context: storeStore1Counter{Count: 0},
		On: map[string]xstore.StoreAssigner[storeStore1Counter]{
			"start": func(c storeStore1Counter, _ xs.Event, enq *xstore.EnqueueObject[storeStore1Counter]) (storeStore1Counter, bool) {
				for id := 0; id < 100; id++ {
					enq.Trigger("item", xs.E{"id": id})
				}
				return c, false
			},
			"item": func(c storeStore1Counter, ev xs.Event, enq *xstore.EnqueueObject[storeStore1Counter]) (storeStore1Counter, bool) {
				id := storeStore1Payload(ev)["id"].(int)
				processed = append(processed, id)
				if id < 100 {
					enq.Trigger("item", xs.E{"id": id + 100})
				}
				enq.Effect(func(_ *xstore.StoreEffectEnqueue[storeStore1Counter]) {
					effects = append(effects, id)
				})
				return storeStore1Counter{Count: c.Count + 1}, true
			},
		},
	})

	store.Trigger("start")
	expected := make([]int, 200)
	for i := range expected {
		expected[i] = i
	}
	assert.Equal(t, expected, processed)
	assert.Equal(t, expected, effects)
	assert.Equal(t, 200, store.GetSnapshot().Context.Count)
}

// JS: updates a store with an event without mutating original context
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/store.test.ts#L46
func TestStoreStore_UpdatesAStoreWithAnEventWithoutMutatingOriginalContext(t *testing.T) {
	context := storeStore1Counter{Count: 0}
	store := xstore.CreateStore(xstore.StoreConfig[storeStore1Counter]{
		Context: context,
		On: map[string]xstore.StoreAssigner[storeStore1Counter]{
			"inc": func(c storeStore1Counter, ev xs.Event, _ *xstore.EnqueueObject[storeStore1Counter]) (storeStore1Counter, bool) {
				return storeStore1Counter{Count: c.Count + storeStore1Payload(ev)["by"].(int)}, true
			},
		},
	})

	initial := store.GetInitialSnapshot()

	store.Trigger("inc", xs.E{"by": 1})

	next := store.GetSnapshot()

	assert.Equal(t, storeStore1Counter{Count: 0}, initial.Context)
	assert.Equal(t, storeStore1Counter{Count: 1}, next.Context)
	assert.Equal(t, 0, context.Count)
}

// JS: can update context
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/store.test.ts#L72
func TestStoreStore_CanUpdateContext(t *testing.T) {
	type ctx struct {
		Count    int
		Greeting string
	}
	store := xstore.CreateStore(xstore.StoreConfig[ctx]{
		Context: ctx{Count: 0, Greeting: "hello"},
		On: map[string]xstore.StoreAssigner[ctx]{
			"inc": func(c ctx, _ xs.Event, _ *xstore.EnqueueObject[ctx]) (ctx, bool) {
				return ctx{Count: c.Count + 1, Greeting: c.Greeting}, true
			},
			"updateBoth": func(_ ctx, _ xs.Event, _ *xstore.EnqueueObject[ctx]) (ctx, bool) {
				return ctx{Count: 42, Greeting: "hi"}, true
			},
		},
	})

	store.Trigger("inc")
	assert.Equal(t, ctx{Count: 1, Greeting: "hello"}, store.GetSnapshot().Context)

	store.Trigger("updateBoth")
	assert.Equal(t, ctx{Count: 42, Greeting: "hi"}, store.GetSnapshot().Context)
}

// JS: handles unknown events sent via store.send (does not do anything)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/store.test.ts#L94
func TestStoreStore_HandlesUnknownEventsSentViaStoreSendDoesNotDoAnything(t *testing.T) {
	store := xstore.CreateStore(xstore.StoreConfig[storeStore1Counter]{
		Context: storeStore1Counter{Count: 0},
		On: map[string]xstore.StoreAssigner[storeStore1Counter]{
			"inc": storeStore1Inc,
		},
	})

	store.Send(xs.Ev("unknown"))
	assert.Equal(t, storeStore1Counter{Count: 0}, store.GetSnapshot().Context)
}

// JS: updates state from sent events
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/store.test.ts#L108
func TestStoreStore_UpdatesStateFromSentEvents(t *testing.T) {
	store := xstore.CreateStore(xstore.StoreConfig[storeStore1Counter]{
		Context: storeStore1Counter{Count: 0},
		On: map[string]xstore.StoreAssigner[storeStore1Counter]{
			"inc": func(c storeStore1Counter, ev xs.Event, _ *xstore.EnqueueObject[storeStore1Counter]) (storeStore1Counter, bool) {
				return storeStore1Counter{Count: c.Count + storeStore1Payload(ev)["by"].(int)}, true
			},
			"dec": func(c storeStore1Counter, ev xs.Event, _ *xstore.EnqueueObject[storeStore1Counter]) (storeStore1Counter, bool) {
				return storeStore1Counter{Count: c.Count - storeStore1Payload(ev)["by"].(int)}, true
			},
			"clear": func(_ storeStore1Counter, _ xs.Event, _ *xstore.EnqueueObject[storeStore1Counter]) (storeStore1Counter, bool) {
				return storeStore1Counter{Count: 0}, true
			},
		},
	})

	store.Trigger("inc", xs.E{"by": 9})
	store.Trigger("dec", xs.E{"by": 3})

	assert.Equal(t, storeStore1Counter{Count: 6}, store.GetSnapshot().Context)
	store.Trigger("clear")

	assert.Equal(t, storeStore1Counter{Count: 0}, store.GetSnapshot().Context)
}

// JS: can be observed
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/store.test.ts#L145
func TestStoreStore_CanBeObserved(t *testing.T) {
	store := xstore.CreateStore(xstore.StoreConfig[storeStore1Counter]{
		Context: storeStore1Counter{Count: 0},
		On: map[string]xstore.StoreAssigner[storeStore1Counter]{
			"inc": storeStore1Inc,
		},
	})

	var counts []int

	sub := store.SubscribeNext(func(s *xstore.StoreSnapshot[storeStore1Counter]) {
		counts = append(counts, s.Context.Count)
	})

	assert.Empty(t, counts)

	store.Trigger("inc") // 1
	store.Trigger("inc") // 2
	store.Trigger("inc") // 3

	assert.Equal(t, []int{1, 2, 3}, counts)

	sub.Unsubscribe()

	store.Trigger("inc") // 4
	store.Trigger("inc") // 5
	store.Trigger("inc") // 6

	assert.Equal(t, []int{1, 2, 3}, counts)
}

// JS: does not expose atom internals at runtime
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/store.test.ts#L178
func TestStoreStore_DoesNotExposeAtomInternalsAtRuntime(t *testing.T) {
	t.Skip("N/A: runtime-only — `'_snapshot' in store` probes JS object own-properties; Go struct fields are unexported by construction")
}

// JS: exposes schemas at runtime
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/store.test.ts#L187
func TestStoreStore_ExposesSchemasAtRuntime(t *testing.T) {
	schemas := &xstore.StoreSchemas{
		Context: zObject(map[string]*zSchema{"count": zNumber()}),
		Events: map[string]xstore.Schema{
			"inc": zObject(map[string]*zSchema{"by": zNumber()}),
		},
		Emitted: map[string]xstore.Schema{
			"increased": zObject(map[string]*zSchema{"by": zNumber()}),
		},
	}
	store := xstore.CreateStore(xstore.StoreConfig[storeStore1Counter]{
		Schemas: schemas,
		Context: storeStore1Counter{Count: 0},
		On: map[string]xstore.StoreAssigner[storeStore1Counter]{
			"inc": func(c storeStore1Counter, ev xs.Event, enq *xstore.EnqueueObject[storeStore1Counter]) (storeStore1Counter, bool) {
				by := storeStore1Payload(ev)["by"].(int)
				enq.Emit("increased", xs.E{"by": by})
				return storeStore1Counter{Count: c.Count + by}, true
			},
		},
	})

	assert.Same(t, schemas, store.Schemas())
}

// JS: exposes schemas after extension
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/store.test.ts#L211
func TestStoreStore_ExposesSchemasAfterExtension(t *testing.T) {
	schemas := &xstore.StoreSchemas{
		Context: zObject(map[string]*zSchema{"count": zNumber()}),
	}
	store := xstore.CreateStore(xstore.StoreConfig[storeStore1Counter]{
		Schemas: schemas,
		Context: storeStore1Counter{Count: 0},
		On:      map[string]xstore.StoreAssigner[storeStore1Counter]{},
	}).With(xstore.Reset[storeStore1Counter]())

	assert.Same(t, schemas, store.Schemas())
}

// JS: can be inspected
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/store.test.ts#L224
func TestStoreStore_CanBeInspected(t *testing.T) {
	store := xstore.CreateStore(xstore.StoreConfig[storeStore1Counter]{
		Context: storeStore1Counter{Count: 0},
		On: map[string]xstore.StoreAssigner[storeStore1Counter]{
			"inc": storeStore1Inc,
		},
	})

	var evs []xstore.StoreInspectionEvent

	store.Inspect(func(ev xstore.StoreInspectionEvent) { evs = append(evs, ev) })

	store.Trigger("inc")

	require.Len(t, evs, 2)

	assert.Equal(t, xs.InspectTransition, evs[0].Type)
	assert.Equal(t, xs.Ev("@xstate.init"), evs[0].Event)
	assert.Equal(t, storeStore1Counter{Count: 0}, evs[0].Snapshot.(*xstore.StoreSnapshot[storeStore1Counter]).Context)

	assert.Equal(t, xs.InspectTransition, evs[1].Type)
	assert.Equal(t, xs.Ev("inc"), evs[1].Event)
	assert.Equal(t, storeStore1Counter{Count: 1}, evs[1].Snapshot.(*xstore.StoreSnapshot[storeStore1Counter]).Context)
}

// JS: inspection with @statelyai/inspect typechecks correctly
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/store.test.ts#L256
func TestStoreStore_InspectionWithStatelyaiInspectTypechecksCorrectly(t *testing.T) {
	// JS: createBrowserInspector({autoStart: false}) from @statelyai/inspect has no
	// Go counterpart and the test asserts nothing. The runtime part (an empty
	// store handed an inspect callback) runs here as a smoke test.
	type ctx struct{}
	store := xstore.CreateStore(xstore.StoreConfig[ctx]{
		Context: ctx{},
		On:      map[string]xstore.StoreAssigner[ctx]{},
	})

	assert.NotPanics(t, func() {
		sub := store.Inspect(func(_ xstore.StoreInspectionEvent) {})
		sub.Unsubscribe()
	})
}

// JS: emitted events can be subscribed to
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/store.test.ts#L269
func TestStoreStore_EmittedEventsCanBeSubscribedTo(t *testing.T) {
	store := xstore.CreateStore(xstore.StoreConfig[storeStore1Counter]{
		Context: storeStore1Counter{Count: 0},
		Schemas: &xstore.StoreSchemas{
			Emitted: map[string]xstore.Schema{
				"increased": zObject(map[string]*zSchema{"upBy": zNumber()}),
			},
		},
		On: map[string]xstore.StoreAssigner[storeStore1Counter]{
			"inc": func(c storeStore1Counter, _ xs.Event, enq *xstore.EnqueueObject[storeStore1Counter]) (storeStore1Counter, bool) {
				enq.Emit("increased", xs.E{"upBy": 1})
				return storeStore1Counter{Count: c.Count + 1}, true
			},
		},
	})

	spy := newSpy()

	store.On("increased", func(e xs.Event) { spy.Call(e) })

	store.Trigger("inc")

	assert.Contains(t, spy.Calls(), []any{xs.E{"type": "increased", "upBy": 1}})
}

// JS: emitted events can be unsubscribed to
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/store.test.ts#L299
func TestStoreStore_EmittedEventsCanBeUnsubscribedTo(t *testing.T) {
	store := xstore.CreateStore(xstore.StoreConfig[storeStore1Counter]{
		Context: storeStore1Counter{Count: 0},
		Schemas: &xstore.StoreSchemas{
			Emitted: map[string]xstore.Schema{
				"increased": zObject(map[string]*zSchema{"upBy": zNumber()}),
			},
		},
		On: map[string]xstore.StoreAssigner[storeStore1Counter]{
			"inc": func(c storeStore1Counter, _ xs.Event, enq *xstore.EnqueueObject[storeStore1Counter]) (storeStore1Counter, bool) {
				enq.Emit("increased", xs.E{"upBy": 1})

				return storeStore1Counter{Count: c.Count + 1}, true
			},
		},
	})

	spy := newSpy()
	sub := store.On("increased", func(e xs.Event) { spy.Call(e) })
	store.Trigger("inc")

	assert.Contains(t, spy.Calls(), []any{xs.E{"type": "increased", "upBy": 1}})

	sub.Unsubscribe()
	store.Trigger("inc")

	assert.Equal(t, 1, spy.Count())
}

// JS: emitted events occur after the snapshot is updated
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/store.test.ts#L333
func TestStoreStore_EmittedEventsOccurAfterTheSnapshotIsUpdated(t *testing.T) {
	store := xstore.CreateStore(xstore.StoreConfig[storeStore1Counter]{
		Context: storeStore1Counter{Count: 0},
		Schemas: &xstore.StoreSchemas{
			Emitted: map[string]xstore.Schema{
				"increased": zObject(map[string]*zSchema{"upBy": zNumber()}),
			},
		},
		On: map[string]xstore.StoreAssigner[storeStore1Counter]{
			"inc": func(c storeStore1Counter, _ xs.Event, enq *xstore.EnqueueObject[storeStore1Counter]) (storeStore1Counter, bool) {
				enq.Emit("increased", xs.E{"upBy": 1})

				return storeStore1Counter{Count: c.Count + 1}, true
			},
		},
	})

	// expect.assertions(1): the handler must run exactly once.
	var seen []int

	store.On("increased", func(_ xs.Event) {
		s := store.GetSnapshot()

		seen = append(seen, s.Context.Count)
	})

	store.Trigger("inc")

	assert.Equal(t, []int{1}, seen)
}

// JS: events can be emitted with no payload
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/store.test.ts#L366
func TestStoreStore_EventsCanBeEmittedWithNoPayload(t *testing.T) {
	spy := newSpy()

	store := xstore.CreateStore(xstore.StoreConfig[storeStore1Counter]{
		Schemas: &xstore.StoreSchemas{
			Emitted: map[string]xstore.Schema{
				"incremented":    zObject(map[string]*zSchema{}),
				"decremented":    zObject(map[string]*zSchema{}),
				"expectsPayload": zObject(map[string]*zSchema{"payload": zString()}),
			},
		},
		Context: storeStore1Counter{Count: 0},
		On: map[string]xstore.StoreAssigner[storeStore1Counter]{
			"inc": func(c storeStore1Counter, _ xs.Event, enq *xstore.EnqueueObject[storeStore1Counter]) (storeStore1Counter, bool) {
				enq.Emit("incremented")
				return c, false
			},
			"dec": func(c storeStore1Counter, _ xs.Event, enq *xstore.EnqueueObject[storeStore1Counter]) (storeStore1Counter, bool) {
				enq.Emit("decremented", xs.E{})
				return c, false
			},
			"hasPayload": func(c storeStore1Counter, _ xs.Event, enq *xstore.EnqueueObject[storeStore1Counter]) (storeStore1Counter, bool) {
				// JS: @ts-expect-error Payload expected (never triggered)
				enq.Emit("expectsPayload")
				return c, false
			},
		},
	})

	store.On("incremented", func(e xs.Event) { spy.Call(e) })

	store.Trigger("inc")

	assert.Contains(t, spy.Calls(), []any{xs.E{"type": "incremented"}})
}

// JS: events can be emitted with optional payloads (type check)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/store.test.ts#L402
func TestStoreStore_EventsCanBeEmittedWithOptionalPayloadsTypeCheck(t *testing.T) {
	// JS: the test only creates the store; the handler is never invoked. The
	// `enq.emit.optionalPayload('foo')` @ts-expect-error call has no Go form
	// (payloads are xs.E) and is dropped. Runtime part: CreateStore accepts an
	// emitted schema with an optional payload field.
	type ctx struct{}
	assert.NotPanics(t, func() {
		xstore.CreateStore(xstore.StoreConfig[ctx]{
			Schemas: &xstore.StoreSchemas{
				Emitted: map[string]xstore.Schema{
					"optionalPayload": zObject(map[string]*zSchema{"payload": zOptional(zString())}),
				},
			},
			Context: ctx{},
			On: map[string]xstore.StoreAssigner[ctx]{
				"inc": func(c ctx, _ xs.Event, enq *xstore.EnqueueObject[ctx]) (ctx, bool) {
					enq.Emit("optionalPayload")

					enq.Emit("optionalPayload", xs.E{"payload": "hello"})

					enq.Emit("optionalPayload", xs.E{})

					return c, false
				},
			},
		})
	})
}

// JS: effects can be enqueued
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/store.test.ts#L427
func TestStoreStore_EffectsCanBeEnqueued(t *testing.T) {
	done := newSignal()
	// gate replaces the fixed `setTimeout(5)`: the async part of the effect runs
	// only after the synchronous assertion below, independent of scheduler timing.
	gate := newSignal()
	defer gate.Resolve()
	var store *xstore.Store[storeStore1Counter]
	store = xstore.CreateStore(xstore.StoreConfig[storeStore1Counter]{
		Context: storeStore1Counter{Count: 0},
		On: map[string]xstore.StoreAssigner[storeStore1Counter]{
			"inc": func(c storeStore1Counter, _ xs.Event, enq *xstore.EnqueueObject[storeStore1Counter]) (storeStore1Counter, bool) {
				enq.Effect(func(_ *xstore.StoreEffectEnqueue[storeStore1Counter]) {
					go func() {
						<-gate.ch // JS: await setTimeout(5)
						store.Trigger("dec")
						done.Resolve()
					}()
				})

				return storeStore1Counter{Count: c.Count + 1}, true
			},
			"dec": func(c storeStore1Counter, _ xs.Event, _ *xstore.EnqueueObject[storeStore1Counter]) (storeStore1Counter, bool) {
				return storeStore1Counter{Count: c.Count - 1}, true
			},
		},
	})

	store.Trigger("inc")

	assert.Equal(t, 1, store.GetSnapshot().Context.Count)
	gate.Resolve()

	done.Wait(t) // JS: await setTimeout(10)

	assert.Equal(t, 0, store.GetSnapshot().Context.Count)
}

// JS: events can be enqueued from transitions
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/store.test.ts#L461
func TestStoreStore_EventsCanBeEnqueuedFromTransitions(t *testing.T) {
	type ctx struct {
		Bears  int
		Fishes int
	}
	store := xstore.CreateStore(xstore.StoreConfig[ctx]{
		Context: ctx{Bears: 0, Fishes: 0},
		Schemas: &xstore.StoreSchemas{
			Events: map[string]xstore.Schema{
				"addBear":        zObject(map[string]*zSchema{}),
				"addFish":        zObject(map[string]*zSchema{"amount": zNumber()}),
				"addBearAndFish": zObject(map[string]*zSchema{}),
			},
		},
		On: map[string]xstore.StoreAssigner[ctx]{
			"addBear": func(c ctx, _ xs.Event, _ *xstore.EnqueueObject[ctx]) (ctx, bool) {
				return ctx{Bears: c.Bears + 1, Fishes: c.Fishes}, true
			},
			"addFish": func(c ctx, ev xs.Event, _ *xstore.EnqueueObject[ctx]) (ctx, bool) {
				return ctx{Bears: c.Bears, Fishes: c.Fishes + storeStore1Payload(ev)["amount"].(int)}, true
			},
			"addBearAndFish": func(c ctx, _ xs.Event, enq *xstore.EnqueueObject[ctx]) (ctx, bool) {
				enq.Trigger("addBear")
				enq.Trigger("addFish", xs.E{"amount": 1})

				return c, true
			},
		},
	})

	store.Trigger("addBearAndFish")

	assert.Equal(t, ctx{Bears: 1, Fishes: 1}, store.GetSnapshot().Context)
}

// JS: effect-only transitions should execute effects
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/store.test.ts#L500
func TestStoreStore_EffectOnlyTransitionsShouldExecuteEffects(t *testing.T) {
	spy := newSpy()
	store := xstore.CreateStore(xstore.StoreConfig[storeStore1Counter]{
		Context: storeStore1Counter{Count: 0},
		On: map[string]xstore.StoreAssigner[storeStore1Counter]{
			"justEffect": func(c storeStore1Counter, _ xs.Event, enq *xstore.EnqueueObject[storeStore1Counter]) (storeStore1Counter, bool) {
				enq.Effect(func(_ *xstore.StoreEffectEnqueue[storeStore1Counter]) { spy.Call() })
				return c, false
			},
		},
	})

	store.Trigger("justEffect")

	assert.Equal(t, 1, spy.Count())
}

// JS: emits-only transitions should emit events
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/store.test.ts#L516
func TestStoreStore_EmitsOnlyTransitionsShouldEmitEvents(t *testing.T) {
	spy := newSpy()
	store := xstore.CreateStore(xstore.StoreConfig[storeStore1Counter]{
		Context: storeStore1Counter{Count: 0},
		Schemas: &xstore.StoreSchemas{
			Emitted: map[string]xstore.Schema{
				"emitted": zObject(map[string]*zSchema{}),
			},
		},
		On: map[string]xstore.StoreAssigner[storeStore1Counter]{
			"justEmit": func(c storeStore1Counter, _ xs.Event, enq *xstore.EnqueueObject[storeStore1Counter]) (storeStore1Counter, bool) {
				enq.Emit("emitted")
				return c, false
			},
		},
	})

	store.On("emitted", func(e xs.Event) { spy.Call(e) })

	store.Trigger("justEmit")

	assert.Equal(t, 1, spy.Count())
}

// JS: checks whether events can transition
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/store.test.ts#L539
func TestStoreStore_ChecksWhetherEventsCanTransition(t *testing.T) {
	effectSpy := newSpy()
	emittedSpy := newSpy()
	store := xstore.CreateStore(xstore.StoreConfig[storeStore1Counter]{
		Context: storeStore1Counter{Count: 9},
		Schemas: &xstore.StoreSchemas{
			Events: map[string]xstore.Schema{
				"increment":   zObject(map[string]*zSchema{"by": zNumber()}),
				"noop":        zObject(map[string]*zSchema{}),
				"effectOnly":  zObject(map[string]*zSchema{}),
				"emitOnly":    zObject(map[string]*zSchema{}),
				"triggerOnly": zObject(map[string]*zSchema{}),
				"unavailable": zObject(map[string]*zSchema{}),
			},
			Emitted: map[string]xstore.Schema{
				"emitted": zObject(map[string]*zSchema{}),
			},
		},
		On: map[string]xstore.StoreAssigner[storeStore1Counter]{
			"increment": func(c storeStore1Counter, ev xs.Event, _ *xstore.EnqueueObject[storeStore1Counter]) (storeStore1Counter, bool) {
				by := storeStore1Payload(ev)["by"].(int)
				if c.Count+by > 10 {
					return c, false
				}

				return storeStore1Counter{Count: c.Count + by}, true
			},
			"noop": func(c storeStore1Counter, _ xs.Event, _ *xstore.EnqueueObject[storeStore1Counter]) (storeStore1Counter, bool) {
				return c, true
			},
			"effectOnly": func(c storeStore1Counter, _ xs.Event, enq *xstore.EnqueueObject[storeStore1Counter]) (storeStore1Counter, bool) {
				enq.Effect(func(_ *xstore.StoreEffectEnqueue[storeStore1Counter]) { effectSpy.Call() })
				return c, false
			},
			"emitOnly": func(c storeStore1Counter, _ xs.Event, enq *xstore.EnqueueObject[storeStore1Counter]) (storeStore1Counter, bool) {
				enq.Emit("emitted")
				return c, false
			},
			"triggerOnly": func(c storeStore1Counter, _ xs.Event, enq *xstore.EnqueueObject[storeStore1Counter]) (storeStore1Counter, bool) {
				enq.Trigger("increment", xs.E{"by": 1})
				return c, false
			},
		},
	})

	store.On("emitted", func(e xs.Event) { emittedSpy.Call(e) })

	assert.True(t, store.Can("increment", xs.E{"by": 1}))
	assert.False(t, store.Can("increment", xs.E{"by": 2}))
	assert.True(t, store.Can("noop"))
	assert.True(t, store.Can("effectOnly"))
	assert.True(t, store.Can("emitOnly"))
	assert.True(t, store.Can("triggerOnly"))
	assert.False(t, store.Can("unavailable"))
	assert.Equal(t, storeStore1Counter{Count: 9}, store.GetSnapshot().Context)
	assert.Equal(t, 0, effectSpy.Count())
	assert.Equal(t, 0, emittedSpy.Count())
}

// JS: checks whether Immer transitions can transition without changing context
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/store.test.ts#L592
func TestStoreStore_ChecksWhetherImmerTransitionsCanTransitionWithoutChangingContext(t *testing.T) {
	store := xstore.CreateStore(xstore.StoreConfig[storeStore1Counter]{
		Context: storeStore1Counter{Count: 10},
		On: map[string]xstore.StoreAssigner[storeStore1Counter]{
			"increment": func(c storeStore1Counter, ev xs.Event, _ *xstore.EnqueueObject[storeStore1Counter]) (storeStore1Counter, bool) {
				by := storeStore1Payload(ev)["by"].(int)
				if c.Count+by > 10 {
					return c, false
				}

				// JS: produce(ctx, (draft) => { draft.count += ev.by });
				// Go has no producer; the value copy is the equivalent.
				next := c
				next.Count += by
				return next, true
			},
		},
	})

	snapshot := store.GetSnapshot()

	assert.True(t, store.Can("increment", xs.E{"by": 0}))
	assert.Same(t, snapshot, store.GetSnapshot())
	assert.False(t, store.Can("increment", xs.E{"by": 1}))

	store.Trigger("increment", xs.E{"by": 0})
	assert.Equal(t, storeStore1Counter{Count: 10}, store.GetSnapshot().Context)

	store.Trigger("increment", xs.E{"by": 1})
	assert.Equal(t, storeStore1Counter{Count: 10}, store.GetSnapshot().Context)
}

// JS: wildcard listener receives all emitted events
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/store.test.ts#L621
func TestStoreStore_WildcardListenerReceivesAllEmittedEvents(t *testing.T) {
	spy := newSpy()
	store := xstore.CreateStore(xstore.StoreConfig[storeStore1Counter]{
		Context: storeStore1Counter{Count: 0},
		Schemas: &xstore.StoreSchemas{
			Emitted: map[string]xstore.Schema{
				"increased": zObject(map[string]*zSchema{"upBy": zNumber()}),
				"decreased": zObject(map[string]*zSchema{"downBy": zNumber()}),
			},
		},
		On: map[string]xstore.StoreAssigner[storeStore1Counter]{
			"inc": func(c storeStore1Counter, _ xs.Event, enq *xstore.EnqueueObject[storeStore1Counter]) (storeStore1Counter, bool) {
				enq.Emit("increased", xs.E{"upBy": 1})
				return storeStore1Counter{Count: c.Count + 1}, true
			},
			"dec": func(c storeStore1Counter, _ xs.Event, enq *xstore.EnqueueObject[storeStore1Counter]) (storeStore1Counter, bool) {
				enq.Emit("decreased", xs.E{"downBy": 1})
				return storeStore1Counter{Count: c.Count - 1}, true
			},
		},
	})

	store.On("*", func(e xs.Event) { spy.Call(e) })

	store.Trigger("inc")
	assert.Contains(t, spy.Calls(), []any{xs.E{"type": "increased", "upBy": 1}})

	store.Trigger("dec")
	assert.Contains(t, spy.Calls(), []any{xs.E{"type": "decreased", "downBy": 1}})

	assert.Equal(t, 2, spy.Count())
}

// JS: wildcard listener can be unsubscribed
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/store.test.ts#L654
func TestStoreStore_WildcardListenerCanBeUnsubscribed(t *testing.T) {
	spy := newSpy()
	store := xstore.CreateStore(xstore.StoreConfig[storeStore1Counter]{
		Context: storeStore1Counter{Count: 0},
		Schemas: &xstore.StoreSchemas{
			Emitted: map[string]xstore.Schema{
				"increased": zObject(map[string]*zSchema{"upBy": zNumber()}),
			},
		},
		On: map[string]xstore.StoreAssigner[storeStore1Counter]{
			"inc": func(c storeStore1Counter, _ xs.Event, enq *xstore.EnqueueObject[storeStore1Counter]) (storeStore1Counter, bool) {
				enq.Emit("increased", xs.E{"upBy": 1})
				return storeStore1Counter{Count: c.Count + 1}, true
			},
		},
	})

	sub := store.On("*", func(e xs.Event) { spy.Call(e) })
	store.Trigger("inc")
	assert.Equal(t, 1, spy.Count())

	sub.Unsubscribe()
	store.Trigger("inc")
	assert.Equal(t, 1, spy.Count())
}

// JS: wildcard listener is called after specific listener
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/store.test.ts#L680
func TestStoreStore_WildcardListenerIsCalledAfterSpecificListener(t *testing.T) {
	var order []string
	store := xstore.CreateStore(xstore.StoreConfig[storeStore1Counter]{
		Context: storeStore1Counter{Count: 0},
		Schemas: &xstore.StoreSchemas{
			Emitted: map[string]xstore.Schema{
				"increased": zObject(map[string]*zSchema{"upBy": zNumber()}),
			},
		},
		On: map[string]xstore.StoreAssigner[storeStore1Counter]{
			"inc": func(c storeStore1Counter, _ xs.Event, enq *xstore.EnqueueObject[storeStore1Counter]) (storeStore1Counter, bool) {
				enq.Emit("increased", xs.E{"upBy": 1})
				return storeStore1Counter{Count: c.Count + 1}, true
			},
		},
	})

	store.On("increased", func(_ xs.Event) { order = append(order, "specific") })
	store.On("*", func(_ xs.Event) { order = append(order, "wildcard") })

	store.Trigger("inc")

	assert.Equal(t, []string{"specific", "wildcard"}, order)
}

// JS: async effects can be enqueued
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/store.test.ts#L705
func TestStoreStore_AsyncEffectsCanBeEnqueued(t *testing.T) {
	done := newSignal()
	// gate replaces the fixed `setTimeout(5)`: the async part of the effect runs
	// only after the synchronous assertion below, independent of scheduler timing.
	gate := newSignal()
	defer gate.Resolve()
	var store *xstore.Store[storeStore1Counter]
	store = xstore.CreateStore(xstore.StoreConfig[storeStore1Counter]{
		Context: storeStore1Counter{Count: 0},
		On: map[string]xstore.StoreAssigner[storeStore1Counter]{
			"inc": func(c storeStore1Counter, _ xs.Event, enq *xstore.EnqueueObject[storeStore1Counter]) (storeStore1Counter, bool) {
				enq.Effect(func(_ *xstore.StoreEffectEnqueue[storeStore1Counter]) {
					go func() {
						<-gate.ch // JS: await setTimeout(5)
						store.Trigger("dec")
						done.Resolve()
					}()
				})

				return storeStore1Counter{Count: c.Count + 1}, true
			},
			"dec": func(c storeStore1Counter, _ xs.Event, _ *xstore.EnqueueObject[storeStore1Counter]) (storeStore1Counter, bool) {
				return storeStore1Counter{Count: c.Count - 1}, true
			},
		},
	})

	store.Trigger("inc")

	assert.Equal(t, 1, store.GetSnapshot().Context.Count)
	gate.Resolve()

	done.Wait(t) // JS: await setTimeout(10)

	assert.Equal(t, 0, store.GetSnapshot().Context.Count)
}

// JS: effects receive an enqueue object to trigger events (no closure needed)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/store.test.ts#L738
func TestStoreStore_EffectsReceiveAnEnqueueObjectToTriggerEventsNoClosureNeeded(t *testing.T) {
	type ctx struct {
		Count  int
		Status string
	}
	done := newSignal()
	gate := newSignal() // replaces the fixed `setTimeout(5)`; see "effects can be enqueued"
	defer gate.Resolve()
	logic := xstore.CreateStoreLogic(xstore.StoreConfig[ctx]{
		ContextFn: func(_ any) ctx { return ctx{Count: 0, Status: "idle"} },
		On: map[string]xstore.StoreAssigner[ctx]{
			"inc": func(c ctx, _ xs.Event, enq *xstore.EnqueueObject[ctx]) (ctx, bool) {
				enq.Effect(func(e *xstore.StoreEffectEnqueue[ctx]) {
					go func() {
						<-gate.ch // JS: await setTimeout(5)
						e.Trigger("done")
						done.Resolve()
					}()
				})
				return ctx{Count: c.Count, Status: "loading"}, true
			},
			"done": func(c ctx, _ xs.Event, _ *xstore.EnqueueObject[ctx]) (ctx, bool) {
				return ctx{Count: c.Count + 1, Status: "done"}, true
			},
		},
	})

	store := logic.CreateStore()

	store.Trigger("inc")
	assert.Equal(t, ctx{Count: 0, Status: "loading"}, store.GetSnapshot().Context)
	gate.Resolve()

	done.Wait(t) // JS: await setTimeout(10)
	assert.Equal(t, ctx{Count: 1, Status: "done"}, store.GetSnapshot().Context)
}

// JS: effects can read fresh state after awaiting via enq.getSnapshot()
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/store.test.ts#L762
func TestStoreStore_EffectsCanReadFreshStateAfterAwaitingViaEnqGetSnapshot(t *testing.T) {
	var seen []int
	done := newSignal()
	gate := newSignal() // replaces the fixed `setTimeout(5)`; see "effects can be enqueued"
	defer gate.Resolve()
	store := xstore.CreateStore(xstore.StoreConfig[storeStore1Counter]{
		Context: storeStore1Counter{Count: 0},
		On: map[string]xstore.StoreAssigner[storeStore1Counter]{
			"start": func(c storeStore1Counter, _ xs.Event, enq *xstore.EnqueueObject[storeStore1Counter]) (storeStore1Counter, bool) {
				enq.Effect(func(e *xstore.StoreEffectEnqueue[storeStore1Counter]) {
					go func() {
						// `ctx` is stale by now; GetSnapshot() reflects the bump below
						<-gate.ch // JS: await setTimeout(5)
						seen = append(seen, e.GetSnapshot().Context.Count)
						e.Trigger("done")
						done.Resolve()
					}()
				})
				return storeStore1Counter{Count: c.Count + 1}, true
			},
			"bump": func(c storeStore1Counter, _ xs.Event, _ *xstore.EnqueueObject[storeStore1Counter]) (storeStore1Counter, bool) {
				return storeStore1Counter{Count: c.Count + 10}, true
			},
			"done": func(c storeStore1Counter, _ xs.Event, _ *xstore.EnqueueObject[storeStore1Counter]) (storeStore1Counter, bool) {
				return c, true
			},
		},
	})

	store.Trigger("start") // count -> 1
	store.Trigger("bump")  // count -> 11 (after effect was enqueued, before it runs)
	gate.Resolve()

	done.Wait(t) // JS: await setTimeout(10)

	assert.Equal(t, []int{11}, seen)
}

// JS: sync effects read the current transition snapshot via enq.getSnapshot()
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/store.test.ts#L789
func TestStoreStore_SyncEffectsReadTheCurrentTransitionSnapshotViaEnqGetSnapshot(t *testing.T) {
	var seen []int
	store := xstore.CreateStore(xstore.StoreConfig[storeStore1Counter]{
		Context: storeStore1Counter{Count: 0},
		On: map[string]xstore.StoreAssigner[storeStore1Counter]{
			"start": func(c storeStore1Counter, _ xs.Event, enq *xstore.EnqueueObject[storeStore1Counter]) (storeStore1Counter, bool) {
				enq.Effect(func(e *xstore.StoreEffectEnqueue[storeStore1Counter]) {
					e.Trigger("bump")
				})
				enq.Effect(func(e *xstore.StoreEffectEnqueue[storeStore1Counter]) {
					seen = append(seen, e.GetSnapshot().Context.Count)
				})
				return storeStore1Counter{Count: c.Count + 1}, true
			},
			"bump": func(c storeStore1Counter, _ xs.Event, _ *xstore.EnqueueObject[storeStore1Counter]) (storeStore1Counter, bool) {
				return storeStore1Counter{Count: c.Count + 10}, true
			},
		},
	})

	store.Trigger("start")

	assert.Equal(t, []int{1}, seen)
	assert.Equal(t, 11, store.GetSnapshot().Context.Count)
}

// JS: effects can use enq.send to dispatch events
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/store.test.ts#L813
func TestStoreStore_EffectsCanUseEnqSendToDispatchEvents(t *testing.T) {
	store := xstore.CreateStore(xstore.StoreConfig[storeStore1Counter]{
		Context: storeStore1Counter{Count: 0},
		On: map[string]xstore.StoreAssigner[storeStore1Counter]{
			"inc": func(c storeStore1Counter, _ xs.Event, enq *xstore.EnqueueObject[storeStore1Counter]) (storeStore1Counter, bool) {
				enq.Effect(func(e *xstore.StoreEffectEnqueue[storeStore1Counter]) {
					e.Send(xs.Ev("dec"))
				})
				return storeStore1Counter{Count: c.Count + 1}, true
			},
			"dec": func(c storeStore1Counter, _ xs.Event, _ *xstore.EnqueueObject[storeStore1Counter]) (storeStore1Counter, bool) {
				return storeStore1Counter{Count: c.Count - 1}, true
			},
		},
	})

	store.Trigger("inc")

	sleep(5) // JS: await setTimeout(0)
	assert.Equal(t, 0, store.GetSnapshot().Context.Count)
}

// JS: rejects async handlers in createStoreTransition(...)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/store.test.ts#L833
func TestStoreStore_RejectsAsyncHandlersInCreateStoreTransition(t *testing.T) {
	store := xstore.CreateStore(xstore.StoreConfig[storeStore1Counter]{
		Context: storeStore1Counter{Count: 0},
		On:      map[string]xstore.StoreAssigner[storeStore1Counter]{},
	})

	assert.NotPanics(t, func() {
		store.Transition(store.GetSnapshot(), xs.Ev("bad"))
	})

	// JS second half: createStoreTransition({ bad: async (ctx) => ... }) throws
	// 'Async transition unsupported here'. Go handlers are synchronous, so an
	// async assigner has no Go form: that assertion is N/A-runtime and is not
	// ported (see manifest, "Partial ports").
	//
	// Additive (not a JS assertion): the synchronous counterpart of the dropped
	// half, so CreateStoreTransition stays exercised with a "bad" handler.
	syncTransition := xstore.CreateStoreTransition(map[string]xstore.StoreAssigner[storeStore1Counter]{
		"bad": storeStore1Inc,
	})
	assert.NotPanics(t, func() {
		result := syncTransition(
			&xstore.StoreSnapshot[storeStore1Counter]{Context: storeStore1Counter{Count: 0}, Status: xs.StatusActive},
			xs.Ev("bad"),
		)
		assert.Equal(t, storeStore1Counter{Count: 1}, result.Snapshot.Context)
	})
}

// JS: store.trigger > should allow triggering events with a fluent API
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/store.test.ts#L866
func TestStoreStore_Trigger_ShouldAllowTriggeringEventsWithAFluentApi(t *testing.T) {
	store := xstore.CreateStore(xstore.StoreConfig[storeStore1Counter]{
		Context: storeStore1Counter{Count: 0},
		On: map[string]xstore.StoreAssigner[storeStore1Counter]{
			"increment": func(c storeStore1Counter, ev xs.Event, _ *xstore.EnqueueObject[storeStore1Counter]) (storeStore1Counter, bool) {
				return storeStore1Counter{Count: c.Count + storeStore1Payload(ev)["by"].(int)}, true
			},
		},
	})

	store.Trigger("increment", xs.E{"by": 5})

	assert.Equal(t, 5, store.GetSnapshot().Context.Count)
}

// JS: store.trigger > should provide type safety for event payloads
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/store.test.ts#L881
func TestStoreStore_Trigger_ShouldProvideTypeSafetyForEventPayloads(t *testing.T) {
	store := xstore.CreateStore(xstore.StoreConfig[storeStore1Counter]{
		Context: storeStore1Counter{Count: 0},
		On: map[string]xstore.StoreAssigner[storeStore1Counter]{
			"increment": func(c storeStore1Counter, ev xs.Event, _ *xstore.EnqueueObject[storeStore1Counter]) (storeStore1Counter, bool) {
				return storeStore1Counter{Count: c.Count + storeStore1Payload(ev)["by"].(int)}, true
			},
			"reset": func(_ storeStore1Counter, _ xs.Event, _ *xstore.EnqueueObject[storeStore1Counter]) (storeStore1Counter, bool) {
				return storeStore1Counter{Count: 0}, true
			},
		},
	})

	// JS: the `if (false) { @ts-expect-error ... }` block never runs (type-level only).

	// Valid usage with no payload
	store.Trigger("reset")

	// Valid usage with payload
	store.Trigger("increment", xs.E{"by": 1})
}

// JS: store.trigger > should be equivalent to store.send
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/store.test.ts#L910
func TestStoreStore_Trigger_ShouldBeEquivalentToStoreSend(t *testing.T) {
	// JS: vi.spyOn(store, 'send'). Go methods cannot be spied on, so equivalence
	// is asserted observationally: Trigger("increment", {by: 5}) must leave a
	// store in exactly the state, with exactly the inspected event sequence, that
	// Send({type: "increment", by: 5}) produces on an identical store.
	sendEvent := xs.Event(xs.E{"type": "increment", "by": 5})
	newCounter := func() *xstore.Store[storeStore1Counter] {
		return xstore.CreateStore(xstore.StoreConfig[storeStore1Counter]{
			Context: storeStore1Counter{Count: 0},
			On: map[string]xstore.StoreAssigner[storeStore1Counter]{
				"increment": func(c storeStore1Counter, ev xs.Event, _ *xstore.EnqueueObject[storeStore1Counter]) (storeStore1Counter, bool) {
					return storeStore1Counter{Count: c.Count + storeStore1Payload(ev)["by"].(int)}, true
				},
			},
		})
	}
	inspected := func(s *xstore.Store[storeStore1Counter]) *[]xs.Event {
		var evs []xs.Event
		s.Inspect(func(e xstore.StoreInspectionEvent) { evs = append(evs, e.Event) })
		return &evs
	}

	triggered := newCounter()
	triggeredEvents := inspected(triggered)
	triggered.Trigger("increment", xs.E{"by": 5})

	sent := newCounter()
	sentEvents := inspected(sent)
	sent.Send(sendEvent)

	assert.Equal(t, []xs.Event{xs.Ev("@xstate.init"), sendEvent}, *triggeredEvents)
	assert.Equal(t, *sentEvents, *triggeredEvents)
	assert.Equal(t, sent.GetSnapshot().Context, triggered.GetSnapshot().Context)
	assert.Equal(t, 5, triggered.GetSnapshot().Context.Count)
}

// JS: store.trigger > should fail fast for unknown trigger names on config-based stores
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/store.test.ts#L930
func TestStoreStore_Trigger_ShouldFailFastForUnknownTriggerNamesOnConfigBasedStores(t *testing.T) {
	store := xstore.CreateStore(xstore.StoreConfig[storeStore1Counter]{
		Context: storeStore1Counter{Count: 0},
		On: map[string]xstore.StoreAssigner[storeStore1Counter]{
			"increment": func(c storeStore1Counter, ev xs.Event, _ *xstore.EnqueueObject[storeStore1Counter]) (storeStore1Counter, bool) {
				return storeStore1Counter{Count: c.Count + storeStore1Payload(ev)["by"].(int)}, true
			},
			"reset": func(_ storeStore1Counter, _ xs.Event, _ *xstore.EnqueueObject[storeStore1Counter]) (storeStore1Counter, bool) {
				return storeStore1Counter{Count: 0}, true
			},
		},
	})

	assert.Equal(t, []string{"increment", "reset"}, store.EventTypes())
	// JS: toThrow(TypeError). Go has no TypeError; the contract pinned here is a
	// real panic (not the "not implemented" stub panic) whose message names the
	// unknown trigger.
	msg := panicMessage(func() { store.Trigger("unknown") })
	assert.NotEmpty(t, msg)
	assert.NotContains(t, msg, "not implemented")
	assert.Contains(t, msg, "unknown")
	assert.Equal(t, 0, store.GetSnapshot().Context.Count)
}

// JS: store.trigger > should include extension events in the concrete trigger object
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/store.test.ts#L946
func TestStoreStore_Trigger_ShouldIncludeExtensionEventsInTheConcreteTriggerObject(t *testing.T) {
	store := xstore.CreateStore(xstore.StoreConfig[storeStore1Counter]{
		Context: storeStore1Counter{Count: 0},
		On: map[string]xstore.StoreAssigner[storeStore1Counter]{
			"increment": func(c storeStore1Counter, ev xs.Event, _ *xstore.EnqueueObject[storeStore1Counter]) (storeStore1Counter, bool) {
				return storeStore1Counter{Count: c.Count + storeStore1Payload(ev)["by"].(int)}, true
			},
		},
	}).With(xstore.Reset[storeStore1Counter]())

	assert.Equal(t, []string{"increment", "reset"}, store.EventTypes())

	store.Trigger("increment", xs.E{"by": 2})
	store.Trigger("reset")

	assert.Equal(t, 0, store.GetSnapshot().Context.Count)
}

// JS: store.trigger > should include schema-declared events in the concrete trigger object
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/store.test.ts#L964
func TestStoreStore_Trigger_ShouldIncludeSchemaDeclaredEventsInTheConcreteTriggerObject(t *testing.T) {
	store := xstore.CreateStore(xstore.StoreConfig[storeStore1Counter]{
		Schemas: &xstore.StoreSchemas{
			Events: map[string]xstore.Schema{
				"increment": zObject(map[string]*zSchema{"by": zNumber()}),
				"reset":     zObject(map[string]*zSchema{}),
			},
		},
		Context: storeStore1Counter{Count: 0},
		On: map[string]xstore.StoreAssigner[storeStore1Counter]{
			"increment": func(c storeStore1Counter, ev xs.Event, _ *xstore.EnqueueObject[storeStore1Counter]) (storeStore1Counter, bool) {
				return storeStore1Counter{Count: c.Count + storeStore1Payload(ev)["by"].(int)}, true
			},
		},
	})

	assert.Equal(t, []string{"increment", "reset"}, store.EventTypes())

	store.Trigger("increment", xs.E{"by": 2})
	store.Trigger("reset")
	store.Trigger("reset", xs.E{})

	assert.Equal(t, 2, store.GetSnapshot().Context.Count)
}

// JS: works with typestates
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/store.test.ts#L990
func TestStoreStore_WorksWithTypestates(t *testing.T) {
	t.Skip("N/A: type-level only — discriminated-union context narrowing checked with `satisfies` and @ts-expect-error")
}

// JS: the emit type is not overridden by the payload
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/store.test.ts#L1035
func TestStoreStore_TheEmitTypeIsNotOverriddenByThePayload(t *testing.T) {
	type drawer struct {
		ID string `json:"id"`
	}
	type ctx struct {
		Drawer *drawer
	}
	spy := newSpy()

	drawersBridgeStore := xstore.CreateStore(xstore.StoreConfig[ctx]{
		Schemas: &xstore.StoreSchemas{
			Emitted: map[string]xstore.Schema{
				"drawerOpened": zObject(map[string]*zSchema{
					"drawer": zObject(map[string]*zSchema{"id": zString()}),
				}),
			},
		},
		Context: ctx{Drawer: nil},
		On: map[string]xstore.StoreAssigner[ctx]{
			"openDrawer": func(c ctx, ev xs.Event, enq *xstore.EnqueueObject[ctx]) (ctx, bool) {
				// The whole event (type "openDrawer") is the payload; the emitted type must win.
				enq.Emit("drawerOpened", ev.(xs.E))

				return ctx{Drawer: ev.(xs.E)["drawer"].(*drawer)}, true
			},
		},
	})

	drawersBridgeStore.On("drawerOpened", func(event xs.Event) {
		// expect to be called here
		spy.Call(event)
	})

	d := &drawer{ID: "a"}
	drawersBridgeStore.Trigger("openDrawer", xs.E{"drawer": d})

	assert.Contains(t, spy.Calls(), []any{xs.E{"type": "drawerOpened", "drawer": d}})
}

// JS: store.transition > returns next state and effects for a given state and event
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/store.test.ts#L1084
func TestStoreStore_Transition_ReturnsNextStateAndEffectsForAGivenStateAndEvent(t *testing.T) {
	store := xstore.CreateStore(xstore.StoreConfig[storeStore1Counter]{
		Context: storeStore1Counter{Count: 0},
		Schemas: &xstore.StoreSchemas{
			Emitted: map[string]xstore.Schema{
				"increased": zObject(map[string]*zSchema{"by": zNumber()}),
				"nothing":   zObject(map[string]*zSchema{}),
			},
		},
		On: map[string]xstore.StoreAssigner[storeStore1Counter]{
			"inc": func(c storeStore1Counter, ev xs.Event, enq *xstore.EnqueueObject[storeStore1Counter]) (storeStore1Counter, bool) {
				by := storeStore1Payload(ev)["by"].(int)
				enq.Emit("increased", xs.E{"by": by})
				return storeStore1Counter{Count: c.Count + by}, true
			},
		},
	})

	nextState, effects := store.Transition(store.GetSnapshot(), xs.E{"type": "inc", "by": 2})

	assert.Equal(t, storeStore1Counter{Count: 2}, nextState.Context)
	require.Len(t, effects, 1)
	assert.Equal(t, xs.Event(xs.E{"type": "increased", "by": 2}), effects[0].Emitted)
}

// JS: store.transition > returns unchanged state and empty effects for unknown events
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/store.test.ts#L1113
func TestStoreStore_Transition_ReturnsUnchangedStateAndEmptyEffectsForUnknownEvents(t *testing.T) {
	store := xstore.CreateStore(xstore.StoreConfig[storeStore1Counter]{
		Context: storeStore1Counter{Count: 0},
		On: map[string]xstore.StoreAssigner[storeStore1Counter]{
			"inc": storeStore1Inc,
		},
	})

	currentState := store.GetSnapshot()
	nextState, effects := store.Transition(currentState, xs.Ev("unknown"))

	assert.Same(t, currentState, nextState)
	assert.Empty(t, effects)
}

// JS: store.transition > collects enqueued effects
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/store.test.ts#L1133
func TestStoreStore_Transition_CollectsEnqueuedEffects(t *testing.T) {
	store := xstore.CreateStore(xstore.StoreConfig[storeStore1Counter]{
		Context: storeStore1Counter{Count: 0},
		On: map[string]xstore.StoreAssigner[storeStore1Counter]{
			"inc": func(c storeStore1Counter, _ xs.Event, enq *xstore.EnqueueObject[storeStore1Counter]) (storeStore1Counter, bool) {
				enq.Effect(func(_ *xstore.StoreEffectEnqueue[storeStore1Counter]) {
					// This effect function would normally do something
				})
				return storeStore1Counter{Count: c.Count + 1}, true
			},
		},
	})

	nextState, effects := store.Transition(store.GetSnapshot(), xs.Ev("inc"))

	assert.Equal(t, storeStore1Counter{Count: 1}, nextState.Context)
	require.Len(t, effects, 1)
	assert.NotNil(t, effects[0].Run) // typeof effects[0] === 'function'
}

// JS: store.transition > resolves enqueued trigger events and collects effects in pure transitions
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/store.test.ts#L1157
func TestStoreStore_Transition_ResolvesEnqueuedTriggerEventsAndCollectsEffectsInPureTransitions(t *testing.T) {
	spy := newSpy()
	store := xstore.CreateStore(xstore.StoreConfig[storeStore1Counter]{
		Context: storeStore1Counter{Count: 0},
		Schemas: &xstore.StoreSchemas{
			Events: map[string]xstore.Schema{
				"inc":      zObject(map[string]*zSchema{}),
				"incTwice": zObject(map[string]*zSchema{}),
			},
		},
		On: map[string]xstore.StoreAssigner[storeStore1Counter]{
			"inc": func(c storeStore1Counter, _ xs.Event, enq *xstore.EnqueueObject[storeStore1Counter]) (storeStore1Counter, bool) {
				enq.Effect(func(_ *xstore.StoreEffectEnqueue[storeStore1Counter]) { spy.Call("inc") })

				return storeStore1Counter{Count: c.Count + 1}, true
			},
			"incTwice": func(c storeStore1Counter, _ xs.Event, enq *xstore.EnqueueObject[storeStore1Counter]) (storeStore1Counter, bool) {
				enq.Effect(func(_ *xstore.StoreEffectEnqueue[storeStore1Counter]) { spy.Call("before") })
				enq.Trigger("inc")
				enq.Trigger("inc")
				enq.Effect(func(_ *xstore.StoreEffectEnqueue[storeStore1Counter]) { spy.Call("after") })

				return c, true
			},
		},
	})

	nextState, effects := store.Transition(store.GetSnapshot(), xs.Ev("incTwice"))

	assert.Equal(t, storeStore1Counter{Count: 2}, nextState.Context)
	require.Len(t, effects, 4)
	for _, effect := range effects {
		assert.NotNil(t, effect.Run) // typeof effect === 'function'
	}
	for _, effect := range effects {
		if effect.Run != nil {
			effect.Run(nil)
		}
	}
	require.Equal(t, 4, spy.Count())
	assert.Equal(t, []any{"before"}, spy.Calls()[0])
	assert.Equal(t, []any{"after"}, spy.Calls()[1])
	assert.Equal(t, []any{"inc"}, spy.Calls()[2])
	assert.Equal(t, []any{"inc"}, spy.Calls()[3])
	spy.Reset()

	store.Trigger("incTwice")

	assert.Equal(t, storeStore1Counter{Count: 2}, store.GetSnapshot().Context)
	require.Equal(t, 4, spy.Count())
	assert.Equal(t, []any{"before"}, spy.Calls()[0])
	assert.Equal(t, []any{"after"}, spy.Calls()[1])
	assert.Equal(t, []any{"inc"}, spy.Calls()[2])
	assert.Equal(t, []any{"inc"}, spy.Calls()[3])
}

// JS: can be created with a logic object
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/store.test.ts#L1216
func TestStoreStore_CanBeCreatedWithALogicObject(t *testing.T) {
	store := xstore.CreateStoreFromLogic(xstore.StoreLogic[storeStore1Counter]{
		GetInitialSnapshot: func() *xstore.StoreSnapshot[storeStore1Counter] {
			return &xstore.StoreSnapshot[storeStore1Counter]{
				Context: storeStore1Counter{Count: 0},
				Status:  xs.StatusActive,
			}
		},
		Transition: func(snapshot *xstore.StoreSnapshot[storeStore1Counter], event xs.Event) xstore.StoreTransitionResult[storeStore1Counter] {
			if event.EventType() == "inc" {
				next := *snapshot
				next.Context = storeStore1Counter{Count: snapshot.Context.Count + 1}
				return xstore.StoreTransitionResult[storeStore1Counter]{Snapshot: &next, Effects: []xstore.StoreEffect[storeStore1Counter]{}}
			}
			return xstore.StoreTransitionResult[storeStore1Counter]{Snapshot: snapshot, Effects: []xstore.StoreEffect[storeStore1Counter]{}}
		},
	})

	assert.Equal(t, storeStore1Counter{Count: 0}, store.GetSnapshot().Context)

	store.Trigger("inc")

	assert.Equal(t, storeStore1Counter{Count: 1}, store.GetSnapshot().Context)

	// JS: @ts-expect-error store.trigger.unknown() is still callable at runtime
	// (a logic-object store accepts any event type).
	store.Trigger("unknown")

	// JS: `count satisfies number` / `@ts-expect-error ... satisfies string` are type-level only.
}

// JS: can select from a store
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/store.test.ts#L1255
func TestStoreStore_CanSelectFromAStore(t *testing.T) {
	store := xstore.CreateStore(xstore.StoreConfig[storeStore1Counter]{
		Context: storeStore1Counter{Count: 0},
		On: map[string]xstore.StoreAssigner[storeStore1Counter]{
			"inc": storeStore1Inc,
		},
	})

	countSpy := newSpy()
	evenSpy := newSpy()
	count := xstore.Select(store, func(context storeStore1Counter) int { return context.Count })
	isEven := xstore.Select(store, func(context storeStore1Counter) bool { return context.Count%2 == 0 })

	count.SubscribeNext(func(v int) { countSpy.Call(v) })
	isEven.SubscribeNext(func(v bool) { evenSpy.Call(v) })

	assert.Equal(t, 0, count.Get())
	assert.Equal(t, true, isEven.Get())

	store.Trigger("inc")

	assert.Equal(t, 1, count.Get())
	assert.Equal(t, false, isEven.Get())
	assert.Contains(t, countSpy.Calls(), []any{1})
	assert.Contains(t, evenSpy.Calls(), []any{false})
}

// JS: can create reusable store logic with selectors
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/store.test.ts#L1284
func TestStoreStore_CanCreateReusableStoreLogicWithSelectors(t *testing.T) {
	type input struct{ InitialCount int }
	counterLogic := xstore.CreateStoreLogic(xstore.StoreConfig[storeStore1Counter]{
		ContextFn: func(in any) storeStore1Counter {
			return storeStore1Counter{Count: in.(input).InitialCount}
		},
		Selectors: map[string]func(storeStore1Counter) any{
			"count": func(context storeStore1Counter) any { return context.Count },
			"doubled": func(context storeStore1Counter) any {
				// JS: `context.missing` (@ts-expect-error) is a type-level check only.
				return context.Count * 2
			},
		},
		On: map[string]xstore.StoreAssigner[storeStore1Counter]{
			"inc": func(context storeStore1Counter, _ xs.Event, _ *xstore.EnqueueObject[storeStore1Counter]) (storeStore1Counter, bool) {
				return storeStore1Counter{Count: context.Count + 1}, true
			},
		},
	})

	store := counterLogic.CreateStore(input{InitialCount: 2})

	assert.Equal(t, 2, store.Selectors()["count"].Get())
	assert.Equal(t, 4, store.Selectors()["doubled"].Get())

	store.Trigger("inc")

	assert.Equal(t, 3, store.Selectors()["count"].Get())
	assert.Equal(t, 6, store.Selectors()["doubled"].Get())
}

// JS: preserves selectors through store extensions
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/store.test.ts#L1317
func TestStoreStore_PreservesSelectorsThroughStoreExtensions(t *testing.T) {
	counterLogic := xstore.CreateStoreLogic(xstore.StoreConfig[storeStore1Counter]{
		Context: storeStore1Counter{Count: 0},
		Selectors: map[string]func(storeStore1Counter) any{
			"doubled": func(context storeStore1Counter) any { return context.Count * 2 },
		},
		On: map[string]xstore.StoreAssigner[storeStore1Counter]{
			"inc": storeStore1Inc,
		},
	})

	store := counterLogic.CreateStore().With(xstore.Reset[storeStore1Counter]())

	assert.Equal(t, 0, store.Selectors()["doubled"].Get())

	store.Trigger("inc")
	assert.Equal(t, 2, store.Selectors()["doubled"].Get())

	store.Trigger("reset")
	assert.Equal(t, 0, store.Selectors()["doubled"].Get())
}

// JS: should not trigger update if the snapshot is the same
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/store.test.ts#L1341
func TestStoreStore_ShouldNotTriggerUpdateIfTheSnapshotIsTheSame(t *testing.T) {
	store := xstore.CreateStore(xstore.StoreConfig[storeStore1Counter]{
		Context: storeStore1Counter{Count: 0},
		On: map[string]xstore.StoreAssigner[storeStore1Counter]{
			"doNothing": func(c storeStore1Counter, _ xs.Event, _ *xstore.EnqueueObject[storeStore1Counter]) (storeStore1Counter, bool) {
				// JS: (ctx) => ctx. The handler hands back the identical context, so the
				// store must keep the current snapshot (JS `nextContext !== currentContext`
				// is false) and notify nobody. Requires the implementation to treat a
				// returned context equal to the current one as unchanged; see manifest
				// "Snapshot identity".
				return c, true
			},
		},
	})

	spy := newSpy()
	store.SubscribeNext(func(s *xstore.StoreSnapshot[storeStore1Counter]) { spy.Call(s) })

	store.Trigger("doNothing")
	store.Trigger("doNothing")

	assert.Equal(t, 0, spy.Count())
}

// JS: should not trigger update if the snapshot is the same even if there are effects
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/store.test.ts#L1358
func TestStoreStore_ShouldNotTriggerUpdateIfTheSnapshotIsTheSameEvenIfThereAreEffects(t *testing.T) {
	store := xstore.CreateStore(xstore.StoreConfig[storeStore1Counter]{
		Context: storeStore1Counter{Count: 0},
		On: map[string]xstore.StoreAssigner[storeStore1Counter]{
			"doNothing": func(c storeStore1Counter, _ xs.Event, enq *xstore.EnqueueObject[storeStore1Counter]) (storeStore1Counter, bool) {
				enq.Effect(func(_ *xstore.StoreEffectEnqueue[storeStore1Counter]) {
					// …
				})
				// JS: return ctx (same requirement as the test above: an unchanged
				// context keeps the snapshot even though an effect was enqueued).
				return c, true
			},
		},
	})

	spy := newSpy()
	store.SubscribeNext(func(s *xstore.StoreSnapshot[storeStore1Counter]) { spy.Call(s) })

	store.Trigger("doNothing")
	store.Trigger("doNothing")

	assert.Equal(t, 0, spy.Count())
}

// JS: types > AnyStoreConfig
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/store.test.ts#L1381
func TestStoreStore_Types_AnyStoreConfig(t *testing.T) {
	t.Skip("N/A: type-level only — AnyStoreConfig accepts a config; `transformStoreConfig({})` is a @ts-expect-error")
}

// JS: types > EventFromStoreConfig
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/store.test.ts#L1395
func TestStoreStore_Types_EventFromStoreConfig(t *testing.T) {
	t.Skip("N/A: type-level only — EventFromStoreConfig inferred via `satisfies`")
}

// JS: types > ContextFromStoreConfig
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/store.test.ts#L1417
func TestStoreStore_Types_ContextFromStoreConfig(t *testing.T) {
	t.Skip("N/A: type-level only — ContextFromStoreConfig inferred via `satisfies`")
}

// JS: types > generics can be provided
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/store.test.ts#L1435
func TestStoreStore_Types_GenericsCanBeProvided(t *testing.T) {
	type ctx struct {
		CoffeeBeans int
		Water       int
	}
	store := xstore.CreateStore(xstore.StoreConfig[ctx]{
		Context: ctx{CoffeeBeans: 0, Water: 0},
		On: map[string]xstore.StoreAssigner[ctx]{
			"addWater": func(c ctx, ev xs.Event, _ *xstore.EnqueueObject[ctx]) (ctx, bool) {
				return ctx{CoffeeBeans: c.CoffeeBeans, Water: c.Water + ev.(xs.E)["amount"].(int)}, true
			},
			"grindBeans": func(c ctx, _ xs.Event, enq *xstore.EnqueueObject[ctx]) (ctx, bool) {
				enq.Emit("brewing")

				enq.Emit("beansGround", xs.E{"amount": 1})

				// JS: @ts-expect-error (missing payload; still runs)
				enq.Emit("beansGround")

				enq.Emit("brewing", xs.E{})

				return ctx{CoffeeBeans: c.CoffeeBeans + 1, Water: c.Water}, true
			},
		},
	})

	store.Trigger("addWater", xs.E{"amount": 1})

	store.Trigger("grindBeans")

	// JS: the `if (false) { store.trigger.unknown() }` block never runs (type-level only).
}

// JS: types > localizes TypeScript errors to the specific transition
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/store.test.ts#L1492
func TestStoreStore_Types_LocalizesTypeScriptErrorsToTheSpecificTransition(t *testing.T) {
	t.Skip("N/A: type-level only — the only check is a @ts-expect-error on a mistyped transition")
}

// JS: emitted events work with store extensions
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/store.test.ts#L1508
func TestStoreStore_EmittedEventsWorkWithStoreExtensions(t *testing.T) {
	store := xstore.CreateStore(xstore.StoreConfig[storeStore1Counter]{
		Context: storeStore1Counter{Count: 0},
		Schemas: &xstore.StoreSchemas{
			Emitted: map[string]xstore.Schema{
				"increased": zObject(map[string]*zSchema{"upBy": zNumber()}),
			},
		},
		On: map[string]xstore.StoreAssigner[storeStore1Counter]{
			"inc": func(c storeStore1Counter, _ xs.Event, enq *xstore.EnqueueObject[storeStore1Counter]) (storeStore1Counter, bool) {
				enq.Emit("increased", xs.E{"upBy": 1})
				return storeStore1Counter{Count: c.Count + 1}, true
			},
		},
	}).With(xstore.Reset[storeStore1Counter]())

	spy := newSpy()

	store.On("increased", func(e xs.Event) { spy.Call(e) })

	store.Trigger("inc")

	assert.Contains(t, spy.Calls(), []any{xs.E{"type": "increased", "upBy": 1}})
}
