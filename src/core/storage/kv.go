package storage

import (
	"encoding/json"
	"fmt"

	"arkkb/src/utils/pathutil"
	"go.etcd.io/bbolt"
)

var configBucket = []byte("config")
var metaBucket = []byte("file_meta")

type FileMeta struct {
	Path     string `json:"path"`
	Charset  string `json:"charset"`
	Size     int64  `json:"size"`
	Modified int64  `json:"modified"`
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
		_, err := tx.CreateBucketIfNotExists(metaBucket)
		return err
	})
}

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
