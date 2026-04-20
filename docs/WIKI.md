# ArkKB Wiki

这份文档是当前仓库文档入口索引，用来把产品帮助、开发说明、CI/release、工作区架构和运行时基线串起来。

## 阅读顺序

1. `README.md`
   - 项目概览、核心能力、产物位置、文档入口
2. `docs/HELP.md`
   - 面向使用者的工作区、归档、搜索、预览说明
3. `docs/DEVELOPER.md`
   - 面向开发者的运行时结构、归档浏览模型、Git 约束与文档规范
4. `docs/WORKSPACE_ARCHITECTURE.md`
   - 整个工作区的目录结构、关键文件、源码与产物边界
5. `docs/CI_RELEASE.md`
   - 本地构建、CI、release 与 GitHub artifact / release 行为
6. `route.md`
   - 当前真实运行时架构基线

## 关键源文件

- 前端主工作台：`frontend/src/features/workbench/WorkbenchApp.tsx`
- 归档展开浏览器：`frontend/src/features/workbench/components/ArchiveExplorerView.tsx`
- 归档浏览状态：`frontend/src/features/workbench/archiveExplorer.ts`
- 归档浏览 RPC：`src/bridge/workbench.go`
- sidecar RPC 注册：`src/sidecar/rpc.go`
- 搜索索引实现：`src/core/storage/index.go`
- workflow：`.github/workflows/ci.yml` 与 `.github/workflows/release.yml`

## 同步原则

- 功能改动要同步更新：
  - 测试
  - workflow
  - README / docs
  - 架构基线
- 构建产物不提交到版本库；产物同步目标是 workflow artifact 与 GitHub Release，不是 Git 跟踪目录。
