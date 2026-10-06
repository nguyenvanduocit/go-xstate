package xstate_test

import (
	"sync"
	"testing"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/require"
)

type pointerContext struct {
	Next  *pointerContext
	Alias *pointerContext
	Nil   *pointerContext
	Bag   map[string]any
	Items []any
	Array [1]*pointerContext
	Child xs.ActorRef // Visit back edges before finding the reference to replace.
}

// Go pointer-context regression; JS persistence replaces context actor refs with markers.
// Related JS spawned-ref persistence test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actorLogic.test.ts#L1038
func TestPersistPointerContextRestoresChild(t *testing.T) {
	child := xs.FromTransition(func(n int, _ xs.Event, _ *xs.ActorScope) int { return n + 1 }, nil)
	machine := xs.CreateMachine(xs.MachineConfig[*pointerContext]{ContextFn: func(a xs.ContextArgs) *pointerContext {
		nested := &pointerContext{Child: a.Spawn("child", xs.SpawnOptions{ID: "child"})}
		return &pointerContext{Next: nested, Alias: nested, Array: [1]*pointerContext{nested}, Bag: map[string]any{"nested": nested}, Items: []any{nested}}
	}}, xs.Implementations{Actors: map[string]xs.ActorLogic{"child": child}})
	original := xs.CreateActor(machine).Start()
	old := original.GetSnapshot().Context
	oldChild := old.Next.Child
	persisted := original.GetPersistedSnapshot()
	require.NotSame(t, old, persisted.(map[string]any)["context"])
	require.Same(t, oldChild, old.Next.Child, "persistence must not mutate live context")
	original.Stop()
	restored := xs.CreateActor(machine, xs.WithSnapshot(persisted)).Start()
	defer restored.Stop()
	ctx := restored.GetSnapshot().Context
	newChild := restored.GetSnapshot().Children["child"]
	require.Same(t, newChild, ctx.Next.Child)
	require.Same(t, ctx.Next, ctx.Alias)
	require.Same(t, ctx.Next, ctx.Array[0])
	require.Same(t, ctx.Next, ctx.Bag["nested"])
	require.Same(t, ctx.Next, ctx.Items[0])
	require.Nil(t, ctx.Nil)
	require.Same(t, oldChild, old.Next.Child, "restoration must not mutate original context")
	ctx.Next.Child.Send(xs.Ev("inc"))
	require.Equal(t, 1, xs.As[*xs.TransitionSnapshot[int]](newChild).GetSnapshot().Context)
	again := xs.CreateActor(machine, xs.WithSnapshot(persisted)).Start()
	defer again.Stop()
	require.NotSame(t, newChild, again.GetSnapshot().Context.Next.Child, "persisted markers must survive repeated restores")
}

// Go regression: in-memory context graphs may contain pointer, map, and slice cycles.
// Related JS spawned-ref persistence test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actorLogic.test.ts#L1038
func TestPersistPointerContextCycles(t *testing.T) {
	child := xs.FromTransition(func(n int, _ xs.Event, _ *xs.ActorScope) int { return n + 1 }, nil)
	machine := xs.CreateMachine(xs.MachineConfig[*pointerContext]{ContextFn: func(a xs.ContextArgs) *pointerContext {
		c := &pointerContext{Child: a.Spawn("child", xs.SpawnOptions{ID: "child"})}
		c.Next = c
		c.Bag = map[string]any{"root": c}
		c.Bag["self"] = c.Bag
		c.Items = make([]any, 2)
		c.Items[0] = c.Items
		c.Items[1] = c
		return c
	}}, xs.Implementations{Actors: map[string]xs.ActorLogic{"child": child}})
	original := xs.CreateActor(machine).Start()
	defer original.Stop()
	persisted := original.GetPersistedSnapshot()
	restored := xs.CreateActor(machine, xs.WithSnapshot(persisted)).Start()
	defer restored.Stop()
	ctx := restored.GetSnapshot().Context
	require.True(t, original.GetSnapshot().Context != ctx, "restore must copy the cyclic context")
	require.Same(t, ctx, ctx.Next)
	require.Same(t, ctx, ctx.Bag["root"])
	require.Same(t, ctx, ctx.Bag["self"].(map[string]any)["root"])
	require.Same(t, ctx, ctx.Items[0].([]any)[1])
	require.Same(t, restored.GetSnapshot().Children["child"], ctx.Child)
}

// Go regression: resources outside the replaced actor-ref subgraph retain identity.
// Related JS spawned-ref persistence test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actorLogic.test.ts#L1038
func TestPersistPointerContextPreservesUnchangedResources(t *testing.T) {
	type context struct {
		Child xs.ActorRef
		Lock  *sync.Mutex
		Data  *pointerContext
	}
	lock := new(sync.Mutex)
	data := &pointerContext{}
	data.Next = data
	machine := xs.CreateMachine(xs.MachineConfig[context]{ContextFn: func(a xs.ContextArgs) context {
		return context{Child: a.Spawn("child", xs.SpawnOptions{ID: "child"}), Lock: lock, Data: data}
	}}, xs.Implementations{Actors: map[string]xs.ActorLogic{"child": xs.FromTransition(func(n int, _ xs.Event, _ *xs.ActorScope) int { return n }, nil)}})
	original := xs.CreateActor(machine).Start()
	defer original.Stop()
	lock.Lock()
	persisted := original.GetPersistedSnapshot()
	lock.Unlock()
	ctx := persisted.(map[string]any)["context"].(context)
	require.True(t, ctx.Lock == lock, "persistence must retain the unrelated mutex")
	require.True(t, ctx.Data == data, "persistence must retain an unchanged cyclic subgraph")
	require.True(t, ctx.Lock.TryLock(), "unlocking the original must unlock the persisted mutex")
	ctx.Lock.Unlock()
	restored := xs.CreateActor(machine, xs.WithSnapshot(persisted)).Start()
	defer restored.Stop()
	ctx = restored.GetSnapshot().Context
	require.Same(t, restored.GetSnapshot().Children["child"], ctx.Child)
	require.True(t, ctx.Lock == lock, "revival must retain the unrelated mutex")
	require.True(t, ctx.Data == data, "revival must retain an unchanged cyclic subgraph")
}
