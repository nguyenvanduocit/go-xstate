package store_test

import (
	"encoding/json"
	"testing"
	"time"

	xstore "github.com/nguyenvanduocit/go-xstate/store"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/assert"
)

// storeFromStore1Ctx mirrors the `{ count }` context of every test.
type storeFromStore1Ctx struct {
	Count int `json:"count"`
}

func TestStoreFromStore_RestoresPersistedSnapshot(t *testing.T) {
	logic := xstore.FromStore(xstore.StoreConfig[storeFromStore1Ctx]{
		Context: storeFromStore1Ctx{Count: 42},
		On: map[string]xstore.StoreAssigner[storeFromStore1Ctx]{
			"inc": func(ctx storeFromStore1Ctx, _ xs.Event, _ *xstore.EnqueueObject[storeFromStore1Ctx]) (storeFromStore1Ctx, bool) {
				ctx.Count++
				return ctx, true
			},
		},
	})
	original := xs.CreateActor(logic).Start()
	defer original.Stop()
	original.Send(xs.Ev("inc"))
	persisted := original.GetPersistedSnapshot()
	encoded, err := json.Marshal(persisted)
	if err != nil {
		t.Fatal(err)
	}
	var decoded any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name     string
		snapshot any
	}{
		{"typed", persisted},
		{"JSON decoded", decoded},
	} {
		t.Run(test.name, func(t *testing.T) {
			restored := xs.CreateActor(logic, xs.WithSnapshot(test.snapshot)).Start()
			defer restored.Stop()
			snapshot := restored.GetSnapshot()
			if snapshot.Status != xs.StatusActive || snapshot.Context.Count != 43 {
				t.Fatalf("restored snapshot = %#v, want active with count 43", snapshot)
			}
			restored.Send(xs.Ev("inc"))
			if got := restored.GetSnapshot().Context.Count; got != 44 {
				t.Fatalf("count after transition = %d, want 44", got)
			}
			if got := original.GetSnapshot().Context.Count; got != 43 {
				t.Fatalf("original count changed to %d, want 43", got)
			}
		})
	}
}

func TestStoreFromStore_RestoresJSONPersistedChild(t *testing.T) {
	logic := xstore.FromStore(xstore.StoreConfig[storeFromStore1Ctx]{
		Context: storeFromStore1Ctx{Count: 42},
		On: map[string]xstore.StoreAssigner[storeFromStore1Ctx]{
			"inc": func(ctx storeFromStore1Ctx, _ xs.Event, _ *xstore.EnqueueObject[storeFromStore1Ctx]) (storeFromStore1Ctx, bool) {
				ctx.Count++
				return ctx, true
			},
		},
	})
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Invoke: []xs.InvokeConfig{{ID: "store", Src: "store"}},
	}, xs.Implementations{Actors: map[string]xs.ActorLogic{"store": logic}})
	original := xs.CreateActor(machine).Start()
	defer original.Stop()
	original.GetSnapshot().Children["store"].Send(xs.Ev("inc"))
	encoded, err := json.Marshal(original.GetPersistedSnapshot())
	if err != nil {
		t.Fatal(err)
	}
	var decoded any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	restored := xs.CreateActor(machine, xs.WithSnapshot(decoded)).Start()
	defer restored.Stop()
	child := restored.GetSnapshot().Children["store"]
	if child == nil {
		t.Fatal("restored child is missing")
	}
	if snapshot := child.AnySnapshot().(*xstore.StoreSnapshot[storeFromStore1Ctx]); snapshot.Status != xs.StatusActive || snapshot.Context.Count != 43 {
		t.Fatalf("restored child = %#v, want active with count 43", snapshot)
	}
	child.Send(xs.Ev("inc"))
	if count := child.AnySnapshot().(*xstore.StoreSnapshot[storeFromStore1Ctx]).Context.Count; count != 44 {
		t.Fatalf("child count after transition = %d, want 44", count)
	}
}

// fromStore.test.ts: fromStore > creates an actor from store logic with input
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/fromStore.test.ts#L7
func TestStoreFromStore_CreatesAnActorFromStoreLogicWithInput(t *testing.T) {
	storeLogic := xstore.FromStore(xstore.StoreConfig[storeFromStore1Ctx]{
		ContextFn: func(input any) storeFromStore1Ctx {
			return storeFromStore1Ctx{Count: input.(int)}
		},
		On: map[string]xstore.StoreAssigner[storeFromStore1Ctx]{
			"inc": func(ctx storeFromStore1Ctx, ev xs.Event, _ *xstore.EnqueueObject[storeFromStore1Ctx]) (storeFromStore1Ctx, bool) {
				ctx.Count += ev.(xs.E)["by"].(int)
				return ctx, true
			},
		},
	})

	actor := xs.CreateActor(storeLogic, xs.WithInput(42))

	actor.Start()

	actor.Send(xs.E{"type": "inc", "by": 8})

	assert.Equal(t, 50, actor.GetSnapshot().Context.Count)
}

