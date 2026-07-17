//go:build darwin

package ui

import (
	"syscall"
	"testing"
)

func TestMakeRawTerminalSettings(t *testing.T) {
	original := syscall.Termios{
		Iflag: syscall.BRKINT | syscall.ICRNL | syscall.INPCK | syscall.ISTRIP | syscall.IXON,
		Lflag: syscall.ECHO | syscall.ICANON | syscall.IEXTEN | syscall.ISIG,
	}
	original.Cc[syscall.VMIN] = 7
	original.Cc[syscall.VTIME] = 9

	raw := makeRawTerminalSettings(original)
	if raw.Iflag&(syscall.BRKINT|syscall.ICRNL|syscall.INPCK|syscall.ISTRIP|syscall.IXON) != 0 {
		t.Fatalf("raw input flags were not cleared: %#x", raw.Iflag)
	}
	if raw.Lflag&(syscall.ECHO|syscall.ICANON|syscall.IEXTEN|syscall.ISIG) != 0 {
		t.Fatalf("raw local flags were not cleared: %#x", raw.Lflag)
	}
	if raw.Cflag&syscall.CS8 == 0 {
		t.Fatalf("CS8 was not enabled: %#x", raw.Cflag)
	}
	if raw.Cc[syscall.VMIN] != 1 || raw.Cc[syscall.VTIME] != 0 {
		t.Fatalf("unexpected read timing: VMIN=%d VTIME=%d", raw.Cc[syscall.VMIN], raw.Cc[syscall.VTIME])
	}
	if original.Cc[syscall.VMIN] != 7 || original.Cc[syscall.VTIME] != 9 {
		t.Fatal("original terminal settings were modified")
	}
}
