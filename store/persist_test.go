package store_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	xstore "github.com/nguyenvanduocit/go-xstate/store"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---- file-local helpers (prefix: storePersist1) ----

// storePersist1Counter is the `{ count: 0 }` context used by most tests.
type storePersist1Counter struct {
	Count int `json:"count"`
}

func storePersist1Inc(c storePersist1Counter, _ xs.Event, _ *xstore.EnqueueObject[storePersist1Counter]) (storePersist1Counter, bool) {
	return storePersist1Counter{Count: c.Count + 1}, true
}

// storePersist1CounterConfig is `{ context: { count: 0 }, on: { inc } }`.
func storePersist1CounterConfig() xstore.StoreConfig[storePersist1Counter] {
	return xstore.StoreConfig[storePersist1Counter]{
		Context: storePersist1Counter{Count: 0},
		On: map[string]xstore.StoreAssigner[storePersist1Counter]{
			"inc": storePersist1Inc,
		},
	}
}

// storePersist1NewCounter is createStore(counter config).with(persist(opts)).
func storePersist1NewCounter(opts xstore.PersistOptions[storePersist1Counter]) *xstore.Store[storePersist1Counter] {
	return xstore.CreateStore(storePersist1CounterConfig()).With(xstore.Persist(opts))
}

// storePersist1Seed mirrors `storage.setItem(name, JSON.stringify(v))`.
func storePersist1Seed(t testing.TB, storage xstore.Storage, name string, v any) {
	t.Helper()
	p := storage.SetItem(name, toJSON(t, v))
	_, err := p.Wait()
	require.NoError(t, err)
}

// storePersist1Stored is JSON.parse(storage.getItem(name)) as an object.
func storePersist1Stored(t testing.TB, storage xstore.Storage, name string) map[string]any {
	t.Helper()
	m, ok := storedJSON(t, storage, name).(map[string]any)
	require.True(t, ok, "stored value for %q is not a JSON object", name)
	return m
}

// storePersist1Overlay copies base and overlays the JSON object `persisted`
// onto it (the JS `{ ...base, ...persisted }` spread).
func storePersist1Overlay[T any](persisted any, base T) T {
	b, err := json.Marshal(persisted)
	if err != nil {
		panic(err)
	}
	out := base
	if err := json.Unmarshal(b, &out); err != nil {
		panic(err)
	}
	return out
}

// storePersist1Int reads a numeric event payload that is an int when the
// event was sent live and a float64 when it was replayed from JSON.
func storePersist1Int(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case float64:
		return int(n)
	}
	panic(fmt.Sprintf("not a number: %T", v))
}

// storePersist1Resolved is an already-resolved async setItem/removeItem result.
func storePersist1Resolved() *xstore.Promise[struct{}] {
	p, resolve, _ := xstore.NewPromise[struct{}]()
	resolve(struct{}{})
	return p
}

// storePersist1JSONEqual compares two values by their JSON form.
func storePersist1JSONEqual(a, b any) bool {
	var av, bv any
	ab, err := json.Marshal(a)
	if err != nil {
		return false
	}
	bb, err := json.Marshal(b)
	if err != nil {
		return false
	}
	if json.Unmarshal(ab, &av) != nil || json.Unmarshal(bb, &bv) != nil {
		return false
	}
	return reflect.DeepEqual(av, bv)
}

// storePersist1CalledWith mirrors toHaveBeenCalledWith(want) for a single
// argument compared by JSON form.
func storePersist1CalledWith(sp *spy, want any) bool {
	for _, call := range sp.Calls() {
		if len(call) == 1 && storePersist1JSONEqual(call[0], want) {
			return true
		}
	}
	return false
}

// storePersist1WaitFor polls cond for up to 2s.
func storePersist1WaitFor(t testing.TB, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for condition")
		}
		time.Sleep(time.Millisecond)
	}
}

// storePersist1Strings is a mutex-guarded []string.
type storePersist1Strings struct {
	mu    sync.Mutex
	items []string
}

func (l *storePersist1Strings) Push(s string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.items = append(l.items, s)
}

func (l *storePersist1Strings) Items() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]string(nil), l.items...)
}

// storePersist1CountOf extracts `context.count` from a serialized snapshot.
func storePersist1CountOf(value string) int {
	var v struct {
		Context struct {
			Count int `json:"count"`
		} `json:"context"`
	}
	if err := json.Unmarshal([]byte(value), &v); err != nil {
		panic(err)
	}
	return v.Context.Count
}

// ---- persistence lifecycle regressions ----

// JS: persistence lifecycle regressions > preserves an eligible snapshot when a nested event is filtered (throttle %i)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L17
func TestStorePersist_LifecycleRegressions_PreservesEligibleSnapshotWhenNestedEventIsFiltered(t *testing.T) {
	type valueCtx struct {
		Value int `json:"value"`
	}
	for _, throttle := range []int{0, 100} {
		// JS test (shared case definition): https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L17
		t.Run(fmt.Sprintf("throttle %d", throttle), func(t *testing.T) {
			storage := newMemoryStorage()
			s := xstore.CreateStore(xstore.StoreConfig[valueCtx]{
				Context: valueCtx{Value: 0},
				On: map[string]xstore.StoreAssigner[valueCtx]{
					"outer": func(c valueCtx, _ xs.Event, _ *xstore.EnqueueObject[valueCtx]) (valueCtx, bool) {
						return valueCtx{Value: 1}, true
					},
					"inner": func(c valueCtx, _ xs.Event, _ *xstore.EnqueueObject[valueCtx]) (valueCtx, bool) {
						return valueCtx{Value: 2}, true
					},
				},
			}).With(xstore.Persist(xstore.PersistOptions[valueCtx]{
				Name:     "filtered-nested",
				Storage:  storage,
				Throttle: ms(throttle),
				Clock:    xs.NewSimulatedClock(),
				Filter:   func(event xs.Event) bool { return event.EventType() == "outer" },
			}))
			s.SubscribeNext(func(snapshot *xstore.StoreSnapshot[valueCtx]) {
				if snapshot.Context.Value == 1 {
					s.Trigger("inner")
				}
			})
			s.Trigger("outer")
			assert.Equal(t, 2, s.GetSnapshot().Context.Value)
			_, err := xstore.FlushStorage(s).Wait()
			require.NoError(t, err)
			stored := storePersist1Stored(t, storage, "filtered-nested")
			assert.Equal(t, map[string]any{"value": float64(1)}, stored["context"])
		})
	}
}

// JS: persistence lifecycle regressions > orders new async writes after a queued clear
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L47
func TestStorePersist_LifecycleRegressions_OrdersNewAsyncWritesAfterQueuedClear(t *testing.T) {
	operations := &storePersist1Strings{}
	s := storePersist1NewCounter(xstore.PersistOptions[storePersist1Counter]{
		Name: "counter",
		Storage: xstore.StorageFuncs{
			GetItemFunc: func(string) (*string, *xstore.Promise[*string]) { return nil, nil },
			SetItemFunc: func(_ string, value string) *xstore.Promise[struct{}] {
				operations.Push(fmt.Sprintf("write %d", storePersist1CountOf(value)))
				return storePersist1Resolved()
			},
			RemoveItemFunc: func(string) *xstore.Promise[struct{}] {
				operations.Push("clear")
				return storePersist1Resolved()
			},
		},
	})

	s.Trigger("inc")
	cleared := xstore.ClearStorage(s)
	s.Trigger("inc")
	_, err := cleared.Wait()
	require.NoError(t, err)
	_, err = xstore.FlushStorage(s).Wait()
	require.NoError(t, err)
	assert.Equal(t, []string{"write 1", "clear", "write 2"}, operations.Items())
}

// JS: persistence lifecycle regressions > waits for writes queued by onDone callbacks
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L75
func TestStorePersist_LifecycleRegressions_WaitsForWritesQueuedByOnDoneCallbacks(t *testing.T) {
	var mu sync.Mutex
	var written []int
	var s *xstore.Store[storePersist1Counter]
	s = storePersist1NewCounter(xstore.PersistOptions[storePersist1Counter]{
		Name: "counter",
		Storage: xstore.StorageFuncs{
			GetItemFunc: func(string) (*string, *xstore.Promise[*string]) { return nil, nil },
			SetItemFunc: func(_ string, value string) *xstore.Promise[struct{}] {
				p, resolve, _ := xstore.NewPromise[struct{}]()
				count := storePersist1CountOf(value)
				go func() {
					mu.Lock()
					written = append(written, count)
					mu.Unlock()
					resolve(struct{}{})
				}()
				return p
			},
			RemoveItemFunc: func(string) *xstore.Promise[struct{}] { return nil },
		},
		OnDone: func(data any) {
			var c storePersist1Counter
			b, err := json.Marshal(data)
			if err != nil {
				panic(err)
			}
			if err := json.Unmarshal(b, &c); err != nil {
				panic(err)
			}
			if c.Count == 1 {
				s.Trigger("inc")
			}
		},
	})

	s.Trigger("inc")
	_, err := xstore.FlushStorage(s).Wait()
	require.NoError(t, err)
	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, []int{1, 2}, written)
}

