//go:build windows

package ui

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

const (
	enableEchoInput                 = 0x0004
	enableLineInput                 = 0x0002
	enableVirtualTerminalInput      = 0x0200
	enableVirtualTerminalProcessing = 0x0004
)

var (
	consoleKernel32            = syscall.NewLazyDLL("kernel32.dll")
	getConsoleMode             = consoleKernel32.NewProc("GetConsoleMode")
	setConsoleMode             = consoleKernel32.NewProc("SetConsoleMode")
	getConsoleScreenBufferInfo = consoleKernel32.NewProc("GetConsoleScreenBufferInfo")
)

type coordinate struct {
	x int16
	y int16
}

type smallRectangle struct {
	left   int16
	top    int16
	right  int16
	bottom int16
}

type consoleScreenBufferInformation struct {
	size              coordinate
	cursorPosition    coordinate
	attributes        uint16
	window            smallRectangle
	maximumWindowSize coordinate
}

type terminalState struct {
	inputHandle  syscall.Handle
	outputHandle syscall.Handle
	inputMode    uint32
	outputMode   uint32
}

func enterRaw() (*terminalState, error) {
	inputHandle := syscall.Handle(os.Stdin.Fd())
	outputHandle := syscall.Handle(os.Stdout.Fd())
	var inputMode uint32
	if result, _, callErr := getConsoleMode.Call(uintptr(inputHandle), uintptr(unsafe.Pointer(&inputMode))); result == 0 {
		return nil, fmt.Errorf("标准输入不是终端: %w", callErr)
	}
	var outputMode uint32
	if result, _, callErr := getConsoleMode.Call(uintptr(outputHandle), uintptr(unsafe.Pointer(&outputMode))); result == 0 {
		return nil, fmt.Errorf("标准输出不是终端: %w", callErr)
	}
	newInputMode := inputMode &^ (enableEchoInput | enableLineInput)
	newInputMode |= enableVirtualTerminalInput
	if result, _, callErr := setConsoleMode.Call(uintptr(inputHandle), uintptr(newInputMode)); result == 0 {
		return nil, fmt.Errorf("切换终端输入模式: %w", callErr)
	}
	newOutputMode := outputMode | enableVirtualTerminalProcessing
	if result, _, callErr := setConsoleMode.Call(uintptr(outputHandle), uintptr(newOutputMode)); result == 0 {
		_, _, _ = setConsoleMode.Call(uintptr(inputHandle), uintptr(inputMode))
		return nil, fmt.Errorf("切换终端输出模式: %w", callErr)
	}
	return &terminalState{inputHandle: inputHandle, outputHandle: outputHandle, inputMode: inputMode, outputMode: outputMode}, nil
}

func leaveRaw(state *terminalState) {
	if state == nil {
		return
	}
	_, _, _ = setConsoleMode.Call(uintptr(state.inputHandle), uintptr(state.inputMode))
	_, _, _ = setConsoleMode.Call(uintptr(state.outputHandle), uintptr(state.outputMode))
	fmt.Print("\x1b[?25h")
}

func terminalDimensions() (int, int) {
	var information consoleScreenBufferInformation
	if result, _, _ := getConsoleScreenBufferInfo.Call(uintptr(os.Stdout.Fd()), uintptr(unsafe.Pointer(&information))); result != 0 {
		width := int(information.window.right-information.window.left) + 1
		height := int(information.window.bottom-information.window.top) + 1
		if width > 0 && height > 0 {
			return width, height
		}
	}
	return 80, 24
}
