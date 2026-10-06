package xstate_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// JS: system > should register an invoked actor
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/system.test.ts#L25
func TestSystem_ShouldRegisterAnInvokedActor(t *testing.T) {
	sig := newSignal()

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "parent",
		Initial: "a",
		States: xs.States{
			{Key: "a", Invoke: []xs.InvokeConfig{
				{
					Logic: xs.FromCallback(func(a xs.CallbackArgs) func() {
						a.Receive(func(event xs.Event) {
							assert.Equal(t, "HELLO", event.EventType())
							sig.Resolve()
						})
						return nil
					}),
					SystemID: "receiver",
				},
				{
					Logic: xs.CreateMachine(xs.MachineConfig[any]{
						ID: "childmachine",
						Entry: xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[any]) {
							if a.System == nil {
								return
							}
							receiver := a.System.Get("receiver")
							if receiver != nil {
								receiver.Send(xs.Ev("HELLO"))
							}
						})},
					}),
				},
			}},
		},
	})

	xs.CreateActor(machine).Start()

	sig.Wait(t)
}

// JS: system > should register a spawned actor
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/system.test.ts#L70
func TestSystem_ShouldRegisterASpawnedActor(t *testing.T) {
	type ctx struct {
		Ref        xs.ActorRef
		MachineRef xs.ActorRef
	}
	sig := newSignal()

	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		ID: "parent",
		ContextFn: func(a xs.ContextArgs) ctx {
			return ctx{
				Ref: a.Spawn(
					xs.FromCallback(func(a xs.CallbackArgs) func() {
						a.Receive(func(event xs.Event) {
							assert.Equal(t, "HELLO", event.EventType())
							sig.Resolve()
						})
						return nil
					}),
					xs.SpawnOptions{SystemID: "receiver"},
				),
			}
		},
		On: map[string]xs.Transitions{
			"toggle": {{Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
				c := a.Context
				c.MachineRef = a.Spawn(xs.CreateMachine(xs.MachineConfig[any]{
					ID: "childmachine",
					Entry: xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[any]) {
						var receiver xs.ActorRef
						if a.System != nil {
							receiver = a.System.Get("receiver")
						}
						if receiver != nil {
							receiver.Send(xs.Ev("HELLO"))
						} else {
							panic(errors.New("no"))
						}
					})},
				}))
				return c
			})}}},
		},
	})

	actor := xs.CreateActor(machine).Start()

	actor.Send(xs.Ev("toggle"))

	sig.Wait(t)
}

// JS: system > system can be immediately accessed outside the actor
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/system.test.ts#L128
func TestSystem_SystemCanBeImmediatelyAccessedOutsideTheActor(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Invoke: []xs.InvokeConfig{{
			SystemID: "someChild",
			Logic:    xs.CreateMachine(xs.MachineConfig[any]{}),
		}},
	})

	// no .Start() here is important for the test
	actor := xs.CreateActor(machine)

	assert.NotNil(t, actor.System().Get("someChild"))
}

// JS: system > root actor can be given the systemId
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/system.test.ts#L142
func TestSystem_RootActorCanBeGivenTheSystemId(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{})
	actor := xs.CreateActor(machine, xs.WithSystemID("test"))
	assert.Same(t, actor, actor.System().Get("test"))
}

// JS: system > should remove invoked actor from receptionist if stopped
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/system.test.ts#L148
func TestSystem_ShouldRemoveInvokedActorFromReceptionistIfStopped(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "active",
		States: xs.States{
			{
				Key: "active",
				Invoke: []xs.InvokeConfig{{
					Logic:    xs.CreateMachine(xs.MachineConfig[any]{}),
					SystemID: "test",
				}},
				On: map[string]xs.Transitions{
					"toggle": {{Target: "inactive"}},
				},
			},
			{Key: "inactive"},
		},
	})

	actor := xs.CreateActor(machine).Start()

	assert.NotNil(t, actor.System().Get("test"))

	actor.Send(xs.Ev("toggle"))

	assert.Nil(t, actor.System().Get("test"))
}

