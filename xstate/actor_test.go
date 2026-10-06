package xstate_test

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// actor1EventField reads a payload field of a dynamic xs.E event.
func actor1EventField(e xs.Event, key string) any {
	m, ok := e.(xs.E)
	if !ok {
		return nil
	}
	return m[key]
}

// actor1ObservableCtx returns event.snapshot.context of an
// "xstate.snapshot.*" event emitted by an observable child of ints.
func actor1ObservableCtx(e xs.Event) (int, bool) {
	ev, ok := e.(xs.SnapshotEvent)
	if !ok {
		return 0, false
	}
	s, ok := ev.Snapshot.(*xs.ObservableSnapshot[int])
	if !ok {
		return 0, false
	}
	return s.Context, true
}

// JS: spawning machines > should spawn machines
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actor.test.ts#L123
func TestActor_SpawningMachines_ShouldSpawnMachines(t *testing.T) {
	type ctx struct{ TodoRefs map[any]xs.ActorRef }

	sig := newSignal()
	todoMachine := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "todo",
		Initial: "incomplete",
		States: xs.States{
			{Key: "incomplete", On: map[string]xs.Transitions{"SET_COMPLETE": {{Target: "complete"}}}},
			{Key: "complete", Entry: xs.Actions{xs.SendParent(xs.Ev("TODO_COMPLETED"))}},
		},
	})

	todosMachine := xs.CreateMachine(xs.MachineConfig[ctx]{
		ID:      "todos",
		Context: ctx{TodoRefs: map[any]xs.ActorRef{}},
		Initial: "active",
		States: xs.States{
			{Key: "active", On: map[string]xs.Transitions{"TODO_COMPLETED": {{Target: "success"}}}},
			{Key: "success", Type: xs.Final},
		},
		On: map[string]xs.Transitions{
			"ADD": {{Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
				next := make(map[any]xs.ActorRef, len(a.Context.TodoRefs)+1)
				for k, v := range a.Context.TodoRefs {
					next[k] = v
				}
				next[actor1EventField(a.Event, "id")] = a.Spawn(todoMachine)
				return ctx{TodoRefs: next}
			})}}},
			"SET_COMPLETE": {{Actions: xs.Actions{xs.SendTo(
				xs.NewExpr(func(a xs.ExprArgs[ctx]) any {
					return a.Context.TodoRefs[actor1EventField(a.Event, "id")]
				}),
				xs.Ev("SET_COMPLETE"),
			)}}},
		},
	})
	service := xs.CreateActor(todosMachine)
	service.Subscribe(xs.Observer[*xs.MachineSnapshot[ctx]]{Complete: sig.Resolve})
	service.Start()

	service.Send(xs.E{"type": "ADD", "id": 42})
	service.Send(xs.E{"type": "SET_COMPLETE", "id": 42})
	sig.Wait(t)
}

// JS: spawning machines > should spawn referenced machines
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actor.test.ts#L188
func TestActor_SpawningMachines_ShouldSpawnReferencedMachines(t *testing.T) {
	type ctx struct{ Ref xs.ActorRef }

	childMachine := xs.CreateMachine(xs.MachineConfig[any]{
		Entry: xs.Actions{xs.SendParent(xs.Ev("DONE"))},
	})

	parentMachine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Context: ctx{Ref: nil},
		Initial: "waiting",
		States: xs.States{
			{
				Key: "waiting",
				Entry: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
					return ctx{Ref: a.Spawn("child")}
				})},
				On: map[string]xs.Transitions{"DONE": {{Target: "success"}}},
			},
			{Key: "success", Type: xs.Final},
		},
	}, xs.Implementations{Actors: map[string]xs.ActorLogic{"child": childMachine}})

	actor := xs.CreateActor(parentMachine)
	actor.Start()
	assert.Equal(t, "success", actor.GetSnapshot().Value)
}

// JS: spawning machines > should allow bidirectional communication between parent/child actors
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actor.test.ts#L225
func TestActor_SpawningMachines_ShouldAllowBidirectionalCommunicationBetweenParentChildActors(t *testing.T) {
	// describe-level serverMachine / clientMachine (JS L54-121)
	type clientContext struct{ Server xs.ActorRef }

	serverMachine := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "server",
		Initial: "waitPing",
		States: xs.States{
			{Key: "waitPing", On: map[string]xs.Transitions{"PING": {{Target: "sendPong"}}}},
			{
				Key:   "sendPong",
				Entry: xs.Actions{xs.SendParent(xs.Ev("PONG")), xs.Raise(xs.Ev("SUCCESS"))},
				On:    map[string]xs.Transitions{"SUCCESS": {{Target: "waitPing"}}},
			},
		},
	})

	clientMachine := xs.CreateMachine(xs.MachineConfig[clientContext]{
		ID:      "client",
		Initial: "init",
		Context: clientContext{Server: nil},
		States: xs.States{
			{
				Key: "init",
				Entry: xs.Actions{
					xs.Assign(func(a xs.AssignArgs[clientContext]) clientContext {
						c := a.Context
						c.Server = a.Spawn(serverMachine)
						return c
					}),
					xs.Raise(xs.Ev("SUCCESS")),
				},
				On: map[string]xs.Transitions{"SUCCESS": {{Target: "sendPing"}}},
			},
			{
				Key: "sendPing",
				Entry: xs.Actions{
					xs.SendTo(xs.NewExpr(func(a xs.ExprArgs[clientContext]) any { return a.Context.Server }), xs.Ev("PING")),
					xs.Raise(xs.Ev("SUCCESS")),
				},
				On: map[string]xs.Transitions{"SUCCESS": {{Target: "waitPong"}}},
			},
			{Key: "waitPong", On: map[string]xs.Transitions{"PONG": {{Target: "complete"}}}},
			{Key: "complete", Type: xs.Final},
		},
	})

	sig := newSignal()
	actor := xs.CreateActor(clientMachine)
	actor.Subscribe(xs.Observer[*xs.MachineSnapshot[clientContext]]{Complete: sig.Resolve})
	actor.Start()
	sig.Wait(t)
}

// JS: spawning promises > should be able to spawn a promise
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actor.test.ts#L241
func TestActor_SpawningPromises_ShouldBeAbleToSpawnAPromise(t *testing.T) {
	type ctx struct{ PromiseRef xs.ActorRef }

	sig := newSignal()
	promiseMachine := xs.CreateMachine(xs.MachineConfig[ctx]{
		ID:      "promise",
		Initial: "idle",
		Context: ctx{PromiseRef: nil},
		States: xs.States{
			{
				Key: "idle",
				Entry: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
					ref := a.Spawn(
						xs.FromPromise(func(_ context.Context, _ xs.PromiseArgs) (string, error) {
							return "response", nil
						}),
						xs.SpawnOptions{ID: "my-promise"},
					)
					return ctx{PromiseRef: ref}
				})},
				On: map[string]xs.Transitions{
					"xstate.done.actor.my-promise": {{
						Target: "success",
						Guard: xs.GuardFunc(func(a xs.GuardArgs[ctx]) bool {
							ev, ok := a.Event.(xs.DoneActorEvent)
							return ok && ev.Output == "response"
						}),
					}},
				},
			},
			{Key: "success", Type: xs.Final},
		},
	})

	promiseService := xs.CreateActor(promiseMachine)
	promiseService.Subscribe(xs.Observer[*xs.MachineSnapshot[ctx]]{Complete: sig.Resolve})
	promiseService.Start()
	sig.Wait(t)
}

