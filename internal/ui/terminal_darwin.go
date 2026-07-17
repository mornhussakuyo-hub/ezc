//go:build darwin

package ui

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

type terminalState struct {
	termios syscall.Termios
}

type windowSize struct {
	rows    uint16
	columns uint16
	xpixel  uint16
	ypixel  uint16
}

func enterRaw() (*terminalState, error) {
	fileDescriptor := os.Stdin.Fd()
	var original syscall.Termios
	if _, _, errno := syscall.Syscall6(syscall.SYS_IOCTL, fileDescriptor, syscall.TIOCGETA, uintptr(unsafe.Pointer(&original)), 0, 0, 0); errno != 0 {
		return nil, fmt.Errorf("标准输入不是终端: %w", errno)
	}

	raw := makeRawTerminalSettings(original)
	if _, _, errno := syscall.Syscall6(syscall.SYS_IOCTL, fileDescriptor, syscall.TIOCSETA, uintptr(unsafe.Pointer(&raw)), 0, 0, 0); errno != 0 {
		return nil, fmt.Errorf("切换终端模式: %w", errno)
	}
	return &terminalState{termios: original}, nil
}

func leaveRaw(state *terminalState) {
	if state == nil {
		return
	}
	// Flush unread input while restoring canonical mode so Darwin does not
	// leave the PENDIN flag set after the menu exits.
	_, _, _ = syscall.Syscall6(syscall.SYS_IOCTL, os.Stdin.Fd(), syscall.TIOCSETAF, uintptr(unsafe.Pointer(&state.termios)), 0, 0, 0)
	fmt.Print("\x1b[?25h")
}

func makeRawTerminalSettings(original syscall.Termios) syscall.Termios {
	raw := original
	raw.Iflag &^= syscall.BRKINT | syscall.ICRNL | syscall.INPCK | syscall.ISTRIP | syscall.IXON
	raw.Lflag &^= syscall.ECHO | syscall.ICANON | syscall.IEXTEN | syscall.ISIG
	raw.Cflag |= syscall.CS8
	raw.Cc[syscall.VMIN] = 1
	raw.Cc[syscall.VTIME] = 0
	return raw
}

func terminalDimensions() (int, int) {
	var size windowSize
	if _, _, errno := syscall.Syscall6(syscall.SYS_IOCTL, os.Stdout.Fd(), syscall.TIOCGWINSZ, uintptr(unsafe.Pointer(&size)), 0, 0, 0); errno == 0 && size.columns > 0 && size.rows > 0 {
		return int(size.columns), int(size.rows)
	}
	return 80, 24
}
