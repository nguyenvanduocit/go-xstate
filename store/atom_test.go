package store_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	xstore "github.com/nguyenvanduocit/go-xstate/store"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/assert"
)

// storeAtom1WaitUntil polls cond until it holds (stands in for the JS
// `await Promise.resolve()` / `await setTimeout(...)` used to let async work
// settle). It fails the test after 2s.
func storeAtom1WaitUntil(t testing.TB, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for condition")
		}
		time.Sleep(time.Millisecond)
	}
}

// storeAtom1Undefined stands for a JS `throw undefined`: Go has no
// "undefined" panic value, so a sentinel is thrown and compared by value.
type storeAtom1Undefined struct{}

// storeAtom1SortedJoin mirrors Array.from(items).sort().join(',').
func storeAtom1SortedJoin(items []int) string {
	sorted := slices.Clone(items)
	slices.Sort(sorted)
	parts := make([]string, len(sorted))
	for i, v := range sorted {
		parts[i] = strconv.Itoa(v)
	}
	return strings.Join(parts, ",")
}

// JS: creates an atom
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/atom.test.ts#L9
func TestStoreAtom_CreatesAnAtom(t *testing.T) {
	atom := xstore.CreateAtom(42)

	assert.Equal(t, 42, atom.Get())
}

// JS: creates an atom from atom config
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/atom.test.ts#L15
func TestStoreAtom_CreatesAnAtomFromAtomConfig(t *testing.T) {
	config := xstore.CreateAtomConfig(42)
	atom := config.CreateAtom()
	otherAtom := config.CreateAtom()

	atom.Set(100)

	assert.Equal(t, 100, atom.Get())
	assert.Equal(t, 42, otherAtom.Get())
}

// JS: creates an atom from atom config and input
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/atom.test.ts#L26
func TestStoreAtom_CreatesAnAtomFromAtomConfigAndInput(t *testing.T) {
	type input struct{ InitialCount int }

	config := xstore.CreateAtomConfigFunc(func(in input) int {
		return in.InitialCount
	})
	atom := config.CreateAtom(input{InitialCount: 10})

	assert.Equal(t, 10, atom.Get())
}

// JS: sets the value of the atom using a function
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/atom.test.ts#L35
func TestStoreAtom_SetsTheValueOfTheAtomUsingAFunction(t *testing.T) {
	atom := xstore.CreateAtom(0)

	atom.Update(func(prev int) int { return prev + 1 })

	assert.Equal(t, 1, atom.Get())

	atom.Update(func(prev int) int { return prev + 1 })

	assert.Equal(t, 2, atom.Get())
}

// JS: does not subscribe a writable atom to reads inside its updater
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/atom.test.ts#L47
func TestStoreAtom_DoesNotSubscribeAWritableAtomToReadsInsideItsUpdater(t *testing.T) {
	source := xstore.CreateAtom(1)
	target := xstore.CreateAtom(10)
	observer := newSpy()
	subscription := target.SubscribeNext(func(v int) { observer.Call(v) })

	target.Update(func(previous int) int { return previous + source.Get() })
	source.Set(2)

	assert.Equal(t, 11, target.Get())
	assert.Equal(t, [][]any{{11}}, observer.Calls())
	subscription.Unsubscribe()
}

// JS: drains notifications before rethrowing the first subscriber error
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/atom.test.ts#L61
func TestStoreAtom_DrainsNotificationsBeforeRethrowingTheFirstSubscriberError(t *testing.T) {
	source := xstore.CreateAtom(0)
	unrelated := xstore.CreateAtom(0)
	subscriberErr := errors.New("subscriber failed")
	first := source.SubscribeNext(func(int) {
		panic(subscriberErr)
	})
	observer := newSpy()
	second := source.SubscribeNext(func(v int) { observer.Call(v) })
	other := unrelated.SubscribeNext(func(int) {})

	assert.PanicsWithValue(t, subscriberErr, func() { source.Set(1) })
	assert.Equal(t, [][]any{{1}}, observer.Calls())
	unrelated.Set(1)
	assert.Equal(t, [][]any{{1}}, observer.Calls())
	assert.PanicsWithValue(t, subscriberErr, func() { source.Set(2) })
	assert.Equal(t, [][]any{{1}, {2}}, observer.Calls())

	first.Unsubscribe()
	second.Unsubscribe()
	other.Unsubscribe()
}

// JS: rethrows undefined and still delivers reentrant notifications
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/atom.test.ts#L84
func TestStoreAtom_RethrowsUndefinedAndStillDeliversReentrantNotifications(t *testing.T) {
	source := xstore.CreateAtom(0)
	nested := xstore.CreateAtom(0)
	order := []string{}
	subscriptions := []xs.Subscription{
		nested.SubscribeNext(func(int) {
			order = append(order, "nested")
			panic(storeAtom1Undefined{})
		}),
		nested.SubscribeNext(func(int) {
			order = append(order, "nested-second")
			panic(errors.New("later error"))
		}),
		source.SubscribeNext(func(int) {
			order = append(order, "first")
			nested.Set(1)
		}),
		source.SubscribeNext(func(int) {
			order = append(order, "second")
		}),
	}
	didThrow := false
	var thrown any
	func() {
		defer func() {
			if r := recover(); r != nil {
				didThrow = true
				thrown = r
			}
		}()
		source.Set(1)
	}()
	assert.True(t, didThrow)
	assert.Equal(t, storeAtom1Undefined{}, thrown)
	assert.Equal(t, []string{"first", "second", "nested", "nested-second"}, order)
	for _, subscription := range subscriptions {
		subscription.Unsubscribe()
	}
}

// JS: can set the value to undefined
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/atom.test.ts#L117
func TestStoreAtom_CanSetTheValueToUndefined(t *testing.T) {
	// JS `number | undefined`: nil stands for undefined.
	atom := xstore.CreateAtom[any](1)
	assert.Equal(t, 1, atom.Get())

	atom.Set(nil)
	assert.Nil(t, atom.Get())
}

