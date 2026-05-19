package postgres

import (
	"fmt"
	"strings"
)

// AllowedKeywords PostgreSQL允许的SQL关键字（只读操作）
var AllowedKeywords = []string{
	"SELECT",   // 查询数据
	"SHOW",     // 显示信息
	"DESCRIBE", // 描述表结构
	"DESC",     // DESCRIBE的简写
	"EXPLAIN",  // 解释查询执行计划
	"WITH",     // CTE (Common Table Expression)
}

// ForbiddenKeywords PostgreSQL禁止的SQL关键字（修改操作）
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
	"COPY",     // COPY命令
	"VACUUM",   // VACUUM操作
	"REINDEX",  // 重建索引
	"CLUSTER",  // 聚簇操作
	"ANALYZE",  // 分析操作
}

// ForbiddenPatterns PostgreSQL禁止的操作模式
var ForbiddenPatterns = []string{
	"INSERT INTO",
	"UPDATE ",
	"DELETE FROM",
	"DROP TABLE",
	"DROP DATABASE",
	"ALTER TABLE",
	"CREATE TABLE",
	"TRUNCATE TABLE",
	"REPLACE INTO",
	"COPY ",
	"VACUUM ",
	"REINDEX ",
	"CLUSTER ",
	"ANALYZE ",
}

// ValidatePostgresQuery 验证PostgreSQL查询是否为只读操作
func ValidatePostgresQuery(query string) error {
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
	firstWord := strings.ToUpper(strings.Fields(query)[0])

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

	// 检查子查询中的禁止操作
	if containsForbiddenOperations(query) {
		return fmt.Errorf("查询包含禁止的操作：只允许只读操作")
	}

	return nil
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
	return ValidatePostgresQuery(query) == nil
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