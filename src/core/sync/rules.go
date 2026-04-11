package sync

import (
	"path/filepath"
	"strings"

	"arkkb/src/core/config"
	"arkkb/src/utils/pathutil"
)

var textIndexExtensions = map[string]bool{
	".c": true, ".cpp": true, ".css": true, ".go": true, ".h": true, ".html": true,
	".java": true, ".js": true, ".json": true, ".log": true, ".md": true, ".py": true,
	".rs": true, ".sh": true, ".sql": true, ".svelte": true, ".toml": true, ".ts": true,
	".tsx": true, ".txt": true, ".xml": true, ".yaml": true, ".yml": true,
}

func MatchDirectoryRule(rootPath, targetPath, rule string) bool {
	rule = strings.TrimSpace(strings.ToLower(filepath.ToSlash(rule)))
	if rule == "" {
		return false
	}

	normalizedRoot, err := pathutil.NormalizePath(rootPath)
	if err != nil {
		return false
	}
	normalizedTarget, err := pathutil.NormalizePath(targetPath)
	if err != nil {
		return false
	}

	rel, err := filepath.Rel(filepath.FromSlash(normalizedRoot), filepath.FromSlash(normalizedTarget))
	if err != nil {
		return false
	}
	rel = strings.ToLower(filepath.ToSlash(rel))
	segments := strings.Split(rel, "/")

	if strings.Contains(rule, "/") {
		return rel == rule || strings.HasPrefix(rel, rule+"/")
	}
	for _, segment := range segments {
		if segment == rule {
			return true
		}
	}
	return false
}

func MatchFileTypeRule(ext string, rules []string) bool {
	ext = strings.TrimSpace(strings.ToLower(ext))
	if ext == "" {
		return false
	}
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	for _, rule := range rules {
		if strings.TrimSpace(strings.ToLower(rule)) == ext {
			return true
		}
	}
	return false
}

func IsDirectoryBlocked(rootPath, targetPath string, policy config.PolicyConfig) bool {
	for _, rule := range policy.DirectoryBlocklist {
		if MatchDirectoryRule(rootPath, targetPath, rule) {
			return true
		}
	}
	return false
}

func IsDirectoryAllowlisted(rootPath, targetPath string, policy config.PolicyConfig) bool {
	for _, rule := range policy.DirectoryAllowlist {
		if MatchDirectoryRule(rootPath, targetPath, rule) {
			return true
		}
	}
	return false
}

func IsFileBlocked(rootPath, targetPath, ext string, policy config.PolicyConfig) bool {
	if MatchFileTypeRule(ext, policy.FileTypeBlocklist) {
		return true
	}
	return IsDirectoryBlocked(rootPath, filepath.Dir(targetPath), policy)
}

func IsFileAllowlisted(rootPath, targetPath, ext string, policy config.PolicyConfig) bool {
	if MatchFileTypeRule(ext, policy.FileTypeAllowlist) {
		return true
	}
	return IsDirectoryAllowlisted(rootPath, filepath.Dir(targetPath), policy)
}

func ShouldIndexContent(path string, size int64, policy config.PolicyConfig) bool {
	ext := strings.ToLower(filepath.Ext(path))
	if size > policy.MaxIndexedFileSize {
		return false
	}
	return textIndexExtensions[ext]
}
