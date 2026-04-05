# 📜 ArkKB 项目架构规范与 AI 执行蓝图 (The Master Constitution)

## 第一章：核心哲学 (The Dogma)
1. **绝对轻量 (Absolute Lightweight)**：禁止引入任何常驻后台的高耗能进程。无主动任务且失去窗口焦点时，CPU 占用与磁盘 I/O 必须严格为 0%。
2. **极速克制 (Extreme Restraint)**：宁可功能降级，也绝不在关键数据路径上做高时间复杂度 $O(N)$ 的运算。杜绝过度防守引发的性能损耗。
3. **原生跨端与激进迭代 (Native Dual-Path & Radical Modernity)**：严格秉承 Windows 10+ 与 Ubuntu 22+ 的纯血双路径实现，不对任何旧版系统（如 Win7/8、老旧 Linux 发行版）包袱负责。一切架构及依赖必须按当时最新操作系统的技术基准（如 WebView2、现代内核 API）进行绝对优化。禁止使用 CGO，严禁引入 SQLite 或 Tree-sitter 等重量级 C/C++ 阻塞性依赖。跨平台差异全权通过底层抽象机制抹平，拒绝补丁式代码。
4. **极致优雅与原生质感 (Supreme Elegance & Native Feel)**：彻底抛弃传统 Web 套壳的廉价感。UI 必须苛求极致的视觉审美与交互顺滑度。从排版的呼吸感、精致的微动画，到丝滑的深色模式阻尼感，必须打造出具有极高执行效率与如同桌面原生级别控制感的顶级视觉体验。

## 第二章：目录树与 Git 规范 (Directory & Git Flow)
项目必须保持极度纯净，远程仓库严禁出现构建产物与临时文件。

### 2.1 物理目录隔离
```text
.
├── src/           # Go 后端业务逻辑（按领域驱动拆分，如 core/, bridge/, utils/）
├── frontend/      # Svelte 4/5 + TS + @sveltejs/adapter-static 纯静态工程（严格 SPA 模式）
├── scripts/       # 自动化脚本目录（仅存放 dev.go）
├── .gitignore     # 严格屏蔽以下动态目录
├── go.mod / wails.json / route.md

被 .gitignore 严格屏蔽的动态目录：
- .tmp/：用于大文件原子重组的临时缓冲区、遇错救援文件（Rescue Files）、测试沙盒。
- bin/ / dist/：Wails 构建输出的最终可执行文件与压缩包。
- ~/.arkkb/（运行期生成）：存放 bbolt.db (轻量配置) 和 bluge_index/ (倒排索引库)。
```

### 2.2 Git 协作与自动化卡点
- **主干极简**：保持 main 分支的绝对纯净。
- **自动化卡点**：所有合入主干的代码，必须在本地通过 `go run scripts/dev.go bench` 的沙盒压测跑通，确保没有引入性能退化。

## 第三章：运行机制与底层调度 (Runtime & Core Mechanisms)
### 3.1 视窗焦点驱动同步 (Focus-Driven Sync)
- **禁止常驻监听**：彻底废弃 fsnotify / inotify 守护进程。
- **惰性触发**：基于 Wails v3 的统一事件系统，仅在监听 `events.Common.WindowFocus` 时，执行目录树 mtime 的快速比对。
- **防抖限流与原子更新 (Bluge Batch)**：单次变动文件 < 100 个，在后台使用 `bluge.NewBatch()` 实施合并原子更新（要么全成功，要么不污染原索引）；> 100 个（如 git pull 后），中断静默并在 UI 弹出提示，将消耗算力的决定权交还给用户。

### 3.2 零触碰管道与跨平台抹平 (Zero-Touch Pipe)
- **换行符零干预**：Go 内存中严禁执行 \r\n 替换。通过前端 CodeMirror 6 (CM6) 与外部 LSP 在握手时强制声明 `PositionEncodingKind: 'utf-8'` 解决偏移对齐。
- **历史乱码极速断崖式嗅探**：采用纯 Go 无 CGO 依赖的轻量嗅探库（如 `github.com/wlynxg/chardet`）。为严格防守极端 OOM，**要求必须采用“仅提取文件头部前 4KB 字节作截断试探”** 机制，且仅在后台索引构建期触发。试探结果长驻 Bluge Metadata，后续前端读取直接信赖缓存结构，从源头上抹杀大型文件字符集扫描带来的高昂 CPU 连环暴击。
- **路径去异化**：数据主键统一为跨平台格式。大小写兼容全权交由 Bluge 引擎的 Lowercase Token Filter 分词器处理，Go 业务侧零字符串拼接转换损耗。
- **权限边界与状态缓存**：外部工具探测仅在应用冷启动执行一次，避免运行期验证 +x 执行权限产生的死锁级阻塞。

