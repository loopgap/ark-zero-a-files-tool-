package bridge

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"arkkb/src/core/config"
	"arkkb/src/core/storage"
	coreSync "arkkb/src/core/sync"
	"arkkb/src/utils/pathutil"
)

type TreeNode struct {
	ID               string     `json:"id"`
	Name             string     `json:"name"`
	Path             string     `json:"path"`
	Kind             string     `json:"kind"`
	RootID           string     `json:"rootId"`
	Children         []TreeNode `json:"children,omitempty"`
	Expanded         bool       `json:"expanded,omitempty"`
	VirtualFolderIDs []string   `json:"virtualFolderIds,omitempty"`
	Extension        string     `json:"extension,omitempty"`
}

type SearchHit struct {
	Path             string   `json:"path"`
	Name             string   `json:"name"`
	RootID           string   `json:"rootId"`
	VirtualFolderIDs []string `json:"virtualFolderIds"`
	MatchKind        string   `json:"matchKind"`
	Extension        string   `json:"extension"`
}

type SearchOptions struct {
	Keyword         string `json:"keyword"`
	RootID          string `json:"rootId"`
	VirtualFolderID string `json:"virtualFolderId"`
	AutoCategory    string `json:"autoCategory"`
	MatchField      string `json:"matchField"`
	CaseSensitive   bool   `json:"caseSensitive"`
	FileType        string `json:"fileType"`
	Limit           int    `json:"limit"`
}

type ArchiveBrowseRequest struct {
	SourceKind    string `json:"sourceKind"`
	SourceID      string `json:"sourceId"`
	FolderPath    string `json:"folderPath"`
	Query         string `json:"query"`
	SearchMode    string `json:"searchMode"`
	SortBy        string `json:"sortBy"`
	SortDirection string `json:"sortDirection"`
	PageSize      int    `json:"pageSize"`
	Cursor        int    `json:"cursor"`
}

type ArchiveBrowseFolder struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	Count int    `json:"count"`
}

type ArchiveBrowseFile struct {
	Path             string   `json:"path"`
	Name             string   `json:"name"`
	RootID           string   `json:"rootId"`
	VirtualFolderIDs []string `json:"virtualFolderIds"`
	MatchKind        string   `json:"matchKind"`
	Extension        string   `json:"extension"`
	RelativePath     string   `json:"relativePath"`
	Directory        string   `json:"directory"`
	ModifiedAt       int64    `json:"modifiedAt"`
	LastOpenedAt     int64    `json:"lastOpenedAt"`
}

type ArchiveBrowseResponse struct {
	Folders           []ArchiveBrowseFolder `json:"folders"`
	Files             []ArchiveBrowseFile   `json:"files"`
	TotalFiles        int                   `json:"totalFiles"`
	TotalFolders      int                   `json:"totalFolders"`
	NextCursor        int                   `json:"nextCursor"`
	CurrentFolderPath string                `json:"currentFolderPath"`
}

type HelpDoc struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Path  string `json:"path"`
}

type WorkbenchState struct {
	Workspace        config.WorkspaceSession  `json:"workspace"`
	PhysicalRoots    []TreeNode               `json:"physicalRoots"`
	VirtualFolders   []config.VirtualFolder   `json:"virtualFolders"`
	AutoCategories   []config.AutoCategory    `json:"autoCategories"`
	Policy           config.PolicyConfig      `json:"policy"`
	RecentItems      []config.RecentItem      `json:"recentItems"`
	RecentWorkspaces []config.RecentWorkspace `json:"recentWorkspaces"`
	HelpDocs         []HelpDoc                `json:"helpDocs"`
	Language         string                   `json:"language"`
	Theme            string                   `json:"theme"`
}

type CreateTarget struct {
	RootID string `json:"rootId"`
	Path   string `json:"path"`
}

func (b *Bridge) GetWorkbenchState() (*WorkbenchState, error) {
	cfg, err := b.storage.KV.GetAppConfig()
	if err != nil {
		return nil, err
	}

	memberships, err := b.storage.KV.ListAllVirtualFolderMemberships()
	if err != nil {
		return nil, err
	}

	state := &WorkbenchState{
		Workspace:        cfg.Workspace,
		PhysicalRoots:    []TreeNode{},
		VirtualFolders:   cfg.VirtualFolders,
		AutoCategories:   cfg.AutoCategories,
		Policy:           cfg.Policy,
		RecentItems:      cfg.RecentItems,
		RecentWorkspaces: cfg.RecentWorkspaces,
		HelpDocs:         b.readableHelpDocs(),
		Language:         cfg.Language,
		Theme:            config.NormalizeThemeName(stringValue(cfg.ViewSettings["theme"], "minimal-dark")),
	}

	for _, root := range cfg.Workspace.Roots {
		node, treeErr := b.buildTreeNode(root, root.Path, memberships, cfg.Policy, 1)
		if treeErr != nil {
			continue
		}
		state.PhysicalRoots = append(state.PhysicalRoots, node)
	}
	return state, nil
}

func (b *Bridge) ListWorkspaceRoots() ([]config.WorkspaceRoot, error) {
	cfg, err := b.storage.KV.GetAppConfig()
	if err != nil {
		return nil, err
	}
	return cfg.Workspace.Roots, nil
}

func (b *Bridge) ListRecentItems() ([]config.RecentItem, error) {
	cfg, err := b.storage.KV.GetAppConfig()
	if err != nil {
		return nil, err
	}
	return cfg.RecentItems, nil
}

func (b *Bridge) ListVirtualFolderItems(folderID string) ([]SearchHit, error) {
	return b.SearchFiles(SearchOptions{VirtualFolderID: folderID})
}

func (b *Bridge) BrowseArchive(request ArchiveBrowseRequest) (*ArchiveBrowseResponse, error) {
	cfg, err := b.storage.KV.GetAppConfig()
	if err != nil {
		return nil, err
	}
	metas, err := b.storage.KV.ListFileMetas()
	if err != nil {
		return nil, err
	}
	memberships, err := b.storage.KV.ListAllVirtualFolderMemberships()
	if err != nil {
		return nil, err
	}

	recentAccess := map[string]int64{}
	for _, item := range cfg.RecentItems {
		recentAccess[item.Path] = item.LastAccessed
	}

	files, err := b.buildArchiveBrowseFiles(cfg, metas, memberships, request, recentAccess)
	if err != nil {
		return nil, err
	}
	return buildArchiveBrowseResponse(files, request), nil
}

