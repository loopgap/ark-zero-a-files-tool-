# ArkKB

ArkKB 是一个遵循“绝对轻量”、“极速克制”和“原生无阻”哲学的本地项目。

## 架构规范 (The Master Constitution)

作为项目的核心基石，我们在所有的开发、重构和迭代中，必须严格贯彻极其纯净与高效的架构标准。

👉 请务必在开始任何开发前阅读最高指导原则：**[📜 ArkKB 项目架构规范与 AI 执行蓝图](route.md)**

## 已实现核心模块

### 1. 进程单例 PID 锁 (`src/utils/lock/`)
- **功能**：实现在 `~/.arkkb/.lock` 的进程锁。
- **孤儿锁自愈**：通过 `isProcessAlive` 探测 PID 是否真实存活。
- **锁管理**：封装 `Lock()` 和 `Unlock()`，支持 Windows 和 Unix。

### 2. 存储层封装 (`src/core/storage/`)
- **`kv.go` (bbolt)**：
    - **环境配置**：存储 `< 4KB` 的环境配置，强制检查 4KB 长度限制。
    - **文件元数据 (FileMeta)**：使用 `json` 存储文件的 `Charset`, `Size`, `Modified` 等元数据，确保字符集信息可持久化。
    - **去异化 Key**：所有路径在存储前均经过 `NormalizePath` 处理，确保跨平台路径一致性。
- **`index.go` (bluge)**：作为系统唯一的“Source of Truth”，实现基于 `bluge.NewBatch()` 的原子更新逻辑。
- **`manager.go`**：统一管理生命周期，路径统一指向 `~/.arkkb/`。

### 3. 文件处理与字符集工具 (`src/core/file/` & `src/utils/charset/` & `src/utils/pathutil/`)
- **`sniffer.go`**：集成 `chardet` 实现轻量级编码嗅探（UTF-8 vs GBK），仅在索引构建期调用以降低开销。
- **`normalize.go`**：实现路径去异化处理，统一 Windows/Unix 路径分隔符，确保索引 Key 的唯一性。
- **`atomic.go`**：实现 `SafeSave` 三段式原子落地（`.tmp` 写入 -> `Sync` 落盘 -> `Rename` 替换），并集成路径去异化。
- **`rescue.go`**：增强救援逻辑，支持自动从残留的 `.tmp` 文件中恢复数据或清理冗余，确保崩溃自愈。
- **`tombstone.go`**：实现 `SoftDelete` 软删除机制，将物理文件移入 `~/.arkkb/trash/` 节点，并为后续索引标记 `__DELETED__` 提供状态支撑。

### 4. 前端核心逻辑 (`frontend/`)
- **Svelte 5 (Runes) SPA 外壳**：使用 Svelte 5 构建极简、响应式的 UI 布局。
- **三层路由分发协议**：
    - **深度集成层**：文本文件（`.md`, `.rs`, `.py` 等）由 CodeMirror 6 接管，支持高亮与大文件加载。
    - **轻量预览层**：PDF 和 HTML 文件通过原生 `<iframe>` 零开销渲染。
    - **外部穿透层**：二进制文件提示并引导用户使用系统原生软件打开。
- **LSP 全链路对接**：
    - **实时诊断**：集成 LSP `textDocument/publishDiagnostics`，在编辑器中以波浪线实时标注错误与警告。
    - **智能补全**：对接 LSP `textDocument/completion`，提供丝滑的代码补全体验。
    - **全局警报系统**：在 UI 顶层实现极简警报栏，捕捉并展示 LSP 关键错误。

### 5. Wails 桥接层与通信 (`src/bridge/`)
- **`bridge.go`**：作为核心枢纽，封装 `LSPManager` 与 `StorageManager`，为前端提供统一的 RPC 接口。
- **事件转发机制**：实现后端 LSP 通知到前端 Svelte 事件的零延迟转发。
- **资产直取协议**：优化 `ReadFile` 与 `SaveFile` 接口，配合 `wails://file/` 实现大文件的高效存取。

### 6. 自动化运维工具 (`scripts/dev.go`)
- **`doctor` (环境测谎机)**：检查 `CGO_ENABLED` 是否为 0，并检测 Windows (WebView2) 或 Linux (WebKit2GTK) 的运行时依赖。
- **`bench` (海量压测器)**：自动生成 10,000+ 随机文件，压测 Bluge 索引构建速度与 WindowFocus 触发下的 mtime 比对性能。
- **`build` (极限出包)**：整合前端构建与后端编译，强制执行无 CGO 链接并自动调用 `upx` 压缩产物。

遵循此蓝图，我们将确保在此项目中：
- CPU 和磁盘 IO 真正的零空耗。
- CGO、重量级动态跨平台依赖等彻底剔除。
- 对所有大文件的原生支持与真正的极致流转（避免 IPC 内存爆炸）。
- 从设计层面以原子的重定向、无锁化和单写者机制解决并发碰撞和数据撕裂。
