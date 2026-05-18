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

// TestMySQLSchemaExtraction 测试MySQL结构提取（需要真实数据库）
func TestMySQLSchemaExtraction(t *testing.T) {
	// TODO: 使用testcontainers启动MySQL容器进行实际测试
	// 这里提供测试框架

	t.Skip("需要真实MySQL容器")

	// 测试框架代码：
	/*
		config := createTestMySQLConfig()
		driver := mysql.NewMySQLDriver(config)

		ctx := context.Background()
		if err := driver.Connect(ctx); err != nil {
			t.Fatalf("连接失败: %v", err)
		}

		// 测试DescribeTable
		schema, err := driver.DescribeTable(ctx, "test_table")
		if err != nil {
			t.Errorf("结构提取失败: %v", err)
		}

		if schema.TableName != "test_table" {
			t.Error("表名设置错误")
		}

		if len(schema.Fields) == 0 {
			t.Error("字段列表为空")
		}
	*/
}

// TestMySQLIndexExtraction 测试MySQL索引提取（需要真实数据库）
func TestMySQLIndexExtraction(t *testing.T) {
	t.Skip("需要真实MySQL容器")

	// 测试框架代码：
	/*
		config := createTestMySQLConfig()
		driver := mysql.NewMySQLDriver(config)

		ctx := context.Background()
		driver.Connect(ctx)

		indexes, err := driver.ShowIndexes(ctx, "test_table")
		if err != nil {
			t.Errorf("索引提取失败: %v", err)
		}

		if len(indexes.Indexes) == 0 {
			t.Error("索引列表为空")
		}

		// 检查主键索引
		primaryIndex, err := driver.GetPrimaryKey(ctx, "test_table")
		if err != nil {
			t.Error("主键索引不存在")
		}

		if primaryIndex.IndexName != "PRIMARY" {
			t.Error("主键索引名称错误")
		}
	*/
}

// TestMySQLTableListing 测试MySQL表列表（需要真实数据库）
func TestMySQLTableListing(t *testing.T) {
	t.Skip("需要真实MySQL容器")

	// 测试框架代码：
	/*
		config := createTestMySQLConfig()
		driver := mysql.NewMySQLDriver(config)

		ctx := context.Background()
		driver.Connect(ctx)

		tables, err := driver.ShowTables(ctx)
		if err != nil {
			t.Errorf("表列表获取失败: %v", err)
		}

		if len(tables) == 0 {
			t.Error("表列表为空")
		}
	*/
}

// 辅助函数：创建测试MySQL配置
func createTestMySQLConfig() database.DatabaseConfig {
	return database.DatabaseConfig{
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
}