func (b *Bridge) ListAutoCategories() ([]config.AutoCategory, error) {
	return b.computeAutoCategories()
}

func (b *Bridge) SavePolicyConfig(policy config.PolicyConfig) error {
	cfg, err := b.storage.KV.GetAppConfig()
	if err != nil {
		return err
	}
	cfg.Policy = policy
	if err := b.storage.KV.SaveAppConfig(cfg); err != nil {
		return err
	}
	b.queueWorkspaceSync(cfg)
	return nil
}

func (b *Bridge) ReadHelpDoc(docID string) (string, error) {
	data, err := b.readHelpDoc(docID)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (b *Bridge) AddWorkspaceRoot(path string) (*config.WorkspaceRoot, error) {
	if path == "" {
		selected, err := b.pickDirectory("Add Workspace Root")
		if err != nil {
			return nil, err
		}
		path = selected
	}
	if path == "" {
		return nil, nil
	}

	normalizedPath, err := canonicalizeMaybeMissingPath(path, false)
	if err != nil {
		return nil, err
	}
	cfg, err := b.storage.KV.GetAppConfig()
	if err != nil {
		return nil, err
	}
	for _, root := range cfg.Workspace.Roots {
		if samePath(root.Path, normalizedPath) {
			cfg.Workspace.ActiveRootID = root.ID
			cfg.RecentWorkspaces = upsertRecentWorkspace(cfg.RecentWorkspaces, normalizedPath, root.Label)
			return &root, b.storage.KV.SaveAppConfig(cfg)
		}
	}

	root := config.NewWorkspaceRoot(normalizedPath)
	cfg.Workspace.Roots = append(cfg.Workspace.Roots, root)
	cfg.Workspace.ActiveRootID = root.ID
	if cfg.Workspace.DefaultRootID == "" {
		cfg.Workspace.DefaultRootID = root.ID
	}
	cfg.RecentWorkspaces = upsertRecentWorkspace(cfg.RecentWorkspaces, normalizedPath, root.Label)
	if err := b.storage.KV.SaveAppConfig(cfg); err != nil {
		return nil, err
	}
	b.queueWorkspaceSync(cfg)
	return &root, nil
}

func (b *Bridge) RemoveWorkspaceRoot(rootID string) error {
	cfg, err := b.storage.KV.GetAppConfig()
	if err != nil {
		return err
	}
	var next []config.WorkspaceRoot
	for _, root := range cfg.Workspace.Roots {
		if root.ID != rootID {
			next = append(next, root)
		}
	}
	cfg.Workspace.Roots = next
	if len(cfg.Workspace.Roots) > 0 {
		cfg.Workspace.ActiveRootID = cfg.Workspace.Roots[0].ID
		cfg.Workspace.DefaultRootID = cfg.Workspace.Roots[0].ID
		cfg.LastWorkspace = cfg.Workspace.Roots[0].Path
	} else {
		cfg.Workspace.ActiveRootID = ""
		cfg.Workspace.DefaultRootID = ""
		cfg.LastWorkspace = "."
	}
	if err := b.storage.KV.SaveAppConfig(cfg); err != nil {
		return err
	}
	b.queueWorkspaceSync(cfg)
	return nil
}

func (b *Bridge) SetActiveRoot(rootID string) error {
	cfg, err := b.storage.KV.GetAppConfig()
	if err != nil {
		return err
	}
	if rootPath := rootPathByID(cfg, rootID); rootPath != "" {
		cfg.LastWorkspace = rootPath
		cfg.RecentWorkspaces = upsertRecentWorkspace(cfg.RecentWorkspaces, rootPath, filepath.Base(filepath.FromSlash(rootPath)))
	}
	cfg.Workspace.ActiveRootID = rootID
	return b.storage.KV.SaveAppConfig(cfg)
}

func (b *Bridge) SetDefaultRoot(rootID string) error {
	cfg, err := b.storage.KV.GetAppConfig()
	if err != nil {
		return err
	}
	cfg.Workspace.DefaultRootID = rootID
	return b.storage.KV.SaveAppConfig(cfg)
}

func (b *Bridge) ResolveDefaultCreateTarget(parentPath string, preferredRootID string, virtualFolderID string) (*CreateTarget, error) {
	cfg, err := b.storage.KV.GetAppConfig()
	if err != nil {
		return nil, err
	}
	targetPath, rootID, err := resolveCreateTarget(cfg, parentPath, preferredRootID, b.findVirtualFolder(cfg, virtualFolderID))
	if err != nil {
		return nil, err
	}
	return &CreateTarget{RootID: rootID, Path: targetPath}, nil
}

func (b *Bridge) CreateVirtualFolder(name string) (*config.VirtualFolder, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("virtual folder name is required")
	}
	cfg, err := b.storage.KV.GetAppConfig()
	if err != nil {
		return nil, err
	}
	folder := config.VirtualFolder{
		ID:              config.NewID("vf"),
		WorkspaceID:     cfg.Workspace.ID,
		Name:            name,
		PreferredRootID: cfg.Workspace.ActiveRootID,
		CreatedAt:       time.Now().Unix(),
		LastUsedAt:      time.Now().Unix(),
	}
	if targetPath, rootID, err := resolveCreateTarget(cfg, "", folder.PreferredRootID, nil); err == nil {
		folder.PreferredRootID = rootID
		folder.PreferredParentPath = targetPath
	}
	cfg.VirtualFolders = append(cfg.VirtualFolders, folder)
	if err := b.storage.KV.SaveAppConfig(cfg); err != nil {
		return nil, err
	}
	return &folder, nil
}

func (b *Bridge) RenameVirtualFolder(folderID string, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("virtual folder name is required")
	}
	cfg, err := b.storage.KV.GetAppConfig()
	if err != nil {
		return err
	}
	for i := range cfg.VirtualFolders {
		if cfg.VirtualFolders[i].ID == folderID {
			cfg.VirtualFolders[i].Name = name
			cfg.VirtualFolders[i].LastUsedAt = time.Now().Unix()
			return b.storage.KV.SaveAppConfig(cfg)
		}
	}
	return nil
}

