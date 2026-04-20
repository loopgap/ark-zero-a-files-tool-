# ArkKB Frontend

这是 ArkKB 的桌面前端工作区，使用 React 19、TypeScript、Vite 和 Tauri 2。

## 主要职责

- 工作台 UI 与状态编排
- 工作区树、归档紧凑态与归档展开浏览台
- 文本编辑、Markdown 预览、文件预览
- 通过 Tauri bridge 调用 Go sidecar RPC

## 常用命令

```bash
npm ci
npm run check
npm run build
npm run dev
```

## 目录

- `src/main.tsx`：应用入口
- `src/lib`：共享类型、RPC helper、工作台工具函数
- `src/features/workbench`：桌面主界面
- `src/features/workbench/components/ArchiveExplorerView.tsx`：归档展开浏览器
- `src/features/workbench/components/WorkbenchBrowserContent.tsx`：左侧紧凑浏览区
- `src-tauri`：Tauri 壳、窗口与打包配置

## 产物

- Vite 生产构建输出到 `build/`
- `build/` 已由 `frontend/.gitignore` 忽略，不进入版本库
- 桌面最终打包由根目录 `scripts/dev.go` 统一驱动，不建议在这里单独维护另一套 release 脚本
