package store_test

import (
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	xstore "github.com/nguyenvanduocit/go-xstate/store"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// storePersistClear1Ctx mirrors the JS context `{ count: number }`.
type storePersistClear1Ctx struct {
	Count int `json:"count"`
}

// storePersistClear1Opts mirrors the options bag of the JS createCounter plus
// the explicit `strategy` used by the tests that build the store inline.
type storePersistClear1Opts struct {
	Strategy  xstore.PersistStrategy
	Throttle  time.Duration
	MaxEvents int
	Clock     xs.Clock
}

// storePersistClear1NewStore mirrors the inline
// createStore({context: {count: 0}, on: {add}}).with(persist({name: 'counter', storage, strategy})).
func storePersistClear1NewStore(storage xstore.Storage, o storePersistClear1Opts) *xstore.Store[storePersistClear1Ctx] {
	return xstore.CreateStore(xstore.StoreConfig[storePersistClear1Ctx]{
		Context: storePersistClear1Ctx{Count: 0},
		On: map[string]xstore.StoreAssigner[storePersistClear1Ctx]{
			"add": func(c storePersistClear1Ctx, ev xs.Event, _ *xstore.EnqueueObject[storePersistClear1Ctx]) (storePersistClear1Ctx, bool) {
				return storePersistClear1Ctx{Count: c.Count + ev.(xs.E)["amount"].(int)}, true
			},
		},
	}).With(xstore.Persist(xstore.PersistOptions[storePersistClear1Ctx]{
		Name:      "counter",
		Storage:   storage,
		Strategy:  o.Strategy,
		Throttle:  o.Throttle,
		MaxEvents: o.MaxEvents,
		Clock:     o.Clock,
	}))
}

// storePersistClear1NewCounter mirrors createCounter(storage, options): an
// event-strategy persisted counter.
func storePersistClear1NewCounter(storage xstore.Storage, o ...storePersistClear1Opts) *xstore.Store[storePersistClear1Ctx] {
	var opts storePersistClear1Opts
	if len(o) > 0 {
		opts = o[0]
	}
	opts.Strategy = xstore.PersistEvent
	return storePersistClear1NewStore(storage, opts)
}

// storePersistClear1Add mirrors store.trigger.add({ amount }).
func storePersistClear1Add(s *xstore.Store[storePersistClear1Ctx], amount int) {
	s.Trigger("add", xs.E{"amount": amount})
}

// storePersistClear1Log is the parsed event-strategy storage value
// `{ events: [{type: 'add', amount}...], checkpoint: {count}, version: 0 }`.
func storePersistClear1Log(checkpoint int, amounts ...int) map[string]any {
	events := []any{}
	for _, a := range amounts {
		events = append(events, map[string]any{"type": "add", "amount": float64(a)})
	}
	return map[string]any{
		"events":     events,
		"checkpoint": map[string]any{"count": float64(checkpoint)},
		"version":    float64(0),
	}
}

// storePersistClear1Snap is the parsed snapshot-strategy storage value
// `{ context: { count }, version: 0 }`.
func storePersistClear1Snap(count int) map[string]any {
	return map[string]any{
		"context": map[string]any{"count": float64(count)},
		"version": float64(0),
	}
}

// JS: clearStorage event history > starts a new event log from the live context (throttle %i)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persistClear.test.ts#L43
func TestStorePersistClear_StartsNewEventLogFromLiveContext(t *testing.T) {
	for _, throttle := range []int{0, 100} {
		// JS test (shared case definition): https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persistClear.test.ts#L43
		t.Run(fmt.Sprintf("throttle %d", throttle), func(t *testing.T) {
			clock := xs.NewSimulatedClock()
			storage := newMemoryStorage()
			store := storePersistClear1NewCounter(storage, storePersistClear1Opts{Throttle: ms(throttle), Clock: clock})
			storePersistClear1Add(store, 2)
			storePersistClear1Add(store, 5)
			// Object.freeze has no Go equivalent; the snapshot pointer is kept
			// and checked for identity and content below.
			beforeClear := store.GetSnapshot()

			// clearStorage only requires getSnapshot, and must not mutate snapshots.
			assert.Nil(t, xstore.ClearStorage(store))
			assert.Same(t, beforeClear, store.GetSnapshot())
			assert.Equal(t, storePersistClear1Ctx{Count: 7}, store.GetSnapshot().Context)
			clock.Increment(ms(100))
			assert.Nil(t, getItem(t, storage, "counter"))

			storePersistClear1Add(store, 3)
			xstore.FlushStorage(store)
			assert.Equal(t, storePersistClear1Log(7, 3), storedJSON(t, storage, "counter"))
			assert.Equal(t, storePersistClear1Ctx{Count: 7}, beforeClear.Context)
			assert.Equal(t, storePersistClear1Ctx{Count: 10}, storePersistClear1NewCounter(storage).GetSnapshot().Context)
		})
	}
}

// JS: clearStorage event history > replaces an existing checkpoint and still truncates the new log correctly
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persistClear.test.ts#L75
func TestStorePersistClear_ReplacesExistingCheckpointAndTruncatesNewLog(t *testing.T) {
	storage := newMemoryStorage()
	store := storePersistClear1NewCounter(storage, storePersistClear1Opts{MaxEvents: 2})
	for _, amount := range []int{2, 3, 5} {
		storePersistClear1Add(store, amount)
	}
	assert.Equal(t,
		map[string]any{"count": float64(2)},
		storedJSON(t, storage, "counter").(map[string]any)["checkpoint"],
	)

	xstore.ClearStorage(store)
	storePersistClear1Add(store, 7)
	assert.Equal(t, storePersistClear1Log(10, 7), storedJSON(t, storage, "counter"))
	for _, amount := range []int{11, 13} {
		storePersistClear1Add(store, amount)
	}

	assert.Equal(t, storePersistClear1Log(17, 11, 13), storedJSON(t, storage, "counter"))
	assert.Equal(t, storePersistClear1Ctx{Count: 41}, storePersistClear1NewCounter(storage).GetSnapshot().Context)
}

// JS: clearStorage event history > does not consume the reset during %s evaluations
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persistClear.test.ts#L109
func TestStorePersistClear_DoesNotConsumeResetDuringEvaluations(t *testing.T) {
	for _, evaluation := range []string{"can", "transition"} {
		// JS test (shared case definition): https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persistClear.test.ts#L109
		t.Run(evaluation, func(t *testing.T) {
			storage := newMemoryStorage()
			store := storePersistClear1NewCounter(storage)
			storePersistClear1Add(store, 2)
			xstore.ClearStorage(store)

			if evaluation == "can" {
				assert.True(t, store.Can("add", xs.E{"amount": 20}))
			} else {
				store.Transition(store.GetSnapshot(), xs.E{"type": "add", "amount": 30})
			}
			xstore.FlushStorage(store)
			assert.Nil(t, getItem(t, storage, "counter"))
			assert.Equal(t, storePersistClear1Ctx{Count: 2}, store.GetSnapshot().Context)

			storePersistClear1Add(store, 3)
			assert.Equal(t, storePersistClear1Log(2, 3), storedJSON(t, storage, "counter"))
		})
	}
}

// JS: clearStorage event history > keeps history cleared when rehydrating empty storage
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persistClear.test.ts#L135
func TestStorePersistClear_KeepsHistoryClearedWhenRehydratingEmptyStorage(t *testing.T) {
	storage := newMemoryStorage()
	store := storePersistClear1NewCounter(storage)
	storePersistClear1Add(store, 2)
	xstore.ClearStorage(store)
	_, err := xstore.RehydrateStore(store).Wait()
	require.NoError(t, err)
	storePersistClear1Add(store, 3)

	assert.Equal(t, storePersistClear1Log(2, 3), storedJSON(t, storage, "counter"))
}

// JS: clearStorage event history > keeps newly hydrated history after clearing an earlier log
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persistClear.test.ts#L150
func TestStorePersistClear_KeepsNewlyHydratedHistoryAfterClearingEarlierLog(t *testing.T) {
	storage := newMemoryStorage()
	store := storePersistClear1NewCounter(storage)
	storePersistClear1Add(store, 2)
	xstore.ClearStorage(store)
	storage.SetItem("counter", toJSON(t, storePersistClear1Log(10, 4)))

	_, err := xstore.RehydrateStore(store).Wait()
	require.NoError(t, err)
	storePersistClear1Add(store, 3)
	assert.Equal(t, storePersistClear1Log(10, 4, 3), storedJSON(t, storage, "counter"))
	assert.Equal(t, storePersistClear1Ctx{Count: 17}, storePersistClear1NewCounter(storage).GetSnapshot().Context)
}

// JS: clearStorage event history > starts a fresh log after each clear, including repeated clears without an event
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persistClear.test.ts#L177
func TestStorePersistClear_StartsFreshLogAfterEachClearIncludingRepeatedClears(t *testing.T) {
	storage := newMemoryStorage()
	store := storePersistClear1NewCounter(storage)
	storePersistClear1Add(store, 2)
	xstore.ClearStorage(store)
	xstore.ClearStorage(store)
	storePersistClear1Add(store, 3)
	xstore.ClearStorage(store)
	storePersistClear1Add(store, 4)

	assert.Equal(t, storePersistClear1Log(5, 4), storedJSON(t, storage, "counter"))
	assert.Equal(t, storePersistClear1Ctx{Count: 9}, storePersistClear1NewCounter(storage).GetSnapshot().Context)
}

// JS: clearStorage event history > invalidates %s persistence effects computed before clearing
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persistClear.test.ts#L195
func TestStorePersistClear_InvalidatesPersistenceEffectsComputedBeforeClearing(t *testing.T) {
	for _, strategy := range []xstore.PersistStrategy{xstore.PersistSnapshot, xstore.PersistEvent} {
		// JS test (shared case definition): https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persistClear.test.ts#L195
		t.Run(string(strategy), func(t *testing.T) {
			storage := newMemoryStorage()
			store := storePersistClear1NewStore(storage, storePersistClear1Opts{Strategy: strategy})
			storePersistClear1Add(store, 2)
			_, effects := store.Transition(store.GetSnapshot(), xs.E{"type": "add", "amount": 10})

			xstore.ClearStorage(store)
			for _, effect := range effects {
				if effect.Run != nil {
					effect.Run(nil)
				}
			}
			assert.Nil(t, getItem(t, storage, "counter"))

			storePersistClear1Add(store, 3)
			if strategy == xstore.PersistEvent {
				assert.Equal(t, storePersistClear1Log(2, 3), storedJSON(t, storage, "counter"))
			} else {
				assert.Equal(t, storePersistClear1Snap(5), storedJSON(t, storage, "counter"))
			}
		})
	}
}

// JS: clearStorage event history > discards a %s read that started before clearing
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persistClear.test.ts#L234
func TestStorePersistClear_DiscardsReadThatStartedBeforeClearing(t *testing.T) {
	for _, strategy := range []xstore.PersistStrategy{xstore.PersistSnapshot, xstore.PersistEvent} {
		// JS test (shared case definition): https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persistClear.test.ts#L234
		t.Run(string(strategy), func(t *testing.T) {
			var mu sync.Mutex
			values := map[string]string{}
			deferReads := false
			var finishRead func()
			storage := xstore.StorageFuncs{
				GetItemFunc: func(name string) (*string, *xstore.Promise[*string]) {
					mu.Lock()
					defer mu.Unlock()
					var value *string
					if v, ok := values[name]; ok {
						value = &v
					}
					if !deferReads {
						return value, nil
					}
					p, resolve, _ := xstore.NewPromise[*string]()
					finishRead = func() { resolve(value) }
					return nil, p
				},
				SetItemFunc: func(name, value string) *xstore.Promise[struct{}] {
					mu.Lock()
					defer mu.Unlock()
					values[name] = value
					return nil
				},
				RemoveItemFunc: func(name string) *xstore.Promise[struct{}] {
					mu.Lock()
					defer mu.Unlock()
					delete(values, name)
					return nil
				},
			}
			setDeferReads := func(v bool) {
				mu.Lock()
				defer mu.Unlock()
				deferReads = v
			}
			store := storePersistClear1NewStore(storage, storePersistClear1Opts{Strategy: strategy})
			storePersistClear1Add(store, 2)

			// The read captures the pre-clear value, then resolves after clearing.
			setDeferReads(true)
			hydrating := xstore.RehydrateStore(store)
			xstore.ClearStorage(store)
			storePersistClear1Add(store, 5)
			setDeferReads(false)
			mu.Lock()
			finish := finishRead
			mu.Unlock()
			require.NotNil(t, finish)
			finish()
			_, err := hydrating.Wait()
			require.NoError(t, err)

			assert.Equal(t, storePersistClear1Ctx{Count: 7}, store.GetSnapshot().Context)
			storePersistClear1Add(store, 3)
			if strategy == xstore.PersistEvent {
				assert.Equal(t, storePersistClear1Log(2, 5, 3), storedJSON(t, storage, "counter"))
			} else {
				assert.Equal(t, storePersistClear1Snap(10), storedJSON(t, storage, "counter"))
			}
		})
	}
}

// JS: clearStorage event history > does not restore a pre-clear %s from its deferred persistence effect
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persistClear.test.ts#L291
func TestStorePersistClear_DoesNotRestorePreClearFromDeferredPersistenceEffect(t *testing.T) {
	for _, strategy := range []xstore.PersistStrategy{xstore.PersistSnapshot, xstore.PersistEvent} {
		// JS test (shared case definition): https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persistClear.test.ts#L291
		t.Run(string(strategy), func(t *testing.T) {
			storage := newMemoryStorage()
			store := storePersistClear1NewStore(storage, storePersistClear1Opts{Strategy: strategy})
			subscription := store.SubscribeNext(func(snapshot *xstore.StoreSnapshot[storePersistClear1Ctx]) {
				if snapshot.Context.Count == 2 {
					xstore.ClearStorage(store)
				}
			})

			storePersistClear1Add(store, 2)
			subscription.Unsubscribe()
			xstore.FlushStorage(store)
			assert.Nil(t, getItem(t, storage, "counter"))

			storePersistClear1Add(store, 3)
			if strategy == xstore.PersistEvent {
				assert.Equal(t, storePersistClear1Log(2, 3), storedJSON(t, storage, "counter"))
			} else {
				assert.Equal(t, storePersistClear1Snap(5), storedJSON(t, storage, "counter"))
			}
		})
	}
}

// JS: clearStorage event history > orders new history after an in-flight write and asynchronous removal
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persistClear.test.ts#L327
func TestStorePersistClear_OrdersNewHistoryAfterInFlightWriteAndAsyncRemoval(t *testing.T) {
	var mu sync.Mutex
	var operations []string
	var saved *string
	var finishFirstWrite func()
	var finishRemoval func()
	removalStarted := newSignal()
	firstWrite := true
	getOperations := func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), operations...)
	}
	storage := xstore.StorageFuncs{
		GetItemFunc: func(string) (*string, *xstore.Promise[*string]) {
			mu.Lock()
			defer mu.Unlock()
			return saved, nil
		},
		SetItemFunc: func(_, value string) *xstore.Promise[struct{}] {
			var parsed struct {
				Events []map[string]any `json:"events"`
			}
			assert.NoError(t, json.Unmarshal([]byte(value), &parsed))
			mu.Lock()
			defer mu.Unlock()
			operations = append(operations, fmt.Sprintf("write %d", int(parsed.Events[0]["amount"].(float64))))
			if firstWrite {
				firstWrite = false
				p, resolve, _ := xstore.NewPromise[struct{}]()
				finishFirstWrite = func() {
					mu.Lock()
					saved = &value
					mu.Unlock()
					resolve(struct{}{})
				}
				return p
			}
			saved = &value
			return nil
		},
		RemoveItemFunc: func(string) *xstore.Promise[struct{}] {
			mu.Lock()
			operations = append(operations, "clear")
			p, resolve, _ := xstore.NewPromise[struct{}]()
			finishRemoval = func() {
				mu.Lock()
				saved = nil
				mu.Unlock()
				resolve(struct{}{})
			}
			mu.Unlock()
			removalStarted.Resolve()
			return p
		},
	}
	store := storePersistClear1NewCounter(storage)
	storePersistClear1Add(store, 2)
	cleared := xstore.ClearStorage(store)
	storePersistClear1Add(store, 3)
	assert.Equal(t, []string{"write 2"}, getOperations())

	mu.Lock()
	finish := finishFirstWrite
	mu.Unlock()
	require.NotNil(t, finish)
	finish()
	removalStarted.Wait(t)
	assert.Equal(t, []string{"write 2", "clear"}, getOperations())
	mu.Lock()
	finishRm := finishRemoval
	mu.Unlock()
	require.NotNil(t, finishRm)
	finishRm()
	_, err := cleared.Wait()
	require.NoError(t, err)
	_, err = xstore.FlushStorage(store).Wait()
	require.NoError(t, err)

	assert.Equal(t, []string{"write 2", "clear", "write 3"}, getOperations())
	assert.Equal(t, storePersistClear1Log(2, 3), storedJSON(t, storage, "counter"))
	assert.Equal(t, storePersistClear1Ctx{Count: 5}, storePersistClear1NewCounter(storage).GetSnapshot().Context)
}
