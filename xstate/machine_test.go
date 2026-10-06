package xstate_test

import (
	"sort"
	"testing"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/assert"
)

// machine1PedestrianStates mirrors the top-level `pedestrianStates` object
// (machine.test.ts lines 3-18), spread into the `red` state of lightMachine.
func machine1PedestrianStates(cfg xs.StateConfig) xs.StateConfig {
	cfg.Initial = "walk"
	cfg.States = xs.States{
		{Key: "walk", On: map[string]xs.Transitions{"PED_COUNTDOWN": {{Target: "wait"}}}},
		{Key: "wait", On: map[string]xs.Transitions{"PED_COUNTDOWN": {{Target: "stop"}}}},
		{Key: "stop"},
	}
	return cfg
}

// machine1LightMachine mirrors the top-level `lightMachine` (machine.test.ts
// lines 20-44). It is a func (not a package var) so the stubbed CreateMachine
// panic stays inside the tests that use it.
func machine1LightMachine() *xs.StateMachine[any] {
	return xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "green",
		States: xs.States{
			{Key: "green", On: map[string]xs.Transitions{
				"TIMER":           {{Target: "yellow"}},
				"POWER_OUTAGE":    {{Target: "red"}},
				"FORBIDDEN_EVENT": nil,
			}},
			{Key: "yellow", On: map[string]xs.Transitions{
				"TIMER":        {{Target: "red"}},
				"POWER_OUTAGE": {{Target: "red"}},
			}},
			machine1PedestrianStates(xs.StateConfig{Key: "red", On: map[string]xs.Transitions{
				"TIMER":        {{Target: "green"}},
				"POWER_OUTAGE": {{Target: "red"}},
			}}),
		},
	})
}

// JS: machine > machine.states > should properly register machine states
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/machine.test.ts#L48
func TestMachine_MachineStates_ShouldProperlyRegisterMachineStates(t *testing.T) {
	lightMachine := machine1LightMachine()

	keys := []string{}
	for _, child := range lightMachine.Root.ChildStates() {
		keys = append(keys, child.Key)
	}
	assert.Equal(t, []string{"green", "yellow", "red"}, keys)
}

// JS: machine > machine.events > should return the set of events accepted by machine
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/machine.test.ts#L58
func TestMachine_MachineEvents_ShouldReturnTheSetOfEventsAcceptedByMachine(t *testing.T) {
	lightMachine := machine1LightMachine()

	// JS order is ['TIMER', 'POWER_OUTAGE', 'PED_COUNTDOWN'] (key insertion
	// order); the Go API returns events sorted.
	assert.Equal(t, []string{"PED_COUNTDOWN", "POWER_OUTAGE", "TIMER"}, lightMachine.Events())
}

// JS: machine > machine.config > state node config should reference original machine config
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/machine.test.ts#L68
func TestMachine_MachineConfig_StateNodeConfigShouldReferenceOriginalMachineConfig(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "one",
		States: xs.States{
			{
				Key:     "one",
				Initial: "deep",
				States:  xs.States{{Key: "deep"}},
			},
		},
	})

	oneState := machine.States["one"]

	assert.Equal(t, machine.Config.States[0], oneState.Config)

	deepState := machine.States["one"].States["deep"]

	assert.Equal(t, machine.Config.States[0].States[0], deepState.Config)

	// JS also mutates `deepState.config.meta = 'testing meta'` and expects the
	// change to be visible through `machine.config.states.one.states.deep.meta`
	// (object identity). StateNode.Config is a StateConfig value in the Go
	// contract, so reference identity / mutation propagation is not
	// expressible; see the manifest.
}

