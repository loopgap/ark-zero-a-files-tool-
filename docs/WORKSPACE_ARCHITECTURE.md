# Workspace Architecture

## 顶层目录

```text
.
├── .github/workflows/              # CI 与 release 工作流
├── docs/                           # 用户帮助、开发文档、wiki 索引、架构说明
├── frontend/                       # React + Vite 前端与 Tauri desktop shell
│   ├── src/                        # React 源码
│   ├── src-tauri/                  # Tauri Rust 壳与打包配置
│   ├── build/                      # 前端生产产物（忽略）
│   └── package.json                # 前端脚本
├── scripts/                        # 本地 doctor / preflight / build / release 入口
├── src/                            # Go sidecar 与核心域代码
│   ├── bridge/                     # RPC 编排、工作区动作、路径边界、归档浏览
│   ├── core/
│   │   ├── backup/                 # 归档与导出相关逻辑
│   │   ├── config/                 # 配置模型、克隆、读写
│   │   ├── file/                   # 文件读写、原子保存、外部打开
│   │   ├── lsp/                    # LSP 管理入口
│   │   ├── render/                 # Markdown / 预览渲染
│   │   ├── storage/                # bbolt + Bluge
│   │   └── sync/                   # 扫描、索引、同步调度
│   ├── sidecar/                    # HTTP server 与 RPC 注册
│   └── utils/                      # path / charset / lock 等基础工具
├── bin/                            # 本地桌面 smoke 与 sidecar 构建产物（忽略）
├── dist/                           # workflow 中间产物目录（忽略）
├── VERSION                         # 发布版本源
├── README.md                       # 项目入口说明
└── route.md                        # 运行时架构基线
```

## 前端重点文件

- `frontend/src/main.tsx`
  - 全局样式入口与应用挂载
- `frontend/src/styles.tokens.css`
  - 视觉 token
- `frontend/src/app.css`
  - 工作台主样式与 archive explorer 布局样式
- `frontend/src/lib/types.ts`
  - UI 与 RPC 共享类型，包括 `ArchiveBrowseRequest/Response`
- `frontend/src/lib/workbench.ts`
  - 打开标签、扩展名分类、工作台辅助逻辑
- `frontend/src/features/workbench/WorkbenchApp.tsx`
  - 主状态编排
- `frontend/src/features/workbench/components/WorkbenchBrowserContent.tsx`
  - 左侧浏览区紧凑态
- `frontend/src/features/workbench/components/ArchiveExplorerView.tsx`
  - 归档展开浏览台
- `frontend/src/features/workbench/archiveExplorer.ts`
  - 展开态状态工厂、source key 与请求构造

## 后端重点文件

- `src/bridge/workbench.go`
  - 搜索、帮助、归档浏览与工作台数据装配
- `src/bridge/path_resolution.go`
  - canonical path 解析与工作区边界保护
- `src/bridge/bridge.go`
  - 同步调度、应用级桥接动作
- `src/sidecar/server.go`
  - `/file`、`/render`、`/help` 本地 HTTP 服务
- `src/sidecar/rpc.go`
  - RPC 路由表
- `src/core/storage/index.go`
  - Bluge 查询与 reader 生命周期
- `src/core/sync/workspace_sync.go`
  - 索引文档写入与文件元数据同步

## 产物边界

源码与产物必须明确分开：

- 需要跟踪：源码、测试、workflow、文档、配置
- 不需要跟踪：
  - `frontend/build/`
  - `bin/`
  - `dist/`
  - `frontend/src-tauri/target/`
  - 临时二进制 `arkkb-sidecar`

这也是 `git status` 判断是否“真实脏”的基础。
