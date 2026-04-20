package bridge

import (
	"fmt"
	"path/filepath"
	"testing"

	"arkkb/src/core/config"
	"arkkb/src/core/storage"
	"arkkb/src/utils/pathutil"
	"go.etcd.io/bbolt"
)

func TestBrowseArchiveSortsAndPaginatesFiles(t *testing.T) {
	rootPath := t.TempDir()
	bridge, cfg := newArchiveBrowseTestBridge(t, rootPath)
	cfg.VirtualFolders = []config.VirtualFolder{{ID: "vf-1", Name: "Inbox"}}

	alphaPath := mustNormalizedPath(t, filepath.Join(rootPath, "alpha.md"))
	betaPath := mustNormalizedPath(t, filepath.Join(rootPath, "beta.go"))
	gammaPath := mustNormalizedPath(t, filepath.Join(rootPath, "gamma.md"))
	cfg.RecentItems = []config.RecentItem{{Path: betaPath, LastAccessed: 500}, {Path: alphaPath, LastAccessed: 300}}
	saveArchiveBrowseConfig(t, bridge, cfg)

	saveArchiveBrowseMeta(t, bridge, storage.FileMeta{Path: alphaPath, RootID: "root-1", Name: "alpha.md", Extension: ".md", Modified: 100})
	saveArchiveBrowseMeta(t, bridge, storage.FileMeta{Path: betaPath, RootID: "root-1", Name: "beta.go", Extension: ".go", Modified: 200})
	saveArchiveBrowseMeta(t, bridge, storage.FileMeta{Path: gammaPath, RootID: "root-1", Name: "gamma.md", Extension: ".md", Modified: 300})
	attachArchiveBrowseMembership(t, bridge, alphaPath, []string{"vf-1"})
	attachArchiveBrowseMembership(t, bridge, betaPath, []string{"vf-1"})
	attachArchiveBrowseMembership(t, bridge, gammaPath, []string{"vf-1"})

	modifiedResponse, err := bridge.BrowseArchive(ArchiveBrowseRequest{SourceKind: "virtual_folder", SourceID: "vf-1", SortBy: "modified", SortDirection: "desc", PageSize: 2})
	if err != nil {
		t.Fatalf("BrowseArchive modified sort returned error: %v", err)
	}
	if modifiedResponse.TotalFiles != 3 {
		t.Fatalf("expected 3 files, got %d", modifiedResponse.TotalFiles)
	}
	if modifiedResponse.NextCursor != 2 {
		t.Fatalf("expected next cursor 2, got %d", modifiedResponse.NextCursor)
	}
	assertArchiveBrowseFileNames(t, modifiedResponse.Files, []string{"gamma.md", "beta.go"})

	lastOpenedResponse, err := bridge.BrowseArchive(ArchiveBrowseRequest{SourceKind: "virtual_folder", SourceID: "vf-1", SortBy: "lastOpened", SortDirection: "desc", PageSize: 10})
	if err != nil {
		t.Fatalf("BrowseArchive lastOpened sort returned error: %v", err)
	}
	assertArchiveBrowseFileNames(t, lastOpenedResponse.Files, []string{"beta.go", "alpha.md", "gamma.md"})

	typeResponse, err := bridge.BrowseArchive(ArchiveBrowseRequest{SourceKind: "virtual_folder", SourceID: "vf-1", SortBy: "type", SortDirection: "asc", PageSize: 10})
	if err != nil {
		t.Fatalf("BrowseArchive type sort returned error: %v", err)
	}
	assertArchiveBrowseFileNames(t, typeResponse.Files, []string{"beta.go", "alpha.md", "gamma.md"})
}

func TestBrowseArchiveBuildsDirectorySummaries(t *testing.T) {
	rootPath := t.TempDir()
	bridge, cfg := newArchiveBrowseTestBridge(t, rootPath)
	cfg.VirtualFolders = []config.VirtualFolder{{ID: "vf-1", Name: "Projects"}}
	saveArchiveBrowseConfig(t, bridge, cfg)

	rootFile := mustNormalizedPath(t, filepath.Join(rootPath, "root.md"))
	docsFile := mustNormalizedPath(t, filepath.Join(rootPath, "docs", "main.md"))
	guidesFile := mustNormalizedPath(t, filepath.Join(rootPath, "docs", "guides", "install.md"))
	notesFile := mustNormalizedPath(t, filepath.Join(rootPath, "notes", "todo.md"))

	for _, meta := range []storage.FileMeta{
		{Path: rootFile, RootID: "root-1", Name: "root.md", Extension: ".md", Modified: 10},
		{Path: docsFile, RootID: "root-1", Name: "main.md", Extension: ".md", Modified: 20},
		{Path: guidesFile, RootID: "root-1", Name: "install.md", Extension: ".md", Modified: 30},
		{Path: notesFile, RootID: "root-1", Name: "todo.md", Extension: ".md", Modified: 40},
	} {
		saveArchiveBrowseMeta(t, bridge, meta)
		attachArchiveBrowseMembership(t, bridge, meta.Path, []string{"vf-1"})
	}

	rootResponse, err := bridge.BrowseArchive(ArchiveBrowseRequest{SourceKind: "virtual_folder", SourceID: "vf-1", SortBy: "name", PageSize: 20})
	if err != nil {
		t.Fatalf("BrowseArchive root response returned error: %v", err)
	}
	if rootResponse.TotalFolders != 2 {
		t.Fatalf("expected 2 folders at root, got %d", rootResponse.TotalFolders)
	}
	if rootResponse.TotalFiles != 1 {
		t.Fatalf("expected 1 root file, got %d", rootResponse.TotalFiles)
	}
	assertArchiveBrowseFolderNames(t, rootResponse.Folders, []string{"docs", "notes"})
	assertArchiveBrowseFileNames(t, rootResponse.Files, []string{"root.md"})

	docsResponse, err := bridge.BrowseArchive(ArchiveBrowseRequest{SourceKind: "virtual_folder", SourceID: "vf-1", FolderPath: "docs", SortBy: "name", PageSize: 20})
	if err != nil {
		t.Fatalf("BrowseArchive docs response returned error: %v", err)
	}
	if docsResponse.TotalFolders != 1 {
		t.Fatalf("expected 1 folder in docs, got %d", docsResponse.TotalFolders)
	}
	if docsResponse.TotalFiles != 1 {
		t.Fatalf("expected 1 direct file in docs, got %d", docsResponse.TotalFiles)
	}
	assertArchiveBrowseFolderNames(t, docsResponse.Folders, []string{"guides"})
	assertArchiveBrowseFileNames(t, docsResponse.Files, []string{"main.md"})
}

