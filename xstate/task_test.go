package xstate_test

import (
	"context"
	"encoding/json"
	"errors"
	"os/exec"
	"sync/atomic"
	"testing"
	"time"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/require"
)

type taskTestContext[O any] struct {
	Output  O
	Failure error `json:"-"`
}

func taskTestMachine[I, O any](t *testing.T, run func(context.Context, I) (O, error), input I) *xs.StateMachine[taskTestContext[O]] {
	t.Helper()
	task, err := xs.NewTask(run)
	require.NoError(t, err)
	invoke, err := xs.InvokeTask(task, xs.Invocation[taskTestContext[O], I, O]{
		ID: "work", DoneTarget: "done", ErrorTarget: "failed",
		Input:  func(taskTestContext[O]) I { return input },
		Done:   func(c taskTestContext[O], out O) taskTestContext[O] { c.Output = out; return c },
		Failed: func(c taskTestContext[O], err error) taskTestContext[O] { c.Failure = err; return c },
	})
	require.NoError(t, err)
	machine, err := xs.Compile(xs.MachineConfig[taskTestContext[O]]{Initial: "working", States: xs.States{
		{Key: "working", Invoke: []xs.InvokeConfig{invoke}}, {Key: "done", Type: xs.Final}, {Key: "failed", Type: xs.Final},
	}})
	require.NoError(t, err)
	return machine
}

func awaitTask[O any](t *testing.T, actor *xs.Actor[*xs.MachineSnapshot[taskTestContext[O]]]) taskTestContext[O] {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	snap, err := xs.Await(ctx, actor, func(s *xs.MachineSnapshot[taskTestContext[O]]) bool { return s.Status == xs.StatusDone })
	require.NoError(t, err)
	return snap.Context
}

func TestInvokeTaskTypedInputOutput(t *testing.T) {
	machine := taskTestMachine(t, func(_ context.Context, n int) (string, error) { return string(rune('A' + n)), nil }, 4)
	actor := xs.CreateActor(machine).Start()
	defer actor.Stop()
	got := awaitTask(t, actor)
	require.NoError(t, got.Failure)
	require.Equal(t, "E", got.Output)
}

func TestInvokeTaskPreservesErrorIdentity(t *testing.T) {
	failure := errors.New("provider unavailable")
	machine := taskTestMachine(t, func(context.Context, int) (int, error) { return 0, failure }, 1)
	actor := xs.CreateActor(machine).Start()
	defer actor.Stop()
	require.ErrorIs(t, awaitTask(t, actor).Failure, failure)
}

func TestInvokeTaskNilInterfaceValues(t *testing.T) {
	machine := taskTestMachine(t, func(_ context.Context, in any) (any, error) { require.Nil(t, in); return nil, nil }, any(nil))
	actor := xs.CreateActor(machine).Start()
	defer actor.Stop()
	got := awaitTask(t, actor)
	require.NoError(t, got.Failure)
	require.Nil(t, got.Output)
	require.Equal(t, "done", actor.GetSnapshot().Value)
}

func TestInvokeTaskStopsOutstandingWork(t *testing.T) {
	started, stopped := make(chan struct{}), make(chan struct{})
	machine := taskTestMachine(t, func(ctx context.Context, n int) (int, error) {
		close(started)
		<-ctx.Done()
		close(stopped)
		return 99, ctx.Err()
	}, 1)
	actor := xs.CreateActor(machine).Start()
	defer actor.Stop()
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("task did not start")
	}
	actor.Stop()
	select {
	case <-stopped:
	case <-time.After(3 * time.Second):
		t.Fatal("task did not receive cancellation")
	}
	require.Equal(t, xs.StatusStopped, actor.GetSnapshot().Status)
	require.Zero(t, actor.GetSnapshot().Context.Output)
}

