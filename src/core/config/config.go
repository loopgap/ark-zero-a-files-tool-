package config

import (
	"crypto/rand"
	"encoding/hex"
	"path/filepath"
	"strings"
	"time"
)

type WorkspaceRoot struct {
	ID    string `json:"id"`
	Path  string `json:"path"`
	Label string `json:"label"`
}

type WorkspaceSession struct {
	ID            string          `json:"id"`
	Name          string          `json:"name"`
	Roots         []WorkspaceRoot `json:"roots"`
	ActiveRootID  string          `json:"activeRootId"`
	DefaultRootID string          `json:"defaultRootId"`
}

type VirtualFolder struct {
	ID                  string `json:"id"`
	WorkspaceID         string `json:"workspaceId"`
	Name                string `json:"name"`
	PreferredRootID     string `json:"preferredRootId"`
	PreferredParentPath string `json:"preferredParentPath"`
	CreatedAt           int64  `json:"createdAt"`
	LastUsedAt          int64  `json:"lastUsedAt"`
}

type AutoCategory struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	Extension string `json:"extension"`
	Count     int    `json:"count"`
}

type PolicyConfig struct {
	DirectoryAllowlist []string `json:"directoryAllowlist"`
	DirectoryBlocklist []string `json:"directoryBlocklist"`
	FileTypeAllowlist  []string `json:"fileTypeAllowlist"`
	FileTypeBlocklist  []string `json:"fileTypeBlocklist"`
	MaxIndexedFileSize int64    `json:"maxIndexedFileSize"`
}

type RecentItem struct {
	Path         string `json:"path"`
	Name         string `json:"name"`
	RootID       string `json:"rootId"`
	LastAccessed int64  `json:"lastAccessed"`
}

type RecentWorkspace struct {
	Path       string `json:"path"`
	Label      string `json:"label"`
	LastOpened int64  `json:"lastOpened"`
}

type LSPSettings struct {
	Executables map[string]string `json:"executables"`
}

type AppConfig struct {
	SchemaVersion    int                    `json:"schemaVersion"`
	Language         string                 `json:"language"`
	Workspace        WorkspaceSession       `json:"workspace"`
	VirtualFolders   []VirtualFolder        `json:"virtualFolders"`
	AutoCategories   []AutoCategory         `json:"autoCategories"`
	Policy           PolicyConfig           `json:"policy"`
	RecentItems      []RecentItem           `json:"recentItems"`
	RecentWorkspaces []RecentWorkspace      `json:"recentWorkspaces"`
	ViewSettings     map[string]interface{} `json:"viewSettings"`
	LSP              LSPSettings            `json:"lsp"`

	LastWorkspace string   `json:"lastWorkspace"`
	Whitelist     []string `json:"whitelist"`
	Blacklist     []string `json:"blacklist"`
	Categories    []string `json:"categories"`
}

const CurrentSchemaVersion = 2

func DefaultPolicy() PolicyConfig {
	return PolicyConfig{
		DirectoryAllowlist: []string{},
		DirectoryBlocklist: []string{".git", "node_modules", ".svelte-kit", "dist", "bin", ".tmp"},
		FileTypeAllowlist:  []string{},
		FileTypeBlocklist:  []string{".exe", ".dll", ".so", ".dylib"},
		MaxIndexedFileSize: 1 << 20,
	}
}

func DefaultConfig() *AppConfig {
	return NormalizeAppConfig(&AppConfig{
		SchemaVersion:    CurrentSchemaVersion,
		Language:         "zh",
		VirtualFolders:   []VirtualFolder{},
		AutoCategories:   []AutoCategory{},
		Policy:           DefaultPolicy(),
		RecentItems:      []RecentItem{},
		RecentWorkspaces: []RecentWorkspace{},
		ViewSettings: map[string]interface{}{
			"theme": "minimal-dark",
		},
		LSP: LSPSettings{
			Executables: map[string]string{},
		},
		LastWorkspace: ".",
		Whitelist:     []string{},
		Blacklist:     []string{".git", "node_modules", ".svelte-kit", "dist", "bin", ".tmp"},
		Categories:    []string{},
	})
}

