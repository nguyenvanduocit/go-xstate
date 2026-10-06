package graph_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/nguyenvanduocit/go-xstate/graph"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// JS: Forbidden attributes > Should not let you declare invocations on your test machine
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/forbiddenAttributes.test.ts#L5
func TestForbiddenAttributes_ShouldNotLetYouDeclareInvocationsOnYourTestMachine(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Invoke: []xs.InvokeConfig{{Src: "myInvoke"}},
	})

	assert.PanicsWithError(t, "Invocations on test machines are not supported", func() {
		graph.CreateTestModel(machine)
	})
}

// JS: Forbidden attributes > Should not let you declare after on your test machine
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/forbiddenAttributes.test.ts#L17
func TestForbiddenAttributes_ShouldNotLetYouDeclareAfterOnYourTestMachine(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		After: map[string]xs.Transitions{
			"5000": {{Actions: xs.Actions{xs.ActionFunc(func(xs.ActionArgs[any]) {})}}},
		},
	})

	assert.PanicsWithError(t, "After events on test machines are not supported", func() {
		graph.CreateTestModel(machine)
	})
}

// JS: Forbidden attributes > Should not let you delayed actions on your machine
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/src/graph/test/forbiddenAttributes.test.ts#L31
func TestForbiddenAttributes_ShouldNotLetYouDelayedActionsOnYourMachine(t *testing.T) {
	machine := xs.CreateMachine(xs.MachineConfig[any]{
		Entry: xs.Actions{
			xs.Raise(xs.Ev("EVENT"), xs.SendOptions{Delay: 1000 * time.Millisecond}),
		},
	})

	assert.PanicsWithError(t, "Delayed actions on test machines are not supported", func() {
		graph.CreateTestModel(machine)
	})
}
