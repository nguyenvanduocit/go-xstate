package xstate_test

import (
	"testing"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/assert"
)

// JS: Initial states > should return the correct initial state
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/initial.test.ts#L4
func TestInitial_ShouldReturnTheCorrectInitialState(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "a",
		States: xs.States{
			{
				Key:     "a",
				Initial: "b",
				States: xs.States{
					{
						Key:     "b",
						Initial: "c",
						States: xs.States{
							{Key: "c"},
						},
					},
				},
			},
			{Key: "leaf"},
		},
	})

	assert.Equal(t, map[string]any{
		"a": map[string]any{"b": "c"},
	}, xs.CreateActor(machine).GetSnapshot().Value)
}

// JS: Initial states > should return the correct initial state (parallel)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/initial.test.ts#L27
func TestInitial_ShouldReturnTheCorrectInitialStateParallel(t *testing.T) {
	// region mirrors the identical `foo` / `bar` state configs written inline in JS.
	region := func(key string) xs.StateConfig {
		return xs.StateConfig{
			Key:     key,
			Initial: "a",
			States: xs.States{
				{
					Key:     "a",
					Initial: "b",
					States: xs.States{
						{
							Key:     "b",
							Initial: "c",
							States: xs.States{
								{Key: "c"},
							},
						},
					},
				},
				{Key: "leaf"},
			},
		}
	}

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Type: xs.Parallel,
		States: xs.States{
			region("foo"),
			region("bar"),
		},
	})

	assert.Equal(t, map[string]any{
		"foo": map[string]any{"a": map[string]any{"b": "c"}},
		"bar": map[string]any{"a": map[string]any{"b": "c"}},
	}, xs.CreateActor(machine).GetSnapshot().Value)
}

// JS: Initial states > should return the correct initial state (deep parallel)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/initial.test.ts#L73
func TestInitial_ShouldReturnTheCorrectInitialStateDeepParallel(t *testing.T) {
	// region mirrors the identical `foo` / `bar` state configs written inline in JS.
	region := func(key string) xs.StateConfig {
		return xs.StateConfig{
			Key:     key,
			Initial: "a",
			States: xs.States{
				{
					Key:     "a",
					Initial: "b",
					States: xs.States{
						{
							Key:     "b",
							Initial: "c",
							States: xs.States{
								{Key: "c"},
							},
						},
					},
				},
				{Key: "leaf"},
			},
		}
	}

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "one",
		States: xs.States{
			{
				Key:  "one",
				Type: xs.Parallel,
				States: xs.States{
					region("foo"),
					region("bar"),
				},
			},
			{
				Key:  "two",
				Type: xs.Parallel,
				States: xs.States{
					region("foo"),
					region("bar"),
				},
			},
		},
	})

	assert.Equal(t, map[string]any{
		"one": map[string]any{
			"foo": map[string]any{"a": map[string]any{"b": "c"}},
			"bar": map[string]any{"a": map[string]any{"b": "c"}},
		},
	}, xs.CreateActor(machine).GetSnapshot().Value)
}
