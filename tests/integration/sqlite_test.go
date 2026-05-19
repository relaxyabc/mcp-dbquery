package integration

import (
	"context"
	"testing"
	"time"

	"github.com/relaxyabc/mcp-dbquery/src/database"
	"github.com/relaxyabc/mcp-dbquery/src/database/sqlite"
	"github.com/relaxyabc/mcp-dbquery/tests/fixtures"
)

var sqliteTestContainers *fixtures.TestContainers
var sqliteTestConfig database.DatabaseConfig

// setupSQLiteTest 设置SQLite测试环境
func setupSQLiteTest(t *testing.T) {
	if sqliteTestContainers != nil {
		return // 已初始化
	}

	ctx := context.Background()
	sqliteTestContainers = fixtures.NewTestContainers()

	// 创建SQLite临时文件
	if err := sqliteTestContainers.CreateSQLiteTempFile(ctx); err != nil {
		t.Logf("[警告] SQLite临时文件创建失败: %s, 测试将跳过", err)
		return
	}

	// 设置测试配置
	sqliteTestConfig = database.DatabaseConfig{
		ID:       "test-sqlite",
		Type:     database.DatabaseTypeSQLite,
		Path:     sqliteTestContainers.GetSQLitePath(),
		PoolSize: 5,
		Timeout:  30,
	}
}

// skipIfSQLiteNotReady 跳过测试如果SQLite未就绪
func skipIfSQLiteNotReady(t *testing.T) {
	if sqliteTestContainers == nil || sqliteTestContainers.GetSQLitePath() == "" {
		t.Skip("SQLite测试环境未就绪")
	}
}

// TestSQLiteConnection 测试SQLite连接
func TestSQLiteConnection(t *testing.T) {
	setupSQLiteTest(t)
	skipIfSQLiteNotReady(t)

	ctx := context.Background()
	driver := sqlite.NewSQLiteDriver(sqliteTestConfig)

	// 测试连接
	if err := driver.Connect(ctx); err != nil {
		t.Fatalf("SQLite连接失败: %s", err)
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

// TestSQLiteSelectQuery 测试SQLite SELECT查询
func TestSQLiteSelectQuery(t *testing.T) {
	setupSQLiteTest(t)
	skipIfSQLiteNotReady(t)

	ctx := context.Background()
	driver := sqlite.NewSQLiteDriver(sqliteTestConfig)

	if err := driver.Connect(ctx); err != nil {
		t.Fatalf("连接失败: %s", err)
	}
	defer driver.Close(ctx)

	// 创建测试表并插入数据（使用原生SQL，绕过验证器）
	// 注意：SQLite在只读模式下无法创建表，需要先创建
	// 这里我们先断开只读模式创建表，然后重新连接只读模式测试查询

	// 对于测试，我们使用另一个可写连接创建测试数据
	// 然后用只读驱动测试查询

	// 执行SELECT查询（SQLite在没有表时会返回空结果）
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

// TestSQLiteCleanup 清理SQLite测试环境
func TestSQLiteCleanup(t *testing.T) {
	if sqliteTestContainers != nil {
		ctx := context.Background()
		sqliteTestContainers.CleanupSQLiteTempFile(ctx)
		sqliteTestContainers = nil
	}
}