## 第四章：渲染与大文件编辑策略 (Rendering & Large Files)
### 4.1 渲染独裁与网络劫持
- **DOM 独裁**：全局只允许存在一个 CM6 实例接管纯文本。Svelte 主要用于框架外壳构建（路由与浮窗），严禁将万行级大型文本塞入 Svelte Store 导致框架无端重绘与 OOM。
- **AssetServer 原生劫持**：拒绝通过 Wails IPC (JSON 包) 传递超量文件内容！前端需通过 Wails `AssetOptions.Handler` (遵循标准 `http.Handler`) 或 `wails://file/` 原生本地流取值，避开进程内反复 Marshal 开销。

### 4.2 虚拟视窗与原子级安全保存 (Virtual Paging & Atomic Write)
- **分块载入与 CM6 断层占位 (Gap)**：前端采用极高的防抖上限，大型文本 Go 后端仅用 `file.ReadAt` 小额切片。CM6 内存不容纳全量，采用视窗内的 “gap” 占位机制实现内存安全级滚动。
- **跨平台三段式原子落地 (Atomic Sync Rename)**：
  1. 必须在**目标文件同级磁盘/分区**内创建 `.tmp`（规避跨挂载盘执行 O(N) 复制大暴走）。
  2. 流式执行三段覆写：复制旧头段 -> 注入新中段 -> 追加旧尾段。
  3. **必须强制 `file.Sync()`** 落盘后，再执行原子的 `os.Rename` 替换元数据，实现“宁可保存失败，绝不导致死机丢档”。
- **bbolt 轻量下放**：仅供 < 4KB 极小值（如环境路径）使用，严防并发事务产生死锁，数据持久化仅需调取轻便 API 即可。

## 第五章：三层文件路由协议 (Three-Tier Dispatch & Routing)
禁止尝试用单一编辑器通吃所有文件，严格遵循以下三级路由：
- **深度集成层 (CM6 编辑)**：匹配纯文本白名单（如 `.c`, `.rs`, `.md`, `.py` 等）。全量启用高亮、片段加载与 LSP 跳转。
- **轻量预览层 (Iframe 预览)**：匹配格式化文档（如 `.pdf`, `.html`）。前端调用原生 `<iframe>` 或 `<embed>` 零开销只读渲染，坚决将解析丢给浏览器原生内核。
- **外部穿透层 (Native 唤醒)**：匹配工程二进制（如 `.pcbdoc`, `.exe`, `.xlsx`）。双击时 Go 包装安全命令阵列交由系统专业软件独立接管。

## 第六章：IPC 与外部通讯 (IPC & External Pipes)
### 6.1 LSP 增量管道协同 (Incremental Sync)
- **管道强生命周期**：LSP 进程（如 clangd 的 `std.Cmd`）与当前 “Tab 页” 严格绑定。Tab 销毁或者重加载瞬间必须 `SIGTERM` 清理干净。
- **增量通信减负**：与 LSP 沟通严格使用 `TextDocumentSyncKind.Incremental` 协议模式 (`textDocument/didChange`)，仅上报用户键入的变更补丁 (Delta JSON)，绝对静止全量互灌（防管道积压）。
- **非必要主动噪音拦截**：强制过滤并丢弃 LSP 擅自推送来的高频 Diagnostics 诊断报警报文，维持界面的极简清爽。

### 6.2 依赖发现与命令防注入
- **防 Shell 注入与安全抛出**：所有 `os/exec` 对系统底层的调用，无论穿透何种路径，均强制采用安全切片参数传值法（不拼接包含路径或指令的任何单行原句）。

