package integration

import (
	"testing"

	"github.com/relaxyabc/mcp-dbquery/src/database/sqlite"
)

// TestSQLiteValidator 测试SQLite验证器
func TestSQLiteValidator(t *testing.T) {
	// 测试合法查询
	validQueries := []string{
		"SELECT * FROM users",
		"PRAGMA table_info(users)",
		"PRAGMA index_list(users)",
		"PRAGMA database_list",
	}

	for _, q := range validQueries {
		if err := sqlite.ValidateSQLiteQuery(q); err != nil {
			t.Errorf("合法查询被拒绝: %s, 错误: %s", q, err)
		}
	}

	// 测试非法查询
	invalidQueries := []string{
		"INSERT INTO users VALUES (1)",
		"UPDATE users SET name='test'",
		"DELETE FROM users",
		"DROP TABLE users",
		"ALTER TABLE users ADD COLUMN x INT",
		"CREATE TABLE test (id INT)",
		"TRUNCATE TABLE users",
		"PRAGMA journal_mode = WAL", // 不安全的PRAGMA
	}

	for _, q := range invalidQueries {
		if err := sqlite.ValidateSQLiteQuery(q); err == nil {
			t.Errorf("非法查询未被拒绝: %s", q)
		}
	}
}