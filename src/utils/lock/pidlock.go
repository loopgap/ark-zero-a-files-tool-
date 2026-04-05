package lock

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"syscall"
)

type PIDLock struct {
	path string
}

func NewPIDLock() (*PIDLock, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	arkkbDir := filepath.Join(homeDir, ".arkkb")
	if err := os.MkdirAll(arkkbDir, 0755); err != nil {
		return nil, err
	}
	return &PIDLock{
		path: filepath.Join(arkkbDir, ".lock"),
	}, nil
}

func (l *PIDLock) Lock() error {
	if data, err := os.ReadFile(l.path); err == nil {
		if pid, err := strconv.Atoi(string(data)); err == nil {
			if isProcessAlive(pid) {
				return fmt.Errorf("arkkb is already running (PID: %d)", pid)
			}
			// Orphan lock detected: process is dead, overwrite
		}
	}
	return os.WriteFile(l.path, []byte(strconv.Itoa(os.Getpid())), 0644)
}

func (l *PIDLock) Unlock() error {
	data, err := os.ReadFile(l.path)
	if err != nil {
		return err
	}
	if string(data) == strconv.Itoa(os.Getpid()) {
		return os.Remove(l.path)
	}
	return nil
}

func isProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	if runtime.GOOS == "windows" {
		const STILL_ACTIVE = 259
		const PROCESS_QUERY_LIMITED_INFORMATION = 0x1000

		h, err := syscall.OpenProcess(PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
		if err != nil {
			return false
		}
		defer syscall.CloseHandle(h)

		var exitCode uint32
		err = syscall.GetExitCodeProcess(h, &exitCode)
		if err != nil {
			return false
		}
		return exitCode == STILL_ACTIVE
	}

	// Unix-like systems
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = process.Signal(syscall.Signal(0))
	return err == nil
}
