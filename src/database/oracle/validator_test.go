package oracle

import (
	"testing"
)

// TestValidateOracleQuery_Valid 测试合法的Oracle查询
func TestValidateOracleQuery_Valid(t *testing.T) {
	validQueries := []string{
		"SELECT * FROM users",
		"SELECT id, name FROM users WHERE id = 1",
		"SELECT COUNT(*) FROM orders",
		"WITH cte AS (SELECT 1 AS n) SELECT * FROM cte",
		"SELECT * FROM all_tables",
		"SELECT * FROM all_tab_columns",
		"SELECT * FROM user_tables",
		"SELECT * FROM user_objects",
		"SELECT * FROM dba_tables", // 需要权限
		"SELECT * FROM sys.all_objects",
		"SELECT * FROM users AS OF SCN 12345",
		"SELECT * FROM users AS OF TIMESTAMP '2023-01-01 12:00:00'",
		"SELECT * FROM users VERSIONS BETWEEN SCN 100 AND 200",
		"SELECT * FROM users VERSIONS BETWEEN TIMESTAMP '2023-01-01' AND '2023-01-02'",
		"DESCRIBE users",
		"DESC users",
	}

	for _, query := range validQueries {
		err := ValidateOracleQuery(query)
		if err != nil {
			t.Errorf("合法查询被拒绝: %s, 错误: %s", query, err)
		}
	}
}

// TestValidateOracleQuery_Invalid 测试非法的Oracle查询
func TestValidateOracleQuery_Invalid(t *testing.T) {
	invalidQueries := []string{
		"INSERT INTO users VALUES (1, 'test')",
		"INSERT INTO users SELECT * FROM other",
		"UPDATE users SET name = 'test'",
		"DELETE FROM users",
		"DROP TABLE users",
		"DROP INDEX idx_users",
		"DROP VIEW view_users",
		"DROP PROCEDURE proc_test",
		"DROP FUNCTION func_test",
		"DROP PACKAGE pkg_test",
		"DROP USER test",
		"DROP ROLE role_test",
		"ALTER TABLE users ADD COLUMN email VARCHAR2(100)",
		"ALTER INDEX idx_users REBUILD",
		"ALTER VIEW view_users COMPILE",
		"ALTER PROCEDURE proc_test COMPILE",
		"ALTER USER test IDENTIFIED BY 'newpass'",
		"ALTER DATABASE SET AUTOEXTEND ON",
		"ALTER SYSTEM SET parameter = value",
		"ALTER SESSION SET nls_date_format = 'YYYY-MM-DD'",
		"CREATE TABLE test (id NUMBER)",
		"CREATE INDEX idx_test ON test(id)",
		"CREATE VIEW view_test AS SELECT 1 FROM dual",
		"CREATE PROCEDURE proc_test AS BEGIN NULL; END;",
		"CREATE FUNCTION func_test RETURN NUMBER AS BEGIN RETURN 1; END;",
		"CREATE PACKAGE pkg_test AS PROCEDURE test; END;",
		"CREATE USER test IDENTIFIED BY 'password'",
		"CREATE ROLE role_test",
		"TRUNCATE TABLE users",
		"MERGE INTO users USING other ON (users.id = other.id) WHEN MATCHED THEN UPDATE SET users.name = other.name WHEN NOT MATCHED THEN INSERT VALUES (other.id, other.name)",
		"FLASHBACK DATABASE TO TIMESTAMP '2023-01-01 12:00:00'",
		"FLASHBACK DATABASE TO SCN 12345",
		"FLASHBACK TABLE users TO TIMESTAMP '2023-01-01 12:00:00'",
		"FLASHBACK TABLE users TO SCN 12345",
		"PURGE TABLE users",
		"PURGE INDEX idx_users",
		"PURGE RECYCLEBIN",
		"EXEC proc_test",
		"EXECUTE proc_test",
		"CALL proc_test",
		"GRANT SELECT ON users TO test",
		"REVOKE SELECT ON users FROM test",
		"AUDIT SELECT ON users",
		"NOAUDIT SELECT ON users",
		"ANALYZE TABLE users COMPUTE STATISTICS",
		"ANALYZE INDEX idx_users COMPUTE STATISTICS",
		"VALIDATE users",
		"SET CONSTRAINT ALL IMMEDIATE",
		"SET TRANSACTION ISOLATION LEVEL READ COMMITTED",
		"LOCK TABLE users IN EXCLUSIVE MODE",
		"COMMENT ON TABLE users IS 'test'",
		"RENAME users TO new_users",
		"SHUTDOWN IMMEDIATE",
		"STARTUP",
		"ARCHIVE LOG CURRENT",
		"RECOVER DATABASE",
		"BEGIN NULL; END;",
		"DECLARE v NUMBER; BEGIN v := 1; END;",
	}

	for _, query := range invalidQueries {
		err := ValidateOracleQuery(query)
		if err == nil {
			t.Errorf("非法查询未被拒绝: %s", query)
		}
	}
}

