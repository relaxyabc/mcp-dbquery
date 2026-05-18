#!/bin/bash

# MCP Database Query Tool - Test Runner
# 运行所有测试（单元测试 + 集成测试）

set -e

echo "=========================================="
echo "MCP Database Query Tool - Test Runner"
echo "=========================================="

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 测试结果统计
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# 运行单元测试
run_unit_tests() {
    echo ""
    echo "运行单元测试..."
    echo "----------------------------------------"

    if go test ./... -v -short 2>&1 | tee test_output.txt; then
        echo "${GREEN}✓ 单元测试通过${NC}"
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        echo "${RED}✗ 单元测试失败${NC}"
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi

    TOTAL_TESTS=$((TOTAL_TESTS + 1))
}

# 运行集成测试（需要Docker）
run_integration_tests() {
    echo ""
    echo "运行集成测试..."
    echo "----------------------------------------"

    # 检查Docker是否可用
    if ! command -v docker &> /dev/null; then
        echo "${YELLOW}⚠ Docker未安装，跳过集成测试${NC}"
        return
    fi

    # 检查testcontainers是否可用
    if ! go list -m github.com/testcontainers/testcontainers-go &> /dev/null; then
        echo "${YELLOW}⚠ testcontainers未安装，跳过容器测试${NC}"
        return
    fi

    # 运行集成测试（跳过需要真实容器的测试）
    if go test ./tests/integration/... -v -short 2>&1 | tee integration_output.txt; then
        echo "${GREEN}✓ 集成测试通过${NC}"
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        echo "${RED}✗ 集成测试失败${NC}"
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi

    TOTAL_TESTS=$((TOTAL_TESTS + 1))
}

# 运行宪章合规测试
run_constitution_tests() {
    echo ""
    echo "运行宪章合规测试..."
    echo "----------------------------------------"

    # 测试只读强制执行
    echo "检查只读验证..."
    if go test ./tests/integration/... -run TestReadOnlyEnforcement -v 2>&1; then
        echo "${GREEN}✓ 只读强制执行测试通过${NC}"
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        echo "${RED}✗ 只读强制执行测试失败${NC}"
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi

    # 测试API密钥认证
    echo "检查API密钥认证..."
    if go test ./tests/integration/... -run TestAPIKeyValidation -v 2>&1; then
        echo "${GREEN}✓ API密钥认证测试通过${NC}"
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        echo "${RED}✗ API密钥认证测试失败${NC}"
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi

    TOTAL_TESTS=$((TOTAL_TESTS + 2))
}

# 运行构建验证
run_build_verification() {
    echo ""
    echo "运行构建验证..."
    echo "----------------------------------------"

    # 清理并构建
    go clean

    if go build -o bin/db-tools ./cmd/server; then
        echo "${GREEN}✓ 构建成功${NC}"

        # 检查二进制文件大小
        SIZE=$(ls -lh bin/db-tools | awk '{print $5}')
        echo "  二进制文件大小: $SIZE"

        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        echo "${RED}✗ 构建失败${NC}"
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi

    TOTAL_TESTS=$((TOTAL_TESTS + 1))
}

# 运行安全审查
run_security_audit() {
    echo ""
    echo "运行安全审查..."
    echo "----------------------------------------"

    # 检查密码遮蔽
    echo "检查密码遮蔽实现..."
    if grep -r "REDACTED" src/utils/logger.go &> /dev/null; then
        echo "${GREEN}✓ 密码遮蔽实现存在${NC}"
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        echo "${RED}✗ 密码遮蔽实现缺失${NC}"
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi

    # 检查只读验证
    echo "检查MySQL只读验证..."
    if grep -r "ForbiddenKeywords" src/database/mysql/validator.go &> /dev/null; then
        echo "${GREEN}✓ MySQL只读验证存在${NC}"
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        echo "${RED}✗ MySQL只读验证缺失${NC}"
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi

    echo "检查MongoDB只读验证..."
    if grep -r "ForbiddenOperations" src/database/mongodb/validator.go &> /dev/null; then
        echo "${GREEN}✓ MongoDB只读验证存在${NC}"
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        echo "${RED}✗ MongoDB只读验证缺失${NC}"
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi

    # 检查API密钥长度验证
    echo "检查API密钥长度验证..."
    if grep -r "32" src/server/auth.go | grep -i "length" &> /dev/null; then
        echo "${GREEN}✓ API密钥长度验证存在${NC}"
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        echo "${RED}✗ API密钥长度验证缺失${NC}"
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi

    TOTAL_TESTS=$((TOTAL_TESTS + 4))
}

# 输出测试报告
print_report() {
    echo ""
    echo "=========================================="
    echo "测试报告"
    echo "=========================================="
    echo "总测试数: $TOTAL_TESTS"
    echo "${GREEN}通过: $PASSED_TESTS${NC}"
    echo "${RED}失败: $FAILED_TESTS${NC}"

    if [ $FAILED_TESTS -eq 0 ]; then
        echo ""
        echo "${GREEN}✓ 所有测试通过！${NC}"
        exit 0
    else
        echo ""
        echo "${RED}✗ 有测试失败，请检查日志${NC}"
        exit 1
    fi
}

# 主函数
main() {
    # 运行所有测试
    run_build_verification
    run_unit_tests
    run_constitution_tests
    run_security_audit
    # run_integration_tests  # 需要Docker时启用

    # 输出报告
    print_report
}

# 执行主函数
main "$@"