// fromStore.test.ts: fromStore > emits events
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/fromStore.test.ts#L31
func TestStoreFromStore_EmitsEvents(t *testing.T) {
	spy := newSpy()

	storeLogic := xstore.FromStore(xstore.StoreConfig[storeFromStore1Ctx]{
		ContextFn: func(input any) storeFromStore1Ctx {
			return storeFromStore1Ctx{Count: input.(int)}
		},
		Schemas: &xstore.StoreSchemas{
			Emitted: map[string]xstore.Schema{
				"increased": zObject(map[string]*zSchema{"upBy": zNumber()}),
			},
		},
		On: map[string]xstore.StoreAssigner[storeFromStore1Ctx]{
			"inc": func(ctx storeFromStore1Ctx, ev xs.Event, enq *xstore.EnqueueObject[storeFromStore1Ctx]) (storeFromStore1Ctx, bool) {
				by := ev.(xs.E)["by"].(int)
				enq.Emit("increased", xs.E{"upBy": by})
				ctx.Count += by
				return ctx, true
			},
		},
	})

	actor := xs.CreateActor(storeLogic, xs.WithInput(42))

	actor.On("increased", func(e xs.Event) { spy.Call(e) })

	actor.Start()

	actor.Send(xs.E{"type": "inc", "by": 8})

	assert.Equal(t, 50, actor.GetSnapshot().Context.Count)
	assert.Equal(t, 1, spy.Count())
	assert.Equal(t, [][]any{{xs.E{"type": "increased", "upBy": 8}}}, spy.Calls())
}

// fromStore.test.ts: fromStore > enq.getSnapshot() in a sync effect reflects the post-transition state (matches createStore)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/fromStore.test.ts#L67
func TestStoreFromStore_EnqGetSnapshotInASyncEffectReflectsThePostTransitionStateMatchesCreateStore(t *testing.T) {
	var seen *int

	storeLogic := xstore.FromStore(xstore.StoreConfig[storeFromStore1Ctx]{
		ContextFn: func(any) storeFromStore1Ctx { return storeFromStore1Ctx{Count: 0} },
		On: map[string]xstore.StoreAssigner[storeFromStore1Ctx]{
			"inc": func(ctx storeFromStore1Ctx, _ xs.Event, enq *xstore.EnqueueObject[storeFromStore1Ctx]) (storeFromStore1Ctx, bool) {
				enq.Effect(func(e *xstore.StoreEffectEnqueue[storeFromStore1Ctx]) {
					v := e.GetSnapshot().Context.Count
					seen = &v
				})
				ctx.Count++
				return ctx, true
			},
		},
	})

	actor := xs.CreateActor(storeLogic).Start()
	actor.Send(xs.Ev("inc"))

	if assert.NotNil(t, seen) {
		assert.Equal(t, 1, *seen)
	}
}

// fromStore.test.ts: fromStore > enq.getSnapshot() in an async effect reflects the latest committed state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/fromStore.test.ts#L90
func TestStoreFromStore_EnqGetSnapshotInAnAsyncEffectReflectsTheLatestCommittedState(t *testing.T) {
	release := make(chan struct{})
	seen := make(chan int, 1)

	storeLogic := xstore.FromStore(xstore.StoreConfig[storeFromStore1Ctx]{
		ContextFn: func(any) storeFromStore1Ctx { return storeFromStore1Ctx{Count: 0} },
		On: map[string]xstore.StoreAssigner[storeFromStore1Ctx]{
			"start": func(ctx storeFromStore1Ctx, _ xs.Event, enq *xstore.EnqueueObject[storeFromStore1Ctx]) (storeFromStore1Ctx, bool) {
				enq.Effect(func(e *xstore.StoreEffectEnqueue[storeFromStore1Ctx]) {
					go func() {
						<-release
						seen <- e.GetSnapshot().Context.Count
					}()
				})
				ctx.Count++
				return ctx, true
			},
			"bump": func(ctx storeFromStore1Ctx, _ xs.Event, _ *xstore.EnqueueObject[storeFromStore1Ctx]) (storeFromStore1Ctx, bool) {
				return storeFromStore1Ctx{Count: ctx.Count + 10}, true
			},
		},
	})

	actor := xs.CreateActor(storeLogic).Start()
	defer actor.Stop()
	actor.Send(xs.Ev("start")) // count -> 1
	actor.Send(xs.Ev("bump"))  // count -> 11 before the async effect resumes
	close(release)

	select {
	case count := <-seen:
		assert.Equal(t, 11, count)
	case <-time.After(2 * time.Second):
		t.Fatal("async effect did not report the latest committed state")
	}
}