// JS: spawning promises > should be able to spawn a referenced promise
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actor.test.ts#L293
func TestActor_SpawningPromises_ShouldBeAbleToSpawnAReferencedPromise(t *testing.T) {
	type ctx struct{ PromiseRef xs.ActorRef }

	sig := newSignal()
	promiseMachine := xs.NewSetup[ctx](xs.Implementations{
		Actors: map[string]xs.ActorLogic{
			"somePromise": xs.FromPromise(func(_ context.Context, _ xs.PromiseArgs) (string, error) {
				return "response", nil
			}),
		},
	}).CreateMachine(xs.MachineConfig[ctx]{
		ID:      "promise",
		Initial: "idle",
		Context: ctx{PromiseRef: nil},
		States: xs.States{
			{
				Key: "idle",
				Entry: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
					return ctx{PromiseRef: a.Spawn("somePromise", xs.SpawnOptions{ID: "my-promise"})}
				})},
				On: map[string]xs.Transitions{
					"xstate.done.actor.my-promise": {{
						Target: "success",
						Guard: xs.GuardFunc(func(a xs.GuardArgs[ctx]) bool {
							ev, ok := a.Event.(xs.DoneActorEvent)
							return ok && ev.Output == "response"
						}),
					}},
				},
			},
			{Key: "success", Type: xs.Final},
		},
	})

	promiseService := xs.CreateActor(promiseMachine)
	promiseService.Subscribe(xs.Observer[*xs.MachineSnapshot[ctx]]{Complete: sig.Resolve})
	promiseService.Start()
	sig.Wait(t)
}

// JS: spawning callbacks > should be able to spawn an actor from a callback
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actor.test.ts#L340
func TestActor_SpawningCallbacks_ShouldBeAbleToSpawnAnActorFromACallback(t *testing.T) {
	type ctx struct{ CallbackRef xs.ActorRef }

	sig := newSignal()
	callbackMachine := xs.CreateMachine(xs.MachineConfig[ctx]{
		ID:      "callback",
		Initial: "idle",
		Context: ctx{CallbackRef: nil},
		States: xs.States{
			{
				Key: "idle",
				Entry: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
					return ctx{CallbackRef: a.Spawn(xs.FromCallback(func(cb xs.CallbackArgs) func() {
						cb.Receive(func(event xs.Event) {
							if event.EventType() == "START" {
								time.AfterFunc(ms(10), func() {
									cb.SendBack(xs.Ev("SEND_BACK"))
								})
							}
						})
						return nil
					}))}
				})},
				On: map[string]xs.Transitions{
					"START_CB": {{Actions: xs.Actions{xs.SendTo(
						xs.NewExpr(func(a xs.ExprArgs[ctx]) any { return a.Context.CallbackRef }),
						xs.Ev("START"),
					)}}},
					"SEND_BACK": {{Target: "success"}},
				},
			},
			{Key: "success", Type: xs.Final},
		},
	})

	callbackService := xs.CreateActor(callbackMachine)
	callbackService.Subscribe(xs.Observer[*xs.MachineSnapshot[ctx]]{Complete: sig.Resolve})
	callbackService.Start()
	callbackService.Send(xs.Ev("START_CB"))
	sig.Wait(t)
}

// JS: spawning callbacks > should not deliver events sent to the parent after the callback actor gets stopped
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actor.test.ts#L396
func TestActor_SpawningCallbacks_ShouldNotDeliverEventsSentToParentAfterCallbackActorGetsStopped(t *testing.T) {
	s := newSpy()

	var mu sync.Mutex
	var sendToParent func()

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{
				Key: "a",
				Invoke: []xs.InvokeConfig{{
					Logic: xs.FromCallback(func(cb xs.CallbackArgs) func() {
						mu.Lock()
						sendToParent = func() { cb.SendBack(xs.Ev("FROM_CALLBACK")) }
						mu.Unlock()
						return nil
					}),
				}},
				On: map[string]xs.Transitions{"NEXT": {{Target: "b"}}},
			},
			{Key: "b"},
		},
		On: map[string]xs.Transitions{
			"FROM_CALLBACK": {{Actions: xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[any]) {
				s.Call(a)
			})}}},
		},
	})

	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.Ev("NEXT"))

	mu.Lock()
	send := sendToParent
	mu.Unlock()
	require.NotNil(t, send)
	send()

	assert.Equal(t, 0, s.Count())
}

// JS: spawning observables > should spawn an observable
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actor.test.ts#L436
func TestActor_SpawningObservables_ShouldSpawnAnObservable(t *testing.T) {
	type ctx struct{ ObservableRef xs.ActorRef }

	sig := newSignal()
	observableLogic := xs.FromObservable(func(xs.ObservableArgs) xs.Subscribable[int] { return rxInterval(10) })
	observableMachine := xs.CreateMachine(xs.MachineConfig[ctx]{
		ID:      "observable",
		Initial: "idle",
		Context: ctx{ObservableRef: nil},
		States: xs.States{
			{
				Key: "idle",
				Entry: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
					ref := a.Spawn(observableLogic, xs.SpawnOptions{ID: "int", SyncSnapshot: true})
					return ctx{ObservableRef: ref}
				})},
				On: map[string]xs.Transitions{
					"xstate.snapshot.int": {{
						Target: "success",
						Guard: xs.GuardFunc(func(a xs.GuardArgs[ctx]) bool {
							v, ok := actor1ObservableCtx(a.Event)
							return ok && v == 5
						}),
					}},
				},
			},
			{Key: "success", Type: xs.Final},
		},
	})

	observableService := xs.CreateActor(observableMachine)
	observableService.Subscribe(xs.Observer[*xs.MachineSnapshot[ctx]]{Complete: sig.Resolve})
	observableService.Start()
	sig.Wait(t)
}

// JS: spawning observables > should spawn a referenced observable
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actor.test.ts#L481
func TestActor_SpawningObservables_ShouldSpawnAReferencedObservable(t *testing.T) {
	type ctx struct{ ObservableRef xs.ActorRef }

	sig := newSignal()
	observableMachine := xs.CreateMachine(xs.MachineConfig[ctx]{
		ID:      "observable",
		Initial: "idle",
		Context: ctx{ObservableRef: nil},
		States: xs.States{
			{
				Key: "idle",
				Entry: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
					return ctx{ObservableRef: a.Spawn("interval", xs.SpawnOptions{ID: "int", SyncSnapshot: true})}
				})},
				On: map[string]xs.Transitions{
					"xstate.snapshot.int": {{
						Target: "success",
						Guard: xs.GuardFunc(func(a xs.GuardArgs[ctx]) bool {
							v, ok := actor1ObservableCtx(a.Event)
							return ok && v == 5
						}),
					}},
				},
			},
			{Key: "success", Type: xs.Final},
		},
	}, xs.Implementations{Actors: map[string]xs.ActorLogic{
		"interval": xs.FromObservable(func(xs.ObservableArgs) xs.Subscribable[int] { return rxInterval(10) }),
	}})

	observableService := xs.CreateActor(observableMachine)
	observableService.Subscribe(xs.Observer[*xs.MachineSnapshot[ctx]]{Complete: sig.Resolve})
	observableService.Start()
	sig.Wait(t)
}