// JS: persistence lifecycle regressions > restarts event history after clearStorage
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L103
func TestStorePersist_LifecycleRegressions_RestartsEventHistoryAfterClearStorage(t *testing.T) {
	storage := newMemoryStorage()
	s := storePersist1NewCounter(xstore.PersistOptions[storePersist1Counter]{
		Name: "counter", Storage: storage, Strategy: xstore.PersistEvent,
	})

	s.Trigger("inc")
	xstore.ClearStorage(s)
	s.Trigger("inc")

	stored := storePersist1Stored(t, storage, "counter")
	assert.Len(t, stored["events"], 1)
	assert.Equal(t, map[string]any{"count": float64(1)}, stored["checkpoint"])

	restored := storePersist1NewCounter(xstore.PersistOptions[storePersist1Counter]{
		Name: "counter", Storage: storage, Strategy: xstore.PersistEvent,
	})
	assert.Equal(t, 2, restored.GetSnapshot().Context.Count)
}

// JS: persistence lifecycle regressions > reports a rejected initial async read (%s)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L125
func TestStorePersist_LifecycleRegressions_ReportsRejectedInitialAsyncRead(t *testing.T) {
	type emptyCtx struct{}
	for _, strategy := range []xstore.PersistStrategy{xstore.PersistSnapshot, xstore.PersistEvent} {
		// JS test (shared case definition): https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L125
		t.Run(string(strategy), func(t *testing.T) {
			readErr := errors.New("read failed")
			rejected, _, reject := xstore.NewPromise[*string]()
			reject(readErr)
			onError := newSpy()
			xstore.CreateStore(xstore.StoreConfig[emptyCtx]{
				Context: emptyCtx{},
				On:      map[string]xstore.StoreAssigner[emptyCtx]{},
			}).With(xstore.Persist(xstore.PersistOptions[emptyCtx]{
				Name:     "counter",
				Strategy: strategy,
				OnError:  func(err any) { onError.Call(err) },
				Storage: xstore.StorageFuncs{
					GetItemFunc:    func(string) (*string, *xstore.Promise[*string]) { return nil, rejected },
					SetItemFunc:    func(string, string) *xstore.Promise[struct{}] { return nil },
					RemoveItemFunc: func(string) *xstore.Promise[struct{}] { return nil },
				},
			}))
			storePersist1WaitFor(t, func() bool { return onError.Count() >= 1 })
			sleep(20)
			calls := onError.Calls()
			require.Len(t, calls, 1)
			require.Len(t, calls[0], 1)
			assert.Equal(t, readErr, calls[0][0])
		})
	}
}

// JS: persistence lifecycle regressions > applies pick once when flushing a throttled update
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L150
func TestStorePersist_LifecycleRegressions_AppliesPickOnceWhenFlushingThrottledUpdate(t *testing.T) {
	storage := newMemoryStorage()
	pick := newSpy()
	s := storePersist1NewCounter(xstore.PersistOptions[storePersist1Counter]{
		Name: "counter", Storage: storage, Throttle: ms(100), Clock: xs.NewSimulatedClock(),
		Pick: func(c storePersist1Counter) any {
			pick.Call(c)
			return storePersist1Counter{Count: c.Count + 1}
		},
	})

	s.Trigger("inc")
	xstore.FlushStorage(s)
	assert.Equal(t, 1, pick.Count())
	assert.Equal(t, map[string]any{"count": float64(2)}, storePersist1Stored(t, storage, "counter")["context"])
}

// JS: persistence lifecycle regressions > preserves updates triggered by onDone during a throttled flush (%s)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L168
func TestStorePersist_LifecycleRegressions_PreservesUpdatesTriggeredByOnDoneDuringThrottledFlush(t *testing.T) {
	for _, strategy := range []xstore.PersistStrategy{xstore.PersistSnapshot, xstore.PersistEvent} {
		// JS test (shared case definition): https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L168
		t.Run(string(strategy), func(t *testing.T) {
			storage := newMemoryStorage()
			firstWrite := true
			var s *xstore.Store[storePersist1Counter]
			s = storePersist1NewCounter(xstore.PersistOptions[storePersist1Counter]{
				Name: "counter", Storage: storage, Strategy: strategy,
				Throttle: ms(100), Clock: xs.NewSimulatedClock(),
				OnDone: func(any) {
					if firstWrite {
						firstWrite = false
						s.Trigger("inc")
					}
				},
			})

			s.Trigger("inc")
			xstore.FlushStorage(s)
			xstore.FlushStorage(s)
			saved := storePersist1Stored(t, storage, "counter")
			if strategy == xstore.PersistEvent {
				assert.Len(t, saved["events"], 2)
			} else {
				assert.Equal(t, map[string]any{"count": float64(2)}, saved["context"])
			}
		})
	}
}

// JS: persistence lifecycle regressions > cancels buffered writes when storage is cleared (%s)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L201
func TestStorePersist_LifecycleRegressions_CancelsBufferedWritesWhenStorageIsCleared(t *testing.T) {
	for _, strategy := range []xstore.PersistStrategy{xstore.PersistSnapshot, xstore.PersistEvent} {
		// JS test (shared case definition): https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L201
		t.Run(string(strategy), func(t *testing.T) {
			clock := xs.NewSimulatedClock()
			storage := newMemoryStorage()
			s := storePersist1NewCounter(xstore.PersistOptions[storePersist1Counter]{
				Name: "counter", Strategy: strategy, Storage: storage,
				Throttle: ms(100), Clock: clock,
			})

			s.Trigger("inc")
			assert.Nil(t, xstore.ClearStorage(s))
			clock.Increment(ms(100))
			xstore.FlushStorage(s)
			assert.Nil(t, getItem(t, storage, "counter"))
		})
	}
}

// JS: persistence lifecycle regressions > removes storage after an in-flight async write finishes
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L223
func TestStorePersist_LifecycleRegressions_RemovesStorageAfterInFlightAsyncWriteFinishes(t *testing.T) {
	var mu sync.Mutex
	var completeWrite func()
	var saved *string
	removeItem := newSpy()
	s := storePersist1NewCounter(xstore.PersistOptions[storePersist1Counter]{
		Name: "counter",
		Storage: xstore.StorageFuncs{
			GetItemFunc: func(string) (*string, *xstore.Promise[*string]) { return nil, nil },
			SetItemFunc: func(_ string, value string) *xstore.Promise[struct{}] {
				p, resolve, _ := xstore.NewPromise[struct{}]()
				mu.Lock()
				defer mu.Unlock()
				completeWrite = func() {
					mu.Lock()
					saved = &value
					mu.Unlock()
					resolve(struct{}{})
				}
				return p
			},
			RemoveItemFunc: func(name string) *xstore.Promise[struct{}] {
				removeItem.Call(name)
				mu.Lock()
				saved = nil
				mu.Unlock()
				return nil
			},
		},
	})

	s.Trigger("inc")
	cleared := xstore.ClearStorage(s)
	assert.Equal(t, 0, removeItem.Count())
	mu.Lock()
	complete := completeWrite
	mu.Unlock()
	require.NotNil(t, complete)
	complete()
	_, err := cleared.Wait()
	require.NoError(t, err)
	require.Equal(t, 1, removeItem.Count())
	assert.Equal(t, []any{"counter"}, removeItem.Calls()[0])
	mu.Lock()
	defer mu.Unlock()
	assert.Nil(t, saved)
}

// ---- persist ----

// JS: persist > should persist context to storage after each event
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L286
func TestStorePersist_Persist_ShouldPersistContextToStorageAfterEachEvent(t *testing.T) {
	storage := newMemoryStorage()
	s := storePersist1NewCounter(xstore.PersistOptions[storePersist1Counter]{Name: "test", Storage: storage})

	s.Trigger("inc")

	stored := storePersist1Stored(t, storage, "test")
	assert.Equal(t, map[string]any{"count": float64(1)}, stored["context"])
	assert.Equal(t, float64(0), stored["version"])
}

// JS: persist > should restore context from storage on creation
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L300
func TestStorePersist_Persist_ShouldRestoreContextFromStorageOnCreation(t *testing.T) {
	storage := newMemoryStorage()
	storePersist1Seed(t, storage, "test", map[string]any{
		"context": map[string]any{"count": 42},
		"version": 0,
	})

	s := storePersist1NewCounter(xstore.PersistOptions[storePersist1Counter]{Name: "test", Storage: storage})

	assert.Equal(t, 42, s.GetSnapshot().Context.Count)
}

// JS: persist > should set _persist.hydrated to true on sync hydration
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L318
func TestStorePersist_Persist_ShouldSetHydratedToTrueOnSyncHydration(t *testing.T) {
	storage := newMemoryStorage()
	s := storePersist1NewCounter(xstore.PersistOptions[storePersist1Counter]{Name: "test", Storage: storage})

	assert.True(t, xstore.IsHydrated(s))
}

// JS: persist > should set _persist.hydrated to true when storage is empty
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L328
func TestStorePersist_Persist_ShouldSetHydratedToTrueWhenStorageIsEmpty(t *testing.T) {
	storage := newMemoryStorage()
	s := storePersist1NewCounter(xstore.PersistOptions[storePersist1Counter]{Name: "test", Storage: storage})

	assert.True(t, xstore.IsHydrated(s))
}

// JS: persist > should preserve _persist metadata across transitions
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L338
func TestStorePersist_Persist_ShouldPreservePersistMetadataAcrossTransitions(t *testing.T) {
	storage := newMemoryStorage()
	s := storePersist1NewCounter(xstore.PersistOptions[storePersist1Counter]{Name: "test", Storage: storage})

	s.Trigger("inc")
	assert.True(t, xstore.IsHydrated(s))
}

// ---- persist - pick ----

