//go:build !windows

package clipboard

import "syscall"

var crossDeviceRenameTestError = syscall.EXDEV