// JS: can subscribe to atom changes
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/atom.test.ts#L125
func TestStoreAtom_CanSubscribeToAtomChanges(t *testing.T) {
	log := newSpy()
	atom := xstore.CreateAtom(0)

	atom.SubscribeNext(func(v int) { log.Call(v) })

	atom.Set(1)

	assert.Contains(t, log.Calls(), []any{1})

	atom.Set(2)

	assert.Contains(t, log.Calls(), []any{2})
}

// JS: can unsubscribe from atom changes
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/atom.test.ts#L140
func TestStoreAtom_CanUnsubscribeFromAtomChanges(t *testing.T) {
	log := newSpy()
	atom := xstore.CreateAtom(0)

	sub := atom.SubscribeNext(func(v int) { log.Call(v) })

	atom.Set(1)

	assert.Contains(t, log.Calls(), []any{1})

	sub.Unsubscribe()

	atom.Set(2)

	assert.Equal(t, 1, log.Count())
}

// JS: can create a combined atom
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/atom.test.ts#L157
func TestStoreAtom_CanCreateACombinedAtom(t *testing.T) {
	nameAtom := xstore.CreateAtom("a")
	numAtom := xstore.CreateAtom(3)
	combinedAtom := xstore.CreateComputedAtom(func(prev *string) string {
		return strings.Repeat(nameAtom.Get(), numAtom.Get())
	})

	assert.Equal(t, "aaa", combinedAtom.Get())

	nameAtom.Set("b")

	assert.Equal(t, "bbb", combinedAtom.Get())

	numAtom.Set(5)
}

// JS: allows updates to a computed dependency during a subscription callback
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/atom.test.ts#L171
func TestStoreAtom_AllowsUpdatesToAComputedDependencyDuringASubscriptionCallback(t *testing.T) {
	type abc struct{ A, B, C int }

	// Pointers keep Object.is semantics: every JS `{...ctx}` is a new object,
	// so every Update below is a change.
	atom := xstore.CreateAtom(&abc{})

	a0 := xstore.CreateComputedAtom(func(prev *int) int { return atom.Get().A })
	a1 := xstore.CreateComputedAtom(func(prev *int) int { return a0.Get() })
	a1.SubscribeNext(func(int) {
		atom.Update(func(ctx *abc) *abc { next := *ctx; next.B = ctx.A; return &next })
	})

	b0 := xstore.CreateComputedAtom(func(prev *int) int { return atom.Get().B })
	b1 := xstore.CreateComputedAtom(func(prev *int) int { return b0.Get() })
	b1.SubscribeNext(func(int) {
		atom.Update(func(ctx *abc) *abc { next := *ctx; next.C = ctx.B; return &next })
	})

	incA := func(ctx *abc) *abc { next := *ctx; next.A = ctx.A + 1; return &next }
	atom.Update(incA)
	atom.Update(incA)
	atom.Update(incA)

	assert.Equal(t, 3, a0.Get())
	assert.Equal(t, 3, a1.Get())
	assert.Equal(t, 3, b0.Get())
	assert.Equal(t, 3, b1.Get())
	assert.Equal(t, 3, atom.Get().A)
	assert.Equal(t, 3, atom.Get().B)
	assert.Equal(t, 3, atom.Get().C)
}

// JS: does not loop when updating a computed dependency which affects an atoms own state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/atom.test.ts#L195
func TestStoreAtom_DoesNotLoopWhenUpdatingAComputedDependencyWhichAffectsAnAtomsOwnState(t *testing.T) {
	count := xstore.CreateAtom(0)
	a := xstore.CreateComputedAtom(func(prev *int) int { return count.Get() })

	a.SubscribeNext(func(int) { count.Update(func(val int) int { return val + 1 }) })

	count.Update(func(val int) int { return val + 1 })

	assert.Equal(t, 2, a.Get())
	assert.Equal(t, 2, count.Get())
}

// JS: works with a mix of atoms and stores
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/atom.test.ts#L207
func TestStoreAtom_WorksWithAMixOfAtomsAndStores(t *testing.T) {
	type ctx struct{ Name string }

	countAtom := xstore.CreateAtom(0)
	store := xstore.CreateStore(xstore.StoreConfig[ctx]{
		Context: ctx{Name: "David"},
		On: map[string]xstore.StoreAssigner[ctx]{
			"nameUpdated": func(c ctx, ev xs.Event, enq *xstore.EnqueueObject[ctx]) (ctx, bool) {
				return ctx{Name: ev.(xs.E)["name"].(string)}, true
			},
		},
	})

	log := newSpy()

	combinedAtom := xstore.CreateComputedAtom(func(prev *string) string {
		return store.Get().Context.Name + fmt.Sprintf(" %d", countAtom.Get())
	})

	combinedAtom.SubscribeNext(func(v string) { log.Call(v) })

	assert.Equal(t, "David 0", combinedAtom.Get())

	store.Send(xs.E{"type": "nameUpdated", "name": "John"})

	assert.Equal(t, "John 0", combinedAtom.Get())

	countAtom.Set(1)

	assert.Equal(t, 2, log.Count())
	assert.Contains(t, log.Calls(), []any{"John 1"})
	assert.Equal(t, 1, countAtom.Get())
	assert.Equal(t, "John 1", combinedAtom.Get())
}

// JS: works with stores
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/atom.test.ts#L241
func TestStoreAtom_WorksWithStores(t *testing.T) {
	type nameCtx struct{ Name string }
	type countCtx struct{ Count int }

	nameStore := xstore.CreateStore(xstore.StoreConfig[nameCtx]{
		Context: nameCtx{Name: "David"},
		On: map[string]xstore.StoreAssigner[nameCtx]{
			"nameUpdated": func(c nameCtx, ev xs.Event, enq *xstore.EnqueueObject[nameCtx]) (nameCtx, bool) {
				return nameCtx{Name: ev.(xs.E)["name"].(string)}, true
			},
		},
	})

	countStore := xstore.CreateStore(xstore.StoreConfig[countCtx]{
		Context: countCtx{Count: 0},
		On: map[string]xstore.StoreAssigner[countCtx]{
			"increment": func(c countCtx, ev xs.Event, enq *xstore.EnqueueObject[countCtx]) (countCtx, bool) {
				return countCtx{Count: c.Count + 1}, true
			},
		},
	})

	combinedAtom := xstore.CreateComputedAtom(func(prev *string) string {
		return nameStore.Get().Context.Name + fmt.Sprintf(" %d", countStore.Get().Context.Count)
	})

	assert.Equal(t, "David 0", combinedAtom.Get())

	nameStore.Trigger("nameUpdated", xs.E{"name": "John"})

	assert.Equal(t, "John 0", combinedAtom.Get())

	countStore.Trigger("increment")

	assert.Equal(t, "John 1", combinedAtom.Get())
}

