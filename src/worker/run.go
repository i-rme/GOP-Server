package worker

import (
	"context"
	"os/exec"
	"pfg/src/handler/errors"
	"pfg/src/server/config"
	"runtime"
	"strings"
	"time"
)

// Run calls a binary with timeout returning the output
func Run(path, parameters string, seconds int) (string, bool) {

	if runtime.GOOS == "windows" || !config.RunScriptsAsNobody {
		return runGeneric(path, parameters, seconds)
	} else {
		return runLinux(path, parameters, seconds)
	}
}

// runGeneric calls a binary with timeout returning the output
func runGeneric(path, parameters string, seconds int) (string, bool) {

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(seconds)*time.Second)
	defer cancel()

	command := exec.CommandContext(ctx, path, `-PARAMETERS=`+parameters)
	outputBytes, err := command.CombinedOutput()
	output := strings.TrimSuffix(string(outputBytes), "\n")

	if ctx.Err() == context.DeadlineExceeded {
		return errors.Render(errors.ExecutionTimeout, path), false
	}

	if err != nil {
		return err.Error() + "\n" + output, false
	}

	return output, true
}

// runLinux calls a binary with timeout returning the output
func runLinux(path, parameters string, seconds int) (string, bool) {

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(seconds)*time.Second)
	defer cancel()

	// Runs as nobody
	command := exec.CommandContext(ctx, `setpriv`, `--no-new-privs`, `--reuid=nobody`, path, `-PARAMETERS=`+parameters)

	outputBytes, err := command.CombinedOutput()
	output := strings.TrimSuffix(string(outputBytes), "\n")

	if ctx.Err() == context.DeadlineExceeded {
		return errors.Render(errors.ExecutionTimeout, path), false
	}

	if err != nil {
		return err.Error() + "\n" + output, false
	}

	return output, true
}

// runWithoutTimeout calls a binary and returns the ouput (DEPRECATED)
func runWithoutTimeout(path string) (string, bool) {
	command := exec.Command(path)
	outputBytes, err := command.CombinedOutput()
	output := strings.TrimSuffix(string(outputBytes), "\n")

	if err != nil { // TO-DO handle the error
		return err.Error() + "\n" + output, false
	}

	return output, true
}
