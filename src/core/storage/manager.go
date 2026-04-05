// Package storage provides data persistence using bbolt (KV) and Bluge (Index).
package storage

import (
	"os"
	"path/filepath"

	"github.com/blugelabs/bluge"
	"go.etcd.io/bbolt"
)

// StorageManager orchestrates both key-value (bbolt) and full-text index (Bluge) storage.
type StorageManager struct {
	KV    *KVStorage
	Index *IndexStorage
	db    *bbolt.DB
}

// NewStorageManager returns a new, uninitialized StorageManager.
func NewStorageManager() *StorageManager {
	return &StorageManager{}
}

// Init sets up the storage directory in ~/.arkkb and initializes both bbolt and Bluge engines.
func (m *StorageManager) Init() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	arkkbDir := filepath.Join(homeDir, ".arkkb")
	if err := os.MkdirAll(arkkbDir, 0755); err != nil {
		return err
	}

	// Init bbolt
	dbPath := filepath.Join(arkkbDir, "bbolt.db")
	db, err := bbolt.Open(dbPath, 0600, nil)
	if err != nil {
		return err
	}
	m.db = db
	m.KV = NewKVStorage(db)
	if err := m.KV.Init(); err != nil {
		return err
	}

	// Init Bluge
	indexPath := filepath.Join(arkkbDir, "bluge_index")
	config := bluge.DefaultConfig(indexPath)
	writer, err := bluge.OpenWriter(config)
	if err != nil {
		return err
	}
	m.Index = NewIndexStorage(writer)

	return nil
}

// Close gracefully closes all underlying storage engines.
func (m *StorageManager) Close() {
	if m.Index != nil {
		m.Index.Close()
	}
	if m.db != nil {
		m.db.Close()
	}
}
