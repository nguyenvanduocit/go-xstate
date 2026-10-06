package todomvc_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nguyenvanduocit/go-xstate/examples/internal/tracetest"
	todomvc "github.com/nguyenvanduocit/go-xstate/examples/todomvc"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

func roundTrip(t *testing.T, v any) map[string]any {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	var out map[string]any
	require.NoError(t, json.Unmarshal(b, &out))
	return out
}

// TestTodoCommitTrace replays testdata/todo-commit.golden.json, recorded by
// scripts/trace/todomvc-react/todo-commit.ts: one todo actor wired to the todos
// actor by TodoMachineFor (Todo.tsx's provide()). tracetest.Run only drives one actor,
// so the replay loop lives here.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/todomvc-react/src/Todo.tsx#L77
// JS trace: scripts/trace/todomvc-react/todo-commit.ts.
// JS trace: scripts/trace/todomvc-react/todo.ts.
// JS trace: scripts/trace/todomvc-react/todos.ts.
func TestTodoCommitTrace(t *testing.T) {
	raw, err := os.ReadFile("testdata/todo-commit.golden.json")
	require.NoError(t, err)
	var g struct {
		Steps []struct {
			Step     json.RawMessage `json:"step"`
			Snapshot map[string]any  `json:"snapshot"`
		} `json:"steps"`
	}
	require.NoError(t, json.Unmarshal(raw, &g))
	require.NotEmpty(t, g.Steps)

	todos := xs.CreateActor(todomvc.Machine(func() string { t.Fatal("no todo is created in this trace"); return "" }))
	todos.Start()
	defer todos.Stop()
	first := todos.GetSnapshot().Context.Todos[0]
	todo := xs.CreateActor(todomvc.TodoMachineFor(todos, first), xs.WithInput(todomvc.TodoInput{Todo: first}))
	todo.Start()
	defer todo.Stop()

	snapshot := func() map[string]any {
		return roundTrip(t, map[string]any{
			"todo":  tracetest.View(todo.GetSnapshot()),
			"todos": tracetest.View(todos.GetSnapshot()),
		})
	}
	for i, s := range g.Steps {
		var name string
		if json.Unmarshal(s.Step, &name) == nil {
			require.Equal(t, "start", name)
		} else {
			var step struct {
				To   string         `json:"to"`
				Send map[string]any `json:"send"`
			}
			require.NoError(t, json.Unmarshal(s.Step, &step))
			switch step.To {
			case "todo":
				todo.Send(xs.E(step.Send))
			case "todos":
				todos.Send(xs.E(step.Send))
			default:
				t.Fatalf("step %d: unknown target %q", i, step.To)
			}
		}
		require.Equal(t, s.Snapshot, snapshot(), "step %d (%s)", i, s.Step)
	}
}

// TestPersistedTrace replays testdata/persisted.golden.json, recorded by
// scripts/trace/todomvc-react/persisted.ts: the localStorage round trip of App.tsx and
// Todos.tsx. Session 2 starts from the snapshot the JS run persisted.
// JS reference: https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/todomvc-react/src/todosMachine.ts#L11
// JS trace: scripts/trace/todomvc-react/persisted.ts.
func TestPersistedTrace(t *testing.T) {
	raw, err := os.ReadFile("testdata/persisted.golden.json")
	require.NoError(t, err)
	var g struct {
		Sessions []struct {
			RestoredFrom any `json:"restoredFrom"`
			Steps        []struct {
				Step      json.RawMessage `json:"step"`
				Snapshot  map[string]any  `json:"snapshot"`
				Persisted map[string]any  `json:"persisted"`
			} `json:"steps"`
		} `json:"sessions"`
	}
	require.NoError(t, json.Unmarshal(raw, &g))
	require.Len(t, g.Sessions, 2)

	// Math.random is stubbed once for both sessions in persisted.ts, so the ids continue.
	newID := seqIDs("xkxayr", "bta69r")
	for n, session := range g.Sessions {
		opts := []xs.ActorOption{}
		if session.RestoredFrom != nil {
			opts = append(opts, xs.WithSnapshot(session.RestoredFrom))
		}
		actor := xs.CreateActor(todomvc.Machine(newID), opts...)
		actor.Start()
		for i, s := range session.Steps {
			var name string
			if json.Unmarshal(s.Step, &name) == nil {
				require.Equal(t, "start", name)
			} else {
				var step struct {
					Send map[string]any `json:"send"`
				}
				require.NoError(t, json.Unmarshal(s.Step, &step))
				actor.Send(xs.E(step.Send))
			}
			require.Equal(t, s.Snapshot, roundTrip(t, tracetest.View(actor.GetSnapshot())), "session %d step %d snapshot", n, i)
			// JSON.stringify drops the `undefined` output/error of an active machine; Go carries
			// them as null (docs/porting/notes/examples-lib-findings.md, persisted-donut-maker).
			persisted := roundTrip(t, actor.GetPersistedSnapshot())
			for _, k := range []string{"output", "error"} {
				if v, ok := persisted[k]; ok && v == nil {
					delete(persisted, k)
				}
			}
			require.Equal(t, s.Persisted, persisted, "session %d step %d persisted", n, i)
		}
		actor.Stop()
	}
}
