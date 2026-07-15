package ui

import (
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
