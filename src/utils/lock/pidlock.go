package lock

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
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
