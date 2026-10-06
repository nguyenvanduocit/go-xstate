package xstate_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// rehydration1JSONRoundTrip mirrors JSON.parse(JSON.stringify(v)).
func rehydration1JSONRoundTrip(t *testing.T, v any) map[string]any {
	t.Helper()
	raw, err := json.Marshal(v)
	require.NoError(t, err)
	var out map[string]any
	require.NoError(t, json.Unmarshal(raw, &out))
	return out
}

// JS: rehydration > using persisted state > should be able to use `hasTag` immediately
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/rehydration.test.ts#L14
func TestRehydration_UsingPersistedState_ShouldBeAbleToUseHasTagImmediately(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", Tags: xs.Tags{"foo"}},
		},
	})

	actorRef := xs.CreateActor(machine).Start()
	persistedState := rehydration1JSONRoundTrip(t, actorRef.GetPersistedSnapshot())
	actorRef.Stop()

	service := xs.CreateActor(machine, xs.WithSnapshot(persistedState)).Start()

	assert.Equal(t, true, service.GetSnapshot().HasTag("foo"))
}

// JS: rehydration > using persisted state > should not call exit actions when machine gets stopped immediately
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/rehydration.test.ts#L35
func TestRehydration_UsingPersistedState_ShouldNotCallExitActionsWhenStoppedImmediately(t *testing.T) {
	actual := []string{}
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Exit:    xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[any]) { actual = append(actual, "root") })},
		Initial: "a",
		States: xs.States{
			{Key: "a", Exit: xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[any]) { actual = append(actual, "a") })}},
		},
	})

	actorRef := xs.CreateActor(machine).Start()
	persistedState := rehydration1JSONRoundTrip(t, actorRef.GetPersistedSnapshot())
	actorRef.Stop()

	xs.CreateActor(machine, xs.WithSnapshot(persistedState)).Start().Stop()

	assert.Equal(t, []string{}, actual)
}

// JS: rehydration > using persisted state > should get correct result back from `can` immediately
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/rehydration.test.ts#L58
func TestRehydration_UsingPersistedState_ShouldGetCorrectResultFromCanImmediately(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		On: map[string]xs.Transitions{
			"FOO": {{Actions: xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[any]) {})}}},
		},
	})

	// JSON.stringify(snapshot) uses snapshot.toJSON().
	restoredState := rehydration1JSONRoundTrip(t, xs.CreateActor(machine).Start().GetSnapshot().ToJSON())
	service := xs.CreateActor(machine, xs.WithSnapshot(restoredState)).Start()

	assert.Equal(t, true, service.GetSnapshot().Can(xs.Ev("FOO")))
}

// JS: rehydration > using state value > should be able to use `hasTag` immediately
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/rehydration.test.ts#L80
func TestRehydration_UsingStateValue_ShouldBeAbleToUseHasTagImmediately(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "inactive",
		States: xs.States{
			{Key: "inactive", On: map[string]xs.Transitions{"NEXT": {{Target: "active"}}}},
			{Key: "active", Tags: xs.Tags{"foo"}},
		},
	})

	activeState := machine.ResolveState(xs.ResolveStateConfig[any]{Value: "active"})
	service := xs.CreateActor(machine, xs.WithSnapshot(activeState))

	service.Start()

	assert.Equal(t, true, service.GetSnapshot().HasTag("foo"))
}

// JS: rehydration > using state value > should not call exit actions when machine gets stopped immediately
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/rehydration.test.ts#L103
func TestRehydration_UsingStateValue_ShouldNotCallExitActionsWhenStoppedImmediately(t *testing.T) {
	actual := []string{}
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Exit:    xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[any]) { actual = append(actual, "root") })},
		Initial: "inactive",
		States: xs.States{
			{Key: "inactive", On: map[string]xs.Transitions{"NEXT": {{Target: "active"}}}},
			{Key: "active", Exit: xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[any]) { actual = append(actual, "active") })}},
		},
	})

	xs.CreateActor(machine, xs.WithSnapshot(machine.ResolveState(xs.ResolveStateConfig[any]{Value: "active"}))).
		Start().
		Stop()

	assert.Equal(t, []string{}, actual)
}

