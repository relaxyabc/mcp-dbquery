.PHONY: build build-linux build-win build-all clean test test-integration run help

# 项目信息
VERSION := 1.0.0
BUILD_DATE := $(shell date +%Y-%m-%d)
BINARY := db-tools
MAIN_PATH := ./cmd/server

# Go 命令
GO := go
GOFLAGS := -v
GOBUILD := $(GO) build $(GOFLAGS)
GOTEST := $(GO) test $(GOFLAGS)
GOCLEAN := $(GO) clean
GOGET := $(GO) get
GOMOD := $(GO) mod

# 默认目标
all: build

# 构建
build:
	@echo "构建 $(BINARY)..."
	$(GOBUILD) -ldflags "-X main.Version=$(VERSION) -X main.BuildDate=$(BUILD_DATE)" -o bin/$(BINARY) $(MAIN_PATH)
	@echo "构建完成: bin/$(BINARY)"

# Linux 构建
build-linux:
	@echo "构建 $(BINARY) (Linux)..."
	GOOS=linux GOARCH=amd64 $(GOBUILD) -ldflags "-X main.Version=$(VERSION) -X main.BuildDate=$(BUILD_DATE)" -o bin/$(BINARY)-linux $(MAIN_PATH)
	@echo "构建完成: bin/$(BINARY)-linux"

# Windows 构建
build-win:
	@echo "构建 $(BINARY).exe..."
	GOOS=windows GOARCH=amd64 $(GOBUILD) -ldflags "-X main.Version=$(VERSION) -X main.BuildDate=$(BUILD_DATE)" -o bin/$(BINARY).exe $(MAIN_PATH)
	@echo "构建完成: bin/$(BINARY).exe"

# 构建所有平台
build-all: build build-linux build-win
	@echo "所有平台构建完成"

# 清理
clean:
	@echo "清理..."
	$(GOCLEAN)
	rm -rf bin/
	rm -f test_output.txt integration_output.txt
	@echo "清理完成"

# 依赖管理
deps:
	@echo "下载依赖..."
	$(GOMOD) download
	$(GOMOD) tidy
	@echo "依赖已下载"

# 单元测试
test:
	@echo "运行单元测试..."
	$(GOTEST) ./... -short
	@echo "测试完成"

# 测试覆盖率
test-coverage:
	@echo "运行测试并生成覆盖率报告..."
	$(GOTEST) ./... -coverprofile=coverage.out
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "覆盖率报告: coverage.html"

# 集成测试
test-integration:
	@echo "运行集成测试..."
	bash scripts/docker-test.sh all

# Docker测试环境
docker-start:
	@echo "启动Docker测试环境..."
	bash scripts/docker-test.sh start

docker-stop:
	@echo "停止Docker测试环境..."
	bash scripts/docker-test.sh stop

docker-cleanup:
	@echo "清理Docker测试环境..."
	bash scripts/docker-test.sh cleanup

# 运行服务器
run: build
	@echo "启动服务器 (STDIO模式)..."
	./bin/$(BINARY)

# 运行HTTP服务器
run-http: build
	@echo "启动服务器 (HTTP模式)..."
	./bin/$(BINARY) -transport http

# 测试STDIO模式
test-stdio: build
	@echo "测试STDIO模式..."
	@echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}' | timeout 5 ./bin/$(BINARY) || echo "STDIO测试完成"

# 安全审查
security-audit:
	@echo "运行安全审查..."
	bash scripts/test.sh

# 安装到系统
install: build
	@echo "安装到系统..."
	cp bin/$(BINARY) /usr/local/bin/
	@echo "安装完成"

# 格式化代码
fmt:
	@echo "格式化代码..."
	$(GO) fmt ./...
	@echo "格式化完成"

# 代码检查
lint:
	@echo "代码检查..."
	@which golangci-lint > /dev/null || (echo "安装 golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run ./...
	@echo "检查完成"

# 帮助
help:
	@echo "MCP Database Query Tool - Makefile"
	@echo ""
	@echo "用法: make [目标]"
	@echo ""
	@echo "目标:"
	@echo "  build           构建二进制文件 (当前系统)"
	@echo "  build-linux     构建Linux二进制文件 (amd64)"
	@echo "  build-win       构建Windows二进制文件 (amd64)"
	@echo "  build-all       构建所有平台二进制文件"
	@echo "  clean           清理构建文件"
	@echo "  deps            下载并整理依赖"
	@echo "  test            运行单元测试"
	@echo "  test-coverage   运行测试并生成覆盖率报告"
	@echo "  test-integration 运行集成测试（需要Docker）"
	@echo "  test-stdio      测试STDIO模式"
	@echo "  docker-start    启动Docker测试环境"
	@echo "  docker-stop     停止Docker测试环境"
	@echo "  docker-cleanup  清理Docker测试环境"
	@echo "  run             构建并运行服务器 (STDIO模式)"
	@echo "  run-http        构建并运行服务器 (HTTP模式)"
	@echo "  security-audit  运行安全审查"
	@echo "  install         安装到系统"
	@echo "  fmt             格式化代码"
	@echo "  lint            代码检查"
	@echo "  help            显示帮助信息"
	@echo ""
	@echo "示例:"
	@echo "  make build        # 构建当前系统"
	@echo "  make build-linux  # 构建Linux版本"
	@echo "  make build-all    # 构建所有平台"
	@echo "  make test         # 运行测试"
	@echo "  make run          # 运行服务器 (STDIO模式)"
	@echo "  make run-http     # 运行服务器 (HTTP模式)"
	@echo "  make test-stdio   # 测试STDIO传输"