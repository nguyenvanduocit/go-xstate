package xstate_test

import (
	"testing"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/assert"
)

// assign1CounterCtx mirrors `interface CounterContext { count; foo; maybe? }`.
// Maybe is a pointer so that "undefined" (nil) is distinguishable from "".
type assign1CounterCtx struct {
	Count int
	Foo   string
	Maybe *string
}

func assign1StrPtr(s string) *string { return &s }

// assign1CreateCounterMachine mirrors createCounterMachine(context). JS merges
// `{ count: 0, foo: 'bar', ...context }`; every JS caller passes either nothing
// (=> {count: 0, foo: 'bar'}) or both fields, so callers pass the full context.
func assign1CreateCounterMachine(context assign1CounterCtx) *xs.StateMachine[assign1CounterCtx] {
	type C = assign1CounterCtx
	return xs.CreateMachine(xs.MachineConfig[C]{
		Initial: "counting",
		Context: context,
		States: xs.States{
			{Key: "counting", On: map[string]xs.Transitions{
				// assign(({ context }) => ({ count: context.count + 1 }))
				"INC": {{
					Target: "counting",
					Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[C]) C {
						c := a.Context
						c.Count = a.Context.Count + 1
						return c
					})},
				}},
				// assign({ count: ({ context }) => context.count - 1 })
				"DEC": {{
					Target: "counting",
					Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[C]) C {
						c := a.Context
						c.Count = a.Context.Count - 1
						return c
					})},
				}},
				// assign({ count: () => 100, foo: () => 'win' })
				"WIN_PROP": {{
					Target: "counting",
					Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[C]) C {
						c := a.Context
						c.Count = 100
						c.Foo = "win"
						return c
					})},
				}},
				// assign({ count: 100, foo: 'win' })
				"WIN_STATIC": {{
					Target: "counting",
					Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[C]) C {
						c := a.Context
						c.Count = 100
						c.Foo = "win"
						return c
					})},
				}},
				// assign({ count: () => 100, foo: 'win' })
				"WIN_MIX": {{
					Target: "counting",
					Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[C]) C {
						c := a.Context
						c.Count = 100
						c.Foo = "win"
						return c
					})},
				}},
				// assign(() => ({ count: 100, foo: 'win' }))
				"WIN": {{
					Target: "counting",
					Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[C]) C {
						c := a.Context
						c.Count = 100
						c.Foo = "win"
						return c
					})},
				}},
				// assign({ maybe: 'defined' }) — targetless
				"SET_MAYBE": {{
					Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[C]) C {
						c := a.Context
						c.Maybe = assign1StrPtr("defined")
						return c
					})},
				}},
			}},
		},
	})
}

// JS: assign > applies the assignment to the external state (property assignment)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/assign.test.ts#L94
func TestAssign_AppliesAssignmentToExternalStatePropertyAssignment(t *testing.T) {
	counterMachine := assign1CreateCounterMachine(assign1CounterCtx{Count: 0, Foo: "bar"})

	actorRef := xs.CreateActor(counterMachine).Start()
	actorRef.Send(xs.Ev("DEC"))
	oneState := actorRef.GetSnapshot()

	assert.Equal(t, "counting", oneState.Value)
	assert.Equal(t, assign1CounterCtx{Count: -1, Foo: "bar"}, oneState.Context)

	actorRef.Send(xs.Ev("DEC"))
	twoState := actorRef.GetSnapshot()

	assert.Equal(t, "counting", twoState.Value)
	assert.Equal(t, assign1CounterCtx{Count: -2, Foo: "bar"}, twoState.Context)
}

// JS: assign > applies the assignment to the external state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/assign.test.ts#L113
func TestAssign_AppliesAssignmentToExternalState(t *testing.T) {
	counterMachine := assign1CreateCounterMachine(assign1CounterCtx{Count: 0, Foo: "bar"})

	actorRef := xs.CreateActor(counterMachine).Start()
	actorRef.Send(xs.Ev("INC"))
	oneState := actorRef.GetSnapshot()

	assert.Equal(t, "counting", oneState.Value)
	assert.Equal(t, assign1CounterCtx{Count: 1, Foo: "bar"}, oneState.Context)

	actorRef.Send(xs.Ev("INC"))
	twoState := actorRef.GetSnapshot()

	assert.Equal(t, "counting", twoState.Value)
	assert.Equal(t, assign1CounterCtx{Count: 2, Foo: "bar"}, twoState.Context)
}

