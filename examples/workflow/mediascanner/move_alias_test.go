package mediascanner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// Go regression: overwrite checks must account for symlinks in parent paths.
func TestMoveDirectoryThroughAlias(t *testing.T) {
	for _, relation := range []string{"same", "ancestor", "descendant", "missing descendant"} {
		t.Run(relation, func(t *testing.T) {
			root := t.TempDir()
			real := filepath.Join(root, "real")
			movie := filepath.Join(real, "movie")
			file := filepath.Join(movie, "video.mp4")
			writeFile(t, file, "video")
			alias := filepath.Join(root, "alias")
			require.NoError(t, os.Symlink(real, alias))
			source, destination := filepath.Join(alias, "movie"), movie
			switch relation {
			case "ancestor":
				destination = real
			case "descendant":
				source, destination = real, filepath.Join(alias, "movie")
			case "missing descendant":
				source, destination = real, filepath.Join(alias, "new", "movie")
			}
			err := MoveDirectory(source, destination)
			if relation == "same" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
			contents, err := os.ReadFile(file)
			require.NoError(t, err, "a rejected or no-op move must preserve the source")
			require.Equal(t, "video", string(contents))
		})
	}
}

// Go regression: case aliases on insensitive volumes must not bypass overwrite checks.
func TestMoveDirectoryThroughCaseAlias(t *testing.T) {
	for _, relation := range []string{"same", "ancestor", "descendant", "missing descendant"} {
		t.Run(relation, func(t *testing.T) {
			root := t.TempDir()
			real := filepath.Join(root, "MixedCase")
			source := filepath.Join(real, "Movie")
			file := filepath.Join(source, "video.mp4")
			writeFile(t, file, "video")
			alias := filepath.Join(root, "mixedcase", "movie")
			aliasInfo, err := os.Stat(alias)
			if os.IsNotExist(err) {
				t.Skip("filesystem is case-sensitive")
			}
			require.NoError(t, err)
			sourceInfo, err := os.Stat(source)
			require.NoError(t, err)
			require.True(t, os.SameFile(sourceInfo, aliasInfo))
			destination := alias
			switch relation {
			case "ancestor":
				destination = filepath.Dir(alias)
			case "descendant":
				destination = filepath.Join(alias, "child")
			case "missing descendant":
				destination = filepath.Join(alias, "missing", "child")
			}
			err = MoveDirectory(source, destination)
			if relation == "same" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
			contents, err := os.ReadFile(file)
			require.NoError(t, err, "a rejected or no-op move must preserve the source")
			require.Equal(t, "video", string(contents))
		})
	}
}

// Go regression: resolving parents must not dereference a symlink being moved or replaced.
func TestMoveDirectoryPreservesLeafSymlinkSemantics(t *testing.T) {
	root := t.TempDir()
	real := filepath.Join(root, "real")
	writeFile(t, filepath.Join(real, "video.mp4"), "video")
	link := filepath.Join(root, "link")
	require.NoError(t, os.Symlink(real, link))
	moved := filepath.Join(root, "nested", "moved-link")
	require.NoError(t, MoveDirectory(link, moved))
	target, err := os.Readlink(moved)
	require.NoError(t, err)
	require.Equal(t, real, target)
	require.FileExists(t, filepath.Join(real, "video.mp4"))

	source := filepath.Join(root, "source")
	writeFile(t, filepath.Join(source, "replacement.mp4"), "replacement")
	require.NoError(t, MoveDirectory(source, moved))
	require.FileExists(t, filepath.Join(moved, "replacement.mp4"))
	require.FileExists(t, filepath.Join(real, "video.mp4"))
}
