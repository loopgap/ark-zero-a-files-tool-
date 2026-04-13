# ArkKB 本地 CI 检查指南

## 📋 概述

本项目使用本地 CI 流程来确保代码质量和提交安全性。所有问题会在 `git commit` 前被发现和解决，避免远端构建失败。

## 🚀 快速开始

### 1. 安装 Git Hooks
```bash
# 在项目根目录运行
bash scripts/setup-hooks.sh
```

### 2. 手动运行检查
```bash
# 综合检查（推荐）
bash scripts/check-all.sh

# 仅 Go 检查
bash scripts/check-go.sh

# 仅前端检查
bash scripts/check-frontend.sh
```

### 3. 提交代码
```bash
git add .
git commit -m "fix(bridge): 修复内存泄漏问题"
```

提交时会自动运行检查，失败则提交被阻止。

## 📝 检查项目

### Go 检查
- ✅ 代码格式化 (gofmt)
- ✅ 静态分析 (go vet)
- ✅ 编译测试 (go build)
- ✅ 单元测试 (go test)
- ✅ 依赖完整性 (go mod verify)

### 前端检查
- ✅ TypeScript 类型检查 (tsc)
- ✅ 构建测试 (npm run build)
- ✅ 依赖安全扫描 (npm audit)

### Tauri 检查
- ⚠️ Rust 编译检查 (cargo check)

## 🔧 提交消息格式

```
<type>(<scope>): <description>

类型:
  feat     - 新功能
  fix      - 修复 bug
  docs     - 文档更新
  style    - 代码格式
  refactor - 重构
  test     - 测试相关
  chore    - 构建/工具变更
  perf     - 性能优化
  ci       - CI 配置变更
  build    - 构建系统变更
```

示例:
```
fix(editor): 修复内存泄漏问题
feat(bridge): 添加工作区同步功能
docs: 更新 API 文档
```

## ⚠️ 常见问题

### Q: 检查失败怎么办？
A: 根据错误信息修复问题后重新运行检查脚本。

### Q: 如何跳过检查临时提交？
A: 不建议这样做。使用 `git commit --no-verify` 会绕过检查。

### Q: bash 脚本无法运行？
A: Windows 用户需要安装 Git Bash 或 WSL。

## 📊 检查统计

每次运行会显示：
- 检查时间
- 各模块检查状态
- 失败检查列表

## 🔗 相关文件

- `scripts/check-all.sh` - 综合检查脚本
- `scripts/check-go.sh` - Go 检查脚本
- `scripts/check-frontend.sh` - 前端检查脚本
- `scripts/setup-hooks.sh` - Git hooks 安装脚本
