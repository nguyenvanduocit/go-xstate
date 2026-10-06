package xstate_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// interpreter1LightMachine mirrors the top-level `lightMachine` of
// interpreter.test.ts (a func so CreateMachine is not called at package init).
func interpreter1LightMachine() *xs.StateMachine[any] {
	return xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "light",
		Initial: "green",
		States: xs.States{
			{
				Key:   "green",
				Entry: xs.Actions{xs.Raise(xs.Ev("TIMER"), xs.SendOptions{ID: "TIMER1", Delay: ms(10)})},
				On: map[string]xs.Transitions{
					"TIMER":      {{Target: "yellow"}},
					"KEEP_GOING": {{Actions: xs.Actions{xs.Cancel("TIMER1")}}},
				},
			},
			{
				Key:   "yellow",
				Entry: xs.Actions{xs.Raise(xs.Ev("TIMER"), xs.SendOptions{Delay: ms(10)})},
				On: map[string]xs.Transitions{
					"TIMER": {{Target: "red"}},
				},
			},
			{
				Key: "red",
				After: map[string]xs.Transitions{
					"10": {{Target: "green"}},
				},
			},
		},
	})
}

// interpreter1SendMachine mirrors `sendMachine` of the `.send()` describe.
func interpreter1SendMachine() *xs.StateMachine[any] {
	return xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "send",
		Initial: "inactive",
		States: xs.States{
			{
				Key: "inactive",
				On: map[string]xs.Transitions{
					"EVENT": {{
						Target: "active",
						Guard: xs.GuardFunc(func(a xs.GuardArgs[any]) bool {
							e, _ := a.Event.(xs.E)
							return e["id"] == 42
						}),
					}},
					"ACTIVATE": {{Target: "active"}},
				},
			},
			{Key: "active", Type: xs.Final},
		},
	})
}

// JS: interpreter > initial state > .getSnapshot returns the initial state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/interpreter.test.ts#L50
func TestInterpreter_InitialState_GetSnapshotReturnsTheInitialState(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "foo",
		States: xs.States{
			{Key: "bar"},
			{Key: "foo"},
		},
	})
	service := xs.CreateActor(machine)

	assert.Equal(t, "foo", service.GetSnapshot().Value)
}

// JS: interpreter > initial state > initially spawned actors should not be spawned when reading initial state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/interpreter.test.ts#L63
func TestInterpreter_InitialState_InitiallySpawnedActorsNotSpawnedWhenReadingInitialState(t *testing.T) {
	var promiseSpawned atomic.Int32

	type ctx struct{ Actor xs.ActorRef }

	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Initial: "idle",
		Context: ctx{Actor: nil},
		States: xs.States{
			{
				Key: "idle",
				Entry: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
					c := a.Context
					c.Actor = a.Spawn(xs.FromPromise(func(pctx context.Context, _ xs.PromiseArgs) (any, error) {
						// mirrors `new Promise(() => { promiseSpawned++ })`: never settles
						promiseSpawned.Add(1)
						<-pctx.Done()
						return nil, pctx.Err()
					}))
					return c
				})},
			},
		},
	})

	service := xs.CreateActor(machine)

	assert.Equal(t, int32(0), promiseSpawned.Load())

	service.GetSnapshot()
	service.GetSnapshot()
	service.GetSnapshot()

	assert.Equal(t, int32(0), promiseSpawned.Load())

	service.Start()

	sleep(100)
	assert.Equal(t, int32(1), promiseSpawned.Load())
}

// JS: interpreter > initial state > does not execute actions from a restored state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/interpreter.test.ts#L109
func TestInterpreter_InitialState_DoesNotExecuteActionsFromARestoredState(t *testing.T) {
	var called atomic.Bool
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "green",
		States: xs.States{
			{
				Key: "green",
				On: map[string]xs.Transitions{
					"TIMER": {{
						Target:  "yellow",
						Actions: xs.Actions{xs.ActionFunc(func(xs.ActionArgs[any]) { called.Store(true) })},
					}},
				},
			},
			{
				Key: "yellow",
				On: map[string]xs.Transitions{
					"TIMER": {{Target: "red"}},
				},
			},
			{
				Key: "red",
				On: map[string]xs.Transitions{
					"TIMER": {{Target: "green"}},
				},
			},
		},
	})

	actorRef := xs.CreateActor(machine).Start()

	actorRef.Send(xs.Ev("TIMER"))
	called.Store(false)
	persisted := actorRef.GetPersistedSnapshot()
	actorRef = xs.CreateActor(machine, xs.WithSnapshot(persisted)).Start()

	assert.False(t, called.Load())
}

// JS: interpreter > initial state > should not execute actions that are not part of the actual persisted state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/interpreter.test.ts#L147
func TestInterpreter_InitialState_ShouldNotExecuteActionsNotPartOfActualPersistedState(t *testing.T) {
	var called atomic.Bool
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{
				Key: "a",
				Entry: xs.Actions{xs.ActionFunc(func(xs.ActionArgs[any]) {
					// this should not be called when starting from a different state
					called.Store(true)
				})},
				Always: xs.Transitions{{Target: "b"}},
			},
			{Key: "b"},
		},
	})

	actorRef := xs.CreateActor(machine).Start()
	called.Store(false)
	assert.Equal(t, "b", actorRef.GetSnapshot().Value)
	persisted := actorRef.GetPersistedSnapshot()

	xs.CreateActor(machine, xs.WithSnapshot(persisted)).Start()

	assert.False(t, called.Load())
}

// JS: interpreter > subscribing > should not notify subscribers of the current state upon subscription (subscribe)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/interpreter.test.ts#L182
func TestInterpreter_Subscribing_ShouldNotNotifySubscribersOfCurrentStateUponSubscription(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "active",
		States: xs.States{
			{Key: "active"},
		},
	})

	spy := newSpy()
	service := xs.CreateActor(machine).Start()

	service.SubscribeNext(func(s *xs.MachineSnapshot[any]) { spy.Call(s) })

	assert.Equal(t, 0, spy.Count())
}

// JS: interpreter > send with delay > can send an event after a delay
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/interpreter.test.ts#L193
func TestInterpreter_SendWithDelay_CanSendAnEventAfterADelay(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "foo",
		States: xs.States{
			{
				Key:   "foo",
				Entry: xs.Actions{xs.Raise(xs.Ev("TIMER"), xs.SendOptions{Delay: ms(10)})},
				On: map[string]xs.Transitions{
					"TIMER": {{Target: "bar"}},
				},
			},
			{Key: "bar"},
		},
	})
	actorRef := xs.CreateActor(machine)
	assert.Equal(t, "foo", actorRef.GetSnapshot().Value)

	sleep(10)
	assert.Equal(t, "foo", actorRef.GetSnapshot().Value)

	actorRef.Start()
	assert.Equal(t, "foo", actorRef.GetSnapshot().Value)

	sleep(5)
	assert.Equal(t, "foo", actorRef.GetSnapshot().Value)

	sleep(10)
	assert.Equal(t, "bar", actorRef.GetSnapshot().Value)
}