func TestBrowseArchiveSupportsLargeAutoCategoryListings(t *testing.T) {
	rootPath := t.TempDir()
	bridge, cfg := newArchiveBrowseTestBridge(t, rootPath)
	saveArchiveBrowseConfig(t, bridge, cfg)

	for index := 0; index < 350; index++ {
		meta := storage.FileMeta{
			Path:      mustNormalizedPath(t, filepath.Join(rootPath, fmt.Sprintf("file-%03d.md", index))),
			RootID:    "root-1",
			Name:      fmt.Sprintf("file-%03d.md", index),
			Extension: ".md",
			Modified:  int64(index),
		}
		saveArchiveBrowseMeta(t, bridge, meta)
	}

	response, err := bridge.BrowseArchive(ArchiveBrowseRequest{SourceKind: "auto_category", SourceID: ".md", SortBy: "name", SortDirection: "asc", PageSize: 400})
	if err != nil {
		t.Fatalf("BrowseArchive large auto category returned error: %v", err)
	}
	if response.TotalFiles != 350 {
		t.Fatalf("expected 350 files, got %d", response.TotalFiles)
	}
	if len(response.Files) != 350 {
		t.Fatalf("expected 350 paged files, got %d", len(response.Files))
	}
	if response.NextCursor != -1 {
		t.Fatalf("expected no next cursor, got %d", response.NextCursor)
	}
	if response.Files[0].Name != "file-000.md" || response.Files[len(response.Files)-1].Name != "file-349.md" {
		t.Fatalf("unexpected name range: %s ... %s", response.Files[0].Name, response.Files[len(response.Files)-1].Name)
	}
}

func newArchiveBrowseTestBridge(t *testing.T, rootPath string) (*Bridge, *config.AppConfig) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "archive-browse.db")
	db, err := bbolt.Open(dbPath, 0o600, nil)
	if err != nil {
		t.Fatalf("open bbolt: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	kv := storage.NewKVStorage(db)
	if err := kv.Init(); err != nil {
		t.Fatalf("init kv: %v", err)
	}
	manager := &storage.StorageManager{KV: kv}
	bridge := NewBridge(manager, nil, nil)
	cfg := testConfig(t, rootPath)
	saveArchiveBrowseConfig(t, bridge, cfg)
	return bridge, cfg
}

func saveArchiveBrowseConfig(t *testing.T, bridge *Bridge, cfg *config.AppConfig) {
	t.Helper()
	if err := bridge.storage.KV.SaveAppConfig(cfg); err != nil {
		t.Fatalf("save app config: %v", err)
	}
}

func saveArchiveBrowseMeta(t *testing.T, bridge *Bridge, meta storage.FileMeta) {
	t.Helper()
	if err := bridge.storage.KV.SaveFileMeta(meta); err != nil {
		t.Fatalf("save file meta: %v", err)
	}
}

func attachArchiveBrowseMembership(t *testing.T, bridge *Bridge, path string, folderIDs []string) {
	t.Helper()
	if err := bridge.storage.KV.SetVirtualFolderMemberships(path, folderIDs); err != nil {
		t.Fatalf("save virtual folder memberships: %v", err)
	}
}

func mustNormalizedPath(t *testing.T, path string) string {
	t.Helper()
	normalized, err := pathutil.NormalizePath(path)
	if err != nil {
		t.Fatalf("normalize path %s: %v", path, err)
	}
	return normalized
}

func assertArchiveBrowseFileNames(t *testing.T, files []ArchiveBrowseFile, expected []string) {
	t.Helper()
	if len(files) != len(expected) {
		t.Fatalf("expected %d files, got %d", len(expected), len(files))
	}
	for index, name := range expected {
		if files[index].Name != name {
			t.Fatalf("expected file %d to be %s, got %s", index, name, files[index].Name)
		}
	}
}

func assertArchiveBrowseFolderNames(t *testing.T, folders []ArchiveBrowseFolder, expected []string) {
	t.Helper()
	if len(folders) != len(expected) {
		t.Fatalf("expected %d folders, got %d", len(expected), len(folders))
	}
	for index, name := range expected {
		if folders[index].Name != name {
			t.Fatalf("expected folder %d to be %s, got %s", index, name, folders[index].Name)
		}
	}
}

func TestArchiveBrowseRelativePathHandlesDriveRoot(t *testing.T) {
	roots := []config.WorkspaceRoot{{ID: "root-d", Path: "D:/", Label: "D盘"}}
	relativePath := archiveBrowseRelativePath("D:/国产单片机/项目/readme.md", "root-d", roots)
	if relativePath != "国产单片机/项目/readme.md" {
		t.Fatalf("expected relative path to trim drive root, got %s", relativePath)
	}
}
