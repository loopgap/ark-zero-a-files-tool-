package bridge

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"arkkb/src/core/backup"
	"arkkb/src/core/config"
	"arkkb/src/core/file"
	"arkkb/src/core/lsp"
	"arkkb/src/core/storage"
	coreSync "arkkb/src/core/sync"
)

type SyncRequest struct {
	Snapshot *config.AppConfig
	Reason   string
	Dirty    bool
}

type Bridge struct {
	lspMgr             *lsp.LSPManager
	storage            *storage.StorageManager
	syncEngine         *coreSync.SyncEngine
	dirPicker          func(string) (string, error)
	savePicker         func() (string, error)
	readHelp           func(string) ([]byte, error)
	syncWorkspace      func(*config.AppConfig) error
	afterWorkspaceSync func()
	syncMu             sync.Mutex
	syncRunning        bool
	syncPending        *SyncRequest
}

func NewBridge(storage *storage.StorageManager, syncEngine *coreSync.SyncEngine, readHelp func(string) ([]byte, error)) *Bridge {
	b := &Bridge{
		storage:    storage,
		syncEngine: syncEngine,
		readHelp:   readHelp,
	}
	if syncEngine != nil {
		b.syncWorkspace = syncEngine.SyncWorkspace
	}
	b.afterWorkspaceSync = b.refreshAutoCategories
	b.lspMgr = lsp.NewLSPManager(b.onLSPNotify)
	return b
}

func (b *Bridge) SetDirectoryPicker(fn func(string) (string, error)) {
	b.dirPicker = fn
}

func (b *Bridge) SetSavePicker(fn func() (string, error)) {
	b.savePicker = fn
}

func (b *Bridge) onLSPNotify(id string, method string, params interface{}) {
	log.Printf("[LSP Notify] ID: %s, Method: %s", id, method)
}

// --- Workspace Management ---

func (b *Bridge) PickWorkspace() (string, error) {
	if b.dirPicker == nil {
		return "", os.ErrInvalid
	}
	result, err := b.dirPicker("Select ArkKB Workspace")
	if err != nil {
		return "", err
	}
	if result != "" {
		if err := b.OpenRecentWorkspace(result); err != nil {
			return "", err
		}
	}
	return result, nil
}

func (b *Bridge) OpenRecentWorkspace(path string) error {
	canonicalPath, err := canonicalizeMaybeMissingPath(path, false)
	if err != nil {
		return err
	}
	cfg, err := b.storage.KV.GetAppConfig()
	if err != nil {
		return err
	}
	root := config.NewWorkspaceRoot(canonicalPath)
	cfg.Workspace.Roots = []config.WorkspaceRoot{root}
	cfg.Workspace.ActiveRootID = root.ID
	cfg.Workspace.DefaultRootID = root.ID
	cfg.LastWorkspace = canonicalPath
	cfg.RecentWorkspaces = upsertRecentWorkspace(cfg.RecentWorkspaces, canonicalPath, root.Label)
	if err := b.storage.KV.SaveAppConfig(cfg); err != nil {
		return err
	}
	b.queueWorkspaceSync(cfg)
	return nil
}

// --- Standard IO ---

func (b *Bridge) CreateFile(parentDir string, name string) error {
	if err := validateEntryName(name); err != nil {
		return err
	}
	targetDir, rootID, err := b.resolveCreateParent(parentDir)
	if err != nil {
		return err
	}
	resolvedTarget, err := resolveWorkspacePath(config.CloneAppConfig(mustConfig(b.storage)), filepath.Join(targetDir, name), resolvePathOptions{AllowMissingLeaf: true})
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.FromSlash(resolvedTarget.CanonicalPath), []byte(""), 0644); err != nil {
		return err
	}
	if b.syncEngine != nil {
		return b.syncEngine.SyncPath(rootID, filepath.FromSlash(resolvedTarget.CanonicalPath))
	}
	return nil
}

