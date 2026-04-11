package bridge

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"arkkb/src/core/config"
	coreSync "arkkb/src/core/sync"
	"arkkb/src/utils/pathutil"
)

func resolveCreateTarget(cfg *config.AppConfig, explicitParent string, preferredRootID string, folder *config.VirtualFolder) (string, string, error) {
	if explicitParent != "" && explicitParent != "." {
		return normalizeExistingDirectory(cfg, explicitParent)
	}

	if folder != nil && folder.PreferredParentPath != "" {
		if targetPath, rootID, err := normalizeExistingDirectory(cfg, folder.PreferredParentPath); err == nil {
			return targetPath, rootID, nil
		}
	}

	for _, root := range orderedRoots(cfg, preferredRootID, folder) {
		targetPath := preferredDirectoryForRoot(root, cfg.Policy)
		if targetPath == "" {
			continue
		}
		return targetPath, root.ID, nil
	}

	if len(cfg.Workspace.Roots) == 0 {
		return "", "", fmt.Errorf("no workspace roots available")
	}

	fallback := cfg.Workspace.Roots[0]
	return fallback.Path, fallback.ID, nil
}

func orderedRoots(cfg *config.AppConfig, preferredRootID string, folder *config.VirtualFolder) []config.WorkspaceRoot {
	seen := map[string]bool{}
	var roots []config.WorkspaceRoot
	push := func(rootID string) {
		if rootID == "" || seen[rootID] {
			return
		}
		for _, root := range cfg.Workspace.Roots {
			if root.ID == rootID {
				roots = append(roots, root)
				seen[rootID] = true
				return
			}
		}
	}

	if folder != nil {
		push(folder.PreferredRootID)
	}
	push(preferredRootID)
	push(cfg.Workspace.ActiveRootID)
	push(cfg.Workspace.DefaultRootID)
	for _, root := range cfg.Workspace.Roots {
		push(root.ID)
	}
	return roots
}

func preferredDirectoryForRoot(root config.WorkspaceRoot, policy config.PolicyConfig) string {
	if len(policy.DirectoryAllowlist) == 0 {
		return root.Path
	}

	for _, rule := range policy.DirectoryAllowlist {
		candidate := filepath.Join(root.Path, filepath.FromSlash(rule))
		if existingDirectory(candidate) && !coreSync.IsDirectoryBlocked(root.Path, candidate, policy) {
			if normalized, err := pathutil.NormalizePath(candidate); err == nil {
				return normalized
			}
		}
	}
	return root.Path
}

func normalizeExistingDirectory(cfg *config.AppConfig, path string) (string, string, error) {
	normalizedPath, err := pathutil.NormalizePath(path)
	if err != nil {
		return "", "", err
	}

	info, err := os.Stat(filepath.FromSlash(normalizedPath))
	if err != nil {
		return "", "", err
	}
	if !info.IsDir() {
		normalizedPath, err = pathutil.NormalizePath(filepath.Dir(filepath.FromSlash(normalizedPath)))
		if err != nil {
			return "", "", err
		}
	}

	rootID := rootIDForPath(cfg, normalizedPath)
	if rootID == "" {
		return "", "", fmt.Errorf("path %s is outside the workspace", normalizedPath)
	}
	rootPath := rootPathByID(cfg, rootID)
	if coreSync.IsDirectoryBlocked(rootPath, filepath.FromSlash(normalizedPath), cfg.Policy) {
		return "", "", fmt.Errorf("path %s is blocked by policy", normalizedPath)
	}
	return normalizedPath, rootID, nil
}

func existingDirectory(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func rootIDForPath(cfg *config.AppConfig, path string) string {
	normalizedPath, err := pathutil.NormalizePath(path)
	if err != nil {
		return ""
	}
	lowerPath := strings.ToLower(filepath.ToSlash(normalizedPath))
	for _, root := range cfg.Workspace.Roots {
		lowerRoot := strings.ToLower(filepath.ToSlash(root.Path))
		if lowerPath == lowerRoot || strings.HasPrefix(lowerPath, lowerRoot+"/") {
			return root.ID
		}
	}
	return ""
}

func rootPathByID(cfg *config.AppConfig, rootID string) string {
	for _, root := range cfg.Workspace.Roots {
		if root.ID == rootID {
			return root.Path
		}
	}
	return ""
}
