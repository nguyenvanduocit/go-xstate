// Package hello ports references/xstate/examples/workflow-hello/main.ts
// (serverless workflow "hello world").
package hello

import (
	"fmt"
	"io"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// Context is the machine's extended state. The JS machine declares none, so a
// snapshot carries `{}`.
type Context struct{}

// Output is the output of the `Hello State` final state: `{ result: 'Hello World!' }`.
type Output struct {
	Result string `json:"result"`
}

// Machine mirrors `workflow`: its initial state is final, so the actor is done as
// soon as it starts.
func Machine() *xs.StateMachine[Context] {
	return xs.CreateMachine(xs.MachineConfig[Context]{
		ID:      "helloworld",
		Initial: "Hello State",
		States: xs.States{
			{
				Key:    "Hello State",
				Type:   xs.Final,
				Output: Output{Result: "Hello World!"},
			},
		},
	})
}

// Run mirrors the entry of main.ts: it starts the actor and prints
// "workflow completed <output>" to w when it completes.
func Run(w io.Writer) {
	actor := xs.CreateActor(Machine())
	actor.Subscribe(xs.Observer[*xs.MachineSnapshot[Context]]{
		Complete: func() {
			fmt.Fprintln(w, "workflow completed", formatOutput(actor.GetSnapshot().Output))
		},
	})
	actor.Start()
}

// formatOutput renders the machine output the way console.log prints it in the
// JS runtime: the root snapshot's output is `undefined` (a final child state's
// `output` only feeds the parent's done event, and this machine has no root
// `output`: getMachineOutput in stateUtils.ts returns early), so a nil output
// prints `undefined`.
func formatOutput(out any) string {
	if out == nil {
		return "undefined"
	}
	return fmt.Sprint(out)
}