// JS: persist - pick > should only persist selected fields
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L351
func TestStorePersist_Pick_ShouldOnlyPersistSelectedFields(t *testing.T) {
	type secretCtx struct {
		Count  int    `json:"count"`
		Secret string `json:"secret"`
	}
	storage := newMemoryStorage()
	s := xstore.CreateStore(xstore.StoreConfig[secretCtx]{
		Context: secretCtx{Count: 0, Secret: "do-not-persist"},
		On: map[string]xstore.StoreAssigner[secretCtx]{
			"inc": func(c secretCtx, _ xs.Event, _ *xstore.EnqueueObject[secretCtx]) (secretCtx, bool) {
				c.Count++
				return c, true
			},
		},
	}).With(xstore.Persist(xstore.PersistOptions[secretCtx]{
		Name:    "test",
		Storage: storage,
		Pick:    func(c secretCtx) any { return map[string]any{"count": c.Count} },
	}))

	s.Trigger("inc")

	stored := storePersist1Stored(t, storage, "test")
	assert.Equal(t, map[string]any{"count": float64(1)}, stored["context"])
	_, hasSecret := stored["context"].(map[string]any)["secret"]
	assert.False(t, hasSecret)
}

// JS: persist - pick > should merge picked data with full context on restore
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L373
func TestStorePersist_Pick_ShouldMergePickedDataWithFullContextOnRestore(t *testing.T) {
	type nameCtx struct {
		Count int    `json:"count"`
		Name  string `json:"name"`
	}
	storage := newMemoryStorage()
	storePersist1Seed(t, storage, "test", map[string]any{
		"context": map[string]any{"count": 42},
		"version": 0,
	})

	s := xstore.CreateStore(xstore.StoreConfig[nameCtx]{
		Context: nameCtx{Count: 0, Name: "Ada"},
		On: map[string]xstore.StoreAssigner[nameCtx]{
			"inc": func(c nameCtx, _ xs.Event, _ *xstore.EnqueueObject[nameCtx]) (nameCtx, bool) {
				c.Count++
				return c, true
			},
		},
	}).With(xstore.Persist(xstore.PersistOptions[nameCtx]{
		Name:    "test",
		Storage: storage,
		Pick:    func(c nameCtx) any { return map[string]any{"count": c.Count} },
	}))

	assert.Equal(t, nameCtx{Count: 42, Name: "Ada"}, s.GetSnapshot().Context)
}

// ---- persist - version + migrate ----

type storePersist1Labeled struct {
	Count int    `json:"count"`
	Label string `json:"label"`
}

func storePersist1LabeledConfig() xstore.StoreConfig[storePersist1Labeled] {
	return xstore.StoreConfig[storePersist1Labeled]{
		Context: storePersist1Labeled{Count: 0, Label: "default"},
		On: map[string]xstore.StoreAssigner[storePersist1Labeled]{
			"inc": func(c storePersist1Labeled, _ xs.Event, _ *xstore.EnqueueObject[storePersist1Labeled]) (storePersist1Labeled, bool) {
				c.Count++
				return c, true
			},
		},
	}
}

// JS: persist - version + migrate > should migrate persisted state when version differs
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L399
func TestStorePersist_VersionMigrate_ShouldMigratePersistedStateWhenVersionDiffers(t *testing.T) {
	storage := newMemoryStorage()
	storePersist1Seed(t, storage, "test", map[string]any{
		"context": map[string]any{"count": 10},
		"version": 1,
	})

	s := xstore.CreateStore(storePersist1LabeledConfig()).With(xstore.Persist(xstore.PersistOptions[storePersist1Labeled]{
		Name:    "test",
		Storage: storage,
		Version: 2,
		Migrate: func(persisted any, version any) storePersist1Labeled {
			out := storePersist1Overlay(persisted, storePersist1Labeled{})
			if fmt.Sprint(version) == "1" {
				out.Label = "migrated"
			}
			return out
		},
	}))

	assert.Equal(t, 10, s.GetSnapshot().Context.Count)
	assert.Equal(t, "migrated", s.GetSnapshot().Context.Label)
}

// JS: persist - version + migrate > should migrate with string versions
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L430
func TestStorePersist_VersionMigrate_ShouldMigrateWithStringVersions(t *testing.T) {
	storage := newMemoryStorage()
	storePersist1Seed(t, storage, "test", map[string]any{
		"context": map[string]any{"count": 10},
		"version": "1.0.0",
	})

	s := xstore.CreateStore(storePersist1LabeledConfig()).With(xstore.Persist(xstore.PersistOptions[storePersist1Labeled]{
		Name:    "test",
		Storage: storage,
		Version: "2.0.0",
		Migrate: func(persisted any, version any) storePersist1Labeled {
			out := storePersist1Overlay(persisted, storePersist1Labeled{})
			if version == "1.0.0" {
				out.Label = "migrated"
			}
			return out
		},
	}))

	assert.Equal(t, 10, s.GetSnapshot().Context.Count)
	assert.Equal(t, "migrated", s.GetSnapshot().Context.Label)
}

// JS: persist - version + migrate > should not migrate when version matches
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L461
func TestStorePersist_VersionMigrate_ShouldNotMigrateWhenVersionMatches(t *testing.T) {
	storage := newMemoryStorage()
	storePersist1Seed(t, storage, "test", map[string]any{
		"context": map[string]any{"count": 5},
		"version": 2,
	})

	migrateFn := newSpy()

	s := storePersist1NewCounter(xstore.PersistOptions[storePersist1Counter]{
		Name:    "test",
		Storage: storage,
		Version: 2,
		Migrate: func(persisted any, version any) storePersist1Counter {
			migrateFn.Call(persisted, version)
			return storePersist1Overlay(persisted, storePersist1Counter{})
		},
	})

	assert.Equal(t, 0, migrateFn.Count())
	assert.Equal(t, 5, s.GetSnapshot().Context.Count)
}

// ---- persist - merge ----

// JS: persist - merge > should use custom merge strategy
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L491
func TestStorePersist_Merge_ShouldUseCustomMergeStrategy(t *testing.T) {
	type itemsCtx struct {
		Count int      `json:"count"`
		Items []string `json:"items"`
	}
	storage := newMemoryStorage()
	storePersist1Seed(t, storage, "test", map[string]any{
		"context": map[string]any{"count": 42, "items": []string{"a"}},
		"version": 0,
	})

	s := xstore.CreateStore(xstore.StoreConfig[itemsCtx]{
		Context: itemsCtx{Count: 0, Items: []string{"b", "c"}},
		On: map[string]xstore.StoreAssigner[itemsCtx]{
			"inc": func(c itemsCtx, _ xs.Event, _ *xstore.EnqueueObject[itemsCtx]) (itemsCtx, bool) {
				c.Count++
				return c, true
			},
		},
	}).With(xstore.Persist(xstore.PersistOptions[itemsCtx]{
		Name:    "test",
		Storage: storage,
		Merge: func(persisted any, current itemsCtx) itemsCtx {
			// { ...current, ...persisted, items: [...current.items, ...(persisted.items ?? [])] }
			base := itemsCtx{Count: current.Count}
			merged := storePersist1Overlay(persisted, base)
			merged.Items = append(append([]string{}, current.Items...), storePersist1Overlay(persisted, itemsCtx{}).Items...)
			return merged
		},
	}))

	assert.Equal(t, 42, s.GetSnapshot().Context.Count)
	assert.Equal(t, []string{"b", "c", "a"}, s.GetSnapshot().Context.Items)
}

// ---- persist - serialize / deserialize ----

// JS: persist - serialize / deserialize > should use custom serializer and deserializer
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L522
func TestStorePersist_SerializeDeserialize_ShouldUseCustomSerializerAndDeserializer(t *testing.T) {
	storage := newMemoryStorage()
	prefix := "CUSTOM:"
	opts := func() xstore.PersistOptions[storePersist1Counter] {
		return xstore.PersistOptions[storePersist1Counter]{
			Name:    "test",
			Storage: storage,
			Serialize: func(value xstore.PersistStorageValue) string {
				b, err := json.Marshal(value)
				if err != nil {
					panic(err)
				}
				return prefix + string(b)
			},
			Deserialize: func(str string) xstore.PersistStorageValue {
				var out xstore.PersistStorageValue
				if err := json.Unmarshal([]byte(str[len(prefix):]), &out); err != nil {
					panic(err)
				}
				return out
			},
		}
	}

	s := storePersist1NewCounter(opts())

	s.Trigger("inc")

	raw := getItem(t, storage, "test")
	require.NotNil(t, raw)
	assert.True(t, len(*raw) >= len(prefix) && (*raw)[:len(prefix)] == prefix)

	// Verify roundtrip: create new store from same storage
	store2 := storePersist1NewCounter(opts())

	assert.Equal(t, 1, store2.GetSnapshot().Context.Count)
}

// ---- persist - throttle ----

// JS: persist - throttle > should batch writes with throttle
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L567
func TestStorePersist_Throttle_ShouldBatchWritesWithThrottle(t *testing.T) {
	clock := xs.NewSimulatedClock()
	storage := newMemoryStorage()
	s := storePersist1NewCounter(xstore.PersistOptions[storePersist1Counter]{
		Name: "test", Storage: storage, Throttle: ms(100), Clock: clock,
	})

	s.Trigger("inc") // count = 1
	s.Trigger("inc") // count = 2
	s.Trigger("inc") // count = 3

	// Not yet written
	assert.Nil(t, getItem(t, storage, "test"))

	clock.Increment(ms(100))

	// Only last value written
	stored := storePersist1Stored(t, storage, "test")
	assert.Equal(t, map[string]any{"count": float64(3)}, stored["context"])
}

