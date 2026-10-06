package mediascanner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
)

type Dimensions struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// Handlers injects the probe, access check, directory mover, and log destination.
type Handlers struct {
	Probe  func(context.Context, string) (Dimensions, error)
	Access func(string) error
	Move   func(string, string) error
	Log    io.Writer
}

func DefaultHandlers(w io.Writer) Handlers {
	return Handlers{Probe: ProbeFile, Access: func(path string) error { return syscall.Access(path, 6) }, Move: MoveDirectory, Log: w}
}

type FileError struct {
	Message      string    `json:"message"`
	Cause        error     `json:"error,omitempty"`
	DirsToReport *[]string `json:"dirsToReport,omitempty"`
	DirsToMove   *[]string `json:"dirsToMove,omitempty"`
}

func (e *FileError) Error() string { return e.Message }
func (e *FileError) Unwrap() error { return e.Cause }

func (h Handlers) ScanDirectories(basePath string) ([]string, error) {
	entries, err := os.ReadDir(basePath)
	if err != nil {
		return nil, fmt.Errorf("Unable to scan directory: %w", err)
	}
	dirs := []string{}
	for _, entry := range entries {
		path := basePath + "/" + entry.Name()
		stat, err := os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("Unable to scan directory: %w", err)
		}
		if stat.IsDir() {
			dirs = append(dirs, path)
		}
	}
	if len(dirs) == 0 {
		return nil, errors.New("Unable to scan directory: No valid directories found")
	}
	return dirs, nil
}

func (h Handlers) CheckFilePermissions(dirs []string) (PermissionResult, error) {
	fmt.Fprintln(h.Log, "checking directory permissions...")
	result := PermissionResult{DirsToEvaluate: []string{}, DirsToReport: []string{}}
	for _, dir := range dirs {
		if err := h.Access(dir); err != nil {
			fmt.Fprintf(h.Log, "cannot access directory %s. Error: %v\n", dir, err)
			result.DirsToReport = append(result.DirsToReport, dir)
		} else {
			fmt.Fprintf(h.Log, "directory %s is accessible\n", dir)
			result.DirsToEvaluate = append(result.DirsToEvaluate, dir)
		}
	}
	if len(result.DirsToEvaluate) == 0 {
		return PermissionResult{}, &FileError{Message: "Error checking file permissions", Cause: &FileError{Message: "No accessible files found to move", DirsToReport: &result.DirsToReport}}
	}
	return result, nil
}

func (h Handlers) EvaluateFiles(ctx context.Context, dirs, types []string) (EvaluationResult, error) {
	fmt.Fprintln(h.Log, "checking files in directories...")
	result := EvaluationResult{DirsToMove: []string{}}
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			fmt.Fprintf(h.Log, "error reading directory %s: %v\n", dir, err)
			continue
		}
		for _, entry := range entries {
			parts := strings.Split(entry.Name(), ".")
			extension := parts[len(parts)-1]
			if extension == "" || !slices.Contains(types, extension) {
				continue
			}
			path := filepath.Join(dir, entry.Name())
			dimensions, err := h.Probe(ctx, path)
			if err != nil {
				fmt.Fprintf(h.Log, "error reading directory %s: %v\n", dir, err)
				break
			}
			if dimensions.Width > 1920 && dimensions.Height > 1080 {
				fmt.Fprintf(h.Log, "file %s is greater than 1080p. Assuming 4K\n", entry.Name())
				result.DirsToMove = append(result.DirsToMove, path)
			}
		}
	}
	if len(result.DirsToMove) == 0 {
		return EvaluationResult{}, &FileError{Message: "Error evaluating files", Cause: &FileError{Message: "No files found to move", DirsToMove: &result.DirsToMove}}
	}
	return result, nil
}

type MoveFailure struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Error       error  `json:"error"`
}
type MoveResult struct {
	Message string        `json:"message"`
	Errors  []MoveFailure `json:"errors,omitempty"`
}

func (h Handlers) MoveFiles(files []string, destination string) (MoveResult, error) {
	fmt.Fprintln(h.Log, "moving files...")
	result := MoveResult{Message: "all files moved successfully"}
	for _, file := range files {
		parent := filepath.Dir(file)
		target := filepath.Join(destination, filepath.Base(parent))
		if err := h.Move(parent, target); err != nil {
			fmt.Fprintf(h.Log, "failed to move %s to %s. Error: %v\n", file, target, err)
			result.Errors = append(result.Errors, MoveFailure{file, target, err})
		} else {
			fmt.Fprintf(h.Log, "moved %s to %s successfully\n", file, target)
		}
	}
	if len(result.Errors) > 0 {
		result.Message = "files moved with errors"
	}
	return result, nil
}

