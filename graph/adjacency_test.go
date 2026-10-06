package graph_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nguyenvanduocit/go-xstate/graph"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// JS: adjacency maps > model generates an adjacency map (converted to an array)
//
// JS iterates each state's events in `on` key insertion order; the Go API
// returns own events sorted, so the expected order is re-derived with sorted
// event descriptors (see docs/porting/manifest/graph.md).
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/adjacency.test.ts#L5
func TestAdjacency_ModelGeneratesAnAdjacencyMapConvertedToAnArray(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "standing",
		States: xs.States{
			{Key: "standing", On: map[string]xs.Transitions{
				"left":  {{Target: "walking"}},
				"right": {{Target: "walking"}},
				"down":  {{Target: "crouching"}},
				"up":    {{Target: "jumping"}},
			}},
			{Key: "walking", On: map[string]xs.Transitions{
				"up":   {{Target: "jumping"}},
				"stop": {{Target: "standing"}},
			}},
			{Key: "jumping", On: map[string]xs.Transitions{
				"land": {{Target: "standing"}},
			}},
			{Key: "crouching", On: map[string]xs.Transitions{
				"release_down": {{Target: "standing"}},
			}},
		},
	})
	model := graph.CreateTestModel(machine)

	var got []string
	for _, e := range graph.AdjacencyMapToArray(model.GetAdjacencyMap()) {
		got = append(got, fmt.Sprintf("Given Mario is %v, when %s, then %v", e.State.Value, e.Event.EventType(), e.NextState.Value))
	}

	assert.Equal(t, []string{
		"Given Mario is standing, when down, then crouching",
		"Given Mario is standing, when left, then walking",
		"Given Mario is standing, when right, then walking",
		"Given Mario is standing, when up, then jumping",
		"Given Mario is crouching, when release_down, then standing",
		"Given Mario is walking, when stop, then standing",
		"Given Mario is walking, when up, then jumping",
		"Given Mario is walking, when stop, then standing",
		"Given Mario is walking, when up, then jumping",
		"Given Mario is jumping, when land, then standing",
		"Given Mario is standing, when down, then crouching",
		"Given Mario is standing, when left, then walking",
		"Given Mario is standing, when right, then walking",
		"Given Mario is standing, when up, then jumping",
		"Given Mario is standing, when down, then crouching",
		"Given Mario is standing, when left, then walking",
		"Given Mario is standing, when right, then walking",
		"Given Mario is standing, when up, then jumping",
		"Given Mario is jumping, when land, then standing",
		"Given Mario is standing, when down, then crouching",
		"Given Mario is standing, when left, then walking",
		"Given Mario is standing, when right, then walking",
		"Given Mario is standing, when up, then jumping",
	}, got)
}

// JS: adjacency maps > function generates an adjacency map (converted to an array)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/adjacency.test.ts#L71
func TestAdjacency_FunctionGeneratesAnAdjacencyMapConvertedToAnArray(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "green",
		States: xs.States{
			{Key: "green", On: map[string]xs.Transitions{"TIMER": {{Target: "yellow"}}}},
			{Key: "yellow", On: map[string]xs.Transitions{"TIMER": {{Target: "red"}}}},
			{Key: "red", On: map[string]xs.Transitions{"TIMER": {{Target: "green"}}}},
		},
	})

	arr := graph.AdjacencyMapToArray(graph.CreateTestModel(machine).GetAdjacencyMap())

	type entry struct {
		Event     string
		NextState any
		State     any
	}
	var got []entry
	for _, x := range arr {
		got = append(got, entry{Event: x.Event.EventType(), NextState: x.NextState.Value, State: x.State.Value})
	}

	assert.Equal(t, []entry{
		{Event: "TIMER", NextState: "yellow", State: "green"},
		{Event: "TIMER", NextState: "red", State: "yellow"},
		{Event: "TIMER", NextState: "green", State: "red"},
		{Event: "TIMER", NextState: "yellow", State: "green"},
	}, got)
}
