package ui

import "os/exec"

// execCmd launches a command in the background (fire-and-forget) without
// waiting for it to complete. On Windows the console window is hidden
// via execHiddenStart (see exec_windows.go).
func execCmd(name string, args ...string) error {
	return execHiddenStart(name, args...)
}

// keep os/exec import alive
var _ = exec.Command
