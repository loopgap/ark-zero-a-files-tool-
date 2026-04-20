# ArkKB

ArkKB 是一个本地优先的轻量桌面工作台，用来把真实文件系统、多根工作区、虚拟归档、全文检索和内建帮助整合进同一套桌面壳。

## 当前已落地

- 多根工作区与活动根目录 / 默认根目录管理
- 虚拟归档、自动分类与文件挂接
- 归档展开浏览台：分类 / 归档 / 目录三段导航、排序、分组、快速预览
- 文件名 / 路径 / 目录 / 类型 / 正文检索
- 文本编辑、Markdown 预览、图片 / PDF / HTML 预览
- 本地帮助文档、设置面板、同步与索引策略
- GitHub CI、双平台 release、校验与产物上传链路

## 目录

- `.github/workflows`：CI 与 release 工作流
- `docs`：帮助、开发文档、工作区架构、CI/release 说明、wiki 索引
- `frontend`：React + Vite 工作台前端与 Tauri 桌面壳
- `src/bridge`：RPC 编排、工作区动作、归档浏览和路径边界校验
- `src/core/config`：工作区、归档、策略与最近项配置
- `src/core/storage`：bbolt + Bluge 元数据与索引
- `src/core/sync`：焦点驱动同步与索引写入
- `src/sidecar`：本地 HTTP 与 RPC 服务入口
- `scripts/dev.go`：本地 doctor / preflight / build / release 统一入口
- `route.md`：运行时架构基线
- `VERSION`：发布版本源

## 本地开发

```bash
go run scripts/dev.go ci
go test ./...
go run scripts/dev.go doctor
go run scripts/dev.go preflight
go run scripts/dev.go build
go run scripts/dev.go release
```

## 产物与 Git 状态

- 前端生产产物输出到 `frontend/build/`，由 `frontend/.gitignore` 忽略。
- 本地桌面 smoke 构建输出到 `bin/`，发布整理输出到 `bin/release/<os>/`，由根 `.gitignore` 忽略。
- 临时验证产生的 `arkkb-sidecar` 也已忽略，避免污染 `git status`。
- 需要提交的始终是源码、workflow 与文档，不是构建产物目录。

## CI 与 Release

- `preflight` 现在只做校验，不会回写版本文件；发布前必须先显式提交 `VERSION`、`frontend/package.json`、`frontend/package-lock.json`、`frontend/src-tauri/tauri.conf.json` 与 `frontend/src-tauri/Cargo.toml` 的目标版本。
- `ci` 工作流负责 preflight、后端测试、前端类型检查、前端构建，以及 Windows / Ubuntu 双平台桌面 smoke build 与产物上传。
- `release` 工作流先在公共 `validate` job 完成 preflight、Go tests 和前端检查，再执行 Windows / Linux matrix 打包、checksum 汇总与 GitHub Release 发布。
- 发布顺序固定为：先提交目标版本源码并推送 `develop`，等待双平台 CI 通过，再推 `vX.Y.Z` tag 触发正式 release。

## 文档入口

- 用户帮助：`docs/HELP.md`
- 开发说明：`docs/DEVELOPER.md` (包含文档规范与 Git 约束)
- 工作区文件架构：`docs/WORKSPACE_ARCHITECTURE.md`
- CI / Release 说明：`docs/CI_RELEASE.md`
- 文档索引 / Wiki：`docs/WIKI.md`
- 运行时架构基线：`route.md`
