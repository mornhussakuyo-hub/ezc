package clipboard

import (
	"errors"
	"syscall"
)

const errorNotSameDevice syscall.Errno = 17

func isCrossDeviceRename(err error) bool {
	return errors.Is(err, errorNotSameDevice)
}