// JS: machine > machine.provide > should override an action
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/machine.test.ts#L98
func TestMachine_MachineProvide_ShouldOverrideAnAction(t *testing.T) {
	originalEntry := newSpy()
	overridenEntry := newSpy()

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Entry: xs.Actions{xs.ActionRef{Type: "entryAction"}},
	}, xs.Implementations{
		Actions: map[string]xs.Action{
			"entryAction": xs.ActionFunc(func(a xs.ActionArgs[any]) { originalEntry.Call() }),
		},
	})
	differentMachine := machine.Provide(xs.Implementations{
		Actions: map[string]xs.Action{
			"entryAction": xs.ActionFunc(func(a xs.ActionArgs[any]) { overridenEntry.Call() }),
		},
	})

	xs.CreateActor(differentMachine).Start()

	assert.Equal(t, 0, originalEntry.Count())
	assert.Equal(t, 1, overridenEntry.Count())
}

// JS: machine > machine.provide > should override a guard
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/machine.test.ts#L124
func TestMachine_MachineProvide_ShouldOverrideAGuard(t *testing.T) {
	originalGuard := newSpy()
	overridenGuard := newSpy()

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		On: map[string]xs.Transitions{
			"EVENT": {{
				Guard:   xs.GuardRef{Type: "someCondition"},
				Actions: xs.Actions{xs.ActionFunc(func(a xs.ActionArgs[any]) {})},
			}},
		},
	}, xs.Implementations{
		Guards: map[string]xs.Guard{
			"someCondition": xs.GuardFunc(func(a xs.GuardArgs[any]) bool {
				originalGuard.Call()
				return true
			}),
		},
	})

	differentMachine := machine.Provide(xs.Implementations{
		Guards: map[string]xs.Guard{
			"someCondition": xs.GuardFunc(func(a xs.GuardArgs[any]) bool {
				overridenGuard.Call()
				return true
			}),
		},
	})

	actorRef := xs.CreateActor(differentMachine).Start()
	actorRef.Send(xs.Ev("EVENT"))

	assert.Equal(t, 0, originalGuard.Count())
	assert.Equal(t, 1, overridenGuard.Count())
}

// JS: machine > machine.provide > should not override context if not defined
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/machine.test.ts#L155
func TestMachine_MachineProvide_ShouldNotOverrideContextIfNotDefined(t *testing.T) {
	type ctx struct{ Foo string }

	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Context: ctx{Foo: "bar"},
	})
	differentMachine := machine.Provide(xs.Implementations{})
	actorRef := xs.CreateActor(differentMachine).Start()
	assert.Equal(t, ctx{Foo: "bar"}, actorRef.GetSnapshot().Context)
}

// JS: machine > machine.provide > should override context (second argument)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/machine.test.ts#L166
func TestMachine_MachineProvide_ShouldOverrideContextSecondArgument(t *testing.T) {
	t.Skip("skipped in JS")
}

// JS: machine > machine.provide > should throw if initial state is missing in a compound state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/machine.test.ts#L177
func TestMachine_MachineProvide_ShouldThrowIfInitialStateIsMissingInACompoundState(t *testing.T) {
	assert.Panics(t, func() {
		xs.CreateMachine(xs.MachineConfig[any]{
			Initial: "first",
			States: xs.States{
				{
					Key: "first",
					States: xs.States{
						{Key: "second"},
						{Key: "third"},
					},
				},
			},
		})
	})
}

// JS: machine > machine.provide > machines defined without context should have a default empty object for context
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/machine.test.ts#L193
func TestMachine_MachineProvide_MachinesWithoutContextShouldHaveDefaultEmptyObjectContext(t *testing.T) {
	assert.Equal(t, map[string]any{},
		xs.CreateActor(xs.CreateMachine(xs.MachineConfig[map[string]any]{})).GetSnapshot().Context)
}

