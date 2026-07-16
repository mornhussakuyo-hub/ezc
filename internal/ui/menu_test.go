package ui

import (
	"bufio"
	"strings"
	"testing"
)

func TestFitAndPadUsesTerminalDisplayWidth(t *testing.T) {
	result := fitAndPad("目录/file.txt", 16)
	if width := displayWidth(result); width != 16 {
		t.Fatalf("expected display width 16, got %d for %q", width, result)
	}
}

func TestFitAndPadTruncatesLongPaths(t *testing.T) {
	result := fitAndPad("一个非常长的中文目录名称/file.txt", 12)
	if width := displayWidth(result); width != 12 {
		t.Fatalf("expected display width 12, got %d for %q", width, result)
	}
	if !strings.Contains(result, "…") {
		t.Fatalf("expected ellipsis in %q", result)
	}
}

func TestReadKeyTreatsQAsSearchInputAndEscapeAsCancel(t *testing.T) {
	key, err := readKey(bufio.NewReader(strings.NewReader("q")))
	if err != nil || key != "q" {
		t.Fatalf("expected q search input, got %q, %v", key, err)
	}

	key, err = readKey(bufio.NewReader(strings.NewReader("\x1b")))
	if err != nil || key != "cancel" {
		t.Fatalf("expected Escape to cancel, got %q, %v", key, err)
	}
}

func TestReadKeyRecognizesArrowAndBackspace(t *testing.T) {
	key, err := readKey(bufio.NewReader(strings.NewReader("\x1b[A")))
	if err != nil || key != "up" {
		t.Fatalf("expected up arrow, got %q, %v", key, err)
	}

	key, err = readKey(bufio.NewReader(strings.NewReader("\x7f")))
	if err != nil || key != "backspace" {
		t.Fatalf("expected backspace, got %q, %v", key, err)
	}
}
