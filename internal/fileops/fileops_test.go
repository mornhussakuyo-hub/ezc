package fileops

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPasteCopiesDirectoryRecursively(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	target := filepath.Join(root, "target")
	if err := os.MkdirAll(filepath.Join(source, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "nested", "文件.txt"), []byte("hello"), 0o640); err != nil {
		t.Fatal(err)
	}

	destination, err := Paste(source, target, false, cancelConflict)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(destination, "nested", "文件.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello" {
		t.Fatalf("unexpected copied content: %q", data)
	}
	if _, err := os.Stat(source); err != nil {
		t.Fatalf("copy removed source: %v", err)
	}
}

func TestPasteMovesAndRemovesSource(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "源.txt")
	target := filepath.Join(root, "target")
	if err := os.WriteFile(source, []byte("move"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}

	destination, err := Paste(source, target, true, cancelConflict)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(source); !os.IsNotExist(err) {
		t.Fatalf("source still exists after move: %v", err)
	}
	if data, err := os.ReadFile(destination); err != nil || string(data) != "move" {
		t.Fatalf("unexpected destination: data=%q err=%v", data, err)
	}
}

func TestPasteRenamesConflict(t *testing.T) {
	root := t.TempDir()
	sourceDirectory := filepath.Join(root, "source")
	target := filepath.Join(root, "target")
	if err := os.MkdirAll(sourceDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(sourceDirectory, "note.txt")
	if err := os.WriteFile(source, []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, "note.txt"), []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}

	destination, err := Paste(source, target, false, func(string) (ConflictAction, error) {
		return ConflictRename, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(destination) != "note (1).txt" {
		t.Fatalf("unexpected renamed destination: %s", destination)
	}
}

func TestPasteRejectsDirectoryInsideItself(t *testing.T) {
	source := filepath.Join(t.TempDir(), "source")
	target := filepath.Join(source, "nested")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Paste(source, target, false, cancelConflict); err == nil {
		t.Fatal("expected recursive destination error")
	}
}

func TestPasteRejectsSymlinkedTargetInsideSource(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	nested := filepath.Join(source, "nested")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	linkedTarget := filepath.Join(root, "linked-target")
	if err := os.Symlink(nested, linkedTarget); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := Paste(source, linkedTarget, false, cancelConflict); err == nil {
		t.Fatal("expected recursive symlink destination error")
	}
}

func TestPasteDoesNotOverwriteSourceWithItself(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "same.txt")
	if err := os.WriteFile(source, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Paste(source, root, false, func(string) (ConflictAction, error) {
		return ConflictOverwrite, nil
	}); err == nil {
		t.Fatal("expected self-overwrite error")
	}
	data, err := os.ReadFile(source)
	if err != nil || string(data) != "keep" {
		t.Fatalf("source was damaged: data=%q err=%v", data, err)
	}
}

func cancelConflict(string) (ConflictAction, error) {
	return ConflictCancel, nil
}