func (b *Bridge) DeleteVirtualFolder(folderID string) error {
	cfg, err := b.storage.KV.GetAppConfig()
	if err != nil {
		return err
	}
	var next []config.VirtualFolder
	for _, folder := range cfg.VirtualFolders {
		if folder.ID != folderID {
			next = append(next, folder)
		}
	}
	cfg.VirtualFolders = next
	if err := b.storage.KV.SaveAppConfig(cfg); err != nil {
		return err
	}

	memberships, err := b.storage.KV.ListAllVirtualFolderMemberships()
	if err != nil {
		return err
	}
	for path, folderIDs := range memberships {
		var filtered []string
		for _, id := range folderIDs {
			if id != folderID {
				filtered = append(filtered, id)
			}
		}
		_ = b.storage.KV.SetVirtualFolderMemberships(path, filtered)
		if b.syncEngine != nil {
			_ = b.syncEngine.SyncPath(b.rootIDForPath(path), filepath.FromSlash(path))
		}
	}
	return nil
}

func (b *Bridge) AttachFileToVirtualFolder(path string, folderID string) error {
	resolvedPath, err := b.ResolveWorkspacePath(path)
	if err != nil {
		return err
	}
	if err := b.storage.KV.AddVirtualFolderMembership(resolvedPath.CanonicalPath, folderID); err != nil {
		return err
	}
	if err := b.updateVirtualFolderTarget(folderID, resolvedPath.CanonicalPath, resolvedPath.RootID); err != nil {
		return err
	}
	if b.syncEngine != nil {
		return b.syncEngine.SyncPath(resolvedPath.RootID, filepath.FromSlash(resolvedPath.CanonicalPath))
	}
	return nil
}

func (b *Bridge) DetachFileFromVirtualFolder(path string, folderID string) error {
	resolvedPath, err := b.ResolveWorkspacePath(path)
	if err != nil {
		return err
	}
	if err := b.storage.KV.RemoveVirtualFolderMembership(resolvedPath.CanonicalPath, folderID); err != nil {
		return err
	}
	if b.syncEngine != nil {
		return b.syncEngine.SyncPath(resolvedPath.RootID, filepath.FromSlash(resolvedPath.CanonicalPath))
	}
	return nil
}

func (b *Bridge) CreateVirtualFile(folderID string, name string, parentPath string, preferredRootID string) (*SearchHit, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("file name is required")
	}

	cfg, err := b.storage.KV.GetAppConfig()
	if err != nil {
		return nil, err
	}
	folder := b.findVirtualFolder(cfg, folderID)
	if folder == nil {
		return nil, fmt.Errorf("virtual folder not found")
	}
	targetPath, _, err := resolveCreateTarget(cfg, parentPath, preferredRootID, folder)
	if err != nil {
		return nil, err
	}
	resolvedTarget, err := resolveWorkspacePath(cfg, filepath.Join(filepath.FromSlash(targetPath), name), resolvePathOptions{AllowMissingLeaf: true})
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.FromSlash(resolvedTarget.CanonicalPath), []byte(""), 0644); err != nil {
		return nil, err
	}
	if err := b.storage.KV.AddVirtualFolderMembership(resolvedTarget.CanonicalPath, folderID); err != nil {
		return nil, err
	}
	if err := b.updateVirtualFolderTarget(folderID, resolvedTarget.CanonicalPath, resolvedTarget.RootID); err != nil {
		return nil, err
	}
	if b.syncEngine != nil {
		if err := b.syncEngine.SyncPath(resolvedTarget.RootID, filepath.FromSlash(resolvedTarget.CanonicalPath)); err != nil {
			return nil, err
		}
	}
	_ = b.RecordRecentItem(resolvedTarget.CanonicalPath)

	return &SearchHit{
		Path:             resolvedTarget.CanonicalPath,
		Name:             name,
		RootID:           resolvedTarget.RootID,
		VirtualFolderIDs: []string{folderID},
		MatchKind:        "name",
		Extension:        filepath.Ext(name),
	}, nil
}

func (b *Bridge) SearchFiles(options SearchOptions) ([]SearchHit, error) {
	if normalizeSearchMatchField(options.MatchField) == "content" && options.CaseSensitive {
		return nil, fmt.Errorf("case-sensitive content search is not supported")
	}

	documents, err := b.storage.Index.SearchDocumentsWithQuery(context.Background(), storage.SearchQuery{
		Keyword:         strings.TrimSpace(options.Keyword),
		Limit:           searchResultLimit(options),
		Fields:          searchQueryFields(options),
		RootID:          options.RootID,
		VirtualFolderID: options.VirtualFolderID,
		Extension:       normalizeSearchExtension(options.AutoCategory),
	})
	if err != nil {
		return nil, err
	}

	cfg, err := b.storage.KV.GetAppConfig()
	if err != nil {
		return nil, err
	}
	recentScore := map[string]int{}
	for idx, item := range cfg.RecentItems {
		recentScore[item.Path] = len(cfg.RecentItems) - idx
	}

	var results []SearchHit
	var scored []struct {
		hit   SearchHit
		score int
	}

	for _, doc := range documents {
		hit := SearchHit{
			Path:             doc.Path,
			Name:             doc.Name,
			RootID:           doc.RootID,
			VirtualFolderIDs: append([]string{}, doc.VirtualFolderIDs...),
			Extension:        doc.Extension,
		}
		if hit.Path == "" {
			continue
		}
		if options.RootID != "" && hit.RootID != options.RootID {
			continue
		}
		if options.VirtualFolderID != "" && !containsString(hit.VirtualFolderIDs, options.VirtualFolderID) {
			continue
		}
		if options.AutoCategory != "" && !searchTypeMatches(hit.Extension, options.AutoCategory) {
			continue
		}
		ok, matchKind := b.matchesSearchHit(hit, options)
		if !ok {
			continue
		}
		hit.MatchKind = matchKind
		score := b.scoreSearchHit(cfg, hit, recentScore, options.RootID, options.VirtualFolderID, options)
		scored = append(scored, struct {
			hit   SearchHit
			score int
		}{hit: hit, score: score})
	}

	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].score == scored[j].score {
			return strings.ToLower(scored[i].hit.Name) < strings.ToLower(scored[j].hit.Name)
		}
		return scored[i].score > scored[j].score
	})

	for _, item := range scored {
		results = append(results, item.hit)
	}
	return results, nil
}

