// Package mediascanner ports XState's media-scanner workflow and filesystem helpers.
package mediascanner

import (
	"context"
	"fmt"
	"io"

	xs "github.com/nguyenvanduocit/go-xstate/xstate"
)

type Input struct {
	BasePath        string `json:"basePath"`
	DestinationPath string `json:"destinationPath"`
}

type Context struct {
	BasePath           string    `json:"basePath"`
	DestinationPath    string    `json:"destinationPath"`
	DirectoriesToCheck []string  `json:"directoriesToCheck"`
	DirsToEvaluate     []string  `json:"dirsToEvaluate"`
	DirsToMove         []string  `json:"dirsToMove"`
	FilesToEmail       []string  `json:"filesToEmail"`
	DirsToReport       *[]string `json:"dirsToReport,omitempty"`
	ProcessedFiles     []string  `json:"processedFiles"`
	AcceptedFileTypes  []string  `json:"acceptedFileTypes"`
}

type PermissionResult struct {
	DirsToEvaluate []string `json:"dirsToEvaluate"`
	DirsToReport   []string `json:"dirsToReport"`
}
type EvaluationResult struct {
	DirsToMove []string `json:"dirsToMove"`
}
type ScanInput struct {
	BasePath string `json:"basePath"`
}
type PermissionInput struct {
	DirectoriesToCheck []string `json:"directoriesToCheck"`
}
type EvaluationInput struct {
	DirsToEvaluate    []string `json:"dirsToEvaluate"`
	AcceptedFileTypes []string `json:"acceptedFileTypes"`
}
type MoveInput struct {
	DirsToMove      []string `json:"dirsToMove"`
	DestinationPath string   `json:"destinationPath"`
}
type Actors struct{ ScanLibrary, CheckFilePermissions, EvaluateFiles, MoveFiles xs.ActorLogic }

func DefaultActors(h Handlers) Actors {
	return Actors{
		ScanLibrary: xs.FromPromise(func(_ context.Context, a xs.PromiseArgs) ([]string, error) {
			return h.ScanDirectories(a.Input.(ScanInput).BasePath)
		}),
		CheckFilePermissions: xs.FromPromise(func(_ context.Context, a xs.PromiseArgs) (PermissionResult, error) {
			return h.CheckFilePermissions(a.Input.(PermissionInput).DirectoriesToCheck)
		}),
		EvaluateFiles: xs.FromPromise(func(ctx context.Context, a xs.PromiseArgs) (EvaluationResult, error) {
			in := a.Input.(EvaluationInput)
			return h.EvaluateFiles(ctx, in.DirsToEvaluate, in.AcceptedFileTypes)
		}),
		MoveFiles: xs.FromPromise(func(_ context.Context, a xs.PromiseArgs) (MoveResult, error) {
			in := a.Input.(MoveInput)
			return h.MoveFiles(in.DirsToMove, in.DestinationPath)
		}),
	}
}

func NewMachine(actors Actors, w io.Writer) *xs.StateMachine[Context] {
	return xs.NewSetup[Context](xs.Implementations{
		Actors: map[string]xs.ActorLogic{
			"scanLibrary":          actors.ScanLibrary,
			"checkFilePermissions": actors.CheckFilePermissions,
			"evaluateFiles":        actors.EvaluateFiles,
			"moveFiles":            actors.MoveFiles,
		},
		Actions: map[string]xs.Action{
			"emailErrors": xs.ActionFunc(func(xs.ActionArgs[Context]) { fmt.Fprintln(w, "Emailing errors") }),
		},
	}).CreateMachine(xs.MachineConfig[Context]{
		ID:      "mediaScanner",
		Initial: "idle",
		ContextFn: func(a xs.ContextArgs) Context {
			in := a.Input.(Input)
			report := []string{}
			return Context{
				BasePath:           in.BasePath,
				DestinationPath:    in.DestinationPath,
				DirectoriesToCheck: []string{},
				DirsToEvaluate:     []string{},
				DirsToMove:         []string{},
				FilesToEmail:       []string{},
				DirsToReport:       &report,
				ProcessedFiles:     []string{},
				AcceptedFileTypes:  []string{"mp4", "mkv", "avi", "mov", "m4v", "mpg", "mpeg", "wmv", "flv", "ts", "mts"},
			}
		},
		States: xs.States{
			{Key: "idle", On: map[string]xs.Transitions{"START_SCAN": {{Target: "Scanning"}}}},
			{
				Key: "Scanning",
				Invoke: []xs.InvokeConfig{{
					ID: "scanLibrary", Src: "scanLibrary",
					Input: xs.NewExpr(func(a xs.ExprArgs[Context]) any { return ScanInput{a.Context.BasePath} }),
					OnDone: xs.Transitions{{
						Target: "CheckingFilePermissions",
						Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[Context]) Context {
							c := a.Context
							c.DirectoriesToCheck = a.Event.(xs.DoneActorEvent).Output.([]string)
							return c
						})},
					}},
					OnError: xs.Transitions{{Target: "ReportingErrors"}},
				}},
			},
			{
				Key: "CheckingFilePermissions",
				Invoke: []xs.InvokeConfig{{
					ID: "checkFilePermissions", Src: "checkFilePermissions",
					Input: xs.NewExpr(func(a xs.ExprArgs[Context]) any { return PermissionInput{a.Context.DirectoriesToCheck} }),
					OnDone: xs.Transitions{{
						Target: "EvaluatingFiles",
						Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[Context]) Context {
							c := a.Context
							out := a.Event.(xs.DoneActorEvent).Output.(PermissionResult)
							c.DirsToEvaluate = out.DirsToEvaluate
							c.DirsToReport = &out.DirsToReport
							return c
						})},
					}},
					OnError: xs.Transitions{{
						Target: "ReportingErrors",
						Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[Context]) Context {
							c := a.Context
							c.DirsToReport = nil
							if err, ok := a.Event.(xs.ErrorActorEvent).Error.(*FileError); ok {
								c.DirsToReport = err.DirsToReport
							}
							return c
						})},
					}},
				}},
			},
			{
				Key:   "ReportingErrors",
				Entry: xs.Actions{xs.ActionRef{Type: "emailErrors"}},
				On:    map[string]xs.Transitions{"RESTART": {{Target: "idle"}}},
			},
			{
				Key: "EvaluatingFiles",
				Invoke: []xs.InvokeConfig{{
					ID: "evaluatingFiles", Src: "evaluateFiles",
					Input: xs.NewExpr(func(a xs.ExprArgs[Context]) any {
						return EvaluationInput{a.Context.DirsToEvaluate, a.Context.AcceptedFileTypes}
					}),
					OnDone: xs.Transitions{{
						Target: "MovingFiles",
						Actions: xs.Actions{xs.Assign(func(a xs.AssignArgs[Context]) Context {
							c := a.Context
							c.DirsToMove = a.Event.(xs.DoneActorEvent).Output.(EvaluationResult).DirsToMove
							return c
						})},
					}},
				}},
			},
			{
				Key: "MovingFiles",
				Invoke: []xs.InvokeConfig{{
					ID: "moveFiles", Src: "moveFiles",
					Input:   xs.NewExpr(func(a xs.ExprArgs[Context]) any { return MoveInput{a.Context.DirsToMove, a.Context.DestinationPath} }),
					OnDone:  xs.Transitions{{Target: "idle"}},
					OnError: xs.Transitions{{Target: "ReportingErrors"}},
				}},
			},
		},
	})
}