// JS: spawning observables > should read the latest snapshot of the event's origin while handling that event
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actor.test.ts#L526
func TestActor_SpawningObservables_ShouldReadLatestSnapshotOfEventsOriginWhileHandlingThatEvent(t *testing.T) {
	type ctx struct{ ObservableRef xs.ActorRef }

	sig := newSignal()
	observableLogic := xs.FromObservable(func(xs.ObservableArgs) xs.Subscribable[int] { return rxInterval(10) })
	observableMachine := xs.CreateMachine(xs.MachineConfig[ctx]{
		ID:      "observable",
		Initial: "idle",
		Context: ctx{ObservableRef: nil},
		States: xs.States{
			{
				Key: "idle",
				Entry: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
					ref := a.Spawn(observableLogic, xs.SpawnOptions{ID: "int", SyncSnapshot: true})
					return ctx{ObservableRef: ref}
				})},
				On: map[string]xs.Transitions{
					"xstate.snapshot.int": {{
						Target: "success",
						Guard: xs.GuardFunc(func(a xs.GuardArgs[ctx]) bool {
							v, ok := actor1ObservableCtx(a.Event)
							if !ok || v != 1 {
								return false
							}
							latest, ok := a.Context.ObservableRef.AnySnapshot().(*xs.ObservableSnapshot[int])
							return ok && latest.Context == 1
						}),
					}},
				},
			},
			{Key: "success", Type: xs.Final},
		},
	})

	observableService := xs.CreateActor(observableMachine)
	observableService.Subscribe(xs.Observer[*xs.MachineSnapshot[ctx]]{Complete: sig.Resolve})
	observableService.Start()
	sig.Wait(t)
}

// JS: spawning observables > should notify direct child listeners with final snapshot before it gets stopped
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actor.test.ts#L576
func TestActor_SpawningObservables_ShouldNotifyDirectChildListenersWithFinalSnapshotBeforeStopped(t *testing.T) {
	intervalActor := xs.FromObservable(func(xs.ObservableArgs) xs.Subscribable[int] { return rxInterval(10) })

	parentMachine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "active",
		States: xs.States{
			{
				Key: "active",
				Invoke: []xs.InvokeConfig{{
					ID:  "childActor",
					Src: "interval",
					OnSnapshot: xs.Transitions{{
						Target: "success",
						Guard: xs.GuardFunc(func(a xs.GuardArgs[any]) bool {
							v, ok := actor1ObservableCtx(a.Event)
							return ok && v == 3
						}),
					}},
				}},
			},
			{Key: "success", Type: xs.Final},
		},
	}, xs.Implementations{Actors: map[string]xs.ActorLogic{"interval": intervalActor}})

	actorRef := xs.CreateActor(parentMachine)
	actorRef.Start()

	_, err := xs.WaitFor(context.Background(), actorRef, func(s *xs.MachineSnapshot[any]) bool {
		return s.Matches("active")
	}).Wait()
	require.NoError(t, err)

	s := newSpy()

	child := actorRef.GetSnapshot().Children["childActor"]
	require.NotNil(t, child)
	xs.As[*xs.ObservableSnapshot[int]](child).SubscribeNext(func(data *xs.ObservableSnapshot[int]) {
		s.Call(data.Context)
	})

	_, err = xs.WaitFor(context.Background(), actorRef, func(s *xs.MachineSnapshot[any]) bool {
		return s.Status != xs.StatusActive
	}).Wait()
	require.NoError(t, err)

	// expect(spy).toHaveBeenCalledWith(3)
	assert.Contains(t, s.Calls(), []any{3})
}

// JS: spawning observables > should not notify direct child listeners after it gets stopped
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actor.test.ts#L630
func TestActor_SpawningObservables_ShouldNotNotifyDirectChildListenersAfterItGetsStopped(t *testing.T) {
	intervalActor := xs.FromObservable(func(xs.ObservableArgs) xs.Subscribable[int] { return rxInterval(10) })

	parentMachine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "active",
		States: xs.States{
			{
				Key: "active",
				Invoke: []xs.InvokeConfig{{
					ID:  "childActor",
					Src: "interval",
					OnSnapshot: xs.Transitions{{
						Target: "success",
						Guard: xs.GuardFunc(func(a xs.GuardArgs[any]) bool {
							v, ok := actor1ObservableCtx(a.Event)
							return ok && v == 3
						}),
					}},
				}},
			},
			{Key: "success", Type: xs.Final},
		},
	}, xs.Implementations{Actors: map[string]xs.ActorLogic{"interval": intervalActor}})

	actorRef := xs.CreateActor(parentMachine)
	actorRef.Start()

	_, err := xs.WaitFor(context.Background(), actorRef, func(s *xs.MachineSnapshot[any]) bool {
		return s.Matches("active")
	}).Wait()
	require.NoError(t, err)

	s := newSpy()

	child := actorRef.GetSnapshot().Children["childActor"]
	require.NotNil(t, child)
	xs.As[*xs.ObservableSnapshot[int]](child).SubscribeNext(func(data *xs.ObservableSnapshot[int]) {
		s.Call(data)
	})

	_, err = xs.WaitFor(context.Background(), actorRef, func(s *xs.MachineSnapshot[any]) bool {
		return s.Status != xs.StatusActive
	}).Wait()
	require.NoError(t, err)
	// spy.mockClear(): only calls recorded from here on count.
	callsAtClear := s.Count()

	// wait for potential next event from the interval actor
	sleep(15)

	assert.Equal(t, callsAtClear, s.Count(), "spy must not be called after the child was stopped")
}

// JS: spawning event observables > should spawn an event observable
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actor.test.ts#L690
func TestActor_SpawningEventObservables_ShouldSpawnAnEventObservable(t *testing.T) {
	type ctx struct{ ObservableRef xs.ActorRef }

	sig := newSignal()
	eventObservableLogic := xs.FromEventObservable(func(xs.ObservableArgs) xs.Subscribable[xs.Event] {
		return rxMap(rxInterval(10), func(val int) xs.Event { return xs.E{"type": "COUNT", "val": val} })
	})
	observableMachine := xs.CreateMachine(xs.MachineConfig[ctx]{
		ID:      "observable",
		Initial: "idle",
		Context: ctx{ObservableRef: nil},
		States: xs.States{
			{
				Key: "idle",
				Entry: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
					ref := a.Spawn(eventObservableLogic, xs.SpawnOptions{ID: "int"})
					return ctx{ObservableRef: ref}
				})},
				On: map[string]xs.Transitions{
					"COUNT": {{
						Target: "success",
						Guard: xs.GuardFunc(func(a xs.GuardArgs[ctx]) bool {
							return actor1EventField(a.Event, "val") == 5
						}),
					}},
				},
			},
			{Key: "success", Type: xs.Final},
		},
	})

	observableService := xs.CreateActor(observableMachine)
	observableService.Subscribe(xs.Observer[*xs.MachineSnapshot[ctx]]{Complete: sig.Resolve})
	observableService.Start()
	sig.Wait(t)
}