func (b *Bridge) buildArchiveBrowseFiles(
	cfg *config.AppConfig,
	metas []storage.FileMeta,
	memberships map[string][]string,
	request ArchiveBrowseRequest,
	recentAccess map[string]int64,
) ([]ArchiveBrowseFile, error) {
	sourceKind := normalizeArchiveSourceKind(request.SourceKind)
	sourceID := strings.TrimSpace(request.SourceID)
	if sourceKind == "" || sourceID == "" {
		return []ArchiveBrowseFile{}, nil
	}

	candidates := make([]ArchiveBrowseFile, 0, len(metas))
	for _, meta := range metas {
		folderIDs := memberships[meta.Path]
		if !archiveMetaInSource(sourceKind, sourceID, meta, folderIDs) {
			continue
		}
		candidates = append(candidates, archiveBrowseFileFromMeta(meta, folderIDs, cfg.Workspace.Roots, recentAccess[meta.Path]))
	}

	query := strings.TrimSpace(request.Query)
	if query == "" {
		return candidates, nil
	}

	if normalizeArchiveSearchMode(request.SearchMode) != "content" {
		filtered := make([]ArchiveBrowseFile, 0, len(candidates))
		for _, file := range candidates {
			if ok, matchKind := matchArchiveBrowseQuickSearch(file, query); ok {
				file.MatchKind = matchKind
				filtered = append(filtered, file)
			}
		}
		return filtered, nil
	}

	searchOptions := SearchOptions{
		Keyword:    query,
		RootID:     "",
		MatchField: "content",
		Limit:      maxInt(len(candidates), request.Cursor+normalizedArchivePageSize(request.PageSize)+1),
	}
	if sourceKind == "virtual_folder" {
		searchOptions.VirtualFolderID = sourceID
	} else {
		searchOptions.AutoCategory = sourceID
	}
	results, err := b.SearchFiles(searchOptions)
	if err != nil {
		return nil, err
	}

	matches := map[string]string{}
	for _, hit := range results {
		matches[hit.Path] = hit.MatchKind
	}

	filtered := make([]ArchiveBrowseFile, 0, len(results))
	for _, file := range candidates {
		matchKind, ok := matches[file.Path]
		if !ok {
			continue
		}
		file.MatchKind = matchKind
		filtered = append(filtered, file)
	}
	return filtered, nil
}

func buildArchiveBrowseResponse(files []ArchiveBrowseFile, request ArchiveBrowseRequest) *ArchiveBrowseResponse {
	folderPath := normalizeArchiveFolderPath(request.FolderPath)
	currentSegments := splitArchivePath(folderPath)
	directoryCounts := map[string]int{}
	currentFiles := make([]ArchiveBrowseFile, 0, len(files))

	for _, file := range files {
		fileDirectory := normalizeArchiveFolderPath(file.Directory)
		folderSegments := splitArchivePath(fileDirectory)
		if !archiveFolderHasPrefix(folderSegments, currentSegments) {
			continue
		}
		if len(folderSegments) > len(currentSegments) {
			nextDirectory := joinArchiveFolderPath(folderPath, folderSegments[len(currentSegments)])
			directoryCounts[nextDirectory]++
			continue
		}
		currentFiles = append(currentFiles, file)
	}

	folders := make([]ArchiveBrowseFolder, 0, len(directoryCounts))
	for path, count := range directoryCounts {
		folders = append(folders, ArchiveBrowseFolder{
			Name:  archiveFolderBaseName(path),
			Path:  path,
			Count: count,
		})
	}
	sort.SliceStable(folders, func(i, j int) bool {
		return strings.ToLower(folders[i].Name) < strings.ToLower(folders[j].Name)
	})

	sortArchiveBrowseFiles(currentFiles, request.SortBy, request.SortDirection)
	pageSize := normalizedArchivePageSize(request.PageSize)
	cursor := request.Cursor
	if cursor < 0 {
		cursor = 0
	}
	if cursor > len(currentFiles) {
		cursor = len(currentFiles)
	}
	end := cursor + pageSize
	nextCursor := -1
	if end < len(currentFiles) {
		nextCursor = end
	} else {
		end = len(currentFiles)
	}

	pagedFiles := append([]ArchiveBrowseFile(nil), currentFiles[cursor:end]...)
	return &ArchiveBrowseResponse{
		Folders:           folders,
		Files:             pagedFiles,
		TotalFiles:        len(currentFiles),
		TotalFolders:      len(folders),
		NextCursor:        nextCursor,
		CurrentFolderPath: folderPath,
	}
}

func archiveMetaInSource(sourceKind string, sourceID string, meta storage.FileMeta, folderIDs []string) bool {
	switch sourceKind {
	case "virtual_folder":
		return containsString(folderIDs, sourceID)
	case "auto_category":
		return normalizeSearchExtension(meta.Extension) == normalizeSearchExtension(sourceID)
	default:
		return false
	}
}

func archiveBrowseFileFromMeta(meta storage.FileMeta, folderIDs []string, roots []config.WorkspaceRoot, lastOpenedAt int64) ArchiveBrowseFile {
	relativePath := archiveBrowseRelativePath(meta.Path, meta.RootID, roots)
	return ArchiveBrowseFile{
		Path:             meta.Path,
		Name:             meta.Name,
		RootID:           meta.RootID,
		VirtualFolderIDs: append([]string{}, folderIDs...),
		MatchKind:        "name",
		Extension:        meta.Extension,
		RelativePath:     relativePath,
		Directory:        archiveBrowseDirectory(relativePath),
		ModifiedAt:       meta.Modified,
		LastOpenedAt:     lastOpenedAt,
	}
}

