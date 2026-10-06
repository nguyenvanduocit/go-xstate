package xstate_test

import (
	"context"
	"testing"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// JS: predictableExec > should call mixed custom and builtin actions in the definitions order
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/predictableExec.test.ts#L12
func TestPredictableExec_ShouldCallMixedCustomAndBuiltinActionsInTheDefinitionsOrder(t *testing.T) {
	type ctx struct{}
	actual := []string{}

	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Initial: "a",
		Context: ctx{},
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{"NEXT": {{Target: "b"}}}},
			{Key: "b", Entry: xs.Actions{
				xs.ActionFunc(func(a xs.ActionArgs[ctx]) {
					actual = append(actual, "custom")
				}),
				xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
					actual = append(actual, "assign")
					return ctx{}
				}),
			}},
		},
	})

	service := xs.CreateActor(machine).Start()
	service.Send(xs.Ev("NEXT"))

	assert.Equal(t, []string{"custom", "assign"}, actual)
}

// JS: predictableExec > should call initial custom actions when starting a service
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/predictableExec.test.ts#L42
func TestPredictableExec_ShouldCallInitialCustomActionsWhenStartingAService(t *testing.T) {
	called := false
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Entry: xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[any]) {
			called = true
		})},
	})

	assert.False(t, called)

	xs.CreateActor(machine).Start()

	assert.True(t, called)
}

// JS: predictableExec > should resolve initial assign actions before starting a service
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/predictableExec.test.ts#L57
func TestPredictableExec_ShouldResolveInitialAssignActionsBeforeStartingAService(t *testing.T) {
	type ctx struct{ Called bool }
	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Context: ctx{Called: false},
		Entry: xs.Actions{
			xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
				c := a.Context
				c.Called = true
				return c
			}),
		},
	})

	assert.True(t, xs.CreateActor(machine).GetSnapshot().Context.Called)
}

// JS: predictableExec > should call raised transition custom actions with raised event
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/predictableExec.test.ts#L72
func TestPredictableExec_ShouldCallRaisedTransitionCustomActionsWithRaisedEvent(t *testing.T) {
	var eventArg xs.Event
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{"NEXT": {{Target: "b"}}}},
			{
				Key: "b",
				On: map[string]xs.Transitions{
					"RAISED": {{
						Target: "c",
						Actions: xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[any]) {
							eventArg = a.Event
						})},
					}},
				},
				Entry: xs.Actions{xs.Raise(xs.Ev("RAISED"))},
			},
			{Key: "c"},
		},
	})

	service := xs.CreateActor(machine).Start()
	service.Send(xs.Ev("NEXT"))

	require.NotNil(t, eventArg)
	assert.Equal(t, "RAISED", eventArg.EventType())
}

// JS: predictableExec > should call raised transition builtin actions with raised event
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/predictableExec.test.ts#L101
func TestPredictableExec_ShouldCallRaisedTransitionBuiltinActionsWithRaisedEvent(t *testing.T) {
	type ctx struct{}
	var eventArg xs.Event
	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Context: ctx{},
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{"NEXT": {{Target: "b"}}}},
			{
				Key: "b",
				On: map[string]xs.Transitions{
					"RAISED": {{
						Target: "c",
						Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
							eventArg = a.Event
							return ctx{}
						})},
					}},
				},
				Entry: xs.Actions{xs.Raise(xs.Ev("RAISED"))},
			},
			{Key: "c"},
		},
	})

	service := xs.CreateActor(machine).Start()
	service.Send(xs.Ev("NEXT"))

	require.NotNil(t, eventArg)
	assert.Equal(t, "RAISED", eventArg.EventType())
}

