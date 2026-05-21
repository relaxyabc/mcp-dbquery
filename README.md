# mcp-dbquery

一个基于 MCP (Model Context Protocol) 的数据库查询工具，支持多种数据库的只读查询操作。

## 功能特性

- **多数据库支持**: MySQL、MongoDB、PostgreSQL、SQLite、SQL Server、Oracle
- **MySQL 协议兼容**: ClickHouse、Doris、MariaDB、TiDB（复用 MySQL 驱动）
- **严格只读限制**: 仅允许 SELECT、find、aggregate 等查询操作
- **双传输模式**: STDIO（MCP 客户端）和 HTTP（API 服务）
- **连接池管理**: 多数据库连接池复用
- **安全日志**: 密码自动掩码为 `[REDACTED]`
- **API Key 认证**: HTTP 模式支持 API Key 认证
- **密码加密**: 配置文件密码支持加密存储，CLI 工具加密/解密

## 支持的数据库

| 数据库 | 驱动 | 说明 |
|--------|------|------|
| MySQL | go-sql-driver/mysql | 基准实现 |
| MongoDB | mongo-driver | 文档数据库 |
| PostgreSQL | pgx/v5 | 支持 WITH CTE |
| SQLite | go-sqlite3 (CGO) | 安全 PRAGMA 白名单 |
| SQL Server | go-mssqldb | EXEC sp_* 系统存储过程 |
| Oracle | go-ora/v2 | FLASHBACK 查询 |
| ClickHouse | MySQL 协议 | 使用 MySQL 驱动 + 自定义验证器 |
| Doris | MySQL 协议 | 使用 MySQL 驱动 + 自定义验证器 |
| MariaDB | MySQL 协议 | 透明复用 MySQL 验证器 |
| TiDB | MySQL 协议 | 透明复用 MySQL 验证器 |

## MCP 工具

| 工具名 | 描述 |
|--------|------|
| `query_mysql_data` | 执行 SQL SELECT 查询（MySQL/PostgreSQL/SQLite/SQL Server/Oracle） |
| `query_mongodb_data` | 执行 MongoDB find 查询 |
| `aggregate_mongodb_data` | 执行 MongoDB 聚合管道查询 |
| `count_mongodb_data` | 执行 MongoDB 计数查询 |
| `distinct_mongodb_data` | 执行 MongoDB distinct 查询（获取字段唯一值） |
| `get_schema` | 获取表/集合结构信息 |
| `get_indexes` | 获取索引元数据 |
| `list_tables` | 列出所有表/集合 |

## 安装

### 前置要求

- Go 1.25+
- SQLite 需要 CGO 支持（安装 C 编译器）

### 构建

```bash
# 克隆项目
git clone https://github.com/relaxyabc/mcp-dbquery.git
cd mcp-dbquery

# 构建
make build        # Linux/Mac: 输出到 bin/db-tools
make build-win    # Windows: 输出到 bin/db-tools.exe
```

## 密码加密

为增强安全性，配置文件中的数据库密码可以使用加密格式存储。

### 加密算法