// JS: spawning event observables > should spawn a referenced event observable
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actor.test.ts#L734
func TestActor_SpawningEventObservables_ShouldSpawnAReferencedEventObservable(t *testing.T) {
	type ctx struct{ ObservableRef xs.ActorRef }

	sig := newSignal()
	observableMachine := xs.CreateMachine(xs.MachineConfig[ctx]{
		ID:      "observable",
		Initial: "idle",
		Context: ctx{ObservableRef: nil},
		States: xs.States{
			{
				Key: "idle",
				Entry: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
					return ctx{ObservableRef: a.Spawn("interval", xs.SpawnOptions{ID: "int"})}
				})},
				On: map[string]xs.Transitions{
					"COUNT": {{
						Target: "success",
						Guard: xs.GuardFunc(func(a xs.GuardArgs[ctx]) bool {
							return actor1EventField(a.Event, "val") == 5
						}),
					}},
				},
			},
			{Key: "success", Type: xs.Final},
		},
	}, xs.Implementations{Actors: map[string]xs.ActorLogic{
		"interval": xs.FromEventObservable(func(xs.ObservableArgs) xs.Subscribable[xs.Event] {
			return rxMap(rxInterval(10), func(val int) xs.Event { return xs.E{"type": "COUNT", "val": val} })
		}),
	}})

	observableService := xs.CreateActor(observableMachine)
	observableService.Subscribe(xs.Observer[*xs.MachineSnapshot[ctx]]{Complete: sig.Resolve})
	observableService.Start()
	sig.Wait(t)
}

// JS: communicating with spawned actors > should treat an interpreter as an actor
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actor.test.ts#L782
func TestActor_CommunicatingWithSpawnedActors_ShouldTreatAnInterpreterAsAnActor(t *testing.T) {
	type ctx struct{ ExistingRef xs.ActorRef }

	sig := newSignal()
	existingMachine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "inactive",
		States: xs.States{
			{Key: "inactive", On: map[string]xs.Transitions{"ACTIVATE": {{Target: "active"}}}},
			{
				Key: "active",
				Entry: xs.Actions{xs.SendTo(
					xs.NewExpr(func(a xs.ExprArgs[any]) any { return actor1EventField(a.Event, "origin") }),
					xs.Ev("EXISTING.DONE"),
				)},
			},
		},
	})

	existingService := xs.CreateActor(existingMachine).Start()

	parentMachine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Initial: "pending",
		Context: ctx{ExistingRef: nil},
		States: xs.States{
			{
				Key: "pending",
				Entry: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
					// No need to spawn an existing service:
					return ctx{ExistingRef: existingService}
				})},
				On: map[string]xs.Transitions{"EXISTING.DONE": {{Target: "success"}}},
				After: map[string]xs.Transitions{
					"100": {{Actions: xs.Actions{xs.SendTo(
						xs.NewExpr(func(a xs.ExprArgs[ctx]) any { return a.Context.ExistingRef }),
						xs.NewExpr(func(a xs.ExprArgs[ctx]) any {
							return xs.E{"type": "ACTIVATE", "origin": a.Self}
						}),
					)}}},
				},
			},
			{Key: "success", Type: xs.Final},
		},
	})

	parentService := xs.CreateActor(parentMachine)
	parentService.Subscribe(xs.Observer[*xs.MachineSnapshot[ctx]]{Complete: sig.Resolve})
	parentService.Start()
	sig.Wait(t)
}

// JS: actors > should only spawn actors defined on initial state once
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actor.test.ts#L852
func TestActor_Actors_ShouldOnlySpawnActorsDefinedOnInitialStateOnce(t *testing.T) {
	type ctx struct {
		Items []int
		Refs  []xs.ActorRef
	}

	var count atomic.Int64

	startMachine := xs.CreateMachine(xs.MachineConfig[ctx]{
		ID:      "start",
		Initial: "start",
		Context: ctx{Items: []int{0, 1, 2, 3}, Refs: []xs.ActorRef{}},
		States: xs.States{
			{
				Key: "start",
				Entry: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
					count.Add(1)
					refs := make([]xs.ActorRef, 0, len(a.Context.Items))
					for _, item := range a.Context.Items {
						refs = append(refs, a.Spawn(xs.FromPromise(func(_ context.Context, _ xs.PromiseArgs) (int, error) {
							return item, nil
						})))
					}
					return ctx{Items: a.Context.Items, Refs: refs}
				})},
			},
		},
	})

	actor := xs.CreateActor(startMachine)
	actor.SubscribeNext(func(*xs.MachineSnapshot[ctx]) {
		assert.Equal(t, int64(1), count.Load())
	})
	actor.Start()
}

// JS: actors > should spawn an actor in an initial state of a child that gets invoked in the initial state of a parent when the parent gets started
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actor.test.ts#L886
func TestActor_Actors_ShouldSpawnActorInInitialStateOfChildInvokedInInitialStateOfParentOnStart(t *testing.T) {
	type testContext struct{ Promise xs.ActorRef }

	var spawnCounter atomic.Int64

	child := xs.CreateMachine(xs.MachineConfig[testContext]{
		Initial: "bar",
		Context: testContext{},
		States: xs.States{
			{
				Key: "bar",
				Entry: xs.Actions{xs.Assign(func(a xs.AssignArgs[testContext]) testContext {
					return testContext{Promise: a.Spawn(xs.FromPromise(func(_ context.Context, _ xs.PromiseArgs) (string, error) {
						spawnCounter.Add(1)
						return "answer", nil
					}))}
				})},
			},
		},
	})

	parent := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "foo",
		States: xs.States{
			{
				Key:    "foo",
				Invoke: []xs.InvokeConfig{{Logic: child, OnDone: xs.Transitions{{Target: "end"}}}},
			},
			{Key: "end", Type: xs.Final},
		},
	})
	xs.CreateActor(parent).Start()
	// JS asserts synchronously after start(): the promise creator runs
	// synchronously in JS, but on its own goroutine in Go (logic.go FromPromise).
	// Wait for the first call, then confirm it is not called a second time.
	assert.Eventually(t, func() bool { return spawnCounter.Load() >= 1 }, time.Second, time.Millisecond)
	sleep(10)
	assert.Equal(t, int64(1), spawnCounter.Load())
}

// JS: actors > should only spawn an initial actor once when it synchronously responds with an event
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actor.test.ts#L930
func TestActor_Actors_ShouldOnlySpawnAnInitialActorOnceWhenItSynchronouslyRespondsWithAnEvent(t *testing.T) {
	type ctx struct{ Ref xs.ActorRef }

	var spawnCalled atomic.Int64
	anotherMachine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "hello",
		States: xs.States{
			{Key: "hello", Entry: xs.Actions{xs.SendParent(xs.Ev("ping"))}},
		},
	})

	testMachine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Initial: "testing",
		ContextFn: func(a xs.ContextArgs) ctx {
			n := spawnCalled.Add(1)
			// throw in case of an infinite loop
			if !assert.Equal(t, int64(1), n) {
				panic("spawned more than once")
			}
			return ctx{Ref: a.Spawn(anotherMachine)}
		},
		States: xs.States{
			{Key: "testing", On: map[string]xs.Transitions{"ping": {{Target: "done"}}}},
			{Key: "done"},
		},
	})

	service := xs.CreateActor(testMachine).Start()
	assert.Equal(t, "done", service.GetSnapshot().Value)
}