func TestInvokeTaskRestoresActiveJSONInput(t *testing.T) {
	type input struct {
		Number int
		Labels []string
	}
	var calls atomic.Int32
	started, stopped := make(chan struct{}), make(chan struct{})
	machine := taskTestMachine(t, func(ctx context.Context, in input) (string, error) {
		if calls.Add(1) == 1 {
			close(started)
			<-ctx.Done()
			close(stopped)
			return "", ctx.Err()
		}
		if in.Number != 7 || len(in.Labels) != 1 || in.Labels[0] != "E5" {
			return "", errors.New("input lost during restore")
		}
		return in.Labels[0], nil
	}, input{Number: 7, Labels: []string{"E5"}})
	original := xs.CreateActor(machine).Start()
	defer original.Stop()
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("task did not start")
	}
	encoded, err := json.Marshal(original.GetPersistedSnapshot())
	require.NoError(t, err)
	original.Stop()
	select {
	case <-stopped:
	case <-time.After(3 * time.Second):
		t.Fatal("old task did not stop")
	}
	var persisted map[string]any
	require.NoError(t, json.Unmarshal(encoded, &persisted))
	restored := xs.CreateActor(machine, xs.WithSnapshot(persisted)).Start()
	defer restored.Stop()
	got := awaitTask(t, restored)
	require.NoError(t, got.Failure)
	require.Equal(t, "E5", got.Output)
	require.Equal(t, int32(2), calls.Load())
}

func TestInvokeTaskRejectsInvalidConfiguration(t *testing.T) {
	_, err := xs.NewTask[int, int](nil)
	require.Error(t, err)
	_, err = xs.InvokeTask(xs.Task[int, int]{}, xs.Invocation[int, int, int]{})
	require.Error(t, err)
	task, err := xs.NewTask(func(context.Context, int) (int, error) { return 0, nil })
	require.NoError(t, err)
	cfg := xs.Invocation[int, int, int]{ID: "work", DoneTarget: "done", ErrorTarget: "failed", Input: func(int) int { return 0 }, Done: func(c, out int) int { return out }, Failed: func(c int, _ error) int { return c }}
	for _, field := range []string{"input", "done", "failed", "id", "done target", "error target"} {
		t.Run(field, func(t *testing.T) {
			invalid := cfg
			switch field {
			case "input":
				invalid.Input = nil
			case "done":
				invalid.Done = nil
			case "failed":
				invalid.Failed = nil
			case "id":
				invalid.ID = ""
			case "done target":
				invalid.DoneTarget = ""
			case "error target":
				invalid.ErrorTarget = ""
			}
			_, err := xs.InvokeTask(task, invalid)
			require.Error(t, err)
		})
	}
}

func TestInvokeTaskRestoredMalformedOutputReachesParent(t *testing.T) {
	var calls atomic.Int32
	machine := taskTestMachine(t, func(context.Context, int) (int, error) {
		calls.Add(1)
		return 1, nil
	}, 1)
	original := xs.CreateActor(machine)
	defer original.Stop()
	encoded, err := json.Marshal(original.GetPersistedSnapshot())
	require.NoError(t, err)
	var persisted map[string]any
	require.NoError(t, json.Unmarshal(encoded, &persisted))
	child := persisted["children"].(map[string]any)["work"].(map[string]any)
	child["snapshot"] = map[string]any{"status": "done", "output": map[string]any{"json": "1 garbage"}}
	restored := xs.CreateActor(machine, xs.WithSnapshot(persisted)).Start()
	defer restored.Stop()
	got := awaitTask(t, restored)
	require.Equal(t, "failed", restored.GetSnapshot().Value)
	require.ErrorContains(t, got.Failure, "restore task snapshot")
	require.ErrorContains(t, got.Failure, "task payload")
	require.Zero(t, calls.Load(), "restoration failure must not execute user work")
}

func TestInvokeTaskMismatchedOutputDoesNotCompile(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, "go", "test", "./testdata/typed_task_invalid").CombinedOutput()
	require.NoError(t, ctx.Err(), "compiler timed out: %s", output)
	require.Error(t, err, "invalid output type compiled")
	require.Contains(t, string(output), "InvokeTask")
	require.Contains(t, string(output), "bool")
	require.Contains(t, string(output), "string")
}
