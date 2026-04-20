#!/bin/bash
# ArkKB 综合检查脚本 - 在 git commit 前运行所有检查

set -e

echo "🚀 ArkKB 综合代码检查"
echo "=============================="

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 检查函数
check_pass() {
    echo -e "${GREEN}✅ $1${NC}"
}

check_fail() {
    echo -e "${RED}❌ $1${NC}"
    exit 1
}

check_warn() {
    echo -e "${YELLOW}⚠️ $1${NC}"
}

check_section() {
    echo ""
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}========================================${NC}"
}

# 获取项目根目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
cd "$PROJECT_ROOT"

# 统计开始时间
START_TIME=$(date +%s)

# 记录失败的检查
FAILED_CHECKS=()

# ========================================
# 第一阶段：Go 代码检查
# ========================================
check_section "第一阶段：Go 代码检查"

# 1. 格式检查
echo "📋 检查代码格式..."
if [ -n "$(gofmt -l ./src)" ]; then
    check_fail "Go 文件未格式化。请运行: gofmt -w ./src"
else
    check_pass "代码格式正确"
fi

# 2. 静态分析
echo "📋 运行 go vet..."
if go vet ./src/...; then
    check_pass "go vet 检查通过"
else
    FAILED_CHECKS+=("go vet")
    check_fail "go vet 发现问题"
fi

# 3. 编译测试
echo "📋 测试编译..."
if go build -o /tmp/arkkb-sidecar-test ./src/sidecar 2>/dev/null; then
    check_pass "Sidecar 编译成功"
    rm -f /tmp/arkkb-sidecar-test
else
    FAILED_CHECKS+=("sidecar build")
    check_fail "Sidecar 编译失败"
fi

# 4. 单元测试
echo "📋 运行单元测试..."
if go test -short ./src/...; then
    check_pass "单元测试通过"
else
    FAILED_CHECKS+=("unit tests")
    check_fail "单元测试失败"
fi

# 5. 依赖检查
echo "📋 检查依赖完整性..."
if go mod verify; then
    check_pass "依赖完整性检查通过"
else
    FAILED_CHECKS+=("dependency verify")
    check_fail "依赖完整性检查失败"
fi

# ========================================
# 第二阶段：前端代码检查
# ========================================
check_section "第二阶段：前端代码检查"

cd frontend

# 1. 依赖安装
echo "📋 检查依赖..."
if [ ! -d "node_modules" ]; then
    echo "📦 安装依赖..."
    npm install > /dev/null 2>&1
fi
check_pass "依赖已检查"

# 2. TypeScript 类型检查
echo "📋 运行 TypeScript 类型检查..."
if npx tsc --noEmit 2>/dev/null; then
    check_pass "TypeScript 类型检查通过"
else
    FAILED_CHECKS+=("typescript")
    check_fail "TypeScript 类型检查失败"
fi

# 3. 构建测试
echo "📋 测试前端构建..."
if npm run build 2>/dev/null; then
    check_pass "前端构建成功"
else
    FAILED_CHECKS+=("frontend build")
    check_fail "前端构建失败"
fi

cd ..

# ========================================
# 第三阶段：Rust/Tauri 检查（可选）
# ========================================
check_section "第三阶段：Tauri 检查"

cd frontend/src-tauri

# 1. Cargo 检查
echo "📋 检查 Rust 编译..."
if cargo check 2>/dev/null; then
    check_pass "Rust 检查通过"
else
    check_warn "Rust 检查失败（可能未安装 Rust）"
fi

cd ../..

# ========================================
# 结果统计
# ========================================
check_section "检查结果"

END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))

if [ ${#FAILED_CHECKS[@]} -eq 0 ]; then
    echo -e "${GREEN}🎉 所有检查通过！${NC}"
    echo ""
    echo "📊 检查统计:"
    echo "  - 检查时间: ${DURATION} 秒"
    echo "  - Go 检查: ✅"
    echo "  - 前端检查: ✅"
    echo "  - 失败检查: 0"
    echo ""
    echo "✨ 可以安全提交代码！"
    exit 0
else
    echo -e "${RED}❌ 发现 ${#FAILED_CHECKS[@]} 个问题${NC}"
    echo ""
    echo "📊 检查统计:"
    echo "  - 检查时间: ${DURATION} 秒"
    echo "  - 失败检查:"
    for check in "${FAILED_CHECKS[@]}"; do
        echo "    - $check"
    done
    echo ""
    echo "🔧 请修复上述问题后重新运行此脚本"
    exit 1
fi
