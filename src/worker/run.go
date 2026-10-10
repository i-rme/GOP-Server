// Package worker runs the go tool and compiled scripts.
package worker

import (
	"context"
	"errors"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/i-rme/gop-server/src/parser"
)

// ErrTimeout means a script ran longer than it was allowed to
var ErrTimeout = errors.New("execution timeout")

// BuildAndRun runs a Go file with `go run --work`. It returns the program's output
// and the work directory holding the binary, which is empty when the go command
// itself failed. On failure the output starts with the error.
func BuildAndRun(path, parameters string) (output, workDir string, err error) {
	command := exec.Command("go", "run", "--work", path, "-PARAMETERS="+parameters)
	combined, err := command.CombinedOutput()
	output, workDir = parser.SplitOutputAndWorkDirectory(strings.TrimSuffix(string(combined), "\n"))
	if err != nil {
		return err.Error() + "\n" + output, workDir, err
	}
	return output, workDir, nil
}

// Run runs a compiled script for at most timeout, as nobody when asNobody is set
// on systems that have setpriv. On failure the output starts with the error.
func Run(binary, parameters string, timeout time.Duration, asNobody bool) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	args := []string{binary, "-PARAMETERS=" + parameters}
	if asNobody && runtime.GOOS != "windows" {
		args = append([]string{"setpriv", "--no-new-privs", "--reuid=nobody"}, args...)
	}
	combined, err := exec.CommandContext(ctx, args[0], args[1:]...).CombinedOutput()
	output := strings.TrimSuffix(string(combined), "\n")

	if ctx.Err() == context.DeadlineExceeded {
		return output, ErrTimeout
	}
	if err != nil {
		return err.Error() + "\n" + output, err
	}
	return output, nil
}