func (b *Bridge) CreateFolder(parentDir string, name string) error {
	if err := validateEntryName(name); err != nil {
		return err
	}
	targetDir, _, err := b.resolveCreateParent(parentDir)
	if err != nil {
		return err
	}
	resolvedTarget, err := resolveWorkspacePath(config.CloneAppConfig(mustConfig(b.storage)), filepath.Join(targetDir, name), resolvePathOptions{AllowMissingLeaf: true})
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.FromSlash(resolvedTarget.CanonicalPath), 0755); err != nil {
		return err
	}
	if b.syncEngine != nil {
		cfg, cfgErr := b.storage.KV.GetAppConfig()
		if cfgErr == nil {
			b.queueWorkspaceSync(cfg)
		}
	}
	return nil
}

func (b *Bridge) Rename(oldPath string, newName string) error {
	if err := validateEntryName(newName); err != nil {
		return err
	}
	resolvedOldPath, cfg, err := b.resolveWorkspacePathWithConfig(oldPath, resolvePathOptions{})
	if err != nil {
		return err
	}
	newPath := filepath.Join(filepath.Dir(filepath.FromSlash(resolvedOldPath.CanonicalPath)), newName)
	resolvedNewPath, err := resolveWorkspacePath(cfg, newPath, resolvePathOptions{AllowMissingLeaf: true})
	if err != nil {
		return err
	}
	if err := os.Rename(filepath.FromSlash(resolvedOldPath.CanonicalPath), filepath.FromSlash(resolvedNewPath.CanonicalPath)); err != nil {
		return err
	}
	if memberships, err := b.storage.KV.GetVirtualFolderMemberships(resolvedOldPath.CanonicalPath); err == nil {
		_ = b.storage.KV.SetVirtualFolderMemberships(resolvedNewPath.CanonicalPath, memberships)
		_ = b.storage.KV.DeleteVirtualFolderMemberships(resolvedOldPath.CanonicalPath)
	}
	if b.syncEngine != nil {
		_ = b.syncEngine.SyncPath(resolvedOldPath.RootID, filepath.FromSlash(resolvedOldPath.CanonicalPath))
		return b.syncEngine.SyncPath(resolvedNewPath.RootID, filepath.FromSlash(resolvedNewPath.CanonicalPath))
	}
	return nil
}

func (b *Bridge) GetConfig() (*config.AppConfig, error) { return b.storage.KV.GetAppConfig() }
func (b *Bridge) SaveConfig(cfg *config.AppConfig) error {
	if err := b.storage.KV.SaveAppConfig(cfg); err != nil {
		return err
	}
	b.queueWorkspaceSync(cfg)
	return nil
}
func (b *Bridge) ReadFile(path string) (string, error) {
	resolvedPath, err := b.ResolveWorkspacePath(path)
	if err != nil {
		return "", err
	}
	content, err := os.ReadFile(filepath.FromSlash(resolvedPath.CanonicalPath))
	return string(content), err
}
func (b *Bridge) SaveFile(path string, content string) error {
	resolvedPath, err := b.ResolveWorkspacePath(path)
	if err != nil {
		return err
	}
	if err := file.SafeSave(filepath.FromSlash(resolvedPath.CanonicalPath), []byte(content)); err != nil {
		return err
	}
	if b.syncEngine != nil {
		if err := b.syncEngine.SyncPath(resolvedPath.RootID, filepath.FromSlash(resolvedPath.CanonicalPath)); err != nil {
			return err
		}
	}
	return b.RecordRecentItem(resolvedPath.CanonicalPath)
}
func (b *Bridge) SaveBinaryFile(path string, contentBase64 string) error {
	data, err := base64.StdEncoding.DecodeString(contentBase64)
	if err != nil {
		return err
	}
	resolvedPath, err := b.ResolveWorkspacePath(path)
	if err != nil {
		return err
	}
	if err := file.SafeSave(filepath.FromSlash(resolvedPath.CanonicalPath), data); err != nil {
		return err
	}
	if b.syncEngine != nil {
		if err := b.syncEngine.SyncPath(resolvedPath.RootID, filepath.FromSlash(resolvedPath.CanonicalPath)); err != nil {
			return err
		}
	}
	return b.RecordRecentItem(resolvedPath.CanonicalPath)
}
func (b *Bridge) OpenWithExternalApp(path string) error {
	resolvedPath, err := b.ResolveWorkspacePath(path)
	if err != nil {
		return err
	}
	return file.OpenWithExternalApp(filepath.FromSlash(resolvedPath.CanonicalPath))
}
func (b *Bridge) SoftDelete(path string) error {
	resolvedPath, err := b.ResolveWorkspacePath(path)
	if err != nil {
		return err
	}
	if err := file.SoftDelete(filepath.FromSlash(resolvedPath.CanonicalPath)); err != nil {
		return err
	}
	_ = b.storage.KV.DeleteVirtualFolderMemberships(resolvedPath.CanonicalPath)
	if b.syncEngine != nil {
		return b.syncEngine.SyncPath(resolvedPath.RootID, filepath.FromSlash(resolvedPath.CanonicalPath))
	}
	return nil
}

