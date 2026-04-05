package sync

import (
	"errors"
	"io/fs"
	"path/filepath"
	"sync"
	"time"

	"arkkb/src/core/storage"
	"github.com/blugelabs/bluge"
)

var ErrTooManyChanges = errors.New("too many changes detected, user confirmation required")

type SyncEngine struct {
	storage *storage.StorageManager
	mu      sync.Mutex
}

func NewSyncEngine(mgr *storage.StorageManager) *SyncEngine {
	return &SyncEngine{storage: mgr}
}

func (s *SyncEngine) OnFocus(workspaceRoot string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var changedDocs []bluge.Document
	changeCount := 0

	err := filepath.WalkDir(workspaceRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}

		info, _ := d.Info()
		mtime := info.ModTime().Unix()

		doc := bluge.NewDocument(path).
			AddField(bluge.NewKeywordField("path", path).StoreValue()).
			AddField(bluge.NewDateTimeField("mtime", time.Unix(mtime, 0)).StoreValue())
		
		changedDocs = append(changedDocs, *doc)
		changeCount++

		if changeCount > 100 {
			return ErrTooManyChanges
		}
		return nil
	})

	if err == ErrTooManyChanges {
		return err
	}

	if len(changedDocs) > 0 {
		return s.storage.Index.UpdateBatch(changedDocs)
	}
	return nil
}
