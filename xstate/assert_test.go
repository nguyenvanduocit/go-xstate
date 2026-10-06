package xstate_test

import (
	"testing"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/assert"
)

// JS: assertion helpers > assertEvent asserts the correct event type
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/assert.test.ts#L4
func TestAssert_AssertEventAssertsCorrectEventType(t *testing.T) {
	sig := newSignal()
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		On: map[string]xs.Transitions{
			"greet": {{Actions: xs.Actions{xs.ActionRef{Type: "greet"}}}},
			"count": {{Actions: xs.Actions{xs.ActionRef{Type: "greet"}}}},
		},
	}, xs.Implementations{
		Actions: map[string]xs.Action{
			"greet": xs.ActionFunc(func(a xs.ActionArgs[any]) {
				// JS `// @ts-expect-error event.message` / `event.count` and
				// `event.message satisfies string` are type-level only.
				xs.AssertEvent(a.Event, "greet")
			}),
		},
	})

	actor := xs.CreateActor(machine)

	actor.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{
		Error: func(err any) {
			e, ok := err.(error)
			if assert.True(t, ok, "expected an error value, got %#v", err) {
				assert.EqualError(t, e, `Expected event {"type":"count","value":42} to have type matching "greet"`)
			}
			sig.Resolve()
		},
	})

	actor.Start()

	actor.Send(xs.E{"type": "count", "value": 42})

	sig.Wait(t)
}

// JS: assertion helpers > assertEvent asserts multiple event types
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/assert.test.ts#L52
func TestAssert_AssertEventAssertsMultipleEventTypes(t *testing.T) {
	sig := newSignal()
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		On: map[string]xs.Transitions{
			"greet": {{Actions: xs.Actions{xs.ActionRef{Type: "greet"}}}},
			"count": {{Actions: xs.Actions{xs.ActionRef{Type: "greet"}}}},
		},
	}, xs.Implementations{
		Actions: map[string]xs.Action{
			"greet": xs.ActionFunc(func(a xs.ActionArgs[any]) {
				// JS `// @ts-expect-error` accesses and `satisfies` checks are
				// type-level only; the runtime calls are the two assertEvent calls.
				xs.AssertEvent(a.Event, "greet", "notify")

				xs.AssertEvent(a.Event, "notify")
			}),
		},
	})

	actor := xs.CreateActor(machine)

	actor.Subscribe(xs.Observer[*xs.MachineSnapshot[any]]{
		Error: func(err any) {
			e, ok := err.(error)
			if assert.True(t, ok, "expected an error value, got %#v", err) {
				assert.EqualError(t, e, `Expected event {"type":"count","value":42} to have one of types matching "greet", "notify"`)
			}
			sig.Resolve()
		},
	})

	actor.Start()

	actor.Send(xs.E{"type": "count", "value": 42})

	sig.Wait(t)
}
