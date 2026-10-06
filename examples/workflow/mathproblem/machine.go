// Package mathproblem ports references/xstate/examples/workflow-math-problem/main.ts
// (serverless workflow "solving math problems").
package mathproblem

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// SolveDelay is the per-problem delay of the real batchMathFunction.
const SolveDelay = time.Second

// Input is the machine input: `{ expressions: string[] }`.
type Input struct {
	Expressions []string `json:"expressions"`
}

// BatchInput is the input of the batchMathFunction actor: `{ problems: string[] }`.
type BatchInput struct {
	Problems []string `json:"problems"`
}

// Solved is one element of the batchMathFunction output: `{ problem, result }`.
type Solved struct {
	Problem string `json:"problem"`
	Result  string `json:"result"`
}

// Context is the machine's extended state: `{ results: string[] | undefined }`.
// A nil Results marshals as an absent key (JS `undefined`), an empty non-nil
// slice as `[]`.
type Context struct {
	Results []string
}

// MarshalJSON keeps `undefined` (nil) and `[]` (empty) apart, like JSON.stringify.
func (c Context) MarshalJSON() ([]byte, error) {
	if c.Results == nil {
		return []byte(`{}`), nil
	}
	return json.Marshal(struct {
		Results []string `json:"results"`
	}{c.Results})
}

// Output is the output of the `Solved` final state: `{ results }`.
type Output = Context

// SolveBatch mirrors the body of batchMathFunction: every problem logs
// `solving <problem>` in order, waits delay concurrently with the others and
// resolves to `Solved <problem>`. Cancelling ctx abandons the wait.
func SolveBatch(ctx context.Context, w io.Writer, delay time.Duration, problems []string) ([]Solved, error) {
	out := make([]Solved, len(problems))
	var wg sync.WaitGroup
	for i, problem := range problems {
		fmt.Fprintln(w, "solving", problem)
		wg.Add(1)
		go func() {
			defer wg.Done()
			timer := time.NewTimer(delay)
			defer timer.Stop()
			select {
			case <-ctx.Done():
			case <-timer.C:
				out[i] = Solved{Problem: problem, Result: "Solved " + problem}
			}
		}()
	}
	wg.Wait()
	if err := context.Cause(ctx); err != nil {
		return nil, err
	}
	return out, nil
}

// BatchMathFunction is the batchMathFunction actor of main.ts, printing to w.
func BatchMathFunction(w io.Writer, delay time.Duration) xs.ActorLogic {
	return xs.FromPromise(func(ctx context.Context, a xs.PromiseArgs) ([]Solved, error) {
		return SolveBatch(ctx, w, delay, a.Input.(BatchInput).Problems)
	})
}

// NewMachine mirrors `workflow` with the batchMathFunction actor given.
func NewMachine(batchMathFunction xs.ActorLogic) *xs.StateMachine[Context] {
	return xs.NewSetup[Context](xs.Implementations{
		Actors: map[string]xs.ActorLogic{"batchMathFunction": batchMathFunction},
	}).CreateMachine(xs.MachineConfig[Context]{
		ID:      "math-problem",
		Initial: "Solve",
		Context: Context{},
		States: xs.States{
			{
				Key: "Solve",
				Invoke: []xs.InvokeConfig{{
					Src: "batchMathFunction",
					Input: xs.NewExpr(func(a xs.ExprArgs[Context]) any {
						return BatchInput{Problems: a.Event.(xs.InitEvent).Input.(Input).Expressions}
					}),
					OnDone: xs.Transitions{{
						Target: "Solved",
						Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[Context]) Context {
							solved := a.Event.(xs.DoneActorEvent).Output.([]Solved)
							results := make([]string, len(solved))
							for i, r := range solved {
								results[i] = r.Result
							}
							return Context{Results: results}
						})},
					}},
				}},
			},
			{
				Key:  "Solved",
				Type: xs.Final,
				Output: xs.NewExpr(func(a xs.ExprArgs[Context]) any {
					return Output{Results: a.Context.Results}
				}),
			},
		},
	})
}

// Run mirrors the entry of main.ts with the real 1000 ms batchMathFunction, printing to w.
func Run(w io.Writer) {
	RunWith(w, SolveDelay)
}

// RunWith is Run with batchMathFunction's delay injected.
func RunWith(w io.Writer, delay time.Duration) {
	var mu sync.Mutex
	locked := &lockedWriter{mu: &mu, w: w}
	done := make(chan struct{})
	actor := xs.CreateActor(NewMachine(BatchMathFunction(locked, delay)), xs.WithInput(Input{
		Expressions: []string{"2+2", "4-1", "10x3", "20/2"},
	}))
	actor.Subscribe(xs.Observer[*xs.MachineSnapshot[Context]]{
		Complete: func() {
			fmt.Fprintln(locked, "workflow completed", formatOutput(actor.GetSnapshot().Output))
			close(done)
		},
	})
	actor.Start()
	<-done
}

// formatOutput renders the machine output the way console.log prints it in the
// JS runtime. The machine declares no root `output`, so the snapshot output is
// `undefined`.
func formatOutput(out any) string {
	if out == nil {
		return "undefined"
	}
	return fmt.Sprint(out)
}

type lockedWriter struct {
	mu *sync.Mutex
	w  io.Writer
}

func (l *lockedWriter) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.w.Write(p)
}