// JS: interpreter > send with delay > can send an event after a delay (expression)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/interpreter.test.ts#L222
func TestInterpreter_SendWithDelay_CanSendAnEventAfterADelayExpression(t *testing.T) {
	type ctx struct{ InitialDelay int }

	delayExprMachine := xs.CreateMachine(xs.MachineConfig[ctx]{
		ID:      "delayExpr",
		Context: ctx{InitialDelay: 100},
		Initial: "idle",
		States: xs.States{
			{
				Key: "idle",
				On: map[string]xs.Transitions{
					"ACTIVATE": {{Target: "pending"}},
				},
			},
			{
				Key: "pending",
				Entry: xs.Actions{xs.Raise(xs.Ev("FINISH"), xs.SendOptions{
					Delay: xs.NewExpr(func(a xs.ExprArgs[ctx]) any {
						wait := 0
						if e, ok := a.Event.(xs.E); ok {
							if w, ok := e["wait"]; ok {
								wait = w.(int)
							}
						}
						return ms(a.Context.InitialDelay + wait)
					}),
				})},
				On: map[string]xs.Transitions{
					"FINISH": {{Target: "finished"}},
				},
			},
			{Key: "finished", Type: xs.Final},
		},
	})

	var stopped atomic.Bool

	clock := xs.NewSimulatedClock()

	delayExprService := xs.CreateActor(delayExprMachine, xs.WithClock(clock))
	delayExprService.Subscribe(xs.Observer[*xs.MachineSnapshot[ctx]]{
		Complete: func() { stopped.Store(true) },
	})
	delayExprService.Start()

	delayExprService.Send(xs.E{"type": "ACTIVATE", "wait": 50})

	clock.Increment(ms(101))

	assert.False(t, stopped.Load())

	clock.Increment(ms(50))

	assert.True(t, stopped.Load())
}

// JS: interpreter > send with delay > can send an event after a delay (expression using _event)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/interpreter.test.ts#L291
func TestInterpreter_SendWithDelay_CanSendAnEventAfterADelayExpressionUsingEvent(t *testing.T) {
	type ctx struct{ InitialDelay int }

	delayExprMachine := xs.CreateMachine(xs.MachineConfig[ctx]{
		ID:      "delayExpr",
		Context: ctx{InitialDelay: 100},
		Initial: "idle",
		States: xs.States{
			{
				Key: "idle",
				On: map[string]xs.Transitions{
					"ACTIVATE": {{Target: "pending"}},
				},
			},
			{
				Key: "pending",
				Entry: xs.Actions{xs.Raise(xs.Ev("FINISH"), xs.SendOptions{
					Delay: xs.NewExpr(func(a xs.ExprArgs[ctx]) any {
						xs.AssertEvent(a.Event, "ACTIVATE")
						return ms(a.Context.InitialDelay + a.Event.(xs.E)["wait"].(int))
					}),
				})},
				On: map[string]xs.Transitions{
					"FINISH": {{Target: "finished"}},
				},
			},
			{Key: "finished", Type: xs.Final},
		},
	})

	var stopped atomic.Bool

	clock := xs.NewSimulatedClock()

	delayExprService := xs.CreateActor(delayExprMachine, xs.WithClock(clock))
	delayExprService.Subscribe(xs.Observer[*xs.MachineSnapshot[ctx]]{
		Complete: func() { stopped.Store(true) },
	})
	delayExprService.Start()

	delayExprService.Send(xs.E{"type": "ACTIVATE", "wait": 50})

	clock.Increment(ms(101))

	assert.False(t, stopped.Load())

	clock.Increment(ms(50))

	assert.True(t, stopped.Load())
}

// JS: interpreter > send with delay > can send an event after a delay (delayed transitions)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/interpreter.test.ts#L369
func TestInterpreter_SendWithDelay_CanSendAnEventAfterADelayDelayedTransitions(t *testing.T) {
	type ctx struct{ Delay int }

	sig := newSignal()
	clock := xs.NewSimulatedClock()
	letterMachine := xs.CreateMachine(xs.MachineConfig[ctx]{
		ID:      "letter",
		Context: ctx{Delay: 100},
		Initial: "a",
		States: xs.States{
			{Key: "a", After: map[string]xs.Transitions{"delayA": {{Target: "b"}}}},
			{Key: "b", After: map[string]xs.Transitions{"someDelay": {{Target: "c"}}}},
			{
				Key:   "c",
				Entry: xs.Actions{xs.Raise(xs.E{"type": "FIRE_DELAY", "value": 200}, xs.SendOptions{Delay: ms(20)})},
				On: map[string]xs.Transitions{
					"FIRE_DELAY": {{Target: "d"}},
				},
			},
			{Key: "d", After: map[string]xs.Transitions{"delayD": {{Target: "e"}}}},
			{Key: "e", After: map[string]xs.Transitions{"someDelay": {{Target: "f"}}}},
			{Key: "f", Type: xs.Final},
		},
	}, xs.Implementations{
		Delays: map[string]any{
			"someDelay": xs.NewExpr(func(a xs.ExprArgs[ctx]) any {
				return ms(a.Context.Delay + 50)
			}),
			"delayA": xs.NewExpr(func(a xs.ExprArgs[ctx]) any { return ms(a.Context.Delay) }),
			"delayD": xs.NewExpr(func(a xs.ExprArgs[ctx]) any {
				return ms(a.Context.Delay + a.Event.(xs.E)["value"].(int))
			}),
		},
	})

	actor := xs.CreateActor(letterMachine, xs.WithClock(clock))
	actor.Subscribe(xs.Observer[*xs.MachineSnapshot[ctx]]{
		Complete: func() { sig.Resolve() },
	})
	actor.Start()

	assert.Equal(t, "a", actor.GetSnapshot().Value)
	clock.Increment(ms(100))
	assert.Equal(t, "b", actor.GetSnapshot().Value)
	clock.Increment(ms(100 + 50))
	assert.Equal(t, "c", actor.GetSnapshot().Value)
	clock.Increment(ms(20))
	assert.Equal(t, "d", actor.GetSnapshot().Value)
	clock.Increment(ms(100 + 200))
	assert.Equal(t, "e", actor.GetSnapshot().Value)
	clock.Increment(ms(100 + 50))

	sig.Wait(t)
}

// JS: interpreter > activities (deprecated) > should start activities
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/interpreter.test.ts#L447
func TestInterpreter_Activities_ShouldStartActivities(t *testing.T) {
	spy := newSpy()

	activityMachine := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "activity",
		Initial: "on",
		States: xs.States{
			{
				Key:    "on",
				Invoke: []xs.InvokeConfig{{Src: "myActivity"}},
				On: map[string]xs.Transitions{
					"TURN_OFF": {{Target: "off"}},
				},
			},
			{Key: "off"},
		},
	}, xs.Implementations{
		Actors: map[string]xs.ActorLogic{
			// mirrors fromCallback(spy): the spy is the callback itself
			"myActivity": xs.FromCallback(func(a xs.CallbackArgs) func() {
				spy.Call(a)
				return nil
			}),
		},
	})
	service := xs.CreateActor(activityMachine)

	service.Start()

	assert.Greater(t, spy.Count(), 0)
}

// JS: interpreter > activities (deprecated) > should stop activities
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/interpreter.test.ts#L479
func TestInterpreter_Activities_ShouldStopActivities(t *testing.T) {
	spy := newSpy()

	activityMachine := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "activity",
		Initial: "on",
		States: xs.States{
			{
				Key:    "on",
				Invoke: []xs.InvokeConfig{{Src: "myActivity"}},
				On: map[string]xs.Transitions{
					"TURN_OFF": {{Target: "off"}},
				},
			},
			{Key: "off"},
		},
	}, xs.Implementations{
		Actors: map[string]xs.ActorLogic{
			// mirrors fromCallback(() => spy): the spy is the cleanup
			"myActivity": xs.FromCallback(func(xs.CallbackArgs) func() {
				return func() { spy.Call() }
			}),
		},
	})
	service := xs.CreateActor(activityMachine)

	service.Start()

	assert.Equal(t, 0, spy.Count())

	service.Send(xs.Ev("TURN_OFF"))

	assert.Greater(t, spy.Count(), 0)
}