// JS: actors > should spawn null actors if not used within a service
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actor.test.ts#L968
func TestActor_Actors_ShouldSpawnNullActorsIfNotUsedWithinAService(t *testing.T) {
	type ctx struct{ Ref xs.ActorRef }

	nullActorMachine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Initial: "foo",
		Context: ctx{Ref: nil},
		States: xs.States{
			{
				Key: "foo",
				Entry: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
					return ctx{Ref: a.Spawn(xs.FromPromise(func(_ context.Context, _ xs.PromiseArgs) (int, error) {
						return 42, nil
					}))}
				})},
			},
		},
	})

	// expect(createActor(nullActorMachine).getSnapshot().context.ref!.id).toBe('null'); // TODO: identify null actors
	// expect(ref.send).toBeDefined(): the ref exists and is a usable ActorRef.
	ref := xs.CreateActor(nullActorMachine).GetSnapshot().Context.Ref
	assert.NotNil(t, ref)
}

// JS: actors > should stop multiple inline spawned actors that have no explicit ids
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actor.test.ts#L988
func TestActor_Actors_ShouldStopMultipleInlineSpawnedActorsThatHaveNoExplicitIds(t *testing.T) {
	type ctx struct{ Ref1, Ref2 xs.ActorRef }

	cleanup1 := newSpy()
	cleanup2 := newSpy()

	parent := xs.CreateMachine(xs.MachineConfig[ctx]{
		ContextFn: func(a xs.ContextArgs) ctx {
			return ctx{
				Ref1: a.Spawn(xs.FromCallback(func(xs.CallbackArgs) func() { return func() { cleanup1.Call() } })),
				Ref2: a.Spawn(xs.FromCallback(func(xs.CallbackArgs) func() { return func() { cleanup2.Call() } })),
			}
		},
	})
	actorRef := xs.CreateActor(parent).Start()

	assert.Len(t, actorRef.GetSnapshot().Children, 2)

	actorRef.Stop()

	assert.Equal(t, 1, cleanup1.Count())
	assert.Equal(t, 1, cleanup2.Count())
}

// JS: actors > should stop multiple referenced spawned actors that have no explicit ids
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actor.test.ts#L1008
func TestActor_Actors_ShouldStopMultipleReferencedSpawnedActorsThatHaveNoExplicitIds(t *testing.T) {
	type ctx struct{ Ref1, Ref2 xs.ActorRef }

	cleanup1 := newSpy()
	cleanup2 := newSpy()

	parent := xs.CreateMachine(xs.MachineConfig[ctx]{
		ContextFn: func(a xs.ContextArgs) ctx {
			return ctx{
				Ref1: a.Spawn("child1"),
				Ref2: a.Spawn("child2"),
			}
		},
	}, xs.Implementations{Actors: map[string]xs.ActorLogic{
		"child1": xs.FromCallback(func(xs.CallbackArgs) func() { return func() { cleanup1.Call() } }),
		"child2": xs.FromCallback(func(xs.CallbackArgs) func() { return func() { cleanup2.Call() } }),
	}})
	actorRef := xs.CreateActor(parent).Start()

	assert.Len(t, actorRef.GetSnapshot().Children, 2)

	actorRef.Stop()

	assert.Equal(t, 1, cleanup1.Count())
	assert.Equal(t, 1, cleanup2.Count())
}

// JS: actors > with actor logic > should work with a transition function logic
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actor.test.ts#L1037
func TestActor_Actors_WithActorLogic_ShouldWorkWithATransitionFunctionLogic(t *testing.T) {
	type ctx struct{ Count xs.ActorRef }

	sig := newSignal()
	countLogic := xs.FromTransition(func(count int, event xs.Event, _ *xs.ActorScope) int {
		switch event.EventType() {
		case "INC":
			return count + 1
		case "DEC":
			return count - 1
		}
		return count
	}, func(xs.TransitionInitArgs) int { return 0 })

	childCount := func(c ctx) (int, bool) {
		if c.Count == nil {
			return 0, false
		}
		s, ok := c.Count.AnySnapshot().(*xs.TransitionSnapshot[int])
		if !ok {
			return 0, false
		}
		return s.Context, true
	}

	countMachine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Context: ctx{Count: nil},
		Entry: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
			return ctx{Count: a.Spawn(countLogic)}
		})},
		On: map[string]xs.Transitions{
			"INC": {{Actions: xs.Actions{xs.ForwardTo(
				xs.NewExpr(func(a xs.ExprArgs[ctx]) any { return a.Context.Count }),
			)}}},
		},
	})

	countService := xs.CreateActor(countMachine)
	countService.SubscribeNext(func(state *xs.MachineSnapshot[ctx]) {
		if v, ok := childCount(state.Context); ok && v == 2 {
			sig.Resolve()
		}
	})
	countService.Start()

	countService.Send(xs.Ev("INC"))
	countService.Send(xs.Ev("INC"))

	v, ok := childCount(countService.GetSnapshot().Context)
	assert.True(t, ok)
	assert.Equal(t, 2, v)

	sig.Wait(t)
}

// JS: actors > with actor logic > should work with a promise logic (fulfill)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actor.test.ts#L1083
func TestActor_Actors_WithActorLogic_ShouldWorkWithAPromiseLogicFulfill(t *testing.T) {
	type ctx struct{ Count xs.ActorRef }

	sig := newSignal()
	countMachine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Context: ctx{Count: nil},
		Entry: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
			return ctx{Count: a.Spawn(
				xs.FromPromise(func(_ context.Context, _ xs.PromiseArgs) (int, error) {
					sleep(0) // setTimeout(() => res(42))
					return 42, nil
				}),
				xs.SpawnOptions{ID: "test"},
			)}
		})},
		Initial: "pending",
		States: xs.States{
			{
				Key: "pending",
				On: map[string]xs.Transitions{
					"xstate.done.actor.test": {{
						Target: "success",
						Guard: xs.GuardFunc(func(a xs.GuardArgs[ctx]) bool {
							ev, ok := a.Event.(xs.DoneActorEvent)
							return ok && ev.Output == 42
						}),
					}},
				},
			},
			{Key: "success", Type: xs.Final},
		},
	})

	countService := xs.CreateActor(countMachine)
	countService.Subscribe(xs.Observer[*xs.MachineSnapshot[ctx]]{Complete: sig.Resolve})
	countService.Start()
	sig.Wait(t)
}

// JS: actors > with actor logic > should work with a promise logic (reject)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actor.test.ts#L1132
func TestActor_Actors_WithActorLogic_ShouldWorkWithAPromiseLogicReject(t *testing.T) {
	type ctx struct{ Count xs.ActorRef }

	sig := newSignal()
	// JS rejects with the string itself and compares with ===; the Go
	// rejection is this exact error value, compared by identity.
	errorMessage := errors.New("An error occurred")
	countMachine := xs.CreateMachine(xs.MachineConfig[ctx]{
		ContextFn: func(a xs.ContextArgs) ctx {
			return ctx{Count: a.Spawn(
				xs.FromPromise(func(_ context.Context, _ xs.PromiseArgs) (int, error) {
					sleep(1)
					return 0, errorMessage
				}),
				xs.SpawnOptions{ID: "test"},
			)}
		},
		Initial: "pending",
		States: xs.States{
			{
				Key: "pending",
				On: map[string]xs.Transitions{
					"xstate.error.actor.test": {{
						Target: "success",
						Guard: xs.GuardFunc(func(a xs.GuardArgs[ctx]) bool {
							ev, ok := a.Event.(xs.ErrorActorEvent)
							return ok && ev.Error == errorMessage
						}),
					}},
				},
			},
			{Key: "success", Type: xs.Final},
		},
	})

	countService := xs.CreateActor(countMachine)
	countService.Subscribe(xs.Observer[*xs.MachineSnapshot[ctx]]{Complete: sig.Resolve})
	countService.Start()
	sig.Wait(t)
}