// JS: persist - throttle > should not write again if no events between throttle intervals
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L588
func TestStorePersist_Throttle_ShouldNotWriteAgainIfNoEventsBetweenThrottleIntervals(t *testing.T) {
	clock := xs.NewSimulatedClock()
	storage := newMemoryStorage()
	onDone := newSpy()
	s := storePersist1NewCounter(xstore.PersistOptions[storePersist1Counter]{
		Name: "test", Storage: storage, Throttle: ms(100), Clock: clock,
		OnDone: func(data any) { onDone.Call(data) },
	})

	s.Trigger("inc")
	clock.Increment(ms(100))

	assert.Equal(t, 1, onDone.Count())

	clock.Increment(ms(100))
	// No extra writes
	assert.Equal(t, 1, onDone.Count())
}

// ---- persist - flushStorage ----

// JS: persist - flushStorage > should force immediate write of pending throttled context
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L615
func TestStorePersist_FlushStorage_ShouldForceImmediateWriteOfPendingThrottledContext(t *testing.T) {
	storage := newMemoryStorage()
	s := storePersist1NewCounter(xstore.PersistOptions[storePersist1Counter]{
		Name: "test", Storage: storage, Throttle: ms(1000), Clock: xs.NewSimulatedClock(),
	})

	s.Trigger("inc")
	s.Trigger("inc")

	assert.Nil(t, getItem(t, storage, "test"))

	xstore.FlushStorage(s)

	stored := storePersist1Stored(t, storage, "test")
	assert.Equal(t, map[string]any{"count": float64(2)}, stored["context"])
}

// JS: persist - flushStorage > should throw when store has no persist extension
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L633
func TestStorePersist_FlushStorage_ShouldThrowWhenStoreHasNoPersistExtension(t *testing.T) {
	s := xstore.CreateStore(storePersist1CounterConfig())

	assert.Contains(t,
		panicMessage(func() { xstore.FlushStorage(s) }),
		"flushStorage: store does not have a persist extension")
}

// JS: persist - flushStorage > should return a promise for async storage flushes
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L644
func TestStorePersist_FlushStorage_ShouldReturnAPromiseForAsyncStorageFlushes(t *testing.T) {
	storage := newAsyncMemoryStorage()
	s := storePersist1NewCounter(xstore.PersistOptions[storePersist1Counter]{
		Name: "test", Storage: storage, Throttle: ms(1000), Clock: xs.NewSimulatedClock(),
	})

	_, err := xstore.RehydrateStore(s).Wait()
	require.NoError(t, err)

	s.Trigger("inc")
	s.Trigger("inc")

	_, err = xstore.FlushStorage(s).Wait()
	require.NoError(t, err)

	stored := storePersist1Stored(t, storage, "test")
	assert.Equal(t, map[string]any{"count": float64(2)}, stored["context"])
}

// ---- persist - onDone / onError ----

// JS: persist - onDone / onError > should call onDone after successful write
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L664
func TestStorePersist_OnDoneOnError_ShouldCallOnDoneAfterSuccessfulWrite(t *testing.T) {
	storage := newMemoryStorage()
	onDone := newSpy()
	s := storePersist1NewCounter(xstore.PersistOptions[storePersist1Counter]{
		Name: "test", Storage: storage, OnDone: func(data any) { onDone.Call(data) },
	})

	s.Trigger("inc")

	assert.True(t, storePersist1CalledWith(onDone, map[string]any{"count": 1}))
}

// JS: persist - onDone / onError > should call onDone with picked context when pick is used
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L677
func TestStorePersist_OnDoneOnError_ShouldCallOnDoneWithPickedContextWhenPickIsUsed(t *testing.T) {
	type secretCtx struct {
		Count  int    `json:"count"`
		Secret string `json:"secret"`
	}
	storage := newMemoryStorage()
	onDone := newSpy()
	s := xstore.CreateStore(xstore.StoreConfig[secretCtx]{
		Context: secretCtx{Count: 0, Secret: "hidden"},
		On: map[string]xstore.StoreAssigner[secretCtx]{
			"inc": func(c secretCtx, _ xs.Event, _ *xstore.EnqueueObject[secretCtx]) (secretCtx, bool) {
				c.Count++
				return c, true
			},
		},
	}).With(xstore.Persist(xstore.PersistOptions[secretCtx]{
		Name:    "test",
		Storage: storage,
		OnDone:  func(data any) { onDone.Call(data) },
		Pick:    func(c secretCtx) any { return map[string]any{"count": c.Count} },
	}))

	s.Trigger("inc")

	assert.True(t, storePersist1CalledWith(onDone, map[string]any{"count": 1}))
	// not.toHaveBeenCalledWith(expect.objectContaining({ secret: 'hidden' }))
	for _, call := range onDone.Calls() {
		require.Len(t, call, 1)
		b, err := json.Marshal(call[0])
		require.NoError(t, err)
		var m map[string]any
		require.NoError(t, json.Unmarshal(b, &m))
		assert.NotEqual(t, "hidden", m["secret"])
	}
}

// JS: persist - onDone / onError > should call onError on write failure
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L700
func TestStorePersist_OnDoneOnError_ShouldCallOnErrorOnWriteFailure(t *testing.T) {
	writeErr := errors.New("quota exceeded")
	failStorage := xstore.StorageFuncs{
		GetItemFunc:    func(string) (*string, *xstore.Promise[*string]) { return nil, nil },
		SetItemFunc:    func(string, string) *xstore.Promise[struct{}] { panic(writeErr) },
		RemoveItemFunc: func(string) *xstore.Promise[struct{}] { return nil },
	}
	onError := newSpy()

	s := storePersist1NewCounter(xstore.PersistOptions[storePersist1Counter]{
		Name: "test", Storage: failStorage, OnError: func(err any) { onError.Call(err) },
	})

	s.Trigger("inc")

	found := false
	for _, call := range onError.Calls() {
		if len(call) == 1 && call[0] == any(writeErr) {
			found = true
		}
	}
	assert.True(t, found, "onError should be called with the thrown error")
}

// JS: persist - onDone / onError > should call onError on read failure during hydration
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L721
func TestStorePersist_OnDoneOnError_ShouldCallOnErrorOnReadFailureDuringHydration(t *testing.T) {
	failStorage := xstore.StorageFuncs{
		GetItemFunc: func(string) (*string, *xstore.Promise[*string]) {
			panic(errors.New("read failed"))
		},
		SetItemFunc:    func(string, string) *xstore.Promise[struct{}] { return nil },
		RemoveItemFunc: func(string) *xstore.Promise[struct{}] { return nil },
	}
	onError := newSpy()

	storePersist1NewCounter(xstore.PersistOptions[storePersist1Counter]{
		Name: "test", Storage: failStorage, OnError: func(err any) { onError.Call(err) },
	})

	assert.NotZero(t, onError.Count())
}

// ---- persist - filter ----

// JS: persist - filter > should skip persisting when filter returns false
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L741
func TestStorePersist_Filter_ShouldSkipPersistingWhenFilterReturnsFalse(t *testing.T) {
	type mouse struct {
		X int `json:"x"`
		Y int `json:"y"`
	}
	type mouseCtx struct {
		Count int   `json:"count"`
		Mouse mouse `json:"mouse"`
	}
	storage := newMemoryStorage()
	onDone := newSpy()
	s := xstore.CreateStore(xstore.StoreConfig[mouseCtx]{
		Context: mouseCtx{Count: 0, Mouse: mouse{X: 0, Y: 0}},
		On: map[string]xstore.StoreAssigner[mouseCtx]{
			"inc": func(c mouseCtx, _ xs.Event, _ *xstore.EnqueueObject[mouseCtx]) (mouseCtx, bool) {
				c.Count++
				return c, true
			},
			"mousemove": func(c mouseCtx, e xs.Event, _ *xstore.EnqueueObject[mouseCtx]) (mouseCtx, bool) {
				ev := e.(xs.E)
				c.Mouse = mouse{X: storePersist1Int(ev["x"]), Y: storePersist1Int(ev["y"])}
				return c, true
			},
		},
	}).With(xstore.Persist(xstore.PersistOptions[mouseCtx]{
		Name:    "test",
		Storage: storage,
		OnDone:  func(data any) { onDone.Call(data) },
		Filter:  func(event xs.Event) bool { return event.EventType() != "mousemove" },
	}))

	s.Trigger("mousemove", xs.E{"x": 10, "y": 20})
	assert.Equal(t, 0, onDone.Count())

	s.Trigger("inc")
	assert.Equal(t, 1, onDone.Count())
}

// ---- persist - skipHydration ----

// JS: persist - skipHydration > should not hydrate when skipHydration is true
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L771
func TestStorePersist_SkipHydration_ShouldNotHydrateWhenSkipHydrationIsTrue(t *testing.T) {
	storage := newMemoryStorage()
	storePersist1Seed(t, storage, "test", map[string]any{
		"context": map[string]any{"count": 99},
		"version": 0,
	})

	s := storePersist1NewCounter(xstore.PersistOptions[storePersist1Counter]{
		Name: "test", Storage: storage, SkipHydration: true,
	})

	assert.Equal(t, 0, s.GetSnapshot().Context.Count)
	assert.False(t, xstore.IsHydrated(s))
}

