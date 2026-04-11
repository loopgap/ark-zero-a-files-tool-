package config

import "testing"

func TestNormalizeAppConfigInitializesSlices(t *testing.T) {
	cfg := NormalizeAppConfig(&AppConfig{
		Workspace: WorkspaceSession{},
		Policy:    PolicyConfig{},
	})

	if cfg.Workspace.Roots == nil {
		t.Fatal("expected workspace roots to be initialized")
	}
	if cfg.VirtualFolders == nil {
		t.Fatal("expected virtual folders to be initialized")
	}
	if cfg.AutoCategories == nil {
		t.Fatal("expected auto categories to be initialized")
	}
	if cfg.Policy.DirectoryAllowlist == nil {
		t.Fatal("expected directory allowlist to be initialized")
	}
	if cfg.Policy.DirectoryBlocklist == nil {
		t.Fatal("expected directory blocklist to be initialized")
	}
	if cfg.Policy.FileTypeAllowlist == nil {
		t.Fatal("expected file type allowlist to be initialized")
	}
	if cfg.Policy.FileTypeBlocklist == nil {
		t.Fatal("expected file type blocklist to be initialized")
	}
	if cfg.RecentItems == nil {
		t.Fatal("expected recent items to be initialized")
	}
	if cfg.RecentWorkspaces == nil {
		t.Fatal("expected recent workspaces to be initialized")
	}
}

func TestDefaultConfigInitializesSlices(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Workspace.Roots == nil {
		t.Fatal("expected workspace roots to be initialized")
	}
	if cfg.VirtualFolders == nil {
		t.Fatal("expected virtual folders to be initialized")
	}
	if cfg.Policy.DirectoryAllowlist == nil {
		t.Fatal("expected directory allowlist to be initialized")
	}
	if cfg.Policy.FileTypeAllowlist == nil {
		t.Fatal("expected file type allowlist to be initialized")
	}
	if cfg.RecentItems == nil {
		t.Fatal("expected recent items to be initialized")
	}
	if cfg.RecentWorkspaces == nil {
		t.Fatal("expected recent workspaces to be initialized")
	}
}
