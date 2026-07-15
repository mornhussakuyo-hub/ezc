//go:build windows

package app

import (
	"strings"
	"syscall"
	"unsafe"
)

const fileAttributeHidden = 0x2

var getFileAttributesW = syscall.NewLazyDLL("kernel32.dll").NewProc("GetFileAttributesW")

func isHidden(name string) bool {
	if strings.HasPrefix(name, ".") {
		return true
	}
	pointer, err := syscall.UTF16PtrFromString(name)
	if err != nil {
		return false
	}
	attributes, _, _ := getFileAttributesW.Call(uintptr(unsafe.Pointer(pointer)))
	return attributes != ^uintptr(0) && attributes&fileAttributeHidden != 0
}