// JS: interpreter > activities (deprecated) > should stop activities upon stopping the service
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/interpreter.test.ts#L515
func TestInterpreter_Activities_ShouldStopActivitiesUponStoppingTheService(t *testing.T) {
	spy := newSpy()

	stopActivityMachine := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "stopActivity",
		Initial: "on",
		States: xs.States{
			{
				Key:    "on",
				Invoke: []xs.InvokeConfig{{Src: "myActivity"}},
				On: map[string]xs.Transitions{
					"TURN_OFF": {{Target: "off"}},
				},
			},
			{Key: "off"},
		},
	}, xs.Implementations{
		Actors: map[string]xs.ActorLogic{
			"myActivity": xs.FromCallback(func(xs.CallbackArgs) func() {
				return func() { spy.Call() }
			}),
		},
	})

	stopActivityService := xs.CreateActor(stopActivityMachine).Start()

	assert.Equal(t, 0, spy.Count())

	stopActivityService.Stop()

	assert.Greater(t, spy.Count(), 0)
}

// JS: interpreter > activities (deprecated) > should restart activities from a compound state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/interpreter.test.ts#L550
func TestInterpreter_Activities_ShouldRestartActivitiesFromACompoundState(t *testing.T) {
	var activityActive atomic.Bool

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "inactive",
		States: xs.States{
			{
				Key: "inactive",
				On:  map[string]xs.Transitions{"TOGGLE": {{Target: "active"}}},
			},
			{
				Key:     "active",
				Invoke:  []xs.InvokeConfig{{Src: "blink"}},
				On:      map[string]xs.Transitions{"TOGGLE": {{Target: "inactive"}}},
				Initial: "A",
				States: xs.States{
					{Key: "A", On: map[string]xs.Transitions{"SWITCH": {{Target: "B"}}}},
					{Key: "B", On: map[string]xs.Transitions{"SWITCH": {{Target: "A"}}}},
				},
			},
		},
	}, xs.Implementations{
		Actors: map[string]xs.ActorLogic{
			"blink": xs.FromCallback(func(xs.CallbackArgs) func() {
				activityActive.Store(true)
				return func() {
					activityActive.Store(false)
				}
			}),
		},
	})

	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.Ev("TOGGLE"))
	actorRef.Send(xs.Ev("SWITCH"))
	bState := actorRef.GetPersistedSnapshot()
	actorRef.Stop()
	activityActive.Store(false)

	xs.CreateActor(machine, xs.WithSnapshot(bState)).Start()

	assert.True(t, activityActive.Load())
}

// JS: interpreter > can cancel a delayed event
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/interpreter.test.ts#L596
func TestInterpreter_CanCancelADelayedEvent(t *testing.T) {
	service := xs.CreateActor(interpreter1LightMachine(), xs.WithClock(xs.NewSimulatedClock()))
	clock := service.Clock().(*xs.SimulatedClock)
	service.Start()

	clock.Increment(ms(5))
	service.Send(xs.Ev("KEEP_GOING"))

	assert.Equal(t, "green", service.GetSnapshot().Value)
	clock.Increment(ms(10))
	assert.Equal(t, "green", service.GetSnapshot().Value)
}

// JS: interpreter > can cancel a delayed event using expression to resolve send id
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/interpreter.test.ts#L611
func TestInterpreter_CanCancelADelayedEventUsingExpressionToResolveSendID(t *testing.T) {
	sig := newSignal()
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "first",
		States: xs.States{
			{
				Key: "first",
				Entry: xs.Actions{
					xs.Raise(xs.Ev("FOO"), xs.SendOptions{ID: "foo", Delay: ms(100)}),
					xs.Raise(xs.Ev("BAR"), xs.SendOptions{Delay: ms(200)}),
					xs.Cancel(xs.NewExpr(func(xs.ExprArgs[any]) any { return "foo" })),
				},
				On: map[string]xs.Transitions{
					"FOO": {{Target: "fail"}},
					"BAR": {{Target: "pass"}},
				},
			},
			{Key: "fail", Type: xs.Final},
			{Key: "pass", Type: xs.Final},
		},
	})

	service := xs.CreateActor(machine).Start()

	service.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{
		Complete: func() {
			assert.Equal(t, "pass", service.GetSnapshot().Value)
			sig.Resolve()
		},
	})
	sig.Wait(t)
}

// JS: interpreter > should not throw an error if an event is sent to an uninitialized interpreter
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/interpreter.test.ts#L658
func TestInterpreter_ShouldNotThrowIfEventSentToUninitializedInterpreter(t *testing.T) {
	actorRef := xs.CreateActor(interpreter1LightMachine())

	assert.NotPanics(t, func() { actorRef.Send(xs.Ev("SOME_EVENT")) })
}

// JS: interpreter > should defer events sent to an uninitialized service
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/interpreter.test.ts#L664
func TestInterpreter_ShouldDeferEventsSentToAnUninitializedService(t *testing.T) {
	sig := newSignal()
	deferMachine := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "defer",
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{"NEXT_A": {{Target: "b"}}}},
			{Key: "b", On: map[string]xs.Transitions{"NEXT_B": {{Target: "c"}}}},
			{Key: "c", Type: xs.Final},
		},
	})

	var mu sync.Mutex
	var state *xs.MachineSnapshot[any]
	deferService := xs.CreateActor(deferMachine)

	deferService.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{
		Next: func(nextState *xs.MachineSnapshot[any]) {
			mu.Lock()
			state = nextState
			mu.Unlock()
		},
		Complete: func() { sig.Resolve() },
	})

	// uninitialized
	deferService.Send(xs.Ev("NEXT_A"))
	deferService.Send(xs.Ev("NEXT_B"))

	mu.Lock()
	assert.Nil(t, state)
	mu.Unlock()

	// initialized
	deferService.Start()
	sig.Wait(t)
}

// JS: interpreter > should throw an error if initial state sent to interpreter is invalid
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/interpreter.test.ts#L703
func TestInterpreter_ShouldThrowAnErrorIfInitialStateSentToInterpreterIsInvalid(t *testing.T) {
	invalidMachine := xs.MachineConfig[any]{
		ID:      "fetchMachine",
		Initial: "create",
		States: xs.States{
			{
				Key:     "edit",
				Initial: "idle",
				States: xs.States{
					{Key: "idle", On: map[string]xs.Transitions{"FETCH": {{Target: "pending"}}}},
					{Key: "pending"},
				},
			},
		},
	}

	snapshot := xs.CreateActor(xs.CreateMachine(invalidMachine)).GetSnapshot()

	assert.Equal(t, xs.StatusError, snapshot.Status)
	err, ok := snapshot.Error.(error)
	require.True(t, ok, "snapshot.Error should be an error, got %#v", snapshot.Error)
	assert.Equal(t, `Initial state node "create" not found on parent state node #fetchMachine`, err.Error())
}

// JS: interpreter > should not update when stopped
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/interpreter.test.ts#L730
func TestInterpreter_ShouldNotUpdateWhenStopped(t *testing.T) {
	warnSpy := newSpy()
	service := xs.CreateActor(interpreter1LightMachine(),
		xs.WithClock(xs.NewSimulatedClock()),
		xs.WithWarnHandler(func(args ...any) { warnSpy.Call(args...) }),
	)

	service.Start()
	service.Send(xs.Ev("TIMER")) // yellow
	assert.Equal(t, "yellow", service.GetSnapshot().Value)

	service.Stop()
	func() {
		defer func() {
			if r := recover(); r != nil {
				assert.Equal(t, "yellow", service.GetSnapshot().Value)
			}
		}()
		service.Send(xs.Ev("TIMER")) // red if interpreter is not stopped
	}()

	// The JS inline snapshot hardcodes `"x:27 (x:27)"`: `${actor.id} (${actor.sessionId})`
	// where the default id is the session id.
	assert.Equal(t, [][]any{{fmt.Sprintf(
		"Event \"TIMER\" was sent to stopped actor \"%s (%s)\". This actor has already reached its final state, and will not transition.\nEvent: {\"type\":\"TIMER\"}",
		service.SessionID(), service.SessionID(),
	)}}, warnSpy.Calls())
}