// ProbeFile reads the first ffprobe stream, matching node-ffprobe's default stream selection.
func ProbeFile(ctx context.Context, path string) (Dimensions, error) {
	raw, err := exec.CommandContext(ctx, "ffprobe", "-v", "quiet", "-print_format", "json", "-show_streams", path).Output()
	if err != nil {
		return Dimensions{}, err
	}
	var result struct {
		Streams []Dimensions `json:"streams"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return Dimensions{}, err
	}
	if len(result.Streams) == 0 {
		return Dimensions{}, errors.New("ffprobe returned no streams")
	}
	return result.Streams[0], nil
}

// MoveDirectory overwrites the destination and supports moves between filesystems.
func MoveDirectory(source, destination string) error {
	source, err := filepath.Abs(source)
	if err != nil {
		return err
	}
	destination, err = filepath.Abs(destination)
	if err != nil {
		return err
	}
	// Resolve parent aliases before deleting anything, but move a leaf symlink
	// itself rather than the directory it points to.
	source, err = resolveMovePath(source)
	if err != nil {
		return err
	}
	destination, err = resolveMovePath(destination)
	if err != nil {
		return err
	}
	if source == destination {
		return nil
	}
	sourceInfo, err := os.Lstat(source)
	if err != nil {
		return err
	}
	destinationInfo, err := os.Lstat(destination)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if destinationInfo != nil && os.SameFile(sourceInfo, destinationInfo) {
		return nil
	}
	if sourceInfo.IsDir() {
		inside, err := directoryInParents(sourceInfo, filepath.Dir(destination))
		if err != nil {
			return err
		}
		if inside {
			return errors.New("cannot move a directory into itself")
		}
	}
	if destinationInfo != nil && destinationInfo.IsDir() {
		inside, err := directoryInParents(destinationInfo, filepath.Dir(source))
		if err != nil {
			return err
		}
		if inside {
			return errors.New("cannot overwrite an ancestor of the source directory")
		}
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
		return err
	}
	if err := os.RemoveAll(destination); err != nil {
		return err
	}
	err = os.Rename(source, destination)
	if !errors.Is(err, syscall.EXDEV) {
		return err
	}
	if err := copyTree(source, destination); err != nil {
		return err
	}
	return os.RemoveAll(source)
}

func resolveMovePath(path string) (string, error) {
	parent, err := resolveMoveDirectory(filepath.Dir(path))
	if err != nil {
		return "", err
	}
	return filepath.Join(parent, filepath.Base(path)), nil
}

func resolveMoveDirectory(path string) (string, error) {
	resolved, err := filepath.EvalSymlinks(path)
	if err == nil {
		return resolved, nil
	}
	parent := filepath.Dir(path)
	if !errors.Is(err, os.ErrNotExist) || parent == path {
		return "", err
	}
	resolved, err = resolveMoveDirectory(parent)
	if err != nil {
		return "", err
	}
	return filepath.Join(resolved, filepath.Base(path)), nil
}

func directoryInParents(directory os.FileInfo, path string) (bool, error) {
	for {
		info, err := os.Stat(path)
		if err == nil && os.SameFile(directory, info) {
			return true, nil
		}
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return false, err
		}
		parent := filepath.Dir(path)
		if parent == path {
			return false, nil
		}
		path = parent
	}
}

func copyTree(source, destination string) error {
	info, err := os.Lstat(source)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(source)
		if err != nil {
			return err
		}
		return os.Symlink(target, destination)
	}
	if info.IsDir() {
		if err := os.Mkdir(destination, info.Mode().Perm()|0700); err != nil {
			return err
		}
		entries, err := os.ReadDir(source)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if err := copyTree(filepath.Join(source, entry.Name()), filepath.Join(destination, entry.Name())); err != nil {
				return err
			}
		}
		return os.Chmod(destination, info.Mode().Perm())
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("unsupported file mode %s", info.Mode())
	}
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	output, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, info.Mode().Perm())
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(output, input)
	closeErr := output.Close()
	return errors.Join(copyErr, closeErr)
}
