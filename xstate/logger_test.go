package xstate_test

import (
	"testing"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/assert"
)

// JS: logger > system logger should be default logger for actors (invoked from machine)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/logger.test.ts#L4
func TestLogger_SystemLoggerShouldBeDefaultLoggerForActorsInvokedFromMachine(t *testing.T) {
	// expect.assertions(1): the logger must be called exactly once, with 'hello'.
	loggerSpy := newSpy()
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Invoke: []xs.InvokeConfig{{
			Logic: xs.CreateMachine(xs.MachineConfig[any]{
				Entry: xs.Actions{xs.Log("hello")},
			}),
		}},
	})

	actor := xs.CreateActor(machine, xs.WithLogger(func(args ...any) {
		loggerSpy.Call(args...)
	})).Start()

	actor.Start()

	calls := loggerSpy.Calls()
	if assert.Len(t, calls, 1) && assert.NotEmpty(t, calls[0]) {
		assert.Equal(t, "hello", calls[0][0])
	}
}

// JS: logger > system logger should be default logger for actors (spawned from machine)
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/logger.test.ts#L23
func TestLogger_SystemLoggerShouldBeDefaultLoggerForActorsSpawnedFromMachine(t *testing.T) {
	// expect.assertions(1): the logger must be called exactly once, with 'hello'.
	loggerSpy := newSpy()
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Entry: xs.Actions{xs.SpawnChild(xs.CreateMachine(xs.MachineConfig[any]{
			Entry: xs.Actions{xs.Log("hello")},
		}))},
	})

	actor := xs.CreateActor(machine, xs.WithLogger(func(args ...any) {
		loggerSpy.Call(args...)
	})).Start()

	actor.Start()

	calls := loggerSpy.Calls()
	if assert.Len(t, calls, 1) && assert.NotEmpty(t, calls[0]) {
		assert.Equal(t, "hello", calls[0][0])
	}
}