// JS: works with selectors
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/atom.test.ts#L274
func TestStoreAtom_WorksWithSelectors(t *testing.T) {
	type ctx struct {
		Name  string
		Count int
	}

	store := xstore.CreateStore(xstore.StoreConfig[ctx]{
		Context: ctx{Name: "David", Count: 0},
		On: map[string]xstore.StoreAssigner[ctx]{
			"increment": func(c ctx, ev xs.Event, enq *xstore.EnqueueObject[ctx]) (ctx, bool) {
				c.Count++
				return c, true
			},
		},
	})

	count := xstore.Select(store, func(c ctx) int { return c.Count })

	combinedAtom := xstore.CreateComputedAtom(func(prev *int) int { return 2 * count.Get() })

	assert.Equal(t, 0, combinedAtom.Get())

	store.Trigger("increment")

	assert.Equal(t, 2, combinedAtom.Get())
}

// JS: allows sending events to the store during a selector subscription
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/atom.test.ts#L293
func TestStoreAtom_AllowsSendingEventsToTheStoreDuringASelectorSubscription(t *testing.T) {
	type ctx struct{ A, B int }

	store := xstore.CreateStore(xstore.StoreConfig[ctx]{
		Context: ctx{A: 0, B: 0},
		On: map[string]xstore.StoreAssigner[ctx]{
			"a": func(c ctx, ev xs.Event, enq *xstore.EnqueueObject[ctx]) (ctx, bool) {
				c.A++
				return c, true
			},
			"b": func(c ctx, ev xs.Event, enq *xstore.EnqueueObject[ctx]) (ctx, bool) {
				c.B++
				return c, true
			},
		},
	})

	a := xstore.Select(store, func(c ctx) int { return c.A })
	a.SubscribeNext(func(int) { store.Trigger("b") })

	store.Trigger("a")
	store.Trigger("a")
	store.Trigger("a")

	assert.Equal(t, 3, store.Get().Context.A)
	assert.Equal(t, 3, store.Get().Context.B)
}

// JS: works with selectors (get API)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/atom.test.ts#L313
func TestStoreAtom_WorksWithSelectorsGetAPI(t *testing.T) {
	type ctx struct {
		Name  string
		Count int
	}

	store := xstore.CreateStore(xstore.StoreConfig[ctx]{
		Context: ctx{Name: "David", Count: 0},
		On: map[string]xstore.StoreAssigner[ctx]{
			"increment": func(c ctx, ev xs.Event, enq *xstore.EnqueueObject[ctx]) (ctx, bool) {
				c.Count++
				return c, true
			},
		},
	})

	count := xstore.Select(store, func(c ctx) int { return c.Count })

	combinedAtom := xstore.CreateComputedAtom(func(prev *int) int { return 2 * count.Get() })

	assert.Equal(t, 0, combinedAtom.Get())

	store.Trigger("increment")

	assert.Equal(t, 2, combinedAtom.Get())
}

// JS: combined atoms should be read-only
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/atom.test.ts#L332
func TestStoreAtom_CombinedAtomsShouldBeReadOnly(t *testing.T) {
	atom1 := xstore.CreateAtom(0)
	atom2 := xstore.CreateAtom(1)
	combinedAtom := xstore.CreateComputedAtom(func(prev *int) int { return atom1.Get() + atom2.Get() })

	assert.Equal(t, 1, combinedAtom.Get())

	// JS: `combinedAtom.set?.(2)` (a type error; a no-op at runtime). In Go the
	// ReadonlyAtom has no Set method, so the optional call is skipped; if the
	// implementation does expose one, call it and expect it to change nothing.
	if set := reflect.ValueOf(combinedAtom).MethodByName("Set"); set.IsValid() {
		set.Call([]reflect.Value{reflect.ValueOf(2)})
	}

	assert.Equal(t, 1, combinedAtom.Get())
}

// JS: combined atom getters accept only prev as an argument
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/atom.test.ts#L345
func TestStoreAtom_CombinedAtomGettersAcceptOnlyPrevAsAnArgument(t *testing.T) {
	t.Skip("N/A: type-level only — checks (@ts-expect-error) that a two-arg (read, prev) getter signature is rejected; CreateComputedAtom takes only prev by signature")
}

// JS: conditionally read atoms are properly read in combined atoms
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/atom.test.ts#L357
func TestStoreAtom_ConditionallyReadAtomsAreProperlyReadInCombinedAtoms(t *testing.T) {
	atom1 := xstore.CreateAtom(true)
	atom2 := xstore.CreateAtom(false)
	activatorAtom := xstore.CreateAtom("inactive")
	combinedAtom := xstore.CreateComputedAtom(func(prev *bool) bool {
		if activatorAtom.Get() == "active" {
			return atom1.Get()
		}
		return atom2.Get()
	})

	assert.Equal(t, false, combinedAtom.Get())

	activatorAtom.Set("active")

	assert.Equal(t, true, combinedAtom.Get())

	activatorAtom.Set("inactive")

	assert.Equal(t, false, combinedAtom.Get())
}

