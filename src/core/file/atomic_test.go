package file

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestSafeSave(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "arkkb_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	targetPath := filepath.Join(tmpDir, "test.txt")
	originalContent := []byte("original content")
	newContent := []byte("new atomic content")

	// 1. Initial write
	if err := os.WriteFile(targetPath, originalContent, 0644); err != nil {
		t.Fatalf("Failed to write initial file: %v", err)
	}

	// 2. Perform SafeSave
	if err := SafeSave(targetPath, newContent); err != nil {
		t.Errorf("SafeSave failed: %v", err)
	}

	// 3. Verify content
	finalContent, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("Failed to read back file: %v", err)
	}

	if !bytes.Equal(finalContent, newContent) {
		t.Errorf("Content mismatch! Got: %s, Want: %s", finalContent, newContent)
	}

	// 4. Verify no .tmp file remains
	tmpFile := targetPath + ".tmp"
	if _, err := os.Stat(tmpFile); !os.IsNotExist(err) {
		t.Errorf("Temp file still exists: %s", tmpFile)
	}
}
