package xstate_test

import (
	"testing"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/assert"
)

// multiple1Region builds a compound region `key` with `initial` and atomic children.
func multiple1Region(key, initial string, children ...string) xs.StateConfig {
	states := make(xs.States, 0, len(children))
	for _, c := range children {
		states = append(states, xs.StateConfig{Key: c})
	}
	return xs.StateConfig{Key: key, Initial: initial, States: states}
}

// multiple1Machine mirrors the shared `machine` declared in describe('multiple').
func multiple1Machine() *xs.StateMachine[any] {
	return xs.CreateMachine(xs.MachineConfig[any]{
		Initial: "simple",
		States: xs.States{
			{
				Key: "simple",
				On: map[string]xs.Transitions{
					"DEEP_M":                     {{Target: "para.K.M"}},
					"DEEP_CM":                    {{Targets: []string{"para.A.C", "para.K.M"}}},
					"DEEP_MR":                    {{Targets: []string{"para.K.M", "para.P.R"}}},
					"DEEP_CMR":                   {{Targets: []string{"para.A.C", "para.K.M", "para.P.R"}}},
					"BROKEN_SAME_REGION":         {{Targets: []string{"para.A.C", "para.A.B"}}},
					"BROKEN_DIFFERENT_REGIONS":   {{Targets: []string{"para.A.C", "para.K.M", "other"}}},
					"BROKEN_DIFFERENT_REGIONS_2": {{Targets: []string{"para.A.C", "para2.K2.M2"}}},
					"BROKEN_DIFFERENT_REGIONS_3": {{Targets: []string{"para2.K2.L2.L2A", "other"}}},
					"BROKEN_DIFFERENT_REGIONS_4": {{Targets: []string{"para2.K2.L2.L2A.L2C", "para2.K2.M2"}}},
					"INITIAL":                    {{Target: "para"}},
				},
			},
			multiple1Region("other", "X", "X"),
			{
				Key:  "para",
				Type: xs.Parallel,
				States: xs.States{
					multiple1Region("A", "B", "B", "C"),
					multiple1Region("K", "L", "L", "M"),
					multiple1Region("P", "Q", "Q", "R"),
				},
			},
			{
				Key:  "para2",
				Type: xs.Parallel,
				States: xs.States{
					multiple1Region("A2", "B2", "B2", "C2"),
					{
						Key:     "K2",
						Initial: "L2",
						States: xs.States{
							{
								Key:  "L2",
								Type: xs.Parallel,
								States: xs.States{
									multiple1Region("L2A", "L2B", "L2B", "L2C"),
									multiple1Region("L2K", "L2L", "L2L", "L2M"),
									multiple1Region("L2P", "L2Q", "L2Q", "L2R"),
								},
							},
							{
								Key:  "M2",
								Type: xs.Parallel,
								States: xs.States{
									multiple1Region("M2A", "M2B", "M2B", "M2C"),
									multiple1Region("M2K", "M2L", "M2L", "M2M"),
									multiple1Region("M2P", "M2Q", "M2Q", "M2R"),
								},
							},
						},
					},
					multiple1Region("P2", "Q2", "Q2", "R2"),
				},
			},
		},
	})
}

// JS: multiple > transitions to parallel states > should enter initial states of parallel states
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/multiple.test.ts#L139
func TestMultiple_TransitionsToParallelStates_ShouldEnterInitialStatesOfParallelStates(t *testing.T) {
	actorRef := xs.CreateActor(multiple1Machine()).Start()
	actorRef.Send(xs.Ev("INITIAL"))
	assert.Equal(t, map[string]any{
		"para": map[string]any{"A": "B", "K": "L", "P": "Q"},
	}, actorRef.GetSnapshot().Value)
}

// JS: multiple > transitions to parallel states > should enter specific states in one region
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/multiple.test.ts#L147
func TestMultiple_TransitionsToParallelStates_ShouldEnterSpecificStatesInOneRegion(t *testing.T) {
	actorRef := xs.CreateActor(multiple1Machine()).Start()
	actorRef.Send(xs.Ev("DEEP_M"))
	assert.Equal(t, map[string]any{
		"para": map[string]any{"A": "B", "K": "M", "P": "Q"},
	}, actorRef.GetSnapshot().Value)
}

