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

// execHidden runs a command with its console window hidden, waits for it
// to finish, and returns stdout. Used by the PowerShell file picker.
func execHidden(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: hideWindowFlags,
	}
	return cmd.Output()
}

// execHiddenStart launches a command in the background without waiting
// for it to finish and without showing a console window. Used by openInOS
// to launch the default application for a file.
func execHiddenStart(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: hideWindowFlags,
	}
	return cmd.Start()
}
