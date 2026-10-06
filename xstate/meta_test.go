package xstate_test

import (
	"encoding/json"
	"testing"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// meta1PedestrianStates mirrors the describe-level `pedestrianStates` object
// (meta.test.ts L4-29) that is spread into the `red` state.
func meta1PedestrianStates() xs.StateConfig {
	return xs.StateConfig{
		Initial: "walk",
		States: xs.States{
			{
				Key:   "walk",
				Meta:  map[string]any{"walkData": "walk data"},
				On:    map[string]xs.Transitions{"PED_COUNTDOWN": {{Target: "wait"}}},
				Entry: xs.Actions{xs.ActionRef{Type: "enter_walk"}},
				Exit:  xs.Actions{xs.ActionRef{Type: "exit_walk"}},
			},
			{
				Key:   "wait",
				Meta:  map[string]any{"waitData": "wait data"},
				On:    map[string]xs.Transitions{"PED_COUNTDOWN": {{Target: "stop"}}},
				Entry: xs.Actions{xs.ActionRef{Type: "enter_wait"}},
				Exit:  xs.Actions{xs.ActionRef{Type: "exit_wait"}},
			},
			{
				Key:   "stop",
				Meta:  map[string]any{"stopData": "stop data"},
				Entry: xs.Actions{xs.ActionRef{Type: "enter_stop"}},
				Exit:  xs.Actions{xs.ActionRef{Type: "exit_stop"}},
			},
		},
	}
}

// meta1LightMachine mirrors the describe-level `lightMachine` (meta.test.ts L31-73).
func meta1LightMachine() *xs.StateMachine[any] {
	pedestrian := meta1PedestrianStates()
	return xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "light",
		Initial: "green",
		States: xs.States{
			{
				Key:  "green",
				Meta: []any{"green", "array", "data"},
				On: map[string]xs.Transitions{
					"TIMER":        {{Target: "yellow"}},
					"POWER_OUTAGE": {{Target: "red"}},
					"NOTHING":      {{Target: "green"}},
				},
				Entry: xs.Actions{xs.ActionRef{Type: "enter_green"}},
				Exit:  xs.Actions{xs.ActionRef{Type: "exit_green"}},
			},
			{
				Key:  "yellow",
				Meta: map[string]any{"yellowData": "yellow data"},
				On: map[string]xs.Transitions{
					"TIMER":        {{Target: "red"}},
					"POWER_OUTAGE": {{Target: "red"}},
				},
				Entry: xs.Actions{xs.ActionRef{Type: "enter_yellow"}},
				Exit:  xs.Actions{xs.ActionRef{Type: "exit_yellow"}},
			},
			{
				Key: "red",
				Meta: map[string]any{
					"redData": map[string]any{
						"nested": map[string]any{
							"red":   "data",
							"array": []any{1, 2, 3},
						},
					},
				},
				On: map[string]xs.Transitions{
					"TIMER":        {{Target: "green"}},
					"POWER_OUTAGE": {{Target: "red"}},
					"NOTHING":      {{Target: "red"}},
				},
				Entry: xs.Actions{xs.ActionRef{Type: "enter_red"}},
				Exit:  xs.Actions{xs.ActionRef{Type: "exit_red"}},
				// ...pedestrianStates
				Initial: pedestrian.Initial,
				States:  pedestrian.States,
			},
		},
	})
}

// JS: state meta data > states should aggregate meta data
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/meta.test.ts#L75
func TestMeta_StateMetaData_StatesShouldAggregateMetaData(t *testing.T) {
	actorRef := xs.CreateActor(meta1LightMachine()).Start()
	actorRef.Send(xs.Ev("TIMER"))
	yellowState := actorRef.GetSnapshot()

	assert.Equal(t, map[string]any{
		"light.yellow": map[string]any{
			"yellowData": "yellow data",
		},
	}, yellowState.GetMeta())
	assert.NotContains(t, yellowState.GetMeta(), "light.green")
	assert.NotContains(t, yellowState.GetMeta(), "light")
}

// JS: state meta data > states should aggregate meta data (deep)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/meta.test.ts#L89
func TestMeta_StateMetaData_StatesShouldAggregateMetaDataDeep(t *testing.T) {
	actorRef := xs.CreateActor(meta1LightMachine()).Start()
	actorRef.Send(xs.Ev("TIMER"))
	actorRef.Send(xs.Ev("TIMER"))
	assert.Equal(t, map[string]any{
		"light.red": map[string]any{
			"redData": map[string]any{
				"nested": map[string]any{
					"array": []any{1, 2, 3},
					"red":   "data",
				},
			},
		},
		"light.red.walk": map[string]any{
			"walkData": "walk data",
		},
	}, actorRef.GetSnapshot().GetMeta())
}

