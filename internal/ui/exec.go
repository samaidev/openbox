package ui

// execCmd is a thin wrapper around exec.Command so the rest of the package
// doesn't need to import os/exec directly.
// On Windows, execHidden (from exec_windows.go) is used to suppress the
// console window; on other platforms this just starts the command.
func execCmd(name string, args ...string) error {
	_, err := execHidden(name, args...)
	return err
}
