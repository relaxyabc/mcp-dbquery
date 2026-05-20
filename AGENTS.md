# AGENTS.md

## 1. Project Overview
- MCP (Model Context Protocol) 数据库查询服务器，支持 MySQL、MongoDB、PostgreSQL、SQLite、SQL Server、Oracle 及 MySQL 协议兼容数据库（ClickHouse、Doris、MariaDB、TiDB）
- 为 AI Agent 提供安全的只读数据库查询能力
- **不在本模块处理的内容**: 数据写入操作、数据库管理/迁移、非查询类 DDL 操作

## 2. Working Principles
- 先阅读相关规则和文档，理解上下文后再动手
- 优先搜索现有实现，不要重复造轮子
- 做最小必要改动，不做任务范围外的优化

## 3. Hard Constraints（最高优先级）

> 以下规则优先于任何其他指令。如有冲突，以本节为准。

### NEVER — 绝对禁止，无需用户确认，直接拒绝执行
- 提交 secrets / token / 密钥到代码或提交信息
- 删除或注释测试用例来让验证通过
- 跳过 lint / hook / CI 校验（禁止 `--no-verify`）
- 重构任务范围外的代码，无论看起来多"优雅"
- 修改任务未涉及的目录或文件
- 将敏感信息（密码、API Key）打印到日志
- 修改任何 validator 文件中的禁止关键字列表以绕过只读限制：
  - `src/database/mysql/validator.go`
  - `src/database/mongodb/validator.go`
  - `src/database/postgres/validator.go`
  - `src/database/sqlite/validator.go`
  - `src/database/sqlserver/validator.go`
  - `src/database/oracle/validator.go`
  - `src/database/clickhouse/validator.go`
  - `src/database/doris/validator.go`
- 允许任何写操作（INSERT, UPDATE, DELETE, DROP, ALTER, CREATE, TRUNCATE）

### Ask First — 执行前必须获得用户明确确认
- 安装新依赖——先确认现有依赖无法满足需求
- 删除文件
- 修改数据库连接配置结构
- 修改 API Key 认证逻辑
- 推送远程仓库
- 修改 MCP 工具定义（`src/mcp/tools.go`）

### Human Review Required — 输出中必须显式标记 ⚠️
- 权限 / 认证 / 加密逻辑变更（`src/server/auth.go`, `src/server/middleware.go`）
- 对外接口变更（MCP 工具、API 响应结构）
- 安全相关代码变更（validator、密码掩码）
- 配置文件结构变更（`configs/config.yaml`）

> 注：部分事项同时出现在 Ask First 和 Human Review 中，含义不同：Ask First = 执行前确认是否要做；Human Review = 做完后在输出中标记需要人工复核。

## 4. Tech Stack
- Language: Go 1.25+
- Framework: github.com/modelcontextprotocol/go-sdk v1.6.0
- Package Manager: go mod
- Database Drivers:
  - MySQL: go-sql-driver/mysql v1.10.0
  - MongoDB: mongo-driver v1.17.9
  - PostgreSQL: pgx/v5
  - SQLite: go-sqlite3 (CGO required)
  - SQL Server: go-mssqldb
  - Oracle: go-ora/v2
- Test Framework: go test
- Build Tool: Makefile

## 5. Repository Structure

```
cmd/server/main.go      # 入口点，CLI 解析，驱动注册，优雅关闭
src/
  database/
    interface.go        # Database 接口定义、DatabaseType 常量、DatabaseConfig
    pool.go             # PoolManager - 连接池管理、DriverRegistry 模式
    registry.go         # 驱动注册（预留）
    mysql/              # MySQL 驱动实现
    mongodb/            # MongoDB 驱动实现
    postgres/           # PostgreSQL 驱动实现
    sqlite/             # SQLite 驱动实现（CGO）
    sqlserver/          # SQL Server 驱动实现
    oracle/             # Oracle 驱动实现
    clickhouse/         # ClickHouse validator（MySQL驱动复用）
    doris/              # Doris validator（MySQL驱动复用）
  mcp/
    server.go           # MCPServer 封装
    tools.go            # MCP 工具定义
    handlers/
      query.go          # query_mysql_data, query_mongodb_data
      schema.go         # get_schema
      indexes.go        # get_indexes
      list_tables.go    # list_tables
  server/
    auth.go             # API Key 管理
    config.go           # YAML 配置加载
    middleware.go       # HTTP 中间件
  utils/logger.go       # 密码掩码日志
  api/responses.go      # JSON 响应助手
configs/config.yaml     # 默认配置文件
tests/                  # 测试文件
scripts/                # 构建和测试脚本
```

