package lock

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/mornhussakuyo-hub/ezc/internal/system"
)

type Info struct {
	Token string `json:"token,omitempty"`
	Ready string `json:"ready,omitempty"`
}

func Acquire(path string) (Info, error) {
	stateDir, err := system.StateDir()
	if err != nil {
		return Info{}, err
	}
	lockDir := filepath.Join(stateDir, "locks")
	if err := os.MkdirAll(lockDir, 0o700); err != nil {
		return Info{}, fmt.Errorf("创建锁目录: %w", err)
	}

	id, err := randomID()
	if err != nil {
		return Info{}, err
	}
	info := Info{
		Token: filepath.Join(lockDir, id+".token"),
		Ready: filepath.Join(lockDir, id+".ready"),
	}
	errorPath := filepath.Join(lockDir, id+".error")
	if err := os.WriteFile(info.Token, []byte(path), 0o600); err != nil {
		return Info{}, fmt.Errorf("创建锁令牌: %w", err)
	}

	executable, err := os.Executable()
	if err != nil {
		_ = os.Remove(info.Token)
		return Info{}, fmt.Errorf("定位 ezc 可执行文件: %w", err)
	}
	command := exec.Command(executable, "__lock-worker", path, info.Token, info.Ready, errorPath)
	configureDetached(command)
	if err := command.Start(); err != nil {
		_ = os.Remove(info.Token)
		return Info{}, fmt.Errorf("启动锁进程: %w", err)
	}
	_ = command.Process.Release()

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(info.Ready); err == nil {
			_ = os.Remove(errorPath)
			return info, nil
		}
		if data, err := os.ReadFile(errorPath); err == nil {
			_ = os.Remove(info.Token)
			_ = os.Remove(errorPath)
			return Info{}, errors.New(string(data))
		}
		time.Sleep(25 * time.Millisecond)
	}

	_ = os.Remove(info.Token)
	_ = os.Remove(info.Ready)
	_ = os.Remove(errorPath)
	return Info{}, errors.New("锁定超时")
}

func Release(info Info) error {
	if info.Token == "" {
		return nil
	}
	if err := os.Remove(info.Token); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("释放锁令牌: %w", err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for info.Ready != "" && time.Now().Before(deadline) {
		if _, err := os.Stat(info.Ready); os.IsNotExist(err) {
			return nil
		}
		time.Sleep(25 * time.Millisecond)
	}
	if info.Ready != "" {
		_ = os.Remove(info.Ready)
	}
	return nil
}

func RunWorker(path, tokenPath, readyPath, errorPath string) error {
	heldLock, err := acquirePlatform(path)
	if err != nil {
		message := fmt.Sprintf("无法锁定 %q: %v", path, err)
		_ = os.WriteFile(errorPath, []byte(message), 0o600)
		return errors.New(message)
	}
	defer heldLock.Close()

	if err := os.WriteFile(readyPath, []byte("ready"), 0o600); err != nil {
		return fmt.Errorf("写入锁状态: %w", err)
	}
	defer os.Remove(readyPath)

	for {
		if _, err := os.Stat(tokenPath); os.IsNotExist(err) {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func randomID() (string, error) {
	data := make([]byte, 16)
	if _, err := rand.Read(data); err != nil {
		return "", fmt.Errorf("生成锁标识: %w", err)
	}
	return hex.EncodeToString(data), nil
}
