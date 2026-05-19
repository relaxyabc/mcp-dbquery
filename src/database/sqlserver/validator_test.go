package sqlserver

import (
	"testing"
)

// TestValidateSQLServerQuery_Valid 测试合法的SQL Server查询
func TestValidateSQLServerQuery_Valid(t *testing.T) {
	validQueries := []string{
		"SELECT * FROM users",
		"SELECT TOP 10 * FROM users",
		"SELECT * FROM users ORDER BY id OFFSET 0 ROWS FETCH NEXT 10 ROWS ONLY",
		"SELECT id, name FROM users WHERE id = 1",
		"SELECT COUNT(*) FROM orders",
		"WITH cte AS (SELECT 1 AS n) SELECT * FROM cte",
		"SELECT * FROM sys.tables",
		"SELECT * FROM sys.columns",
		"SELECT * FROM sys.indexes",
		"SELECT * FROM INFORMATION_SCHEMA.TABLES",
		"SELECT * FROM INFORMATION_SCHEMA.COLUMNS",
		"EXEC sp_help users",
		"EXECUTE sp_help users",
		"EXEC sp_tables",
		"EXEC sp_columns users",
		"EXEC sp_stored_procedures",
		"EXEC sp_pkeys users",
		"EXEC sp_fkeys users",
		"EXEC sp_spaceused",
		"EXEC sp_who",
		"EXEC sp_who2",
		"EXEC sp_lock",
		"sp_help users", // 无EXEC前缀的存储过程调用
		"sp_tables",
	}

	for _, query := range validQueries {
		err := ValidateSQLServerQuery(query)
		if err != nil {
			t.Errorf("合法查询被拒绝: %s, 错误: %s", query, err)
		}
	}
}

// TestValidateSQLServerQuery_Invalid 测试非法的SQL Server查询
func TestValidateSQLServerQuery_Invalid(t *testing.T) {
	invalidQueries := []string{
		"INSERT INTO users VALUES (1, 'test')",
		"INSERT INTO users SELECT * FROM other",
		"UPDATE users SET name = 'test'",
		"DELETE FROM users",
		"DROP TABLE users",
		"DROP DATABASE test",
		"DROP INDEX idx_users",
		"DROP VIEW view_users",
		"DROP PROCEDURE proc_test",
		"DROP FUNCTION func_test",
		"ALTER TABLE users ADD COLUMN email VARCHAR(100)",
		"ALTER DATABASE test SET SINGLE_USER",
		"ALTER INDEX idx_users REBUILD",
		"CREATE TABLE test (id INT)",
		"CREATE DATABASE test",
		"CREATE INDEX idx_test ON test(id)",
		"CREATE VIEW view_test AS SELECT 1",
		"CREATE PROCEDURE proc_test AS SELECT 1",
		"CREATE FUNCTION func_test() RETURNS INT AS BEGIN RETURN 1 END",
		"TRUNCATE TABLE users",
		"BULK INSERT users FROM 'data.csv'",
		"MERGE INTO users USING other ON (users.id = other.id) WHEN MATCHED THEN UPDATE SET users.name = other.name",
		"BACKUP DATABASE test TO DISK = 'backup.bak'",
		"BACKUP LOG test TO DISK = 'log.bak'",
		"RESTORE DATABASE test FROM DISK = 'backup.bak'",
		"DBCC CHECKDB(test)",
		"DBCC SHRINKDATABASE(test)",
		"DBCC FREEPROCCACHE",
		"EXEC sp_adduser test",
		"EXEC sp_dropuser test",
		"EXEC sp_grantlogin test",
		"EXEC sp_revokelogin test",
		"EXEC sp_configure 'max connections', 100",
		"EXECUTE sp_adduser test",
		"KILL 53",
		"SHUTDOWN",
		"GRANT SELECT ON users TO test",
		"REVOKE SELECT ON users FROM test",
		"DENY SELECT ON users TO test",
		"LOAD DATABASE test",
		"DETACH DATABASE test",
		"ATTACH DATABASE test",
	}

	for _, query := range invalidQueries {
		err := ValidateSQLServerQuery(query)
		if err == nil {
			t.Errorf("非法查询未被拒绝: %s", query)
		}
	}
}

// TestValidateSQLServerQuery_EdgeCases 测试边界情况
func TestValidateSQLServerQuery_EdgeCases(t *testing.T) {
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
			name:    "不允许的存储过程",
			query:   "EXEC sp_executesql 'SELECT 1'",
			wantErr: true,
		},
		{
			name:    "无EXEC的不允许存储过程",
			query:   "sp_adduser test",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSQLServerQuery(tt.query)
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
		{"EXEC sp_help users", "EXEC"},
		{"EXECUTE sp_tables", "EXEC"},
		{"sp_help users", "EXEC"},
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
		"EXEC sp_help users",
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
		"EXEC sp_adduser test",
	}

	for _, query := range writeQueries {
		if IsReadOnlyQuery(query) {
			t.Errorf("写操作查询被判定为只读: %s", query)
		}
	}
}