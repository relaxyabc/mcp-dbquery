package clickhouse

import (
	"fmt"
	"strings"
)

// AllowedKeywords ClickHouse允许的SQL关键字（只读操作）
var AllowedKeywords = []string{
	"SELECT",   // 查询数据
	"SHOW",     // SHOW DATABASES, SHOW TABLES
	"DESCRIBE", // DESCRIBE TABLE
	"DESC",     // DESC简写
	"EXPLAIN",  // EXPLAIN查询计划
	"WITH",     // WITH子句（ClickHouse支持）
	"SET",      // SET设置（仅允许安全的查询设置）
}

// ForbiddenKeywords ClickHouse禁止的SQL关键字（修改操作）
var ForbiddenKeywords = []string{
	"INSERT",   // 插入数据
	"UPDATE",   // 更新数据（ClickHouse不支持UPDATE，但仍禁止）
	"DELETE",   // 删除数据（ClickHouse不支持DELETE，但仍禁止）
	"DROP",     // 删除表/数据库
	"ALTER",    // 修改表结构
	"CREATE",   // 创建表/索引
	"TRUNCATE", // 清空表
	"OPTIMIZE", // OPTIMIZE TABLE（ClickHouse特有优化操作）
	"SYSTEM",   // SYSTEM命令（ClickHouse系统命令）
	"KILL",     // KILL QUERY
	"GRANT",    // 授权
	"REVOKE",   // 撤销权限
	"ATTACH",   // ATTACH PARTITION
	"DETACH",   // DETACH PARTITION
	"RENAME",   // 重命名表
}

// ForbiddenPatterns ClickHouse禁止的操作模式
var ForbiddenPatterns = []string{
	"INSERT INTO",
	"UPDATE ",
	"DELETE FROM",
	"DROP TABLE",
	"DROP DATABASE",
	"DROP DICTIONARY",
	"DROP VIEW",
	"ALTER TABLE",
	"ALTER DATABASE",
	"ALTER USER",
	"CREATE TABLE",
	"CREATE DATABASE",
	"CREATE DICTIONARY",
	"CREATE VIEW",
	"CREATE USER",
	"CREATE FUNCTION",
	"TRUNCATE TABLE",
	"OPTIMIZE TABLE",
	"SYSTEM STOP",
	"SYSTEM START",
	"SYSTEM FLUSH",
	"SYSTEM RELOAD",
	"KILL QUERY",
	"KILL TRANSACTION",
	"GRANT ",
	"REVOKE ",
	"ATTACH PARTITION",
	"DETACH PARTITION",
	"RENAME TABLE",
	"SET ROLE",
	"SET PASSWORD",
}

// AllowedSystemTables ClickHouse允许查询的系统表
var AllowedSystemTables = []string{
	"system.tables",        // 表信息
	"system.columns",       // 列信息
	"system.databases",     // 数据库列表
	"system.parts",         // 分区信息
	"system.parts_columns", // 分区列信息
	"system.processes",     // 查询进程（只读）
	"system.metrics",       // 指标
	"system.events",        // 事件
	"system.asynchronous_metrics", // 异步指标
	"system.functions",     // 函数列表
	"system.users",         // 用户信息
	"system.roles",         // 角色信息
	"system.clusters",      // 集群信息
	"system.settings",      // 设置信息
	"system.query_log",     // 查询日志（只读）
}

