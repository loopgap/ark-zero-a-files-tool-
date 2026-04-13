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
	"arkkb/src/utils/pathutil"
)

type Bridge struct {
	lspMgr     *lsp.LSPManager
	storage    *storage.StorageManager
	syncEngine *coreSync.SyncEngine
	dirPicker  func(string) (string, error)
	savePicker func() (string, error)
	syncMu     sync.Mutex
	syncBusy   bool
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
	if err := validateEntryName(name); err != nil {
		return err
	}
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
	if err := validateEntryName(name); err != nil {
		return err
	}
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
	if err := validateEntryName(newName); err != nil {
		return err
	}
	normalizedOldPath, cfg, err := b.normalizeWorkspacePathWithConfig(oldPath)
	if err != nil {
		return err
	}
	dir := filepath.Dir(filepath.FromSlash(normalizedOldPath))
	newPath := filepath.Join(dir, newName)
	normalizedNewPath, err := pathutil.NormalizePath(newPath)
	if err != nil {
		return err
	}
	if rootIDForPath(cfg, normalizedNewPath) == "" {
		return fmt.Errorf("path is outside the workspace")
	}
	if err := os.Rename(filepath.FromSlash(normalizedOldPath), filepath.FromSlash(normalizedNewPath)); err != nil {
		return err
	}
	if memberships, err := b.storage.KV.GetVirtualFolderMemberships(normalizedOldPath); err == nil {
		_ = b.storage.KV.SetVirtualFolderMemberships(normalizedNewPath, memberships)
		_ = b.storage.KV.DeleteVirtualFolderMemberships(normalizedOldPath)
	}
	if b.syncEngine != nil {
		rootID := rootIDForPath(cfg, normalizedNewPath)
		_ = b.syncEngine.SyncPath(rootID, filepath.FromSlash(normalizedOldPath))
		return b.syncEngine.SyncPath(rootID, filepath.FromSlash(normalizedNewPath))
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
	normalizedPath, err := b.NormalizeWorkspacePath(path)
	if err != nil {
		return "", err
	}
	content, err := os.ReadFile(filepath.FromSlash(normalizedPath))
	return string(content), err
}
func (b *Bridge) SaveFile(path string, content string) error {
	normalizedPath, err := b.NormalizeWorkspacePath(path)
	if err != nil {
		return err
	}
	if err := file.SafeSave(filepath.FromSlash(normalizedPath), []byte(content)); err != nil {
		return err
	}
	rootID := b.rootIDForPath(normalizedPath)
	if b.syncEngine != nil {
		if err := b.syncEngine.SyncPath(rootID, filepath.FromSlash(normalizedPath)); err != nil {
			return err
		}
	}
	return b.RecordRecentItem(normalizedPath)
}
func (b *Bridge) SaveBinaryFile(path string, contentBase64 string) error {
	data, err := base64.StdEncoding.DecodeString(contentBase64)
	if err != nil {
		return err
	}
	normalizedPath, err := b.NormalizeWorkspacePath(path)
	if err != nil {
		return err
	}
	if err := file.SafeSave(filepath.FromSlash(normalizedPath), data); err != nil {
		return err
	}
	rootID := b.rootIDForPath(normalizedPath)
	if b.syncEngine != nil {
		if err := b.syncEngine.SyncPath(rootID, filepath.FromSlash(normalizedPath)); err != nil {
			return err
		}
	}
	return b.RecordRecentItem(normalizedPath)
}
func (b *Bridge) OpenWithExternalApp(path string) error {
	normalizedPath, err := b.NormalizeWorkspacePath(path)
	if err != nil {
		return err
	}
	return file.OpenWithExternalApp(filepath.FromSlash(normalizedPath))
}
func (b *Bridge) SoftDelete(path string) error {
	normalizedPath, err := b.NormalizeWorkspacePath(path)
	if err != nil {
		return err
	}
	if err := file.SoftDelete(filepath.FromSlash(normalizedPath)); err != nil {
		return err
	}
	_ = b.storage.KV.DeleteVirtualFolderMemberships(normalizedPath)
	if b.syncEngine != nil {
		return b.syncEngine.SyncPath(b.rootIDForPath(normalizedPath), filepath.FromSlash(normalizedPath))
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

	// 创建配置快照，确保 goroutine 使用的是不可变的配置
	snapshot := config.NormalizeAppConfig(cfg)

	b.syncMu.Lock()
	if b.syncBusy {
		b.syncMu.Unlock()
		return // 有正在进行的同步，忽略新的请求
	}
	b.syncBusy = true
	b.syncMu.Unlock()

	go func(snapshot *config.AppConfig) {
		defer func() {
			b.syncMu.Lock()
			b.syncBusy = false
			b.syncMu.Unlock()
		}()

		// 记录错误，不要忽略
		if err := b.syncEngine.SyncWorkspace(snapshot); err != nil {
			log.Printf("workspace sync error: %v", err)
		}

		// 在单独的锁保护下刷新自动分类
		b.RefreshAutoCategories()
	}(snapshot)
}

func (b *Bridge) NormalizeWorkspacePath(path string) (string, error) {
	normalizedPath, _, err := b.normalizeWorkspacePathWithConfig(path)
	return normalizedPath, err
}

func (b *Bridge) normalizeWorkspacePathWithConfig(path string) (string, *config.AppConfig, error) {
	normalizedPath, err := pathutil.NormalizePath(path)
	if err != nil {
		return "", nil, err
	}
	cfg, err := b.storage.KV.GetAppConfig()
	if err != nil {
		return "", nil, err
	}
	if rootIDForPath(cfg, normalizedPath) == "" {
		return "", nil, fmt.Errorf("path is outside the workspace")
	}
	return normalizedPath, cfg, nil
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