// JS: machine > machine.provide > should lazily create context for all interpreter instances created from the same machine template created by `provide`
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/machine.test.ts#L197
func TestMachine_MachineProvide_ShouldLazilyCreateContextForAllInstancesFromProvide(t *testing.T) {
	type foo struct{ Prop string }
	type ctx struct{ Foo *foo }

	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		ContextFn: func(a xs.ContextArgs) ctx {
			return ctx{Foo: &foo{Prop: "baz"}}
		},
	})

	copiedMachine := machine.Provide(xs.Implementations{})

	a := xs.CreateActor(copiedMachine).Start()
	b := xs.CreateActor(copiedMachine).Start()

	assert.NotSame(t, a.GetSnapshot().Context.Foo, b.GetSnapshot().Context.Foo)
}

// JS: machine > machine function context > context from a function should be lazily evaluated
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/machine.test.ts#L215
func TestMachine_MachineFunctionContext_ContextFromAFunctionShouldBeLazilyEvaluated(t *testing.T) {
	type bar struct{ Bar string }
	type ctx struct{ Foo *bar }

	config := xs.MachineConfig[ctx]{
		Initial: "active",
		ContextFn: func(a xs.ContextArgs) ctx {
			return ctx{Foo: &bar{Bar: "baz"}}
		},
		States: xs.States{{Key: "active"}},
	}
	testMachine1 := xs.CreateMachine(config)
	testMachine2 := xs.CreateMachine(config)

	initialState1 := xs.CreateActor(testMachine1).GetSnapshot()
	initialState2 := xs.CreateActor(testMachine2).GetSnapshot()

	// JS: expect(initialState1.context).not.toBe(initialState2.context).
	// Go contexts are struct values; identity is checked on the nested object.
	assert.NotSame(t, initialState1.Context.Foo, initialState2.Context.Foo)

	assert.Equal(t, ctx{Foo: &bar{Bar: "baz"}}, initialState1.Context)

	assert.Equal(t, ctx{Foo: &bar{Bar: "baz"}}, initialState2.Context)
}

// machine1ResolveMachine mirrors `resolveMachine` declared in the
// 'machine.resolveStateValue()' describe (machine.test.ts lines 247-286).
func machine1ResolveMachine() *xs.StateMachine[any] {
	return xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "resolve",
		Initial: "foo",
		States: xs.States{
			{
				Key:     "foo",
				Initial: "one",
				States: xs.States{
					{
						Key:  "one",
						Type: xs.Parallel,
						States: xs.States{
							{Key: "a", Initial: "aa", States: xs.States{{Key: "aa"}}},
							{Key: "b", Initial: "bb", States: xs.States{{Key: "bb"}}},
						},
						On: map[string]xs.Transitions{"TO_TWO": {{Target: "two"}}},
					},
					{
						Key: "two",
						On:  map[string]xs.Transitions{"TO_ONE": {{Target: "one"}}},
					},
				},
				On: map[string]xs.Transitions{"TO_BAR": {{Target: "bar"}}},
			},
			{
				Key: "bar",
				On:  map[string]xs.Transitions{"TO_FOO": {{Target: "foo"}}},
			},
		},
	})
}

// JS: machine > machine.resolveStateValue() > should resolve the state value
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/machine.test.ts#L283
func TestMachine_MachineResolveStateValue_ShouldResolveTheStateValue(t *testing.T) {
	resolveMachine := machine1ResolveMachine()

	resolvedState := resolveMachine.ResolveState(xs.ResolveStateConfig[any]{Value: "foo"})

	assert.Equal(t, map[string]any{
		"foo": map[string]any{"one": map[string]any{"a": "aa", "b": "bb"}},
	}, resolvedState.Value)
}

// JS: machine > machine.resolveStateValue() > should resolve `status: done`
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/machine.test.ts#L291
func TestMachine_MachineResolveStateValue_ShouldResolveStatusDone(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "foo",
		States: xs.States{
			{Key: "foo", On: map[string]xs.Transitions{"NEXT": {{Target: "bar"}}}},
			{Key: "bar", Type: xs.Final},
		},
	})

	resolvedState := machine.ResolveState(xs.ResolveStateConfig[any]{Value: "bar"})

	assert.Equal(t, xs.StatusDone, resolvedState.Status)
}

