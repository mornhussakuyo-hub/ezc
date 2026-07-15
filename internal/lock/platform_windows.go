//go:build windows

package lock

import (
	"fmt"
	"os/exec"
	"syscall"
	"unsafe"
)

const (
	genericRead             = 0x80000000
	openExisting            = 3
	fileFlagBackupSemantics = 0x02000000
	createNewProcessGroup   = 0x00000200
	detachedProcess         = 0x00000008
)

var (
	kernel32      = syscall.NewLazyDLL("kernel32.dll")
	createFileW   = kernel32.NewProc("CreateFileW")
	closeHandle   = kernel32.NewProc("CloseHandle")
	invalidHandle = ^uintptr(0)
)

type platformLock struct {
	handle syscall.Handle
}

func acquirePlatform(path string) (*platformLock, error) {
	pathPointer, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	handle, _, callErr := createFileW.Call(
		uintptr(unsafe.Pointer(pathPointer)),
		genericRead,
		0,
		0,
		openExisting,
		fileFlagBackupSemantics,
		0,
	)
	if handle == invalidHandle {
		return nil, fmt.Errorf("文件已被锁定或占用: %w", callErr)
	}
	return &platformLock{handle: syscall.Handle(handle)}, nil
}

func (held *platformLock) Close() error {
	result, _, callErr := closeHandle.Call(uintptr(held.handle))
	if result == 0 {
		return callErr
	}
	return nil
}

func configureDetached(command *exec.Cmd) {
	command.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: createNewProcessGroup | detachedProcess,
		HideWindow:    true,
	}
}