使用 PBEWithMD5AndDES (PKCS#5 v1.5) 算法，密文格式为：

```
enc:v1:<base64(salt+ciphertext)>
```

### CLI 加密命令

**密钥优先级**: 命令行 > 配置文件 > 环境变量

| 来源 | 参数/配置 | 优先级 |
|------|-----------|--------|
| 命令行密钥 | `--key <密钥>` | 最高 |
| 命令行密钥文件 | `--key-file <路径>` | 高 |
| 配置文件密钥 | `encryption_key: <密钥>` | 中 |
| 配置文件密钥文件 | `encryption_key_file: <路径>` | 低 |
| 环境变量 | `DBQUERY_ENCRYPTION_KEY` | 最低 |

**加密密码**:
```bash
# 使用环境变量密钥
export DBQUERY_ENCRYPTION_KEY="yourSecretKey123"
./bin/db-tools encrypt mySecretPassword
# 输出: enc:v1:U2FsdGVkX1+abc123def456=

# 使用命令行密钥（优先级最高）
./bin/db-tools encrypt mySecretPassword --key myCustomKey

# 或使用密钥文件
./bin/db-tools encrypt mySecretPassword --key-file /path/to/key.txt
```

**解密验证**:
```bash
./bin/db-tools decrypt "enc:v1:U2FsdGVkX1+abc123def456="
# 输出: mySecretPassword
```

**服务启动时指定密钥**:
```bash
# 命令行密钥
./bin/db-tools --key yourSecretKey

# 命令行密钥文件
./bin/db-tools --key-file /path/to/key.txt
```

### 配置文件使用加密密码

将加密后的密文写入配置文件：

```yaml
server:
  transport: stdio
  # 加密密钥配置（可选，优先级低于命令行）
  encryption_key: yourSecretKey           # 配置文件密钥
  encryption_key_file: /path/to/key.txt   # 配置文件密钥文件

databases:
  mysql-primary:
    type: mysql
    host: localhost
    port: 3306
    username: root
    password: enc:v1:U2FsdGVkX1+abc123def456=  # 加密密码
    database: mydb
```

服务启动时会自动解密加密密码并连接数据库。

### 安全说明

- 加密密钥优先级: 命令行 `--key` > 命令行 `--key-file` > 配置文件 `encryption_key` > 配置文件 `encryption_key_file` > 环境变量 `DBQUERY_ENCRYPTION_KEY`
- 明文密码仍支持（向后兼容），但会发出警告
- 日志中密码始终显示为 `[ENCRYPTED]` 或 `[REDACTED]`
- PBEWithMD5AndDES 是兼容性算法，建议在生产环境使用更安全的密钥管理方案

## 配置

配置文件位于 `configs/config.yaml`：

```yaml
server:
  transport: stdio  # 传输模式: stdio (默认) 或 http
  host: 0.0.0.0
  port: 8080
  api_key: ${API_KEY}  # HTTP 模式需要，至少 32 字符
  # 加密密钥配置（可选，优先级低于命令行）
  # encryption_key: yourSecretKey           # 配置文件密钥
  # encryption_key_file: /path/to/key.txt   # 配置文件密钥文件

databases:
  mysql-primary:
    type: mysql
    host: ${MYSQL_HOST}
    port: 3306
    username: ${MYSQL_USER}
    password: ${MYSQL_PASSWORD}
    database: ${MYSQL_DATABASE}
    pool_size: 5
    timeout: 30

  postgres-analytics:
    type: postgres
    host: ${PG_HOST}
    port: 5432
    username: ${PG_USER}
    password: ${PG_PASSWORD}
    database: analytics

  mongodb-docs:
    type: mongodb
    host: ${MONGO_HOST}
    port: 27017
    username: ${MONGO_USER}
    password: ${MONGO_PASSWORD}
    database: documents

  sqlite-local:
    type: sqlite
    path: /data/local.db    # SQLite 文件路径

  clickhouse-prod:
    type: clickhouse
    protocol_compatible: clickhouse  # MySQL 驱动 + ClickHouse 验证器
    host: localhost
    port: 9000
    username: default
    password: ""
    database: default
```

### 环境变量

```bash
export MYSQL_HOST=localhost
export MYSQL_USER=root
export MYSQL_PASSWORD=your_password
export MYSQL_DATABASE=your_database
export API_KEY=your_32_char_api_key  # HTTP 模式需要
export DBQUERY_ENCRYPTION_KEY=yourSecretKey  # 加密密码解密密钥
```

## MCP 使用示例

### STDIO 模式

STDIO 模式适用于 MCP 客户端（如 Claude Desktop、Cursor、VS Code MCP 扩展）：

**启动服务**:
```bash
./bin/db-tools -c configs/config.yaml
# 或使用 Makefile
make run
```

**Claude Desktop 配置** (`claude_desktop_config.json`):
```json
{
  "mcpServers": {
    "db-query": {
      "command": "/absolute/path/to/bin/db-tools",
      "args": ["-c", "/absolute/path/to/configs/config.yaml"]
    }
  }
}
```

**Cursor / VS Code MCP 配置**:
```json
{
  "mcp": {
    "servers": {
      "db-query": {
        "command": "./bin/db-tools",
        "args": ["-c", "configs/config.yaml"],
        "cwd": "/path/to/mcp-dbquery"
      }
    }
  }
}
```

启动后，AI 客户端可通过 MCP 工具执行查询：
- `query_mysql_data`: "查询 users 表前 10 条记录"
- `get_schema`: "查看 orders 表结构"
- `list_tables`: "列出所有表"

### HTTP 模式

HTTP 模式适用于需要 REST API 访问的场景，或远程 MCP 客户端连接：

**启动服务**:
```bash
./bin/db-tools -c configs/config.yaml -t http
# 或使用 Makefile
make run-http
```

服务启动后监听 `http://0.0.0.0:8080`

**MCP 客户端 HTTP 配置**:

Claude Desktop (`claude_desktop_config.json`):
```json
{
  "mcpServers": {
    "db-query": {
      "url": "http://localhost:8080/mcp",
      "headers": {
        "X-API-Key": "your_32_char_api_key"
      }
    }
  }
}
```

Cursor / VS Code MCP:
```json
{
  "mcp": {
    "servers": {
      "db-query": {
        "url": "http://localhost:8080/mcp",
        "headers": {
          "X-API-Key": "your_32_char_api_key"
        }
      }
    }
  }
}
```

**API 调用示例**:

> **注意**: MCP 使用 JSON-RPC 2.0 协议，请求必须包含 `jsonrpc` 和 `id` 字段。

初始化会话（首次请求必需）:
```bash
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your_32_char_api_key" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "initialize",
    "params": {
      "protocolVersion": "2025-03-26",
      "capabilities": {},
      "clientInfo": {"name": "curl-client", "version": "1.0"}
    }
  }'
```

查询数据:
```bash
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your_32_char_api_key" \
  -d '{
    "jsonrpc": "2.0",
    "id": 2,
    "method": "tools/call",
    "params": {
      "name": "query_mysql_data",
      "arguments": {
        "database_id": "mysql-primary",
        "query": "SELECT * FROM users LIMIT 10"
      }
    }
  }'
```

获取表结构:
```bash
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your_32_char_api_key" \
  -d '{
    "jsonrpc": "2.0",
    "id": 3,
    "method": "tools/call",
    "params": {
      "name": "get_schema",
      "arguments": {
        "database_id": "mysql-primary",
        "table_name": "users"
      }
    }
  }'
```

列出所有表:
```bash
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your_32_char_api_key" \
  -d '{
    "jsonrpc": "2.0",
    "id": 4,
    "method": "tools/call",
    "params": {
      "name": "list_tables",
      "arguments": {
        "database_id": "mysql-primary"
      }
    }
  }'
```

MongoDB 查询:
```bash
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your_32_char_api_key" \
  -d '{
    "jsonrpc": "2.0",
    "id": 5,
    "method": "tools/call",
    "params": {
      "name": "query_mongodb_data",
      "arguments": {
        "database_id": "mongo-analytics",
        "collection": "users",
        "filter": {"status": "active"},
        "limit": 10
      }
    }
  }'
```

MongoDB 聚合查询:
```bash
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your_32_char_api_key" \
  -d '{
    "jsonrpc": "2.0",
    "id": 6,
    "method": "tools/call",
    "params": {
      "name": "aggregate_mongodb_data",
      "arguments": {
        "database_id": "mongo-analytics",
        "collection": "orders",
        "pipeline": [
          {"$match": {"status": "completed"}},
          {"$group": {"_id": "$product", "total": {"$sum": "$price"}}}
        ]
      }
    }
  }'
```

MongoDB 计数查询:
```bash
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your_32_char_api_key" \
  -d '{
    "jsonrpc": "2.0",
    "id": 7,
    "method": "tools/call",
    "params": {
      "name": "count_mongodb_data",
      "arguments": {
        "database_id": "mongo-analytics",
        "collection": "users",
        "filter": {"status": "active"}
      }
    }
  }'
```

MongoDB Distinct 查询:
```bash
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your_32_char_api_key" \
  -d '{
    "jsonrpc": "2.0",
    "id": 8,
    "method": "tools/call",
    "params": {
      "name": "distinct_mongodb_data",
      "arguments": {
        "database_id": "mongo-analytics",
        "collection": "users",
        "field": "country"
      }
    }
  }'
```

## 安全限制

为确保数据安全，本工具严格执行以下限制：

- **只读操作**: 禁止 INSERT、UPDATE、DELETE、DROP、ALTER、CREATE、TRUNCATE
- **密码掩码**: 所有日志中密码显示为 `[REDACTED]`
- **API Key**: HTTP 模式要求 API Key 至少 32 字符
- **查询限制**: 最大返回行数 1000，查询超时 300 秒
- **验证器**: 每种数据库有独立的 SQL/操作验证器，无法绕过

## 开发

```bash
# 运行测试
make test              # 单元测试
make test-coverage     # 生成覆盖率报告

# 代码格式化
make fmt

# 代码检查
make lint

# 构建
make build             # Linux/Mac
make build-win         # Windows
```

## License

Apache License 2.0 - 详见 [LICENSE](LICENSE) 文件