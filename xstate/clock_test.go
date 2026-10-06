package xstate_test

import (
	"testing"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/assert"
)

// JS: clock > system clock should be default clock for actors (invoked from machine)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/clock.test.ts#L4
func TestClock_SystemClockShouldBeDefaultClockForActorsInvokedFromMachine(t *testing.T) {
	clock := xs.NewSimulatedClock()

	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Invoke: []xs.InvokeConfig{{
			ID: "child",
			Logic: xs.CreateMachine(xs.MachineConfig[any]{
				Initial: "a",
				States: xs.States{
					{Key: "a", After: map[string]xs.Transitions{
						"10000": {{Target: "b"}},
					}},
					{Key: "b"},
				},
			}),
		}},
	})

	actor := xs.CreateActor(machine, xs.WithClock(clock)).Start()

	assert.Equal(t, "a", machineSnap[any](actor.GetSnapshot().Children["child"]).Value)

	clock.Increment(ms(10_000))

	assert.Equal(t, "b", machineSnap[any](actor.GetSnapshot().Children["child"]).Value)
}
