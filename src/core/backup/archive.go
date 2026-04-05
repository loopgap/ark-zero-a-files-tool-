// Package backup provides utilities for archiving and disaster recovery of ArkKB data.
package backup

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
)

// ArchiveData implements Chapter 7.1 of the constitution: a cold-backup mechanism.
// It creates a ZIP archive of the entire data directory (containing bbolt and Bluge nodes).
func ArchiveData(destZip string, dataDir string) error {
	zipFile, err := os.Create(destZip)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	archive := zip.NewWriter(zipFile)
	defer archive.Close()

	return filepath.Walk(dataDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}

		relPath, _ := filepath.Rel(dataDir, path)
		f, err := archive.Create(relPath)
		if err != nil {
			return err
		}

		srcFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		_, err = io.Copy(f, srcFile)
		return err
	})
}
