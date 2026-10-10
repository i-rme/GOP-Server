// Package worker builds scripts with the go command and runs the binaries,
// each under a timeout and, optionally, as the nobody user.
package worker

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ErrTimeout means a build or a run took longer than it was allowed to
var ErrTimeout = errors.New("timed out")

// BuildError is a build the go command rejected, usually a compile error
type BuildError struct {
	Output string // the exit status, then the go command's messages
}

func (e *BuildError) Error() string { return e.Output }

// Options configure a Worker
type Options struct {
	CacheRoot        string        // build caches for nobody go here
	BuildTimeout     time.Duration // limit for one go build
	ExecutionTimeout time.Duration // limit for one run of a script
	AsNobody         bool          // build and run as the nobody user
}

// Worker builds and runs scripts
type Worker struct {
	options  Options
	identity *identity // nil to keep the server's own user
	buildEnv []string
	workDir  string // the server's working directory, stripped from compiler messages
}

// New checks that scripts can be built and run as configured
func New(options Options) (*Worker, error) {
	if _, err := exec.LookPath("go"); err != nil {
		return nil, fmt.Errorf("scripts are built with the go command, which is not on the PATH: %w", err)
	}
	workDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	w := &Worker{
		options: options,
		workDir: workDir,
		// -modcacherw keeps a module cache under CacheRoot removable when CacheRoot is reset
		buildEnv: append(os.Environ(), "GOWORK=off", "GOTOOLCHAIN=local", "GOFLAGS=-modcacherw"),
	}

	if options.AsNobody {
		if w.identity, err = nobody(); err != nil {
			return nil, fmt.Errorf("RunScriptsAsNobody: %w", err)
		}
		// nobody cannot write to the server user's Go caches, so it gets its own
		home := filepath.Join(options.CacheRoot, ".nobody")
		if err := w.mkdir(home); err != nil {
			return nil, err
		}
		w.buildEnv = append(w.buildEnv, "HOME="+home, "GOCACHE="+filepath.Join(home, "cache"), "GOPATH="+filepath.Join(home, "go"))
	}
	return w, nil
}

// Build writes files into dir and compiles main there to binary, which appears atomically
func (w *Worker) Build(dir, binary string, files map[string]string, main string) error {
	if err := w.mkdir(dir); err != nil {
		return err
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			return err
		}
	}

	partial := binary + ".partial"
	// -s -w leaves out the symbol table and DWARF, which panics do not need and which make linking slower
	_, stderr, err := w.exec(dir, w.buildEnv, w.options.BuildTimeout, "go", "build", "-ldflags=-s -w", "-o", partial, main)
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return &BuildError{Output: err.Error() + "\n" + strings.TrimSuffix(w.relative(dir, stderr), "\n")}
	}
	if err != nil {
		return err
	}
	return os.Rename(partial, binary)
}

// Result is what a script printed
type Result struct {
	Stdout, Stderr string
}

// Run runs binary with the request parameters, in the server's working directory.
// A script that exits with an error returns an *exec.ExitError together with its output;
// one that runs too long returns ErrTimeout.
func (w *Worker) Run(binary, parameters string) (Result, error) {
	stdout, stderr, err := w.exec("", nil, w.options.ExecutionTimeout, binary, "-PARAMETERS="+parameters)
	return Result{Stdout: stdout, Stderr: stderr}, err
}

// relative rewrites the paths in the go command's messages, which are absolute or relative to
// the build directory dir, to be relative to the server's working directory like configured paths
func (w *Worker) relative(dir, messages string) string {
	sep := string(filepath.Separator)
	if up, err := filepath.Rel(dir, w.workDir); err == nil {
		messages = strings.ReplaceAll(messages, up+sep, "")
	}
	return strings.ReplaceAll(messages, w.workDir+sep, "")
}

// mkdir creates dir, owned by the user scripts run as
func (w *Worker) mkdir(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if w.identity != nil {
		return os.Chown(dir, int(w.identity.Uid), int(w.identity.Gid))
	}
	return nil
}

// exec is the one place a process starts: an argument list (no shell), its own
// process group killed as a whole on timeout, and output captured up to maxOutput
func (w *Worker) exec(dir string, env []string, timeout time.Duration, name string, args ...string) (stdout, stderr string, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Env = env
	var out, errOut capped
	cmd.Stdout, cmd.Stderr = &out, &errOut
	cmd.WaitDelay = time.Second // don't wait on pipes held open by orphaned grandchildren
	isolate(cmd, w.identity)

	err = cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		err = ErrTimeout
	}
	return out.String(), errOut.String(), err
}
