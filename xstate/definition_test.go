package xstate_test

import (
	"testing"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// JS: definition > should provide invoke definitions
// JS test: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/packages/core/test/definition.test.ts#L4
func TestDefinition_ShouldProvideInvokeDefinitions(t *testing.T) {
	invokeMachine := xs.CreateMachine(xs.MachineConfig[any]{
		ID:      "invoke",
		Invoke:  []xs.InvokeConfig{{Src: "foo"}, {Src: "bar"}},
		Initial: "idle",
		States: xs.States{
			{Key: "idle"},
		},
	})

	def := invokeMachine.Root.Definition()
	require.Contains(t, def, "invoke")
	assert.Len(t, def["invoke"], 2)
}
