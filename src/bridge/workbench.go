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
		HelpDocs:         readableHelpDocs(),
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
	data, err := readHelpDoc(docID)
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

	normalizedPath, err := pathutil.NormalizePath(path)
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
	normalizedPath, err := pathutil.NormalizePath(path)
	if err != nil {
		return err
	}
	if err := b.storage.KV.AddVirtualFolderMembership(normalizedPath, folderID); err != nil {
		return err
	}
	if err := b.updateVirtualFolderTarget(folderID, normalizedPath, b.rootIDForPath(normalizedPath)); err != nil {
		return err
	}
	if b.syncEngine != nil {
		return b.syncEngine.SyncPath(b.rootIDForPath(normalizedPath), filepath.FromSlash(normalizedPath))
	}
	return nil
}

func (b *Bridge) DetachFileFromVirtualFolder(path string, folderID string) error {
	normalizedPath, err := pathutil.NormalizePath(path)
	if err != nil {
		return err
	}
	if err := b.storage.KV.RemoveVirtualFolderMembership(normalizedPath, folderID); err != nil {
		return err
	}
	if b.syncEngine != nil {
		return b.syncEngine.SyncPath(b.rootIDForPath(normalizedPath), filepath.FromSlash(normalizedPath))
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
	targetPath, rootID, err := resolveCreateTarget(cfg, parentPath, preferredRootID, folder)
	if err != nil {
		return nil, err
	}
	fullPath := filepath.Join(filepath.FromSlash(targetPath), name)
	if err := os.WriteFile(fullPath, []byte(""), 0644); err != nil {
		return nil, err
	}
	normalizedPath, err := pathutil.NormalizePath(fullPath)
	if err != nil {
		return nil, err
	}
	if err := b.storage.KV.AddVirtualFolderMembership(normalizedPath, folderID); err != nil {
		return nil, err
	}
	if err := b.updateVirtualFolderTarget(folderID, normalizedPath, rootID); err != nil {
		return nil, err
	}
	if b.syncEngine != nil {
		if err := b.syncEngine.SyncPath(rootID, fullPath); err != nil {
			return nil, err
		}
	}
	_ = b.RecordRecentItem(normalizedPath)

	return &SearchHit{
		Path:             normalizedPath,
		Name:             name,
		RootID:           rootID,
		VirtualFolderIDs: []string{folderID},
		MatchKind:        "name",
		Extension:        filepath.Ext(name),
	}, nil
}

func (b *Bridge) SearchFiles(options SearchOptions) ([]SearchHit, error) {
	iter, err := b.storage.Index.SearchDocuments(context.Background(), options.Keyword, 300)
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

	match, err := iter.Next()
	for err == nil && match != nil {
		var hit SearchHit
		hit.VirtualFolderIDs = []string{}
		_ = match.VisitStoredFields(func(field string, value []byte) bool {
			switch field {
			case "_id", "path":
				hit.Path = string(value)
			case "name":
				hit.Name = string(value)
			case "root_id":
				hit.RootID = string(value)
			case "ext":
				hit.Extension = string(value)
			case "virtual_folder":
				hit.VirtualFolderIDs = append(hit.VirtualFolderIDs, string(value))
			}
			return true
		})
		if hit.Path == "" {
			match, err = iter.Next()
			continue
		}
		if options.RootID != "" && hit.RootID != options.RootID {
			match, err = iter.Next()
			continue
		}
		if options.VirtualFolderID != "" && !containsString(hit.VirtualFolderIDs, options.VirtualFolderID) {
			match, err = iter.Next()
			continue
		}
		ok, matchKind := b.matchesSearchHit(hit, options)
		if !ok {
			match, err = iter.Next()
			continue
		}
		hit.MatchKind = matchKind
		score := b.scoreSearchHit(cfg, hit, recentScore, options.RootID, options.VirtualFolderID, options)
		scored = append(scored, struct {
			hit   SearchHit
			score int
		}{hit: hit, score: score})
		match, err = iter.Next()
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

func (b *Bridge) RecordRecentItem(path string) error {
	cfg, err := b.storage.KV.GetAppConfig()
	if err != nil {
		return err
	}
	normalizedPath, err := pathutil.NormalizePath(path)
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
	if rootID := rootIDForPath(cfg, path); rootID != "" {
		return rootID
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
	score := 0

	if name == keywordNormalized {
		score += 240
	} else if strings.TrimSuffix(name, extension) == keywordNormalized {
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

	for _, token := range strings.FieldsFunc(keywordNormalized, splitSearchToken) {
		if token == "" {
			continue
		}
		if strings.Contains(name, token) {
			score += 18
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
		return searchContains(name, keyword, options.CaseSensitive), "name"
	case "path":
		return searchContains(path, keyword, options.CaseSensitive), "path"
	case "directory":
		return searchContains(directory, keyword, options.CaseSensitive), "directory"
	case "type":
		return searchTypeMatches(extension, keyword), "type"
	case "content":
		if options.CaseSensitive {
			return b.fileContainsKeyword(hit.Path, keyword, true), "content"
		}
		return true, "content"
	default:
		if searchContains(name, keyword, options.CaseSensitive) {
			return true, "name"
		}
		if searchContains(directory, keyword, options.CaseSensitive) {
			return true, "directory"
		}
		if searchTypeMatches(extension, keyword) {
			return true, "type"
		}
		if searchContains(path, keyword, options.CaseSensitive) {
			return true, "path"
		}
		if options.CaseSensitive {
			return b.fileContainsKeyword(hit.Path, keyword, true), "content"
		}
		return true, "content"
	}
}

func (b *Bridge) fileContainsKeyword(path string, keyword string, caseSensitive bool) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	content := string(data)
	if caseSensitive {
		return strings.Contains(content, keyword)
	}
	return strings.Contains(strings.ToLower(content), strings.ToLower(keyword))
}

func defaultHelpDocs() []HelpDoc {
	return []HelpDoc{
		{ID: "help", Title: "Help", Path: "docs/HELP.md"},
		{ID: "developer", Title: "Developer", Path: "docs/DEVELOPER.md"},
	}
}

func readableHelpDocs() []HelpDoc {
	docs := defaultHelpDocs()
	filtered := make([]HelpDoc, 0, len(docs))
	for _, doc := range docs {
		if _, err := readHelpDoc(doc.ID); err == nil {
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

func readHelpDoc(docID string) ([]byte, error) {
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
