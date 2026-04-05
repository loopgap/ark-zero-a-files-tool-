package file

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"arkkb/src/utils/pathutil"
)

// SoftDelete 软删除机制：物理移入 ~/.arkkb/trash/ 节点。
// 1. 将物理文件移入 ~/.arkkb/trash/ 节点。
// 2. 返回操作状态，以便后续更新 Bluge 索引（添加 __DELETED__ 标记）。
func SoftDelete(path string) error {
	// 路径去异化处理
	path, err := pathutil.NormalizePath(path)
	if err != nil {
		return fmt.Errorf("failed to normalize path: %w", err)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Printf("[Tombstone] 获取 Home 目录失败: %v", err)
		return fmt.Errorf("failed to get home dir: %w", err)
	}

	trashDir := filepath.Join(homeDir, ".arkkb", "trash")
	if err := os.MkdirAll(trashDir, 0755); err != nil {
		log.Printf("[Tombstone] 无法创建回收站目录 %s: %v", trashDir, err)
		return fmt.Errorf("failed to create trash dir: %w", err)
	}

	// 获取文件名并追加时间戳，以防回收站内同名冲突
	baseName := filepath.Base(path)
	timestamp := time.Now().Format("20060102-150405")
	destPath := filepath.Join(trashDir, fmt.Sprintf("%s_%s", timestamp, baseName))

	// 执行物理移动 (Atomic Move)
	// 注意：跨分区的移动在 Rename 时可能会报错，但在大多数桌面场景下，~/.arkkb 与用户数据多在同一分区。
	if err := os.Rename(path, destPath); err != nil {
		log.Printf("[Tombstone] 软删除失败 %s -> %s: %v", path, destPath, err)
		return fmt.Errorf("failed to move file to trash: %w", err)
	}

	log.Printf("[Tombstone] 文件已软删除并移入回收站: %s -> %s", path, destPath)
	return nil
}
