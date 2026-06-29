//go:build !windows
// +build !windows

package ui

import "os/exec"

// execHidden runs a command, waits for it, and returns stdout.
// On non-Windows there's no console window to hide.
func execHidden(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).Output()
}

// execHiddenStart launches a command in the background without waiting.
func execHiddenStart(name string, args ...string) error {
	return exec.Command(name, args...).Start()
}
