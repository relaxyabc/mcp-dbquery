package sqlite

import (
	"testing"
)

// TestValidateSQLiteQuery_Valid 测试合法的SQLite查询
func TestValidateSQLiteQuery_Valid(t *testing.T) {
	validQueries := []string{
		"SELECT * FROM users",
		"SELECT id, name FROM users WHERE id = 1",
		"SELECT COUNT(*) FROM orders",
		"PRAGMA table_info(users)",
		"PRAGMA index_list(users)",
		"PRAGMA index_info(idx_users_email)",
		"PRAGMA database_list",
		"PRAGMA compile_options",
		"PRAGMA foreign_key_list(users)",
		"PRAGMA collation_list",
		"PRAGMA function_list",
		"PRAGMA module_list",
		"PRAGMA pragma_list",
	}

	for _, query := range validQueries {
		err := ValidateSQLiteQuery(query)
		if err != nil {
			t.Errorf("合法查询被拒绝: %s, 错误: %s", query, err)
		}
	}
}

// TestValidateSQLiteQuery_Invalid 测试非法的SQLite查询
func TestValidateSQLiteQuery_Invalid(t *testing.T) {
	invalidQueries := []string{
		"INSERT INTO users VALUES (1, 'test')",
		"UPDATE users SET name = 'test'",
		"DELETE FROM users",
		"DROP TABLE users",
		"ALTER TABLE users ADD COLUMN email TEXT",
		"CREATE TABLE test (id INT)",
		"CREATE INDEX idx_test ON test(id)",
		"TRUNCATE TABLE users",
		"REPLACE INTO users VALUES (1, 'test')",
		"ATTACH DATABASE 'test.db' AS test",
		"DETACH DATABASE test",
		"GRANT SELECT ON users TO test",
		"REVOKE SELECT ON users FROM test",
		"RENAME TABLE users TO new_users",
		"DROP INDEX idx_users",
		"INSERT INTO users SELECT * FROM other",
	}

	for _, query := range invalidQueries {
		err := ValidateSQLiteQuery(query)
		if err == nil {
			t.Errorf("非法查询未被拒绝: %s", query)
		}
	}
}

// TestValidateSQLiteQuery_EdgeCases 测试边界情况
func TestValidateSQLiteQuery_EdgeCases(t *testing.T) {
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
			name:    "不安全的PRAGMA",
			query:   "PRAGMA journal_mode = WAL",
			wantErr: true,
		},
		{
			name:    "PRAGMA设置值",
			query:   "PRAGMA synchronous = OFF",
			wantErr: true,
		},
		{
			name:    "PRAGMA writable_schema",
			query:   "PRAGMA writable_schema = ON",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSQLiteQuery(tt.query)
			if tt.wantErr && err == nil {
				t.Errorf("期望错误但查询被接受: %s", tt.query)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("期望成功但查询被拒绝: %s, 错误: %s", tt.query, err)
			}
		})
	}
}

// TestIsSafePragma 测试安全PRAGMA检测
func TestIsSafePragma(t *testing.T) {
	safePragmas := []string{
		"PRAGMA table_info(users)",
		"PRAGMA index_list(users)",
		"PRAGMA index_info(idx_email)",
		"PRAGMA database_list",
		"PRAGMA compile_options",
		"PRAGMA foreign_key_list(users)",
		"PRAGMA collation_list",
		"PRAGMA function_list",
		"PRAGMA module_list",
		"PRAGMA pragma_list",
	}

	for _, query := range safePragmas {
		if !isSafePragma(query) {
			t.Errorf("安全PRAGMA被判定为不安全: %s", query)
		}
	}

	unsafePragmas := []string{
		"PRAGMA journal_mode",
		"PRAGMA synchronous",
		"PRAGMA cache_size",
		"PRAGMA temp_store",
		"PRAGMA locking_mode",
		"PRAGMA foreign_keys",
		"PRAGMA writable_schema",
		"PRAGMA ignore_check_constraints",
		"PRAGMA auto_vacuum",
	}

	for _, query := range unsafePragmas {
		if isSafePragma(query) {
			t.Errorf("不安全PRAGMA被判定为安全: %s", query)
		}
	}
}

// TestGetQueryType 测试查询类型获取
func TestGetQueryType(t *testing.T) {
	tests := []struct {
		query    string
		expected string
	}{
		{"SELECT * FROM users", "SELECT"},
		{"PRAGMA table_info(users)", "PRAGMA"},
		// 写操作类型返回UNKNOWN（因为不是合法类型）
		{"INSERT INTO users VALUES (1)", "UNKNOWN"},
		{"UPDATE users SET name = 'test'", "UNKNOWN"},
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
		"PRAGMA table_info(users)",
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
	}

	for _, query := range writeQueries {
		if IsReadOnlyQuery(query) {
			t.Errorf("写操作查询被判定为只读: %s", query)
		}
	}
}