// ---- persist - rehydrateStore ----

// JS: persist - rehydrateStore > should rehydrate from sync storage
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L792
func TestStorePersist_RehydrateStore_ShouldRehydrateFromSyncStorage(t *testing.T) {
	storage := newMemoryStorage()
	storePersist1Seed(t, storage, "test", map[string]any{
		"context": map[string]any{"count": 99},
		"version": 0,
	})

	s := storePersist1NewCounter(xstore.PersistOptions[storePersist1Counter]{
		Name: "test", Storage: storage, SkipHydration: true,
	})

	assert.Equal(t, 0, s.GetSnapshot().Context.Count)

	_, err := xstore.RehydrateStore(s).Wait()
	require.NoError(t, err)

	assert.Equal(t, 99, s.GetSnapshot().Context.Count)
	assert.True(t, xstore.IsHydrated(s))
}

// JS: persist - rehydrateStore > should rehydrate from async storage
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L815
func TestStorePersist_RehydrateStore_ShouldRehydrateFromAsyncStorage(t *testing.T) {
	storage := newAsyncMemoryStorage()
	storePersist1Seed(t, storage, "test", map[string]any{
		"context": map[string]any{"count": 77},
		"version": 0,
	})

	s := storePersist1NewCounter(xstore.PersistOptions[storePersist1Counter]{
		Name: "test", Storage: storage, SkipHydration: true,
	})

	assert.Equal(t, 0, s.GetSnapshot().Context.Count)

	_, err := xstore.RehydrateStore(s).Wait()
	require.NoError(t, err)

	assert.Equal(t, 77, s.GetSnapshot().Context.Count)
	assert.True(t, xstore.IsHydrated(s))
}

// JS: persist - rehydrateStore > should merge with events sent before rehydration
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L838
func TestStorePersist_RehydrateStore_ShouldMergeWithEventsSentBeforeRehydration(t *testing.T) {
	storage := newMemoryStorage()
	storePersist1Seed(t, storage, "test", map[string]any{
		"context": map[string]any{"count": 10},
		"version": 0,
	})

	s := storePersist1NewCounter(xstore.PersistOptions[storePersist1Counter]{
		Name: "test", Storage: storage, SkipHydration: true,
	})

	// Send events before rehydration
	s.Trigger("inc") // count = 1

	_, err := xstore.RehydrateStore(s).Wait()
	require.NoError(t, err)

	// Default merge: persisted overwrites current, so count = 10
	assert.Equal(t, 10, s.GetSnapshot().Context.Count)
}

// JS: persist - rehydrateStore > should handle empty storage gracefully
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L862
func TestStorePersist_RehydrateStore_ShouldHandleEmptyStorageGracefully(t *testing.T) {
	storage := newMemoryStorage()

	s := storePersist1NewCounter(xstore.PersistOptions[storePersist1Counter]{
		Name: "test", Storage: storage, SkipHydration: true,
	})

	_, err := xstore.RehydrateStore(s).Wait()
	require.NoError(t, err)

	assert.Equal(t, 0, s.GetSnapshot().Context.Count)
	assert.True(t, xstore.IsHydrated(s))
}

// JS: persist - rehydrateStore > should apply migration during rehydration
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L876
func TestStorePersist_RehydrateStore_ShouldApplyMigrationDuringRehydration(t *testing.T) {
	storage := newMemoryStorage()
	storePersist1Seed(t, storage, "test", map[string]any{
		"context": map[string]any{"count": 5},
		"version": 1,
	})

	s := xstore.CreateStore(xstore.StoreConfig[storePersist1Labeled]{
		Context: storePersist1Labeled{Count: 0, Label: ""},
		On:      storePersist1LabeledConfig().On,
	}).With(xstore.Persist(xstore.PersistOptions[storePersist1Labeled]{
		Name:          "test",
		Storage:       storage,
		Version:       2,
		SkipHydration: true,
		Migrate: func(persisted any, version any) storePersist1Labeled {
			out := storePersist1Overlay(persisted, storePersist1Labeled{})
			if fmt.Sprint(version) == "1" {
				out.Label = "migrated"
			}
			return out
		},
	}))

	_, err := xstore.RehydrateStore(s).Wait()
	require.NoError(t, err)

	assert.Equal(t, 5, s.GetSnapshot().Context.Count)
	assert.Equal(t, "migrated", s.GetSnapshot().Context.Label)
}

// JS: persist - rehydrateStore > should throw when store has no persist extension
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L910
func TestStorePersist_RehydrateStore_ShouldThrowWhenStoreHasNoPersistExtension(t *testing.T) {
	s := xstore.CreateStore(storePersist1CounterConfig())

	// JS: the returned promise rejects. The Go contract documents a panic;
	// either form carries the message.
	var rejection error
	msg := panicMessage(func() {
		_, rejection = xstore.RehydrateStore(s).Wait()
	})
	if rejection != nil {
		msg = rejection.Error()
	}
	assert.Contains(t, msg, "rehydrateStore: store does not have a persist extension")
}

// ---- persist - clearStorage ----

// JS: persist - clearStorage > should remove persisted data from storage
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L923
func TestStorePersist_ClearStorage_ShouldRemovePersistedDataFromStorage(t *testing.T) {
	storage := newMemoryStorage()
	s := storePersist1NewCounter(xstore.PersistOptions[storePersist1Counter]{Name: "test", Storage: storage})

	s.Trigger("inc")
	assert.NotNil(t, getItem(t, storage, "test"))

	xstore.ClearStorage(s)
	assert.Nil(t, getItem(t, storage, "test"))
}

// JS: persist - clearStorage > should throw when store has no persist extension
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L937
func TestStorePersist_ClearStorage_ShouldThrowWhenStoreHasNoPersistExtension(t *testing.T) {
	s := xstore.CreateStore(storePersist1CounterConfig())

	assert.Contains(t,
		panicMessage(func() { xstore.ClearStorage(s) }),
		"clearStorage: store does not have a persist extension")
}

// JS: persist - clearStorage > should return a promise for async storage removals
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L948
func TestStorePersist_ClearStorage_ShouldReturnAPromiseForAsyncStorageRemovals(t *testing.T) {
	storage := newAsyncMemoryStorage()
	s := storePersist1NewCounter(xstore.PersistOptions[storePersist1Counter]{Name: "test", Storage: storage})

	s.Trigger("inc")
	sleep(1) // await Promise.resolve()

	_, err := xstore.ClearStorage(s).Wait()
	require.NoError(t, err)

	assert.Nil(t, getItem(t, storage, "test"))
}

// ---- persist - createJSONStorage ----

// JS: persist - createJSONStorage > should return noop storage when getStorage throws
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L965
func TestStorePersist_CreateJSONStorage_ShouldReturnNoopStorageWhenGetStorageThrows(t *testing.T) {
	storage := xstore.CreateJSONStorage(func() xstore.Storage {
		panic(errors.New("no localStorage"))
	})

	v, p := storage.GetItem("test")
	assert.Nil(t, v)
	assert.Nil(t, p)
	assert.NotPanics(t, func() { storage.SetItem("test", "value") })
	assert.NotPanics(t, func() { storage.RemoveItem("test") })
}

// JS: persist - createJSONStorage > should wrap a working storage adapter
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L975
func TestStorePersist_CreateJSONStorage_ShouldWrapAWorkingStorageAdapter(t *testing.T) {
	mockStorage := newMemoryStorage()
	storage := xstore.CreateJSONStorage(func() xstore.Storage { return mockStorage })

	storage.SetItem("foo", "bar")
	got := getItem(t, storage, "foo")
	require.NotNil(t, got)
	assert.Equal(t, "bar", *got)

	storage.RemoveItem("foo")
	assert.Nil(t, getItem(t, storage, "foo"))
}

// JS: persist - createJSONStorage > should preserve async storage semantics
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L986
func TestStorePersist_CreateJSONStorage_ShouldPreserveAsyncStorageSemantics(t *testing.T) {
	mockStorage := newAsyncMemoryStorage()
	storage := xstore.CreateJSONStorage(func() xstore.Storage { return mockStorage })

	_, err := storage.SetItem("foo", "bar").Wait()
	require.NoError(t, err)
	got := getItem(t, storage, "foo")
	require.NotNil(t, got)
	assert.Equal(t, "bar", *got)

	_, err = storage.RemoveItem("foo").Wait()
	require.NoError(t, err)
	assert.Nil(t, getItem(t, storage, "foo"))
}

// ---- persist - SSR-safe defaults ----

// JS: persist - SSR-safe defaults > should not require localStorage when using default storage
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L999
func TestStorePersist_SSRSafeDefaults_ShouldNotRequireLocalStorageWhenUsingDefaultStorage(t *testing.T) {
	// JS stubs a throwing globalThis.localStorage; the Go default (nil Storage)
	// is the no-op storage.
	s := storePersist1NewCounter(xstore.PersistOptions[storePersist1Counter]{Name: "test"})

	assert.True(t, xstore.IsHydrated(s))
	assert.NotPanics(t, func() { s.Trigger("inc") })
}

