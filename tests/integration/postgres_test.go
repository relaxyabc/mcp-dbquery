package integration

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/relaxyabc/mcp-dbquery/src/database"
	"github.com/relaxyabc/mcp-dbquery/src/database/postgres"
	"github.com/relaxyabc/mcp-dbquery/tests/fixtures"
)

var pgTestContainers *fixtures.TestContainers
var pgTestConfig database.DatabaseConfig

// setupPostgresTest 设置PostgreSQL测试环境
func setupPostgresTest(t *testing.T) {
	if pgTestContainers != nil {
		return // 已初始化
	}

	ctx := context.Background()
	pgTestContainers = fixtures.NewTestContainers()

	// 启动PostgreSQL测试容器
	if err := pgTestContainers.StartPostgreSQL(ctx); err != nil {
		t.Logf("[警告] PostgreSQL容器启动失败: %s, 测试将跳过", err)
		return
	}

	// 等待容器就绪
	if err := pgTestContainers.WaitForReady(ctx, 10*time.Second); err != nil {
		t.Logf("[警告] PostgreSQL容器未就绪: %s", err)
		pgTestContainers.StopAll(ctx)
		pgTestContainers = nil
		return
	}

	// 设置测试配置
	host, port := pgTestContainers.GetPostgreSQLConnection()
	pgTestConfig = database.DatabaseConfig{
		ID:       "test-pg",
		Type:     database.DatabaseTypePostgreSQL,
		Host:     host,
		Port:     port,
		Username: "postgres",
		Password: "testpassword",
		Database: "testdb",
		PoolSize: 5,
		Timeout:  30,
	}
}

// skipIfPostgresNotReady 跳过测试如果PostgreSQL未就绪
func skipIfPostgresNotReady(t *testing.T) {
	if pgTestContainers == nil || !pgTestContainers.IsRunning() {
		t.Skip("PostgreSQL测试容器未启动")
	}
}

// TestPostgresConnection 测试PostgreSQL连接
func TestPostgresConnection(t *testing.T) {
	setupPostgresTest(t)
	skipIfPostgresNotReady(t)

	ctx := context.Background()
	driver := postgres.NewPostgresDriver(pgTestConfig)

	// 测试连接
	if err := driver.Connect(ctx); err != nil {
		t.Fatalf("PostgreSQL连接失败: %s", err)
	}

	// 验证连接状态
	if !driver.IsConnected() {
		t.Fatal("连接状态应为connected")
	}

	// 关闭连接
	if err := driver.Close(ctx); err != nil {
		t.Fatalf("关闭连接失败: %s", err)
	}
}

// TestPostgresSelectQuery 测试PostgreSQL SELECT查询
func TestPostgresSelectQuery(t *testing.T) {
	setupPostgresTest(t)
	skipIfPostgresNotReady(t)

	ctx := context.Background()
	driver := postgres.NewPostgresDriver(pgTestConfig)

	if err := driver.Connect(ctx); err != nil {
		t.Fatalf("连接失败: %s", err)
	}
	defer driver.Close(ctx)

	// 执行SELECT查询
	result, err := driver.ExecuteQuery(ctx, "SELECT 1 as id, 'test' as name", 1000, 30*time.Second)
	if err != nil {
		t.Fatalf("查询失败: %s", err)
	}

	// 验证结果
	if result.RowCount != 1 {
		t.Errorf("期望返回1行，实际返回%d行", result.RowCount)
	}

	if len(result.Data) != 1 {
		t.Errorf("期望数据长度为1，实际为%d", len(result.Data))
	}

	row := result.Data[0]
	if row["id"] != int64(1) {
		t.Errorf("期望id=1，实际为%v", row["id"])
	}

	if row["name"] != "test" {
		t.Errorf("期望name='test'，实际为%v", row["name"])
	}
}

// TestPostgresSchema 测试PostgreSQL Schema获取
func TestPostgresSchema(t *testing.T) {
	setupPostgresTest(t)
	skipIfPostgresNotReady(t)

	ctx := context.Background()
	driver := postgres.NewPostgresDriver(pgTestConfig)

	if err := driver.Connect(ctx); err != nil {
		t.Fatalf("连接失败: %s", err)
	}
	defer driver.Close(ctx)

	// 创建测试表
	_, err := driver.Pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS test_schema_table (
			id SERIAL PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			description TEXT
		)
	`)
	if err != nil {
		t.Fatalf("创建测试表失败: %s", err)
	}

	// 获取Schema
	schema, err := driver.GetSchema(ctx, "test_schema_table")
	if err != nil {
		t.Fatalf("获取Schema失败: %s", err)
	}

	// 验证Schema
	if schema.TableName != "test_schema_table" {
		t.Errorf("期望表名test_schema_table，实际为%s", schema.TableName)
	}

	if len(schema.Fields) != 3 {
		t.Errorf("期望3个字段，实际为%d", len(schema.Fields))
	}

	// 验证字段
	foundPK := false
	for _, field := range schema.Fields {
		if field.Name == "id" && field.PrimaryKey {
			foundPK = true
		}
	}
	if !foundPK {
		t.Error("未找到主键字段id")
	}
}

// TestPostgresListTables 测试PostgreSQL表列表
func TestPostgresListTables(t *testing.T) {
	setupPostgresTest(t)
	skipIfPostgresNotReady(t)

	ctx := context.Background()
	driver := postgres.NewPostgresDriver(pgTestConfig)

	if err := driver.Connect(ctx); err != nil {
		t.Fatalf("连接失败: %s", err)
	}
	defer driver.Close(ctx)

	// 获取表列表
	tables, err := driver.ListTables(ctx)
	if err != nil {
		t.Fatalf("获取表列表失败: %s", err)
	}

	// 验证结果
	fmt.Printf("PostgreSQL表列表: %v\n", tables)
}

// TestPostgresIndex 测试PostgreSQL索引获取
func TestPostgresIndex(t *testing.T) {
	setupPostgresTest(t)
	skipIfPostgresNotReady(t)

	ctx := context.Background()
	driver := postgres.NewPostgresDriver(pgTestConfig)

	if err := driver.Connect(ctx); err != nil {
		t.Fatalf("连接失败: %s", err)
	}
	defer driver.Close(ctx)

	// 创建测试表和索引
	_, err := driver.Pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS test_index_table (
			id SERIAL PRIMARY KEY,
			email VARCHAR(100)
		);
		CREATE INDEX IF NOT EXISTS idx_test_email ON test_index_table(email)
	`)
	if err != nil {
		t.Fatalf("创建测试表和索引失败: %s", err)
	}

	// 获取索引
	indexes, err := driver.GetIndexes(ctx, "test_index_table")
	if err != nil {
		t.Fatalf("获取索引失败: %s", err)
	}

	// 验证索引
	if len(indexes.Indexes) < 1 {
		t.Errorf("期望至少1个索引，实际为%d", len(indexes.Indexes))
	}

	// 查找主键索引
	foundPK := false
	for _, idx := range indexes.Indexes {
		if idx.Unique && strings.Contains(idx.IndexName, "pkey") {
			foundPK = true
		}
	}
	if !foundPK {
		t.Error("未找到主键索引")
	}
}

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

// TestPostgresCleanup 清理PostgreSQL测试环境
func TestPostgresCleanup(t *testing.T) {
	if pgTestContainers != nil {
		ctx := context.Background()
		pgTestContainers.StopAll(ctx)
		pgTestContainers = nil
	}
}