// JS: conditionally read atoms are properly unsubscribed when no longer needed
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/atom.test.ts#L376
func TestStoreAtom_ConditionallyReadAtomsAreProperlyUnsubscribedWhenNoLongerNeeded(t *testing.T) {
	atom1 := xstore.CreateAtom(true)
	activatorAtom := xstore.CreateAtom("active")
	combinedAtom := xstore.CreateComputedAtom(func(prev *any) any {
		if activatorAtom.Get() == "active" {
			return atom1.Get()
		}
		// JS `{}`: a fresh object on every computation.
		return map[string]any{}
	})

	vals := []any{}

	combinedAtom.SubscribeNext(func(val any) {
		vals = append(vals, val)
	})

	assert.Equal(t, []any{}, vals)

	atom1.Set(false)

	assert.Equal(t, []any{false}, vals)

	atom1.Set(true)

	assert.Equal(t, []any{false, true}, vals)

	activatorAtom.Set("inactive")

	// From here, atom1 should no longer be subscribed to
	// Without the unsubscribe logic, this would be [false, true, {}, {}, ...]

	assert.Equal(t, []any{false, true, map[string]any{}}, vals)

	atom1.Set(false)

	assert.Equal(t, []any{false, true, map[string]any{}}, vals)

	atom1.Set(true)

	assert.Equal(t, []any{false, true, map[string]any{}}, vals)

	// Subscribing again should cause atom1 to be subscribed again

	activatorAtom.Set("active")

	assert.Equal(t, []any{false, true, map[string]any{}, true}, vals)

	atom1.Set(false)

	assert.Equal(t, []any{false, true, map[string]any{}, true, false}, vals)
}

// JS: handles diamond dependencies with single update
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/atom.test.ts#L425
func TestStoreAtom_HandlesDiamondDependenciesWithSingleUpdate(t *testing.T) {
	log := newSpy()
	sourceAtom := xstore.CreateAtom(1)

	pathA := xstore.CreateComputedAtom(func(prev *int) int { return sourceAtom.Get() * 2 })
	pathB := xstore.CreateComputedAtom(func(prev *int) int { return sourceAtom.Get() * 3 })

	bottomAtom := xstore.CreateComputedAtom(func(prev *int) int { return pathA.Get() + pathB.Get() })

	bottomAtom.SubscribeNext(func(x int) {
		log.Call(x)
	})

	// Initial value: (1 * 2) + (1 * 3) = 5
	assert.Equal(t, 5, bottomAtom.Get())
	assert.Equal(t, 0, log.Count())

	// Update source: (2 * 2) + (2 * 3) = 10
	sourceAtom.Set(2)

	result := bottomAtom.Get()

	assert.Equal(t, 10, result)

	// Without proper diamond problem handling, log might be called multiple times
	// as the update propagates through both paths
	assert.Equal(t, 1, log.Count())
	assert.Contains(t, log.Calls(), []any{10})
}

// JS: handles complex diamond dependencies correctly
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/atom.test.ts#L455
func TestStoreAtom_HandlesComplexDiamondDependenciesCorrectly(t *testing.T) {
	log := newSpy()

	// Base atom D
	atomD := xstore.CreateAtom(1)

	// Level 1 - C depends on D
	atomC := xstore.CreateComputedAtom(func(prev *int) int { return atomD.Get() * 2 })

	// Level 2 - B depends on C and D
	atomB := xstore.CreateComputedAtom(func(prev *int) int { return atomC.Get() + atomD.Get() })

	// Level 3 - A depends on B, C, and D
	atomA := xstore.CreateComputedAtom(func(prev *int) int { return atomB.Get() + atomC.Get() + atomD.Get() })

	atomA.SubscribeNext(func(v int) { log.Call(v) })

	// Initial computation:
	// D = 1
	// C = D * 2 = 2
	// B = C + D = 3
	// A = B + C + D = 6
	assert.Equal(t, 6, atomA.Get())
	assert.Equal(t, 0, log.Count())

	// Update base atom D
	atomD.Set(2)

	// After update:
	// D = 2
	// C = D * 2 = 4
	// B = C + D = 6
	// A = B + C + D = 12
	assert.Equal(t, 12, atomA.Get())

	// Should only trigger one update despite multiple dependency paths
	assert.Equal(t, 1, log.Count())
	assert.Contains(t, log.Calls(), []any{12})

	// Verify intermediate values
	assert.Equal(t, 6, atomB.Get())
	assert.Equal(t, 4, atomC.Get())
	assert.Equal(t, 2, atomD.Get())
}

// JS: supports custom equality functions through compare option
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/atom.test.ts#L500
func TestStoreAtom_SupportsCustomEqualityFunctionsThroughCompareOption(t *testing.T) {
	type coord struct{ X, Y int }

	log := newSpy()

	coordAtom := xstore.CreateAtom(
		coord{X: 0, Y: 0},
		xstore.AtomOptions[coord]{
			Compare: func(prev, next coord) bool { return prev.X == next.X && prev.Y == next.Y },
		},
	)

	coordAtom.SubscribeNext(func(c coord) { log.Call(c) })

	// Initial value
	assert.Equal(t, coord{X: 0, Y: 0}, coordAtom.Get())
	assert.Equal(t, 0, log.Count())

	// Setting same values shouldn't trigger update
	coordAtom.Set(coord{X: 0, Y: 0})
	assert.Equal(t, 0, log.Count())

	// Different x value should trigger update
	coordAtom.Set(coord{X: 1, Y: 0})
	assert.Equal(t, 1, log.Count())
	assert.Contains(t, log.Calls(), []any{coord{X: 1, Y: 0}})

	// Different y value should trigger update
	coordAtom.Set(coord{X: 1, Y: 2})
	assert.Equal(t, 2, log.Count())
	calls := log.Calls()
	assert.Equal(t, []any{coord{X: 1, Y: 2}}, calls[len(calls)-1])

	// Setting same values should not trigger update
	coordAtom.Set(coord{X: 1, Y: 2})
	assert.Equal(t, 2, log.Count())
}

// JS: uses Object.is as default equality function
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/atom.test.ts#L535
func TestStoreAtom_UsesObjectIsAsDefaultEqualityFunction(t *testing.T) {
	type obj struct{ Value int }

	log := newSpy()
	// Pointers compare by identity, like JS objects under Object.is.
	objAtom := xstore.CreateAtom(&obj{Value: 0})

	objAtom.SubscribeNext(func(o *obj) { log.Call(o) })

	// Initial value
	assert.Equal(t, &obj{Value: 0}, objAtom.Get())
	assert.Equal(t, 0, log.Count())

	// Setting with same shape but new object should trigger update
	objAtom.Set(&obj{Value: 0})
	assert.Equal(t, 1, log.Count())

	// Setting with same object reference shouldn't trigger update
	o := &obj{Value: 1}
	objAtom.Set(o)
	assert.Equal(t, 2, log.Count())
	objAtom.Set(o)
	assert.Equal(t, 2, log.Count())
}

