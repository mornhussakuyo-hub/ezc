//go:build !windows

package app

import "strings"

func isHidden(name string) bool {
	return strings.HasPrefix(name, ".")
}