// JS: assign > applies the assignment to multiple properties (property assignment)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/assign.test.ts#L132
func TestAssign_AppliesAssignmentToMultiplePropertiesPropertyAssignment(t *testing.T) {
	counterMachine := assign1CreateCounterMachine(assign1CounterCtx{Count: 0, Foo: "bar"})
	actorRef := xs.CreateActor(counterMachine).Start()
	actorRef.Send(xs.Ev("WIN_PROP"))

	assert.Equal(t, assign1CounterCtx{Count: 100, Foo: "win"}, actorRef.GetSnapshot().Context)
}

// JS: assign > applies the assignment to multiple properties (static)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/assign.test.ts#L142
func TestAssign_AppliesAssignmentToMultiplePropertiesStatic(t *testing.T) {
	counterMachine := assign1CreateCounterMachine(assign1CounterCtx{Count: 0, Foo: "bar"})
	actorRef := xs.CreateActor(counterMachine).Start()
	actorRef.Send(xs.Ev("WIN_STATIC"))

	assert.Equal(t, assign1CounterCtx{Count: 100, Foo: "win"}, actorRef.GetSnapshot().Context)
}

// JS: assign > applies the assignment to multiple properties (static + prop assignment)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/assign.test.ts#L152
func TestAssign_AppliesAssignmentToMultiplePropertiesStaticPlusPropAssignment(t *testing.T) {
	counterMachine := assign1CreateCounterMachine(assign1CounterCtx{Count: 0, Foo: "bar"})
	actorRef := xs.CreateActor(counterMachine).Start()
	actorRef.Send(xs.Ev("WIN_MIX"))

	assert.Equal(t, assign1CounterCtx{Count: 100, Foo: "win"}, actorRef.GetSnapshot().Context)
}

// JS: assign > applies the assignment to multiple properties
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/assign.test.ts#L162
func TestAssign_AppliesAssignmentToMultipleProperties(t *testing.T) {
	counterMachine := assign1CreateCounterMachine(assign1CounterCtx{Count: 0, Foo: "bar"})
	actorRef := xs.CreateActor(counterMachine).Start()
	actorRef.Send(xs.Ev("WIN"))

	assert.Equal(t, assign1CounterCtx{Count: 100, Foo: "win"}, actorRef.GetSnapshot().Context)
}

// JS: assign > applies the assignment to the explicit external state (property assignment)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/assign.test.ts#L172
func TestAssign_AppliesAssignmentToExplicitExternalStatePropertyAssignment(t *testing.T) {
	machine := assign1CreateCounterMachine(assign1CounterCtx{Count: 50, Foo: "bar"})
	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.Ev("DEC"))
	oneState := actorRef.GetSnapshot()

	assert.Equal(t, "counting", oneState.Value)
	assert.Equal(t, assign1CounterCtx{Count: 49, Foo: "bar"}, oneState.Context)

	actorRef.Send(xs.Ev("DEC"))
	twoState := actorRef.GetSnapshot()

	assert.Equal(t, "counting", twoState.Value)
	assert.Equal(t, assign1CounterCtx{Count: 48, Foo: "bar"}, twoState.Context)

	machine2 := assign1CreateCounterMachine(assign1CounterCtx{Count: 100, Foo: "bar"})

	actorRef2 := xs.CreateActor(machine2).Start()
	actorRef2.Send(xs.Ev("DEC"))
	threeState := actorRef2.GetSnapshot()

	assert.Equal(t, "counting", threeState.Value)
	assert.Equal(t, assign1CounterCtx{Count: 99, Foo: "bar"}, threeState.Context)
}

// JS: assign > applies the assignment to the explicit external state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/assign.test.ts#L197
func TestAssign_AppliesAssignmentToExplicitExternalState(t *testing.T) {
	machine := assign1CreateCounterMachine(assign1CounterCtx{Count: 50, Foo: "bar"})
	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.Ev("INC"))
	oneState := actorRef.GetSnapshot()

	assert.Equal(t, "counting", oneState.Value)
	assert.Equal(t, assign1CounterCtx{Count: 51, Foo: "bar"}, oneState.Context)

	actorRef.Send(xs.Ev("INC"))
	twoState := actorRef.GetSnapshot()

	assert.Equal(t, "counting", twoState.Value)
	assert.Equal(t, assign1CounterCtx{Count: 52, Foo: "bar"}, twoState.Context)

	machine2 := assign1CreateCounterMachine(assign1CounterCtx{Count: 102, Foo: "bar"})

	actorRef2 := xs.CreateActor(machine2).Start()
	actorRef2.Send(xs.Ev("INC"))
	threeState := actorRef2.GetSnapshot()

	assert.Equal(t, "counting", threeState.Value)
	assert.Equal(t, assign1CounterCtx{Count: 103, Foo: "bar"}, threeState.Context)
}