// JS: Atom-specific properties should not be exposed
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/atom.test.ts#L557
func TestStoreAtom_AtomSpecificPropertiesShouldNotBeExposed(t *testing.T) {
	t.Skip("N/A: type-level only — every statement is a bare property access under @ts-expect-error (_subs, _subsTail, _snapshot, _flags, _deps, _depsTail); there are no runtime expectations")
}

// JS: computed atoms can use their previous value in the getter
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/atom.test.ts#L607
func TestStoreAtom_ComputedAtomsCanUseTheirPreviousValueInTheGetter(t *testing.T) {
	count := xstore.CreateAtom(1)
	accumulated := xstore.CreateComputedAtom(func(prev *int) int {
		p := 0
		if prev != nil {
			p = *prev
		}
		return count.Get() + p
	})

	assert.Equal(t, 1, accumulated.Get()) // 0 + 1 = 1

	count.Set(2)
	assert.Equal(t, 3, accumulated.Get()) // 1 + 2 = 3

	count.Set(3)
	assert.Equal(t, 6, accumulated.Get()) // 3 + 3 = 6
}

// JS: reducer atoms > updates from current state and sent event
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/atom.test.ts#L621
func TestStoreAtom_ReducerAtoms_UpdatesFromCurrentStateAndSentEvent(t *testing.T) {
	counter := xstore.CreateReducerAtom(0, func(state int, event int) int {
		return min(10, state+event)
	})

	counter.Send(13)

	assert.Equal(t, 10, counter.Get())
}

// JS: reducer atoms > can receive arbitrary event values
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/atom.test.ts#L631
func TestStoreAtom_ReducerAtoms_CanReceiveArbitraryEventValues(t *testing.T) {
	// JS `string | number` event: any, concatenated like JS `state + event`.
	value := xstore.CreateReducerAtom("", func(state string, event any) string {
		return state + fmt.Sprint(event)
	})

	value.Send("x")
	value.Send(1)

	assert.Equal(t, "x1", value.Get())
}

// JS: reducer atoms > notifies subscribers when the reducer changes state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/atom.test.ts#L643
func TestStoreAtom_ReducerAtoms_NotifiesSubscribersWhenTheReducerChangesState(t *testing.T) {
	counter := xstore.CreateReducerAtom(0, func(state int, event int) int {
		return state + event
	})
	listener := newSpy()

	counter.SubscribeNext(func(v int) { listener.Call(v) })
	counter.Send(1)
	counter.Send(0)
	counter.Send(2)

	assert.Equal(t, 2, listener.Count())
	calls := listener.Calls()
	assert.Equal(t, []any{1}, calls[0])
	assert.Equal(t, []any{3}, calls[1])
}

// JS: reducer atoms > can be used by derived atoms
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/atom.test.ts#L660
func TestStoreAtom_ReducerAtoms_CanBeUsedByDerivedAtoms(t *testing.T) {
	counter := xstore.CreateReducerAtom(0, func(state int, event int) int {
		return state + event
	})
	doubled := xstore.CreateComputedAtom(func(prev *int) int { return counter.Get() * 2 })

	assert.Equal(t, 0, doubled.Get())

	counter.Send(2)

	assert.Equal(t, 4, doubled.Get())
}

// JS: reducer atoms > does not track atom reads inside the reducer
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/atom.test.ts#L674
func TestStoreAtom_ReducerAtoms_DoesNotTrackAtomReadsInsideTheReducer(t *testing.T) {
	multiplier := xstore.CreateAtom(2)
	counter := xstore.CreateReducerAtom(1, func(state int, event int) int {
		return state + event*multiplier.Get()
	})
	listener := newSpy()

	counter.SubscribeNext(func(v int) { listener.Call(v) })
	counter.Send(3)
	multiplier.Set(10)

	assert.Equal(t, 7, counter.Get())
	assert.Equal(t, 1, listener.Count())
}

// JS: async atoms > should recompute after a dependency changes following %s settlement
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/atom.test.ts#L692
func TestStoreAtom_AsyncAtoms_ShouldRecomputeAfterADependencyChangesFollowingSettlement(t *testing.T) {
	// JS it.each(['done', 'error'])
	for _, status := range []xstore.AsyncStatus{xstore.AsyncDone, xstore.AsyncError} {
		// JS test (shared case definition): https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/atom.test.ts#L692
		t.Run(string(status), func(t *testing.T) {
			count := xstore.CreateAtom(1)
			initialErr := errors.New("initial failure")
			// gate: JS settles the second run in a later microtask, after the
			// synchronous `pending` assertion; the getter goroutine waits for it
			// so that assertion does not depend on scheduler timing.
			gate := newSignal()
			defer gate.Resolve()
			atom := xstore.CreateAsyncAtom(func(ctx context.Context) (int, error) {
				value := count.Get()
				if value != 1 {
					<-gate.ch
				}
				if status == xstore.AsyncError && value == 1 {
					return 0, initialErr
				}
				return value * 2, nil
			})
			selected := xstore.CreateComputedAtom(func(prev *any) any {
				state := atom.Get()
				if state.Status == xstore.AsyncDone {
					return state.Data
				}
				return string(state.Status)
			})
			observer := newSpy()
			subscription := selected.SubscribeNext(func(v any) { observer.Call(v) })

			storeAtom1WaitUntil(t, func() bool { return atom.Get().Status != xstore.AsyncPending })
			sleep(20) // let the settlement notification drain
			if status == xstore.AsyncDone {
				assert.Equal(t, xstore.AsyncAtomState[int]{Status: status, Data: 2}, atom.Get())
			} else {
				assert.Equal(t, xstore.AsyncAtomState[int]{Status: status, Error: initialErr}, atom.Get())
			}
			observer.Reset()

			count.Set(2)
			assert.Equal(t, xstore.AsyncAtomState[int]{Status: xstore.AsyncPending}, atom.Get())
			gate.Resolve()
			storeAtom1WaitUntil(t, func() bool { return atom.Get().Status != xstore.AsyncPending })
			sleep(20)

			assert.Equal(t, xstore.AsyncAtomState[int]{Status: xstore.AsyncDone, Data: 4}, atom.Get())
			assert.Equal(t, [][]any{{"pending"}, {4}}, observer.Calls())
			subscription.Unsubscribe()
		})
	}
}

