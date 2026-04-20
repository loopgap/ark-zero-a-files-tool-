package bridge

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveWorkspacePathRejectsSymlinkEscape(t *testing.T) {
	rootPath := t.TempDir()
	outsidePath := t.TempDir()
	escapePath := filepath.Join(rootPath, "escape")
	if err := os.Symlink(outsidePath, escapePath); err != nil {
		t.Skipf("symlink creation is unavailable: %v", err)
	}

	cfg := testConfig(t, rootPath)
	if _, err := resolveWorkspacePath(cfg, filepath.Join(escapePath, "secret.txt"), resolvePathOptions{AllowMissingLeaf: true}); err == nil {
		t.Fatal("expected symlink escape to be rejected")
	}
}

func TestResolveWorkspacePathAllowsMissingLeafInsideWorkspace(t *testing.T) {
	rootPath := t.TempDir()
	cfg := testConfig(t, rootPath)

	resolvedPath, err := resolveWorkspacePath(cfg, filepath.Join(rootPath, "docs", "note.md"), resolvePathOptions{AllowMissingLeaf: true})
	if err != nil {
		t.Fatalf("resolveWorkspacePath returned error: %v", err)
	}
	if resolvedPath.RootID != "root-1" {
		t.Fatalf("expected root-1, got %s", resolvedPath.RootID)
	}
	if filepath.Base(resolvedPath.CanonicalPath) != "note.md" {
		t.Fatalf("expected note.md, got %s", resolvedPath.CanonicalPath)
	}
}

func TestPathWithinRootHandlesDriveRoot(t *testing.T) {
	if !pathWithinRoot("D:/国产单片机/demo.txt", "D:/") {
		t.Fatal("expected file under drive root to be treated as within workspace")
	}
	if pathWithinRoot("E:/other/demo.txt", "D:/") {
		t.Fatal("expected file on another drive to be treated as outside workspace")
	}
}