// JS: persist - SSR-safe defaults > should detect persist rehydrate event collisions in development
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L1029
func TestStorePersist_SSRSafeDefaults_ShouldDetectPersistRehydrateEventCollisionsInDevelopment(t *testing.T) {
	storage := newMemoryStorage()

	msg := panicMessage(func() {
		xstore.CreateStore(xstore.StoreConfig[storePersist1Counter]{
			Context: storePersist1Counter{Count: 0},
			On: map[string]xstore.StoreAssigner[storePersist1Counter]{
				"__persist.rehydrate": func(c storePersist1Counter, _ xs.Event, _ *xstore.EnqueueObject[storePersist1Counter]) (storePersist1Counter, bool) {
					return c, true
				},
			},
		}).With(xstore.Persist(xstore.PersistOptions[storePersist1Counter]{Name: "test", Storage: storage}))
	})
	assert.Contains(t, msg,
		`The "persist" store extension uses reserved event type(s): "__persist.rehydrate".`)
}

// ---- persist - async storage auto-detection ----

// JS: persist - async storage auto-detection > should detect async storage and skip sync hydration
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L1046
func TestStorePersist_AsyncStorageAutoDetection_ShouldDetectAsyncStorageAndSkipSyncHydration(t *testing.T) {
	storage := newAsyncMemoryStorage()
	storePersist1Seed(t, storage, "test", map[string]any{
		"context": map[string]any{"count": 50},
		"version": 0,
	})

	s := storePersist1NewCounter(xstore.PersistOptions[storePersist1Counter]{Name: "test", Storage: storage})

	// Should NOT have hydrated (async storage detected)
	assert.Equal(t, 0, s.GetSnapshot().Context.Count)
	assert.False(t, xstore.IsHydrated(s))
}

// ---- persist - composability ----

// JS: persist - composability > should work with undoRedo extension
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L1068
func TestStorePersist_Composability_ShouldWorkWithUndoRedoExtension(t *testing.T) {
	// Import would be needed in real code but this tests the pattern
	storage := newMemoryStorage()
	s := storePersist1NewCounter(xstore.PersistOptions[storePersist1Counter]{Name: "test", Storage: storage})

	s.Trigger("inc")
	s.Trigger("inc")

	stored := storePersist1Stored(t, storage, "test")
	assert.Equal(t, map[string]any{"count": float64(2)}, stored["context"])
	assert.Equal(t, 2, s.GetSnapshot().Context.Count)
}

// JS: persist - composability > should persist and restore correctly across store instances
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L1084
func TestStorePersist_Composability_ShouldPersistAndRestoreCorrectlyAcrossStoreInstances(t *testing.T) {
	type nameCtx struct {
		Count int    `json:"count"`
		Name  string `json:"name"`
	}
	newStore := func(config xstore.PersistOptions[nameCtx]) *xstore.Store[nameCtx] {
		return xstore.CreateStore(xstore.StoreConfig[nameCtx]{
			Context: nameCtx{Count: 0, Name: "test"},
			On: map[string]xstore.StoreAssigner[nameCtx]{
				"inc": func(c nameCtx, _ xs.Event, _ *xstore.EnqueueObject[nameCtx]) (nameCtx, bool) {
					c.Count++
					return c, true
				},
				"setName": func(c nameCtx, e xs.Event, _ *xstore.EnqueueObject[nameCtx]) (nameCtx, bool) {
					c.Name = e.(xs.E)["name"].(string)
					return c, true
				},
			},
		}).With(xstore.Persist(config))
	}
	storage := newMemoryStorage()
	config := xstore.PersistOptions[nameCtx]{Name: "test", Storage: storage}

	// Store 1: write
	store1 := newStore(config)

	store1.Trigger("inc")
	store1.Trigger("inc")
	store1.Trigger("setName", xs.E{"name": "updated"})

	// Store 2: read
	store2 := newStore(config)

	assert.Equal(t, 2, store2.GetSnapshot().Context.Count)
	assert.Equal(t, "updated", store2.GetSnapshot().Context.Name)
}

// ---- persist - strategy: event ----

// JS: persist - strategy: event > should persist events to storage
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L1119
func TestStorePersist_StrategyEvent_ShouldPersistEventsToStorage(t *testing.T) {
	storage := newMemoryStorage()
	s := storePersist1NewCounter(xstore.PersistOptions[storePersist1Counter]{
		Name: "test", Storage: storage, Strategy: xstore.PersistEvent,
	})

	s.Trigger("inc")
	s.Trigger("inc")

	stored := storePersist1Stored(t, storage, "test")
	events, ok := stored["events"].([]any)
	require.True(t, ok)
	assert.Len(t, events, 2)
	assert.Equal(t, "inc", events[0].(map[string]any)["type"])
	assert.Equal(t, float64(0), stored["version"])
}

// JS: persist - strategy: event > should restore state by replaying events
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L1135
func TestStorePersist_StrategyEvent_ShouldRestoreStateByReplayingEvents(t *testing.T) {
	storage := newMemoryStorage()
	opts := xstore.PersistOptions[storePersist1Counter]{Name: "test", Storage: storage, Strategy: xstore.PersistEvent}

	// Store 1: produce events
	store1 := storePersist1NewCounter(opts)

	store1.Trigger("inc")
	store1.Trigger("inc")
	store1.Trigger("inc")

	// Store 2: restore from events
	store2 := storePersist1NewCounter(opts)

	assert.Equal(t, 3, store2.GetSnapshot().Context.Count)
}

// JS: persist - strategy: event > should restore state with event payloads
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L1157
func TestStorePersist_StrategyEvent_ShouldRestoreStateWithEventPayloads(t *testing.T) {
	storage := newMemoryStorage()
	newStore := func() *xstore.Store[storePersist1Counter] {
		return xstore.CreateStore(xstore.StoreConfig[storePersist1Counter]{
			Context: storePersist1Counter{Count: 0},
			On: map[string]xstore.StoreAssigner[storePersist1Counter]{
				"inc": storePersist1Inc,
				"add": func(c storePersist1Counter, e xs.Event, _ *xstore.EnqueueObject[storePersist1Counter]) (storePersist1Counter, bool) {
					return storePersist1Counter{Count: c.Count + storePersist1Int(e.(xs.E)["amount"])}, true
				},
			},
		}).With(xstore.Persist(xstore.PersistOptions[storePersist1Counter]{
			Name: "test", Storage: storage, Strategy: xstore.PersistEvent,
		}))
	}

	store1 := newStore()

	store1.Trigger("inc")
	store1.Trigger("add", xs.E{"amount": 10})

	store2 := newStore()

	assert.Equal(t, 11, store2.GetSnapshot().Context.Count)
}

// JS: persist - strategy: event > should set _persist.hydrated to true on sync hydration
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L1182
func TestStorePersist_StrategyEvent_ShouldSetHydratedToTrueOnSyncHydration(t *testing.T) {
	storage := newMemoryStorage()
	s := storePersist1NewCounter(xstore.PersistOptions[storePersist1Counter]{
		Name: "test", Storage: storage, Strategy: xstore.PersistEvent,
	})

	assert.True(t, xstore.IsHydrated(s))
}

// JS: persist - strategy: event > should respect maxEvents option
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L1192
func TestStorePersist_StrategyEvent_ShouldRespectMaxEventsOption(t *testing.T) {
	storage := newMemoryStorage()
	opts := xstore.PersistOptions[storePersist1Counter]{
		Name: "test", Storage: storage, Strategy: xstore.PersistEvent, MaxEvents: 2,
	}
	s := storePersist1NewCounter(opts)

	s.Trigger("inc") // 1
	s.Trigger("inc") // 2
	s.Trigger("inc") // 3

	stored := storePersist1Stored(t, storage, "test")
	assert.Len(t, stored["events"], 2)
	assert.Equal(t, map[string]any{"count": float64(1)}, stored["checkpoint"])

	// New store replays last 2 events from checkpoint, getting correct total
	store2 := storePersist1NewCounter(opts)

	assert.Equal(t, 3, store2.GetSnapshot().Context.Count)
}

// JS: persist - strategy: event > should not hydrate when skipHydration is true
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L1220
func TestStorePersist_StrategyEvent_ShouldNotHydrateWhenSkipHydrationIsTrue(t *testing.T) {
	storage := newMemoryStorage()
	storePersist1Seed(t, storage, "test", map[string]any{
		"events":  []any{map[string]any{"type": "inc"}, map[string]any{"type": "inc"}},
		"version": 0,
	})

	s := storePersist1NewCounter(xstore.PersistOptions[storePersist1Counter]{
		Name: "test", Storage: storage, Strategy: xstore.PersistEvent, SkipHydration: true,
	})

	assert.Equal(t, 0, s.GetSnapshot().Context.Count)
	assert.False(t, xstore.IsHydrated(s))
}

// JS: persist - strategy: event > should rehydrate from async storage
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L1241
func TestStorePersist_StrategyEvent_ShouldRehydrateFromAsyncStorage(t *testing.T) {
	storage := newAsyncMemoryStorage()
	storePersist1Seed(t, storage, "test", map[string]any{
		"events": []any{
			map[string]any{"type": "inc"},
			map[string]any{"type": "inc"},
			map[string]any{"type": "inc"},
		},
		"version": 0,
	})

	s := storePersist1NewCounter(xstore.PersistOptions[storePersist1Counter]{
		Name: "test", Storage: storage, Strategy: xstore.PersistEvent, SkipHydration: true,
	})

	assert.Equal(t, 0, s.GetSnapshot().Context.Count)

	_, err := xstore.RehydrateStore(s).Wait()
	require.NoError(t, err)

	assert.Equal(t, 3, s.GetSnapshot().Context.Count)
	assert.True(t, xstore.IsHydrated(s))
}

