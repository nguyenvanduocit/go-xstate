// This negative fixture must not compile: the output types do not match.
package main

import (
	"context"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

func main() {
	task, _ := xs.NewTask(func(context.Context, int) (string, error) { return "E5", nil })
	_, _ = xs.InvokeTask(task, xs.Invocation[int, int, bool]{})
}
