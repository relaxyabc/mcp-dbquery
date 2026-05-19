package oracle

import (
	"fmt"
	"strings"
)

// AllowedKeywords Oracle允许的SQL关键字（只读操作）
var AllowedKeywords = []string{
	"SELECT",   // 查询数据
	"WITH",     // WITH子句（CTE）
	"DESCRIBE", // DESCRIBE表结构
	"DESC",     // DESC简写
}

// ForbiddenKeywords Oracle禁止的SQL关键字（修改操作）
var ForbiddenKeywords = []string{
	"INSERT",     // 插入数据
	"UPDATE",     // 更新数据
	"DELETE",     // 删除数据
	"DROP",       // 删除表/索引/用户
	"ALTER",      // 修改表结构
	"CREATE",     // 创建表/索引/用户
	"TRUNCATE",   // 清空表
	"MERGE",      // MERGE语句
	"FLASHBACK",  // FLASHBACK（部分禁止）
	"PURGE",      // PURGE回收站
	"EXEC",       // EXEC执行
	"EXECUTE",    // EXECUTE执行
	"CALL",       // CALL调用存储过程
	"GRANT",      // 授权
	"REVOKE",     // 撤销权限
	"AUDIT",      // 审计
	"NOAUDIT",    // 取消审计
	"ANALYZE",    // 分析统计
	"VALIDATE",   // 验证结构
	"SET",        // SET约束/事务
	"LOCK",       // LOCK TABLE
	"COMMENT",    // COMMENT ON（添加注释）
	"RENAME",     // RENAME表
	"SHUTDOWN",   // 关闭数据库
	"STARTUP",    // 启动数据库
	"ARCHIVE",    // 归档日志
	"RECOVER",    // 恢复数据库
	"BEGIN",      // PL/SQL块开始
	"DECLARE",    // PL/SQL声明
}

// ForbiddenPatterns Oracle禁止的操作模式
var ForbiddenPatterns = []string{
	"INSERT INTO",
	"UPDATE ",
	"DELETE FROM",
	"DROP TABLE",
	"DROP INDEX",
	"DROP VIEW",
	"DROP PROCEDURE",
	"DROP FUNCTION",
	"DROP PACKAGE",
	"DROP USER",
	"DROP ROLE",
	"DROP DATABASE",
	"ALTER TABLE",
	"ALTER INDEX",
	"ALTER VIEW",
	"ALTER PROCEDURE",
	"ALTER FUNCTION",
	"ALTER PACKAGE",
	"ALTER USER",
	"ALTER ROLE",
	"ALTER DATABASE",
	"ALTER SYSTEM",
	"ALTER SESSION", // ALTER SESSION SET 禁止修改参数
	"CREATE TABLE",
	"CREATE INDEX",
	"CREATE VIEW",
	"CREATE PROCEDURE",
	"CREATE FUNCTION",
	"CREATE PACKAGE",
	"CREATE USER",
	"CREATE ROLE",
	"CREATE DATABASE",
	"TRUNCATE TABLE",
	"MERGE INTO",
	"FLASHBACK DATABASE",
	"FLASHBACK TABLE",
	"PURGE TABLE",
	"PURGE INDEX",
	"PURGE RECYCLEBIN",
	"EXEC ",
	"EXECUTE ",
	"CALL ",
	"GRANT ",
	"REVOKE ",
	"AUDIT ",
	"NOAUDIT ",
	"ANALYZE TABLE",
	"ANALYZE INDEX",
	"VALIDATE ",
	"SET CONSTRAINT",
	"SET TRANSACTION",
	"LOCK TABLE",
	"COMMENT ON",
	"RENAME ",
	"SHUTDOWN",
	"STARTUP",
	"ARCHIVE LOG",
	"RECOVER DATABASE",
	"BEGIN ",
	"DECLARE ",
}

// AllowedFlashbackCommands Oracle允许的FLASHBACK查询命令
// FLASHBACK QUERY允许查看历史数据，但不允许恢复
var AllowedFlashbackCommands = []string{
	"SELECT ", // SELECT ... AS OF SCN/TIMESTAMP 是允许的
}

// ValidateOracleQuery 验证Oracle查询是否为只读操作
func ValidateOracleQuery(query string) error {
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
			return fmt.Errorf("操作 %s 不被允许：只允许只读操作 (SELECT, WITH, DESCRIBE)", forbidden)
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
		return fmt.Errorf("操作 %s 不在允许列表中：只允许只读操作 (SELECT, WITH, DESCRIBE)", firstWord)
	}

	// 特殊检查FLASHBACK命令（允许FLASHBACK查询，禁止FLASHBACK恢复）
	if strings.Contains(upperQuery, "FLASHBACK") {
		if !isAllowedFlashbackQuery(upperQuery) {
			return fmt.Errorf("FLASHBACK恢复操作不被允许：只允许FLASHBACK查询 (SELECT AS OF)")
		}
	}

	// 检查查询中的禁止操作模式
	if containsForbiddenOperations(query) {
		return fmt.Errorf("查询包含禁止的操作：只允许只读操作")
	}

	return nil
}

// isAllowedFlashbackQuery 检查FLASHBACK查询是否为允许的只读类型
// 允许：SELECT ... AS OF SCN/TIMESTAMP, SELECT ... VERSIONS BETWEEN
// 禁止：FLASHBACK TABLE TO, FLASHBACK DATABASE TO
func isAllowedFlashbackQuery(upperQuery string) bool {
	// FLASHBACK查询必须是SELECT语句
	if !strings.HasPrefix(upperQuery, "SELECT") {
		return false
	}

	// 允许的FLASHBACK查询模式
	allowedPatterns := []string{
		"AS OF SCN",
		"AS OF TIMESTAMP",
		"VERSIONS BETWEEN SCN",
		"VERSIONS BETWEEN TIMESTAMP",
	}

	for _, pattern := range allowedPatterns {
		if strings.Contains(upperQuery, pattern) {
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
	return ValidateOracleQuery(query) == nil
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
	case "WITH":
		return "WITH"
	case "DESCRIBE", "DESC":
		return "DESCRIBE"
	default:
		return "UNKNOWN"
	}
}