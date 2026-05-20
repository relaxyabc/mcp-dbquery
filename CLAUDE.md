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