func NormalizeAppConfig(cfg *AppConfig) *AppConfig {
	if cfg == nil {
		return DefaultConfig()
	}
	if cfg.SchemaVersion < CurrentSchemaVersion {
		cfg.SchemaVersion = CurrentSchemaVersion
	}

	if cfg.Language == "" {
		cfg.Language = "zh"
	}
	if cfg.ViewSettings == nil {
		cfg.ViewSettings = map[string]interface{}{"theme": "minimal-dark"}
	}
	if _, ok := cfg.ViewSettings["theme"]; !ok {
		cfg.ViewSettings["theme"] = "minimal-dark"
	}
	cfg.ViewSettings["theme"] = NormalizeThemeName(stringValue(cfg.ViewSettings["theme"]))
	if cfg.LSP.Executables == nil {
		cfg.LSP.Executables = map[string]string{}
	}
	if cfg.VirtualFolders == nil {
		cfg.VirtualFolders = []VirtualFolder{}
	}
	if cfg.AutoCategories == nil {
		cfg.AutoCategories = []AutoCategory{}
	}

	mergePolicyDefaults(&cfg.Policy, cfg.Whitelist, cfg.Blacklist)
	normalizeWorkspace(cfg)
	normalizeVirtualFolders(cfg)
	normalizeRecents(cfg)

	return cfg
}

func mergePolicyDefaults(policy *PolicyConfig, oldWhitelist []string, oldBlacklist []string) {
	defaults := DefaultPolicy()
	if policy.MaxIndexedFileSize <= 0 {
		policy.MaxIndexedFileSize = defaults.MaxIndexedFileSize
	}
	if len(policy.DirectoryBlocklist) == 0 {
		policy.DirectoryBlocklist = append([]string{}, defaults.DirectoryBlocklist...)
	}
	if len(policy.FileTypeBlocklist) == 0 {
		policy.FileTypeBlocklist = append([]string{}, defaults.FileTypeBlocklist...)
	}
	if len(policy.DirectoryAllowlist) == 0 && len(oldWhitelist) > 0 {
		policy.DirectoryAllowlist = append([]string{}, oldWhitelist...)
	}
	if len(policy.DirectoryBlocklist) == 0 && len(oldBlacklist) > 0 {
		policy.DirectoryBlocklist = append([]string{}, oldBlacklist...)
	}

	policy.DirectoryAllowlist = normalizeRules(policy.DirectoryAllowlist)
	policy.DirectoryBlocklist = normalizeRules(policy.DirectoryBlocklist)
	policy.FileTypeAllowlist = normalizeExtRules(policy.FileTypeAllowlist)
	policy.FileTypeBlocklist = normalizeExtRules(policy.FileTypeBlocklist)
	if policy.DirectoryAllowlist == nil {
		policy.DirectoryAllowlist = []string{}
	}
	if policy.DirectoryBlocklist == nil {
		policy.DirectoryBlocklist = []string{}
	}
	if policy.FileTypeAllowlist == nil {
		policy.FileTypeAllowlist = []string{}
	}
	if policy.FileTypeBlocklist == nil {
		policy.FileTypeBlocklist = []string{}
	}
}

func normalizeWorkspace(cfg *AppConfig) {
	if cfg.Workspace.ID == "" {
		cfg.Workspace.ID = NewID("workspace")
	}
	if cfg.Workspace.Name == "" {
		cfg.Workspace.Name = "ArkKB Workspace"
	}
	if cfg.Workspace.Roots == nil {
		cfg.Workspace.Roots = []WorkspaceRoot{}
	}
	if len(cfg.Workspace.Roots) == 0 && cfg.LastWorkspace != "" && cfg.LastWorkspace != "." {
		cfg.Workspace.Roots = []WorkspaceRoot{
			NewWorkspaceRoot(cfg.LastWorkspace),
		}
	}
	for i := range cfg.Workspace.Roots {
		if cfg.Workspace.Roots[i].ID == "" {
			cfg.Workspace.Roots[i].ID = NewID("root")
		}
		if cfg.Workspace.Roots[i].Label == "" {
			cfg.Workspace.Roots[i].Label = filepath.Base(cfg.Workspace.Roots[i].Path)
			if cfg.Workspace.Roots[i].Label == "" {
				cfg.Workspace.Roots[i].Label = cfg.Workspace.Roots[i].Path
			}
		}
	}
	if len(cfg.Workspace.Roots) > 0 {
		if cfg.Workspace.ActiveRootID == "" {
			cfg.Workspace.ActiveRootID = cfg.Workspace.Roots[0].ID
		}
		if cfg.Workspace.DefaultRootID == "" {
			cfg.Workspace.DefaultRootID = cfg.Workspace.Roots[0].ID
		}
		cfg.LastWorkspace = cfg.Workspace.Roots[0].Path
	}
}