// JS: rehydration > using state value > should error on incompatible state value (shallow)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/rehydration.test.ts#L127
func TestRehydration_UsingStateValue_ShouldErrorOnIncompatibleStateValueShallow(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "valid",
		States: xs.States{
			{Key: "valid"},
		},
	})

	// toThrowError(/invalid/): panicMessage returns "" when nothing panics.
	assert.Regexp(t, "invalid", panicMessage(func() {
		machine.ResolveState(xs.ResolveStateConfig[any]{Value: "invalid"})
	}))
}

// JS: rehydration > using state value > should error on incompatible state value (deep)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/rehydration.test.ts#L140
func TestRehydration_UsingStateValue_ShouldErrorOnIncompatibleStateValueDeep(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "parent",
		States: xs.States{
			{Key: "parent", Initial: "valid", States: xs.States{
				{Key: "valid"},
			}},
		},
	})

	// toThrowError(/invalid/): panicMessage returns "" when nothing panics.
	assert.Regexp(t, "invalid", panicMessage(func() {
		machine.ResolveState(xs.ResolveStateConfig[any]{Value: map[string]any{"parent": "invalid"}})
	}))
}

// JS: rehydration > should not replay actions when starting from a persisted state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/rehydration.test.ts#L159
func TestRehydration_ShouldNotReplayActionsWhenStartingFromPersistedState(t *testing.T) {
	entrySpy := newSpy()
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Entry: xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[any]) { entrySpy.Call(a) })},
	})

	actor := xs.CreateActor(machine).Start()

	assert.Equal(t, 1, entrySpy.Count())

	persistedState := actor.GetPersistedSnapshot()

	actor.Stop()

	xs.CreateActor(machine, xs.WithSnapshot(persistedState)).Start()

	assert.Equal(t, 1, entrySpy.Count())
}

// JS: rehydration > should be able to stop a rehydrated child
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/rehydration.test.ts#L178
func TestRehydration_ShouldBeAbleToStopRehydratedChild(t *testing.T) {
	// JS `Promise.resolve(11)` settles in a microtask, and the JS test body
	// never awaits, so the promise cannot settle before the assertions below.
	// The promise body is held until the test body returns to keep that
	// ordering (Go promise bodies run on their own goroutines).
	release := make(chan struct{})
	defer close(release)
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{
				Key: "a",
				Invoke: []xs.InvokeConfig{{
					Logic: xs.FromPromise(func(_ context.Context, _ xs.PromiseArgs) (int, error) {
						<-release
						return 11, nil
					}),
					OnDone: xs.Transitions{{Target: "b"}},
				}},
				On: map[string]xs.Transitions{"NEXT": {{Target: "c"}}},
			},
			{Key: "b"},
			{Key: "c"},
		},
	})

	actor := xs.CreateActor(machine).Start()
	persistedState := actor.GetPersistedSnapshot()
	actor.Stop()

	rehydratedActor := xs.CreateActor(machine, xs.WithSnapshot(persistedState)).Start()

	assert.NotPanics(t, func() {
		rehydratedActor.Send(xs.Ev("NEXT"))
	})

	assert.Equal(t, "c", rehydratedActor.GetSnapshot().Value)
}

// JS: rehydration > a rehydrated active child should be registered in the system
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/rehydration.test.ts#L213
func TestRehydration_RehydratedActiveChildShouldBeRegisteredInSystem(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		ContextFn: func(a xs.ContextArgs) any {
			a.Spawn("foo", xs.SpawnOptions{SystemID: "mySystemId"})
			return map[string]any{}
		},
	}, xs.Implementations{Actors: map[string]xs.ActorLogic{
		"foo": xs.CreateMachine(xs.MachineConfig[any]{}),
	}})

	actor := xs.CreateActor(machine).Start()
	persistedState := actor.GetPersistedSnapshot()
	actor.Stop()

	rehydratedActor := xs.CreateActor(machine, xs.WithSnapshot(persistedState)).Start()

	assert.NotNil(t, rehydratedActor.System().Get("mySystemId"))
}

// JS: rehydration > a rehydrated done child should not be registered in the system
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/rehydration.test.ts#L241
func TestRehydration_RehydratedDoneChildShouldNotBeRegisteredInSystem(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		ContextFn: func(a xs.ContextArgs) any {
			a.Spawn("foo", xs.SpawnOptions{SystemID: "mySystemId"})
			return map[string]any{}
		},
	}, xs.Implementations{Actors: map[string]xs.ActorLogic{
		"foo": xs.CreateMachine(xs.MachineConfig[any]{Type: xs.Final}),
	}})

	actor := xs.CreateActor(machine).Start()
	persistedState := actor.GetPersistedSnapshot()
	actor.Stop()

	rehydratedActor := xs.CreateActor(machine, xs.WithSnapshot(persistedState)).Start()

	assert.Nil(t, rehydratedActor.System().Get("mySystemId"))
}