// JS: machine > initial state > should follow always transition
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/machine.test.ts#L311
func TestMachine_InitialState_ShouldFollowAlwaysTransition(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", Always: xs.Transitions{{Target: "b"}}},
			{Key: "b"},
		},
	})

	assert.Equal(t, "b", xs.CreateActor(machine).GetSnapshot().Value)
}

// JS: machine > versioning > should allow a version to be specified
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/machine.test.ts#L327
func TestMachine_Versioning_ShouldAllowAVersionToBeSpecified(t *testing.T) {
	versionMachine := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "version",
		Version: "1.0.4",
		States:  xs.States{},
	})

	assert.Equal(t, "1.0.4", versionMachine.Version)
}

// JS: machine > id > should represent the ID
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/machine.test.ts#L339
func TestMachine_ID_ShouldRepresentTheID(t *testing.T) {
	idMachine := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "some-id",
		Initial: "idle",
		States:  xs.States{{Key: "idle"}},
	})

	assert.Equal(t, "some-id", idMachine.ID)
}

// JS: machine > id > should represent the ID (state node)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/machine.test.ts#L349
func TestMachine_ID_ShouldRepresentTheIDStateNode(t *testing.T) {
	idMachine := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "some-id",
		Initial: "idle",
		States:  xs.States{{Key: "idle", ID: "idle"}},
	})

	assert.Equal(t, "idle", idMachine.States["idle"].ID)
}

// JS: machine > id > should use the key as the ID if no ID is provided (state node)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/machine.test.ts#L363
func TestMachine_ID_ShouldUseTheKeyAsTheIDIfNoIDIsProvidedStateNode(t *testing.T) {
	noStateNodeIDMachine := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "some-id",
		Initial: "idle",
		States:  xs.States{{Key: "idle"}},
	})

	assert.Equal(t, "some-id.idle", noStateNodeIDMachine.States["idle"].ID)
}

// JS: machine > combinatorial machines > should support combinatorial machines (single-state)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/machine.test.ts#L375
func TestMachine_CombinatorialMachines_ShouldSupportCombinatorialMachinesSingleState(t *testing.T) {
	type ctx struct{ Value int }

	testMachine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Context: ctx{Value: 42},
		On: map[string]xs.Transitions{
			"INC": {{Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
				c := a.Context
				c.Value = a.Context.Value + 1
				return c
			})}}},
		},
	})

	actorRef := xs.CreateActor(testMachine)
	assert.Equal(t, map[string]any{}, actorRef.GetSnapshot().Value)

	actorRef.Start()
	actorRef.Send(xs.Ev("INC"))

	assert.Equal(t, 43, actorRef.GetSnapshot().Context.Value)
}

// JS: machine > should pass through schemas
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/machine.test.ts#L396
func TestMachine_ShouldPassThroughSchemas(t *testing.T) {
	machine := xs.NewSetup[any](xs.Implementations{}).
		WithSchemas(map[string]any{
			"context": map[string]any{"count": map[string]any{"type": "number"}},
		}).
		CreateMachine(xs.MachineConfig[any]{})

	assert.Equal(t, map[string]any{
		"context": map[string]any{"count": map[string]any{"type": "number"}},
	}, machine.Schemas())
}

// JS: StateNode > should list transitions
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/machine.test.ts#L410
func TestMachine_StateNode_ShouldListTransitions(t *testing.T) {
	lightMachine := machine1LightMachine()
	greenNode := lightMachine.States["green"]

	transitions := greenNode.Transitions()

	keys := make([]string, 0, len(transitions))
	for k := range transitions {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	// JS order is ['TIMER', 'POWER_OUTAGE', 'FORBIDDEN_EVENT'] (Map insertion
	// order); Go maps are unordered, so the expectation is sorted.
	assert.Equal(t, []string{"FORBIDDEN_EVENT", "POWER_OUTAGE", "TIMER"}, keys)
}