## 第七章：数据边界与容灾归档 (Data Boundaries & Archiving)
### 7.1 反冗余与职责割裂
- **Bluge 的绝对统治与克制阈值**：所有路径追踪、全文查询、倒排索引及元数据标签的“Source of Truth”必须唯一处于 Bluge。但为坚守“极速克制”防线，**针对大于 1MB 的超大文件（如日志或 SQL Dump），强制熔断其全本倒排分词链路**，仅建立文件系统基础索引（文件名/挂载点/大小）。宁可舍弃大文件的内容搜索能力，也绝对拒绝 O(N) 及以上的海量算力陷阱。
- **bbolt 功能下沉**：绝不和 Bluge 搞事务对齐（双写灾难），此 KV 小库仅保存应用基本冷环境。
- **单件全压入冷备**：通过 Go `archive/zip` 将小得可怜的 `bbolt.db` 与 `bluge_index/` 双节点结构，一键制裁成全压缩快照传输。

### 7.2 软删除机制 (Tombstone)
- 应用内删除实质上降级为：“物理移入 `~/.arkkb/trash/` 节点” 并 “追加一条属于源路径的 `__DELETED__` 标记进索引”。

## 第八章：自动化流统筹 (The dev.go Orchestrator)
摒弃离散配置，将指令闭环到纯 Go 脚本统筹 (Go 自举原则)，杜绝额外配置包：
- `go run scripts/dev.go doctor`：纯环境测谎机（检查无 CGO 安全性、WebView2 / WebKit2GTK 依赖等）。
- `go run scripts/dev.go bench`：打穿 `bluge` 并建立海量伪文件的压测器。
- `go run scripts/dev.go clean` / `build`：极限出包并搭配 upx 进行剥离指令处理。

## 第九章：极致异常处理与文件保护机制 (Extreme Error Handling & File Protection)
为从设计层面规避竞争条件 (Race Conditions)、冒险现象及同步问题，并实现最轻量化的保险机制，特制定以下保底保护规范：

### 9.1 无锁设计与所有权唯一性 (Lock-Free & Ownership)
- **绝对单写者 (Single Writer Principle)**：禁止多线程/协程同时对单一文件执行写入。系统依靠唯一文件描述符令牌驱动，令牌流转类似于所有权机制，保证同一时刻写入路径上只有单一控制流，从源头彻底斩断数据竞争和冒险。
- **读写分离与瞬态句柄 (Ephemeral Reader)**：读取操作全量降维为纯只读行为（`os.O_RDONLY`），无视系统加锁机制。针对 Windows 系统下正在被读取的文件执行 `os.Rename` 时必定抛出 `Access Denied` 的跨端巨大差异，**系统强制要求所有切片加载的读取行为必须是瞬态关闭的**（Open -> ReadAt -> 立即 Close）。这种用完即抛的文件符管理让沙盒中的 `.tmp` 文件在执行最后的 `os.Rename` 时绝无句柄被占用的可能，从设计哲学上以最为轻巧的手法完美规避跨系统锁行为不对齐的设计灾难。

### 9.2 破坏性操作的“延迟保险索” (Deferred Fallback & Quick Rescue)
- **静默现场留存**：任何涉及核心数据重写的异常（如磁盘写满、设备意外断电模拟等情况），捕捉后的第一动作是“冻结”。保留废弃的 `.tmp` 和原始未被污染的旧文件。绝不盲目进行循环重试引发次生灾害。
- **崩溃安全与强硬阻断 (Fail-Safe)**：若是遭遇 `bbolt` 与 `bluge` 的系统级故障，强行 Panic 或优雅安全退出当前树逻辑，并通过明确的 UI 发送错误快照警报。宁可让用户知道功能异常，也坚决不要输出静默的残次脏数据将原本良好数据给覆盖。

### 9.3 进程实例与主权监控 (Singleton & Ownership Monitored)
- **轻量进程自锁 (PID Lock)**：应用启动即在 `~/.arkkb/` 创建携带当前实例 PID 的进程级共享锁控制文件 `.lock`。如果由于用户暴力点按或跨桌面调起，产生了多实例并发启动并试图同时操作数据库/索引库的情况，新实例仅嗅探到死锁将立刻告警并自行销毁，绝对杜绝库文件撕裂与多线程碰撞。
- **孤儿锁自愈判定**：若上次是非法掉电退出导致 `.lock` 遗留，只需向系统探测该 PID 是否仍然存活（极小号命令调用或系统调取）。若确认前对象死亡，新的实例将主动覆写，避免复杂的跨进程同步死锁带来的灾难性心智负担。