// JS: interpreter > should be able to log (log action)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/interpreter.test.ts#L757
func TestInterpreter_ShouldBeAbleToLogLogAction(t *testing.T) {
	type ctx struct{ Count int }
	var logs []any

	logMachine := xs.CreateMachine(xs.MachineConfig[ctx]{
		ID:      "log",
		Initial: "x",
		Context: ctx{Count: 0},
		States: xs.States{
			{
				Key: "x",
				On: map[string]xs.Transitions{
					"LOG": {{Actions: xs.Actions{
						xs.Assign(func(a xs.AssignArgs[ctx]) ctx { return ctx{Count: a.Context.Count + 1} }),
						xs.Log(xs.NewExpr(func(a xs.ExprArgs[ctx]) any { return a.Context })),
					}}},
				},
			},
		},
	})

	service := xs.CreateActor(logMachine, xs.WithLogger(func(args ...any) {
		var msg any
		if len(args) > 0 {
			msg = args[0]
		}
		logs = append(logs, msg)
	})).Start()

	service.Send(xs.Ev("LOG"))
	service.Send(xs.Ev("LOG"))

	assert.Len(t, logs, 2)
	assert.Equal(t, []any{ctx{Count: 1}, ctx{Count: 2}}, logs)
}

// JS: interpreter > should receive correct event (log action)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/interpreter.test.ts#L790
func TestInterpreter_ShouldReceiveCorrectEventLogAction(t *testing.T) {
	var logs []any
	logAction := xs.Log(xs.NewExpr(func(a xs.ExprArgs[any]) any { return a.Event.EventType() }))

	parentMachine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "foo",
		States: xs.States{
			{
				Key: "foo",
				On: map[string]xs.Transitions{
					"EXTERNAL_EVENT": {{Actions: xs.Actions{xs.Raise(xs.Ev("RAISED_EVENT")), logAction}}},
				},
			},
		},
		On: map[string]xs.Transitions{
			"*": {{Actions: xs.Actions{logAction}}},
		},
	})

	service := xs.CreateActor(parentMachine, xs.WithLogger(func(args ...any) {
		var msg any
		if len(args) > 0 {
			msg = args[0]
		}
		logs = append(logs, msg)
	})).Start()

	service.Send(xs.Ev("EXTERNAL_EVENT"))

	assert.Len(t, logs, 2)
	assert.Equal(t, []any{"EXTERNAL_EVENT", "RAISED_EVENT"}, logs)
}

// JS: interpreter > send() event expressions > should resolve send event expressions
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/interpreter.test.ts#L856
func TestInterpreter_SendEventExpressions_ShouldResolveSendEventExpressions(t *testing.T) {
	type ctx struct{ Password string }

	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		ID:      "sendexpr",
		Initial: "start",
		Context: ctx{Password: "foo"},
		States: xs.States{
			{
				Key: "start",
				Entry: xs.Actions{xs.Raise(xs.NewExpr(func(a xs.ExprArgs[ctx]) any {
					return xs.E{"type": "NEXT", "password": a.Context.Password}
				}))},
				On: map[string]xs.Transitions{
					"NEXT": {{
						Target: "finish",
						Guard: xs.GuardFunc(func(a xs.GuardArgs[ctx]) bool {
							return a.Event.(xs.E)["password"] == "foo"
						}),
					}},
				},
			},
			{Key: "finish", Type: xs.Final},
		},
	})

	sig := newSignal()
	actor := xs.CreateActor(machine)
	actor.Subscribe(xs.Observer[*xs.MachineSnapshot[ctx]]{Complete: func() { sig.Resolve() }})
	actor.Start()
	sig.Wait(t)
}

// JS: interpreter > sendParent() event expressions > should resolve sendParent event expressions
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/interpreter.test.ts#L866
func TestInterpreter_SendParentEventExpressions_ShouldResolveSendParentEventExpressions(t *testing.T) {
	type childCtx struct{ Password string }
	type childInput struct{ Password string }

	sig := newSignal()
	childMachine := xs.CreateMachine(xs.MachineConfig[childCtx]{
		ID:      "child",
		Initial: "start",
		ContextFn: func(a xs.ContextArgs) childCtx {
			return childCtx{Password: a.Input.(childInput).Password}
		},
		States: xs.States{
			{
				Key: "start",
				Entry: xs.Actions{xs.SendParent(xs.NewExpr(func(a xs.ExprArgs[childCtx]) any {
					return xs.E{"type": "NEXT", "password": a.Context.Password}
				}))},
			},
		},
	})

	parentMachine := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "parent",
		Initial: "start",
		States: xs.States{
			{
				Key: "start",
				Invoke: []xs.InvokeConfig{{
					ID:    "child",
					Logic: childMachine,
					Input: childInput{Password: "foo"},
				}},
				On: map[string]xs.Transitions{
					"NEXT": {{
						Target: "finish",
						Guard: xs.GuardFunc(func(a xs.GuardArgs[any]) bool {
							return a.Event.(xs.E)["password"] == "foo"
						}),
					}},
				},
			},
			{Key: "finish", Type: xs.Final},
		},
	})

	actor := xs.CreateActor(parentMachine)
	actor.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{
		Next: func(state *xs.MachineSnapshot[any]) {
			if state.Matches("start") {
				childActor := state.Children["child"]

				// JS: expect(typeof childActor!.send).toBe('function') — the child
				// must exist (a missing child throws a TypeError in JS).
				assert.NotNil(t, childActor)
			}
		},
		Complete: func() { sig.Resolve() },
	})
	actor.Start()
	sig.Wait(t)
}

// JS: interpreter > .send() > can send events with a string
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/interpreter.test.ts#L952
func TestInterpreter_Send_CanSendEventsWithAString(t *testing.T) {
	sig := newSignal()
	service := xs.CreateActor(interpreter1SendMachine())
	service.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{Complete: func() { sig.Resolve() }})
	service.Start()

	service.Send(xs.Ev("ACTIVATE"))
	sig.Wait(t)
}

// JS: interpreter > .send() > can send events with an object
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/interpreter.test.ts#L962
func TestInterpreter_Send_CanSendEventsWithAnObject(t *testing.T) {
	sig := newSignal()
	service := xs.CreateActor(interpreter1SendMachine())
	service.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{Complete: func() { sig.Resolve() }})
	service.Start()

	service.Send(xs.Ev("ACTIVATE"))
	sig.Wait(t)
}

// JS: interpreter > .send() > can send events with an object with payload
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/interpreter.test.ts#L972
func TestInterpreter_Send_CanSendEventsWithAnObjectWithPayload(t *testing.T) {
	sig := newSignal()
	service := xs.CreateActor(interpreter1SendMachine())
	service.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{Complete: func() { sig.Resolve() }})
	service.Start()

	service.Send(xs.E{"type": "EVENT", "id": 42})
	sig.Wait(t)
}

// JS: interpreter > .send() > should receive and process all events sent simultaneously
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/interpreter.test.ts#L982
func TestInterpreter_Send_ShouldReceiveAndProcessAllEventsSentSimultaneously(t *testing.T) {
	sig := newSignal()
	toggleMachine := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "toggle",
		Initial: "inactive",
		States: xs.States{
			{Key: "fail"},
			{
				Key: "inactive",
				On: map[string]xs.Transitions{
					"INACTIVATE": {{Target: "fail"}},
					"ACTIVATE":   {{Target: "active"}},
				},
			},
			{
				Key: "active",
				On: map[string]xs.Transitions{
					"INACTIVATE": {{Target: "success"}},
				},
			},
			{Key: "success", Type: xs.Final},
		},
	})

	toggleService := xs.CreateActor(toggleMachine)
	toggleService.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{
		Complete: func() { sig.Resolve() },
	})
	toggleService.Start()

	toggleService.Send(xs.Ev("ACTIVATE"))
	toggleService.Send(xs.Ev("INACTIVATE"))
	sig.Wait(t)
}

