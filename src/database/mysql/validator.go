package mysql

import (
	"fmt"
	"strings"
)

// AllowedKeywords MySQL允许的SQL关键字（只读操作）
// 宪章原则 I 要求：仅允许SELECT、SHOW、DESCRIBE、EXPLAIN
var AllowedKeywords = []string{
	"SELECT",   // 查询数据
	"SHOW",     // 显示信息（表、索引、状态等）
	"DESCRIBE", // 描述表结构
	"DESC",     // DESCRIBE的简写
	"EXPLAIN",  // 解释查询执行计划
}

// ForbiddenKeywords MySQL禁止的SQL关键字（修改操作）
// 宪章原则 I 要求：严格禁止所有修改操作
var ForbiddenKeywords = []string{
	"INSERT",   // 插入数据
	"UPDATE",   // 更新数据
	"DELETE",   // 删除数据
	"DROP",     // 删除表/数据库
	"ALTER",    // 修改表结构
	"CREATE",   // 创建表/数据库
	"TRUNCATE", // 清空表
	"REPLACE",  // 替换数据
	"GRANT",    // 授权
	"REVOKE",   // 撤销权限
	"RENAME",   // 重命名表
	"LOCK",     // 锁表
	"UNLOCK",   // 解锁表
	"LOAD",     // 加载数据
	"CALL",     // 调用存储过程
	"EXECUTE",  // 执行语句
}

// ValidateMySQLQuery 验证MySQL查询是否为只读操作
func ValidateMySQLQuery(query string) error {
	// 去除前后空白
	query = strings.TrimSpace(query)

	// 检查空查询
	if query == "" {
		return fmt.Errorf("查询语句不能为空")
	}

	// 检查多语句查询（防止SQL注入）
	if strings.Contains(query, ";") {
		// 允许末尾的分号
		if !strings.HasSuffix(query, ";") || strings.Count(query, ";") > 1 {
			return fmt.Errorf("多语句查询不被允许（安全限制）")
		}
	}

	// 获取第一个关键字
	firstWord := strings.ToUpper(strings.Fields(query)[0])

	// 检查是否为禁止的关键字
	for _, forbidden := range ForbiddenKeywords {
		if firstWord == forbidden {
			return fmt.Errorf("操作 %s 不被允许：只允许只读操作 (SELECT, SHOW, DESCRIBE, EXPLAIN)", forbidden)
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
		return fmt.Errorf("操作 %s 不在允许列表中：只允许只读操作 (SELECT, SHOW, DESCRIBE, EXPLAIN)", firstWord)
	}

	// 检查子查询中的禁止操作（深度检查）
	if containsForbiddenOperations(query) {
		return fmt.Errorf("查询包含禁止的操作：只允许只读操作")
	}

	return nil
}

// containsForbiddenOperations 检查查询中是否包含禁止操作（深度检查）
func containsForbiddenOperations(query string) bool {
	upperQuery := strings.ToUpper(query)

	// 检查常见的修改模式
	forbiddenPatterns := []string{
		"INSERT INTO",
		"UPDATE ",
		"DELETE FROM",
		"DROP TABLE",
		"DROP DATABASE",
		"ALTER TABLE",
		"CREATE TABLE",
		"TRUNCATE TABLE",
		"REPLACE INTO",
	}

	for _, pattern := range forbiddenPatterns {
		if strings.Contains(upperQuery, pattern) {
			return true
		}
	}

	return false
}

// IsReadOnlyQuery 快速判断查询是否为只读（简化版）
func IsReadOnlyQuery(query string) bool {
	return ValidateMySQLQuery(query) == nil
}

// GetQueryType 获取查询类型（SELECT/SHOW/DESCRIBE/EXPLAIN）
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
	default:
		return "UNKNOWN"
	}
}
