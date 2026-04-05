package pathutil

import (
	"path/filepath"
)

// NormalizePath cleans the path, converts to absolute, and unifies slashes to forward slashes.
// This ensures cross-platform consistency for storage keys and index IDs.
func NormalizePath(path string) (string, error) {
	// 1. Convert to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	// 2. Clean redundant parts (./, ../, //)
	cleanPath := filepath.Clean(absPath)

	// 3. Unify slashes to forward slashes for cross-platform consistency
	return filepath.ToSlash(cleanPath), nil
}
