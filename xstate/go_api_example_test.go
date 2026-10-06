package xstate_test

import (
	"context"
	"fmt"
	"time"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

func ExampleInvokeTask() {
	type Turn struct{ Point string }
	type Decision struct{ Point string }
	type Match struct {
		Point string
		Err   error
	}

	choose, err := xs.NewTask(func(ctx context.Context, turn Turn) (Decision, error) {
		return Decision{Point: turn.Point}, nil
	})
	if err != nil {
		panic(err)
	}
	invocation, err := xs.InvokeTask(choose, xs.Invocation[Match, Turn, Decision]{
		ID: "choose", DoneTarget: "done", ErrorTarget: "failed",
		Input:  func(Match) Turn { return Turn{Point: "E5"} },
		Done:   func(c Match, d Decision) Match { c.Point = d.Point; return c },
		Failed: func(c Match, err error) Match { c.Err = err; return c },
	})
	if err != nil {
		panic(err)
	}
	machine, err := xs.Compile(xs.MachineConfig[Match]{Initial: "thinking", States: xs.States{
		{Key: "thinking", Invoke: []xs.InvokeConfig{invocation}},
		{Key: "done", Type: xs.Final}, {Key: "failed", Type: xs.Final},
	}})
	if err != nil {
		panic(err)
	}
	actor := xs.CreateActor(machine).Start()
	defer actor.Stop()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	snapshot, err := xs.Await(ctx, actor, func(s *xs.MachineSnapshot[Match]) bool { return s.Status == xs.StatusDone })
	if err != nil {
		panic(err)
	}
	if snapshot.Context.Err != nil {
		panic(snapshot.Context.Err)
	}
	fmt.Println(snapshot.Context.Point)
	// Output: E5
}