// JS: predictableExec > should call invoke creator with raised event
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/predictableExec.test.ts#L134
func TestPredictableExec_ShouldCallInvokeCreatorWithRaisedEvent(t *testing.T) {
	type ctx struct{}
	var eventArg xs.Event
	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Context: ctx{},
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{"NEXT": {{Target: "b"}}}},
			{
				Key:   "b",
				On:    map[string]xs.Transitions{"RAISED": {{Target: "c"}}},
				Entry: xs.Actions{xs.Raise(xs.Ev("RAISED"))},
			},
			{
				Key: "c",
				Invoke: []xs.InvokeConfig{{
					Logic: xs.FromCallback(func(a xs.CallbackArgs) func() {
						eventArg = a.Input.(map[string]any)["event"].(xs.Event)
						return nil
					}),
					Input: xs.NewExpr(func(a xs.ExprArgs[ctx]) any {
						return map[string]any{"event": a.Event}
					}),
				}},
			},
		},
	})

	service := xs.CreateActor(machine).Start()
	service.Send(xs.Ev("NEXT"))

	require.NotNil(t, eventArg)
	assert.Equal(t, "RAISED", eventArg.EventType())
}

// JS: predictableExec > invoked child should be available on the new state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/predictableExec.test.ts#L168
func TestPredictableExec_InvokedChildShouldBeAvailableOnTheNewState(t *testing.T) {
	type ctx struct{}
	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Context: ctx{},
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{"NEXT": {{Target: "b"}}}},
			{
				Key: "b",
				Invoke: []xs.InvokeConfig{{
					ID:    "myChild",
					Logic: xs.FromCallback(func(a xs.CallbackArgs) func() { return nil }),
				}},
			},
		},
	})

	service := xs.CreateActor(machine).Start()
	service.Send(xs.Ev("NEXT"))

	assert.NotNil(t, service.GetSnapshot().Children["myChild"])
}

// JS: predictableExec > invoked child should not be available on the state after leaving invoking state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/predictableExec.test.ts#L193
func TestPredictableExec_InvokedChildShouldNotBeAvailableAfterLeavingInvokingState(t *testing.T) {
	type ctx struct{}
	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Context: ctx{},
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{"NEXT": {{Target: "b"}}}},
			{
				Key: "b",
				Invoke: []xs.InvokeConfig{{
					ID:    "myChild",
					Logic: xs.FromCallback(func(a xs.CallbackArgs) func() { return nil }),
				}},
				On: map[string]xs.Transitions{"NEXT": {{Target: "c"}}},
			},
			{Key: "c"},
		},
	})

	service := xs.CreateActor(machine).Start()
	service.Send(xs.Ev("NEXT"))
	service.Send(xs.Ev("NEXT"))

	assert.Nil(t, service.GetSnapshot().Children["myChild"])
}

// JS: predictableExec > should correctly provide intermediate context value to a custom action executed in between assign actions
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/predictableExec.test.ts#L223
func TestPredictableExec_ShouldProvideIntermediateContextToCustomActionBetweenAssigns(t *testing.T) {
	type ctx struct{ Counter int }
	calledWith := 0
	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Context: ctx{Counter: 0},
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{"NEXT": {{Target: "b"}}}},
			{Key: "b", Entry: xs.Actions{
				xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
					c := a.Context
					c.Counter = 1
					return c
				}),
				xs.ActionFunc(func(a xs.ActionArgs[ctx]) {
					calledWith = a.Context.Counter
				}),
				xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
					c := a.Context
					c.Counter = 2
					return c
				}),
			}},
		},
	})

	service := xs.CreateActor(machine).Start()
	service.Send(xs.Ev("NEXT"))

	assert.Equal(t, 1, calledWith)
}

// JS: predictableExec > initial actions should receive context updated only by preceding assign actions
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/predictableExec.test.ts#L252
func TestPredictableExec_InitialActionsShouldReceiveContextUpdatedOnlyByPrecedingAssigns(t *testing.T) {
	type ctx struct{ Count int }
	actual := []int{}

	push := xs.ActionFunc(func(a xs.ActionArgs[ctx]) {
		actual = append(actual, a.Context.Count)
	})
	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Context: ctx{Count: 0},
		Entry: xs.Actions{
			push,
			xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
				c := a.Context
				c.Count = 1
				return c
			}),
			push,
			xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
				c := a.Context
				c.Count = 2
				return c
			}),
			push,
		},
	})

	xs.CreateActor(machine).Start()

	assert.Equal(t, []int{0, 1, 2}, actual)
}

