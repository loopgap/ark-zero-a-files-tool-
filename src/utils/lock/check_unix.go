//go:build !windows

package lock

import (
	"os"
	"syscall"
)

func isProcessAlive(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// On Unix, FindProcess always succeeds.
	// Signal(0) checks for existence.
	err = process.Signal(syscall.Signal(0))
	return err == nil
}