func archiveBrowseRelativePath(path string, rootID string, roots []config.WorkspaceRoot) string {
	normalizedPath := filepath.ToSlash(path)
	var root *config.WorkspaceRoot
	for idx := range roots {
		if roots[idx].ID == rootID {
			root = &roots[idx]
			break
		}
	}
	if root == nil {
		return archiveFolderBaseName(normalizedPath)
	}

	rootPath := strings.TrimRight(filepath.ToSlash(root.Path), "/")
	if rootPath == "" {
		rootPath = "/"
	}
	rootPathLower := normalizeRootBoundary(root.Path)
	normalizedPathLower := strings.ToLower(normalizedPath)
	relativePath := archiveFolderBaseName(normalizedPath)
	switch {
	case normalizedPathLower == rootPathLower:
		relativePath = root.Label
	case rootPathLower == "/" && strings.HasPrefix(normalizedPathLower, "/"):
		relativePath = strings.TrimPrefix(normalizedPath, "/")
	case strings.HasPrefix(normalizedPathLower, rootPathLower+"/"):
		relativePath = strings.TrimPrefix(normalizedPath, rootPath+"/")
	}
	if len(roots) > 1 {
		return joinArchiveFolderPath(root.Label, relativePath)
	}
	return relativePath
}

func archiveBrowseDirectory(relativePath string) string {
	directory := filepath.ToSlash(filepath.Dir(relativePath))
	if directory == "." {
		return ""
	}
	return normalizeArchiveFolderPath(directory)
}

func matchArchiveBrowseQuickSearch(file ArchiveBrowseFile, query string) (bool, string) {
	needle := strings.TrimSpace(query)
	if needle == "" {
		return true, "name"
	}
	extension := strings.TrimPrefix(strings.ToLower(file.Extension), ".")
	normalizedNeedle := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(query)), ".")

	if searchSmartMatch(file.Name, needle, false) {
		return true, "name"
	}
	if file.Directory != "" && searchSmartMatch(file.Directory, needle, false) {
		return true, "directory"
	}
	if extension != "" && (extension == normalizedNeedle || strings.Contains(extension, normalizedNeedle)) {
		return true, "type"
	}
	if searchSmartMatch(file.RelativePath, needle, false) {
		return true, "path"
	}
	return false, ""
}

func normalizeArchiveSourceKind(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "virtual_folder":
		return "virtual_folder"
	case "auto_category":
		return "auto_category"
	default:
		return ""
	}
}

func normalizeArchiveSearchMode(value string) string {
	if strings.EqualFold(strings.TrimSpace(value), "content") {
		return "content"
	}
	return "quick"
}

func normalizeArchiveFolderPath(value string) string {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\\", "/"))
	value = strings.Trim(value, "/")
	if value == "." {
		return ""
	}
	return value
}

func normalizedArchivePageSize(pageSize int) int {
	if pageSize <= 0 {
		return 80
	}
	if pageSize > 500 {
		return 500
	}
	return pageSize
}

func splitArchivePath(path string) []string {
	normalized := normalizeArchiveFolderPath(path)
	if normalized == "" {
		return []string{}
	}
	return strings.Split(normalized, "/")
}

func archiveFolderHasPrefix(folderSegments []string, currentSegments []string) bool {
	if len(currentSegments) > len(folderSegments) {
		return false
	}
	for idx := range currentSegments {
		if folderSegments[idx] != currentSegments[idx] {
			return false
		}
	}
	return true
}

func joinArchiveFolderPath(base string, segment string) string {
	base = normalizeArchiveFolderPath(base)
	segment = normalizeArchiveFolderPath(segment)
	switch {
	case base == "":
		return segment
	case segment == "":
		return base
	default:
		return base + "/" + segment
	}
}

func archiveFolderBaseName(path string) string {
	normalized := normalizeArchiveFolderPath(path)
	if normalized == "" {
		return ""
	}
	parts := strings.Split(normalized, "/")
	return parts[len(parts)-1]
}

func sortArchiveBrowseFiles(files []ArchiveBrowseFile, sortBy string, sortDirection string) {
	sortBy = normalizeArchiveSortBy(sortBy)
	direction := normalizeArchiveSortDirection(sortDirection)
	sort.SliceStable(files, func(i, j int) bool {
		cmp := compareArchiveBrowseFiles(files[i], files[j], sortBy)
		if cmp == 0 {
			cmp = compareArchiveBrowseFiles(files[i], files[j], "name")
		}
		if cmp == 0 {
			cmp = strings.Compare(strings.ToLower(files[i].RelativePath), strings.ToLower(files[j].RelativePath))
		}
		if direction == "desc" {
			return cmp > 0
		}
		return cmp < 0
	})
}

func compareArchiveBrowseFiles(left ArchiveBrowseFile, right ArchiveBrowseFile, sortBy string) int {
	switch sortBy {
	case "modified":
		return compareInt64(left.ModifiedAt, right.ModifiedAt)
	case "lastOpened":
		return compareInt64(left.LastOpenedAt, right.LastOpenedAt)
	case "type":
		if cmp := strings.Compare(strings.ToLower(left.Extension), strings.ToLower(right.Extension)); cmp != 0 {
			return cmp
		}
		return strings.Compare(strings.ToLower(left.Name), strings.ToLower(right.Name))
	case "directory":
		if cmp := strings.Compare(strings.ToLower(left.Directory), strings.ToLower(right.Directory)); cmp != 0 {
			return cmp
		}
		return strings.Compare(strings.ToLower(left.Name), strings.ToLower(right.Name))
	default:
		return strings.Compare(strings.ToLower(left.Name), strings.ToLower(right.Name))
	}
}

func normalizeArchiveSortBy(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "modified":
		return "modified"
	case "lastopened":
		return "lastOpened"
	case "type":
		return "type"
	case "directory":
		return "directory"
	default:
		return "name"
	}
}

func normalizeArchiveSortDirection(value string) string {
	if strings.EqualFold(strings.TrimSpace(value), "desc") {
		return "desc"
	}
	return "asc"
}

func compareInt64(left int64, right int64) int {
	switch {
	case left < right:
		return -1
	case left > right:
		return 1
	default:
		return 0
	}
}

