// Package filesystem holds the few file helpers the server shares.
package filesystem

import (
	"errors"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// ErrOutsideRoot means a request path leads out of the document root, through .. or a symlink
var ErrOutsideRoot = errors.New("path is outside the document root")

// RealDir returns dir as an absolute path with every symlink resolved, the form ResolveInRoot takes
func RealDir(dir string) (string, error) {
	real, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return "", err
	}
	return filepath.Abs(real)
}

// ResolveInRoot returns the real path a URL path names inside realRoot, a directory
// as RealDir returns it. Every request path goes through it. It fails when the path
// does not exist or, after following symlinks, is not inside the root.
func ResolveInRoot(realRoot, urlPath string) (string, error) {
	// Cleaning against "/" drops any .. that would climb above the root
	joined := filepath.Join(realRoot, filepath.FromSlash(path.Clean("/"+urlPath)))
	real, err := filepath.EvalSymlinks(joined)
	if err != nil {
		return "", err
	}
	if real != realRoot && !strings.HasPrefix(real, realRoot+string(filepath.Separator)) {
		return "", ErrOutsideRoot
	}
	return real, nil
}

// Exists reports whether path is an existing regular file (not a directory)
func Exists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// ResetDir empties dir, creating it if needed
func ResetDir(dir string) error {
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	return os.MkdirAll(dir, 0o755)
}
