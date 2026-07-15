package clipboard

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStoreUpsertDeduplicatesAndMovesToTop(t *testing.T) {
	store, err := NewAt(t.TempDir(), "boot-1")
	if err != nil {
		t.Fatal(err)
	}
	first := filepath.Join(t.TempDir(), "first")
	second := filepath.Join(t.TempDir(), "second")
	if _, err := store.Upsert(Entry{Path: first, Operation: Copy}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Upsert(Entry{Path: second, Operation: Cut}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Upsert(Entry{Path: first, Operation: Cut, AddedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}

	entries, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Path != first || entries[0].Operation != Cut {
		t.Fatalf("unexpected top entry: %#v", entries[0])
	}
}

func TestStoreResetsAfterBootChanges(t *testing.T) {
	directory := t.TempDir()
	store, err := NewAt(directory, "boot-1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Upsert(Entry{Path: "/tmp/example", Operation: Copy}); err != nil {
		t.Fatal(err)
	}

	restartedStore, err := NewAt(directory, "boot-2")
	if err != nil {
		t.Fatal(err)
	}
	entries, err := restartedStore.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected reset clipboard, got %#v", entries)
	}
}

func TestStoreCleansMissingPaths(t *testing.T) {
	directory := t.TempDir()
	store, err := NewAt(filepath.Join(directory, "state"), "boot-1")
	if err != nil {
		t.Fatal(err)
	}
	existing := filepath.Join(directory, "existing")
	if err := os.WriteFile(existing, []byte("ok"), 0o600); err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(directory, "missing")
	_, _ = store.Upsert(Entry{Path: existing, Operation: Copy})
	_, _ = store.Upsert(Entry{Path: missing, Operation: Copy})

	removed, err := store.CleanMissing()
	if err != nil {
		t.Fatal(err)
	}
	if len(removed) != 1 || removed[0].Path != missing {
		t.Fatalf("unexpected removed entries: %#v", removed)
	}
	entries, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Path != existing {
		t.Fatalf("unexpected remaining entries: %#v", entries)
	}
}
