package xstate_test

import (
	"context"
	"testing"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/assert"
)

// JS: input > should create a machine with input
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/input.test.ts#L12
func TestInput_ShouldCreateMachineWithInput(t *testing.T) {
	type ctx struct{ Count int }
	type input struct{ StartCount int }
	s := newSpy()

	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		ContextFn: func(a xs.ContextArgs) ctx {
			return ctx{Count: a.Input.(input).StartCount}
		},
		Entry: xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[ctx]) {
			s.Call(a.Context.Count)
		})},
	})

	xs.CreateActor(machine, xs.WithInput(input{StartCount: 42})).Start()

	assert.Contains(t, s.Calls(), []any{42})
}

// JS: input > initial event should have input property
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/input.test.ts#L33
func TestInput_InitialEventShouldHaveInputProperty(t *testing.T) {
	type input struct{ Greeting string }
	sig := newSignal()

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Entry: xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[any]) {
			assert.Equal(t, "hello", a.Event.(xs.InitEvent).Input.(input).Greeting)
			sig.Resolve()
		})},
	})

	xs.CreateActor(machine, xs.WithInput(input{Greeting: "hello"})).Start()

	sig.Wait(t)
}

// JS: input > should error if input is expected but not provided
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/input.test.ts#L47
func TestInput_ShouldErrorIfInputIsExpectedButNotProvided(t *testing.T) {
	type input struct{ Greeting string }
	type ctx struct{ Message string }

	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		ContextFn: func(a xs.ContextArgs) ctx {
			// JS reads `input.greeting` on undefined input and throws a TypeError;
			// the type assertion on a nil input panics the same way.
			return ctx{Message: "Hello, " + a.Input.(input).Greeting}
		},
	})

	snapshot := xs.CreateActor(machine).GetSnapshot()

	assert.Equal(t, xs.StatusError, snapshot.Status)
}

// JS: input > should retain the machine snapshot interface when resolving input throws
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/input.test.ts#L64
func TestInput_ShouldRetainMachineSnapshotInterfaceWhenResolvingInputThrows(t *testing.T) {
	type input struct{ Greeting string }
	type ctx struct{ Message string }

	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		ContextFn: func(a xs.ContextArgs) ctx {
			return ctx{Message: "Hello, " + a.Input.(input).Greeting}
		},
		Initial: "saving",
		States: xs.States{
			{Key: "saving"},
		},
	})

	// Missing input deliberately exercises initialization.
	snapshot := xs.CreateActor(machine).GetSnapshot()

	assert.Equal(t, xs.StatusError, snapshot.Status)
	assert.True(t, snapshot.Matches("saving"))
}

// JS: input > should be a type error if input is not expected yet provided
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/input.test.ts#L86
func TestInput_ShouldBeTypeErrorIfInputIsNotExpectedYetProvided(t *testing.T) {
	type ctx struct{ Count int }

	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Context: ctx{Count: 42},
	})

	assert.NotPanics(t, func() {
		xs.CreateActor(machine).Start()
	})
}

// JS: input > should provide input data to invoked machines
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/input.test.ts#L97
func TestInput_ShouldProvideInputDataToInvokedMachines(t *testing.T) {
	type greeting struct{ Greeting string }
	sig := newSignal()

	invokedMachine := xs.CreateMachine(xs.MachineConfig[greeting]{
		ContextFn: func(a xs.ContextArgs) greeting { return a.Input.(greeting) },
		Entry: xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[greeting]) {
			assert.Equal(t, "hello", a.Context.Greeting)
			assert.Equal(t, "hello", a.Event.(xs.InitEvent).Input.(greeting).Greeting)
			sig.Resolve()
		})},
	})

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Invoke: []xs.InvokeConfig{{
			Logic: invokedMachine,
			Input: greeting{Greeting: "hello"},
		}},
	})

	xs.CreateActor(machine).Start()

	sig.Wait(t)
}

// JS: input > should provide input data to spawned machines
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/input.test.ts#L124
func TestInput_ShouldProvideInputDataToSpawnedMachines(t *testing.T) {
	type greeting struct{ Greeting string }
	type ctx struct{ Ref xs.ActorRef }
	sig := newSignal()

	spawnedMachine := xs.CreateMachine(xs.MachineConfig[greeting]{
		ContextFn: func(a xs.ContextArgs) greeting { return a.Input.(greeting) },
		Entry: xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[greeting]) {
			assert.Equal(t, "hello", a.Context.Greeting)
			assert.Equal(t, "hello", a.Event.(xs.InitEvent).Input.(greeting).Greeting)
			sig.Resolve()
		})},
	})

	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Entry: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
			return ctx{
				Ref: a.Spawn(spawnedMachine, xs.SpawnOptions{Input: greeting{Greeting: "hello"}}),
			}
		})},
	})

	xs.CreateActor(machine).Start()

	sig.Wait(t)
}

// JS: input > should create a promise with input
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/input.test.ts#L154
func TestInput_ShouldCreatePromiseWithInput(t *testing.T) {
	type count struct{ Count int }

	promiseLogic := xs.FromPromise(func(_ context.Context, a xs.PromiseArgs) (count, error) {
		return a.Input.(count), nil
	})

	promiseActor := xs.CreateActor(promiseLogic, xs.WithInput(count{Count: 42})).Start()

	sleep(5)

	assert.Equal(t, count{Count: 42}, promiseActor.GetSnapshot().Output)
}