// JS: predictableExec > parent should be able to read the updated state of a child when receiving an event from it
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/predictableExec.test.ts#L271
func TestPredictableExec_ParentShouldReadUpdatedStateOfChildWhenReceivingEventFromIt(t *testing.T) {
	sig := newSignal()

	child := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{
				Key: "a",
				// we need to clear the call stack before we send the event to the parent
				After: map[string]xs.Transitions{"1": {{Target: "b"}}},
			},
			{Key: "b", Entry: xs.Actions{xs.SendParent(xs.Ev("CHILD_UPDATED"))}},
		},
	})

	var service *xs.Actor[*xs.MachineSnapshot[any]]

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Invoke:  []xs.InvokeConfig{{ID: "myChild", Logic: child}},
		Initial: "initial",
		States: xs.States{
			{
				Key: "initial",
				On: map[string]xs.Transitions{
					"CHILD_UPDATED": {
						{
							Guard: xs.GuardFunc(func(a xs.GuardArgs[any]) bool {
								return machineSnap[any](service.GetSnapshot().Children["myChild"]).Value == "b"
							}),
							Target: "success",
						},
						{Target: "fail"},
					},
				},
			},
			{Key: "success", Type: xs.Final},
			{Key: "fail", Type: xs.Final},
		},
	})

	service = xs.CreateActor(machine)
	service.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{
		Complete: func() {
			assert.Equal(t, "success", service.GetSnapshot().Value)
			sig.Resolve()
		},
	})
	service.Start()

	sig.Wait(t)
}

// JS: predictableExec > should be possible to send immediate events to initially invoked actors
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/predictableExec.test.ts#L337
func TestPredictableExec_ShouldBePossibleToSendImmediateEventsToInitiallyInvokedActors(t *testing.T) {
	child := xs.CreateMachine(xs.MachineConfig[any]{
		On: map[string]xs.Transitions{
			"PING": {{Actions: xs.Actions{xs.SendParent(xs.Ev("PONG"))}}},
		},
	})

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "waiting",
		States: xs.States{
			{
				Key:    "waiting",
				Invoke: []xs.InvokeConfig{{ID: "ponger", Logic: child}},
				Entry:  xs.Actions{xs.SendTo("ponger", xs.Ev("PING"))},
				On:     map[string]xs.Transitions{"PONG": {{Target: "done"}}},
			},
			{Key: "done", Type: xs.Final},
		},
	})

	service := xs.CreateActor(machine).Start()

	assert.Equal(t, "done", service.GetSnapshot().Value)
}

// JS: predictableExec > should create invoke based on context updated by entry actions of the same state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/predictableExec.test.ts#L370
func TestPredictableExec_ShouldCreateInvokeBasedOnContextUpdatedByEntryActionsOfSameState(t *testing.T) {
	type ctx struct{ Updated bool }
	sig := newSignal()

	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Context: ctx{Updated: false},
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{"NEXT": {{Target: "b"}}}},
			{
				Key: "b",
				Entry: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
					c := a.Context
					c.Updated = true
					return c
				})},
				Invoke: []xs.InvokeConfig{{
					Logic: xs.FromPromise(func(_ context.Context, a xs.PromiseArgs) (any, error) {
						in, _ := a.Input.(map[string]any)
						assert.Equal(t, true, in["updated"])
						sig.Resolve()
						return nil, nil
					}),
					Input: xs.NewExpr(func(a xs.ExprArgs[ctx]) any {
						return map[string]any{"updated": a.Context.Updated}
					}),
				}},
			},
		},
	})

	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.Ev("NEXT"))

	sig.Wait(t)
}

// JS: predictableExec > should deliver events sent from the entry actions to a service invoked in the same state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/predictableExec.test.ts#L406
func TestPredictableExec_ShouldDeliverEntrySentEventsToServiceInvokedInSameState(t *testing.T) {
	type ctx struct{ Updated bool }
	var received xs.Event

	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Context: ctx{Updated: false},
		Initial: "a",
		States: xs.States{
			{Key: "a", On: map[string]xs.Transitions{"NEXT": {{Target: "b"}}}},
			{
				Key:   "b",
				Entry: xs.Actions{xs.SendTo("myChild", xs.Ev("KNOCK_KNOCK"))},
				Invoke: []xs.InvokeConfig{{
					ID: "myChild",
					Logic: xs.CreateMachine(xs.MachineConfig[any]{
						On: map[string]xs.Transitions{
							"*": {{Actions: xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[any]) {
								received = a.Event
							})}}},
						},
					}),
				}},
			},
		},
	})

	service := xs.CreateActor(machine).Start()
	service.Send(xs.Ev("NEXT"))

	assert.Equal(t, xs.Ev("KNOCK_KNOCK"), received)
}

