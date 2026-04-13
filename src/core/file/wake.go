package file

import (
	"fmt"
	"os/exec"
	"runtime"
)

// OpenWithExternalApp Chapter 5: External Penetration Layer - safely wake system associated software
func OpenWithExternalApp(filePath string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer.exe", filePath)
	case "darwin":
		cmd = exec.Command("open", filePath)
	case "linux":
		cmd = exec.Command("xdg-open", filePath)
	default:
		return fmt.Errorf("unsupported platform for external wake: %s", runtime.GOOS)
	}

	return cmd.Start()
}