// JS: persist - strategy: event > should continue accumulating events after rehydration
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L1266
func TestStorePersist_StrategyEvent_ShouldContinueAccumulatingEventsAfterRehydration(t *testing.T) {
	storage := newMemoryStorage()
	opts := xstore.PersistOptions[storePersist1Counter]{Name: "test", Storage: storage, Strategy: xstore.PersistEvent}

	// Store 1
	store1 := storePersist1NewCounter(opts)

	store1.Trigger("inc")
	store1.Trigger("inc")

	// Store 2: restore then continue
	store2 := storePersist1NewCounter(opts)

	assert.Equal(t, 2, store2.GetSnapshot().Context.Count)

	store2.Trigger("inc")

	stored := storePersist1Stored(t, storage, "test")
	assert.Len(t, stored["events"], 3)
	assert.Equal(t, 3, store2.GetSnapshot().Context.Count)
}

// storePersist2Ctx mirrors `{ count: number }`.
type storePersist2Ctx struct {
	Count int `json:"count"`
}

// storePersist2Inc mirrors the `inc: (ctx) => ({ count: ctx.count + 1 })` handler.
func storePersist2Inc(c storePersist2Ctx, _ xs.Event, _ *xstore.EnqueueObject[storePersist2Ctx]) (storePersist2Ctx, bool) {
	return storePersist2Ctx{Count: c.Count + 1}, true
}

// storePersist2EventStore mirrors the createStore({context: {count: 0}, on: {inc}})
// shared by the `persist - strategy: event` tests.
func storePersist2EventStore(opts xstore.PersistOptions[storePersist2Ctx]) *xstore.Store[storePersist2Ctx] {
	return xstore.CreateStore(xstore.StoreConfig[storePersist2Ctx]{
		Context: storePersist2Ctx{Count: 0},
		On: map[string]xstore.StoreAssigner[storePersist2Ctx]{
			"inc": storePersist2Inc,
		},
	}).With(xstore.Persist(opts))
}

// storePersist2Await mirrors `await promise` for a persist helper promise
// (nil means already settled).
func storePersist2Await(t *testing.T, p *xstore.Promise[struct{}]) error {
	t.Helper()
	select {
	case <-p.Done():
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for promise")
	}
	_, err := p.Wait()
	return err
}

// storePersist2Unmarshal mirrors JSON.parse(value).
func storePersist2Unmarshal(raw string, out *map[string]any) error {
	return json.Unmarshal([]byte(raw), out)
}

// storePersist2Write mirrors an entry of the JS `writes` array.
type storePersist2Write struct {
	value   string
	resolve func()
	reject  func(err error)
}

// storePersist2Writes mirrors the JS `writes` array shared with an async
// setItem; guarded because persist may write from other goroutines.
type storePersist2Writes struct {
	mu      sync.Mutex
	entries []storePersist2Write
}

func (w *storePersist2Writes) push(e storePersist2Write) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.entries = append(w.entries, e)
}

func (w *storePersist2Writes) Len() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return len(w.entries)
}

func (w *storePersist2Writes) At(i int) storePersist2Write {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.entries[i]
}

// WaitLen mirrors `await vi.waitFor(() => expect(writes).toHaveLength(n))`.
func (w *storePersist2Writes) WaitLen(t *testing.T, n int) {
	t.Helper()
	assert.Eventually(t, func() bool { return w.Len() == n }, 2*time.Second, time.Millisecond)
}

// storePersist2PendingStorage mirrors a StateStorage whose setItem returns a
// promise settled by the test (via writes[i].resolve / reject).
func storePersist2PendingStorage(writes *storePersist2Writes, getItem func() *string, onResolve func(value string)) xstore.Storage {
	return xstore.StorageFuncs{
		GetItemFunc: func(string) (*string, *xstore.Promise[*string]) { return getItem(), nil },
		SetItemFunc: func(_, value string) *xstore.Promise[struct{}] {
			p, resolve, reject := xstore.NewPromise[struct{}]()
			writes.push(storePersist2Write{
				value: value,
				resolve: func() {
					if onResolve != nil {
						onResolve(value)
					}
					resolve(struct{}{})
				},
				reject: reject,
			})
			return p
		},
	}
}

// JS: persist - strategy: event > should migrate events when version differs
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L1293
func TestStorePersist_StrategyEvent_ShouldMigrateEventsWhenVersionDiffers(t *testing.T) {
	storage := newMemoryStorage()
	storage.SetItem("test", toJSON(t, map[string]any{
		"events":  []any{map[string]any{"type": "increment"}, map[string]any{"type": "increment"}},
		"version": 1,
	}))

	s := storePersist2EventStore(xstore.PersistOptions[storePersist2Ctx]{
		Name:     "test",
		Storage:  storage,
		Strategy: xstore.PersistEvent,
		Version:  2,
		MigrateEvents: func(events []xs.Event, _ any) []xs.Event {
			out := make([]xs.Event, 0, len(events))
			for _, e := range events {
				if e.EventType() == "increment" {
					next := xs.E{}
					for k, v := range e.(xs.E) {
						next[k] = v
					}
					next["type"] = "inc"
					out = append(out, next)
				} else {
					out = append(out, e)
				}
			}
			return out
		},
	})

	assert.Equal(t, 2, s.GetSnapshot().Context.Count)
}

// JS: persist - strategy: event > should work with clearStorage
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L1322
func TestStorePersist_StrategyEvent_ShouldWorkWithClearStorage(t *testing.T) {
	storage := newMemoryStorage()
	s := storePersist2EventStore(xstore.PersistOptions[storePersist2Ctx]{
		Name: "test", Storage: storage, Strategy: xstore.PersistEvent,
	})

	s.Trigger("inc")
	assert.NotNil(t, getItem(t, storage, "test"))

	xstore.ClearStorage(s)
	assert.Nil(t, getItem(t, storage, "test"))
}

// JS: persist - strategy: event > should handle empty storage gracefully
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L1336
func TestStorePersist_StrategyEvent_ShouldHandleEmptyStorageGracefully(t *testing.T) {
	storage := newMemoryStorage()
	s := storePersist2EventStore(xstore.PersistOptions[storePersist2Ctx]{
		Name: "test", Storage: storage, Strategy: xstore.PersistEvent,
	})

	assert.Equal(t, 0, s.GetSnapshot().Context.Count)
	assert.True(t, xstore.IsHydrated(s))
}

// JS: persist - strategy: event > should call onError on read failure during hydration
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L1347
func TestStorePersist_StrategyEvent_ShouldCallOnErrorOnReadFailureDuringHydration(t *testing.T) {
	failStorage := xstore.StorageFuncs{
		GetItemFunc: func(string) (*string, *xstore.Promise[*string]) {
			panic(errors.New("read failed"))
		},
		SetItemFunc:    func(_, _ string) *xstore.Promise[struct{}] { return nil },
		RemoveItemFunc: func(string) *xstore.Promise[struct{}] { return nil },
	}
	onError := newSpy()

	storePersist2EventStore(xstore.PersistOptions[storePersist2Ctx]{
		Name:     "test",
		Storage:  failStorage,
		Strategy: xstore.PersistEvent,
		OnError:  func(err any) { onError.Call(err) },
	})

	assert.Positive(t, onError.Count())
}

// JS: persist - strategy: event > should work with throttle
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L1372
func TestStorePersist_StrategyEvent_ShouldWorkWithThrottle(t *testing.T) {
	clock := xs.NewSimulatedClock()
	storage := newMemoryStorage()
	s := storePersist2EventStore(xstore.PersistOptions[storePersist2Ctx]{
		Name: "test", Storage: storage, Strategy: xstore.PersistEvent,
		Throttle: ms(100), Clock: clock,
	})

	s.Trigger("inc")
	s.Trigger("inc")
	s.Trigger("inc")

	assert.Nil(t, getItem(t, storage, "test"))

	clock.Increment(ms(100))

	stored := storedJSON(t, storage, "test").(map[string]any)
	assert.Len(t, stored["events"], 3)
}

// JS: persist - strategy: event > should work with flushStorage
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L1395
func TestStorePersist_StrategyEvent_ShouldWorkWithFlushStorage(t *testing.T) {
	clock := xs.NewSimulatedClock()
	storage := newMemoryStorage()
	s := storePersist2EventStore(xstore.PersistOptions[storePersist2Ctx]{
		Name: "test", Storage: storage, Strategy: xstore.PersistEvent,
		Throttle: ms(1000), Clock: clock,
	})

	s.Trigger("inc")
	s.Trigger("inc")

	assert.Nil(t, getItem(t, storage, "test"))

	xstore.FlushStorage(s)

	stored := storedJSON(t, storage, "test").(map[string]any)
	assert.Len(t, stored["events"], 2)
}

