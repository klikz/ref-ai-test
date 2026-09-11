//go:build !windows

package utils

import "os/exec"

func assignCmdToPrintWorkersJob(_ *exec.Cmd) {}

// StartPrintWorkers is a no-op on non-Windows builds.
func StartPrintWorkers() {}

// StopAllPrintWorkers is a no-op on non-Windows builds.
func StopAllPrintWorkers() {}
