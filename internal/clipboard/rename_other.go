//go:build !windows

package clipboard

import (
	"errors"
	"syscall"
)

func isCrossDeviceRename(err error) bool {
	return errors.Is(err, syscall.EXDEV)
}