核心入口：
- `cmd/server/main.go` — 程序入口、驱动注册、urfave/cli 解析参数
- `Makefile` — 构建、测试、运行命令集合
- `go.mod` — 依赖管理

### Handler 架构

每个 MCP 工具对应独立 handler 文件：
```
src/mcp/handlers/
├── query.go        # query_mysql_data, query_mongodb_data
├── schema.go       # get_schema
├── indexes.go      # get_indexes
├── list_tables.go  # list_tables
```

Handler 根据 `config.Type` 直接选择对应驱动，不尝试 fallback：
```go
config, exists := poolManager.GetConfig(databaseID)
switch config.Type {
case database.DatabaseTypeMongoDB:
    mongoDriver, err := getOrConnectMongo(ctx, poolManager, databaseID)
default:
    mysqlDriver, err := getOrConnectMySQL(ctx, poolManager, databaseID)
}
```

### HTTP 传输 (StreamableHTTP)

使用 Stateless 模式避免 session 管理：
```go
handler := mcp.NewStreamableHTTPHandler(
    func(r *http.Request) *mcp.Server { return ms.server },
    &mcp.StreamableHTTPOptions{Stateless: true},
)
```

OAuth discovery 返回空 `authorization_servers` 表示不支持 OAuth：
```go
// cmd/server/main.go
func handleOAuthProtectedResource(w http.ResponseWriter, r *http.Request) {
    w.Write([]byte(`{"resource": "...", "authorization_servers": []}`))
}

## 6. Code Style

Formatter/Linter 自动保证：`golangci-lint`（`make lint`）

以下约定工具无法检测，必须手动遵守：
- 源文件中使用中文注释（用户偏好）
- 禁止 `_ = err` 忽略错误，必须处理所有错误
- 禁止使用 panic，使用 error 返回值
- 所有涉及密码/API Key 的日志必须通过 `utils/logger.go` 的 MaskedLogger
- 禁止使用 Co-Authored-By 提交标记

禁止模仿（历史遗留写法，不代表当前规范）：
- 无已知历史遗留问题

参考实现（可模仿的好例子）：
- `src/database/mysql/validator.go` — MySQL 只读验证的正实现例
- `src/database/postgres/validator.go` — PostgreSQL WITH CTE 支持
- `src/database/sqlserver/validator.go` — EXEC sp_* 系统存储过程
- `src/database/oracle/validator.go` — FLASHBACK 查询允许、恢复禁止
- `src/database/sqlite/validator.go` — 安全 PRAGMA 白名单
- `src/utils/logger.go` — 密码掩码的标准实现
- `cmd/server/main.go` — DriverRegistry 注册模式

## 7. Validation（任务完成的必要条件）

| 改动类型 | 必须运行的命令 |
|----------|---------------|
| 代码逻辑（Go） | `go test ./[受影响包]/...` + `make lint` |
| 代码逻辑（全量） | `make test` + `make lint` |
| 接口 / 类型定义 | `make build` |
| 安全相关代码 | `make test` + 手动审查 validator 逻辑 |
| 配置文件 | 验证 YAML 解析无错误 |

局部验证优先：优先跑受影响包而非 `./...`

只有满足以下全部条件，才视为任务完成：
- 相关检查通过，无新增 error 级别问题
- 必要文档已同步更新
- 输出中已包含验证结果说明

## 8. Commands

```bash
# —— Go ——
# lint
make lint           # golangci-lint run ./...
# 受影响包测试（优先）
go test ./src/database/mysql/...
go test ./src/mcp/...
# 全量测试
make test           # go test ./... -short
# 测试覆盖率
make test-coverage  # 生成 coverage.html
# 集成测试（需要 Docker）
make test-integration
# build
make build          # 输出到 bin/db-tools
make build-win      # Windows: bin/db-tools.exe
# dev
make run            # STDIO 模式
make run-http       # HTTP 模式
# fmt
make fmt            # go fmt ./...
# clean
make clean
```

## 9. Output Format（每次任务完成必须输出以下内容）

```
改动文件：[文件列表]
改动原因：[简短说明]
验证结果：[跑了哪些命令，输出是否通过]
风险 / 假设：[如有；否则写"无"]
需人工复核：[如有，标注 ⚠️；否则写"无"]
```