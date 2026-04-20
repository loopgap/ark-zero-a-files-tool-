#!/bin/bash
# ArkKB 前端代码检查脚本

set -e

echo "🔍 开始前端代码检查..."

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
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

cd frontend

# 1. 依赖安装检查
echo "📋 检查依赖安装..."
if [ ! -d "node_modules" ]; then
    echo "📦 安装依赖..."
    npm install
fi
check_pass "依赖已安装"

# 2. TypeScript 类型检查
echo "📋 运行 TypeScript 类型检查..."
if npx tsc --noEmit; then
    check_pass "TypeScript 类型检查通过"
else
    check_fail "TypeScript 类型检查失败"
fi

# 3. ESLint 检查
echo "📋 运行 ESLint 检查..."
if [ -f ".eslintrc.json" ] || [ -f ".eslintrc.js" ]; then
    if npx eslint src/ --ext .ts,.tsx; then
        check_pass "ESLint 检查通过"
    else
        check_fail "ESLint 检查失败"
    fi
else
    check_warn "ESLint 配置不存在，跳过检查"
fi

# 4. Prettier 格式检查
echo "📋 检查代码格式..."
if [ -f "prettier.config.js" ] || [ -f ".prettierrc" ]; then
    if npx prettier --check "src/**/*.{ts,tsx,css}"; then
        check_pass "代码格式正确"
    else
        check_fail "代码格式不正确。请运行: npm run format"
    fi
else
    check_warn "Prettier 配置不存在，跳过检查"
fi

# 5. 依赖安全扫描
echo "📋 运行依赖安全扫描..."
if npm audit --audit-level=high 2>&1 | tee /tmp/npm_audit.txt; then
    if grep -q "found 0 vulnerabilities" /tmp/npm_audit.txt; then
        check_pass "依赖安全检查通过"
    else
        check_fail "发现高危安全漏洞，请运行 npm audit fix"
    fi
else
    check_warn "依赖安全扫描失败"
fi

# 6. 构建测试
echo "📋 测试构建..."
if npm run build; then
    check_pass "前端构建成功"
else
    check_fail "前端构建失败"
fi

# 7. 单元测试（如果存在）
if [ -f "package.json" ] && grep -q '"test"' package.json; then
    echo "📋 运行单元测试..."
    if npm test -- --passWithNoTests; then
        check_pass "单元测试通过"
    else
        check_fail "单元测试失败"
    fi
else
    check_warn "未配置单元测试，跳过"
fi

cd ..

echo ""
echo -e "${GREEN}🎉 所有前端检查通过！${NC}"