// JS: state meta data > services started from a persisted state should calculate meta data
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/meta.test.ts#L109
func TestMeta_StateMetaData_ServicesStartedFromPersistedStateShouldCalculateMetaData(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "test",
		Initial: "first",
		States: xs.States{
			{Key: "first", Meta: map[string]any{"name": "first state"}},
			{Key: "second", Meta: map[string]any{"name": "second state"}},
		},
	})

	actor := xs.CreateActor(machine,
		xs.WithSnapshot(machine.ResolveState(xs.ResolveStateConfig[any]{Value: "second"})),
	)
	actor.Start()

	assert.Equal(t, map[string]any{
		"test.second": map[string]any{
			"name": "second state",
		},
	}, actor.GetSnapshot().GetMeta())
}

// JS: state meta data > meta keys are strongly-typed
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/meta.test.ts#L139
func TestMeta_StateMetaData_MetaKeysAreStronglyTyped(t *testing.T) {
	t.Skip("N/A: type-level only — `satisfies` / @ts-expect-error checks that getMeta() keys are the state IDs and values have the setup({ types: { meta } }) type; no runtime expectations")
}

// JS: state meta data > TS should error with unexpected meta property
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/meta.test.ts#L186
func TestMeta_StateMetaData_TSShouldErrorWithUnexpectedMetaProperty(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error that a state meta object rejects a property not in setup({ types: { meta } }); no runtime expectations")
}

// JS: state meta data > TS should error with wrong meta value type
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/meta.test.ts#L211
func TestMeta_StateMetaData_TSShouldErrorWithWrongMetaValueType(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error that a state meta value of the wrong type is rejected; no runtime expectations")
}

// JS: state meta data > should allow states to omit meta
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/meta.test.ts#L236
func TestMeta_StateMetaData_ShouldAllowStatesToOmitMeta(t *testing.T) {
	t.Skip("N/A: type-level only — checks that a state without meta type-checks when setup({ types: { meta } }) is set; no runtime expectations")
}

// JS: state meta data > TS should error with unexpected transition meta property
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/meta.test.ts#L256
func TestMeta_StateMetaData_TSShouldErrorWithUnexpectedTransitionMetaProperty(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error that a transition meta object rejects a property not in setup({ types: { meta } }); no runtime expectations")
}

// JS: state meta data > TS should error with wrong transition meta value type
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/meta.test.ts#L280
func TestMeta_StateMetaData_TSShouldErrorWithWrongTransitionMetaValueType(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error that a transition meta value of the wrong type is rejected; no runtime expectations")
}

// JS: state meta data > should support typing meta properties (no ts-expected errors)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/meta.test.ts#L304
func TestMeta_StateMetaData_ShouldSupportTypingMetaPropertiesNoTSExpectedErrors(t *testing.T) {
	t.Skip("N/A: type-level only — `satisfies` checks that getMeta()['(machine)'] has the setup({ types: { meta } }) type; no runtime expectations")
}

// JS: state meta data > should strongly type the state IDs in snapshot.getMeta()
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/meta.test.ts#L344
func TestMeta_StateMetaData_ShouldStronglyTypeTheStateIDsInGetMeta(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error checks on which state-ID keys getMeta() accepts; no runtime expectations")
}

// JS: state meta data > should strongly type the state IDs in snapshot.getMeta() (no root ID)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/meta.test.ts#L381
func TestMeta_StateMetaData_ShouldStronglyTypeTheStateIDsInGetMetaNoRootID(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error checks on which '(machine)'-prefixed state-ID keys getMeta() accepts; no runtime expectations")
}

// JS: transition meta data > infers distinct metadata types with createMachine
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/meta.test.ts#L420
func TestMeta_TransitionMetaData_InfersDistinctMetadataTypesWithCreateMachine(t *testing.T) {
	t.Skip("N/A: type-level only — `satisfies` checks that types.meta and types.transitionMeta are inferred separately; no runtime expectations")
}

// JS: transition meta data > supports distinct state and transition meta types
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/meta.test.ts#L438
func TestMeta_TransitionMetaData_SupportsDistinctStateAndTransitionMetaTypes(t *testing.T) {
	t.Skip("N/A: type-level only — `satisfies` / @ts-expect-error checks that state meta and transition meta have distinct types; no runtime expectations")
}

// JS: transition meta data > rejects state and transition metadata in the wrong positions
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/meta.test.ts#L477
func TestMeta_TransitionMetaData_RejectsStateAndTransitionMetadataInTheWrongPositions(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error that transition meta is rejected on a state node and state meta on a transition; no runtime expectations")
}