// JS: async atoms > should recompute lazily after a settled dependency changes
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/atom.test.ts#L727
func TestStoreAtom_AsyncAtoms_ShouldRecomputeLazilyAfterASettledDependencyChanges(t *testing.T) {
	count := xstore.CreateAtom(1)
	getter := newSpy()
	atom := xstore.CreateAsyncAtom(func(ctx context.Context) (int, error) {
		getter.Call()
		return count.Get() * 2, nil
	})

	assert.Equal(t, 0, getter.Count())
	assert.Equal(t, xstore.AsyncAtomState[int]{Status: xstore.AsyncPending}, atom.Get())
	storeAtom1WaitUntil(t, func() bool { return atom.Get().Status == xstore.AsyncDone })
	assert.Equal(t, xstore.AsyncAtomState[int]{Status: xstore.AsyncDone, Data: 2}, atom.Get())

	count.Set(2)
	assert.Equal(t, 1, getter.Count())
	assert.Equal(t, xstore.AsyncAtomState[int]{Status: xstore.AsyncPending}, atom.Get())
	storeAtom1WaitUntil(t, func() bool { return atom.Get().Status == xstore.AsyncDone })
	assert.Equal(t, xstore.AsyncAtomState[int]{Status: xstore.AsyncDone, Data: 4}, atom.Get())
	assert.Equal(t, 2, getter.Count())
}

// JS: async atoms > should retain dependencies when the async comparator suppresses a result
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/atom.test.ts#L745
func TestStoreAtom_AsyncAtoms_ShouldRetainDependenciesWhenTheAsyncComparatorSuppressesAResult(t *testing.T) {
	count := xstore.CreateAtom(1)
	atom := xstore.CreateAsyncAtom(
		func(ctx context.Context) (int, error) { return count.Get() % 2, nil },
		xstore.AtomOptions[xstore.AsyncAtomState[int]]{
			Compare: func(previous, next xstore.AsyncAtomState[int]) bool {
				return next.Status == xstore.AsyncPending ||
					(previous.Status == xstore.AsyncDone &&
						next.Status == xstore.AsyncDone &&
						previous.Data == next.Data)
			},
		},
	)
	observer := newSpy()
	subscription := atom.SubscribeNext(func(s xstore.AsyncAtomState[int]) { observer.Call(s) })

	storeAtom1WaitUntil(t, func() bool { return atom.Get().Status == xstore.AsyncDone })
	assert.Equal(t, xstore.AsyncAtomState[int]{Status: xstore.AsyncDone, Data: 1}, atom.Get())
	sleep(20) // let the settlement notification drain
	observer.Reset()

	count.Set(3)
	sleep(50)
	assert.Equal(t, 0, observer.Count())

	count.Set(4)
	storeAtom1WaitUntil(t, func() bool {
		return atom.Get() == xstore.AsyncAtomState[int]{Status: xstore.AsyncDone, Data: 0}
	})
	sleep(20)
	assert.Equal(t, xstore.AsyncAtomState[int]{Status: xstore.AsyncDone, Data: 0}, atom.Get())
	assert.Equal(t, [][]any{{xstore.AsyncAtomState[int]{Status: xstore.AsyncDone, Data: 0}}}, observer.Calls())
	subscription.Unsubscribe()
}

// JS: async atoms > async atoms should work (fulfilled)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/atom.test.ts#L775
func TestStoreAtom_AsyncAtoms_AsyncAtomsShouldWorkFulfilled(t *testing.T) {
	atom := xstore.CreateAsyncAtom(func(ctx context.Context) (string, error) { return "hello", nil })

	assert.Equal(t, xstore.AsyncAtomState[string]{Status: xstore.AsyncPending}, atom.Get())

	storeAtom1WaitUntil(t, func() bool { return atom.Get().Status != xstore.AsyncPending })

	assert.Equal(t, xstore.AsyncAtomState[string]{Status: xstore.AsyncDone, Data: "hello"}, atom.Get())
}

// JS: async atoms > async atoms should work (rejected)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/atom.test.ts#L785
func TestStoreAtom_AsyncAtoms_AsyncAtomsShouldWorkRejected(t *testing.T) {
	atom := xstore.CreateAsyncAtom(func(ctx context.Context) (string, error) {
		return "", errors.New("test")
	})

	assert.Equal(t, xstore.AsyncAtomState[string]{Status: xstore.AsyncPending}, atom.Get())

	storeAtom1WaitUntil(t, func() bool { return atom.Get().Status != xstore.AsyncPending })

	state := atom.Get()
	assert.Equal(t, xstore.AsyncError, state.Status)
	assert.Error(t, state.Error) // expect.any(Error)
}

// JS: async atoms > should only call getValue once for multiple concurrent reads
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/atom.test.ts#L800
func TestStoreAtom_AsyncAtoms_ShouldOnlyCallGetValueOnceForMultipleConcurrentReads(t *testing.T) {
	var getValueCallCount atomic.Int32
	const resolvedValue = "test-value"

	// gate: JS settles in a later microtask, after both synchronous `pending`
	// reads; the getter goroutine waits for it so those reads do not depend on
	// scheduler timing.
	gate := newSignal()
	defer gate.Resolve()
	myAsyncAtom := xstore.CreateAsyncAtom(func(ctx context.Context) (string, error) {
		getValueCallCount.Add(1)
		<-gate.ch
		return resolvedValue, nil
	})

	// Initial reads should show pending status
	assert.Equal(t, xstore.AsyncAtomState[string]{Status: xstore.AsyncPending}, myAsyncAtom.Get())
	assert.Equal(t, xstore.AsyncAtomState[string]{Status: xstore.AsyncPending}, myAsyncAtom.Get())
	gate.Resolve()

	// Let the promise resolve
	storeAtom1WaitUntil(t, func() bool { return myAsyncAtom.Get().Status != xstore.AsyncPending })

	// Both reads should now show the resolved value
	assert.Equal(t, xstore.AsyncAtomState[string]{Status: xstore.AsyncDone, Data: resolvedValue}, myAsyncAtom.Get())
	assert.Equal(t, xstore.AsyncAtomState[string]{Status: xstore.AsyncDone, Data: resolvedValue}, myAsyncAtom.Get())

	// getValue should have only been called once
	assert.Equal(t, int32(1), getValueCallCount.Load())

	// Additional reads after resolution should still use cached value
	assert.Equal(t, xstore.AsyncAtomState[string]{Status: xstore.AsyncDone, Data: resolvedValue}, myAsyncAtom.Get())
	assert.Equal(t, int32(1), getValueCallCount.Load())
}

