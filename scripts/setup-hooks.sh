#!/bin/bash
# ArkKB Git Hooks 安装脚本

set -e

echo "🔧 安装 Git Hooks..."

# 获取项目根目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
HOOKS_DIR="$PROJECT_ROOT/.git/hooks"

# 确保 hooks 目录存在
mkdir -p "$HOOKS_DIR"

# 复制预提交钩子
echo "📄 创建 pre-commit 钩子..."
cat > "$HOOKS_DIR/pre-commit" << 'EOF'
#!/bin/bash
# ArkKB 预提交钩子 - 自动运行代码检查

set -e

echo "🔍 运行预提交检查..."

# 获取项目根目录
GIT_DIR="$(git rev-parse --git-dir)"
PROJECT_ROOT="$(cd "$GIT_DIR/.." && pwd)"

# 切换到项目根目录
cd "$PROJECT_ROOT"

# 运行综合检查
if [ -f "scripts/check-all.sh" ]; then
    bash scripts/check-all.sh
else
    echo "⚠️ 未找到检查脚本，跳过自动检查"
fi
EOF

# 复制提交消息钩子
echo "📄 创建 commit-msg 钩子..."
cat > "$HOOKS_DIR/commit-msg" << 'EOF'
#!/bin/bash
# ArkKB 提交消息验证钩子

commit_msg=$(cat "$1")

# 提交消息格式: <type>(<scope>): <description>
# Types: feat, fix, docs, style, refactor, test, chore, perf, ci, build
pattern="^(feat|fix|docs|style|refactor|test|chore|perf|ci|build)(\(.+\))?: .{1,50}"

if ! echo "$commit_msg" | grep -qE "$pattern"; then
    echo "❌ 提交消息格式不正确"
    echo ""
    echo "正确格式: <type>(<scope>): <description>"
    echo ""
    echo "类型说明:"
    echo "  feat     - 新功能"
    echo "  fix      - 修复 bug"
    echo "  docs     - 文档更新"
    echo "  style    - 代码格式（不影响功能）"
    echo "  refactor - 重构（不影响功能）"
    echo "  test     - 测试相关"
    echo "  chore    - 构建/工具变更"
    echo "  perf     - 性能优化"
    echo "  ci       - CI 配置变更"
    echo "  build    - 构建系统变更"
    echo ""
    echo "示例:"
    echo "  feat(editor): 添加文件加载缓存"
    echo "  fix(bridge): 修复内存泄漏问题"
    echo "  docs: 更新 README"
    exit 1
fi

echo "✅ 提交消息格式正确"
EOF

# 设置执行权限
chmod +x "$HOOKS_DIR/pre-commit"
chmod +x "$HOOKS_DIR/commit-msg"

echo ""
echo "✅ Git Hooks 安装完成！"
echo ""
echo "已安装的钩子:"
echo "  - pre-commit: 自动运行代码检查"
echo "  - commit-msg: 验证提交消息格式"
echo ""
echo "下次提交时将自动运行检查。"
