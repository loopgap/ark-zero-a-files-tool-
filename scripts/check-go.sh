#!/bin/bash
# ArkKB Go 代码检查脚本

set -e

echo "🔍 开始 Go 代码检查..."

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

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

# 1. 格式检查
echo "📋 检查代码格式..."
if [ -n "$(gofmt -l ./src)" ]; then
    check_fail "Go 文件未格式化。请运行: gofmt -w ./src"
else
    check_pass "代码格式正确"
fi

# 2. 导入检查
echo "📋 检查导入格式..."
if command -v goimports &> /dev/null; then
    if [ -n "$(goimports -l ./src)" ]; then
        check_fail "导入未格式化。请运行: goimports -w ./src"
    else
        check_pass "导入格式正确"
    fi
else
    check_warn "goimports 未安装，跳过导入检查"
fi

# 3. 静态分析
echo "📋 运行 go vet..."
if go vet ./src/...; then
    check_pass "go vet 检查通过"
else
    check_fail "go vet 发现问题"
fi

# 4. 编译测试
echo "📋 测试编译..."
if go build -o /tmp/arkkb-sidecar-test ./src/sidecar; then
    check_pass "Sidecar 编译成功"
    rm -f /tmp/arkkb-sidecar-test
else
    check_fail "Sidecar 编译失败"
fi

echo "📋 测试主程序编译..."
if go build -o /tmp/arkkb-test ./main.go; then
    check_pass "主程序编译成功"
    rm -f /tmp/arkkb-test
else
    check_fail "主程序编译失败"
fi

# 5. 单元测试（short 模式）
echo "📋 运行单元测试..."
if go test -short -v ./src/...; then
    check_pass "单元测试通过"
else
    check_fail "单元测试失败"
fi

# 6. 并发安全检测（可选，失败不阻塞）
echo "📋 运行并发安全检测..."
if command -v go &> /dev/null && go test -race -short ./src/bridge/... ./src/core/... 2>&1 | tee /tmp/race_output.txt; then
    if grep -q "WARNING: DATA RACE" /tmp/race_output.txt; then
        check_warn "发现潜在并发问题，请检查 /tmp/race_output.txt"
    else
        check_pass "并发安全检测通过"
    fi
else
    check_warn "并发安全检测跳过（可能需要较长时间）"
fi

# 7. 依赖检查
echo "📋 检查依赖完整性..."
if go mod verify; then
    check_pass "依赖完整性检查通过"
else
    check_fail "依赖完整性检查失败"
fi

echo ""
echo -e "${GREEN}🎉 所有 Go 检查通过！${NC}"
