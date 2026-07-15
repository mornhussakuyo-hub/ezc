//go:build !linux && !windows

package system

import "runtime"

func BootID() (string, error) {
	return runtime.GOOS + "-current", nil
}
