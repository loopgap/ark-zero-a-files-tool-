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
go run scripts/dev.go build
go run scripts/dev.go release
```

## 文档

- 架构约束：`route.md`
- 用户帮助：`docs/HELP.md`
- 开发说明：`docs/DEVELOPER.md`
- 远端发布：`.github/workflows/release.yml`
