package sqlserver

import (
	"fmt"
	"strings"
)

// AllowedKeywords SQL Server允许的SQL关键字（只读操作）
var AllowedKeywords = []string{
	"SELECT",   // 查询数据
	"WITH",     // WITH子句（CTE）
	"EXEC",     // EXEC执行存储过程（只允许特定的系统存储过程）
	"EXECUTE",  // EXECUTE同义词
	"SP_HELP",  // sp_help获取表信息
	"SP_HELPT", // sp_helptext获取对象定义
	"SP_SPACE", // sp_spaceused获取空间使用
	"SP_WHO",   // sp_who获取进程信息
	"SP_LOCK",  // sp_lock获取锁信息
}

// ForbiddenKeywords SQL Server禁止的SQL关键字（修改操作）
var ForbiddenKeywords = []string{
	"INSERT",     // 插入数据
	"UPDATE",     // 更新数据
	"DELETE",     // 删除数据
	"DROP",       // 删除表/数据库/索引
	"ALTER",      // 修改表结构
	"CREATE",     // 创建表/索引/存储过程
	"TRUNCATE",   // 清空表
	"GRANT",      // 授权
	"REVOKE",     // 撤销权限
	"DENY",       // 拒绝权限
	"BACKUP",     // 备份操作
	"RESTORE",    // 恢复操作
	"BULK",       // BULK INSERT
	"MERGE",      // MERGE语句（包含INSERT/UPDATE）
	"DBCC",       // DBCC命令（数据库控制命令）
	"KILL",       // KILL进程
	"SETUSER",    // SETUSER（权限切换）
	"SHUTDOWN",   // SHUTDOWN关闭数据库
	"RECONFIGURE", // RECONFIGURE配置更新
	"LOAD",       // LOAD数据库
	"DETACH",     // DETACH数据库
	"ATTACH",     // ATTACH数据库
}

// ForbiddenPatterns SQL Server禁止的操作模式
var ForbiddenPatterns = []string{
	"INSERT INTO",
	"UPDATE ",
	"DELETE FROM",
	"DROP TABLE",
	"DROP DATABASE",
	"DROP INDEX",
	"DROP VIEW",
	"DROP PROCEDURE",
	"DROP FUNCTION",
	"ALTER TABLE",
	"ALTER DATABASE",
	"ALTER INDEX",
	"ALTER VIEW",
	"ALTER PROCEDURE",
	"ALTER FUNCTION",
	"ALTER USER",
	"ALTER ROLE",
	"ALTER LOGIN",
	"CREATE TABLE",
	"CREATE DATABASE",
	"CREATE INDEX",
	"CREATE VIEW",
	"CREATE PROCEDURE",
	"CREATE FUNCTION",
	"CREATE USER",
	"CREATE ROLE",
	"CREATE LOGIN",
	"TRUNCATE TABLE",
	"BULK INSERT",
	"MERGE INTO",
	"BACKUP DATABASE",
	"BACKUP LOG",
	"RESTORE DATABASE",
	"RESTORE LOG",
	"DBCC CHECKDB",
	"DBCC CHECKTABLE",
	"DBCC SHRINK",
	"DBCC FREEPROCCACHE",
	"DBCC DROPCLEANBUFFERS",
	"EXEC sp_add",
	"EXEC sp_drop",
	"EXEC sp_grant",
	"EXEC sp_revoke",
	"EXEC sp_deny",
	"EXEC sp_configure",
	"EXECUTE sp_add",
	"EXECUTE sp_drop",
	"EXECUTE sp_grant",
	"EXECUTE sp_revoke",
	"EXECUTE sp_deny",
	"EXECUTE sp_configure",
	"KILL ",
	"SHUTDOWN",
	"GRANT ",
	"REVOKE ",
	"DENY ",
	"LOAD DATABASE",
	"DETACH DATABASE",
	"ATTACH DATABASE",
}

// AllowedSystemProcedures SQL Server允许查询的系统存储过程
var AllowedSystemProcedures = []string{
	"sp_help",         // 获取对象信息
	"sp_helptext",     // 获取对象定义
	"sp_spaceused",    // 获取空间使用
	"sp_who",          // 获取进程信息
	"sp_who2",         // 获取详细进程信息
	"sp_lock",         // 获取锁信息
	"sp_monitor",      // 监控统计
	"sp_columns",      // 获取列信息
	"sp_tables",       // 获取表列表
	"sp_stored_procedures", // 存储过程列表
	"sp_pkeys",        // 主键信息
	"sp_fkeys",        // 外键信息
	"sp_statistics",   // 统计信息
	"sp_indexinfo",    // 索引信息
	"sp_sproc_columns", // 存储过程参数
	"sp_getapplock",   // 获取应用锁（只读模式）
	"sp_releaseapplock", // 释放应用锁
}

// ValidateSQLServerQuery 验证SQL Server查询是否为只读操作
func ValidateSQLServerQuery(query string) error {
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

	// 特殊处理EXEC/EXECUTE命令
	if firstWord == "EXEC" || firstWord == "EXECUTE" {
		if !isAllowedExecCommand(upperQuery) {
			return fmt.Errorf("EXEC命令不安全：只允许查询类系统存储过程 (sp_help, sp_tables 等)")
		}
		return nil
	}

	// 特殊处理sp_开头的存储过程调用（无EXEC前缀）
	if strings.HasPrefix(firstWord, "SP_") {
		if !isAllowedSystemProcedure(firstWord) {
			return fmt.Errorf("存储过程 %s 不在允许列表中：只允许查询类系统存储过程", firstWord)
		}
		return nil
	}

	// 检查是否为禁止的关键字
	for _, forbidden := range ForbiddenKeywords {
		if firstWord == forbidden {
			return fmt.Errorf("操作 %s 不被允许：只允许只读操作 (SELECT, WITH, 安全的EXEC)", forbidden)
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
		return fmt.Errorf("操作 %s 不在允许列表中：只允许只读操作 (SELECT, WITH, 安全的EXEC)", firstWord)
	}

	// 检查查询中的禁止操作模式
	if containsForbiddenOperations(query) {
		return fmt.Errorf("查询包含禁止的操作：只允许只读操作")
	}

	return nil
}

// isAllowedExecCommand 检查EXEC命令是否为允许的系统存储过程
func isAllowedExecCommand(upperQuery string) bool {
	// 移除EXEC/EXECUTE关键字
	content := strings.TrimPrefix(upperQuery, "EXEC ")
	content = strings.TrimPrefix(content, "EXECUTE ")
	content = strings.TrimSpace(content)

	// 获取存储过程名（可能包含参数）
	parts := strings.Fields(content)
	if len(parts) == 0 {
		return false
	}

	procName := parts[0]
	return isAllowedSystemProcedure(procName)
}

// isAllowedSystemProcedure 检查存储过程是否在允许列表中
func isAllowedSystemProcedure(procName string) bool {
	// 移除括号（如果有参数）
	procName = strings.TrimSuffix(procName, "(")

	for _, allowed := range AllowedSystemProcedures {
		if strings.ToUpper(procName) == strings.ToUpper(allowed) {
			return true
		}
		// 支持部分匹配（如 sp_helpuser -> sp_help）
		if strings.HasPrefix(strings.ToUpper(procName), strings.ToUpper(allowed)) {
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
	return ValidateSQLServerQuery(query) == nil
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
	case "EXEC", "EXECUTE":
		return "EXEC"
	default:
		if strings.HasPrefix(firstWord, "SP_") {
			return "EXEC"
		}
		return "UNKNOWN"
	}
}