// JS: system > should remove spawned actor from receptionist if stopped
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/system.test.ts#L174
func TestSystem_ShouldRemoveSpawnedActorFromReceptionistIfStopped(t *testing.T) {
	type ctx struct{ Ref xs.ActorRef }

	childMachine := xs.CreateMachine(xs.MachineConfig[any]{})
	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		ContextFn: func(a xs.ContextArgs) ctx {
			return ctx{Ref: a.Spawn(childMachine, xs.SpawnOptions{SystemID: "test"})}
		},
		On: map[string]xs.Transitions{
			"toggle": {{Actions: xs.Actions{xs.StopChild(xs.NewExpr(func(a xs.ExprArgs[ctx]) any {
				return a.Context.Ref
			}))}}},
		},
	})

	actor := xs.CreateActor(machine).Start()

	assert.NotNil(t, actor.System().Get("test"))

	actor.Send(xs.Ev("toggle"))

	assert.Nil(t, actor.System().Get("test"))
}

// JS: system > should throw an error if an actor with the system ID already exists
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/system.test.ts#L203
func TestSystem_ShouldThrowAnErrorIfAnActorWithTheSystemIDAlreadyExists(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "inactive",
		States: xs.States{
			{Key: "inactive", On: map[string]xs.Transitions{
				"toggle": {{Target: "active"}},
			}},
			{Key: "active", Invoke: []xs.InvokeConfig{
				{Logic: xs.CreateMachine(xs.MachineConfig[any]{}), SystemID: "test"},
				{Logic: xs.CreateMachine(xs.MachineConfig[any]{}), SystemID: "test"},
			}},
		},
	})

	errorSpy := newSpy()

	actorRef := xs.CreateActor(machine, xs.WithSystemID("test"))
	actorRef.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{
		Error: func(err any) { errorSpy.Call(err) },
	})
	actorRef.Start()
	actorRef.Send(xs.Ev("toggle"))

	calls := errorSpy.Calls()
	require.Len(t, calls, 1)
	require.Len(t, calls[0], 1)
	err, ok := calls[0][0].(error)
	require.True(t, ok, "expected an error value, got %T", calls[0][0])
	assert.EqualError(t, err, "Actor with system ID 'test' already exists.")
}

// JS: system > should cleanup stopped actors
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/system.test.ts#L245
func TestSystem_ShouldCleanupStoppedActors(t *testing.T) {
	type ctx struct{ Ref xs.ActorRef }

	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		ContextFn: func(a xs.ContextArgs) ctx {
			return ctx{Ref: a.Spawn(
				xs.FromPromise(func(context.Context, xs.PromiseArgs) (any, error) { return nil, nil }),
				xs.SpawnOptions{SystemID: "test"},
			)}
		},
		On: map[string]xs.Transitions{
			"stop": {{Actions: xs.Actions{xs.StopChild(xs.NewExpr(func(a xs.ExprArgs[ctx]) any {
				return a.Context.Ref
			}))}}},
			"start": {{Actions: xs.Actions{xs.SpawnChild(
				xs.FromPromise(func(context.Context, xs.PromiseArgs) (any, error) { return nil, nil }),
				xs.SpawnOptions{SystemID: "test"},
			)}}},
		},
	})

	actor := xs.CreateActor(machine).Start()

	actor.Send(xs.Ev("stop"))

	assert.NotPanics(t, func() {
		actor.Send(xs.Ev("start"))
	})
}

// JS: system > should be accessible in inline custom actions
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/system.test.ts#L284
func TestSystem_ShouldBeAccessibleInInlineCustomActions(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Invoke: []xs.InvokeConfig{{
			Logic:    xs.CreateMachine(xs.MachineConfig[any]{}),
			SystemID: "test",
		}},
		Entry: xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[any]) {
			assert.NotNil(t, a.System.Get("test"))
		})},
	})

	xs.CreateActor(machine).Start()
}

