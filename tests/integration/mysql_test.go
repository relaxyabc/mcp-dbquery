package integration

import (
	"context"
	"testing"
	"time"

	"github.com/relaxyabc/mcp-dbquery/src/database"
	"github.com/relaxyabc/mcp-dbquery/src/database/mysql"
	"github.com/relaxyabc/mcp-dbquery/src/utils"
)

// TestMySQLConnection 测试MySQL连接
func TestMySQLConnection(t *testing.T) {
	// 创建测试配置（使用环境变量或测试容器）
	config := database.DatabaseConfig{
		ID:       "test-mysql",
		Type:     database.DatabaseTypeMySQL,
		Host:     "localhost",
		Port:     3306,
		Username: "test_user",
		Password: "test_password",
		Database: "test_db",
		PoolSize: 5,
		Timeout:  30,
	}

	driver := mysql.NewMySQLDriver(config)

	// 创建上下文（用于实际连接测试）
	_, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 注意：实际测试需要真实MySQL容器
	// 这里只测试驱动初始化逻辑
	if driver.GetID() != "test-mysql" {
		t.Error("驱动ID设置错误")
	}

	if driver.GetType() != database.DatabaseTypeMySQL {
		t.Error("驱动类型设置错误")
	}

	utils.GlobalLogger.Info("MySQL驱动初始化测试通过")
}

// TestMySQLReadOnlyValidation 测试MySQL只读验证
func TestMySQLReadOnlyValidation(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		wantErr bool
	}{
		{"允许SELECT查询", "SELECT * FROM users", false},
		{"允许SHOW查询", "SHOW TABLES", false},
		{"允许DESCRIBE查询", "DESCRIBE users", false},
		{"允许EXPLAIN查询", "EXPLAIN SELECT * FROM users", false},
		{"禁止INSERT操作", "INSERT INTO users VALUES (1, 'test')", true},
		{"禁止UPDATE操作", "UPDATE users SET name = 'test'", true},
		{"禁止DELETE操作", "DELETE FROM users", true},
		{"禁止DROP操作", "DROP TABLE users", true},
		{"禁止ALTER操作", "ALTER TABLE users ADD COLUMN age INT", true},
		{"禁止CREATE操作", "CREATE TABLE test (id INT)", true},
		{"禁止TRUNCATE操作", "TRUNCATE TABLE users", true},
		{"禁止多语句查询", "SELECT * FROM users; INSERT INTO users VALUES (1);", true},
		{"允许带分号的SELECT", "SELECT * FROM users;", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mysql.ValidateMySQLQuery(tt.query)

			if tt.wantErr && err == nil {
				t.Errorf("期望错误但未返回错误: %s", tt.query)
			}

			if !tt.wantErr && err != nil {
				t.Errorf("期望成功但返回错误: %s, 错误: %v", tt.query, err)
			}

			utils.GlobalLogger.Info("测试案例通过 [查询=%s] [期望错误=%v] [实际错误=%v]",
				tt.name, tt.wantErr, err != nil)
		})
	}
}

// TestMySQLQueryType 测试MySQL查询类型识别
func TestMySQLQueryType(t *testing.T) {
	tests := []struct {
		query    string
		expected string
	}{
		{"SELECT * FROM users", "SELECT"},
		{"SHOW TABLES", "SHOW"},
		{"DESCRIBE users", "DESCRIBE"},
		{"EXPLAIN SELECT * FROM users", "EXPLAIN"},
		{"INSERT INTO users VALUES (1)", "UNKNOWN"},
	}

	for _, tt := range tests {
		queryType := mysql.GetQueryType(tt.query)
		if queryType != tt.expected {
			t.Errorf("查询类型识别错误: 查询=%s, 期望=%s, 实际=%s",
				tt.query, tt.expected, queryType)
		}
	}
}