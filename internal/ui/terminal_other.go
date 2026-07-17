//go:build !darwin && !linux && !windows

package ui

import "errors"

type terminalState struct{}

func enterRaw() (*terminalState, error) {
	return nil, errors.New("当前平台不支持终端菜单")
}

func leaveRaw(state *terminalState) {}

func terminalDimensions() (int, int) {
	return 80, 24
}