// JS: interpreter > .start() > should initialize the service
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/interpreter.test.ts#L1021
func TestInterpreter_Start_ShouldInitializeTheService(t *testing.T) {
	contextSpy := newSpy()
	entrySpy := newSpy()

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		// JS `context: contextSpy`: a context factory returning undefined
		ContextFn: func(a xs.ContextArgs) any {
			contextSpy.Call(a)
			return nil
		},
		Entry:   xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[any]) { entrySpy.Call(a) })},
		Initial: "foo",
		States: xs.States{
			{Key: "foo"},
		},
	})
	actor := xs.CreateActor(machine)
	actor.Start()

	assert.Greater(t, contextSpy.Count(), 0)
	assert.Greater(t, entrySpy.Count(), 0)
	assert.NotNil(t, actor.GetSnapshot())
	assert.True(t, actor.GetSnapshot().Matches("foo"))
}

// JS: interpreter > .start() > should not reinitialize a started service
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/interpreter.test.ts#L1042
func TestInterpreter_Start_ShouldNotReinitializeAStartedService(t *testing.T) {
	contextSpy := newSpy()
	entrySpy := newSpy()

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		ContextFn: func(a xs.ContextArgs) any {
			contextSpy.Call(a)
			return nil
		},
		Entry: xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[any]) { entrySpy.Call(a) })},
	})
	actor := xs.CreateActor(machine)
	actor.Start()
	actor.Start()

	assert.Equal(t, 1, contextSpy.Count())
	assert.Equal(t, 1, entrySpy.Count())
}

// JS: interpreter > .start() > should be able to be initialized at a custom state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/interpreter.test.ts#L1058
func TestInterpreter_Start_ShouldBeAbleToBeInitializedAtACustomState(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "foo",
		States: xs.States{
			{Key: "foo"},
			{Key: "bar"},
		},
	})
	actor := xs.CreateActor(machine, xs.WithSnapshot(
		machine.ResolveState(xs.ResolveStateConfig[any]{Value: "bar"}),
	))

	assert.True(t, actor.GetSnapshot().Matches("bar"))
	actor.Start()
	assert.True(t, actor.GetSnapshot().Matches("bar"))
}

// JS: interpreter > .start() > should be able to be initialized at a custom state value
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/interpreter.test.ts#L1075
func TestInterpreter_Start_ShouldBeAbleToBeInitializedAtACustomStateValue(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "foo",
		States: xs.States{
			{Key: "foo"},
			{Key: "bar"},
		},
	})
	actor := xs.CreateActor(machine, xs.WithSnapshot(
		machine.ResolveState(xs.ResolveStateConfig[any]{Value: "bar"}),
	))

	assert.True(t, actor.GetSnapshot().Matches("bar"))
	actor.Start()
	assert.True(t, actor.GetSnapshot().Matches("bar"))
}

// JS: interpreter > .start() > should be able to resolve a custom initialized state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/interpreter.test.ts#L1092
func TestInterpreter_Start_ShouldBeAbleToResolveACustomInitializedState(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "start",
		Initial: "foo",
		States: xs.States{
			{
				Key:     "foo",
				Initial: "one",
				States: xs.States{
					{Key: "one"},
				},
			},
			{Key: "bar"},
		},
	})
	actor := xs.CreateActor(machine, xs.WithSnapshot(
		machine.ResolveState(xs.ResolveStateConfig[any]{Value: "foo"}),
	))

	assert.True(t, actor.GetSnapshot().Matches(map[string]any{"foo": "one"}))
	actor.Start()
	assert.True(t, actor.GetSnapshot().Matches(map[string]any{"foo": "one"}))
}

// JS: interpreter > .stop() > should cancel delayed events
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/interpreter.test.ts#L1117
func TestInterpreter_Stop_ShouldCancelDelayedEvents(t *testing.T) {
	var called atomic.Bool
	delayedMachine := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "delayed",
		Initial: "foo",
		States: xs.States{
			{
				Key: "foo",
				After: map[string]xs.Transitions{
					"50": {{
						Target:  "bar",
						Actions: xs.Actions{xs.ActionFunc(func(xs.ActionArgs[any]) { called.Store(true) })},
					}},
				},
			},
			{Key: "bar"},
		},
	})

	delayedService := xs.CreateActor(delayedMachine).Start()

	delayedService.Stop()

	sleep(60)
	assert.False(t, called.Load())
}

// JS: interpreter > .stop() > should not execute transitions after being stopped
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/interpreter.test.ts#L1149
func TestInterpreter_Stop_ShouldNotExecuteTransitionsAfterBeingStopped(t *testing.T) {
	warnSpy := newSpy()
	var called atomic.Bool

	testMachine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "waiting",
		States: xs.States{
			{
				Key: "waiting",
				On: map[string]xs.Transitions{
					"TRIGGER": {{Target: "active"}},
				},
			},
			{
				Key:   "active",
				Entry: xs.Actions{xs.ActionFunc(func(xs.ActionArgs[any]) { called.Store(true) })},
			},
		},
	})

	service := xs.CreateActor(testMachine,
		xs.WithWarnHandler(func(args ...any) { warnSpy.Call(args...) }),
	).Start()

	service.Stop()

	service.Send(xs.Ev("TRIGGER"))

	sleep(10)
	assert.False(t, called.Load())
	// The JS inline snapshot hardcodes `"x:43 (x:43)"`: `${actor.id} (${actor.sessionId})`
	// where the default id is the session id.
	assert.Equal(t, [][]any{{fmt.Sprintf(
		"Event \"TRIGGER\" was sent to stopped actor \"%s (%s)\". This actor has already reached its final state, and will not transition.\nEvent: {\"type\":\"TRIGGER\"}",
		service.SessionID(), service.SessionID(),
	)}}, warnSpy.Calls())
}

// JS: interpreter > .stop() > should not throw when sending an unserializable event to a stopped actor
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/interpreter.test.ts#L1191
func TestInterpreter_Stop_ShouldNotThrowWhenSendingUnserializableEventToStoppedActor(t *testing.T) {
	warnSpy := newSpy()

	testMachine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "waiting",
		States: xs.States{
			{
				Key: "waiting",
				On: map[string]xs.Transitions{
					"TRIGGER": {{Target: "active"}},
				},
			},
			{Key: "active"},
		},
	})

	service := xs.CreateActor(testMachine,
		xs.WithWarnHandler(func(args ...any) { warnSpy.Call(args...) }),
	).Start()

	service.Stop()

	// event with a circular reference cannot be JSON-serialized
	circular := xs.E{"type": "TRIGGER"}
	circular["self"] = circular

	assert.NotPanics(t, func() {
		service.Send(circular)
	})

	assert.Equal(t, 1, warnSpy.Count())
}

// JS: interpreter > .stop() > stopping a not-started interpreter should not crash
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/interpreter.test.ts#L1223
func TestInterpreter_Stop_StoppingANotStartedInterpreterShouldNotCrash(t *testing.T) {
	service := xs.CreateActor(xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States:  xs.States{{Key: "a"}},
	}))

	assert.NotPanics(t, func() {
		service.Stop()
	})
}

