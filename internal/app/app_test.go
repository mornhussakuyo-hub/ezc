package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBestDirectoryEntryUsesPinyinSearch(t *testing.T) {
	directory := t.TempDir()
	for _, name := range []string{"测试版本.txt", "测试报告.txt", "草稿.txt"} {
		if err := os.WriteFile(filepath.Join(directory, name), []byte(name), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}

	matched, ok := bestDirectoryEntry(entries, "csbg")
	if !ok || matched.Name() != "测试报告.txt" {
		t.Fatalf("expected 测试报告.txt, got %v, %v", matched, ok)
	}
}

func TestBestDirectoryEntryReturnsNoMatch(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "测试报告.txt"), []byte("ok"), 0o600); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}

	if matched, ok := bestDirectoryEntry(entries, "xyz"); ok {
		t.Fatalf("expected no match, got %v", matched)
	}
}