// JS: system > should be accessible in referenced custom actions
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/system.test.ts#L298
func TestSystem_ShouldBeAccessibleInReferencedCustomActions(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Invoke: []xs.InvokeConfig{{
			Logic:    xs.CreateMachine(xs.MachineConfig[any]{}),
			SystemID: "test",
		}},
		Entry: xs.Actions{xs.ActionRef{Type: "myAction"}},
	}, xs.Implementations{
		Actions: map[string]xs.Action{
			"myAction": xs.ActionFunc(func(a xs.ActionArgs[any]) {
				assert.NotNil(t, a.System.Get("test"))
			}),
		},
	})

	xs.CreateActor(machine).Start()
}

// JS: system > should be accessible in assign actions
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/system.test.ts#L319
func TestSystem_ShouldBeAccessibleInAssignActions(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Invoke: []xs.InvokeConfig{{
			Logic:    xs.CreateMachine(xs.MachineConfig[any]{}),
			SystemID: "test",
		}},
		Initial: "a",
		States: xs.States{
			{Key: "a", Entry: xs.Actions{xs.Assign(func(a xs.AssignArgs[any]) any {
				assert.NotNil(t, a.System.Get("test"))
				return a.Context
			})}},
		},
	})

	xs.CreateActor(machine).Start()
}

// JS: system > should be accessible in sendTo actions
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/system.test.ts#L338
func TestSystem_ShouldBeAccessibleInSendToActions(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Invoke: []xs.InvokeConfig{{
			Logic:    xs.CreateMachine(xs.MachineConfig[any]{}),
			SystemID: "test",
		}},
		Initial: "a",
		States: xs.States{
			{Key: "a", Entry: xs.Actions{xs.SendTo(
				xs.NewExpr(func(a xs.ExprArgs[any]) any {
					assert.NotNil(t, a.System.Get("test"))
					return a.System.Get("test")
				}),
				xs.Ev("FOO"),
			)}},
		},
	})

	xs.CreateActor(machine).Start()
}

// JS: system > should be accessible in promise logic
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/system.test.ts#L361
func TestSystem_ShouldBeAccessibleInPromiseLogic(t *testing.T) {
	// expect.assertions(2)
	var assertions atomic.Int32
	promiseRan := newSignal()

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Invoke: []xs.InvokeConfig{
			{
				Logic:    xs.CreateMachine(xs.MachineConfig[any]{}),
				SystemID: "test",
			},
			{
				Logic: xs.FromPromise(func(_ context.Context, a xs.PromiseArgs) (any, error) {
					assertions.Add(1)
					assert.NotNil(t, a.System.Get("test"))
					promiseRan.Resolve()
					return nil, nil
				}),
			},
		},
	})

	actor := xs.CreateActor(machine).Start()

	assertions.Add(1)
	assert.NotNil(t, actor.System().Get("test"))

	// The promise body runs on its own goroutine in Go; JS runs it
	// synchronously during start. Wait for it before counting assertions.
	promiseRan.Wait(t)
	assert.Equal(t, int32(2), assertions.Load())
}

// JS: system > should be accessible in transition logic
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/system.test.ts#L383
func TestSystem_ShouldBeAccessibleInTransitionLogic(t *testing.T) {
	// expect.assertions(2)
	var assertions atomic.Int32

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Invoke: []xs.InvokeConfig{
			{
				Logic:    xs.CreateMachine(xs.MachineConfig[any]{}),
				SystemID: "test",
			},
			{
				Logic: xs.FromTransition(
					func(_ int, _ xs.Event, scope *xs.ActorScope) int {
						assertions.Add(1)
						assert.NotNil(t, scope.System.Get("test"))
						return 0
					},
					func(xs.TransitionInitArgs) int { return 0 },
				),
				SystemID: "reducer",
			},
		},
	})

	actor := xs.CreateActor(machine).Start()

	assertions.Add(1)
	assert.NotNil(t, actor.System().Get("test"))

	// The assertion won't be checked until the transition function gets an event
	reducer := actor.System().Get("reducer")
	require.NotNil(t, reducer)
	reducer.Send(xs.Ev("a"))

	assert.Equal(t, int32(2), assertions.Load())
}