// JS: multiple > transitions to parallel states > should enter specific states in all regions
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/multiple.test.ts#L155
func TestMultiple_TransitionsToParallelStates_ShouldEnterSpecificStatesInAllRegions(t *testing.T) {
	actorRef := xs.CreateActor(multiple1Machine()).Start()
	actorRef.Send(xs.Ev("DEEP_CMR"))
	assert.Equal(t, map[string]any{
		"para": map[string]any{"A": "C", "K": "M", "P": "R"},
	}, actorRef.GetSnapshot().Value)
}

// JS: multiple > transitions to parallel states > should enter specific states in some regions
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/multiple.test.ts#L163
func TestMultiple_TransitionsToParallelStates_ShouldEnterSpecificStatesInSomeRegions(t *testing.T) {
	actorRef := xs.CreateActor(multiple1Machine()).Start()
	actorRef.Send(xs.Ev("DEEP_MR"))
	assert.Equal(t, map[string]any{
		"para": map[string]any{"A": "B", "K": "M", "P": "R"},
	}, actorRef.GetSnapshot().Value)
}

// JS: multiple > transitions to parallel states > should reject two targets in the same region
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/multiple.test.ts#L171
func TestMultiple_TransitionsToParallelStates_ShouldRejectTwoTargetsInTheSameRegion(t *testing.T) {
	t.Skip("skipped in JS")
	actorRef := xs.CreateActor(multiple1Machine()).Start()
	assert.Panics(t, func() { actorRef.Send(xs.Ev("BROKEN_SAME_REGION")) })
}

// JS: multiple > transitions to parallel states > should reject targets inside and outside a region
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/multiple.test.ts#L176
func TestMultiple_TransitionsToParallelStates_ShouldRejectTargetsInsideAndOutsideARegion(t *testing.T) {
	t.Skip("skipped in JS")
	actorRef := xs.CreateActor(multiple1Machine()).Start()
	assert.Panics(t, func() { actorRef.Send(xs.Ev("BROKEN_DIFFERENT_REGIONS")) })
}

// JS: multiple > transitions to parallel states > should reject two targets in different regions
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/multiple.test.ts#L183
func TestMultiple_TransitionsToParallelStates_ShouldRejectTwoTargetsInDifferentRegions(t *testing.T) {
	t.Skip("skipped in JS")
	actorRef := xs.CreateActor(multiple1Machine()).Start()
	assert.Panics(t, func() { actorRef.Send(xs.Ev("BROKEN_DIFFERENT_REGIONS_2")) })
}

// JS: multiple > transitions to parallel states > should reject two targets in different regions at different levels
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/multiple.test.ts#L190
func TestMultiple_TransitionsToParallelStates_ShouldRejectTwoTargetsInDifferentRegionsAtDifferentLevels(t *testing.T) {
	t.Skip("skipped in JS")
	actorRef := xs.CreateActor(multiple1Machine()).Start()
	assert.Panics(t, func() { actorRef.Send(xs.Ev("BROKEN_DIFFERENT_REGIONS_3")) })
}

// JS: multiple > transitions to parallel states > should reject two deep targets in different regions at top level
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/multiple.test.ts#L197
func TestMultiple_TransitionsToParallelStates_ShouldRejectTwoDeepTargetsInDifferentRegionsAtTopLevel(t *testing.T) {
	t.Skip("skipped in JS")
	// JS TODO: this test has the same body as the one before it (sends BROKEN_DIFFERENT_REGIONS_3).
	actorRef := xs.CreateActor(multiple1Machine()).Start()
	assert.Panics(t, func() { actorRef.Send(xs.Ev("BROKEN_DIFFERENT_REGIONS_3")) })
}

// JS: multiple > transitions to parallel states > should reject two deep targets in different regions at different levels
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/multiple.test.ts#L205
func TestMultiple_TransitionsToParallelStates_ShouldRejectTwoDeepTargetsInDifferentRegionsAtDifferentLevels(t *testing.T) {
	t.Skip("skipped in JS")
	actorRef := xs.CreateActor(multiple1Machine()).Start()
	assert.Panics(t, func() { actorRef.Send(xs.Ev("BROKEN_DIFFERENT_REGIONS_4")) })
}
