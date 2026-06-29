//go:build windows
// +build windows

package ui

import (
	"os/exec"
	"syscall"
)

// hideWindowFlags is the creation flag that prevents a console window
// from flashing up when we spawn a subprocess (e.g. PowerShell for the
// native file picker). CREATE_NO_WINDOW = 0x08000000.
const hideWindowFlags = 0x08000000

// execHidden runs a command with its console window hidden. On Windows
// this uses CREATE_NO_WINDOW so no black cmd box flashes on screen.
// It waits for the command to finish and returns stdout.
func execHidden(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	// Hide the console window.
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: hideWindowFlags,
	}
	return cmd.Output()
}