func maxInt(left int, right int) int {
	if left > right {
		return left
	}
	return right
}
func searchResultLimit(options SearchOptions) int {
	if options.Limit > 0 {
		return options.Limit
	}
	return 500
}
func searchQueryFields(options SearchOptions) []string {
	switch normalizeSearchMatchField(options.MatchField) {
	case "name":
		return []string{"name"}
	case "path", "directory":
		return []string{"path_text"}
	case "content":
		return []string{"body"}
	case "type":
		return []string{"path_text"}
	default:
		if options.CaseSensitive {
			return []string{"name", "path_text"}
		}
		return []string{"name", "path_text", "body"}
	}
}
func (b *Bridge) RecordRecentItem(path string) error {
	cfg, err := b.storage.KV.GetAppConfig()
	if err != nil {
		return err
	}
	normalizedPath, err := canonicalizeMaybeMissingPath(path, false)
	if err != nil {
		return err
	}
	name := filepath.Base(normalizedPath)
	rootID := b.rootIDForPath(normalizedPath)

	var next []config.RecentItem
	next = append(next, config.RecentItem{
		Path:         normalizedPath,
		Name:         name,
		RootID:       rootID,
		LastAccessed: time.Now().Unix(),
	})
	for _, item := range cfg.RecentItems {
		if item.Path != normalizedPath {
			next = append(next, item)
		}
		if len(next) >= 20 {
			break
		}
	}
	cfg.RecentItems = next
	return b.storage.KV.SaveAppConfig(cfg)
}

func (b *Bridge) ListDirectoryChildren(rootID string, dirPath string) ([]TreeNode, error) {
	cfg, err := b.storage.KV.GetAppConfig()
	if err != nil {
		return nil, err
	}
	rootPath := rootPathByID(cfg, rootID)
	if rootPath == "" {
		return []TreeNode{}, nil
	}
	var root config.WorkspaceRoot
	for _, candidate := range cfg.Workspace.Roots {
		if candidate.ID == rootID {
			root = candidate
			break
		}
	}
	memberships, err := b.storage.KV.ListAllVirtualFolderMemberships()
	if err != nil {
		return nil, err
	}
	return b.buildDirectoryChildren(root, dirPath, memberships, cfg.Policy, 0)
}

func (b *Bridge) buildTreeNode(root config.WorkspaceRoot, currentPath string, memberships map[string][]string, policy config.PolicyConfig, depth int) (TreeNode, error) {
	info, err := os.Stat(currentPath)
	if err != nil {
		return TreeNode{}, err
	}

	normalizedPath, err := pathutil.NormalizePath(currentPath)
	if err != nil {
		return TreeNode{}, err
	}
	node := TreeNode{
		ID:               normalizedPath,
		Name:             info.Name(),
		Path:             normalizedPath,
		Kind:             "file",
		RootID:           root.ID,
		VirtualFolderIDs: memberships[normalizedPath],
		Extension:        filepath.Ext(currentPath),
	}
	if currentPath == root.Path {
		node.Name = root.Label
		node.Kind = "workspace-root"
	}
	if info.IsDir() {
		if currentPath != root.Path {
			node.Kind = "dir"
		}
		node.Expanded = currentPath == root.Path
		if depth > 0 || currentPath == root.Path {
			children, childErr := b.buildDirectoryChildren(root, currentPath, memberships, policy, depth-1)
			if childErr == nil {
				node.Children = children
			}
		}
	}
	return node, nil
}

func (b *Bridge) buildDirectoryChildren(root config.WorkspaceRoot, currentPath string, memberships map[string][]string, policy config.PolicyConfig, depth int) ([]TreeNode, error) {
	entries, err := os.ReadDir(currentPath)
	if err != nil {
		return []TreeNode{}, nil
	}
	children := make([]TreeNode, 0, len(entries))
	for _, entry := range entries {
		childPath := filepath.Join(currentPath, entry.Name())
		if entry.IsDir() && coreSync.IsDirectoryBlocked(root.Path, childPath, policy) {
			continue
		}
		if !entry.IsDir() && coreSync.IsFileBlocked(root.Path, childPath, filepath.Ext(childPath), policy) {
			continue
		}
		childNode, childErr := b.buildTreeNode(root, childPath, memberships, policy, depth)
		if childErr == nil {
			children = append(children, childNode)
		}
	}
	sort.SliceStable(children, func(i, j int) bool {
		if children[i].Kind == children[j].Kind {
			return strings.ToLower(children[i].Name) < strings.ToLower(children[j].Name)
		}
		return children[i].Kind != "file"
	})
	return children, nil
}

func (b *Bridge) resolveCreateParent(parentDir string) (string, string, error) {
	cfg, err := b.storage.KV.GetAppConfig()
	if err != nil {
		return "", "", err
	}
	targetPath, rootID, err := resolveCreateTarget(cfg, parentDir, cfg.Workspace.ActiveRootID, nil)
	if err == nil {
		return filepath.FromSlash(targetPath), rootID, nil
	}

	selected, err := b.pickDirectory("Select Workspace Root")
	if err != nil {
		return "", "", err
	}
	if strings.TrimSpace(selected) == "" {
		return "", "", fmt.Errorf("no workspace root selected")
	}
	root, err := b.AddWorkspaceRoot(selected)
	if err != nil || root == nil {
		if err != nil {
			return "", "", err
		}
		return "", "", fmt.Errorf("failed to create workspace root")
	}
	return filepath.FromSlash(root.Path), root.ID, nil
}

func (b *Bridge) rootPathByID(rootID string) string {
	cfg, err := b.storage.KV.GetAppConfig()
	if err != nil {
		return ""
	}
	return rootPathByID(cfg, rootID)
}

func (b *Bridge) rootIDForPath(path string) string {
	cfg, err := b.storage.KV.GetAppConfig()
	if err != nil {
		return ""
	}
	resolvedPath, err := resolveWorkspacePath(cfg, path, resolvePathOptions{AllowMissingLeaf: true})
	if err == nil {
		return resolvedPath.RootID
	}
	return cfg.Workspace.ActiveRootID
}

func (b *Bridge) mustPolicy() config.PolicyConfig {
	cfg, err := b.storage.KV.GetAppConfig()
	if err != nil {
		return config.DefaultPolicy()
	}
	return cfg.Policy
}

func (b *Bridge) pickDirectory(title string) (string, error) {
	if b.dirPicker == nil {
		return "", os.ErrInvalid
	}
	return b.dirPicker(title)
}

