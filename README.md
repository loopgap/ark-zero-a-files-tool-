# ArkKB

ArkKB 是一个本地优先的轻量工作台，目标是把真实文件系统、多根工作区、虚拟归档和内建帮助放到同一套桌面壳里。

## 当前已落地

- 多根工作区
- 虚拟归档与文件挂接
- 目录 / 类型白名单与黑名单策略
- 文件名 / 路径 / 正文搜索
- 文本编辑、Markdown 预览、图片 / PDF / HTML 预览
- 本地帮助文档与设置面板
- Windows / Ubuntu 双端构建与 GitHub Release 流程

## 目录

- `src/bridge`：工作台桥接接口
- `src/core/config`：工作区、归档、策略配置
- `src/core/storage`：bbolt + Bluge
- `src/core/sync`：焦点驱动同步与索引
- `frontend`：React + Vite + Tauri 桌面工作台壳
- `frontend/src-tauri`：桌面壳配置与 Rust bridge
- `scripts/dev.go`：本地验证、构建、发布入口
- `docs`：产品帮助与开发说明

## 开发

```bash
go test ./...
go run scripts/dev.go doctor
go run scripts/dev.go preflight
go run scripts/dev.go build
go run scripts/dev.go release
```

## 版本与发布

- 分支职责：`main` 保持稳定，`develop` 承载开发。
- 版本由 tag 驱动：发布时使用 `vX.Y.Z` 标签（例如 `v0.1.0`）。
- `scripts/dev.go preflight` 会在构建前校验：
	- tag 与语义化版本格式
	- `frontend/package.json`、`frontend/src-tauri/Cargo.toml`、`frontend/src-tauri/tauri.conf.json` 三处版本一致性
	- Tauri `bundle.icon` 资源存在性
- GitHub Actions `release` 工作流已在 build 前增加 preflight 门禁，尽量前移双系统构建失败。

## 文档

- 架构约束：`route.md`
- 用户帮助：`docs/HELP.md`
- 开发说明：`docs/DEVELOPER.md`
- 远端发布：`.github/workflows/release.yml`
