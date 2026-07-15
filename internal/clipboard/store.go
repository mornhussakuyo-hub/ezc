package clipboard

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/mornhussakuyo-hub/ezc/internal/lock"
	"github.com/mornhussakuyo-hub/ezc/internal/system"
)

type Operation string

const (
	Copy Operation = "copy"
	Cut  Operation = "cut"
)

type Entry struct {
	Path      string    `json:"path"`
	Operation Operation `json:"operation"`
	Lock      lock.Info `json:"lock,omitempty"`
	AddedAt   time.Time `json:"added_at"`
}

type state struct {
	BootID  string  `json:"boot_id"`
	Entries []Entry `json:"entries"`
}

type Store struct {
	directory string
	statePath string
	lockPath  string
	bootID    string
}

func New() (*Store, error) {
	directory, err := system.StateDir()
	if err != nil {
		return nil, err
	}
	bootID, err := system.BootID()
	if err != nil {
		return nil, err
	}
	return NewAt(directory, bootID)
}

func NewAt(directory, bootID string) (*Store, error) {
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return nil, fmt.Errorf("创建剪切板目录: %w", err)
	}
	return &Store{
		directory: directory,
		statePath: filepath.Join(directory, "clipboard.json"),
		lockPath:  filepath.Join(directory, "clipboard.lock"),
		bootID:    bootID,
	}, nil
}

func (store *Store) List() ([]Entry, error) {
	var entries []Entry
	err := store.withState(func(current *state) (bool, error) {
		entries = append([]Entry(nil), current.Entries...)
		return false, nil
	})
	return entries, err
}

func (store *Store) Find(path string) (*Entry, error) {
	var found *Entry
	err := store.withState(func(current *state) (bool, error) {
		for index := range current.Entries {
			if samePath(current.Entries[index].Path, path) {
				entry := current.Entries[index]
				found = &entry
				break
			}
		}
		return false, nil
	})
	return found, err
}

func (store *Store) Upsert(entry Entry) (*Entry, error) {
	var previous *Entry
	err := store.withState(func(current *state) (bool, error) {
		filtered := make([]Entry, 0, len(current.Entries)+1)
		for index := range current.Entries {
			if samePath(current.Entries[index].Path, entry.Path) {
				oldEntry := current.Entries[index]
				previous = &oldEntry
				continue
			}
			filtered = append(filtered, current.Entries[index])
		}
		if entry.AddedAt.IsZero() {
			entry.AddedAt = time.Now()
		}
		current.Entries = append([]Entry{entry}, filtered...)
		return true, nil
	})
	return previous, err
}

func (store *Store) Remove(path string) (*Entry, error) {
	var removed *Entry
	err := store.withState(func(current *state) (bool, error) {
		filtered := current.Entries[:0]
		for index := range current.Entries {
			if removed == nil && samePath(current.Entries[index].Path, path) {
				entry := current.Entries[index]
				removed = &entry
				continue
			}
			filtered = append(filtered, current.Entries[index])
		}
		current.Entries = filtered
		return removed != nil, nil
	})
	return removed, err
}

func (store *Store) Replace(entry Entry) error {
	return store.withState(func(current *state) (bool, error) {
		for index := range current.Entries {
			if samePath(current.Entries[index].Path, entry.Path) {
				current.Entries[index] = entry
				return true, nil
			}
		}
		return false, fmt.Errorf("剪切板中不存在 %q", entry.Path)
	})
}

func (store *Store) CleanMissing() ([]Entry, error) {
	var removed []Entry
	err := store.withState(func(current *state) (bool, error) {
		filtered := current.Entries[:0]
		for index := range current.Entries {
			if _, err := os.Lstat(current.Entries[index].Path); os.IsNotExist(err) {
				removed = append(removed, current.Entries[index])
				continue
			}
			filtered = append(filtered, current.Entries[index])
		}
		current.Entries = filtered
		return len(removed) > 0, nil
	})
	return removed, err
}

func (store *Store) withState(action func(*state) (bool, error)) error {
	if err := store.acquire(); err != nil {
		return err
	}
	defer store.release()

	current, reset, err := store.load()
	if err != nil {
		return err
	}
	changed, err := action(current)
	if err != nil {
		return err
	}
	if reset || changed {
		return store.save(current)
	}
	return nil
}

func (store *Store) load() (*state, bool, error) {
	data, err := os.ReadFile(store.statePath)
	if os.IsNotExist(err) {
		return &state{BootID: store.bootID, Entries: []Entry{}}, true, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("读取剪切板: %w", err)
	}
	var current state
	if err := json.Unmarshal(data, &current); err != nil {
		return nil, false, fmt.Errorf("解析剪切板: %w", err)
	}
	if current.BootID != store.bootID {
		_ = os.RemoveAll(filepath.Join(store.directory, "locks"))
		return &state{BootID: store.bootID, Entries: []Entry{}}, true, nil
	}
	return &current, false, nil
}

func (store *Store) save(current *state) error {
	data, err := json.MarshalIndent(current, "", "  ")
	if err != nil {
		return fmt.Errorf("编码剪切板: %w", err)
	}
	temporaryPath := store.statePath + ".tmp"
	if err := os.WriteFile(temporaryPath, data, 0o600); err != nil {
		return fmt.Errorf("写入剪切板: %w", err)
	}
	if runtime.GOOS == "windows" {
		_ = os.Remove(store.statePath)
	}
	if err := os.Rename(temporaryPath, store.statePath); err != nil {
		_ = os.Remove(temporaryPath)
		return fmt.Errorf("保存剪切板: %w", err)
	}
	return nil
}

func (store *Store) acquire() error {
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		file, err := os.OpenFile(store.lockPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err == nil {
			_ = file.Close()
			return nil
		}
		if !errors.Is(err, os.ErrExist) {
			return fmt.Errorf("锁定剪切板: %w", err)
		}
		if info, statErr := os.Stat(store.lockPath); statErr == nil && time.Since(info.ModTime()) > 15*time.Second {
			_ = os.Remove(store.lockPath)
			continue
		}
		time.Sleep(25 * time.Millisecond)
	}
	return errors.New("等待剪切板锁超时")
}

func (store *Store) release() {
	_ = os.Remove(store.lockPath)
}

func samePath(left, right string) bool {
	left = filepath.Clean(left)
	right = filepath.Clean(right)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(left, right)
	}
	return left == right
}
