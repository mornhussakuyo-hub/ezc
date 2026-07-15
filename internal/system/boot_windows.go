//go:build windows

package system

import (
	"fmt"
	"syscall"
	"unsafe"
)

const systemBootEnvironmentInformation = 90

var ntQuerySystemInformation = syscall.NewLazyDLL("ntdll.dll").NewProc("NtQuerySystemInformation")

type bootEnvironmentInformation struct {
	BootIdentifier [16]byte
	FirmwareType   uint32
	BootFlags      uint64
}

func BootID() (string, error) {
	var information bootEnvironmentInformation
	var returnLength uint32
	status, _, _ := ntQuerySystemInformation.Call(
		systemBootEnvironmentInformation,
		uintptr(unsafe.Pointer(&information)),
		unsafe.Sizeof(information),
		uintptr(unsafe.Pointer(&returnLength)),
	)
	if status != 0 {
		return "", fmt.Errorf("读取系统启动标识失败: NTSTATUS 0x%x", status)
	}
	return fmt.Sprintf("windows-%x", information.BootIdentifier), nil
}
