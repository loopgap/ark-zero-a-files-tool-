package bridge

import (
	"os"
	"path/filepath"
	"testing"

	"arkkb/src/core/config"
	"arkkb/src/utils/pathutil"
)

func TestResolveCreateTargetPrefersVirtualFolderPath(t *testing.T) {
	rootPath := t.TempDir()
	docsPath := filepath.Join(rootPath, "docs")
	if err := os.MkdirAll(docsPath, 0o755); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}

	cfg := testConfig(t, rootPath)
	folderPath, _ := pathutil.NormalizePath(docsPath)
	folder := &config.VirtualFolder{
		ID:                  "vf-1",
		WorkspaceID:         cfg.Workspace.ID,
		Name:                "Docs",
		PreferredRootID:     "root-1",
		PreferredParentPath: folderPath,
	}

	targetPath, rootID, err := resolveCreateTarget(cfg, "", "", folder)
	if err != nil {
		t.Fatalf("resolve create target: %v", err)
	}
	if rootID != "root-1" {
		t.Fatalf("expected root-1, got %s", rootID)
	}
	if targetPath != folderPath {
		t.Fatalf("expected %s, got %s", folderPath, targetPath)
	}
}

func TestResolveCreateTargetPrefersAllowlistedDirectory(t *testing.T) {
	rootPath := t.TempDir()
	allowlistedDir := filepath.Join(rootPath, "notes", "project-a")
	if err := os.MkdirAll(allowlistedDir, 0o755); err != nil {
		t.Fatalf("mkdir allowlisted dir: %v", err)
	}

	cfg := testConfig(t, rootPath)
	cfg.Policy.DirectoryAllowlist = []string{"notes/project-a"}
	expected, _ := pathutil.NormalizePath(allowlistedDir)

	targetPath, rootID, err := resolveCreateTarget(cfg, "", "root-1", nil)
	if err != nil {
		t.Fatalf("resolve create target: %v", err)
	}
	if rootID != "root-1" {
		t.Fatalf("expected root-1, got %s", rootID)
	}
	if targetPath != expected {
		t.Fatalf("expected %s, got %s", expected, targetPath)
	}
}

func TestNormalizeExistingDirectoryRejectsOutsideWorkspace(t *testing.T) {
	rootPath := t.TempDir()
	outsidePath := t.TempDir()
	cfg := testConfig(t, rootPath)

	if _, _, err := normalizeExistingDirectory(cfg, outsidePath); err == nil {
		t.Fatal("expected outside path to be rejected")
	}
}

func testConfig(t *testing.T, rootPath string) *config.AppConfig {
	t.Helper()
	normalizedRoot, err := pathutil.NormalizePath(rootPath)
	if err != nil {
		t.Fatalf("normalize root: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.Workspace = config.WorkspaceSession{
		ID:            "workspace-1",
		Name:          "Test Workspace",
		Roots:         []config.WorkspaceRoot{{ID: "root-1", Path: normalizedRoot, Label: filepath.Base(rootPath)}},
		ActiveRootID:  "root-1",
		DefaultRootID: "root-1",
	}
	return config.NormalizeAppConfig(cfg)
}