// JS: interpreter > .unsubscribe() > should remove transition listeners
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/interpreter.test.ts#L1238
func TestInterpreter_Unsubscribe_ShouldRemoveTransitionListeners(t *testing.T) {
	toggleMachine := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "toggle",
		Initial: "inactive",
		States: xs.States{
			{Key: "inactive", On: map[string]xs.Transitions{"TOGGLE": {{Target: "active"}}}},
			{Key: "active", On: map[string]xs.Transitions{"TOGGLE": {{Target: "inactive"}}}},
		},
	})

	toggleService := xs.CreateActor(toggleMachine).Start()

	stateCount := 0

	listener := func(*xs.MachineSnapshot[any]) { stateCount++ }

	sub := toggleService.SubscribeNext(listener)

	assert.Equal(t, 0, stateCount)

	toggleService.Send(xs.Ev("TOGGLE"))

	assert.Equal(t, 1, stateCount)

	toggleService.Send(xs.Ev("TOGGLE"))

	assert.Equal(t, 2, stateCount)

	sub.Unsubscribe()
	toggleService.Send(xs.Ev("TOGGLE"))

	assert.Equal(t, 2, stateCount)
}

// interpreter2IntervalCtx mirrors `const context = { count: 0 }` of the
// describe('observable') block (JS L1342).
type interpreter2IntervalCtx struct{ Count int }

// interpreter2IntervalMachine mirrors the describe-level `intervalMachine`
// of describe('observable') (JS L1343-1369).
func interpreter2IntervalMachine() *xs.StateMachine[interpreter2IntervalCtx] {
	return xs.CreateMachine(xs.MachineConfig[interpreter2IntervalCtx]{
		ID:      "interval",
		Context: interpreter2IntervalCtx{Count: 0},
		Initial: "active",
		States: xs.States{
			{
				Key: "active",
				After: map[string]xs.Transitions{
					"10": {{
						Target:  "active",
						Reenter: true,
						Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[interpreter2IntervalCtx]) interpreter2IntervalCtx {
							c := a.Context
							c.Count = a.Context.Count + 1
							return c
						})},
					}},
				},
				Always: xs.Transitions{{
					Target: "finished",
					Guard: xs.GuardFunc(func(a xs.GuardArgs[interpreter2IntervalCtx]) bool {
						return a.Context.Count >= 5
					}),
				}},
			},
			{Key: "finished", Type: xs.Final},
		},
	})
}

// JS: interpreter > transient states > should transition in correct order
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/interpreter.test.ts#L1278
func TestInterpreter_TransientStates_ShouldTransitionInCorrectOrder(t *testing.T) {
	stateMachine := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "transient",
		Initial: "idle",
		States: xs.States{
			{Key: "idle", On: map[string]xs.Transitions{"START": {{Target: "transient"}}}},
			{Key: "transient", Always: xs.Transitions{{Target: "next"}}},
			{Key: "next", On: map[string]xs.Transitions{"FINISH": {{Target: "end"}}}},
			{Key: "end", Type: xs.Final},
		},
	})

	stateValues := []xs.StateValue{}
	service := xs.CreateActor(stateMachine)
	service.SubscribeNext(func(current *xs.MachineSnapshot[any]) {
		stateValues = append(stateValues, current.Value)
	})
	service.Start()
	service.Send(xs.Ev("START"))

	expectedStateValues := []xs.StateValue{"idle", "next"}
	assert.Equal(t, len(expectedStateValues), len(stateValues))
	for i := 0; i < len(expectedStateValues); i++ {
		if i >= len(stateValues) {
			assert.Failf(t, "missing state value", "index %d", i)
			continue
		}
		assert.Equal(t, expectedStateValues[i], stateValues[i])
	}
}

// JS: interpreter > transient states > should transition in correct order when there is a condition
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/interpreter.test.ts#L1303
func TestInterpreter_TransientStates_ShouldTransitionInCorrectOrderWhenThereIsACondition(t *testing.T) {
	stateMachine := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "transient",
		Initial: "idle",
		States: xs.States{
			{Key: "idle", On: map[string]xs.Transitions{"START": {{Target: "transient"}}}},
			{Key: "transient", Always: xs.Transitions{
				{Target: "end", Guard: xs.GuardRef{Type: "alwaysFalse"}},
				{Target: "next"},
			}},
			{Key: "next", On: map[string]xs.Transitions{"FINISH": {{Target: "end"}}}},
			{Key: "end", Type: xs.Final},
		},
	}, xs.Implementations{
		Guards: map[string]xs.Guard{
			"alwaysFalse": xs.GuardFunc(func(_ xs.GuardArgs[any]) bool { return false }),
		},
	})

	stateValues := []xs.StateValue{}
	service := xs.CreateActor(stateMachine)
	service.SubscribeNext(func(current *xs.MachineSnapshot[any]) {
		stateValues = append(stateValues, current.Value)
	})
	service.Start()
	service.Send(xs.Ev("START"))

	expectedStateValues := []xs.StateValue{"idle", "next"}
	assert.Equal(t, len(expectedStateValues), len(stateValues))
	for i := 0; i < len(expectedStateValues); i++ {
		if i >= len(stateValues) {
			assert.Failf(t, "missing state value", "index %d", i)
			continue
		}
		assert.Equal(t, expectedStateValues[i], stateValues[i])
	}
}

// JS: interpreter > observable > should be subscribable
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/interpreter.test.ts#L1370
func TestInterpreter_Observable_ShouldBeSubscribable(t *testing.T) {
	sig := newSignal()
	var mu sync.Mutex
	var count int
	intervalService := xs.CreateActor(interpreter2IntervalMachine()).Start()

	// typeof intervalService.subscribe === 'function': the actor satisfies
	// the Subscribable interface (checked at compile time).
	var _ xs.Subscribable[*xs.MachineSnapshot[interpreter2IntervalCtx]] = intervalService

	intervalService.Subscribe(xs.Observer[*xs.MachineSnapshot[interpreter2IntervalCtx]]{
		Next: func(state *xs.MachineSnapshot[interpreter2IntervalCtx]) {
			mu.Lock()
			count = state.Context.Count
			mu.Unlock()
		},
		Complete: func() {
			mu.Lock()
			c := count
			mu.Unlock()
			assert.Equal(t, 5, c)
			sig.Resolve()
		},
	})
	sig.Wait(t)
}

// JS: interpreter > observable > should be interoperable with RxJS, etc. via Symbol.observable
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/interpreter.test.ts#L1390
func TestInterpreter_Observable_ShouldBeInteroperableWithRxJSViaSymbolObservable(t *testing.T) {
	sig := newSignal()
	var mu sync.Mutex
	count := 0
	intervalService := xs.CreateActor(interpreter2IntervalMachine()).Start()

	// from(intervalService): the actor is consumed through the generic
	// Subscribable interface, as an observable library would.
	var state xs.Subscribable[*xs.MachineSnapshot[interpreter2IntervalCtx]] = intervalService

	state.Subscribe(xs.Observer[*xs.MachineSnapshot[interpreter2IntervalCtx]]{
		Next: func(_ *xs.MachineSnapshot[interpreter2IntervalCtx]) {
			mu.Lock()
			count += 1
			mu.Unlock()
		},
		Error: nil,
		Complete: func() {
			mu.Lock()
			c := count
			mu.Unlock()
			assert.Equal(t, 5, c)
			sig.Resolve()
		},
	})
	sig.Wait(t)
}

// JS: interpreter > observable > should be unsubscribable
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/interpreter.test.ts#L1410
func TestInterpreter_Observable_ShouldBeUnsubscribable(t *testing.T) {
	type ctx struct{ Count int }
	sig := newSignal()
	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Context: ctx{Count: 0},
		Initial: "active",
		States: xs.States{
			{
				Key: "active",
				Always: xs.Transitions{{
					Target: "finished",
					Guard:  xs.GuardFunc(func(a xs.GuardArgs[ctx]) bool { return a.Context.Count >= 5 }),
				}},
				On: map[string]xs.Transitions{
					"INC": {{Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
						c := a.Context
						c.Count = a.Context.Count + 1
						return c
					})}}},
				},
			},
			{Key: "finished", Type: xs.Final},
		},
	})

	var mu sync.Mutex
	var count int
	service := xs.CreateActor(machine)
	service.Subscribe(xs.Observer[*xs.MachineSnapshot[ctx]]{
		Complete: func() {
			mu.Lock()
			c := count
			mu.Unlock()
			assert.Equal(t, 2, c)
			sig.Resolve()
		},
	})
	service.Start()

	subscription := service.SubscribeNext(func(state *xs.MachineSnapshot[ctx]) {
		mu.Lock()
		count = state.Context.Count
		mu.Unlock()
	})

	service.Send(xs.Ev("INC"))
	service.Send(xs.Ev("INC"))
	subscription.Unsubscribe()
	service.Send(xs.Ev("INC"))
	service.Send(xs.Ev("INC"))
	service.Send(xs.Ev("INC"))
	sig.Wait(t)
}

