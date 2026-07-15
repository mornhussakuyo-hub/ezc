package system

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

func StateDir() (string, error) {
	if runtime.GOOS == "linux" {
		if runtimeDir := os.Getenv("XDG_RUNTIME_DIR"); runtimeDir != "" {
			return filepath.Join(runtimeDir, "ezc"), nil
		}
	}

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("定位用户缓存目录: %w", err)
	}
	return filepath.Join(cacheDir, "ezc"), nil
}
