package file

import (
	"fmt"
	"log"
	"os"

	"arkkb/src/utils/pathutil"
)

func SafeSave(path string, content []byte) error {
	path, err := pathutil.NormalizePath(path)
	if err != nil {
		return fmt.Errorf("failed to normalize path: %w", err)
	}

	tmpPath := path + ".tmp"

	tmpFile, err := os.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	_, err = tmpFile.Write(content)
	if err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write to temp file: %w", err)
	}

	if err := tmpFile.Sync(); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to sync temp file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	log.Printf("[AtomicSave] Success: %s", path)
	return nil
}