// JS: persist committed %s writes > persists nested events in commit order (throttle %i, %s)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L1421
func TestStorePersist_CommittedWrites_PersistsNestedEventsInCommitOrder(t *testing.T) {
	type valueCtx struct {
		Value int `json:"value"`
	}
	for _, strategy := range []xstore.PersistStrategy{xstore.PersistSnapshot, xstore.PersistEvent} {
		for _, tc := range []struct {
			throttle   int
			nestedFrom string
		}{
			{0, "effect"},
			{100, "effect"},
			{0, "subscriber"},
			{100, "subscriber"},
		} {
			// JS test (shared case definition): https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L1421
			t.Run(fmt.Sprintf("%s/throttle %d, %s", strategy, tc.throttle, tc.nestedFrom), func(t *testing.T) {
				storage := newMemoryStorage()
				nestedFrom := tc.nestedFrom
				makeStore := func() *xstore.Store[valueCtx] {
					opts := xstore.PersistOptions[valueCtx]{
						Name:     "nested",
						Strategy: strategy,
						Storage:  storage,
						Throttle: ms(tc.throttle),
					}
					if strategy == xstore.PersistEvent {
						opts.MaxEvents = 1
					}
					return xstore.CreateStore(xstore.StoreConfig[valueCtx]{
						Context: valueCtx{Value: 0},
						On: map[string]xstore.StoreAssigner[valueCtx]{
							"outer": func(c valueCtx, _ xs.Event, enq *xstore.EnqueueObject[valueCtx]) (valueCtx, bool) {
								if nestedFrom == "effect" {
									enq.Effect(func(e *xstore.StoreEffectEnqueue[valueCtx]) {
										e.Trigger("inner")
									})
								}
								return valueCtx{Value: c.Value*10 + 1}, true
							},
							"inner": func(c valueCtx, _ xs.Event, _ *xstore.EnqueueObject[valueCtx]) (valueCtx, bool) {
								return valueCtx{Value: c.Value*10 + 2}, true
							},
						},
					}).With(xstore.Persist(opts))
				}
				s := makeStore()
				assert.True(t, s.Can("outer"))
				s.Transition(s.GetSnapshot(), xs.Ev("outer"))
				require.NoError(t, storePersist2Await(t, xstore.FlushStorage(s)))
				assert.Nil(t, getItem(t, storage, "nested"))

				subscription := s.SubscribeNext(func(snapshot *xstore.StoreSnapshot[valueCtx]) {
					if nestedFrom == "subscriber" && snapshot.Context.Value == 1 {
						s.Trigger("inner")
					}
				})
				s.Trigger("outer")
				subscription.Unsubscribe()
				assert.Equal(t, 12, s.GetSnapshot().Context.Value)
				require.NoError(t, storePersist2Await(t, xstore.FlushStorage(s)))
				saved := storedJSON(t, storage, "nested").(map[string]any)
				if strategy == xstore.PersistSnapshot {
					assert.Equal(t, float64(12), saved["context"].(map[string]any)["value"])
				} else {
					assert.Equal(t, []any{map[string]any{"type": "inner"}}, saved["events"])
					assert.Equal(t, map[string]any{"value": float64(1)}, saved["checkpoint"])
				}
				assert.Equal(t, 12, makeStore().GetSnapshot().Context.Value)

				require.NoError(t, storePersist2Await(t, xstore.RehydrateStore(s)))
				s.Trigger("inner")
				require.NoError(t, storePersist2Await(t, xstore.FlushStorage(s)))
				assert.Equal(t, 122, makeStore().GetSnapshot().Context.Value)
			})
		}
	}
}

// JS: persist committed %s writes > does not buffer %s evaluations
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L1482
func TestStorePersist_CommittedWrites_DoesNotBufferEvaluations(t *testing.T) {
	for _, strategy := range []xstore.PersistStrategy{xstore.PersistSnapshot, xstore.PersistEvent} {
		for _, evaluation := range []string{"can", "transition", "validation"} {
			// JS test (shared case definition): https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L1482
			t.Run(fmt.Sprintf("%s/%s", strategy, evaluation), func(t *testing.T) {
				clock := xs.NewSimulatedClock()
				storage := newMemoryStorage()
				opts := xstore.PersistOptions[storePersist2Ctx]{
					Name:     "committed",
					Strategy: strategy,
					Storage:  storage,
					Throttle: ms(100),
					Clock:    clock,
				}
				if strategy == xstore.PersistEvent {
					opts.MaxEvents = 1
				}
				base := xstore.CreateStore(xstore.StoreConfig[storePersist2Ctx]{
					Schemas: &xstore.StoreSchemas{
						Context: zObject(map[string]*zSchema{"count": zNumberMax(1)}),
					},
					Context: storePersist2Ctx{Count: 0},
					On: map[string]xstore.StoreAssigner[storePersist2Ctx]{
						"inc": storePersist2Inc,
					},
				}).With(xstore.Persist(opts))
				s := base
				if evaluation == "validation" {
					s = base.With(xstore.ValidateSchemas[storePersist2Ctx]())
				}
				s.Trigger("inc")
				switch evaluation {
				case "can":
					assert.True(t, s.Can("inc"))
				case "transition":
					s.Transition(s.GetSnapshot(), xs.Ev("inc"))
				default:
					r := recovered(func() { s.Trigger("inc") })
					err, ok := r.(error)
					require.True(t, ok, "expected a StoreValidationError panic, got %v", r)
					var ve *xstore.StoreValidationError
					assert.ErrorAs(t, err, &ve)
				}
				clock.Increment(ms(100))
				assert.Equal(t, 1, s.GetSnapshot().Context.Count)
				saved := storedJSON(t, storage, "committed").(map[string]any)
				if strategy == xstore.PersistSnapshot {
					assert.Equal(t, float64(1), saved["context"].(map[string]any)["count"])
				} else {
					assert.Equal(t, []any{map[string]any{"type": "inc"}}, saved["events"])
					checkpoint, present := saved["checkpoint"]
					assert.True(t, present, "checkpoint must be stored as null, not omitted")
					assert.Nil(t, checkpoint)
				}
			})
		}
	}
}

// JS: persist committed %s writes > flushes throttled data after an already pending asynchronous write
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L1525
func TestStorePersist_CommittedWrites_FlushesThrottledDataAfterAlreadyPendingAsynchronousWrite(t *testing.T) {
	for _, strategy := range []xstore.PersistStrategy{xstore.PersistSnapshot, xstore.PersistEvent} {
		// JS test (shared case definition): https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L1525
		t.Run(string(strategy), func(t *testing.T) {
			clock := xs.NewSimulatedClock()
			writes := &storePersist2Writes{}
			storage := storePersist2PendingStorage(writes, func() *string { return nil }, nil)
			s := storePersist2EventStore(xstore.PersistOptions[storePersist2Ctx]{
				Name: "throttled", Storage: storage, Strategy: strategy,
				Throttle: ms(100), Clock: clock,
			})
			s.Trigger("inc")
			clock.Increment(ms(100))
			writes.WaitLen(t, 1)
			s.Trigger("inc")
			flushed := xstore.FlushStorage(s)
			assert.Equal(t, 1, writes.Len())
			writes.At(0).resolve()
			writes.WaitLen(t, 2)
			saved := map[string]any{}
			require.NoError(t, storePersist2Unmarshal(writes.At(1).value, &saved))
			if strategy == xstore.PersistSnapshot {
				assert.Equal(t, float64(2), saved["context"].(map[string]any)["count"])
			} else {
				assert.Len(t, saved["events"], 2)
			}
			writes.At(1).resolve()
			require.NoError(t, storePersist2Await(t, flushed))
			clock.Increment(ms(100))
			sleep(20)
			assert.Equal(t, 2, writes.Len())
		})
	}
}

// JS: persist committed %s writes > orders asynchronous writes and recovers after failure: %s
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L1561
func TestStorePersist_CommittedWrites_OrdersAsynchronousWritesAndRecoversAfterFailure(t *testing.T) {
	for _, strategy := range []xstore.PersistStrategy{xstore.PersistSnapshot, xstore.PersistEvent} {
		for _, failFirst := range []bool{false, true} {
			// JS test (shared case definition): https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/persist.test.ts#L1561
			t.Run(fmt.Sprintf("%s/%t", strategy, failFirst), func(t *testing.T) {
				writes := &storePersist2Writes{}
				var mu sync.Mutex
				var saved *string
				onDone := newSpy()
				onError := newSpy()
				storage := storePersist2PendingStorage(writes,
					func() *string {
						mu.Lock()
						defer mu.Unlock()
						return saved
					},
					func(value string) {
						mu.Lock()
						defer mu.Unlock()
						saved = &value
					})
				s := storePersist2EventStore(xstore.PersistOptions[storePersist2Ctx]{
					Name: "ordered", Strategy: strategy, Storage: storage,
					OnDone:  func(data any) { onDone.Call(data) },
					OnError: func(err any) { onError.Call(err) },
				})
				s.Trigger("inc")
				s.Trigger("inc")
				// A fast second write cannot overtake the first: it has not started yet.
				assert.Equal(t, 1, writes.Len())
				flushed := xstore.FlushStorage(s)
				assert.NotNil(t, flushed)
				completed := newSpy()
				go func() {
					flushed.Wait()
					completed.Call()
				}()
				if failFirst {
					writes.At(0).reject(errors.New("write failed"))
				} else {
					writes.At(0).resolve()
				}
				writes.WaitLen(t, 2)
				assert.Equal(t, 0, completed.Count())
				writes.At(1).resolve()
				require.NoError(t, storePersist2Await(t, flushed))
				mu.Lock()
				require.NotNil(t, saved)
				raw := *saved
				mu.Unlock()
				value := map[string]any{}
				require.NoError(t, storePersist2Unmarshal(raw, &value))
				if strategy == xstore.PersistSnapshot {
					assert.Equal(t, float64(2), value["context"].(map[string]any)["count"])
				} else {
					assert.Len(t, value["events"], 2)
				}
				wantErrors, wantDone := 0, 2
				if failFirst {
					wantErrors, wantDone = 1, 1
				}
				assert.Equal(t, wantErrors, onError.Count())
				assert.Equal(t, wantDone, onDone.Count())
			})
		}
	}
}
