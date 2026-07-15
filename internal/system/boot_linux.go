//go:build linux

package system

import (
	"fmt"
	"os"
	"strings"
)

func BootID() (string, error) {
	data, err := os.ReadFile("/proc/sys/kernel/random/boot_id")
	if err != nil {
		return "", fmt.Errorf("读取系统启动标识: %w", err)
	}
	return strings.TrimSpace(string(data)), nil
}
