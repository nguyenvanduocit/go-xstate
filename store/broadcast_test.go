package store_test

import (
	"encoding/json"
	"sync"
	"testing"

	xstore "github.com/nguyenvanduocit/go-xstate/store"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// storeBroadcast1Registry mirrors the JS module-level `channels` map; each
// test owns one (JS clears it in beforeEach/afterEach).
type storeBroadcast1Registry struct {
	mu       sync.Mutex
	channels map[string]map[*storeBroadcast1Channel]struct{}
}

func storeBroadcast1NewRegistry(t *testing.T) *storeBroadcast1Registry {
	r := &storeBroadcast1Registry{channels: map[string]map[*storeBroadcast1Channel]struct{}{}}
	t.Cleanup(func() {
		r.mu.Lock()
		defer r.mu.Unlock()
		r.channels = map[string]map[*storeBroadcast1Channel]struct{}{}
	})
	return r
}

// New mirrors `new MockBroadcastChannel(name)`; it is also the NewChannel
// factory that replaces the mocked global BroadcastChannel.
func (r *storeBroadcast1Registry) New(name string) xstore.BroadcastChannel {
	return r.newChannel(name)
}

func (r *storeBroadcast1Registry) newChannel(name string) *storeBroadcast1Channel {
	c := &storeBroadcast1Channel{registry: r, name: name, listeners: map[int]func(any){}}
	r.mu.Lock()
	defer r.mu.Unlock()
	entries, ok := r.channels[name]
	if !ok {
		entries = map[*storeBroadcast1Channel]struct{}{}
		r.channels[name] = entries
	}
	entries[c] = struct{}{}
	return c
}

// storeBroadcast1Channel mirrors MockBroadcastChannel and implements
// xstore.BroadcastChannel.
type storeBroadcast1Channel struct {
	registry  *storeBroadcast1Registry
	name      string
	mu        sync.Mutex
	nextID    int
	listeners map[int]func(any)
}

func (c *storeBroadcast1Channel) PostMessage(data any) {
	c.registry.mu.Lock()
	var targets []*storeBroadcast1Channel
	for ch := range c.registry.channels[c.name] {
		if ch != c {
			targets = append(targets, ch)
		}
	}
	c.registry.mu.Unlock()

	for _, ch := range targets {
		ch.mu.Lock()
		listeners := make([]func(any), 0, len(ch.listeners))
		for _, l := range ch.listeners {
			listeners = append(listeners, l)
		}
		ch.mu.Unlock()
		for _, l := range listeners {
			l(data)
		}
	}
}

// OnMessage mirrors addEventListener('message', ...); the returned
// subscription mirrors removeEventListener.
func (c *storeBroadcast1Channel) OnMessage(fn func(data any)) xs.Subscription {
	c.mu.Lock()
	defer c.mu.Unlock()
	id := c.nextID
	c.nextID++
	c.listeners[id] = fn
	return xs.SubscriptionFunc(func() {
		c.mu.Lock()
		defer c.mu.Unlock()
		delete(c.listeners, id)
	})
}

func (c *storeBroadcast1Channel) Close() {
	c.registry.mu.Lock()
	delete(c.registry.channels[c.name], c)
	c.registry.mu.Unlock()
	c.mu.Lock()
	c.listeners = map[int]func(any){}
	c.mu.Unlock()
}

// storeBroadcast1Ctx mirrors the `{ count: number }` counter context.
type storeBroadcast1Ctx struct {
	Count int `json:"count"`
}

// storeBroadcast1CounterStore mirrors createCounterStore(storage).
func storeBroadcast1CounterStore(storage xstore.Storage) *xstore.Store[storeBroadcast1Ctx] {
	return xstore.CreateStore(xstore.StoreConfig[storeBroadcast1Ctx]{
		Context: storeBroadcast1Ctx{Count: 0},
		On: map[string]xstore.StoreAssigner[storeBroadcast1Ctx]{
			"inc": func(c storeBroadcast1Ctx, _ xs.Event, _ *xstore.EnqueueObject[storeBroadcast1Ctx]) (storeBroadcast1Ctx, bool) {
				return storeBroadcast1Ctx{Count: c.Count + 1}, true
			},
		},
	}).With(xstore.Persist(xstore.PersistOptions[storeBroadcast1Ctx]{Name: "counter", Storage: storage}))
}

// storeBroadcast1WaitForMicrotask mirrors waitForMicrotask() (setTimeout 0).
func storeBroadcast1WaitForMicrotask() { sleep(20) }

// JS: broadcast storage > broadcasts queued persisted writes only after each async write completes
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/broadcast.test.ts#L98
func TestStoreBroadcast_BroadcastsQueuedPersistedWritesOnlyAfterEachAsyncWriteCompletes(t *testing.T) {
	registry := storeBroadcast1NewRegistry(t)

	var mu sync.Mutex
	var completions []func()
	var saved *string
	completionCount := func() int {
		mu.Lock()
		defer mu.Unlock()
		return len(completions)
	}
	baseStorage := xstore.StorageFuncs{
		GetItemFunc: func(string) (*string, *xstore.Promise[*string]) {
			mu.Lock()
			defer mu.Unlock()
			return saved, nil
		},
		RemoveItemFunc: func(string) *xstore.Promise[struct{}] { return nil },
		SetItemFunc: func(_ string, value string) *xstore.Promise[struct{}] {
			p, resolve, _ := xstore.NewPromise[struct{}]()
			mu.Lock()
			defer mu.Unlock()
			completions = append(completions, func() {
				mu.Lock()
				saved = &value
				mu.Unlock()
				resolve(struct{}{})
			})
			return p
		},
	}
	storage := xstore.CreateBroadcastStorage(baseStorage, xstore.BroadcastStorageOptions{NewChannel: registry.New})
	receiver := registry.newChannel("xstate-store")
	var counts []int
	receiver.OnMessage(func(any) {
		mu.Lock()
		raw := *saved
		mu.Unlock()
		var parsed struct {
			Context struct {
				Count int `json:"count"`
			} `json:"context"`
		}
		require.NoError(t, json.Unmarshal([]byte(raw), &parsed))
		mu.Lock()
		counts = append(counts, parsed.Context.Count)
		mu.Unlock()
	})
	getCounts := func() []int {
		mu.Lock()
		defer mu.Unlock()
		return append([]int(nil), counts...)
	}
	complete := func(i int) {
		mu.Lock()
		fn := completions[i]
		mu.Unlock()
		fn()
	}

	store := storeBroadcast1CounterStore(storage)
	store.Trigger("inc")
	store.Trigger("inc")
	assert.Equal(t, 1, completionCount())
	assert.Empty(t, getCounts())
	flushed := xstore.FlushStorage(store)
	complete(0)
	storeBroadcast1WaitForMicrotask()
	assert.Equal(t, []int{1}, getCounts())
	require.Equal(t, 2, completionCount())
	complete(1)
	_, err := flushed.Wait()
	require.NoError(t, err)
	assert.Equal(t, []int{1, 2}, getCounts())
}

// JS: broadcast storage > broadcasts writes to other storage adapters on the same channel
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/broadcast.test.ts#L133
func TestStoreBroadcast_BroadcastsWritesToOtherStorageAdaptersOnTheSameChannel(t *testing.T) {
	registry := storeBroadcast1NewRegistry(t)

	baseStorage := newMemoryStorage()
	storage := xstore.CreateBroadcastStorage(baseStorage, xstore.BroadcastStorageOptions{NewChannel: registry.New})
	receiver := registry.newChannel("xstate-store")
	var mu sync.Mutex
	var messages []any

	receiver.OnMessage(func(data any) {
		mu.Lock()
		defer mu.Unlock()
		messages = append(messages, data)
	})

	storage.SetItem("counter", toJSON(t, map[string]any{"context": map[string]any{"count": 1}}))

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, []any{
		xstore.BroadcastMessage{Type: "xstate-store-update", Name: "counter"},
	}, messages)
}

