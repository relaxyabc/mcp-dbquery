package doris

import (
	"fmt"
	"strings"
)

// AllowedKeywords Doris允许的SQL关键字（只读操作）
var AllowedKeywords = []string{
	"SELECT",   // 查询数据
	"SHOW",     // SHOW DATABASES, SHOW TABLES, SHOW CATALOGS
	"DESCRIBE", // DESCRIBE TABLE
	"DESC",     // DESC简写
	"EXPLAIN",  // EXPLAIN查询计划
	"WITH",     // WITH子句（Doris支持CTE）
}

// ForbiddenKeywords Doris禁止的SQL关键字（修改操作）
var ForbiddenKeywords = []string{
	"INSERT",   // 插入数据
	"UPDATE",   // 更新数据
	"DELETE",   // 删除数据
	"DROP",     // 删除表/数据库/CATALOG
	"ALTER",    // 修改表结构
	"CREATE",   // 创建表/索引/CATALOG（但SHOW CREATE TABLE允许）
	"TRUNCATE", // 清空表
	"GRANT",    // 授权
	"REVOKE",   // 撤销权限
	"SET",      // SET设置（禁止权限相关）
	"ADMIN",    // ADMIN命令（单独处理）
	"RENAME",   // 重命名表
	"STOP",     // STOP CANCEL
	"CANCEL",   // CANCEL LOAD/EXPORT
	"LOAD",     // LOAD LABEL
	"EXPORT",   // EXPORT TABLE
}

// ForbiddenPatterns Doris禁止的操作模式
var ForbiddenPatterns = []string{
	"INSERT INTO",
	"UPDATE ",
	"DELETE FROM",
	"DROP TABLE",
	"DROP DATABASE",
	"DROP CATALOG",
	"DROP RESOURCE",
	"DROP USER",
	"DROP ROLE",
	"ALTER TABLE",
	"ALTER DATABASE",
	"ALTER CATALOG",
	"ALTER USER",
	"ALTER SYSTEM",
	"CREATE TABLE",
	"CREATE DATABASE",
	"CREATE CATALOG",
	"CREATE RESOURCE",
	"CREATE USER",
	"CREATE ROLE",
	"CREATE FUNCTION",
	"CREATE VIEW",
	"TRUNCATE TABLE",
	"GRANT ",
	"REVOKE ",
	"SET PASSWORD",
	"SET ROLE",
	"ADMIN SET",
	"ADMIN CANCEL",
	"ADMIN SHOW", // 只允许特定ADMIN SHOW
	"RENAME TABLE",
	"STOP CANCEL",
	"CANCEL LOAD",
	"CANCEL EXPORT",
	"LOAD LABEL",
	"EXPORT TABLE",
}

// AllowedAdminCommands Doris允许的ADMIN SHOW命令
var AllowedAdminCommands = []string{
	"ADMIN SHOW FRONTENDS",     // FE节点信息
	"ADMIN SHOW BACKENDS",      // BE节点信息
	"ADMIN SHOW BROKER",        // Broker信息
	"ADMIN SHOW REPLICA",       // 副本信息
	"ADMIN SHOW TABLETS",       // Tablet信息
	"ADMIN SHOW CONFIG",        // 配置信息
	"ADMIN SHOW PROC",          // PROC路径信息
	"ADMIN SHOW REBALANCE",     // 重平衡信息
	"ADMIN SHOW DATA SKETCH",   // 数据概览
}

// ValidateDorisQuery 验证Doris查询是否为只读操作
func ValidateDorisQuery(query string) error {
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

	// 特殊处理ADMIN命令（只允许特定ADMIN SHOW）
	if firstWord == "ADMIN" {
		if !isAllowedAdminCommand(upperQuery) {
			return fmt.Errorf("ADMIN命令不安全：只允许 ADMIN SHOW 相关命令")
		}
		return nil
	}

	// 特殊处理SHOW CREATE命令（包含CREATE但不应被禁止）
	if firstWord == "SHOW" && isAllowedShowCreateCommand(upperQuery) {
		return nil
	}

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
	if containsForbiddenOperations(query) {
		// 特殊允许：SHOW CREATE TABLE等
		if !isAllowedShowCreateCommand(upperQuery) {
			return fmt.Errorf("查询包含禁止的操作：只允许只读操作")
		}
	}

	return nil
}

// isAllowedShowCreateCommand 检查SHOW CREATE命令是否为允许的类型
func isAllowedShowCreateCommand(upperQuery string) bool {
	allowedShows := []string{
		"SHOW CREATE TABLE",
		"SHOW CREATE DATABASE",
		"SHOW CREATE CATALOG",
		"SHOW CREATE RESOURCE",
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

// isAllowedAdminCommand 检查ADMIN命令是否为允许的只读类型
func isAllowedAdminCommand(upperQuery string) bool {
	for _, allowed := range AllowedAdminCommands {
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
	return ValidateDorisQuery(query) == nil
}

// GetQueryType 获取查询类型
func GetQueryType(query string) string {
	query = strings.TrimSpace(query)
	if query == "" {
		return "UNKNOWN"
	}

	words := strings.Fields(strings.ToUpper(query))
	if len(words) == 0 {
		return "UNKNOWN"
	}

	firstWord := words[0]

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
	case "ADMIN":
		return "ADMIN_SHOW"
	default:
		return "UNKNOWN"
	}
}