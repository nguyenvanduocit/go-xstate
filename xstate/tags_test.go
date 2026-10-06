package xstate_test

import (
	"encoding/json"
	"testing"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// JS: tags > supports tagging states
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/tags.test.ts#L4
func TestTags_SupportsTaggingStates(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "green",
		States: xs.States{
			{Key: "green", Tags: xs.Tags{"go"}, On: map[string]xs.Transitions{
				"TIMER": {{Target: "yellow"}},
			}},
			{Key: "yellow", Tags: xs.Tags{"go"}, On: map[string]xs.Transitions{
				"TIMER": {{Target: "red"}},
			}},
			{Key: "red", Tags: xs.Tags{"stop"}},
		},
	})

	actorRef := xs.CreateActor(machine).Start()
	assert.True(t, actorRef.GetSnapshot().HasTag("go"))
	actorRef.Send(xs.Ev("TIMER"))
	assert.True(t, actorRef.GetSnapshot().HasTag("go"))
	actorRef.Send(xs.Ev("TIMER"))
	assert.False(t, actorRef.GetSnapshot().HasTag("go"))
}

// JS: tags > supports tags in compound states
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/tags.test.ts#L34
func TestTags_SupportsTagsInCompoundStates(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "red",
		States: xs.States{
			{Key: "green", Tags: xs.Tags{"go"}},
			{Key: "yellow"},
			{
				Key:     "red",
				Tags:    xs.Tags{"stop"},
				Initial: "walk",
				States: xs.States{
					{Key: "walk", Tags: xs.Tags{"crosswalkLight"}},
					{Key: "wait", Tags: xs.Tags{"crosswalkLight"}},
				},
			},
		},
	})

	actorRef := xs.CreateActor(machine).Start()
	initialState := actorRef.GetSnapshot()

	assert.False(t, initialState.HasTag("go"))
	assert.True(t, initialState.HasTag("stop"))
	assert.True(t, initialState.HasTag("crosswalkLight"))
}

// JS: tags > supports tags in parallel states
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/tags.test.ts#L65
func TestTags_SupportsTagsInParallelStates(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Type: xs.Parallel,
		States: xs.States{
			{
				Key:     "foo",
				Initial: "active",
				States: xs.States{
					{Key: "active", Tags: xs.Tags{"yes"}},
					{Key: "inactive", Tags: xs.Tags{"no"}},
				},
			},
			{
				Key:     "bar",
				Initial: "active",
				States: xs.States{
					{Key: "active", Tags: xs.Tags{"yes"}, On: map[string]xs.Transitions{
						"DEACTIVATE": {{Target: "inactive"}},
					}},
					{Key: "inactive", Tags: xs.Tags{"no"}},
				},
			},
		},
	})

	actorRef := xs.CreateActor(machine).Start()

	// JS compares Sets (order-insensitive, unique): Go Tags is a sorted slice.
	assert.ElementsMatch(t, []string{"yes"}, actorRef.GetSnapshot().Tags)
	actorRef.Send(xs.Ev("DEACTIVATE"))
	assert.ElementsMatch(t, []string{"yes", "no"}, actorRef.GetSnapshot().Tags)
}

// JS: tags > sets tags correctly after not selecting any transition
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/tags.test.ts#L104
func TestTags_SetsTagsCorrectlyAfterNotSelectingAnyTransition(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{Key: "a", Tags: xs.Tags{"myTag"}},
		},
	})

	actorRef := xs.CreateActor(machine).Start()
	actorRef.Send(xs.Ev("UNMATCHED"))
	assert.True(t, actorRef.GetSnapshot().HasTag("myTag"))
}

// JS: tags > tags can be single (not array)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/tags.test.ts#L121
func TestTags_TagsCanBeSingleNotArray(t *testing.T) {
	// JS `tags: 'go'` (single string) — the Go contract only has xs.Tags (a
	// slice), so the single tag is a one-element slice.
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "green",
		States: xs.States{
			{Key: "green", Tags: xs.Tags{"go"}},
		},
	})

	assert.True(t, xs.CreateActor(machine).GetSnapshot().HasTag("go"))
}

// JS: tags > stringifies to an array
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/tags.test.ts#L134
func TestTags_StringifiesToAnArray(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "green",
		States: xs.States{
			{Key: "green", Tags: xs.Tags{"go", "light"}},
		},
	})

	jsonState := xs.CreateActor(machine).GetSnapshot().ToJSON()

	// toEqual(['go', 'light']): an ordered array; compared through JSON so the
	// concrete Go slice type ([]string or []any) does not matter.
	raw, err := json.Marshal(jsonState["tags"])
	require.NoError(t, err)
	assert.JSONEq(t, `["go","light"]`, string(raw))
}