// JS: rehydration > a rehydrated done child should not re-notify the parent about its completion
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/rehydration.test.ts#L269
func TestRehydration_RehydratedDoneChildShouldNotReNotifyParentAboutCompletion(t *testing.T) {
	s := newSpy()

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		ContextFn: func(a xs.ContextArgs) any {
			a.Spawn("foo", xs.SpawnOptions{SystemID: "mySystemId"})
			return map[string]any{}
		},
		On: map[string]xs.Transitions{
			"*": {{Actions: xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[any]) { s.Call(a) })}}},
		},
	}, xs.Implementations{Actors: map[string]xs.ActorLogic{
		"foo": xs.CreateMachine(xs.MachineConfig[any]{Type: xs.Final}),
	}})

	actor := xs.CreateActor(machine).Start()
	persistedState := actor.GetPersistedSnapshot()
	actor.Stop()

	// spy.mockClear(): count calls from here on.
	cleared := s.Count()

	xs.CreateActor(machine, xs.WithSnapshot(persistedState)).Start()

	assert.Equal(t, 0, s.Count()-cleared)
}

// JS: rehydration > should be possible to persist a rehydrated actor that got its children rehydrated
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/rehydration.test.ts#L306
func TestRehydration_ShouldBePossibleToPersistRehydratedActorWithRehydratedChildren(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Invoke: []xs.InvokeConfig{{Src: "foo"}},
	}, xs.Implementations{Actors: map[string]xs.ActorLogic{
		"foo": xs.FromPromise(func(_ context.Context, _ xs.PromiseArgs) (int, error) { return 42, nil }),
	}})

	actor := xs.CreateActor(machine).Start()

	rehydratedActor := xs.CreateActor(machine, xs.WithSnapshot(actor.GetPersistedSnapshot())).Start()

	// (persisted as any).children: the persisted snapshot is a plain JSON-like object.
	persisted := rehydration1JSONRoundTrip(t, rehydratedActor.GetPersistedSnapshot())
	persistedChildren, _ := persisted["children"].(map[string]any)
	assert.Equal(t, 1, len(persistedChildren))
	for _, child := range persistedChildren { // Object.values(persistedChildren)[0]
		childMap, _ := child.(map[string]any)
		assert.Equal(t, "foo", childMap["src"])
	}
}

// JS: rehydration > should complete on a rehydrated final state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/rehydration.test.ts#L332
func TestRehydration_ShouldCompleteOnRehydratedFinalState(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "foo",
		States: xs.States{
			{Key: "foo", On: map[string]xs.Transitions{"NEXT": {{Target: "bar"}}}},
			{Key: "bar", Type: xs.Final},
		},
	})

	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.Ev("NEXT"))
	persistedState := actorRef.GetPersistedSnapshot()

	s := newSpy()
	actorRef2 := xs.CreateActor(machine, xs.WithSnapshot(persistedState))
	actorRef2.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{
		Complete: func() { s.Call() },
	})

	actorRef2.Start()
	assert.Positive(t, s.Count())
}

// JS: rehydration > should error on a rehydrated error state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/rehydration.test.ts#L359
func TestRehydration_ShouldErrorOnRehydratedErrorState(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Invoke: []xs.InvokeConfig{{Src: "failure"}},
	}, xs.Implementations{Actors: map[string]xs.ActorLogic{
		"failure": xs.FromPromise(func(_ context.Context, _ xs.PromiseArgs) (any, error) {
			return nil, errors.New("failure")
		}),
	}})

	// JS `await sleep(0)` waits for the rejection to be processed; the Go
	// promise runs on its own goroutine, so wait for the error notification.
	processed := newSignal()
	actorRef := xs.CreateActor(machine)
	actorRef.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{
		Error: func(err any) { processed.Resolve() }, // preventUnhandledErrorListener
	})
	actorRef.Start()

	processed.Wait(t)

	persistedState := actorRef.GetPersistedSnapshot()

	s := newSpy()
	actorRef2 := xs.CreateActor(machine, xs.WithSnapshot(persistedState))
	actorRef2.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{
		Error: func(err any) { s.Call(err) },
	})
	actorRef2.Start()

	assert.Positive(t, s.Count())
}

