package ui

import "os/exec"

// execCmd is a thin wrapper around exec.Command so the rest of the package
// doesn't need to import os/exec directly.
func execCmd(name string, args ...string) error {
	return exec.Command(name, args...).Start()
}
