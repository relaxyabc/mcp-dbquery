package sqlite

import (
	"fmt"
	"strings"
)

// AllowedKeywords SQLite允许的SQL关键字（只读操作）
var AllowedKeywords = []string{
	"SELECT",   // 查询数据
	"PRAGMA",   // PRAGMA命令（仅安全类）
}

// ForbiddenKeywords SQLite禁止的SQL关键字（修改操作）
var ForbiddenKeywords = []string{
	"INSERT",   // 插入数据
	"UPDATE",   // 更新数据
	"DELETE",   // 删除数据
	"DROP",     // 删除表/数据库
	"ALTER",    // 修改表结构
	"CREATE",   // 创建表/索引
	"TRUNCATE", // 清空表
	"REPLACE",  // 替换数据
	"ATTACH",   // 附加数据库
	"DETACH",   // 分离数据库
	"GRANT",    // 授权
	"REVOKE",   // 撤销权限
	"RENAME",   // 重命名表
}

// ForbiddenPatterns SQLite禁止的操作模式
var ForbiddenPatterns = []string{
	"INSERT INTO",
	"UPDATE ",
	"DELETE FROM",
	"DROP TABLE",
	"DROP INDEX",
	"ALTER TABLE",
	"CREATE TABLE",
	"CREATE INDEX",
	"TRUNCATE TABLE",
	"REPLACE INTO",
	"ATTACH DATABASE",
	"DETACH DATABASE",
}

// SafePragmas SQLite安全的PRAGMA命令列表
var SafePragmas = []string{
	"table_info",         // 获取表结构
	"index_list",         // 列出索引
	"index_info",         // 获取索引信息
	"database_list",      // 列出数据库
	"compile_options",    // 编译选项
	"foreign_key_list",   // 外键列表
	"collation_list",     // 排序规则列表
	"function_list",      // 函数列表
	"module_list",        // 模块列表
	"pragma_list",        // PRAGMA列表
}

// ValidateSQLiteQuery 验证SQLite查询是否为只读操作
func ValidateSQLiteQuery(query string) error {
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
			return fmt.Errorf("操作 %s 不被允许：只允许只读操作 (SELECT, 安全PRAGMA)", forbidden)
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
		return fmt.Errorf("操作 %s 不在允许列表中：只允许只读操作 (SELECT, 安全PRAGMA)", firstWord)
	}

	// 特殊处理PRAGMA命令
	if firstWord == "PRAGMA" {
		if !IsSafePragma(query) {
			return fmt.Errorf("PRAGMA命令不安全：只允许 table_info, index_list 等查询类PRAGMA")
		}
	}

	// 检查子查询中的禁止操作
	if containsForbiddenOperations(query) {
		return fmt.Errorf("查询包含禁止的操作：只允许只读操作")
	}

	return nil
}

// IsSafePragma 检查PRAGMA命令是否为安全的只读类型
func IsSafePragma(query string) bool {
	// 提取PRAGMA后面的命令名
	upperQuery := strings.ToUpper(query)

	// 移除PRAGMA关键字
	pragmaContent := strings.TrimPrefix(upperQuery, "PRAGMA ")
	pragmaContent = strings.TrimSpace(pragmaContent)

	// 获取命令名（可能包含表名等）
	parts := strings.Fields(pragmaContent)
	if len(parts) == 0 {
		return false
	}

	// 检查是否为安全PRAGMA
	pragmaName := strings.ToLower(parts[0])
	for _, safe := range SafePragmas {
		if pragmaName == safe || strings.HasPrefix(pragmaName, safe+"(") {
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
	return ValidateSQLiteQuery(query) == nil
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
	case "PRAGMA":
		return "PRAGMA"
	default:
		return "UNKNOWN"
	}
}