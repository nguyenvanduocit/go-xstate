package xstate_test

import (
	"testing"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/assert"
)

// resolve1FlatParallelMachine mirrors the top-level `flatParallelMachine`
// (from parallel/test3.scxml) in resolve.test.ts.
func resolve1FlatParallelMachine() *xs.StateMachine[any] {
	return xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "fp",
		Initial: "p1",
		States: xs.States{
			{
				Key:  "p1",
				Type: xs.Parallel,
				States: xs.States{
					{
						Key:     "s1",
						Initial: "p2",
						States: xs.States{
							{
								Key:  "p2",
								Type: xs.Parallel,
								States: xs.States{
									{
										Key:     "s3",
										Initial: "s3.1",
										States: xs.States{
											{Key: "s3.1"},
											{Key: "s3.2"},
										},
									},
									{Key: "s4"},
								},
							},
							{
								Key:  "p3",
								Type: xs.Parallel,
								States: xs.States{
									{Key: "s5"},
									{Key: "s6"},
								},
							},
						},
					},
					{
						Key:     "s2",
						Initial: "p4",
						States: xs.States{
							{
								Key:  "p4",
								Type: xs.Parallel,
								States: xs.States{
									{Key: "s7"},
									{Key: "s8"},
								},
							},
							{
								Key:  "p5",
								Type: xs.Parallel,
								States: xs.States{
									{Key: "s9"},
									{Key: "s10"},
								},
							},
						},
					},
				},
			},
		},
	})
}

// JS: resolve() > should resolve parallel states with flat child states
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/resolve.test.ts#L62
func TestResolve_ShouldResolveParallelStatesWithFlatChildStates(t *testing.T) {
	flatParallelMachine := resolve1FlatParallelMachine()

	unresolvedStateValue := map[string]any{
		"p1": map[string]any{
			"s1": map[string]any{"p2": "s4"},
			"s2": map[string]any{"p4": "s8"},
		},
	}

	resolvedStateValue := xs.ResolveStateValue(flatParallelMachine.Root, unresolvedStateValue)

	assert.Equal(t, map[string]any{
		"p1": map[string]any{
			"s1": map[string]any{
				"p2": map[string]any{"s3": "s3.1", "s4": map[string]any{}},
			},
			"s2": map[string]any{
				"p4": map[string]any{"s7": map[string]any{}, "s8": map[string]any{}},
			},
		},
	}, resolvedStateValue)
}