// JS: broadcast storage > rehydrates subscribed stores when another tab writes persisted state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/broadcast.test.ts#L150
func TestStoreBroadcast_RehydratesSubscribedStoresWhenAnotherTabWritesPersistedState(t *testing.T) {
	registry := storeBroadcast1NewRegistry(t)

	baseStorage := newMemoryStorage()
	opts := xstore.BroadcastStorageOptions{NewChannel: registry.New}
	storage1 := xstore.CreateBroadcastStorage(baseStorage, opts)
	storage2 := xstore.CreateBroadcastStorage(baseStorage, opts)
	store1 := storeBroadcast1CounterStore(storage1)
	store2 := storeBroadcast1CounterStore(storage2)
	unsubscribe := xstore.SubscribeToBroadcastStorage(store2)

	store1.Trigger("inc")
	storeBroadcast1WaitForMicrotask()

	assert.Equal(t, 1, store1.GetSnapshot().Context.Count)
	assert.Equal(t, 1, store2.GetSnapshot().Context.Count)

	unsubscribe()
}

// JS: broadcast storage > does not rehydrate from unrelated storage names
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/broadcast.test.ts#L167
func TestStoreBroadcast_DoesNotRehydrateFromUnrelatedStorageNames(t *testing.T) {
	registry := storeBroadcast1NewRegistry(t)

	baseStorage := newMemoryStorage()
	storage := xstore.CreateBroadcastStorage(baseStorage, xstore.BroadcastStorageOptions{NewChannel: registry.New})
	store := storeBroadcast1CounterStore(storage)
	unsubscribe := xstore.SubscribeToBroadcastStorage(store)
	sender := registry.newChannel("xstate-store")

	baseStorage.SetItem("other", toJSON(t, map[string]any{
		"context": map[string]any{"count": 100},
		"version": 0,
	}))
	sender.PostMessage(xstore.BroadcastMessage{Type: "xstate-store-update", Name: "other"})
	storeBroadcast1WaitForMicrotask()

	assert.Equal(t, 0, store.GetSnapshot().Context.Count)

	unsubscribe()
	sender.Close()
}

// JS: broadcast storage > throws when subscribing a store without broadcast storage
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/broadcast.test.ts#L187
func TestStoreBroadcast_ThrowsWhenSubscribingAStoreWithoutBroadcastStorage(t *testing.T) {
	storeBroadcast1NewRegistry(t)

	store := storeBroadcast1CounterStore(newMemoryStorage())

	msg := panicMessage(func() { xstore.SubscribeToBroadcastStorage(store) })
	assert.Contains(t, msg, "subscribeToBroadcastStorage: store storage must be wrapped with createBroadcastStorage()")
}
