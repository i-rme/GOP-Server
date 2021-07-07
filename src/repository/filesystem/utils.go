package filesystem

import (
	"path/filepath"
	"strings"
)

// Directory returns all but the last element of path if its not a directory
func Directory(filePath string) string {

	if filepath.Ext(filePath) == "" { // Probably a directory
		return filePath
	}

	return filepath.Dir(filePath)
}

// Clean returns the shortest path name equivalent to path
func Clean(filePath string) string {
	return filepath.Clean(filePath)
}

// Extension returns the extension from a file path
func Extension(filePath string) string {
	return filepath.Ext(filePath)
}

// Basename returns the last element of path
func Basename(filePath string) string {
	return filepath.Base(filePath)
}

// FilenameWithoutExtension returns the file name from a file path without extension
func FilenameWithoutExtension(filePath string) string {
	return filepath.Base(strings.TrimSuffix(filePath, filepath.Ext(filePath)))
}
