//go:build !linux && !windows

package lock

import (
	"os"
	"os/exec"
)

type platformLock struct {
	file *os.File
}

func acquirePlatform(path string) (*platformLock, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return &platformLock{file: file}, nil
}

func (held *platformLock) Close() error {
	return held.file.Close()
}

func configureDetached(command *exec.Cmd) {}