// JS: rehydration > shouldn't re-notify the parent about the error when rehydrating
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/rehydration.test.ts#L392
func TestRehydration_ShouldNotReNotifyParentAboutErrorWhenRehydrating(t *testing.T) {
	s := newSpy()
	processed := newSignal()

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Invoke: []xs.InvokeConfig{{
			Src: "failure",
			OnError: xs.Transitions{{Actions: xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[any]) {
				s.Call(a)
				processed.Resolve()
			})}}},
		}},
	}, xs.Implementations{Actors: map[string]xs.ActorLogic{
		"failure": xs.FromPromise(func(_ context.Context, _ xs.PromiseArgs) (any, error) {
			return nil, errors.New("failure")
		}),
	}})

	actorRef := xs.CreateActor(machine)
	actorRef.Start()

	// JS `await sleep(0)` waits for the rejection to be processed; the Go
	// promise runs on its own goroutine, so wait for onError to run.
	processed.Wait(t)

	persistedState := actorRef.GetPersistedSnapshot()
	// spy.mockClear(): count calls from here on.
	cleared := s.Count()

	actorRef2 := xs.CreateActor(machine, xs.WithSnapshot(persistedState))
	actorRef2.Start()

	assert.Equal(t, 0, s.Count()-cleared)
}

// JS: rehydration > should continue syncing snapshots
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/rehydration.test.ts#L426
func TestRehydration_ShouldContinueSyncingSnapshots(t *testing.T) {
	subject := newBehaviorSubject(0)
	subjectLogic := xs.FromObservable(func(_ xs.ObservableArgs) xs.Subscribable[int] { return subject })

	s := newSpy()

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Invoke: []xs.InvokeConfig{{
			Src: "service",
			OnSnapshot: xs.Transitions{{Actions: xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[any]) {
				s.Call(a.Event.(xs.SnapshotEvent).Snapshot.(*xs.ObservableSnapshot[int]).Context)
			})}}},
		}},
	}, xs.Implementations{Actors: map[string]xs.ActorLogic{
		"service": subjectLogic,
	}})

	xs.CreateActor(machine, xs.WithSnapshot(xs.CreateActor(machine).GetPersistedSnapshot())).Start()

	// spy.mockClear(): only calls from here on are asserted.
	cleared := s.Count()

	subject.Next(42)
	subject.Next(100)

	assert.Equal(t, [][]any{{42}, {100}}, s.Calls()[cleared:])
}

// JS: rehydration > should be able to rehydrate an actor deep in the tree
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/rehydration.test.ts#L469
func TestRehydration_ShouldBeAbleToRehydrateActorDeepInTree(t *testing.T) {
	type gctx struct{ Count int }

	grandchild := xs.CreateMachine(xs.MachineConfig[gctx]{
		Context: gctx{Count: 0},
		On: map[string]xs.Transitions{
			"INC": {{Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[gctx]) gctx {
				c := a.Context
				c.Count++
				return c
			})}}},
		},
	})
	child := xs.CreateMachine(xs.MachineConfig[any]{
		Invoke: []xs.InvokeConfig{{Src: "grandchild", ID: "grandchild"}},
		On: map[string]xs.Transitions{
			"INC": {{Actions: xs.Actions{xs.SendTo("grandchild", xs.Ev("INC"))}}},
		},
	}, xs.Implementations{Actors: map[string]xs.ActorLogic{"grandchild": grandchild}})
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Invoke: []xs.InvokeConfig{{Src: "child", ID: "child"}},
		On: map[string]xs.Transitions{
			"INC": {{Actions: xs.Actions{xs.SendTo("child", xs.Ev("INC"))}}},
		},
	}, xs.Implementations{Actors: map[string]xs.ActorLogic{"child": child}})

	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.Ev("INC"))

	persistedState := actorRef.GetPersistedSnapshot()
	actorRef2 := xs.CreateActor(machine, xs.WithSnapshot(persistedState))

	childSnap := machineSnap[any](actorRef2.GetSnapshot().Children["child"])
	assert.Equal(t, 1, machineSnap[gctx](childSnap.Children["grandchild"]).Context.Count)
}