// JS: interpreter > observable > should call complete() once a final state is reached
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/interpreter.test.ts#L1458
func TestInterpreter_Observable_ShouldCallCompleteOnceAFinalStateIsReached(t *testing.T) {
	completeCb := newSpy()

	service := xs.CreateActor(xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "idle",
		States: xs.States{
			{Key: "idle", On: map[string]xs.Transitions{"NEXT": {{Target: "done"}}}},
			{Key: "done", Type: xs.Final},
		},
	})).Start()

	service.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{
		Complete: func() { completeCb.Call() },
	})

	service.Send(xs.Ev("NEXT"))

	assert.Equal(t, 1, completeCb.Count())
}

// JS: interpreter > observable > should call complete() once the interpreter is stopped
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/interpreter.test.ts#L1484
func TestInterpreter_Observable_ShouldCallCompleteOnceTheInterpreterIsStopped(t *testing.T) {
	completeCb := newSpy()

	service := xs.CreateActor(xs.CreateMachine(xs.MachineConfig[any]{})).Start()

	service.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{
		Complete: func() {
			completeCb.Call()
		},
	})

	service.Stop()

	assert.Equal(t, 1, completeCb.Count())
}

// JS: interpreter > actors > doesn't crash cryptically on undefined return from the actor creator
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/interpreter.test.ts#L1502
func TestInterpreter_Actors_DoesntCrashCrypticallyOnUndefinedReturnFromTheActorCreator(t *testing.T) {
	child := xs.FromCallback(func(_ xs.CallbackArgs) func() {
		// nothing
		return nil
	})
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "initial",
		States: xs.States{
			{Key: "initial", Invoke: []xs.InvokeConfig{{Src: "testService"}}},
		},
	}, xs.Implementations{
		Actors: map[string]xs.ActorLogic{"testService": child},
	})

	service := xs.CreateActor(machine)
	assert.NotPanics(t, func() { service.Start() })
}

// JS: interpreter > children > state.children should reference invoked child actors (machine)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/interpreter.test.ts#L1536
func TestInterpreter_Children_StateChildrenShouldReferenceInvokedChildActorsMachine(t *testing.T) {
	childMachine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "active",
		States: xs.States{
			{Key: "active", On: map[string]xs.Transitions{
				"FIRE": {{Actions: xs.Actions{xs.SendParent(xs.Ev("FIRED"))}}},
			}},
		},
	})
	parentMachine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "active",
		States: xs.States{
			{
				Key:    "active",
				Invoke: []xs.InvokeConfig{{ID: "childActor", Logic: childMachine}},
				On:     map[string]xs.Transitions{"FIRED": {{Target: "success"}}},
			},
			{Key: "success", Type: xs.Final},
		},
	})

	actor := xs.CreateActor(parentMachine)
	actor.Start()
	childActor := actor.GetSnapshot().Children["childActor"]
	require.NotNil(t, childActor)
	childActor.Send(xs.Ev("FIRE"))

	// the actor should be done by now
	assert.NotContains(t, actor.GetSnapshot().Children, "childActor")
}

// JS: interpreter > children > state.children should reference invoked child actors (promise)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/interpreter.test.ts#L1575
func TestInterpreter_Children_StateChildrenShouldReferenceInvokedChildActorsPromise(t *testing.T) {
	sig := newSignal()
	parentMachine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "active",
		States: xs.States{
			{
				Key: "active",
				Invoke: []xs.InvokeConfig{{
					ID:  "childActor",
					Src: "num",
					OnDone: xs.Transitions{
						{
							Target: "success",
							Guard: xs.GuardFunc(func(a xs.GuardArgs[any]) bool {
								ev, ok := a.Event.(xs.DoneActorEvent)
								return ok && ev.Output == 42
							}),
						},
						{Target: "failure"},
					},
				}},
			},
			{Key: "success", Type: xs.Final},
			{Key: "failure", Type: xs.Final},
		},
	}, xs.Implementations{
		Actors: map[string]xs.ActorLogic{
			"num": xs.FromPromise(func(_ context.Context, _ xs.PromiseArgs) (int, error) {
				time.Sleep(ms(100))
				return 42, nil
			}),
		},
	})

	service := xs.CreateActor(parentMachine)

	service.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{
		Next: func(state *xs.MachineSnapshot[any]) {
			if state.Matches("active") {
				childActor := state.Children["childActor"]

				// toHaveProperty('send'): every ActorRef has Send, so the ref must exist.
				assert.NotNil(t, childActor)
			}
		},
		Complete: func() {
			assert.True(t, service.GetSnapshot().Matches("success"))
			assert.NotContains(t, service.GetSnapshot().Children, "childActor")
			sig.Resolve()
		},
	})

	service.Start()
	sig.Wait(t)
}

// JS: interpreter > children > state.children should reference invoked child actors (observable)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/interpreter.test.ts#L1647
func TestInterpreter_Children_StateChildrenShouldReferenceInvokedChildActorsObservable(t *testing.T) {
	sig := newSignal()
	interval := rxInterval(10)
	intervalLogic := xs.FromObservable(func(_ xs.ObservableArgs) xs.Subscribable[int] { return interval })

	parentMachine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "active",
		States: xs.States{
			{
				Key: "active",
				Invoke: []xs.InvokeConfig{{
					ID:  "childActor",
					Src: "intervalLogic",
					OnSnapshot: xs.Transitions{{
						Target: "success",
						Guard: xs.GuardFunc(func(a xs.GuardArgs[any]) bool {
							ev, ok := a.Event.(xs.SnapshotEvent)
							if !ok {
								return false
							}
							snap, ok := ev.Snapshot.(*xs.ObservableSnapshot[int])
							return ok && snap.Context == 3
						}),
					}},
				}},
			},
			{Key: "success", Type: xs.Final},
		},
	}, xs.Implementations{
		Actors: map[string]xs.ActorLogic{"intervalLogic": intervalLogic},
	})

	service := xs.CreateActor(parentMachine)
	service.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{
		Complete: func() {
			assert.NotContains(t, service.GetSnapshot().Children, "childActor")
			sig.Resolve()
		},
	})

	service.SubscribeNext(func(state *xs.MachineSnapshot[any]) {
		if state.Matches("active") {
			assert.NotNil(t, state.Children["childActor"])
		}
	})

	service.Start()
	sig.Wait(t)
}

// JS: interpreter > children > state.children should reference spawned actors
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/interpreter.test.ts#L1706
func TestInterpreter_Children_StateChildrenShouldReferenceSpawnedActors(t *testing.T) {
	type ctx struct{ FirstNameRef xs.ActorRef }
	childMachine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "idle",
		States:  xs.States{{Key: "idle"}},
	})
	formMachine := xs.CreateMachine(xs.MachineConfig[ctx]{
		ID:      "form",
		Initial: "idle",
		Context: ctx{},
		Entry: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
			c := a.Context
			c.FirstNameRef = a.Spawn(childMachine, xs.SpawnOptions{ID: "child"})
			return c
		})},
		States: xs.States{{Key: "idle"}},
	})

	actor := xs.CreateActor(formMachine)
	actor.Start()
	assert.Contains(t, actor.GetSnapshot().Children, "child")
}