// JS: async atoms > should only call getValue once even when error occurs
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/atom.test.ts#L837
func TestStoreAtom_AsyncAtoms_ShouldOnlyCallGetValueOnceEvenWhenErrorOccurs(t *testing.T) {
	var getValueCallCount atomic.Int32
	const errorMessage = "test error"

	// gate: JS settles in a later microtask, after both synchronous `pending`
	// reads; the getter goroutine waits for it so those reads do not depend on
	// scheduler timing.
	gate := newSignal()
	defer gate.Resolve()
	myAsyncAtom := xstore.CreateAsyncAtom(func(ctx context.Context) (string, error) {
		getValueCallCount.Add(1)
		<-gate.ch
		return "", errors.New(errorMessage)
	})

	// Initial reads should show pending status
	assert.Equal(t, xstore.AsyncAtomState[string]{Status: xstore.AsyncPending}, myAsyncAtom.Get())
	assert.Equal(t, xstore.AsyncAtomState[string]{Status: xstore.AsyncPending}, myAsyncAtom.Get())
	gate.Resolve()

	// Let the promise reject
	storeAtom1WaitUntil(t, func() bool { return myAsyncAtom.Get().Status != xstore.AsyncPending })

	// Both reads should now show the error
	for i := 0; i < 2; i++ {
		state := myAsyncAtom.Get()
		assert.Equal(t, xstore.AsyncError, state.Status)
		assert.Error(t, state.Error) // expect.any(Error)
	}

	// getValue should have only been called once
	assert.Equal(t, int32(1), getValueCallCount.Load())

	// Additional reads after rejection should still use cached error
	state := myAsyncAtom.Get()
	assert.Equal(t, xstore.AsyncError, state.Status)
	assert.Error(t, state.Error)
	assert.Equal(t, int32(1), getValueCallCount.Load())
}

// JS: async atoms > async atoms should not have a .set() method
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/atom.test.ts#L874
func TestStoreAtom_AsyncAtoms_AsyncAtomsShouldNotHaveASetMethod(t *testing.T) {
	atom := xstore.CreateAsyncAtom(func(ctx context.Context) (string, error) { return "hello", nil })

	_, hasSet := reflect.TypeOf(atom).MethodByName("Set")

	assert.False(t, hasSet)
}

// JS: async atoms > should pass an abort signal to async atoms
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/atom.test.ts#L880
func TestStoreAtom_AsyncAtoms_ShouldPassAnAbortSignalToAsyncAtoms(t *testing.T) {
	// The getter runs on its own goroutine, so the context it receives is
	// reported back instead of asserting inside the goroutine.
	got := make(chan context.Context, 1)
	atom := xstore.CreateAsyncAtom(func(ctx context.Context) (string, error) {
		got <- ctx
		return "hello", nil
	})

	assert.Equal(t, xstore.AsyncAtomState[string]{Status: xstore.AsyncPending}, atom.Get())

	select {
	case ctx := <-got:
		assert.NotNil(t, ctx) // AbortSignal instance
		assert.NoError(t, ctx.Err())
	case <-time.After(2 * time.Second):
		t.Fatal("getValue was not called")
	}
}

// JS: async atoms > should abort and ignore stale async atom results
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/atom.test.ts#L889
func TestStoreAtom_AsyncAtoms_ShouldAbortAndIgnoreStaleAsyncAtomResults(t *testing.T) {
	type run struct {
		ctx      context.Context
		resolved chan struct{}
		count    int
	}
	var mu sync.Mutex
	runs := []*run{}
	getRun := func(i int) *run {
		mu.Lock()
		defer mu.Unlock()
		return runs[i]
	}
	runCount := func() int {
		mu.Lock()
		defer mu.Unlock()
		return len(runs)
	}

	count := xstore.CreateAtom(1)
	atom := xstore.CreateAsyncAtom(func(ctx context.Context) (int, error) {
		r := &run{ctx: ctx, resolved: make(chan struct{}), count: count.Get()}
		mu.Lock()
		runs = append(runs, r)
		mu.Unlock()

		<-r.resolved
		return r.count, nil
	})

	assert.Equal(t, xstore.AsyncAtomState[int]{Status: xstore.AsyncPending}, atom.Get())
	storeAtom1WaitUntil(t, func() bool { return runCount() == 1 })

	count.Set(2)
	assert.Equal(t, xstore.AsyncAtomState[int]{Status: xstore.AsyncPending}, atom.Get())
	assert.Error(t, getRun(0).ctx.Err()) // signals[0].aborted
	storeAtom1WaitUntil(t, func() bool { return runCount() == 2 })

	close(getRun(0).resolved)
	sleep(20)
	assert.Equal(t, xstore.AsyncAtomState[int]{Status: xstore.AsyncPending}, atom.Get())

	close(getRun(1).resolved)
	storeAtom1WaitUntil(t, func() bool { return atom.Get().Status == xstore.AsyncDone })
	assert.Equal(t, xstore.AsyncAtomState[int]{Status: xstore.AsyncDone, Data: 2}, atom.Get())
}

