package mediascanner

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
	require.NoError(t, os.WriteFile(path, []byte(body), 0644))
}

// Go-only helper semantics test; upstream declares the helper but has no test.
// https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-media-scanner/src/fileHandlers.ts#L18
// Related workflow recorder: scripts/trace/workflow-media-scanner/scan-error.ts; filesystem assertions are Go-only.
func TestScanDirectories(t *testing.T) {
	root := t.TempDir()
	h := DefaultHandlers(io.Discard)
	_, err := h.ScanDirectories(root)
	require.EqualError(t, err, "Unable to scan directory: No valid directories found")
	writeFile(t, filepath.Join(root, "loose.mp4"), "loose")
	movie := filepath.Join(root, "movie")
	require.NoError(t, os.Mkdir(movie, 0755))
	link := filepath.Join(root, "linked")
	require.NoError(t, os.Symlink(movie, link))
	dirs, err := h.ScanDirectories(root)
	require.NoError(t, err)
	require.Equal(t, []string{link, movie}, dirs)
	require.NoError(t, os.Symlink(filepath.Join(root, "missing"), filepath.Join(root, "broken")))
	_, err = h.ScanDirectories(root)
	require.ErrorContains(t, err, "Unable to scan directory:")
	_, err = h.ScanDirectories(filepath.Join(root, "absent"))
	require.Error(t, err)
}

// Go-only helper semantics test; upstream declares the helper but has no test.
// https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-media-scanner/src/fileHandlers.ts#L42
// Related workflow recorder: scripts/trace/workflow-media-scanner/permissions-error.ts; access assertions are Go-only.
func TestPermissions(t *testing.T) {
	h := DefaultHandlers(io.Discard)
	root := t.TempDir()
	result, err := h.CheckFilePermissions([]string{root, filepath.Join(root, "missing")})
	require.NoError(t, err)
	require.Equal(t, []string{root}, result.DirsToEvaluate)
	require.Equal(t, []string{filepath.Join(root, "missing")}, result.DirsToReport)
	h.Access = func(string) error { return os.ErrPermission }
	_, err = h.CheckFilePermissions([]string{"denied"})
	var outer *FileError
	require.ErrorAs(t, err, &outer)
	require.Equal(t, "Error checking file permissions", outer.Message)
	require.Nil(t, outer.DirsToReport)
	inner := outer.Cause.(*FileError)
	require.Equal(t, []string{"denied"}, *inner.DirsToReport)
	_, err = h.CheckFilePermissions(nil)
	require.Error(t, err)
}

// Go-only helper semantics test; upstream declares the helper but has no test.
// https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-media-scanner/src/fileHandlers.ts#L76
// Related workflow recorder: scripts/trace/workflow-media-scanner/evaluate-error.ts; dimension assertions are Go-only.
func TestEvaluateFiles(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "movie")
	h := DefaultHandlers(io.Discard)
	for _, name := range []string{"a.mkv", "b.mp4", "c.mp4", "d.mp4", "e.MP4", "f.txt", "mp4"} {
		writeFile(t, filepath.Join(dir, name), "video")
	}
	called := []string{}
	h.Probe = func(_ context.Context, path string) (Dimensions, error) {
		name := filepath.Base(path)
		called = append(called, name)
		switch name {
		case "a.mkv":
			return Dimensions{3840, 2160}, nil
		case "b.mp4":
			return Dimensions{1920, 2160}, nil
		case "c.mp4":
			return Dimensions{3840, 1080}, nil
		case "d.mp4":
			return Dimensions{1921, 1081}, nil
		default:
			return Dimensions{3840, 2160}, nil
		}
	}
	result, err := h.EvaluateFiles(context.Background(), []string{filepath.Join(root, "missing"), dir}, []string{"mp4", "mkv"})
	require.NoError(t, err)
	require.Equal(t, []string{filepath.Join(dir, "a.mkv"), filepath.Join(dir, "d.mp4"), filepath.Join(dir, "mp4")}, result.DirsToMove)
	require.Equal(t, []string{"a.mkv", "b.mp4", "c.mp4", "d.mp4", "mp4"}, called)
	h.Probe = func(context.Context, string) (Dimensions, error) { return Dimensions{}, errors.New("bad media") }
	_, err = h.EvaluateFiles(context.Background(), []string{dir}, []string{"mp4", "mkv"})
	require.EqualError(t, err, "Error evaluating files")
	require.Empty(t, *err.(*FileError).Cause.(*FileError).DirsToMove)
}

// Go-only helper semantics regression; upstream has no test case.
// https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-media-scanner/src/fileHandlers.ts#L84
// Related workflow recorder: scripts/trace/workflow-media-scanner/evaluate-error.ts; skip behavior assertions are Go-only.
func TestProbeFailureSkipsRemainderOfDirectory(t *testing.T) {
	root := t.TempDir()
	first := filepath.Join(root, "first")
	second := filepath.Join(root, "second")
	for _, path := range []string{filepath.Join(first, "a.mp4"), filepath.Join(first, "b.mp4"), filepath.Join(second, "c.mp4")} {
		writeFile(t, path, "video")
	}
	h := DefaultHandlers(io.Discard)
	called := []string{}
	h.Probe = func(_ context.Context, path string) (Dimensions, error) {
		called = append(called, filepath.Base(path))
		if filepath.Base(path) == "a.mp4" {
			return Dimensions{}, errors.New("bad stream")
		}
		return Dimensions{3840, 2160}, nil
	}
	result, err := h.EvaluateFiles(context.Background(), []string{first, second}, []string{"mp4"})
	require.NoError(t, err)
	require.Equal(t, []string{"a.mp4", "c.mp4"}, called)
	require.Equal(t, []string{filepath.Join(second, "c.mp4")}, result.DirsToMove)
}

