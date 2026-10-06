package xstate_test

import (
	"errors"
	"testing"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/require"
)

func TestCompileReturnsStaticConfigErrors(t *testing.T) {
	tests := []struct {
		name   string
		config xs.MachineConfig[int]
		state  string
	}{
		{"missing initial", xs.MachineConfig[int]{ID: "root", States: xs.States{{Key: "a"}}}, "root"},
		{"bad initial", xs.MachineConfig[int]{ID: "root", Initial: "missing", States: xs.States{{Key: "a"}}}, "root"},
		{"bad nested initial", xs.MachineConfig[int]{ID: "root", Initial: "a", States: xs.States{{Key: "a"}, {Key: "nested", Initial: "missing", States: xs.States{{Key: "child"}}}}}, "root.nested"},
		{"bad relative target", xs.MachineConfig[int]{ID: "root", Initial: "a", States: xs.States{{Key: "a", On: map[string]xs.Transitions{"next": {{Target: "missing"}}}}}}, "root.a"},
		{"bad id target", xs.MachineConfig[int]{ID: "root", Initial: "a", States: xs.States{{Key: "a", On: map[string]xs.Transitions{"next": {{Target: "#missing"}}}}}}, "root"},
		{"bad root target", xs.MachineConfig[int]{ID: "root", Initial: "a", States: xs.States{{Key: "a"}}, On: map[string]xs.Transitions{"next": {{Target: "a"}}}}, "root"},
		{"null event", xs.MachineConfig[int]{ID: "root", On: map[string]xs.Transitions{"": {{}}}}, "root"},
		{"duplicate key", xs.MachineConfig[int]{ID: "root", Initial: "a", States: xs.States{{Key: "a"}, {Key: "a"}}}, "root"},
		{"duplicate id", xs.MachineConfig[int]{ID: "root", Initial: "a", States: xs.States{{Key: "a", ID: "same"}, {Key: "b", ID: "same"}}}, "same"},
		{"unknown type", xs.MachineConfig[int]{ID: "root", Type: "invalid"}, "root"},
		{"unknown history", xs.MachineConfig[int]{ID: "root", Initial: "a", States: xs.States{{Key: "a"}, {Key: "h", Type: xs.History, History: "invalid"}}}, "root.h"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			machine, err := xs.Compile(tc.config)
			require.Nil(t, machine)
			var configErr *xs.ConfigError
			require.ErrorAs(t, err, &configErr)
			require.Equal(t, tc.state, configErr.StateID)
			require.NotNil(t, errors.Unwrap(configErr))
		})
	}
}

func TestCompileDoesNotExecuteUserCode(t *testing.T) {
	called := false
	machine, err := xs.Compile(xs.MachineConfig[int]{
		ContextFn: func(xs.ContextArgs) int { called = true; return 7 },
		Entry:     xs.Actions{xs.ActionFunc(func(xs.ActionArgs[int]) { called = true })},
	})
	require.NoError(t, err)
	require.False(t, called)
	actor := xs.CreateActor(machine).Start()
	defer actor.Stop()
	require.True(t, called)
	require.Equal(t, 7, actor.GetSnapshot().Context)
}

func TestCompilePreservesMachineSemantics(t *testing.T) {
	cfg := xs.MachineConfig[int]{Context: 3, Initial: "a", States: xs.States{
		{Key: "a", On: map[string]xs.Transitions{"next": {{Target: "b", Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[int]) int { return a.Context + 1 })}}}}},
		{Key: "b", Type: xs.Final},
	}}
	compiled, err := xs.Compile(cfg)
	require.NoError(t, err)
	a, b := xs.CreateActor(compiled).Start(), xs.CreateActor(xs.CreateMachine(cfg)).Start()
	defer a.Stop()
	defer b.Stop()
	a.Send(xs.Ev("next"))
	b.Send(xs.Ev("next"))
	require.Equal(t, b.GetSnapshot().Context, a.GetSnapshot().Context)
	require.Equal(t, b.GetSnapshot().Value, a.GetSnapshot().Value)
	require.Equal(t, b.GetSnapshot().Status, a.GetSnapshot().Status)
}

func TestCreateMachineStillPanicsWithOriginalMessage(t *testing.T) {
	require.PanicsWithError(t, `No initial state specified for compound state node "#root". Try adding { initial: "a" } to the state config.`, func() {
		xs.CreateMachine(xs.MachineConfig[int]{ID: "root", States: xs.States{{Key: "a"}}})
	})
}

func TestCompileRejectsInvalidHistoryDefault(t *testing.T) {
	for _, initial := range []string{"a", "history"} {
		t.Run(initial, func(t *testing.T) {
			machine, err := xs.Compile(xs.MachineConfig[int]{ID: "root", Initial: initial, States: xs.States{
				{Key: "a"}, {Key: "history", Type: xs.History, Target: "missing"},
			}})
			require.Nil(t, machine)
			var configErr *xs.ConfigError
			require.ErrorAs(t, err, &configErr)
		})
	}
}
