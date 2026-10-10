// Package filesystem holds the few file helpers the server shares.
package filesystem

import (
	"os"
	"path/filepath"
)

// Exists reports whether path is an existing regular file (not a directory)
func Exists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// DirectoryExists reports whether path is an existing directory
func DirectoryExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// Directory returns urlPath itself when it has no extension, as directories usually don't,
// and its parent otherwise
func Directory(urlPath string) string {
	if filepath.Ext(urlPath) == "" {
		return urlPath
	}
	return filepath.Dir(urlPath)
}

// WriteTemp writes content to a new file in dir and returns its path
func WriteTemp(dir, pattern, content string) (string, error) {
	file, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return "", err
	}
	_, err = file.WriteString(content)
	if closeErr := file.Close(); err == nil { // Windows keeps an open file locked
		err = closeErr
	}
	if err != nil {
		os.Remove(file.Name())
		return "", err
	}
	return file.Name(), nil
}

// ResetDir empties dir, creating it if needed
func ResetDir(dir string) error {
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	return os.MkdirAll(dir, 0o755)
}
