package todomvc_test

import (
	"testing"

	"github.com/nguyenvanduocit/go-xstate/examples/internal/tracetest"
	todomvc "github.com/nguyenvanduocit/go-xstate/examples/todomvc"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// seqIDs returns a newID stub yielding ids in order. The values are the inputs the JS
// trace feeds through a stubbed Math.random:
// (0.1234567890123).toString(36).substring(7) === "xkxayr", and so on.
func seqIDs(ids ...string) func() string {
	i := 0
	return func() string {
		id := ids[i]
		i++
		return id
	}
}

// TestTodosTrace replays testdata/todos.golden.json, recorded by
// scripts/trace/todomvc-react/todos.ts.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/todomvc-react/src/todosMachine.ts#L11
// JS trace: scripts/trace/todomvc-react/todos.ts.
func TestTodosTrace(t *testing.T) {
	machine := todomvc.Machine(seqIDs("xkxayr", "bta69r", "a4v5h4", "tcp7oc", "5j23v", "janbwe"))
	tracetest.Run(t, "testdata/todos.golden.json", func(clock xs.Clock, input any) *xs.Actor[*xs.MachineSnapshot[todomvc.Context]] {
		return xs.CreateActor(machine)
	})
}

// TestTodoTrace replays testdata/todo.golden.json, recorded by
// scripts/trace/todomvc-react/todo.ts (onCommit and focusInput stay the setup() no-ops).
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/todomvc-react/src/Todo.tsx#L8
// JS trace: scripts/trace/todomvc-react/todo.ts.
func TestTodoTrace(t *testing.T) {
	tracetest.Run(t, "testdata/todo.golden.json", func(clock xs.Clock, input any) *xs.Actor[*xs.MachineSnapshot[todomvc.TodoContext]] {
		return xs.CreateActor(todomvc.TodoMachine(), xs.WithInput(input))
	})
}
