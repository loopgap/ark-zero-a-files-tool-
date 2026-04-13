# ArkKB Developer Notes

## 核心结构

- `src/core/storage`
  - `bbolt` 保存冷配置、最近项、虚拟归档和文件元数据
  - `Bluge` 保存路径、文件名、正文和归档归属索引
- `src/core/sync`
  - 基于窗口焦点触发工作区增量同步
  - 黑名单目录直接跳过
  - 超过阈值的大文件只建立基础索引
- `src/bridge`
  - 统一暴露工作台接口：工作区、归档、新建、搜索、帮助、读写
- `frontend/src/features/workbench/WorkbenchApp.tsx`
  - 单一工作台状态承载 Workspace / Archives / Recent / Help / Search / Tabs
- `frontend/src-tauri`
  - Tauri v2 桌面壳、目录选择器和 Go sidecar 启动桥接
- `scripts/dev.go`
  - 本地 `doctor/preflight/build/release` 与远端 CI/release 共用的构建入口

## 归档模型

虚拟归档不是物理目录。

- `VirtualFolder`
  - `workspaceId`
  - `preferredRootId`
  - `preferredParentPath`
- 文件与归档的关系单独保存在 `virtual_memberships`

这样做的目的：

- 不污染真实目录结构
- 保留跨目录归档能力
- 给默认新建目标提供方向性

## 默认新建目标

`ResolveDefaultCreateTarget` / `CreateVirtualFile` 的优先级：

1. 显式传入的父目录
2. 当前虚拟归档记住的最近父目录
3. 当前虚拟归档记住的根目录
4. 当前活动根目录
5. 默认根目录
6. 其他工作区根目录

在某个根目录内，如果策略里存在允许目录，则优先尝试这些允许目录。

## 搜索排序

搜索命中后会继续按以下信号加权：

- 最近打开项
- 当前活动根目录
- 当前归档
- 目录白名单
- 文件类型白名单
- 命中类型：文件名 > 路径 > 正文

## 当前边界

- Windows 与 Ubuntu 22.04 都走 Tauri 原生构建链。
- 远端发布通过 GitHub Actions matrix 同时验证 `windows-latest` 与 `ubuntu-22.04`。
- LSP 目前仅保留配置入口，编辑器未接入完整诊断/补全流水线。

## 发布约束

- 发布版本采用 `vX.Y.Z` tag 驱动（例如 `v0.1.0`）。
- `scripts/dev.go preflight` 会同步并校验版本来源（`VERSION` + 三处版本文件）并检查 Tauri 图标资源，失败立即中止。
- `.github/workflows/release.yml` 在 matrix build 前执行 preflight，优先拦截版本不一致与资源缺失问题。