// JS: transition meta data > keeps types.meta as the shared metadata type for compatibility
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/meta.test.ts#L500
func TestMeta_TransitionMetaData_KeepsTypesMetaAsTheSharedMetadataTypeForCompatibility(t *testing.T) {
	t.Skip("N/A: type-level only — `satisfies` checks that types.meta types both state and transition meta when transitionMeta is absent; no runtime expectations")
}

// JS: transition meta data > preserves transition meta on all transition definitions
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/meta.test.ts#L518
func TestMeta_TransitionMetaData_PreservesTransitionMetaOnAllTransitionDefinitions(t *testing.T) {
	// The `satisfies` / @ts-expect-error type assertions of the JS test are
	// type-level only; the runtime expectations are translated below.
	machine := xs.NewSetup[any](xs.Implementations{
		Actors: map[string]xs.ActorLogic{
			"child": xs.CreateMachine(xs.MachineConfig[any]{}),
		},
	}).CreateMachine(xs.WithMachineInitialMeta(xs.MachineConfig[any]{
		// initial: { target: 'idle', meta: { source: 'initial' } }
		Initial: "idle",
		States: xs.States{
			{
				Key:    "idle",
				Always: xs.Transitions{{Meta: map[string]any{"source": "always"}}},
				After: map[string]xs.Transitions{
					"100": {{Meta: map[string]any{"source": "after"}}},
				},
				Invoke: []xs.InvokeConfig{{
					Src:        "child",
					OnDone:     xs.Transitions{{Meta: map[string]any{"source": "invoke.done"}}},
					OnError:    xs.Transitions{{Meta: map[string]any{"source": "invoke.error"}}},
					OnSnapshot: xs.Transitions{{Meta: map[string]any{"source": "invoke.snapshot"}}},
				}},
			},
		},
	}, map[string]any{"source": "initial"}))

	assert.Equal(t, map[string]any{"source": "initial"}, machine.Root.Initial().Meta)

	definitionInitial, ok := machine.Definition()["initial"].(map[string]any)
	require.True(t, ok, "machine.definition.initial should be a JSON-like object")
	assert.Equal(t, map[string]any{"source": "initial"}, definitionInitial["meta"])

	// JSON.parse(JSON.stringify(machine)).initial.meta
	raw, err := json.Marshal(machine.ToJSON())
	require.NoError(t, err)
	var parsed map[string]any
	require.NoError(t, json.Unmarshal(raw, &parsed))
	parsedInitial, ok := parsed["initial"].(map[string]any)
	require.True(t, ok, "serialized machine should have an initial object")
	assert.Equal(t, map[string]any{"source": "initial"}, parsedInitial["meta"])

	idle := machine.States["idle"]
	assert.Equal(t, map[string]any{"source": "always"}, idle.Always()[0].Meta)
	assert.Equal(t, map[string]any{"source": "after"}, idle.After()[0].Meta)

	// [...machine.states.idle.transitions.values()].flat().map((t) => t.meta)
	var metas []any
	for _, transitions := range idle.On() {
		for _, transition := range transitions {
			metas = append(metas, transition.Meta)
		}
	}
	assert.Subset(t, metas, []any{
		map[string]any{"source": "invoke.done"},
		map[string]any{"source": "invoke.error"},
		map[string]any{"source": "invoke.snapshot"},
	})
}

// JS: transition meta data > TS should error with unexpected transition meta property
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/meta.test.ts#L597
func TestMeta_TransitionMetaData_TSShouldErrorWithUnexpectedTransitionMetaProperty(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error that a transition meta object rejects a property not in setup({ types: { meta } }); no runtime expectations")
}

// JS: transition meta data > TS should error with wrong transition meta value type
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/meta.test.ts#L621
func TestMeta_TransitionMetaData_TSShouldErrorWithWrongTransitionMetaValueType(t *testing.T) {
	t.Skip("N/A: type-level only — @ts-expect-error that a transition meta value of the wrong type is rejected; no runtime expectations")
}

// JS: state description > state node should have its description
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/meta.test.ts#L647
func TestMeta_StateDescription_StateNodeShouldHaveItsDescription(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "test",
		States: xs.States{
			{Key: "test", Description: "This is a test"},
		},
	})

	assert.Equal(t, "This is a test", machine.States["test"].Description)
}

// JS: transition description > state node should have its description
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/meta.test.ts#L662
func TestMeta_TransitionDescription_StateNodeShouldHaveItsDescription(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		On: map[string]xs.Transitions{
			"EVENT": {{Description: "This is a test"}},
		},
	})

	assert.Equal(t, "This is a test", machine.Root.On()["EVENT"][0].Description)
}
