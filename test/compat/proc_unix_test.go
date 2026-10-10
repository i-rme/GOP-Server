//go:build unix

package compat

import (
	"os/exec"
	"syscall"
)

// The server starts `go run` and script binaries; run it in its own process
// group so a hung script is cleaned up together with the server.
func setProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func killProcessGroup(cmd *exec.Cmd) {
	if cmd.Process != nil {
		syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
}