// JS: system > should be accessible in observable logic
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/system.test.ts#L410
func TestSystem_ShouldBeAccessibleInObservableLogic(t *testing.T) {
	// expect.assertions(2)
	var assertions atomic.Int32

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Invoke: []xs.InvokeConfig{
			{
				Logic:    xs.CreateMachine(xs.MachineConfig[any]{}),
				SystemID: "test",
			},
			{
				Logic: xs.FromObservable(func(a xs.ObservableArgs) xs.Subscribable[int] {
					assertions.Add(1)
					assert.NotNil(t, a.System.Get("test"))
					return rxOf(0)
				}),
			},
		},
	})

	actor := xs.CreateActor(machine).Start()

	assertions.Add(1)
	assert.NotNil(t, actor.System().Get("test"))

	assert.Equal(t, int32(2), assertions.Load())
}

// JS: system > should be accessible in event observable logic
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/system.test.ts#L433
func TestSystem_ShouldBeAccessibleInEventObservableLogic(t *testing.T) {
	// expect.assertions(2)
	var assertions atomic.Int32

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Invoke: []xs.InvokeConfig{
			{
				Logic:    xs.CreateMachine(xs.MachineConfig[any]{}),
				SystemID: "test",
			},
			{
				Logic: xs.FromEventObservable(func(a xs.ObservableArgs) xs.Subscribable[xs.Event] {
					assertions.Add(1)
					assert.NotNil(t, a.System.Get("test"))
					return rxOf[xs.Event](xs.Ev("a"))
				}),
			},
		},
	})

	actor := xs.CreateActor(machine).Start()

	assertions.Add(1)
	assert.NotNil(t, actor.System().Get("test"))

	assert.Equal(t, int32(2), assertions.Load())
}

// JS: system > should be accessible in callback logic
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/system.test.ts#L456
func TestSystem_ShouldBeAccessibleInCallbackLogic(t *testing.T) {
	// expect.assertions(2)
	var assertions atomic.Int32

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Invoke: []xs.InvokeConfig{
			{
				Logic:    xs.CreateMachine(xs.MachineConfig[any]{}),
				SystemID: "test",
			},
			{
				Logic: xs.FromCallback(func(a xs.CallbackArgs) func() {
					assertions.Add(1)
					assert.NotNil(t, a.System.Get("test"))
					return nil
				}),
			},
		},
	})

	actor := xs.CreateActor(machine).Start()

	assertions.Add(1)
	assert.NotNil(t, actor.System().Get("test"))

	assert.Equal(t, int32(2), assertions.Load())
}

// JS: system > should gracefully handle re-registration of a `systemId` during a reentering transition
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/system.test.ts#L477
func TestSystem_ShouldGracefullyHandleReRegistrationOfASystemIdDuringAReenteringTransition(t *testing.T) {
	spy := newSpy()

	var counter atomic.Int32

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "listening",
		States: xs.States{
			{Key: "listening", Invoke: []xs.InvokeConfig{{
				SystemID: "listener",
				Logic: xs.FromCallback(func(a xs.CallbackArgs) func() {
					localID := int(counter.Add(1) - 1)

					a.Receive(func(event xs.Event) {
						spy.Call(localID, event)
					})

					return func() {}
				}),
			}}},
		},
		On: map[string]xs.Transitions{
			"RESTART": {{Target: ".listening"}},
		},
	})

	actorRef := xs.CreateActor(machine).Start()

	actorRef.Send(xs.Ev("RESTART"))
	listener := actorRef.System().Get("listener")
	require.NotNil(t, listener)
	listener.Send(xs.Ev("a"))

	assert.Equal(t, [][]any{
		{1, xs.E{"type": "a"}},
	}, spy.Calls())
}

