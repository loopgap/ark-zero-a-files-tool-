package sync

import (
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"arkkb/src/core/config"
	"arkkb/src/core/storage"
	"arkkb/src/utils/charset"
	"arkkb/src/utils/pathutil"
	"github.com/blugelabs/bluge"
)

func (s *SyncEngine) SyncWorkspace(cfg *config.AppConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cfg = config.NormalizeAppConfig(cfg)
	existingMetas, err := s.storage.KV.ListFileMetas()
	if err != nil {
		return err
	}
	membershipMap, err := s.storage.KV.ListAllVirtualFolderMemberships()
	if err != nil {
		return err
	}

	existingByPath := map[string]storage.FileMeta{}
	for _, meta := range existingMetas {
		existingByPath[meta.Path] = meta
	}
	seen := map[string]bool{}
	var changedDocs []bluge.Document

	for _, root := range cfg.Workspace.Roots {
		if err := s.syncRoot(root, cfg.Policy, existingByPath, membershipMap, seen, &changedDocs); err != nil {
			return err
		}
	}
	if err := s.flushDocuments(changedDocs); err != nil {
		return err
	}
	return s.removeStaleEntries(existingMetas, seen)
}

func (s *SyncEngine) syncRoot(root config.WorkspaceRoot, policy config.PolicyConfig, existingByPath map[string]storage.FileMeta, membershipMap map[string][]string, seen map[string]bool, changedDocs *[]bluge.Document) error {
	return filepath.WalkDir(root.Path, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			if path != root.Path && IsDirectoryBlocked(root.Path, path, policy) {
				return filepath.SkipDir
			}
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}
		ext := filepath.Ext(path)
		if IsFileBlocked(root.Path, path, ext, policy) {
			return nil
		}

		normalizedPath, err := pathutil.NormalizePath(path)
		if err != nil {
			return err
		}
		seen[normalizedPath] = true

		currentMeta := storage.FileMeta{
			Path:      normalizedPath,
			RootID:    root.ID,
			Name:      d.Name(),
			Extension: ext,
			Size:      info.Size(),
			Modified:  info.ModTime().Unix(),
		}

		if existing, ok := existingByPath[normalizedPath]; ok && existing.RootID == currentMeta.RootID && existing.Modified == currentMeta.Modified && existing.Size == currentMeta.Size && existing.Extension == currentMeta.Extension {
			return nil
		}

		content, charsetName := readIndexableContent(path, currentMeta.Size, policy)
		currentMeta.Charset = charsetName
		doc := buildIndexDocument(currentMeta, membershipMap[normalizedPath], content)
		*changedDocs = append(*changedDocs, *doc)
		if len(*changedDocs) >= 128 {
			if err := s.flushDocuments(*changedDocs); err != nil {
				return err
			}
			*changedDocs = nil
		}
		return s.storage.KV.SaveFileMeta(currentMeta)
	})
}

func (s *SyncEngine) SyncPath(rootID string, path string) error {
	cfg, err := s.storage.KV.GetAppConfig()
	if err != nil {
		return err
	}
	var root config.WorkspaceRoot
	found := false
	for _, candidate := range cfg.Workspace.Roots {
		if candidate.ID == rootID {
			root = candidate
			found = true
			break
		}
	}
	if !found {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	normalizedPath, err := pathutil.NormalizePath(path)
	if err != nil {
		return err
	}
	info, err := os.Stat(path)
	if err != nil {
		_ = s.storage.KV.DeleteFileMeta(path)
		_ = s.storage.KV.DeleteVirtualFolderMemberships(path)
		return s.storage.Index.DeleteBatch([]bluge.Identifier{bluge.Identifier(normalizedPath)})
	}

	content, charsetName := readIndexableContent(path, info.Size(), cfg.Policy)
	memberships, err := s.storage.KV.GetVirtualFolderMemberships(path)
	if err != nil {
		return err
	}
	meta := storage.FileMeta{
		Path:      normalizedPath,
		RootID:    root.ID,
		Name:      filepath.Base(path),
		Extension: filepath.Ext(path),
		Charset:   charsetName,
		Size:      info.Size(),
		Modified:  info.ModTime().Unix(),
	}
	if err := s.storage.KV.SaveFileMeta(meta); err != nil {
		return err
	}
	return s.storage.Index.UpdateBatch([]bluge.Document{*buildIndexDocument(meta, memberships, content)})
}

func (s *SyncEngine) removeStaleEntries(existing []storage.FileMeta, seen map[string]bool) error {
	var ids []bluge.Identifier
	for _, meta := range existing {
		if seen[meta.Path] {
			continue
		}
		ids = append(ids, bluge.Identifier(meta.Path))
		_ = s.storage.KV.DeleteFileMeta(meta.Path)
		_ = s.storage.KV.DeleteVirtualFolderMemberships(meta.Path)
	}
	if len(ids) == 0 {
		return nil
	}
	return s.storage.Index.DeleteBatch(ids)
}

func (s *SyncEngine) flushDocuments(docs []bluge.Document) error {
	if len(docs) == 0 {
		return nil
	}
	return s.storage.Index.UpdateBatch(docs)
}

func buildIndexDocument(meta storage.FileMeta, memberships []string, body string) *bluge.Document {
	doc := bluge.NewDocument(meta.Path)
	doc.AddField(bluge.NewKeywordField("path", meta.Path).StoreValue())
	doc.AddField(bluge.NewTextField("path_text", meta.Path))
	doc.AddField(bluge.NewTextField("name", meta.Name).StoreValue())
	doc.AddField(bluge.NewKeywordField("root_id", meta.RootID).StoreValue())
	doc.AddField(bluge.NewKeywordField("ext", meta.Extension).StoreValue())
	doc.AddField(bluge.NewDateTimeField("mtime", time.Unix(meta.Modified, 0)).StoreValue())
	for _, folderID := range memberships {
		doc.AddField(bluge.NewKeywordField("virtual_folder", folderID).StoreValue())
	}
	if body != "" {
		doc.AddField(bluge.NewTextField("body", body))
	}
	return doc
}

func readIndexableContent(path string, size int64, policy config.PolicyConfig) (string, string) {
	if !ShouldIndexContent(path, size, policy) {
		return "", ""
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", ""
	}
	sample := data
	if len(sample) > 4096 {
		sample = sample[:4096]
	}
	charsetName, _ := charset.Sniff(sample)
	return string(data), charsetName
}
