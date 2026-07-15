//go:build linux

package lock

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

type platformLock struct {
	file *os.File
}

func acquirePlatform(path string) (*platformLock, error) {
	file, err := os.Open(path)
	if err != nil {
		file, err = os.OpenFile(path, os.O_WRONLY, 0)
	}
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("文件已被锁定或占用: %w", err)
	}
	return &platformLock{file: file}, nil
}

func (held *platformLock) Close() error {
	_ = syscall.Flock(int(held.file.Fd()), syscall.LOCK_UN)
	return held.file.Close()
}

func configureDetached(command *exec.Cmd) {
	command.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}
