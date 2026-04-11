package bridge

import (
	"context"
	"encoding/base64"
	"log"
	"os"
	"path/filepath"

	"arkkb/src/core/backup"
	"arkkb/src/core/config"
	"arkkb/src/core/file"
	"arkkb/src/core/lsp"
	"arkkb/src/core/storage"
	coreSync "arkkb/src/core/sync"
	"arkkb/src/utils/pathutil"
)

type Bridge struct {
	lspMgr     *lsp.LSPManager
	storage    *storage.StorageManager
	syncEngine *coreSync.SyncEngine
	dirPicker  func(string) (string, error)
	savePicker func() (string, error)
}

func NewBridge(storage *storage.StorageManager, syncEngine *coreSync.SyncEngine) *Bridge {
	b := &Bridge{
		storage:    storage,
		syncEngine: syncEngine,
	}
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
	normalizedPath, err := pathutil.NormalizePath(path)
	if err != nil {
		return err
	}
	cfg, err := b.storage.KV.GetAppConfig()
	if err != nil {
		return err
	}
	root := config.NewWorkspaceRoot(normalizedPath)
	cfg.Workspace.Roots = []config.WorkspaceRoot{root}
	cfg.Workspace.ActiveRootID = root.ID
	cfg.Workspace.DefaultRootID = root.ID
	cfg.LastWorkspace = normalizedPath
	cfg.RecentWorkspaces = upsertRecentWorkspace(cfg.RecentWorkspaces, normalizedPath, root.Label)
	if err := b.storage.KV.SaveAppConfig(cfg); err != nil {
		return err
	}
	b.queueWorkspaceSync(cfg)
	return nil
}

// --- Standard IO ---

func (b *Bridge) CreateFile(parentDir string, name string) error {
	targetDir, rootID, err := b.resolveCreateParent(parentDir)
	if err != nil {
		return err
	}
	path := filepath.Join(targetDir, name)
	if err := os.WriteFile(path, []byte(""), 0644); err != nil {
		return err
	}
	if b.syncEngine != nil {
		return b.syncEngine.SyncPath(rootID, path)
	}
	return nil
}

func (b *Bridge) CreateFolder(parentDir string, name string) error {
	targetDir, _, err := b.resolveCreateParent(parentDir)
	if err != nil {
		return err
	}
	path := filepath.Join(targetDir, name)
	if err := os.MkdirAll(path, 0755); err != nil {
		return err
	}
	if b.syncEngine != nil {
		cfg, cfgErr := b.storage.KV.GetAppConfig()
		if cfgErr == nil {
			return b.syncEngine.SyncWorkspace(cfg)
		}
	}
	return nil
}

func (b *Bridge) Rename(oldPath string, newName string) error {
	dir := filepath.Dir(oldPath)
	newPath := filepath.Join(dir, newName)
	if err := os.Rename(oldPath, newPath); err != nil {
		return err
	}
	if memberships, err := b.storage.KV.GetVirtualFolderMemberships(oldPath); err == nil {
		_ = b.storage.KV.SetVirtualFolderMemberships(newPath, memberships)
		_ = b.storage.KV.DeleteVirtualFolderMemberships(oldPath)
	}
	if b.syncEngine != nil {
		rootID := b.rootIDForPath(newPath)
		_ = b.syncEngine.SyncPath(rootID, oldPath)
		return b.syncEngine.SyncPath(rootID, newPath)
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
	content, err := os.ReadFile(path)
	return string(content), err
}
func (b *Bridge) SaveFile(path string, content string) error {
	if err := file.SafeSave(path, []byte(content)); err != nil {
		return err
	}
	rootID := b.rootIDForPath(path)
	if b.syncEngine != nil {
		if err := b.syncEngine.SyncPath(rootID, path); err != nil {
			return err
		}
	}
	return b.RecordRecentItem(path)
}
func (b *Bridge) SaveBinaryFile(path string, contentBase64 string) error {
	data, err := base64.StdEncoding.DecodeString(contentBase64)
	if err != nil {
		return err
	}
	if err := file.SafeSave(path, data); err != nil {
		return err
	}
	rootID := b.rootIDForPath(path)
	if b.syncEngine != nil {
		if err := b.syncEngine.SyncPath(rootID, path); err != nil {
			return err
		}
	}
	return b.RecordRecentItem(path)
}
func (b *Bridge) OpenWithExternalApp(path string) error {
	return file.OpenWithExternalApp(path)
}
func (b *Bridge) SoftDelete(path string) error {
	if err := file.SoftDelete(path); err != nil {
		return err
	}
	_ = b.storage.KV.DeleteVirtualFolderMemberships(path)
	if b.syncEngine != nil {
		return b.syncEngine.SyncPath(b.rootIDForPath(path), path)
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
	if b.syncEngine == nil || cfg == nil {
		return
	}
	go func(snapshot *config.AppConfig) {
		_ = b.syncEngine.SyncWorkspace(snapshot)
		b.RefreshAutoCategories()
	}(config.NormalizeAppConfig(cfg))
}

func (b *Bridge) RefreshAutoCategories() {
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
