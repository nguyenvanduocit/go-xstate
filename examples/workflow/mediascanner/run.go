package mediascanner

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

// Run uses the placeholder paths from the upstream entry.
func Run(w io.Writer) error {
	return RunWith(context.Background(), w, Input{"YOUR BASE PATH HERE", "YOUR DESTINATION PATH HERE"}, DefaultActors(DefaultHandlers(w)))
}

// RunWith runs one scan with explicit paths and services, stopping on idle or an error report.
func RunWith(ctx context.Context, w io.Writer, input Input, actors Actors) error {
	fmt.Fprintln(w, "Starting the awesome media scanner thingy")
	actor := xs.CreateActor(NewMachine(actors, w), xs.WithInput(input))
	done := make(chan error, 1)
	started := false
	actor.Subscribe(xs.Observer[*xs.MachineSnapshot[Context]]{
		Next: func(snapshot *xs.MachineSnapshot[Context]) {
			printSnapshot(w, snapshot)
			if snapshot.Matches("Scanning") {
				started = true
			}
			if snapshot.Matches("ReportingErrors") || (started && snapshot.Matches("idle")) {
				done <- nil
			}
		},
		Error: func(err any) { done <- fmt.Errorf("%v", err) },
	})
	actor.Start()
	defer actor.Stop()
	actor.Send(xs.Ev("START_SCAN"))
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func printSnapshot(w io.Writer, snapshot *xs.MachineSnapshot[Context]) {
	c := snapshot.Context
	fmt.Fprintf(w, "{\n  state: %s,\n  error: undefined,\n  context: {\n    basePath: %s,\n    destinationPath: %s,\n", strconv.Quote(fmt.Sprint(snapshot.Value)), strconv.Quote(c.BasePath), strconv.Quote(c.DestinationPath))
	for _, field := range []struct {
		name   string
		values []string
	}{{"directoriesToCheck", c.DirectoriesToCheck}, {"dirsToEvaluate", c.DirsToEvaluate}, {"dirsToMove", c.DirsToMove}, {"filesToEmail", c.FilesToEmail}} {
		fmt.Fprintf(w, "    %s: %s,\n", field.name, formatArray(field.values))
	}
	if c.DirsToReport == nil {
		fmt.Fprintln(w, "    dirsToReport: undefined,")
	} else {
		fmt.Fprintf(w, "    dirsToReport: %s,\n", formatArray(*c.DirsToReport))
	}
	fmt.Fprintf(w, "    processedFiles: %s,\n    acceptedFileTypes: [\n      %s\n    ],\n  },\n}\n", formatArray(c.ProcessedFiles), quoted(c.AcceptedFileTypes))
}
func quoted(values []string) string {
	out := make([]string, len(values))
	for i, v := range values {
		out[i] = strconv.Quote(v)
	}
	return strings.Join(out, ", ")
}
func formatArray(values []string) string {
	if len(values) == 0 {
		return "[]"
	}
	return "[ " + quoted(values) + " ]"
}
