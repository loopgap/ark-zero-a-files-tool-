package file

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"arkkb/src/utils/pathutil"
)

// RescueTempFiles scans for residual .tmp files for a given main path.
func RescueTempFiles(path string) {
	// Normalize path first
	normalizedPath, err := pathutil.NormalizePath(path)
	if err != nil {
		log.Printf("[Rescue] Failed to normalize path: %v", err)
		return
	}
	path = normalizedPath
	tmpPath := path + ".tmp"

	// 1. Check main file and tmp file
	_, errMain := os.Stat(path)
	_, errTmp := os.Stat(tmpPath)

	if os.IsNotExist(errMain) && errTmp == nil {
		// Case A: Main file is missing, but .tmp exists. Rescue it.
		if err := os.Rename(tmpPath, path); err != nil {
			log.Printf("[Rescue] Failed to rename %s -> %s: %v", tmpPath, path, err)
		} else {
			log.Printf("[Rescue] Successfully rescued from: %s", path)
		}
	} else if errMain == nil && errTmp == nil {
		// Case B: Both exist. Rename didn't happen yet.
		// For safety, remove the potentially incomplete .tmp file.
		if err := os.Remove(tmpPath); err != nil {
			log.Printf("[Rescue] Failed to remove redundant tmp file %s: %v", tmpPath, err)
		} else {
			log.Printf("[Rescue] Removed redundant tmp file: %s", tmpPath)
		}
	}
}

// GlobalRescue scans the root directory for residual .tmp files and processes them.
func GlobalRescue(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".tmp") {
			mainPath := strings.TrimSuffix(path, ".tmp")
			RescueTempFiles(mainPath)
		}
		return nil
	})
}
