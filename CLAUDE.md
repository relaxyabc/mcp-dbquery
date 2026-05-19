# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

> **重要**: 所有工作原则、约束、验证要求等详见 [AGENTS.md](AGENTS.md)。后续变更请修改 AGENTS.md。

## 快速参考

### 构建 & 测试
```bash
make build          # 构建
make test           # 单元测试
make lint           # 代码检查
make run            # STDIO 模式运行
make run-http       # HTTP 模式运行
```

单包测试（优先）:
```bash
go test ./src/database/mysql/...
go test ./src/database/postgres/...
```

### MCP 工具
| 工具 | 描述 |
|------|------|
| `query_mysql_data` | MySQL SELECT 查询 |
| `query_mongodb_data` | MongoDB find/aggregate |
| `get_schema` | 表/集合结构 |
| `get_indexes` | 索引元数据 |
| `list_tables` | 列出表/集合 |

### 核心原则（宪法级别）
1. **只读操作**: 禁止所有写操作（INSERT/UPDATE/DELETE/DROP/ALTER/CREATE/TRUNCATE）
2. **安全优先**: 密码必须掩码为 `[REDACTED]`，API Key 至少 32 字符
3. **MCP 协议合规**: 使用官方 go-sdk

详细规范和约束请查阅 [AGENTS.md](AGENTS.md)。

## 多数据库架构

### 支持的数据库类型
| 类型 | 驱动 | Validator | 说明 |
|------|------|-----------|------|
| MySQL | go-sql-driver/mysql | mysql/validator.go | 基准实现 |
| MongoDB | mongo-driver | mongodb/validator.go | 文档数据库 |
| PostgreSQL | pgx/v5 | postgres/validator.go | WITH CTE 支持 |
| SQLite | go-sqlite3 (CGO) | sqlite/validator.go | 安全 PRAGMA |
| SQL Server | go-mssqldb | sqlserver/validator.go | EXEC sp_* |
| Oracle | go-ora/v2 | oracle/validator.go | FLASHBACK 查询 |
| ClickHouse | MySQL复用 | clickhouse/validator.go | MySQL协议兼容 |
| Doris | MySQL复用 | doris/validator.go | MySQL协议兼容 |
| MariaDB/TiDB | MySQL复用 | mysql/validator.go | 透明复用 |

### 驱动注册架构

```
PoolManager
├── registry: map[DatabaseType]DriverConstructor  // 驱动注册表
├── drivers:  map[string]Database                 // 驱动实例（按ID）
└── configs:  map[string]DatabaseConfig           // 配置缓存

注册流程（main.go）:
poolManager.RegisterDriver(dbType, func(config) Database {
    return driver.New(config)
})
```

关键文件:
- `src/database/interface.go` — Database 接口、DatabaseType 常量、DatabaseConfig 结构
- `src/database/pool.go` — PoolManager、DriverRegistry 模式、GetOrCreatePool
- `cmd/server/main.go` — 所有驱动注册、connectAllDatabases

### MySQL 协议兼容数据库

配置中使用 `protocol_compatible` 标记：
```yaml
databases:
  clickhouse-prod:
    type: clickhouse
    protocol_compatible: clickhouse  # 使用MySQL驱动+ClickHouse验证器
```

验证器注入:
```go
mysql.NewMySQLDriverWithValidator(config, clickhouse.ValidateClickHouseQuery)
```

### Validator 实现模式

每个数据库类型需要独立的 validator:
```
src/database/[dbtype]/validator.go
├── AllowedKeywords    // 允许的操作关键字
├── ForbiddenKeywords  // 禁止的操作关键字
├── ForbiddenPatterns  // 禁止的操作模式
└── Validate[DBType]Query(query string) error
```

关键约束:
- 每种数据库有特定的允许/禁止关键字
- MySQL 协议兼容库使用独立验证器（ClickHouse/Doris 有特有语法）
- 禁止修改 `AllowedKeywords`/`ForbiddenKeywords` 以绕过限制

## 配置示例

```yaml
databases:
  mysql-primary:
    type: mysql
    host: localhost
    port: 3306
    username: root
    password: ${MYSQL_PASSWORD}
    database: mydb
    pool_size: 5
    timeout: 30

  postgres-analytics:
    type: postgres
    host: localhost
    port: 5432
    username: postgres
    password: ${PG_PASSWORD}
    database: analytics

  sqlite-local:
    type: sqlite
    path: /data/local.db    # SQLite文件路径
    pool_size: 5

  clickhouse-prod:
    type: clickhouse
    protocol_compatible: clickhouse  # MySQL驱动+自定义验证器
    host: localhost
    port: 9000
    username: default
    password: ""
    database: default
```