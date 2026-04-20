# CI and Release

## 本地命令

```bash
go run scripts/dev.go ci
go run scripts/dev.go doctor
go run scripts/dev.go preflight
go run scripts/dev.go build
go run scripts/dev.go release
```

说明：

- `preflight` 只做版本一致性和打包输入校验，不会改写版本文件。
- 发布前需要先显式提交 `VERSION`、`frontend/package.json`、`frontend/package-lock.json`、`frontend/src-tauri/tauri.conf.json`、`frontend/src-tauri/Cargo.toml`。

## CI 工作流

文件：`.github/workflows/ci.yml`

职责：

1. `validate`
   - checkout
   - Go / Node 环境准备
   - `npm ci`
   - `go run scripts/dev.go preflight`
   - `go test ./...`
   - `npm run check`
   - `npm run build`
2. `desktop-smoke`
   - Windows / Ubuntu matrix
   - 安装 Linux Tauri 构建依赖（仅 Linux）
   - `npm ci`
   - `go run scripts/dev.go doctor`
   - `go run scripts/dev.go build`
   - 上传 `bin/` 与 `frontend/build/` artifact

目的：

- 在 PR 和主分支推进时尽早发现前端、后端和桌面壳断裂。
- 让构建产物同步到 GitHub Actions artifact，而不是要求把产物提交进 Git。

## Release 工作流

文件：`.github/workflows/release.yml`

触发方式：

- push `v*` tag
- 手动 `workflow_dispatch`

职责：

1. `prepare`
   - 解析 `release_tag`
   - 决定是否发布 release
2. `validate`
   - 运行 `go run scripts/dev.go preflight`
   - 运行 `go test ./...`
   - 运行 `npm run check`
3. `build`
   - Windows / Ubuntu matrix
   - `doctor`
   - `go run scripts/dev.go release`
   - 上传 `bin/release/<os>/` 产物
4. `publish`
   - 聚合 matrix artifacts
   - 合并 checksum 文件
   - 发布 GitHub Release
   - 自动生成 release notes

## 产物位置

- 本地 smoke build：`bin/`
- 本地前端产物：`frontend/build/`
- 本地 release 打包：`bin/release/<os>/`
- GitHub Actions CI artifact：
  - `arkkb-windows-ci-smoke`
  - `arkkb-linux-ci-smoke`
- GitHub Actions Release artifact：
  - `arkkb-windows-release`
  - `arkkb-linux-release`

## 发布顺序

1. 显式更新并提交目标版本文件。
2. 推送 `develop`。
3. 等待升级后的 Windows / Ubuntu CI 全绿。
4. 推送 `vX.Y.Z` tag 触发正式 release。

## Git 状态约束

- `git status` 里应该关注的是源码、workflow、文档变化。
- 构建产物不同步到版本库，而是通过 artifact / release 分发。
- 如果看到临时二进制或打包目录，应先确认是否已在 `.gitignore` 中。当前已覆盖：
  - `bin/`
  - `dist/`
  - `frontend/src-tauri/target/`
  - `arkkb-sidecar`