// JS: assign > should maintain state after unhandled event
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/assign.test.ts#L222
func TestAssign_ShouldMaintainStateAfterUnhandledEvent(t *testing.T) {
	counterMachine := assign1CreateCounterMachine(assign1CounterCtx{Count: 0, Foo: "bar"})
	actorRef := xs.CreateActor(counterMachine).Start()

	actorRef.Send(xs.Ev("FAKE_EVENT"))
	nextState := actorRef.GetSnapshot()

	// expect(nextState.context).toBeDefined() — the context is a value type in
	// Go; the snapshot itself must exist for the context to be read.
	assert.NotNil(t, nextState)
	assert.Equal(t, assign1CounterCtx{Count: 0, Foo: "bar"}, nextState.Context)
}

// JS: assign > sets undefined properties
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/assign.test.ts#L235
func TestAssign_SetsUndefinedProperties(t *testing.T) {
	counterMachine := assign1CreateCounterMachine(assign1CounterCtx{Count: 0, Foo: "bar"})
	actorRef := xs.CreateActor(counterMachine).Start()

	actorRef.Send(xs.Ev("SET_MAYBE"))

	nextState := actorRef.GetSnapshot()

	assert.NotNil(t, nextState.Context.Maybe)
	assert.Equal(t, assign1CounterCtx{
		Count: 0,
		Foo:   "bar",
		Maybe: assign1StrPtr("defined"),
	}, nextState.Context)
}

// JS: assign > can assign from event
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/assign.test.ts#L253
func TestAssign_CanAssignFromEvent(t *testing.T) {
	type ctx struct{ Count int }

	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Initial: "active",
		Context: ctx{Count: 0},
		States: xs.States{
			{Key: "active", On: map[string]xs.Transitions{
				"INC": {{Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
					c := a.Context
					c.Count = a.Event.(xs.E)["value"].(int)
					return c
				})}}},
			}},
		},
	})

	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.E{"type": "INC", "value": 30})

	assert.Equal(t, 30, actorRef.GetSnapshot().Context.Count)
}

// JS: assign meta > should provide the parametrized action to the assigner
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/assign.test.ts#L284
func TestAssign_Meta_ShouldProvideParametrizedActionToAssigner(t *testing.T) {
	type ctx struct{ Count int }

	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Context: ctx{Count: 1},
		Entry:   xs.Actions{xs.ActionRef{Type: "inc", Params: map[string]any{"by": 10}}},
	}, xs.Implementations{Actions: map[string]xs.Action{
		// assign(({ context }, params) => ({ count: context.count + params.by }))
		"inc": xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
			by := a.Params.(map[string]any)["by"].(int)
			return ctx{Count: a.Context.Count + by}
		}),
	}})

	actor := xs.CreateActor(machine).Start()

	assert.Equal(t, 11, actor.GetSnapshot().Context.Count)
}

// JS: assign meta > should provide the action parameters to the partial assigner
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/assign.test.ts#L310
func TestAssign_Meta_ShouldProvideActionParametersToPartialAssigner(t *testing.T) {
	type ctx struct{ Count int }

	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Context: ctx{Count: 1},
		Entry:   xs.Actions{xs.ActionRef{Type: "inc", Params: map[string]any{"by": 10}}},
	}, xs.Implementations{Actions: map[string]xs.Action{
		// assign({ count: ({ context }, params) => context.count + params.by })
		"inc": xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
			c := a.Context
			c.Count = a.Context.Count + a.Params.(map[string]any)["by"].(int)
			return c
		}),
	}})

	actor := xs.CreateActor(machine).Start()

	assert.Equal(t, 11, actor.GetSnapshot().Context.Count)
}

// JS: assign meta > a parameterized action that resolves to assign() should be provided the params
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/assign.test.ts#L336
func TestAssign_Meta_ParameterizedActionResolvingToAssignShouldBeProvidedParams(t *testing.T) {
	sig := newSignal()
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		On: map[string]xs.Transitions{
			"EVENT": {{Actions: xs.Actions{xs.ActionRef{Type: "inc", Params: map[string]any{"value": 5}}}}},
		},
	}, xs.Implementations{Actions: map[string]xs.Action{
		"inc": xs.Assign(func(a xs.AssignArgs[any]) any {
			assert.Equal(t, map[string]any{"value": 5}, a.Params)
			sig.Resolve()
			return a.Context
		}),
	}})

	service := xs.CreateActor(machine).Start()

	service.Send(xs.Ev("EVENT"))

	sig.Wait(t)
}
