# mcp-dbquery

一个基于 MCP (Model Context Protocol) 的数据库查询工具，支持 MySQL 和 MongoDB 的只读查询操作。

## 功能特性

- 支持 MySQL 和 MongoDB 数据库
- 严格的只读操作限制（仅允许 SELECT、find、aggregate 等查询操作）
- 连接池管理
- 支持 STDIO 和 HTTP 两种传输模式
- 密码掩码日志（安全日志输出）
- API Key 认证（HTTP 模式）
- 查询超时和行数限制

## MCP 工具

| 工具名 | 描述 |
|--------|------|
| `query_mysql_data` | 执行 MySQL SELECT 查询 |
| `query_mongodb_data` | 执行 MongoDB find/aggregate 查询 |
| `get_schema` | 获取表/集合结构信息 |
| `get_indexes` | 获取索引元数据 |
| `list_tables` | 列出所有表/集合 |

## 安装

### 前置要求

- Go 1.25+
- MySQL 或 MongoDB 数据库

### 构建

```bash
# 克隆项目
git clone https://github.com/relaxyabc/mcp-dbquery.git
cd mcp-dbquery

# 构建
make build        # Linux/Mac: 输出到 bin/db-tools
make build-win    # Windows: 输出到 bin/db-tools.exe
```

## 配置

### 配置文件

配置文件位于 `configs/config.yaml`，支持环境变量替换：

```yaml
server:
  transport: stdio  # 传输模式: stdio (默认) 或 http
  host: 0.0.0.0
  port: 8080
  api_key: ${API_KEY}  # HTTP 模式需要，至少 32 字符

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
```

### 环境变量

```bash
export MYSQL_HOST=localhost
export MYSQL_USER=root
export MYSQL_PASSWORD=your_password
export MYSQL_DATABASE=your_database
export API_KEY=your_32_char_api_key  # HTTP 模式需要
```

## 使用

### STDIO 模式（推荐用于 MCP 客户端）

```bash
./bin/db-tools -c configs/config.yaml
```

### HTTP 模式

```bash
./bin/db-tools -c configs/config.yaml -t http
```

### MCP 客户端配置

在 MCP 客户端的配置文件中添加：

```json
{
  "mcpServers": {
    "db-query-tool": {
      "command": "./bin/db-tools",
      "args": ["-config", "configs/config.yaml"]
    }
  }
}
```

## 安全限制

为确保数据安全，本工具严格执行以下限制：

- **只读操作**: 禁止 INSERT、UPDATE、DELETE、DROP、ALTER、CREATE 等写操作
- **密码掩码**: 所有日志中密码显示为 `[REDACTED]`
- **API Key**: HTTP 模式要求 API Key 至少 32 字符
- **查询限制**: 最大返回行数 1000，查询超时 300 秒

## 开发

```bash
# 运行测试
make test              # 单元测试
make test-coverage     # 生成覆盖率报告
make test-integration  # 集成测试（需要 Docker）

# 代码格式化
make fmt

# 代码检查
make lint
```

## License

Apache License 2.0 - 详见 [LICENSE](LICENSE) 文件