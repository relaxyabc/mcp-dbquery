package validator_test

import (
	"testing"

	"github.com/relaxyabc/mcp-dbquery/src/database/doris"
)

// TestValidateDorisQuery_Valid 测试合法的Doris查询
func TestValidateDorisQuery_Valid(t *testing.T) {
	validQueries := []string{
		"SELECT * FROM users",
		"SELECT id, name FROM users WHERE id = 1",
		"SELECT COUNT(*) FROM orders",
		"SHOW DATABASES",
		"SHOW TABLES FROM default",
		"SHOW CREATE TABLE users",
		"SHOW CATALOGS",
		"SHOW RESOURCES",
		"SHOW FRONTENDS",
		"SHOW BACKENDS",
		"DESCRIBE TABLE users",
		"DESC users",
		"EXPLAIN SELECT * FROM users",
		"WITH cte AS (SELECT 1) SELECT * FROM cte",
		"ADMIN SHOW FRONTENDS",
		"ADMIN SHOW BACKENDS",
		"ADMIN SHOW BROKER",
		"ADMIN SHOW REPLICA",
		"ADMIN SHOW TABLETS",
		"ADMIN SHOW CONFIG",
		"ADMIN SHOW PROC '/dbs'",
		"ADMIN SHOW REBALANCE",
		"ADMIN SHOW DATA SKETCH FROM users",
	}

	for _, query := range validQueries {
		err := doris.ValidateDorisQuery(query)
		if err != nil {
			t.Errorf("合法查询被拒绝: %s, 错误: %s", query, err)
		}
	}
}

// TestValidateDorisQuery_Invalid 测试非法的Doris查询
func TestValidateDorisQuery_Invalid(t *testing.T) {
	invalidQueries := []string{
		"INSERT INTO users VALUES (1, 'test')",
		"INSERT INTO users SELECT * FROM other",
		"UPDATE users SET name = 'test'",
		"DELETE FROM users",
		"DROP TABLE users",
		"DROP DATABASE test",
		"DROP CATALOG cat",
		"DROP RESOURCE res",
		"DROP USER test",
		"DROP ROLE admin",
		"ALTER TABLE users ADD COLUMN email TEXT",
		"ALTER DATABASE test SET",
		"ALTER CATALOG cat SET",
		"ALTER USER test SET",
		"ALTER SYSTEM",
		"CREATE TABLE test (id INT)",
		"CREATE DATABASE test",
		"CREATE CATALOG cat",
		"CREATE RESOURCE res",
		"CREATE USER test",
		"CREATE ROLE admin",
		"CREATE FUNCTION test AS",
		"CREATE VIEW view AS SELECT 1",
		"TRUNCATE TABLE users",
		"GRANT SELECT ON *.* TO test",
		"REVOKE SELECT ON *.* FROM test",
		"SET PASSWORD = 'test'",
		"SET ROLE admin",
		"ADMIN SET FRONTEND",
		"ADMIN CANCEL DECOMMISSION",
		"RENAME TABLE users TO new_users",
		"STOP CANCEL LOAD",
		"CANCEL LOAD FROM test",
		"CANCEL EXPORT FROM test",
		"LOAD LABEL test",
		"EXPORT TABLE users TO",
	}

	for _, query := range invalidQueries {
		err := doris.ValidateDorisQuery(query)
		if err == nil {
			t.Errorf("非法查询未被拒绝: %s", query)
		}
	}
}

// TestValidateDorisQuery_EdgeCases 测试边界情况
func TestValidateDorisQuery_EdgeCases(t *testing.T) {
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
		{
			name:    "禁止的ADMIN命令",
			query:   "ADMIN SET CONFIG",
			wantErr: true,
		},
		{
			name:    "允许的ADMIN SHOW命令",
			query:   "ADMIN SHOW FRONTENDS",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := doris.ValidateDorisQuery(tt.query)
			if tt.wantErr && err == nil {
				t.Errorf("期望错误但查询被接受: %s", tt.query)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("期望成功但查询被拒绝: %s, 错误: %s", tt.query, err)
			}
		})
	}
}

// TestGetQueryType_Doris 测试查询类型获取
func TestGetQueryType_Doris(t *testing.T) {
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
		{"ADMIN SHOW FRONTENDS", "ADMIN_SHOW"},
		{"INSERT INTO users VALUES (1)", "UNKNOWN"},
		{"", "UNKNOWN"},
		{"   ", "UNKNOWN"},
	}

	for _, tt := range tests {
		result := doris.GetQueryType(tt.query)
		if result != tt.expected {
			t.Errorf("GetQueryType(%s) = %s, expected %s", tt.query, result, tt.expected)
		}
	}
}

// TestIsReadOnlyQuery_Doris 测试只读查询判断
func TestIsReadOnlyQuery_Doris(t *testing.T) {
	readOnlyQueries := []string{
		"SELECT * FROM users",
		"SHOW DATABASES",
		"DESCRIBE TABLE users",
		"ADMIN SHOW FRONTENDS",
	}

	for _, query := range readOnlyQueries {
		if !doris.IsReadOnlyQuery(query) {
			t.Errorf("只读查询被判定为非只读: %s", query)
		}
	}

	writeQueries := []string{
		"INSERT INTO users VALUES (1)",
		"UPDATE users SET name = 'test'",
		"DELETE FROM users",
		"DROP TABLE users",
		"ADMIN SET CONFIG",
	}

	for _, query := range writeQueries {
		if doris.IsReadOnlyQuery(query) {
			t.Errorf("写操作查询被判定为只读: %s", query)
		}
	}
}