// JS: actors > with actor logic > actor logic should have reference to the parent
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actor.test.ts#L1178
func TestActor_Actors_WithActorLogic_ActorLogicShouldHaveReferenceToTheParent(t *testing.T) {
	type ctx struct{ Ponger xs.ActorRef }

	sig := newSignal()
	pongLogic := &xs.Logic[*xs.BasicSnapshot[any]]{
		Transition: func(state *xs.BasicSnapshot[any], event xs.Event, scope *xs.ActorScope) *xs.BasicSnapshot[any] {
			if event.EventType() == "PING" {
				if p := scope.Self.Parent(); p != nil {
					p.Send(xs.Ev("PONG"))
				}
			}
			return state
		},
		GetInitialSnapshot: func(*xs.ActorScope, any) *xs.BasicSnapshot[any] {
			return &xs.BasicSnapshot[any]{Status: xs.StatusActive, Output: nil, Error: nil}
		},
		GetPersistedSnapshot: func(s *xs.BasicSnapshot[any]) any { return s },
	}

	pingMachine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Initial: "waiting",
		Context: ctx{Ponger: nil},
		Entry: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
			return ctx{Ponger: a.Spawn(pongLogic)}
		})},
		States: xs.States{
			{
				Key: "waiting",
				Entry: xs.Actions{xs.SendTo(
					xs.NewExpr(func(a xs.ExprArgs[ctx]) any { return a.Context.Ponger }),
					xs.Ev("PING"),
				)},
				Invoke: []xs.InvokeConfig{{ID: "ponger", Logic: pongLogic}},
				On:     map[string]xs.Transitions{"PONG": {{Target: "success"}}},
			},
			{Key: "success", Type: xs.Final},
		},
	})

	pingService := xs.CreateActor(pingMachine)
	pingService.Subscribe(xs.Observer[*xs.MachineSnapshot[ctx]]{Complete: sig.Resolve})
	pingService.Start()
	sig.Wait(t)
}

// JS: actors > should be able to spawn callback actors in (lazy) initial context
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actor.test.ts#L1235
func TestActor_Actors_ShouldBeAbleToSpawnCallbackActorsInLazyInitialContext(t *testing.T) {
	type ctx struct{ Ref xs.ActorRef }

	sig := newSignal()
	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		ContextFn: func(a xs.ContextArgs) ctx {
			return ctx{Ref: a.Spawn(xs.FromCallback(func(cb xs.CallbackArgs) func() {
				cb.SendBack(xs.Ev("TEST"))
				return nil
			}))}
		},
		Initial: "waiting",
		States: xs.States{
			{Key: "waiting", On: map[string]xs.Transitions{"TEST": {{Target: "success"}}}},
			{Key: "success", Type: xs.Final},
		},
	})

	actor := xs.CreateActor(machine)
	actor.Subscribe(xs.Observer[*xs.MachineSnapshot[ctx]]{Complete: sig.Resolve})
	actor.Start()
	sig.Wait(t)
}

// JS: actors > should be able to spawn machines in (lazy) initial context
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actor.test.ts#L1267
func TestActor_Actors_ShouldBeAbleToSpawnMachinesInLazyInitialContext(t *testing.T) {
	type ctx struct{ Ref xs.ActorRef }

	sig := newSignal()
	childMachine := xs.CreateMachine(xs.MachineConfig[any]{
		Entry: xs.Actions{xs.SendParent(xs.Ev("TEST"))},
	})

	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		ContextFn: func(a xs.ContextArgs) ctx {
			return ctx{Ref: a.Spawn(childMachine)}
		},
		Initial: "waiting",
		States: xs.States{
			{Key: "waiting", On: map[string]xs.Transitions{"TEST": {{Target: "success"}}}},
			{Key: "success", Type: xs.Final},
		},
	})

	actor := xs.CreateActor(machine)
	actor.Subscribe(xs.Observer[*xs.MachineSnapshot[ctx]]{Complete: sig.Resolve})
	actor.Start()
	sig.Wait(t)
}

// actor2Recorder mirrors the JS `actual: string[]` arrays that callback
// actors push into; guarded because actor callbacks may run off the test
// goroutine.
type actor2Recorder struct {
	mu    sync.Mutex
	items []string
}

func (r *actor2Recorder) push(s string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items = append(r.items, s)
}

// reset mirrors `actual.length = 0`.
func (r *actor2Recorder) reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items = nil
}

func (r *actor2Recorder) get() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, len(r.items))
	copy(out, r.items)
	return out
}

// actor2SyncCompleteObservable mirrors the JS `createEmptyObservable()`
// object literal: its subscribe synchronously calls observer.complete?.().
type actor2SyncCompleteObservable struct{}

func (actor2SyncCompleteObservable) Subscribe(observer xs.Observer[any]) xs.Subscription {
	if observer.Complete != nil {
		observer.Complete()
	}
	return xs.SubscriptionFunc(func() {})
}

// JS: actors > should not crash on child machine sync completion during self-initialization
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actor.test.ts#L1300
func TestActor_Actors_ShouldNotCrashOnChildMachineSyncCompletionDuringSelfInitialization(t *testing.T) {
	type ctx struct{ Child xs.ActorRef }

	childMachine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "idle",
		States: xs.States{
			{Key: "idle", Always: xs.Transitions{{Target: "stopped"}}},
			{Key: "stopped", Type: xs.Final},
		},
	})

	parentMachine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Context: ctx{Child: nil},
		Entry:   xs.Actions{xs.ActionRef{Type: "setup"}},
	}, xs.Implementations{
		Actions: map[string]xs.Action{
			"setup": xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
				c := a.Context
				c.Child = a.Spawn(childMachine)
				return c
			}),
		},
	})
	service := xs.CreateActor(parentMachine)
	assert.NotPanics(t, func() {
		service.Start()
	})
}

// JS: actors > should not crash on child promise-like sync completion during self-initialization
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actor.test.ts#L1341
func TestActor_Actors_ShouldNotCrashOnChildPromiseLikeSyncCompletionDuringSelfInitialization(t *testing.T) {
	type ctx struct{ Child xs.ActorRef }

	// JS returns a thenable `{ then: (fn) => fn(null) }` that resolves
	// synchronously with null; the Go equivalent resolves immediately with nil.
	promiseLogic := xs.FromPromise(func(_ context.Context, _ xs.PromiseArgs) (any, error) {
		return nil, nil
	})
	parentMachine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Context: ctx{Child: nil},
		Entry: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
			c := a.Context
			c.Child = a.Spawn(promiseLogic)
			return c
		})},
	})
	service := xs.CreateActor(parentMachine)
	assert.NotPanics(t, func() {
		service.Start()
	})
}

