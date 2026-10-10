//go:build unix

package worker

import (
	"errors"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"syscall"
)

type identity = syscall.Credential

// nobody is the identity of the unprivileged nobody user; switching to it needs root
func nobody() (*identity, error) {
	if os.Geteuid() != 0 {
		return nil, errors.New("the server must run as root to switch scripts to nobody")
	}
	u, err := user.Lookup("nobody")
	if err != nil {
		return nil, err
	}
	uid, err := strconv.ParseUint(u.Uid, 10, 32)
	if err != nil {
		return nil, err
	}
	gid, err := strconv.ParseUint(u.Gid, 10, 32)
	if err != nil {
		return nil, err
	}
	return &identity{Uid: uint32(uid), Gid: uint32(gid)}, nil
}

// isolate starts cmd in its own process group, killed as a whole when cmd is cancelled,
// and as id when it is set
func isolate(cmd *exec.Cmd, id *identity) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true, Credential: id}
	cmd.Cancel = func() error {
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
}