func normalizeVirtualFolders(cfg *AppConfig) {
	if cfg.VirtualFolders == nil {
		cfg.VirtualFolders = []VirtualFolder{}
	}
	if len(cfg.VirtualFolders) == 0 && len(cfg.Categories) > 0 {
		for _, name := range cfg.Categories {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			cfg.VirtualFolders = append(cfg.VirtualFolders, VirtualFolder{
				ID:        NewID("vf"),
				Name:      name,
				CreatedAt: time.Now().Unix(),
			})
		}
	}
	for i := range cfg.VirtualFolders {
		if cfg.VirtualFolders[i].ID == "" {
			cfg.VirtualFolders[i].ID = NewID("vf")
		}
		if cfg.VirtualFolders[i].WorkspaceID == "" {
			cfg.VirtualFolders[i].WorkspaceID = cfg.Workspace.ID
		}
		if cfg.VirtualFolders[i].PreferredRootID == "" && len(cfg.Workspace.Roots) > 0 {
			cfg.VirtualFolders[i].PreferredRootID = cfg.Workspace.ActiveRootID
			if cfg.VirtualFolders[i].PreferredRootID == "" {
				cfg.VirtualFolders[i].PreferredRootID = cfg.Workspace.Roots[0].ID
			}
		}
		if cfg.VirtualFolders[i].CreatedAt == 0 {
			cfg.VirtualFolders[i].CreatedAt = time.Now().Unix()
		}
		if cfg.VirtualFolders[i].LastUsedAt == 0 {
			cfg.VirtualFolders[i].LastUsedAt = cfg.VirtualFolders[i].CreatedAt
		}
	}
}

func normalizeRecents(cfg *AppConfig) {
	if cfg.RecentItems == nil {
		cfg.RecentItems = []RecentItem{}
	}
	if cfg.RecentWorkspaces == nil {
		cfg.RecentWorkspaces = []RecentWorkspace{}
	}
}

func NewID(prefix string) string {
	var b [6]byte
	if _, err := rand.Read(b[:]); err != nil {
		return prefix + "-" + strings.ReplaceAll(time.Now().Format("150405.000000000"), ".", "")
	}
	return prefix + "-" + hex.EncodeToString(b[:])
}

func NewWorkspaceRoot(path string) WorkspaceRoot {
	label := filepath.Base(path)
	if label == "" {
		label = path
	}
	return WorkspaceRoot{
		ID:    NewID("root"),
		Path:  path,
		Label: label,
	}
}

func normalizeRules(rules []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, rule := range rules {
		rule = strings.TrimSpace(strings.ToLower(filepath.ToSlash(rule)))
		if rule == "" || seen[rule] {
			continue
		}
		seen[rule] = true
		out = append(out, rule)
	}
	return out
}

func normalizeExtRules(rules []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, rule := range rules {
		rule = strings.TrimSpace(strings.ToLower(rule))
		if rule == "" {
			continue
		}
		if !strings.HasPrefix(rule, ".") {
			rule = "." + rule
		}
		if seen[rule] {
			continue
		}
		seen[rule] = true
		out = append(out, rule)
	}
	return out
}

func NormalizeThemeName(theme string) string {
	switch strings.TrimSpace(strings.ToLower(theme)) {
	case "light", "minimal-light":
		return "minimal-light"
	case "dark", "oled", "minimal-dark":
		fallthrough
	default:
		return "minimal-dark"
	}
}

func stringValue(value interface{}) string {
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return text
}
