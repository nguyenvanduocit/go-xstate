package xstate_test

import (
	"testing"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/assert"
)

type order1KeyOrder struct {
	Key   string
	Order int
}

// JS: document order > should specify the correct document order for each state node
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/order.test.ts#L4
func TestOrder_DocumentOrder_ShouldSpecifyTheCorrectDocumentOrderForEachStateNode(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "order",
		Initial: "one",
		States: xs.States{
			{
				Key:     "one",
				Initial: "two",
				States: xs.States{
					{Key: "two"},
					{
						Key:     "three",
						Initial: "four",
						States: xs.States{
							{Key: "four"},
							{
								Key:     "five",
								Initial: "six",
								States: xs.States{
									{Key: "six"},
								},
							},
						},
					},
				},
			},
			{
				Key:  "seven",
				Type: xs.Parallel,
				States: xs.States{
					{
						Key:     "eight",
						Initial: "nine",
						States: xs.States{
							{Key: "nine"},
							{
								Key:     "ten",
								Initial: "eleven",
								States: xs.States{
									{Key: "eleven"},
									{Key: "twelve"},
								},
							},
						},
					},
					{
						Key:  "thirteen",
						Type: xs.Parallel,
						States: xs.States{
							{Key: "fourteen"},
							{Key: "fifteen"},
						},
					},
				},
			},
		},
	})

	// JS iterates Object.keys(node.states) (insertion/document order);
	// Go's StateNode.States is an unordered map, so ChildStates() is used.
	var dfs func(node *xs.StateNode) []*xs.StateNode
	dfs = func(node *xs.StateNode) []*xs.StateNode {
		out := []*xs.StateNode{node}
		for _, child := range node.ChildStates() {
			out = append(out, dfs(child)...)
		}
		return out
	}

	var allStateNodeOrders []order1KeyOrder
	for _, sn := range dfs(machine.Root) {
		allStateNodeOrders = append(allStateNodeOrders, order1KeyOrder{sn.Key, sn.Order})
	}

	assert.Equal(t, []order1KeyOrder{
		{"order", 0},
		{"one", 1},
		{"two", 2},
		{"three", 3},
		{"four", 4},
		{"five", 5},
		{"six", 6},
		{"seven", 7},
		{"eight", 8},
		{"nine", 9},
		{"ten", 10},
		{"eleven", 11},
		{"twelve", 12},
		{"thirteen", 13},
		{"fourteen", 14},
		{"fifteen", 15},
	}, allStateNodeOrders)
}