// JS: async atoms > should ignore stale async atom errors
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/atom.test.ts#L917
func TestStoreAtom_AsyncAtoms_ShouldIgnoreStaleAsyncAtomErrors(t *testing.T) {
	type run struct {
		settle chan error // nil error resolves, non-nil rejects
	}
	var mu sync.Mutex
	runs := []*run{}
	getRun := func(i int) *run {
		mu.Lock()
		defer mu.Unlock()
		return runs[i]
	}
	runCount := func() int {
		mu.Lock()
		defer mu.Unlock()
		return len(runs)
	}

	count := xstore.CreateAtom(1)
	atom := xstore.CreateAsyncAtom(func(ctx context.Context) (int, error) {
		currentCount := count.Get()
		r := &run{settle: make(chan error, 1)}
		mu.Lock()
		runs = append(runs, r)
		mu.Unlock()

		if err := <-r.settle; err != nil {
			return 0, err
		}
		if ctx.Err() != nil {
			return 0, errors.New("aborted")
		}
		return currentCount, nil
	})

	assert.Equal(t, xstore.AsyncAtomState[int]{Status: xstore.AsyncPending}, atom.Get())
	storeAtom1WaitUntil(t, func() bool { return runCount() == 1 })

	count.Set(2)
	assert.Equal(t, xstore.AsyncAtomState[int]{Status: xstore.AsyncPending}, atom.Get())
	storeAtom1WaitUntil(t, func() bool { return runCount() == 2 })

	getRun(0).settle <- errors.New("stale")
	sleep(20)
	assert.Equal(t, xstore.AsyncAtomState[int]{Status: xstore.AsyncPending}, atom.Get())

	getRun(1).settle <- nil
	storeAtom1WaitUntil(t, func() bool { return atom.Get().Status == xstore.AsyncDone })
	assert.Equal(t, xstore.AsyncAtomState[int]{Status: xstore.AsyncDone, Data: 2}, atom.Get())
}

// JS: async atoms > should notify subscribers when async operation completes successfully
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/atom.test.ts#L949
func TestStoreAtom_AsyncAtoms_ShouldNotifySubscribersWhenAsyncOperationCompletesSuccessfully(t *testing.T) {
	log := newSpy()
	atom := xstore.CreateAsyncAtom(func(ctx context.Context) (string, error) {
		sleep(10)
		return "test-value", nil
	})

	atom.SubscribeNext(func(s xstore.AsyncAtomState[string]) { log.Call(s) })

	assert.Equal(t, xstore.AsyncAtomState[string]{Status: xstore.AsyncPending}, atom.Get())
	assert.Equal(t, 0, log.Count())

	sleep(100)

	assert.Equal(t, 1, log.Count())
	assert.Contains(t, log.Calls(), []any{xstore.AsyncAtomState[string]{Status: xstore.AsyncDone, Data: "test-value"}})
	assert.Equal(t, xstore.AsyncAtomState[string]{Status: xstore.AsyncDone, Data: "test-value"}, atom.Get())
}

// JS: async atoms > should notify subscribers when async operation fails
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/atom.test.ts#L968
func TestStoreAtom_AsyncAtoms_ShouldNotifySubscribersWhenAsyncOperationFails(t *testing.T) {
	log := newSpy()
	failure := errors.New("test error")
	atom := xstore.CreateAsyncAtom(func(ctx context.Context) (string, error) {
		sleep(10)
		return "", failure
	})

	atom.SubscribeNext(func(s xstore.AsyncAtomState[string]) { log.Call(s) })

	assert.Equal(t, xstore.AsyncAtomState[string]{Status: xstore.AsyncPending}, atom.Get())
	assert.Equal(t, 0, log.Count())

	sleep(100)

	assert.Equal(t, 1, log.Count())
	assert.Contains(t, log.Calls(), []any{xstore.AsyncAtomState[string]{Status: xstore.AsyncError, Error: failure}})
	assert.Equal(t, xstore.AsyncAtomState[string]{Status: xstore.AsyncError, Error: failure}, atom.Get())
}

// JS: async atoms > should notify multiple subscribers when async operation completes
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/atom.test.ts#L988
func TestStoreAtom_AsyncAtoms_ShouldNotifyMultipleSubscribersWhenAsyncOperationCompletes(t *testing.T) {
	log1 := newSpy()
	log2 := newSpy()
	atom := xstore.CreateAsyncAtom(func(ctx context.Context) (string, error) {
		sleep(10)
		return "multi-test", nil
	})

	atom.SubscribeNext(func(s xstore.AsyncAtomState[string]) { log1.Call(s) })
	atom.SubscribeNext(func(s xstore.AsyncAtomState[string]) { log2.Call(s) })

	assert.Equal(t, xstore.AsyncAtomState[string]{Status: xstore.AsyncPending}, atom.Get())
	assert.Equal(t, 0, log1.Count())
	assert.Equal(t, 0, log2.Count())

	sleep(100)

	done := xstore.AsyncAtomState[string]{Status: xstore.AsyncDone, Data: "multi-test"}
	assert.Equal(t, 1, log1.Count())
	assert.Contains(t, log1.Calls(), []any{done})
	assert.Equal(t, 1, log2.Count())
	assert.Contains(t, log2.Calls(), []any{done})
}

// JS: async atoms > subscribe callback should not track dependencies from .get() calls
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/atom.test.ts#L1011
func TestStoreAtom_AsyncAtoms_SubscribeCallbackShouldNotTrackDependenciesFromGetCalls(t *testing.T) {
	items := xstore.CreateAtom([]int{})
	ids := xstore.CreateComputedAtom(func(prev *string) string { return storeAtom1SortedJoin(items.Get()) })
	log := newSpy()

	ids.SubscribeNext(func(string) {
		_ = items.Get()
		log.Call()
	})

	items.Set([]int{1})
	items.Set([]int{1})
	items.Set([]int{1, 2})
	items.Set([]int{1, 2})

	assert.Equal(t, 2, log.Count())
}

// JS: async atoms > subscribe callback should not track deps on non-computed atoms
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/xstate-store/test/atom.test.ts#L1029
func TestStoreAtom_AsyncAtoms_SubscribeCallbackShouldNotTrackDepsOnNonComputedAtoms(t *testing.T) {
	ids := xstore.CreateAtom("")
	items := xstore.CreateAtom([]int{})
	items.SubscribeNext(func(value []int) { ids.Set(storeAtom1SortedJoin(value)) })
	log := newSpy()

	ids.SubscribeNext(func(string) {
		_ = items.Get()
		log.Call()
	})

	items.Set([]int{1})
	items.Set([]int{1})
	items.Set([]int{1, 2})
	items.Set([]int{1, 2})

	assert.Equal(t, 2, log.Count())
}
