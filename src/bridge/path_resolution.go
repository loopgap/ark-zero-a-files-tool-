package bridge

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"arkkb/src/core/config"
	"arkkb/src/utils/pathutil"
)

type ResolvedPath struct {
	InputPath     string `json:"inputPath"`
	CanonicalPath string `json:"canonicalPath"`
	RootID        string `json:"rootId"`
}

type resolvePathOptions struct {
	AllowMissingLeaf bool
}

type workspaceRootBoundary struct {
	RootID        string
	CanonicalPath string
}

func (b *Bridge) ResolveWorkspacePath(path string) (*ResolvedPath, error) {
	resolved, _, err := b.resolveWorkspacePathWithConfig(path, resolvePathOptions{})
	return resolved, err
}

func (b *Bridge) resolveWorkspacePathWithConfig(path string, options resolvePathOptions) (*ResolvedPath, *config.AppConfig, error) {
	cfg, err := b.storage.KV.GetAppConfig()
	if err != nil {
		return nil, nil, err
	}
	resolved, err := resolveWorkspacePath(cfg, path, options)
	if err != nil {
		return nil, nil, err
	}
	return resolved, cfg, nil
}

func resolveWorkspacePath(cfg *config.AppConfig, path string, options resolvePathOptions) (*ResolvedPath, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("path is required")
	}

	normalizedInput, err := pathutil.NormalizePath(path)
	if err != nil {
		return nil, err
	}
	canonicalPath, err := canonicalizeMaybeMissingPath(normalizedInput, options.AllowMissingLeaf)
	if err != nil {
		return nil, err
	}

	boundaries, err := workspaceRootBoundaries(cfg)
	if err != nil {
		return nil, err
	}
	rootID := lookupWorkspaceRootID(boundaries, canonicalPath)
	if rootID == "" {
		return nil, fmt.Errorf("path is outside the workspace")
	}

	return &ResolvedPath{
		InputPath:     normalizedInput,
		CanonicalPath: canonicalPath,
		RootID:        rootID,
	}, nil
}

func workspaceRootBoundaries(cfg *config.AppConfig) ([]workspaceRootBoundary, error) {
	boundaries := make([]workspaceRootBoundary, 0, len(cfg.Workspace.Roots))
	for _, root := range cfg.Workspace.Roots {
		canonicalPath, err := canonicalizeMaybeMissingPath(root.Path, true)
		if err != nil {
			continue
		}
		boundaries = append(boundaries, workspaceRootBoundary{RootID: root.ID, CanonicalPath: canonicalPath})
	}
	if len(boundaries) == 0 {
		return nil, fmt.Errorf("no workspace roots available")
	}
	sort.SliceStable(boundaries, func(i, j int) bool {
		return len(boundaries[i].CanonicalPath) > len(boundaries[j].CanonicalPath)
	})
	return boundaries, nil
}

func lookupWorkspaceRootID(boundaries []workspaceRootBoundary, canonicalPath string) string {
	for _, boundary := range boundaries {
		if pathWithinRoot(canonicalPath, boundary.CanonicalPath) {
			return boundary.RootID
		}
	}
	return ""
}

func pathWithinRoot(path string, root string) bool {
	normalizedPath := strings.ToLower(filepath.ToSlash(path))
	normalizedRoot := normalizeRootBoundary(root)
	if normalizedRoot == "/" {
		return strings.HasPrefix(normalizedPath, "/")
	}
	return normalizedPath == normalizedRoot || strings.HasPrefix(normalizedPath, normalizedRoot+"/")
}

func normalizeRootBoundary(root string) string {
	normalizedRoot := strings.ToLower(filepath.ToSlash(root))
	if normalizedRoot == "/" {
		return normalizedRoot
	}
	return strings.TrimRight(normalizedRoot, "/")
}

func canonicalizeMaybeMissingPath(path string, allowMissingLeaf bool) (string, error) {
	normalizedPath, err := pathutil.NormalizePath(path)
	if err != nil {
		return "", err
	}
	osPath := filepath.FromSlash(normalizedPath)

	if _, err := os.Lstat(osPath); err == nil {
		resolved, err := filepath.EvalSymlinks(osPath)
		if err != nil {
			return "", fmt.Errorf("resolve canonical path %s: %w", normalizedPath, err)
		}
		return pathutil.NormalizePath(resolved)
	} else if !os.IsNotExist(err) {
		return "", err
	}

	if !allowMissingLeaf {
		return "", os.ErrNotExist
	}

	ancestor, suffix, err := splitExistingAncestor(osPath)
	if err != nil {
		return "", err
	}
	resolvedAncestor, err := filepath.EvalSymlinks(ancestor)
	if err != nil {
		return "", fmt.Errorf("resolve canonical path %s: %w", normalizedPath, err)
	}
	joined := resolvedAncestor
	for _, segment := range suffix {
		joined = filepath.Join(joined, segment)
	}
	return pathutil.NormalizePath(joined)
}

func splitExistingAncestor(path string) (string, []string, error) {
	current := path
	var suffix []string
	for {
		if _, err := os.Lstat(current); err == nil {
			return current, suffix, nil
		} else if !os.IsNotExist(err) {
			return "", nil, err
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", nil, fmt.Errorf("path does not have an existing ancestor: %s", path)
		}
		suffix = append([]string{filepath.Base(current)}, suffix...)
		current = parent
	}
}