// JS: interpreter > children > stopped spawned actors should be cleaned up in parent
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/interpreter.test.ts#L1730
func TestInterpreter_Children_StoppedSpawnedActorsShouldBeCleanedUpInParent(t *testing.T) {
	type ctx struct {
		MachineRef    xs.ActorRef
		PromiseRef    xs.ActorRef
		ObservableRef xs.ActorRef
	}
	childMachine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "idle",
		States:  xs.States{{Key: "idle"}},
	})

	parentMachine := xs.CreateMachine(xs.MachineConfig[ctx]{
		ID:      "form",
		Initial: "present",
		Context: ctx{},
		Entry: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
			c := a.Context
			c.MachineRef = a.Spawn(childMachine, xs.SpawnOptions{ID: "machineChild"})
			c.PromiseRef = a.Spawn(
				// new Promise(() => {}): never settles while the actor is alive.
				xs.FromPromise(func(pctx context.Context, _ xs.PromiseArgs) (any, error) {
					<-pctx.Done()
					return nil, pctx.Err()
				}),
				xs.SpawnOptions{ID: "promiseChild"},
			)
			c.ObservableRef = a.Spawn(
				xs.FromObservable(func(_ xs.ObservableArgs) xs.Subscribable[int] { return rxInterval(1000) }),
				xs.SpawnOptions{ID: "observableChild"},
			)
			return c
		})},
		States: xs.States{
			{Key: "present", On: map[string]xs.Transitions{
				"NEXT": {{
					Target: "gone",
					Actions: xs.Actions{
						xs.StopChild(xs.NewExpr(func(a xs.ExprArgs[ctx]) any { return a.Context.MachineRef })),
						xs.StopChild(xs.NewExpr(func(a xs.ExprArgs[ctx]) any { return a.Context.PromiseRef })),
						xs.StopChild(xs.NewExpr(func(a xs.ExprArgs[ctx]) any { return a.Context.ObservableRef })),
					},
				}},
			}},
			{Key: "gone", Type: xs.Final},
		},
	})

	service := xs.CreateActor(parentMachine).Start()

	assert.Contains(t, service.GetSnapshot().Children, "machineChild")
	assert.Contains(t, service.GetSnapshot().Children, "promiseChild")
	assert.Contains(t, service.GetSnapshot().Children, "observableChild")

	service.Send(xs.Ev("NEXT"))

	assert.Nil(t, service.GetSnapshot().Children["machineChild"])
	assert.Nil(t, service.GetSnapshot().Children["promiseChild"])
	assert.Nil(t, service.GetSnapshot().Children["observableChild"])
}

// JS: interpreter > shouldn't execute actions when reading a snapshot of not started actor
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/interpreter.test.ts#L1798
func TestInterpreter_ShouldntExecuteActionsWhenReadingASnapshotOfNotStartedActor(t *testing.T) {
	s := newSpy()
	actorRef := xs.CreateActor(xs.CreateMachine(xs.MachineConfig[any]{
		Entry: xs.Actions{xs.ActionFunc(func(_ xs.ActionArgs[any]) {
			s.Call()
		})},
	}))

	actorRef.GetSnapshot()

	assert.Equal(t, 0, s.Count())
}

// JS: interpreter > should execute entry actions when starting the actor after reading its snapshot first
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/interpreter.test.ts#L1813
func TestInterpreter_ShouldExecuteEntryActionsWhenStartingTheActorAfterReadingItsSnapshotFirst(t *testing.T) {
	s := newSpy()

	actorRef := xs.CreateActor(xs.CreateMachine(xs.MachineConfig[any]{
		Entry: xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[any]) { s.Call(a) })},
	}))

	actorRef.GetSnapshot()
	assert.Equal(t, 0, s.Count())

	actorRef.Start()

	assert.Greater(t, s.Count(), 0)
}

// JS: interpreter > the first state of an actor should be its initial state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/interpreter.test.ts#L1830
func TestInterpreter_TheFirstStateOfAnActorShouldBeItsInitialState(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{})
	actor := xs.CreateActor(machine)
	initialState := actor.GetSnapshot()

	actor.Start()

	assert.Same(t, initialState, actor.GetSnapshot())
}

// JS: interpreter > should call an onDone callback immediately if the service is already done
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/interpreter.test.ts#L1840
func TestInterpreter_ShouldCallAnOnDoneCallbackImmediatelyIfTheServiceIsAlreadyDone(t *testing.T) {
	sig := newSignal()
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States:  xs.States{{Key: "a", Type: xs.Final}},
	})

	service := xs.CreateActor(machine).Start()

	assert.Equal(t, xs.StatusDone, service.GetSnapshot().Status)

	service.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{
		Complete: func() {
			sig.Resolve()
		},
	})
	sig.Wait(t)
}

// JS: should throw if an event is received
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/interpreter.test.ts#L1864
func TestInterpreter_ShouldThrowIfAnEventIsReceived(t *testing.T) {
	t.Skip("N/A: type-level only — JS sends a bare string ('EVENT') instead of an event object and expects a throw; Go's Send(event xs.Event) rejects a string at compile time")
}

// JS: should not process events sent directly to own actor ref before initial entry actions are processed
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/interpreter.test.ts#L1877
func TestInterpreter_ShouldNotProcessEventsSentDirectlyToOwnActorRefBeforeInitialEntryActionsAreProcessed(t *testing.T) {
	actual := []string{}
	var actorRef *xs.Actor[*xs.MachineSnapshot[any]]
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Entry: xs.Actions{xs.ActionFunc(func(_ xs.ActionArgs[any]) {
			actual = append(actual, "initial root entry start")
			actorRef.Send(xs.Ev("EV"))
			actual = append(actual, "initial root entry end")
		})},
		On: map[string]xs.Transitions{
			"EV": {{Actions: xs.Actions{xs.ActionFunc(func(_ xs.ActionArgs[any]) {
				actual = append(actual, "EV transition")
			})}}},
		},
		Initial: "a",
		States: xs.States{
			{Key: "a", Entry: xs.Actions{xs.ActionFunc(func(_ xs.ActionArgs[any]) {
				actual = append(actual, "initial nested entry")
			})}},
		},
	})

	actorRef = xs.CreateActor(machine)
	actorRef.Start()

	assert.Equal(t, []string{
		"initial root entry start",
		"initial root entry end",
		"initial nested entry",
		"EV transition",
	}, actual)
}

// JS: should not notify the completion observer for an active logic when it gets subscribed before starting
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/interpreter.test.ts#L1915
func TestInterpreter_ShouldNotNotifyCompletionObserverForActiveLogicSubscribedBeforeStarting(t *testing.T) {
	s := newSpy()

	machine := xs.CreateMachine(xs.MachineConfig[any]{})
	xs.CreateActor(machine).Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{
		Complete: func() { s.Call() },
	})

	assert.Equal(t, 0, s.Count())
}

// JS: should notify the error observer for an errored logic when it gets subscribed after it errors
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/interpreter.test.ts#L1924
func TestInterpreter_ShouldNotifyErrorObserverForErroredLogicSubscribedAfterItErrors(t *testing.T) {
	s := newSpy()

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Entry: xs.Actions{xs.ActionFunc(func(_ xs.ActionArgs[any]) {
			panic(errors.New("error"))
		})},
	})
	actorRef := xs.CreateActor(machine)
	actorRef.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{Error: func(_ any) {}})
	actorRef.Start()

	actorRef.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{
		Error: func(err any) { s.Call(err) },
	})

	// toMatchInlineSnapshot: [[Error: error]]
	assert.Equal(t, [][]any{{errors.New("error")}}, s.Calls())
}