// JS: system > should be able to send an event to an ancestor with a registered `systemId` from an initial entry action
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/system.test.ts#L522
func TestSystem_ShouldBeAbleToSendAnEventToAnAncestorWithARegisteredSystemIdFromAnInitialEntryAction(t *testing.T) {
	spy := newSpy()

	child := xs.CreateMachine(xs.MachineConfig[any]{
		Entry: xs.Actions{xs.SendTo(
			xs.NewExpr(func(a xs.ExprArgs[any]) any { return a.System.Get("myRoot") }),
			xs.Ev("EV"),
		)},
	})

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Invoke: []xs.InvokeConfig{{Logic: child}},
		On: map[string]xs.Transitions{
			"EV": {{Actions: xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[any]) { spy.Call(a) })}}},
		},
	})
	xs.CreateActor(machine, xs.WithSystemID("myRoot")).Start()

	assert.Equal(t, 1, spy.Count())
}

// JS: system > system ID should be accessible on the actor
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/system.test.ts#L546
func TestSystem_SystemIDShouldBeAccessibleOnTheActor(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{})
	actor := xs.CreateActor(machine, xs.WithSystemID("test"))
	assert.Equal(t, "test", actor.SystemID())
}

// JS: system > should give a list of runnings actors
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/system.test.ts#L552
func TestSystem_ShouldGiveAListOfRunningsActors(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "root",
		Initial: "happy path",
		States: xs.States{
			{
				Key: "happy path",
				Entry: xs.Actions{xs.SpawnChild(
					xs.CreateMachine(xs.MachineConfig[any]{}),
					xs.SpawnOptions{SystemID: "child1"},
				)},
				Invoke: []xs.InvokeConfig{{
					Logic:    xs.CreateMachine(xs.MachineConfig[any]{ID: "machine"}),
					SystemID: "child2",
				}},
				On: map[string]xs.Transitions{
					"stopChild1": {{Target: "sad path"}},
				},
			},
			{
				Key: "sad path",
				Entry: xs.Actions{xs.StopChild(xs.NewExpr(func(a xs.ExprArgs[any]) any {
					return a.System.Get("child1")
				}))},
			},
		},
	})

	actor := xs.CreateActor(machine).Start()

	assert.Equal(t, map[string]xs.ActorRef{
		"child1": actor.System().Get("child1"),
		"child2": actor.System().Get("child2"),
	}, actor.System().GetAll())

	actor.Send(xs.Ev("stopChild1"))

	// toEqual({}): no registered actors left (nil or empty map).
	assert.Empty(t, actor.System().GetAll())
}

// JS: system > should unregister nested child systemIds when stopping a parent actor
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/system.test.ts#L589
func TestSystem_ShouldUnregisterNestedChildSystemIdsWhenStoppingAParentActor(t *testing.T) {
	subchild := xs.CreateMachine(xs.MachineConfig[any]{})

	child := xs.NewSetup[any](xs.Implementations{
		Actors: map[string]xs.ActorLogic{"subchild": subchild},
	}).CreateMachine(xs.MachineConfig[any]{
		ID: "childSystem",
		Invoke: []xs.InvokeConfig{{
			Src:      "subchild",
			SystemID: "subchild",
		}},
	})

	parent := xs.NewSetup[any](xs.Implementations{
		Actors: map[string]xs.ActorLogic{"child": child},
	}).CreateMachine(xs.MachineConfig[any]{
		Entry: xs.Actions{xs.SpawnChild("child", xs.SpawnOptions{ID: "childId"})},
		On: map[string]xs.Transitions{
			"restart": {{Actions: xs.Actions{
				xs.StopChild("childId"),
				xs.SpawnChild("child", xs.SpawnOptions{ID: "childId"}),
			}}},
		},
	})

	root := xs.CreateActor(parent).Start()

	assert.NotNil(t, root.System().Get("subchild"))

	// This should not panic "Actor with system ID 'subchild' already exists"
	assert.NotPanics(t, func() { root.Send(xs.Ev("restart")) })

	assert.NotNil(t, root.System().Get("subchild"))
}