// JS: input > should create a transition function actor with input
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/input.test.ts#L168
func TestInput_ShouldCreateTransitionFunctionActorWithInput(t *testing.T) {
	type count struct{ Count int }

	transitionLogic := xs.FromTransition(
		func(state count, _ xs.Event, _ *xs.ActorScope) count { return state },
		func(a xs.TransitionInitArgs) count { return a.Input.(count) },
	)

	transitionActor := xs.CreateActor(transitionLogic, xs.WithInput(count{Count: 42})).Start()

	assert.Equal(t, count{Count: 42}, transitionActor.GetSnapshot().Context)
}

// JS: input > should create an observable actor with input
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/input.test.ts#L181
func TestInput_ShouldCreateObservableActorWithInput(t *testing.T) {
	type count struct{ Count int }
	sig := newSignal()

	observableLogic := xs.FromObservable(func(a xs.ObservableArgs) xs.Subscribable[count] {
		return rxOf(a.Input.(count))
	})

	observableActor := xs.CreateActor(observableLogic, xs.WithInput(count{Count: 42}))

	var sub xs.Subscription
	sub = observableActor.SubscribeNext(func(state *xs.ObservableSnapshot[count]) {
		// JS: `state.context?.count !== 42` — before the first emission the Go
		// context is the zero value, which also fails this check.
		if state.Context.Count != 42 {
			return
		}
		assert.Equal(t, count{Count: 42}, state.Context)
		sub.Unsubscribe()
		sig.Resolve()
	})

	observableActor.Start()

	sig.Wait(t)
}

// JS: input > should create a callback actor with input
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/input.test.ts#L204
func TestInput_ShouldCreateCallbackActorWithInput(t *testing.T) {
	type count struct{ Count int }
	sig := newSignal()

	callbackLogic := xs.FromCallback(func(a xs.CallbackArgs) func() {
		assert.Equal(t, count{Count: 42}, a.Input)
		sig.Resolve()
		return nil
	})

	xs.CreateActor(callbackLogic, xs.WithInput(count{Count: 42})).Start()

	sig.Wait(t)
}

// JS: input > should provide a static inline input to the referenced actor
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/input.test.ts#L218
func TestInput_ShouldProvideStaticInlineInputToReferencedActor(t *testing.T) {
	s := newSpy()

	child := xs.CreateMachine(xs.MachineConfig[map[string]any]{
		ContextFn: func(a xs.ContextArgs) map[string]any {
			s.Call(a.Input)
			return map[string]any{}
		},
	})

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Invoke: []xs.InvokeConfig{{
			Src:   "child",
			Input: 42,
		}},
	}, xs.Implementations{Actors: map[string]xs.ActorLogic{"child": child}})

	xs.CreateActor(machine).Start()

	assert.Contains(t, s.Calls(), []any{42})
}

// JS: input > should provide a dynamic inline input to the referenced actor
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/input.test.ts#L250
func TestInput_ShouldProvideDynamicInlineInputToReferencedActor(t *testing.T) {
	type ctx struct{ Count int }
	s := newSpy()

	child := xs.CreateMachine(xs.MachineConfig[map[string]any]{
		ContextFn: func(a xs.ContextArgs) map[string]any {
			s.Call(a.Input)
			return map[string]any{}
		},
	})

	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		ContextFn: func(a xs.ContextArgs) ctx {
			return ctx{Count: a.Input.(int)}
		},
		Invoke: []xs.InvokeConfig{{
			Src: "child",
			Input: xs.NewExpr(func(a xs.ExprArgs[ctx]) any {
				return a.Context.Count + 100
			}),
		}},
	}, xs.Implementations{Actors: map[string]xs.ActorLogic{"child": child}})

	xs.CreateActor(machine, xs.WithInput(42)).Start()

	assert.Contains(t, s.Calls(), []any{142})
}

// JS: input > should call the input factory with self when invoking
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/input.test.ts#L294
func TestInput_ShouldCallInputFactoryWithSelfWhenInvoking(t *testing.T) {
	s := newSpy()

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Invoke: []xs.InvokeConfig{{
			Logic: xs.CreateMachine(xs.MachineConfig[any]{}),
			Input: xs.NewExpr(func(a xs.ExprArgs[any]) any {
				s.Call(a.Self)
				return nil
			}),
		}},
	})

	actor := xs.CreateActor(machine).Start()

	assert.Contains(t, s.Calls(), []any{actor})
}

// JS: input > should call the input factory with self when spawning
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/input.test.ts#L309
func TestInput_ShouldCallInputFactoryWithSelfWhenSpawning(t *testing.T) {
	s := newSpy()

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Entry: xs.Actions{xs.SpawnChild("child", xs.SpawnOptions{
			Input: xs.NewExpr(func(a xs.ExprArgs[any]) any {
				s.Call(a.Self)
				return nil
			}),
		})},
	}, xs.Implementations{Actors: map[string]xs.ActorLogic{
		"child": xs.CreateMachine(xs.MachineConfig[any]{}),
	}})

	actor := xs.CreateActor(machine).Start()

	assert.Contains(t, s.Calls(), []any{actor})
}