func (b *Bridge) CreateBackup() error {
	if b.savePicker == nil {
		return os.ErrInvalid
	}
	result, err := b.savePicker()
	if err != nil {
		return err
	}
	if result == "" {
		return nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	arkkbDir := filepath.Join(home, ".arkkb")

	return backup.ArchiveData(result, arkkbDir)
}

func (b *Bridge) OnStartup(ctx context.Context) {}

func (b *Bridge) queueWorkspaceSync(cfg *config.AppConfig) {
	if b.syncWorkspace == nil || cfg == nil {
		return
	}
	b.enqueueSyncRequest(&SyncRequest{
		Snapshot: config.CloneAppConfig(cfg),
		Reason:   "workspace update",
	})
}

func (b *Bridge) enqueueSyncRequest(request *SyncRequest) {
	if request == nil || request.Snapshot == nil {
		return
	}

	b.syncMu.Lock()
	if !b.syncRunning {
		b.syncRunning = true
		b.syncMu.Unlock()
		go b.runSyncQueue(request)
		return
	}
	if b.syncPending == nil {
		b.syncPending = request
	} else {
		b.syncPending.Snapshot = request.Snapshot
		b.syncPending.Reason = request.Reason
		b.syncPending.Dirty = true
	}
	b.syncMu.Unlock()
}

func (b *Bridge) runSyncQueue(initial *SyncRequest) {
	current := initial
	for current != nil {
		if err := b.syncWorkspace(current.Snapshot); err != nil {
			log.Printf("workspace sync error (%s): %v", current.Reason, err)
		}
		if b.afterWorkspaceSync != nil {
			b.afterWorkspaceSync()
		}

		b.syncMu.Lock()
		next := b.syncPending
		b.syncPending = nil
		if next == nil {
			b.syncRunning = false
			b.syncMu.Unlock()
			return
		}
		b.syncMu.Unlock()
		current = next
	}
}

func (b *Bridge) NormalizeWorkspacePath(path string) (string, error) {
	resolvedPath, err := b.ResolveWorkspacePath(path)
	if err != nil {
		return "", err
	}
	return resolvedPath.CanonicalPath, nil
}

func validateEntryName(name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return fmt.Errorf("name is required")
	}
	if trimmed == "." || trimmed == ".." {
		return fmt.Errorf("invalid name")
	}
	if strings.ContainsAny(trimmed, `/\\`) {
		return fmt.Errorf("name must not include path separators")
	}
	if strings.ContainsAny(trimmed, `:*?"<>|`) {
		return fmt.Errorf("name includes invalid characters")
	}
	return nil
}

func (b *Bridge) RefreshAutoCategories() {
	b.refreshAutoCategories()
}

func (b *Bridge) refreshAutoCategories() {
	autoCategories, err := b.computeAutoCategories()
	if err != nil {
		return
	}
	latest, err := b.storage.KV.GetAppConfig()
	if err != nil {
		return
	}
	latest.AutoCategories = autoCategories
	_ = b.storage.KV.SaveAppConfig(latest)
}

func mustConfig(storageManager *storage.StorageManager) *config.AppConfig {
	cfg, err := storageManager.KV.GetAppConfig()
	if err != nil {
		return config.DefaultConfig()
	}
	return cfg
}
