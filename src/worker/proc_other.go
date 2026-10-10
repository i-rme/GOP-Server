//go:build !unix

package worker

import (
	"errors"
	"os/exec"
)

type identity struct{ Uid, Gid uint32 }

func nobody() (*identity, error) {
	return nil, errors.New("running scripts as nobody is only supported on Linux and other Unix systems")
}

func isolate(cmd *exec.Cmd, id *identity) {}
