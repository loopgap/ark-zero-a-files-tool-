package storage

import (
	"encoding/json"
	"fmt"
	"sort"

	"arkkb/src/core/config"
	"arkkb/src/utils/pathutil"
	"go.etcd.io/bbolt"
)

var configBucket = []byte("config")
var metaBucket = []byte("file_meta")
var membershipBucket = []byte("virtual_memberships")

type FileMeta struct {
	Path      string `json:"path"`
	RootID    string `json:"rootId"`
	Name      string `json:"name"`
	Extension string `json:"extension"`
	Charset   string `json:"charset"`
	Size      int64  `json:"size"`
	Modified  int64  `json:"modified"`
}

type KVStorage struct {
	db *bbolt.DB
}

func NewKVStorage(db *bbolt.DB) *KVStorage {
	return &KVStorage{db: db}
}

func (s *KVStorage) Init() error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists(configBucket); err != nil {
			return err
		}
		if _, err := tx.CreateBucketIfNotExists(metaBucket); err != nil {
			return err
		}
		_, err := tx.CreateBucketIfNotExists(membershipBucket)
		return err
	})
}

// Config Operations
func (s *KVStorage) SaveAppConfig(cfg *config.AppConfig) error {
	cfg = config.NormalizeAppConfig(cfg)
	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	return s.db.Update(func(tx *bbolt.Tx) error {
		return tx.Bucket(configBucket).Put([]byte("app_config"), data)
	})
}

func (s *KVStorage) GetAppConfig() (*config.AppConfig, error) {
	var cfg config.AppConfig
	err := s.db.View(func(tx *bbolt.Tx) error {
		data := tx.Bucket(configBucket).Get([]byte("app_config"))
		if data == nil {
			cfg = *config.DefaultConfig()
			return nil
		}
		return json.Unmarshal(data, &cfg)
	})
	if err != nil {
		return nil, err
	}
	return config.NormalizeAppConfig(&cfg), nil
}

// Meta Operations
func (s *KVStorage) SaveFileMeta(meta FileMeta) error {
	normalizedPath, err := pathutil.NormalizePath(meta.Path)
	if err != nil {
		return err
	}
	meta.Path = normalizedPath

	data, err := json.Marshal(meta)
	if err != nil {
		return err
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		return tx.Bucket(metaBucket).Put([]byte(normalizedPath), data)
	})
}

func (s *KVStorage) GetFileMeta(path string) (*FileMeta, error) {
	normalizedPath, err := pathutil.NormalizePath(path)
	if err != nil {
		return nil, err
	}

	var meta FileMeta
	err = s.db.View(func(tx *bbolt.Tx) error {
		data := tx.Bucket(metaBucket).Get([]byte(normalizedPath))
		if data == nil {
			return fmt.Errorf("file meta not found: %s", normalizedPath)
		}
		return json.Unmarshal(data, &meta)
	})
	if err != nil {
		return nil, err
	}
	return &meta, nil
}

func (s *KVStorage) ListFileMetas() ([]FileMeta, error) {
	var metas []FileMeta
	err := s.db.View(func(tx *bbolt.Tx) error {
		return tx.Bucket(metaBucket).ForEach(func(_, value []byte) error {
			var meta FileMeta
			if err := json.Unmarshal(value, &meta); err != nil {
				return err
			}
			metas = append(metas, meta)
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(metas, func(i, j int) bool { return metas[i].Path < metas[j].Path })
	return metas, nil
}

func (s *KVStorage) DeleteFileMeta(path string) error {
	normalizedPath, err := pathutil.NormalizePath(path)
	if err != nil {
		return err
	}
	return s.db.Update(func(tx *bbolt.Tx) error {
		return tx.Bucket(metaBucket).Delete([]byte(normalizedPath))
	})
}

// Virtual Folder Membership Operations
func (s *KVStorage) SetVirtualFolderMemberships(path string, folderIDs []string) error {
	normalizedPath, err := pathutil.NormalizePath(path)
	if err != nil {
		return err
	}
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(membershipBucket)
		folderIDs = normalizeStringList(folderIDs)
		if len(folderIDs) == 0 {
			return b.Delete([]byte(normalizedPath))
		}
		data, err := json.Marshal(folderIDs)
		if err != nil {
			return err
		}
		return b.Put([]byte(normalizedPath), data)
	})
}

func (s *KVStorage) GetVirtualFolderMemberships(path string) ([]string, error) {
	normalizedPath, err := pathutil.NormalizePath(path)
	if err != nil {
		return nil, err
	}
	var folderIDs []string
	err = s.db.View(func(tx *bbolt.Tx) error {
		data := tx.Bucket(membershipBucket).Get([]byte(normalizedPath))
		if data == nil {
			folderIDs = []string{}
			return nil
		}
		return json.Unmarshal(data, &folderIDs)
	})
	return normalizeStringList(folderIDs), err
}

func (s *KVStorage) AddVirtualFolderMembership(path string, folderID string) error {
	folderIDs, err := s.GetVirtualFolderMemberships(path)
	if err != nil {
		return err
	}
	folderIDs = append(folderIDs, folderID)
	return s.SetVirtualFolderMemberships(path, folderIDs)
}

func (s *KVStorage) RemoveVirtualFolderMembership(path string, folderID string) error {
	folderIDs, err := s.GetVirtualFolderMemberships(path)
	if err != nil {
		return err
	}
	var next []string
	for _, id := range folderIDs {
		if id != folderID {
			next = append(next, id)
		}
	}
	return s.SetVirtualFolderMemberships(path, next)
}

func (s *KVStorage) DeleteVirtualFolderMemberships(path string) error {
	return s.SetVirtualFolderMemberships(path, nil)
}

func (s *KVStorage) ListAllVirtualFolderMemberships() (map[string][]string, error) {
	result := map[string][]string{}
	err := s.db.View(func(tx *bbolt.Tx) error {
		return tx.Bucket(membershipBucket).ForEach(func(key, value []byte) error {
			var folderIDs []string
			if err := json.Unmarshal(value, &folderIDs); err != nil {
				return err
			}
			result[string(key)] = normalizeStringList(folderIDs)
			return nil
		})
	})
	return result, err
}

// Raw KV Operations
func (s *KVStorage) Put(key, value []byte) error {
	if len(value) > 4096 {
		return fmt.Errorf("value exceeds the 4KB lightweight limit (size: %d)", len(value))
	}
	return s.db.Update(func(tx *bbolt.Tx) error {
		return tx.Bucket(configBucket).Put(key, value)
	})
}

func (s *KVStorage) Get(key []byte) ([]byte, error) {
	var val []byte
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(configBucket)
		val = b.Get(key)
		return nil
	})
	return val, err
}

func (s *KVStorage) Delete(key []byte) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		return tx.Bucket(configBucket).Delete(key)
	})
}

func normalizeStringList(items []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, item := range items {
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		out = append(out, item)
	}
	sort.Strings(out)
	return out
}