// JS: actors > should not crash on child observable sync completion during self-initialization
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actor.test.ts#L1362
func TestActor_Actors_ShouldNotCrashOnChildObservableSyncCompletionDuringSelfInitialization(t *testing.T) {
	type ctx struct{ Child xs.ActorRef }

	createEmptyObservable := func(_ xs.ObservableArgs) xs.Subscribable[any] {
		return actor2SyncCompleteObservable{}
	}

	emptyObservableLogic := xs.FromObservable(createEmptyObservable)

	parentMachine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Context: ctx{Child: nil},
		Entry: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
			c := a.Context
			c.Child = a.Spawn(emptyObservableLogic)
			return c
		})},
	})
	service := xs.CreateActor(parentMachine)
	assert.NotPanics(t, func() {
		service.Start()
	})
}

// JS: actors > should receive done event from an immediately completed observable when self-initializing
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actor.test.ts#L1390
func TestActor_Actors_ShouldReceiveDoneEventFromImmediatelyCompletedObservableWhenSelfInitializing(t *testing.T) {
	type ctx struct{ Child xs.ActorRef }

	emptyObservable := xs.FromObservable(func(_ xs.ObservableArgs) xs.Subscribable[any] {
		return rxEmpty[any]()
	})

	parentMachine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Context: ctx{Child: nil},
		Entry: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
			c := a.Context
			c.Child = a.Spawn(emptyObservable, xs.SpawnOptions{ID: "myactor"})
			return c
		})},
		Initial: "init",
		States: xs.States{
			{Key: "init", On: map[string]xs.Transitions{
				"xstate.done.actor.myactor": {{Target: "done"}},
			}},
			{Key: "done"},
		},
	})
	service := xs.CreateActor(parentMachine)

	service.Start()

	assert.Equal(t, "done", service.GetSnapshot().Value)
}

// JS: actors > should not restart a completed observable
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actor.test.ts#L1422
func TestActor_Actors_ShouldNotRestartACompletedObservable(t *testing.T) {
	var subscriptionCount atomic.Int32
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Invoke: []xs.InvokeConfig{{
			ID: "observable",
			Logic: xs.FromObservable(func(_ xs.ObservableArgs) xs.Subscribable[int] {
				subscriptionCount.Add(1)
				return rxOf(42)
			}),
		}},
	})

	actor := xs.CreateActor(machine).Start()
	persistedState := actor.GetPersistedSnapshot()

	xs.CreateActor(machine, xs.WithSnapshot(persistedState)).Start()

	// Will be 2 if the observable is resubscribed
	assert.Equal(t, int32(1), subscriptionCount.Load())
}

// JS: actors > should not restart a completed event observable
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actor.test.ts#L1445
func TestActor_Actors_ShouldNotRestartACompletedEventObservable(t *testing.T) {
	var subscriptionCount atomic.Int32
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Invoke: []xs.InvokeConfig{{
			ID: "observable",
			Logic: xs.FromEventObservable(func(_ xs.ObservableArgs) xs.Subscribable[xs.Event] {
				subscriptionCount.Add(1)
				return rxOf[xs.Event](xs.Ev("TEST"))
			}),
		}},
	})

	actor := xs.CreateActor(machine).Start()
	persistedState := actor.GetPersistedSnapshot()

	xs.CreateActor(machine, xs.WithSnapshot(persistedState)).Start()

	// Will be 2 if the event observable is resubscribed
	assert.Equal(t, int32(1), subscriptionCount.Load())
}

// JS: actors > should be able to restart a spawned actor within a single macrostep
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actor.test.ts#L1468
func TestActor_Actors_ShouldBeAbleToRestartASpawnedActorWithinASingleMacrostep(t *testing.T) {
	type ctx struct{ ActorRef xs.ActorRef }
	actual := &actor2Recorder{}
	invokeCounter := 0

	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Initial: "active",
		ContextFn: func(a xs.ContextArgs) ctx {
			invokeCounter++
			localID := invokeCounter

			return ctx{
				ActorRef: a.Spawn(
					xs.FromCallback(func(_ xs.CallbackArgs) func() {
						actual.push("start " + strconv.Itoa(localID))
						return func() {
							actual.push("stop " + strconv.Itoa(localID))
						}
					}),
					xs.SpawnOptions{ID: "callback-1"},
				),
			}
		},
		States: xs.States{
			{Key: "active", On: map[string]xs.Transitions{
				"update": {{Actions: xs.Actions{
					xs.StopChild(xs.NewExpr(func(a xs.ExprArgs[ctx]) any {
						return a.Context.ActorRef
					})),
					xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
						invokeCounter++
						localID := invokeCounter

						c := a.Context
						c.ActorRef = a.Spawn(
							xs.FromCallback(func(_ xs.CallbackArgs) func() {
								actual.push("start " + strconv.Itoa(localID))
								return func() {
									actual.push("stop " + strconv.Itoa(localID))
								}
							}),
							xs.SpawnOptions{ID: "callback-2"},
						)
						return c
					}),
				}}},
			}},
		},
	})

	service := xs.CreateActor(machine).Start()

	actual.reset()

	service.Send(xs.Ev("update"))

	assert.Equal(t, []string{"stop 1", "start 2"}, actual.get())
}

// JS: actors > should be able to restart a named spawned actor within a single macrostep when stopping by a ref
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actor.test.ts#L1535
func TestActor_Actors_ShouldRestartNamedSpawnedActorWithinSingleMacrostepWhenStoppingByRef(t *testing.T) {
	type ctx struct{ ActorRef xs.ActorRef }
	actual := &actor2Recorder{}
	invokeCounter := 0

	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Initial: "active",
		ContextFn: func(a xs.ContextArgs) ctx {
			invokeCounter++
			localID := invokeCounter

			return ctx{
				ActorRef: a.Spawn(
					xs.FromCallback(func(_ xs.CallbackArgs) func() {
						actual.push("start " + strconv.Itoa(localID))
						return func() {
							actual.push("stop " + strconv.Itoa(localID))
						}
					}),
					xs.SpawnOptions{ID: "my_name"},
				),
			}
		},
		States: xs.States{
			{Key: "active", On: map[string]xs.Transitions{
				"update": {{Actions: xs.Actions{
					xs.StopChild(xs.NewExpr(func(a xs.ExprArgs[ctx]) any { return a.Context.ActorRef })),
					xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
						invokeCounter++
						localID := invokeCounter

						c := a.Context
						c.ActorRef = a.Spawn(
							xs.FromCallback(func(_ xs.CallbackArgs) func() {
								actual.push("start " + strconv.Itoa(localID))
								return func() {
									actual.push("stop " + strconv.Itoa(localID))
								}
							}),
							xs.SpawnOptions{ID: "my_name"},
						)
						return c
					}),
				}}},
			}},
		},
	})

	service := xs.CreateActor(machine).Start()

	actual.reset()

	service.Send(xs.Ev("update"))

	assert.Equal(t, []string{"stop 1", "start 2"}, actual.get())
}

