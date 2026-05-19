package integration

import (
	"testing"

	"github.com/relaxyabc/mcp-dbquery/src/database/postgres"
)

// TestPostgresValidator 测试PostgreSQL验证器
func TestPostgresValidator(t *testing.T) {
	// 测试合法查询
	validQueries := []string{
		"SELECT * FROM users",
		"SHOW TABLES",
		"DESCRIBE users",
		"EXPLAIN SELECT * FROM users",
		"WITH cte AS (SELECT 1) SELECT * FROM cte",
	}

	for _, q := range validQueries {
		if err := postgres.ValidatePostgresQuery(q); err != nil {
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
	}

	for _, q := range invalidQueries {
		if err := postgres.ValidatePostgresQuery(q); err == nil {
			t.Errorf("非法查询未被拒绝: %s", q)
		}
	}
}