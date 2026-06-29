//go:build !windows
// +build !windows

package ui

import "os/exec"

// execHidden runs a command. On non-Windows platforms there's no console
// window to hide, so this is just a plain exec.Output().
func execHidden(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).Output()
}