// TestValidateOracleQuery_EdgeCases 测试边界情况
func TestValidateOracleQuery_EdgeCases(t *testing.T) {
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
			name:    "FLASHBACK查询允许",
			query:   "SELECT * FROM users AS OF SCN 12345",
			wantErr: false,
		},
		{
			name:    "FLASHBACK恢复禁止",
			query:   "FLASHBACK TABLE users TO SCN 12345",
			wantErr: true,
		},
		{
			name:    "FLASHBACK DATABASE禁止",
			query:   "FLASHBACK DATABASE TO TIMESTAMP '2023-01-01'",
			wantErr: true,
		},
		{
			name:    "VERSIONS查询允许",
			query:   "SELECT * FROM users VERSIONS BETWEEN SCN 100 AND 200",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateOracleQuery(tt.query)
			if tt.wantErr && err == nil {
				t.Errorf("期望错误但查询被接受: %s", tt.query)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("期望成功但查询被拒绝: %s, 错误: %s", tt.query, err)
			}
		})
	}
}

// TestGetQueryType 测试查询类型获取
func TestGetQueryType(t *testing.T) {
	tests := []struct {
		query    string
		expected string
	}{
		{"SELECT * FROM users", "SELECT"},
		{"WITH cte AS (SELECT 1) SELECT * FROM cte", "WITH"},
		{"DESCRIBE users", "DESCRIBE"},
		{"DESC users", "DESCRIBE"},
		{"INSERT INTO users VALUES (1)", "UNKNOWN"},
		{"", "UNKNOWN"},
		{"   ", "UNKNOWN"},
	}

	for _, tt := range tests {
		result := GetQueryType(tt.query)
		if result != tt.expected {
			t.Errorf("GetQueryType(%s) = %s, expected %s", tt.query, result, tt.expected)
		}
	}
}

// TestIsReadOnlyQuery 测试只读查询判断
func TestIsReadOnlyQuery(t *testing.T) {
	readOnlyQueries := []string{
		"SELECT * FROM users",
		"WITH cte AS (SELECT 1) SELECT * FROM cte",
		"SELECT * FROM users AS OF SCN 12345",
		"DESCRIBE users",
	}

	for _, query := range readOnlyQueries {
		if !IsReadOnlyQuery(query) {
			t.Errorf("只读查询被判定为非只读: %s", query)
		}
	}

	writeQueries := []string{
		"INSERT INTO users VALUES (1)",
		"UPDATE users SET name = 'test'",
		"DELETE FROM users",
		"DROP TABLE users",
		"FLASHBACK TABLE users TO SCN 12345",
	}

	for _, query := range writeQueries {
		if IsReadOnlyQuery(query) {
			t.Errorf("写操作查询被判定为只读: %s", query)
		}
	}
}