func (b *Bridge) scoreSearchHit(cfg *config.AppConfig, hit SearchHit, recentScore map[string]int, activeRootID string, virtualFolderID string, options SearchOptions) int {
	score := recentScore[hit.Path] * 10
	rootPath := b.rootPathByID(hit.RootID)
	if activeRootID != "" && hit.RootID == activeRootID {
		score += 120
	}
	if virtualFolderID != "" && containsString(hit.VirtualFolderIDs, virtualFolderID) {
		score += 100
	}
	if coreSync.IsFileAllowlisted(rootPath, hit.Path, hit.Extension, cfg.Policy) {
		score += 60
	}
	if coreSync.IsDirectoryAllowlisted(rootPath, filepath.Dir(filepath.FromSlash(hit.Path)), cfg.Policy) {
		score += 40
	}
	switch hit.MatchKind {
	case "name":
		score += 30
	case "directory":
		score += 26
	case "type":
		score += 24
	case "path":
		score += 20
	case "content":
		score += 10
	}
	score += smartKeywordScore(options.Keyword, hit, options.MatchField, options.CaseSensitive)
	return score
}

func smartKeywordScore(keyword string, hit SearchHit, matchField string, caseSensitive bool) int {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return 0
	}

	name := normalizeSearchValue(hit.Name, caseSensitive)
	path := normalizeSearchValue(filepath.ToSlash(hit.Path), caseSensitive)
	directory := normalizeSearchValue(filepath.ToSlash(filepath.Dir(hit.Path)), caseSensitive)
	extension := normalizeSearchExtension(hit.Extension)
	keywordNormalized := normalizeSearchValue(keyword, caseSensitive)
	matchField = normalizeSearchMatchField(matchField)
	baseName := strings.TrimSuffix(name, extension)
	tokens := searchKeywordTokensNormalized(keywordNormalized)
	score := 0

	if name == keywordNormalized {
		score += 240
	} else if baseName == keywordNormalized {
		score += 220
	}

	if strings.HasPrefix(name, keywordNormalized) {
		score += 150
	}
	if strings.Contains(name, keywordNormalized) {
		score += 90
	}
	if strings.HasPrefix(directory, keywordNormalized) {
		score += 82
	}
	if strings.Contains(directory, keywordNormalized) {
		score += 54
	}
	if strings.HasPrefix(path, keywordNormalized) {
		score += 70
	}
	if strings.Contains(path, keywordNormalized) {
		score += 45
	}
	if extension != "" && (keywordNormalized == extension || keywordNormalized == strings.TrimPrefix(extension, ".")) {
		score += 55
	}
	if isSubsequence(name, keywordNormalized) {
		score += 50
	}
	if isSubsequence(directory, keywordNormalized) {
		score += 30
	}
	if isSubsequence(path, keywordNormalized) {
		score += 24
	}
	if len(tokens) > 1 {
		if allSearchTokensMatch(baseName, tokens) {
			score += 150
		}
		if allSearchTokensMatch(name, tokens) {
			score += 110
		}
		if allSearchTokensMatch(directory, tokens) {
			score += 72
		}
		if allSearchTokensMatch(path, tokens) {
			score += 64
		}
	}

	for _, token := range tokens {
		if token == "" {
			continue
		}
		if strings.HasPrefix(name, token) {
			score += 24
		}
		if strings.Contains(name, token) {
			score += 18
		}
		if strings.HasPrefix(directory, token) {
			score += 12
		}
		if strings.Contains(directory, token) {
			score += 10
		}
		if strings.HasPrefix(path, token) {
			score += 12
		}
		if strings.Contains(path, token) {
			score += 10
		}
	}
	if matchField == "content" {
		score += 16
	}
	return score
}

func splitSearchToken(r rune) bool {
	switch {
	case r >= 'a' && r <= 'z':
		return false
	case r >= '0' && r <= '9':
		return false
	default:
		return true
	}
}

func searchKeywordTokensNormalized(keyword string) []string {
	rawTokens := strings.FieldsFunc(strings.TrimSpace(keyword), splitSearchToken)
	if len(rawTokens) == 0 {
		return nil
	}
	out := make([]string, 0, len(rawTokens))
	seen := map[string]bool{}
	for _, token := range rawTokens {
		token = strings.TrimSpace(token)
		if len(token) < 2 || seen[token] {
			continue
		}
		seen[token] = true
		out = append(out, token)
	}
	return out
}

func allSearchTokensMatch(text string, tokens []string) bool {
	if len(tokens) == 0 {
		return false
	}
	for _, token := range tokens {
		if !strings.Contains(text, token) && !isSubsequence(text, token) {
			return false
		}
	}
	return true
}

func searchSmartMatch(haystack string, needle string, caseSensitive bool) bool {
	if searchContains(haystack, needle, caseSensitive) {
		return true
	}
	left := normalizeSearchValue(haystack, caseSensitive)
	right := normalizeSearchValue(strings.TrimSpace(needle), caseSensitive)
	if right == "" {
		return true
	}
	if isSubsequence(left, right) {
		return true
	}
	tokens := searchKeywordTokensNormalized(right)
	if len(tokens) <= 1 {
		return false
	}
	return allSearchTokensMatch(left, tokens)
}

func isSubsequence(text string, query string) bool {
	if query == "" {
		return true
	}
	cursor := 0
	for _, char := range text {
		if cursor < len(query) && byte(char) == query[cursor] {
			cursor++
			if cursor == len(query) {
				return true
			}
		}
	}
	return false
}

func normalizeSearchMatchField(field string) string {
	switch strings.ToLower(strings.TrimSpace(field)) {
	case "name":
		return "name"
	case "path":
		return "path"
	case "directory":
		return "directory"
	case "type":
		return "type"
	case "content":
		return "content"
	default:
		return "all"
	}
}

func normalizeSearchExtension(extension string) string {
	extension = strings.ToLower(strings.TrimSpace(extension))
	if extension == "" {
		return ""
	}
	if !strings.HasPrefix(extension, ".") {
		return "." + extension
	}
	return extension
}

func normalizeSearchValue(value string, caseSensitive bool) string {
	if caseSensitive {
		return value
	}
	return strings.ToLower(value)
}

func searchContains(haystack string, needle string, caseSensitive bool) bool {
	if strings.TrimSpace(needle) == "" {
		return true
	}
	left := normalizeSearchValue(haystack, caseSensitive)
	right := normalizeSearchValue(strings.TrimSpace(needle), caseSensitive)
	return strings.Contains(left, right)
}

