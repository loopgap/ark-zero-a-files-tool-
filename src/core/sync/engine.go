package sync

import (
	"sync"

	"arkkb/src/core/config"
	"arkkb/src/core/storage"
)

type SyncEngine struct {
	storage *storage.StorageManager
	mu      sync.Mutex
}

func NewSyncEngine(mgr *storage.StorageManager) *SyncEngine {
	return &SyncEngine{storage: mgr}
}

func (s *SyncEngine) OnFocus(workspaceRoot string) error {
	cfg, err := s.storage.KV.GetAppConfig()
	if err != nil {
		return err
	}

	if len(cfg.Workspace.Roots) == 0 && workspaceRoot != "" && workspaceRoot != "." {
		root := config.NewWorkspaceRoot(workspaceRoot)
		cfg.Workspace.Roots = []config.WorkspaceRoot{root}
		cfg.Workspace.ActiveRootID = root.ID
		cfg.Workspace.DefaultRootID = root.ID
		cfg.LastWorkspace = workspaceRoot
		if err := s.storage.KV.SaveAppConfig(cfg); err != nil {
			return err
		}
	}

	return s.SyncWorkspace(cfg)
}