// Go-only helper semantics test; upstream declares the helper but has no test.
// https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-media-scanner/src/fileHandlers.ts#L121
// Related workflow recorder: scripts/trace/workflow-media-scanner/partial-move.ts; filesystem assertions are Go-only.
func TestMoveFilesMovesWholeParentAndReportsDuplicates(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source", "movie")
	dest := filepath.Join(root, "dest")
	writeFile(t, filepath.Join(source, "a.mp4"), "A")
	writeFile(t, filepath.Join(source, "b.mp4"), "B")
	writeFile(t, filepath.Join(source, "subtitle.srt"), "subtitle")
	writeFile(t, filepath.Join(dest, "movie", "obsolete"), "old")
	h := DefaultHandlers(io.Discard)
	result, err := h.MoveFiles([]string{filepath.Join(source, "a.mp4"), filepath.Join(source, "b.mp4")}, dest)
	require.NoError(t, err)
	require.Equal(t, "files moved with errors", result.Message)
	require.Len(t, result.Errors, 1)
	require.Equal(t, filepath.Join(source, "b.mp4"), result.Errors[0].Source)
	require.NoDirExists(t, source)
	require.FileExists(t, filepath.Join(dest, "movie", "subtitle.srt"))
	require.NoFileExists(t, filepath.Join(dest, "movie", "obsolete"))
	contents, err := os.ReadFile(filepath.Join(dest, "movie", "a.mp4"))
	require.NoError(t, err)
	require.Equal(t, "A", string(contents))
	result, err = h.MoveFiles(nil, dest)
	require.NoError(t, err)
	require.Equal(t, "all files moved successfully", result.Message)
}

// Go-only filesystem adapter regression for the upstream fs-extra boundary; no JS test case.
// https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-media-scanner/src/fileHandlers.ts#L137
// Related workflow recorder: scripts/trace/workflow-media-scanner/success.ts; copy assertions are Go-only.
func TestCopyTreePreservesFilesAndSymlinks(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	destination := filepath.Join(root, "copy")
	writeFile(t, filepath.Join(source, "nested", "video"), "content")
	require.NoError(t, os.Chmod(filepath.Join(source, "nested", "video"), 0600))
	require.NoError(t, os.Symlink("nested/video", filepath.Join(source, "link")))
	require.NoError(t, copyTree(source, destination))
	contents, err := os.ReadFile(filepath.Join(destination, "link"))
	require.NoError(t, err)
	require.Equal(t, "content", string(contents))
	info, err := os.Stat(filepath.Join(destination, "nested", "video"))
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0600), info.Mode().Perm())
	require.Error(t, MoveDirectory(source, filepath.Join(source, "child")))
	require.Error(t, MoveDirectory(source, root))
	require.FileExists(t, filepath.Join(source, "nested", "video"))
}

// Go-only subprocess adapter regression for the upstream node-ffprobe boundary; no JS test case.
// https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-media-scanner/src/fileHandlers.ts#L95
// Related workflow recorder: scripts/trace/workflow-media-scanner/success.ts; subprocess assertions are Go-only.
func TestProbeCommand(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PATH", root)
	executable := filepath.Join(root, "ffprobe")
	for _, tc := range []struct {
		name, body string
		want       Dimensions
		failure    bool
	}{
		{"first stream", "{\"streams\":[{\"width\":3840,\"height\":2160},{\"width\":1,\"height\":1}]}", Dimensions{3840, 2160}, false},
		{"audio stream", "{\"streams\":[{}]}", Dimensions{}, false},
		{"empty streams", "{\"streams\":[]}", Dimensions{}, true},
		{"bad json", "not json", Dimensions{}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.NoError(t, os.WriteFile(executable, []byte("#!/bin/sh\n[ \"$1\" = '-v' ] && [ \"$5\" = '-show_streams' ] && [ \"$6\" = 'movie.mp4' ] || exit 2\nprintf '%s' '"+tc.body+"'\n"), 0755))
			result, err := ProbeFile(context.Background(), "movie.mp4")
			if tc.failure {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.want, result)
			}
		})
	}
	require.NoError(t, os.Remove(executable))
	_, err := ProbeFile(context.Background(), "movie.mp4")
	require.Error(t, err)
}

// Go-only integration of the upstream promise actor declarations; no JS test case.
// https://github.com/statelyai/xstate/blob/38dcaffb20ec7f3cbb10e6161d8f0ebacca33701/examples/workflow-media-scanner/src/mediaScannerMachine.ts#L33
// Related workflow recorder: scripts/trace/workflow-media-scanner/success.ts; temporary filesystem assertions are Go-only.
func TestRealHelpersThroughPromiseActors(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	destination := filepath.Join(root, "destination")
	writeFile(t, filepath.Join(source, "movie", "film.mp4"), "video")
	writeFile(t, filepath.Join(source, "movie", "subtitles.srt"), "captions")
	h := DefaultHandlers(io.Discard)
	h.Probe = func(_ context.Context, path string) (Dimensions, error) {
		_, err := os.Stat(path)
		return Dimensions{3840, 2160}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	var out strings.Builder
	require.NoError(t, RunWith(ctx, &out, Input{source, destination}, DefaultActors(h)))
	require.FileExists(t, filepath.Join(destination, "movie", "film.mp4"))
	require.FileExists(t, filepath.Join(destination, "movie", "subtitles.srt"))
	require.NoDirExists(t, filepath.Join(source, "movie"))
	for _, state := range []string{"Scanning", "CheckingFilePermissions", "EvaluatingFiles", "MovingFiles", "idle"} {
		require.Contains(t, out.String(), "state: \""+state+"\"")
	}
}