// ValidateClickHouseQuery 验证ClickHouse查询是否为只读操作
func ValidateClickHouseQuery(query string) error {
	// 去除前后空白
	query = strings.TrimSpace(query)

	// 检查空查询
	if query == "" {
		return fmt.Errorf("查询语句不能为空")
	}

	// 移除SQL注释
	query = removeComments(query)

	// 再次去除空白
	query = strings.TrimSpace(query)
	if query == "" {
		return fmt.Errorf("查询语句不能为空")
	}

	// 检查多语句查询
	if strings.Contains(query, ";") {
		if !strings.HasSuffix(query, ";") || strings.Count(query, ";") > 1 {
			return fmt.Errorf("多语句查询不被允许（安全限制）")
		}
	}

	// 获取第一个关键字
	upperQuery := strings.ToUpper(query)
	words := strings.Fields(upperQuery)
	if len(words) == 0 {
		return fmt.Errorf("查询语句不能为空")
	}

	firstWord := words[0]

	// 检查是否为禁止的关键字
	for _, forbidden := range ForbiddenKeywords {
		if firstWord == forbidden {
			return fmt.Errorf("操作 %s 不被允许：只允许只读操作 (SELECT, SHOW, DESCRIBE, EXPLAIN, WITH)", forbidden)
		}
	}

	// 检查是否为允许的关键字
	allowed := false
	for _, permit := range AllowedKeywords {
		if firstWord == permit {
			allowed = true
			break
		}
	}

	if !allowed {
		return fmt.Errorf("操作 %s 不在允许列表中：只允许只读操作 (SELECT, SHOW, DESCRIBE, EXPLAIN, WITH)", firstWord)
	}

	// 检查查询中的禁止操作模式
	// 注意：SHOW CREATE TABLE包含"CREATE"但不应该被禁止
	if containsForbiddenOperations(query) {
		// 特殊允许：SHOW CREATE TABLE等
		if !isAllowedShowCommand(upperQuery) {
			return fmt.Errorf("查询包含禁止的操作：只允许只读操作")
		}
	}

	// 特殊检查SET命令（只禁止权限相关的SET）
	if firstWord == "SET" {
		if isForbiddenSet(query) {
			return fmt.Errorf("SET命令不安全：禁止设置权限、密码、角色等")
		}
	}

	return nil
}

// isAllowedShowCommand 检查SHOW命令是否为允许的类型（即使包含禁止关键字）
func isAllowedShowCommand(upperQuery string) bool {
	allowedShows := []string{
		"SHOW CREATE TABLE",
		"SHOW CREATE DATABASE",
		"SHOW CREATE DICTIONARY",
		"SHOW CREATE VIEW",
		"SHOW CREATE FUNCTION",
	}

	for _, allowed := range allowedShows {
		if strings.HasPrefix(upperQuery, allowed) {
			return true
		}
	}
	return false
}

// containsForbiddenOperations 检查查询中是否包含禁止操作
func containsForbiddenOperations(query string) bool {
	upperQuery := strings.ToUpper(query)

	for _, pattern := range ForbiddenPatterns {
		if strings.Contains(upperQuery, pattern) {
			return true
		}
	}

	return false
}

// isForbiddenSet 检查SET命令是否为禁止的权限相关设置
func isForbiddenSet(query string) bool {
	upperQuery := strings.ToUpper(query)

	// 禁止的SET设置
	forbiddenSets := []string{
		"SET ROLE",
		"SET PASSWORD",
		"SET DEFAULT_ROLE",
		"SET ACCESS",
	}

	for _, forbidden := range forbiddenSets {
		if strings.Contains(upperQuery, forbidden) {
			return true
		}
	}

	// 允许的SET设置（查询相关）
	allowedSets := []string{
		"SET max_rows",
		"SET max_bytes",
		"SET max_execution_time",
		"SET max_memory_usage",
		"SET readonly", // readonly=1是安全的设置
	}

	for _, allowed := range allowedSets {
		if strings.Contains(upperQuery, strings.ToUpper(allowed)) {
			return false
		}
	}

	// 其他SET命令默认禁止
	return true
}

// removeComments 移除SQL查询中的注释
func removeComments(query string) string {
	result := query

	// 移除 /* ... */ 多行注释
	for {
		start := strings.Index(result, "/*")
		if start == -1 {
			break
		}
		end := strings.Index(result[start:], "*/")
		if end == -1 {
			result = result[:start]
			break
		}
		result = result[:start] + result[start+end+2:]
	}

	// 移除 -- 开头的单行注释
	lines := strings.Split(result, "\n")
	filteredLines := []string{}
	for _, line := range lines {
		commentPos := strings.Index(line, "--")
		if commentPos != -1 {
			line = line[:commentPos]
		}
		filteredLines = append(filteredLines, line)
	}

	return strings.Join(filteredLines, " ")
}

// IsReadOnlyQuery 快速判断查询是否为只读
func IsReadOnlyQuery(query string) bool {
	return ValidateClickHouseQuery(query) == nil
}

// GetQueryType 获取查询类型
func GetQueryType(query string) string {
	query = strings.TrimSpace(query)
	if query == "" {
		return "UNKNOWN"
	}

	firstWord := strings.ToUpper(strings.Fields(query)[0])

	switch firstWord {
	case "SELECT":
		return "SELECT"
	case "SHOW":
		return "SHOW"
	case "DESCRIBE", "DESC":
		return "DESCRIBE"
	case "EXPLAIN":
		return "EXPLAIN"
	case "WITH":
		return "WITH"
	default:
		return "UNKNOWN"
	}
}