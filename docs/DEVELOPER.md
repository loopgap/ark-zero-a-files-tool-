# ArkKB Developer Notes

## 运行时结构

- `frontend/src`
  - React 19 + TypeScript 工作台 UI
  - `features/workbench` 是桌面主界面与归档浏览主域
- `frontend/src-tauri`
  - Tauri v2 桌面壳、目录选择器与 sidecar 启动桥接
- `src/sidecar`
  - 本地 HTTP server 与 RPC 注册
- `src/bridge`
  - 工作区、归档、帮助、搜索、文件读写、路径边界与同步编排
- `src/core/storage`
  - `bbolt` 保存配置、最近项、文件元数据、归档归属
  - `Bluge` 保存路径、文件名、扩展名、正文与归档过滤字段
- `src/core/sync`
  - 工作区扫描、索引写入、focus sync 与同步调度
- `scripts/dev.go`
  - 本地 `doctor / preflight / ci / build / release` 和远端 workflow 共用入口

## 当前归档浏览模型

- `archive.browse` 是新的归档浏览主 RPC。
- source 只分两类：
  - `virtual_folder`
  - `auto_category`
- 浏览状态由前端 `ArchiveExplorerState` 维护，核心维度包括：
  - 左侧标签
  - 当前目录
  - 查询词
  - 搜索模式
  - 排序
  - 分组视图
  - 当前选中文件
  - 分页 cursor
  - 展开态/紧凑态
- 展开态主界面采用三段布局：
  - 左列：`分类 / 归档 / 目录`
  - 中列：文件列表、排序、分组、分页
  - 右列：快速预览与元信息

## 工作区与路径边界

- 所有文件读写、重命名、删除、外部打开、HTTP `/file` `/render` 都应走 canonical path 校验。
- `ResolvedPath` 是内部统一边界类型，禁止再回到词法前缀判断。
- embedded help docs 是打包产物的真源，磁盘 docs 仅作本地开发回退。

## 索引与搜索约束

- Bluge 只负责索引驱动检索，禁止请求路径上对命中文件逐个回盘读取。
- 查询结束后必须关闭 reader，避免 snapshot / file handle 泄漏。
- 大文件超过策略阈值时只建立基础索引，不做全文分词。
- `mtime` 由文件元数据读出，`lastOpenedAt` 由最近项映射，不写入全文索引。

## 构建与发布链路

- `preflight` 只校验版本、icon 和打包输入，不会自动改版本文件。
- 发布前需要先显式提交 `VERSION`、`frontend/package.json`、`frontend/package-lock.json`、`frontend/src-tauri/tauri.conf.json`、`frontend/src-tauri/Cargo.toml`。
- 本地严格发布前检查：`go run scripts/dev.go ci`
- 本地 smoke build：`go run scripts/dev.go build`
  - 输出：`frontend/build/` 与 `bin/`
- 发布打包：`go run scripts/dev.go release`
  - 输出：`bin/release/<os>/`
- `ci` workflow：
  - preflight
  - `go test ./...`
  - `npm run check`
  - `npm run build`
  - Windows / Linux 桌面 smoke build 与 artifact 上传
- `release` workflow：
  - preflight
  - `go test ./...`
  - `npm run check`
  - Windows / Ubuntu matrix 打包
  - checksum 汇总
  - GitHub Release 发布
- 正式发布顺序：
  - 提交目标版本代码与版本文件
  - 推送 `develop`
  - 等待双平台 CI 通过
  - 推送 `vX.Y.Z` tag

## Git 约束

- 构建产物目录不进入版本库：`bin/`、`dist/`、`frontend/build/`、`frontend/src-tauri/target/`。
- 源码、workflow、文档和测试需要一起更新，避免“功能已变，文档未同步”。
- 如果 `git status` 仍然脏，优先区分：
  - 真实源码改动
  - 本地产物
  - 与当前任务无关的用户文件