func searchTypeMatches(extension string, keyword string) bool {
	if strings.TrimSpace(keyword) == "" {
		return true
	}
	target := normalizeSearchExtension(extension)
	query := normalizeSearchExtension(keyword)
	if target == "" || query == "" {
		return false
	}
	return target == query || strings.TrimPrefix(target, ".") == strings.TrimPrefix(query, ".")
}

func (b *Bridge) matchesSearchHit(hit SearchHit, options SearchOptions) (bool, string) {
	matchField := normalizeSearchMatchField(options.MatchField)
	keyword := strings.TrimSpace(options.Keyword)
	fileType := normalizeSearchExtension(options.FileType)
	directory := filepath.ToSlash(filepath.Dir(hit.Path))
	name := hit.Name
	path := filepath.ToSlash(hit.Path)
	extension := normalizeSearchExtension(hit.Extension)

	if fileType != "" && !searchTypeMatches(extension, fileType) {
		return false, ""
	}

	if keyword == "" {
		switch matchField {
		case "directory":
			return true, "directory"
		case "type":
			return true, "type"
		case "content":
			return true, "content"
		case "path":
			return true, "path"
		default:
			return true, "name"
		}
	}

	switch matchField {
	case "name":
		return searchSmartMatch(name, keyword, options.CaseSensitive), "name"
	case "path":
		return searchSmartMatch(path, keyword, options.CaseSensitive), "path"
	case "directory":
		return searchSmartMatch(directory, keyword, options.CaseSensitive), "directory"
	case "type":
		return searchTypeMatches(extension, keyword), "type"
	case "content":
		return true, "content"
	default:
		if searchSmartMatch(name, keyword, options.CaseSensitive) {
			return true, "name"
		}
		if searchSmartMatch(directory, keyword, options.CaseSensitive) {
			return true, "directory"
		}
		if searchTypeMatches(extension, keyword) {
			return true, "type"
		}
		if searchSmartMatch(path, keyword, options.CaseSensitive) {
			return true, "path"
		}
		if options.CaseSensitive {
			return false, ""
		}
		return true, "content"
	}
}

func defaultHelpDocs() []HelpDoc {
	return []HelpDoc{
		{ID: "help", Title: "Help", Path: "docs/HELP.md"},
		{ID: "developer", Title: "Developer", Path: "docs/DEVELOPER.md"},
	}
}

func (b *Bridge) readableHelpDocs() []HelpDoc {
	docs := defaultHelpDocs()
	filtered := make([]HelpDoc, 0, len(docs))
	for _, doc := range docs {
		if _, err := b.readHelpDoc(doc.ID); err == nil {
			filtered = append(filtered, doc)
		}
	}
	return filtered
}

func helpDocPath(docID string) (string, error) {
	switch docID {
	case "", "help":
		return filepath.Join("docs", "HELP.md"), nil
	case "developer":
		return filepath.Join("docs", "DEVELOPER.md"), nil
	default:
		return "", fmt.Errorf("unknown help doc: %s", docID)
	}
}

func (b *Bridge) readHelpDoc(docID string) ([]byte, error) {
	if b.readHelp != nil {
		if data, err := b.readHelp(docID); err == nil {
			return data, nil
		}
	}
	return readHelpDocFromDisk(docID)
}

func readHelpDocFromDisk(docID string) ([]byte, error) {
	docPath, err := helpDocPath(docID)
	if err != nil {
		return nil, err
	}
	if data, readErr := os.ReadFile(docPath); readErr == nil {
		return data, nil
	}
	executablePath, execErr := os.Executable()
	if execErr != nil {
		return nil, execErr
	}
	altPath := filepath.Join(filepath.Dir(executablePath), "..", docPath)
	return os.ReadFile(altPath)
}
func upsertRecentWorkspace(items []config.RecentWorkspace, path string, label string) []config.RecentWorkspace {
	now := time.Now().Unix()
	next := []config.RecentWorkspace{{
		Path:       path,
		Label:      label,
		LastOpened: now,
	}}
	for _, item := range items {
		if !samePath(item.Path, path) {
			next = append(next, item)
		}
		if len(next) >= 12 {
			break
		}
	}
	return next
}

func samePath(left string, right string) bool {
	return strings.EqualFold(filepath.ToSlash(left), filepath.ToSlash(right))
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func stringValue(input interface{}, fallback string) string {
	value, ok := input.(string)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func (b *Bridge) computeAutoCategories() ([]config.AutoCategory, error) {
	metas, err := b.storage.KV.ListFileMetas()
	if err != nil {
		return nil, err
	}

	counts := map[string]int{}
	for _, meta := range metas {
		ext := strings.ToLower(strings.TrimSpace(meta.Extension))
		if ext == "" {
			continue
		}
		counts[ext]++
	}

	out := make([]config.AutoCategory, 0, len(counts))
	for ext, count := range counts {
		label := strings.TrimPrefix(ext, ".")
		if label == "" {
			label = "unknown"
		}
		out = append(out, config.AutoCategory{
			ID:        ext,
			Label:     strings.ToUpper(label),
			Extension: ext,
			Count:     count,
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Count == out[j].Count {
			return out[i].Label < out[j].Label
		}
		return out[i].Count > out[j].Count
	})
	return out, nil
}

func (b *Bridge) findVirtualFolder(cfg *config.AppConfig, folderID string) *config.VirtualFolder {
	if folderID == "" {
		return nil
	}
	for i := range cfg.VirtualFolders {
		if cfg.VirtualFolders[i].ID == folderID {
			return &cfg.VirtualFolders[i]
		}
	}
	return nil
}

func (b *Bridge) updateVirtualFolderTarget(folderID string, path string, rootID string) error {
	cfg, err := b.storage.KV.GetAppConfig()
	if err != nil {
		return err
	}
	dir := filepath.Dir(filepath.FromSlash(path))
	normalizedDir, err := pathutil.NormalizePath(dir)
	if err != nil {
		return err
	}
	for i := range cfg.VirtualFolders {
		if cfg.VirtualFolders[i].ID == folderID {
			cfg.VirtualFolders[i].PreferredRootID = rootID
			cfg.VirtualFolders[i].PreferredParentPath = normalizedDir
			cfg.VirtualFolders[i].LastUsedAt = time.Now().Unix()
			return b.storage.KV.SaveAppConfig(cfg)
		}
	}
	return nil
}
