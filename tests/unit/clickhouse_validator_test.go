package validator_test

import (
	"testing"

	"github.com/relaxyabc/mcp-dbquery/src/database/clickhouse"
)

// TestValidateClickHouseQuery_Valid 测试合法的ClickHouse查询
func TestValidateClickHouseQuery_Valid(t *testing.T) {
	validQueries := []string{
		"SELECT * FROM users",
		"SELECT id, name FROM users WHERE id = 1",
		"SELECT COUNT(*) FROM orders",
		"SHOW DATABASES",
		"SHOW TABLES FROM default",
		"SHOW CREATE TABLE users",
		"DESCRIBE TABLE users",
		"DESC users",
		"EXPLAIN SELECT * FROM users",
		"WITH cte AS (SELECT 1) SELECT * FROM cte",
		"SELECT * FROM system.tables",
		"SELECT * FROM system.columns",
		"SELECT * FROM system.databases",
		"SELECT * FROM system.parts",
		"SELECT * FROM system.metrics",
		"SELECT * FROM system.events",
		"SELECT * FROM system.functions",
		"SET max_rows = 1000",
		"SET max_bytes = 1000000",
		"SET max_execution_time = 60",
		"SET readonly = 1",
	}

	for _, query := range validQueries {
		err := clickhouse.ValidateClickHouseQuery(query)
		if err != nil {
			t.Errorf("合法查询被拒绝: %s, 错误: %s", query, err)
		}
	}
}

// TestValidateClickHouseQuery_Invalid 测试非法的ClickHouse查询
func TestValidateClickHouseQuery_Invalid(t *testing.T) {
	invalidQueries := []string{
		"INSERT INTO users VALUES (1, 'test')",
		"INSERT INTO users SELECT * FROM other",
		"UPDATE users SET name = 'test'",
		"DELETE FROM users",
		"DROP TABLE users",
		"DROP DATABASE test",
		"DROP DICTIONARY dict",
		"DROP VIEW view",
		"ALTER TABLE users ADD COLUMN email TEXT",
		"ALTER DATABASE test SET",
		"ALTER USER test SET",
		"CREATE TABLE test (id Int32)",
		"CREATE DATABASE test",
		"CREATE DICTIONARY dict",
		"CREATE VIEW view AS SELECT 1",
		"CREATE USER test",
		"CREATE FUNCTION test AS",
		"TRUNCATE TABLE users",
		"OPTIMIZE TABLE users",
		"SYSTEM STOP MERGES",
		"SYSTEM START MERGES",
		"SYSTEM FLUSH LOGS",
		"SYSTEM RELOAD DICTIONARY",
		"KILL QUERY WHERE query_id = 'test'",
		"KILL TRANSACTION WHERE id = 1",
		"GRANT SELECT ON *.* TO test",
		"REVOKE SELECT ON *.* FROM test",
		"ATTACH PARTITION FROM other",
		"DETACH PARTITION partition",
		"RENAME TABLE users TO new_users",
		"SET ROLE admin",
		"SET PASSWORD = 'test'",
		"SET DEFAULT_ROLE admin TO test",
	}

	for _, query := range invalidQueries {
		err := clickhouse.ValidateClickHouseQuery(query)
		if err == nil {
			t.Errorf("非法查询未被拒绝: %s", query)
		}
	}
}

// TestValidateClickHouseQuery_EdgeCases 测试边界情况
func TestValidateClickHouseQuery_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		wantErr bool
	}{
		{
			name:    "空查询",
			query:   "",
			wantErr: true,
		},
		{
			name:    "只有空白",
			query:   "   ",
			wantErr: true,
		},
		{
			name:    "注释后空查询",
			query:   "-- 这是一个注释",
			wantErr: true,
		},
		{
			name:    "多行注释",
			query:   "/* 多行注释 */ SELECT * FROM users",
			wantErr: false,
		},
		{
			name:    "多语句查询",
			query:   "SELECT * FROM users; SELECT * FROM orders;",
			wantErr: true,
		},
		{
			name:    "单语句带结尾分号",
			query:   "SELECT * FROM users;",
			wantErr: false,
		},
		{
			name:    "子查询中的禁止操作",
			query:   "SELECT * FROM users WHERE id IN (INSERT INTO other VALUES (1))",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := clickhouse.ValidateClickHouseQuery(tt.query)
			if tt.wantErr && err == nil {
				t.Errorf("期望错误但查询被接受: %s", tt.query)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("期望成功但查询被拒绝: %s, 错误: %s", tt.query, err)
			}
		})
	}
}

// TestGetQueryType_ClickHouse 测试查询类型获取
func TestGetQueryType_ClickHouse(t *testing.T) {
	tests := []struct {
		query    string
		expected string
	}{
		{"SELECT * FROM users", "SELECT"},
		{"SHOW DATABASES", "SHOW"},
		{"DESCRIBE TABLE users", "DESCRIBE"},
		{"DESC users", "DESCRIBE"},
		{"EXPLAIN SELECT * FROM users", "EXPLAIN"},
		{"WITH cte AS (SELECT 1) SELECT * FROM cte", "WITH"},
		{"INSERT INTO users VALUES (1)", "UNKNOWN"},
		{"", "UNKNOWN"},
		{"   ", "UNKNOWN"},
	}

	for _, tt := range tests {
		result := clickhouse.GetQueryType(tt.query)
		if result != tt.expected {
			t.Errorf("GetQueryType(%s) = %s, expected %s", tt.query, result, tt.expected)
		}
	}
}

// TestIsReadOnlyQuery_ClickHouse 测试只读查询判断
func TestIsReadOnlyQuery_ClickHouse(t *testing.T) {
	readOnlyQueries := []string{
		"SELECT * FROM users",
		"SHOW DATABASES",
		"DESCRIBE TABLE users",
	}

	for _, query := range readOnlyQueries {
		if !clickhouse.IsReadOnlyQuery(query) {
			t.Errorf("只读查询被判定为非只读: %s", query)
		}
	}

	writeQueries := []string{
		"INSERT INTO users VALUES (1)",
		"UPDATE users SET name = 'test'",
		"DELETE FROM users",
		"DROP TABLE users",
	}

	for _, query := range writeQueries {
		if clickhouse.IsReadOnlyQuery(query) {
			t.Errorf("写操作查询被判定为只读: %s", query)
		}
	}
}