// JS: predictableExec > parent should be able to read the updated state of a child when receiving an event from it
// (second, duplicate JS test with the same name; guard written as an expression-bodied arrow)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/predictableExec.test.ts#L444
func TestPredictableExec_ParentShouldReadUpdatedStateOfChildWhenReceivingEventFromIt_2(t *testing.T) {
	sig := newSignal()

	child := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{
				Key: "a",
				// we need to clear the call stack before we send the event to the parent
				After: map[string]xs.Transitions{"1": {{Target: "b"}}},
			},
			{Key: "b", Entry: xs.Actions{xs.SendParent(xs.Ev("CHILD_UPDATED"))}},
		},
	})

	var service *xs.Actor[*xs.MachineSnapshot[any]]

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Invoke:  []xs.InvokeConfig{{ID: "myChild", Logic: child}},
		Initial: "initial",
		States: xs.States{
			{
				Key: "initial",
				On: map[string]xs.Transitions{
					"CHILD_UPDATED": {
						{
							Guard: xs.GuardFunc(func(a xs.GuardArgs[any]) bool {
								return machineSnap[any](service.GetSnapshot().Children["myChild"]).Value == "b"
							}),
							Target: "success",
						},
						{Target: "fail"},
					},
				},
			},
			{Key: "success", Type: xs.Final},
			{Key: "fail", Type: xs.Final},
		},
	})

	service = xs.CreateActor(machine)
	service.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{
		Complete: func() {
			assert.Equal(t, "success", service.GetSnapshot().Value)
			sig.Resolve()
		},
	})
	service.Start()

	sig.Wait(t)
}

// JS: predictableExec > should be possible to send immediate events to initially invoked actors
// (second, duplicate JS test with the same name and identical body)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/predictableExec.test.ts#L507
func TestPredictableExec_ShouldBePossibleToSendImmediateEventsToInitiallyInvokedActors_2(t *testing.T) {
	child := xs.CreateMachine(xs.MachineConfig[any]{
		On: map[string]xs.Transitions{
			"PING": {{Actions: xs.Actions{xs.SendParent(xs.Ev("PONG"))}}},
		},
	})

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "waiting",
		States: xs.States{
			{
				Key:    "waiting",
				Invoke: []xs.InvokeConfig{{ID: "ponger", Logic: child}},
				Entry:  xs.Actions{xs.SendTo("ponger", xs.Ev("PING"))},
				On:     map[string]xs.Transitions{"PONG": {{Target: "done"}}},
			},
			{Key: "done", Type: xs.Final},
		},
	})

	service := xs.CreateActor(machine).Start()

	assert.Equal(t, "done", service.GetSnapshot().Value)
}

// JS: predictableExec > should deliver events sent from the exit actions to a service invoked in the same state
// https://github.com/statelyai/xstate/issues/3617
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/predictableExec.test.ts#L541
func TestPredictableExec_ShouldDeliverExitSentEventsToServiceInvokedInSameState(t *testing.T) {
	sig := newSignal()

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "active",
		States: xs.States{
			{
				Key: "active",
				Invoke: []xs.InvokeConfig{{
					ID: "my-service",
					Logic: xs.FromCallback(func(a xs.CallbackArgs) func() {
						a.Receive(func(event xs.Event) {
							if event.EventType() == "MY_EVENT" {
								sig.Resolve()
							}
						})
						return nil
					}),
				}},
				Exit: xs.Actions{xs.SendTo("my-service", xs.Ev("MY_EVENT"))},
				On:   map[string]xs.Transitions{"TOGGLE": {{Target: "inactive"}}},
			},
			{Key: "inactive"},
		},
	})

	actor := xs.CreateActor(machine).Start()

	actor.Send(xs.Ev("TOGGLE"))

	sig.Wait(t)
}