// JS: actors > should be able to restart a named spawned actor within a single macrostep when stopping by static name
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actor.test.ts#L1600
func TestActor_Actors_ShouldRestartNamedSpawnedActorWithinSingleMacrostepWhenStoppingByStaticName(t *testing.T) {
	type ctx struct{ ActorRef xs.ActorRef }
	actual := &actor2Recorder{}
	invokeCounter := 0

	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Initial: "active",
		ContextFn: func(a xs.ContextArgs) ctx {
			invokeCounter++
			localID := invokeCounter

			return ctx{
				ActorRef: a.Spawn(
					xs.FromCallback(func(_ xs.CallbackArgs) func() {
						actual.push("start " + strconv.Itoa(localID))
						return func() {
							actual.push("stop " + strconv.Itoa(localID))
						}
					}),
					xs.SpawnOptions{ID: "my_name"},
				),
			}
		},
		States: xs.States{
			{Key: "active", On: map[string]xs.Transitions{
				"update": {{Actions: xs.Actions{
					xs.StopChild("my_name"),
					xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
						invokeCounter++
						localID := invokeCounter

						c := a.Context
						c.ActorRef = a.Spawn(
							xs.FromCallback(func(_ xs.CallbackArgs) func() {
								actual.push("start " + strconv.Itoa(localID))
								return func() {
									actual.push("stop " + strconv.Itoa(localID))
								}
							}),
							xs.SpawnOptions{ID: "my_name"},
						)
						return c
					}),
				}}},
			}},
		},
	})

	service := xs.CreateActor(machine).Start()

	actual.reset()

	service.Send(xs.Ev("update"))

	assert.Equal(t, []string{"stop 1", "start 2"}, actual.get())
}

// JS: actors > should be able to restart a named spawned actor within a single macrostep when stopping by resolved name
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actor.test.ts#L1665
func TestActor_Actors_ShouldRestartNamedSpawnedActorWithinSingleMacrostepWhenStoppingByResolvedName(t *testing.T) {
	type ctx struct{ ActorRef xs.ActorRef }
	actual := &actor2Recorder{}
	invokeCounter := 0

	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Initial: "active",
		ContextFn: func(a xs.ContextArgs) ctx {
			invokeCounter++
			localID := invokeCounter
			actual.push("start " + strconv.Itoa(localID))

			return ctx{
				ActorRef: a.Spawn(
					xs.FromCallback(func(_ xs.CallbackArgs) func() {
						return func() {
							actual.push("stop " + strconv.Itoa(localID))
						}
					}),
					xs.SpawnOptions{ID: "my_name"},
				),
			}
		},
		States: xs.States{
			{Key: "active", On: map[string]xs.Transitions{
				"update": {{Actions: xs.Actions{
					xs.StopChild(xs.NewExpr(func(_ xs.ExprArgs[ctx]) any { return "my_name" })),
					xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
						invokeCounter++
						localID := invokeCounter

						c := a.Context
						c.ActorRef = a.Spawn(
							xs.FromCallback(func(_ xs.CallbackArgs) func() {
								actual.push("start " + strconv.Itoa(localID))
								return func() {
									actual.push("stop " + strconv.Itoa(localID))
								}
							}),
							xs.SpawnOptions{ID: "my_name"},
						)
						return c
					}),
				}}},
			}},
		},
	})

	service := xs.CreateActor(machine).Start()

	actual.reset()

	service.Send(xs.Ev("update"))

	assert.Equal(t, []string{"stop 1", "start 2"}, actual.get())
}

// JS: actors > should be possible to pass `self` as input to a child machine from within the context factory
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actor.test.ts#L1730
func TestActor_Actors_ShouldBePossibleToPassSelfAsInputToChildMachineFromContextFactory(t *testing.T) {
	type childCtx struct{ Parent xs.ActorRef }
	type childInput struct{ Parent xs.ActorRef }
	type parentCtx struct{ ChildRef xs.ActorRef }

	spy := newSpy()

	child := xs.CreateMachine(xs.MachineConfig[childCtx]{
		ContextFn: func(a xs.ContextArgs) childCtx {
			return childCtx{Parent: a.Input.(childInput).Parent}
		},
		Entry: xs.Actions{xs.SendTo(
			xs.NewExpr(func(a xs.ExprArgs[childCtx]) any { return a.Context.Parent }),
			xs.Ev("GREET"),
		)},
	})

	machine := xs.CreateMachine(xs.MachineConfig[parentCtx]{
		ContextFn: func(a xs.ContextArgs) parentCtx {
			return parentCtx{
				ChildRef: a.Spawn(child, xs.SpawnOptions{Input: childInput{Parent: a.Self}}),
			}
		},
		On: map[string]xs.Transitions{
			"GREET": {{Actions: xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[parentCtx]) {
				spy.Call(a)
			})}}},
		},
	})

	xs.CreateActor(machine).Start()

	assert.Equal(t, 1, spy.Count())
}

// JS: actors > catches errors from spawned promise actors
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actor.test.ts#L1766
func TestActor_Actors_CatchesErrorsFromSpawnedPromiseActors(t *testing.T) {
	// expect.assertions(1): the error observer must assert exactly once.
	var assertions atomic.Int32
	sig := newSignal()

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		On: map[string]xs.Transitions{
			"event": {{Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[any]) any {
				a.Spawn(xs.FromPromise(func(_ context.Context, _ xs.PromiseArgs) (any, error) {
					return nil, errors.New("uh oh")
				}))
				return a.Context
			})}}},
		},
	})

	actor := xs.CreateActor(machine)
	actor.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{
		Error: func(err any) {
			assertions.Add(1)
			e, _ := err.(error)
			if assert.NotNil(t, e, "error should be an error value, got %#v", err) {
				assert.Equal(t, "uh oh", e.Error())
			}
			sig.Resolve()
		},
	})
	actor.Start()
	actor.Send(xs.Ev("event"))

	sig.Wait(t)
	assert.Equal(t, int32(1), assertions.Load())
}

// JS: actors > same-position invokes should not leak between machines
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actor.test.ts#L1792
func TestActor_Actors_SamePositionInvokesShouldNotLeakBetweenMachines(t *testing.T) {
	spy := newSpy()

	sharedActors := map[string]xs.ActorLogic{}

	m1 := xs.CreateMachine(xs.MachineConfig[any]{
		Invoke: []xs.InvokeConfig{{
			Logic: xs.FromPromise(func(_ context.Context, _ xs.PromiseArgs) (string, error) {
				return "foo", nil
			}),
			OnDone: xs.Transitions{{Actions: xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[any]) {
				spy.Call(a.Event.(xs.DoneActorEvent).Output)
			})}}},
		}},
	}, xs.Implementations{Actors: sharedActors})

	xs.CreateMachine(xs.MachineConfig[any]{
		Invoke: []xs.InvokeConfig{{
			Logic: xs.FromPromise(func(_ context.Context, _ xs.PromiseArgs) (int, error) {
				return 100, nil
			}),
		}},
	}, xs.Implementations{Actors: sharedActors})

	xs.CreateActor(m1).Start()

	sleep(1)

	assert.Equal(t, 1, spy.Count())
	assert.Contains(t, spy.Calls(), []any{"foo"})
}

// JS: actors > inline invokes should not leak into provided actors object
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/actor.test.ts#L1824
func TestActor_Actors_InlineInvokesShouldNotLeakIntoProvidedActorsObject(t *testing.T) {
	actors := map[string]xs.ActorLogic{}

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Invoke: []xs.InvokeConfig{{
			Logic: xs.FromPromise(func(_ context.Context, _ xs.PromiseArgs) (string, error) {
				return "foo", nil
			}),
		}},
	}, xs.Implementations{Actors: actors})

	xs.CreateActor(machine).Start()

	assert.Equal(t, map[string]xs.ActorLogic{}, actors)
}
