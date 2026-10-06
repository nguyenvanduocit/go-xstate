package xstate_test

import (
	"encoding/json"
	"sort"
	"testing"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// json1RoundTrip mirrors JSON.parse(JSON.stringify(v)).
func json1RoundTrip(t *testing.T, v any) any {
	t.Helper()
	raw, err := json.Marshal(v)
	require.NoError(t, err)
	var out any
	require.NoError(t, json.Unmarshal(raw, &out))
	return out
}

// JS: json > should serialize the machine
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/json.test.ts#L10
func TestJSON_ShouldSerializeTheMachine(t *testing.T) {
	// interface Context { [key: string]: any } -> dynamic keys.
	type ctx = map[string]any

	machine := xs.CreateMachine(xs.MachineConfig[ctx]{
		Initial: "foo",
		Version: "1.0.0",
		Context: ctx{
			"number": 0,
			"string": "hello",
		},
		Invoke: []xs.InvokeConfig{{ID: "invokeId", Src: "invokeSrc"}},
		States: xs.States{
			{
				Key:    "testActions",
				Invoke: []xs.InvokeConfig{{ID: "invokeId", Src: "invokeSrc"}},
				Entry: xs.Actions{
					xs.ActionRef{Type: "stringActionType"},
					xs.ActionRef{Type: "objectActionType"},
					// { type: 'objectActionTypeWithExec', exec: () => true, other: 'any' }:
					// extra `exec`/`other` properties have no field on ActionRef.
					xs.ActionRef{Type: "objectActionTypeWithExec"},
					// function actionFunction() { return true; }
					xs.ActionFunc(func(a xs.ActionArgs[ctx]) {}),
					// assign({ number: 10, string: 'test', evalNumber: () => 42 })
					xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
						next := ctx{}
						for k, v := range a.Context {
							next[k] = v
						}
						next["number"] = 10
						next["string"] = "test"
						next["evalNumber"] = 42
						return next
					}),
					// assign((ctx) => ({ ...ctx })): spreads the assign args object.
					xs.Assign(func(a xs.AssignArgs[ctx]) ctx {
						return ctx{
							"context": a.Context,
							"event":   a.Event,
							"self":    a.Self,
							"system":  a.System,
							"spawn":   a.Spawn,
						}
					}),
				},
				On: map[string]xs.Transitions{
					"TO_FOO": {{
						Targets: []string{"foo", "bar"},
						Guard: xs.GuardFunc(func(a xs.GuardArgs[ctx]) bool {
							s, ok := a.Context["string"].(string)
							return ok && s != ""
						}),
					}},
				},
				After: map[string]xs.Transitions{
					"1000": {{Target: "bar"}},
				},
			},
			{Key: "foo"},
			{Key: "bar"},
			{Key: "testHistory", Type: xs.History, History: xs.Deep},
			{Key: "testFinal", Type: xs.Final, Output: map[string]any{"something": "else"}},
			{
				Key:  "testParallel",
				Type: xs.Parallel,
				States: xs.States{
					{Key: "one", Initial: "inactive", States: xs.States{{Key: "inactive"}}},
					{Key: "two", Initial: "inactive", States: xs.States{{Key: "inactive"}}},
				},
			},
		},
		Output: map[string]any{"result": 42},
	})

	js := json1RoundTrip(t, machine.Definition())

	errs := xs.ValidateMachineSchema(js)

	assert.Nil(t, errs)
}

// JS: json > should detect an invalid machine
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/json.test.ts#L106
func TestJSON_ShouldDetectAnInvalidMachine(t *testing.T) {
	invalidMachineConfig := map[string]any{
		"id":     "something",
		"key":    "something",
		"type":   "invalid type",
		"states": map[string]any{},
	}

	errs := xs.ValidateMachineSchema(invalidMachineConfig)
	assert.NotNil(t, errs)
}

// JS: json > should not double-serialize invoke transitions
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/json.test.ts#L118
func TestJSON_ShouldNotDoubleSerializeInvokeTransitions(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "active",
		States: xs.States{
			{
				Key: "active",
				ID:  "active",
				Invoke: []xs.InvokeConfig{{
					Src:     "someSrc",
					OnDone:  xs.Transitions{{Target: "foo"}},
					OnError: xs.Transitions{{Target: "bar"}},
				}},
				On: map[string]xs.Transitions{
					"EVENT": {{Target: "foo"}},
				},
			},
			{Key: "foo"},
			{Key: "bar"},
		},
	})

	// JSON.stringify(machine) uses machine.toJSON(); JSON.parse revives a plain object.
	machineObject, ok := json1RoundTrip(t, machine.ToJSON()).(map[string]any)
	require.True(t, ok)

	revivedMachine := xs.CreateMachineFromJSON(machineObject)

	// [...revivedMachine.states.active.transitions.values()].flat()
	// stateNode.on is keyed by descriptor; Go map order is sorted by descriptor
	// (matches the JS insertion order here: EVENT, done, error).
	flatten := func(n *xs.StateNode) []*xs.TransitionDefinition {
		on := n.On()
		keys := make([]string, 0, len(on))
		for k := range on {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var out []*xs.TransitionDefinition
		for _, k := range keys {
			out = append(out, on[k]...)
		}
		return out
	}

	active := revivedMachine.States["active"]
	require.NotNil(t, active)
	transitions := flatten(active)

	type expectedTransition struct {
		eventType string
		target    string
	}
	expected := []expectedTransition{
		{eventType: "EVENT", target: "#(machine).foo"},
		{eventType: "xstate.done.actor.0.active", target: "#(machine).foo"},
		{eventType: "xstate.error.actor.0.active", target: "#(machine).bar"},
	}
	require.Len(t, transitions, len(expected))
	for i, exp := range expected {
		tr := transitions[i]
		assert.Empty(t, tr.Actions, "actions[%d]", i)
		assert.Equal(t, exp.eventType, tr.EventType, "eventType[%d]", i)
		assert.Nil(t, tr.Guard, "guard[%d]", i)
		assert.False(t, tr.Reenter, "reenter[%d]", i)
		require.NotNil(t, tr.Source, "source[%d]", i)
		assert.Equal(t, "active", tr.Source.ID, "source[%d]", i)
		require.Len(t, tr.Target, 1, "target[%d]", i)
		assert.Equal(t, exp.target, "#"+tr.Target[0].ID, "target[%d]", i)

		// The inline snapshot prints the transition's toJSON() form.
		j, ok := json1RoundTrip(t, tr.ToJSON()).(map[string]any)
		require.True(t, ok)
		assert.Equal(t, []any{}, j["actions"], "toJSON actions[%d]", i)
		assert.Equal(t, exp.eventType, j["eventType"], "toJSON eventType[%d]", i)
		assert.Nil(t, j["guard"], "toJSON guard[%d]", i)
		assert.Equal(t, false, j["reenter"], "toJSON reenter[%d]", i)
		assert.Equal(t, "#active", j["source"], "toJSON source[%d]", i)
		assert.Equal(t, []any{exp.target}, j["target"], "toJSON target[%d]", i)
	}

	// 1. onDone
	// 2. onError
	// 3. EVENT
	assert.Len(t, flatten(revivedMachine.GetStateNodeByID("active")), 3)
}
