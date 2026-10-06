package mediascanner

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/nguyenvanduocit/go-xstate/examples/internal/tracetest"
	xs "github.com/nguyenvanduocit/go-xstate/xstate"
	"github.com/stretchr/testify/require"
)

func traceActors(t *testing.T, clock xs.Clock, scenario string) Actors {
	t.Helper()
	service := func(id string, wantInput, output, failure any) xs.ActorLogic {
		return xs.FromCallback(func(a xs.CallbackArgs) func() {
			require.Equal(t, wantInput, a.Input, "input for %s", id)
			timer := clock.SetTimeout(func() {
				if failure != nil {
					a.SendBack(xs.ErrorActorEvent{ActorID: id, Error: failure})
				} else {
					a.SendBack(xs.DoneActorEvent{ActorID: id, Output: output})
				}
			}, time.Millisecond)
			return func() { clock.ClearTimeout(timer) }
		})
	}
	var scanError, permissionError, evaluationError, moveError any
	report := []string{"/library/denied"}
	switch scenario {
	case "scan-error":
		scanError = "scan failed"
	case "permissions-error":
		permissionError = &FileError{Message: "Error checking file permissions", Cause: &FileError{Message: "No accessible files found to move", DirsToReport: &report}}
	case "permissions-report":
		permissionError = &FileError{DirsToReport: &report}
	case "evaluate-error":
		evaluationError = "evaluate failed"
	case "move-error":
		moveError = "move failed"
	}
	message := "all files moved successfully"
	if scenario == "partial-move" {
		message = "files moved with errors"
	}
	return Actors{
		ScanLibrary:          service("scanLibrary", ScanInput{"/library"}, []string{"/library/movie", "/library/denied"}, scanError),
		CheckFilePermissions: service("checkFilePermissions", PermissionInput{[]string{"/library/movie", "/library/denied"}}, PermissionResult{[]string{"/library/movie"}, report}, permissionError),
		EvaluateFiles:        service("evaluatingFiles", EvaluationInput{[]string{"/library/movie"}, []string{"mp4", "mkv", "avi", "mov", "m4v", "mpg", "mpeg", "wmv", "flv", "ts", "mts"}}, EvaluationResult{[]string{"/library/movie/video.mkv"}}, evaluationError),
		MoveFiles:            service("moveFiles", MoveInput{[]string{"/library/movie/video.mkv"}, "/archive"}, MoveResult{Message: message}, moveError),
	}
}

// JS example declaration (upstream has no test case).
// https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-media-scanner/src/mediaScannerMachine.ts#L9
// Recorded cases: scripts/trace/workflow-media-scanner/{success,scan-error,permissions-error,permissions-report,evaluate-error,move-error,partial-move}.ts.
func TestGoldenTraces(t *testing.T) {
	for _, scenario := range []string{"success", "scan-error", "permissions-error", "permissions-report", "evaluate-error", "move-error", "partial-move"} {
		t.Run(scenario, func(t *testing.T) {
			tracetest.Run(t, "testdata/"+scenario+".golden.json", func(clock xs.Clock, input any) *xs.Actor[*xs.MachineSnapshot[Context]] {
				in := input.(map[string]any)
				actor := xs.CreateActor(NewMachine(traceActors(t, clock, scenario), io.Discard), xs.WithInput(Input{in["basePath"].(string), in["destinationPath"].(string)}), xs.WithClock(clock))
				actor.Subscribe(xs.Observer[*xs.MachineSnapshot[Context]]{Error: func(any) {}})
				return actor
			})
		})
	}
}

// JS entry declaration (upstream has no test case).
// https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-media-scanner/src/index.ts#L4
// Recorded stdout: scripts/trace/workflow-media-scanner/workflow-media-scanner.stdout.ts.
func TestEntryStdout(t *testing.T) {
	actors := DefaultActors(DefaultHandlers(io.Discard))
	actors.ScanLibrary = xs.FromPromise(func(context.Context, xs.PromiseArgs) ([]string, error) {
		return nil, errors.New("fixture scan failure")
	})
	var out strings.Builder
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	require.NoError(t, RunWith(ctx, &out, Input{"YOUR BASE PATH HERE", "YOUR DESTINATION PATH HERE"}, actors))
	want, err := os.ReadFile("testdata/workflow-media-scanner.stdout.txt")
	require.NoError(t, err)
	require.Equal(t, string(want), out.String())
}

// Go-only cancellation regression; JS entry has no cancellation test.
// https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-media-scanner/src/index.ts#L4
// Related recorder: scripts/trace/workflow-media-scanner/workflow-media-scanner.stdout.ts; cancellation expectations are Go-only.
func TestRunCancellation(t *testing.T) {
	actors := DefaultActors(DefaultHandlers(io.Discard))
	actors.ScanLibrary = xs.FromPromise(func(ctx context.Context, _ xs.PromiseArgs) ([]string, error) { <-ctx.Done(); return nil, ctx.Err() })
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	require.ErrorIs(t, RunWith(ctx, io.Discard, Input{}, actors), context.Canceled)
}
