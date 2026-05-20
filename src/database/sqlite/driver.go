package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3" // SQLite驱动

	"github.com/relaxyabc/mcp-dbquery/src/database"
	"github.com/relaxyabc/mcp-dbquery/src/utils"
)

// init() 自动注册 SQLite 驱动到全局注册表
func init() {
	database.RegisterDriver(database.DatabaseTypeSQLite, func(config database.DatabaseConfig) database.Database {
		return NewSQLiteDriver(config)
	})
}

// SQLiteDriver SQLite数据库驱动实现
type SQLiteDriver struct {
	ID     string
	Config database.DatabaseConfig
	DB     *sql.DB
	State  database.ConnectionState
}

// NewSQLiteDriver 创建SQLite驱动实例
func NewSQLiteDriver(config database.DatabaseConfig) *SQLiteDriver {
	return &SQLiteDriver{
		ID:     config.ID,
		Config: config,
		State:  database.StateDisconnected,
	}
}

// Connect 建立SQLite连接（只读模式）
func (d *SQLiteDriver) Connect(ctx context.Context) error {
	d.State = database.StateConnecting
	utils.GlobalLogger.LogConnection(d.ID, d.GetMaskedConnectionString(), "connecting")

	// 构建连接字符串 - 强制只读模式
	connStr := d.buildConnectionString()

	// 打开数据库连接
	db, err := sql.Open("sqlite3", connStr)
	if err != nil {
		d.State = database.StateError
		utils.GlobalLogger.LogError("CONNECTION_ERROR", "SQLite连接打开失败", err.Error())
		return fmt.Errorf("SQLite连接打开失败: %s", err)
	}

	// 配置连接池
	db.SetMaxOpenConns(d.Config.PoolSize)
	db.SetMaxIdleConns(d.Config.PoolSize / 2)
	db.SetConnMaxLifetime(time.Duration(d.Config.Timeout) * time.Second)

	// 测试连接
	if err := db.PingContext(ctx); err != nil {
		d.State = database.StateError
		utils.GlobalLogger.LogError("CONNECTION_ERROR", "SQLite连接测试失败", err.Error())
		return fmt.Errorf("SQLite连接测试失败: %s", err)
	}

	d.DB = db
	d.State = database.StateConnected
	utils.GlobalLogger.LogConnection(d.ID, d.GetMaskedConnectionString(), "connected")

	return nil
}

// Close 关闭SQLite连接
func (d *SQLiteDriver) Close(ctx context.Context) error {
	if d.DB == nil {
		return nil
	}

	d.State = database.StateClosed
	utils.GlobalLogger.LogConnection(d.ID, d.GetMaskedConnectionString(), "closing")

	if err := d.DB.Close(); err != nil {
		utils.GlobalLogger.LogError("CLOSE_ERROR", "SQLite连接关闭失败", err.Error())
		return err
	}

	d.DB = nil
	utils.GlobalLogger.LogConnection(d.ID, d.GetMaskedConnectionString(), "closed")
	return nil
}

// IsConnected 检查连接状态
func (d *SQLiteDriver) IsConnected() bool {
	if d.DB == nil {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	return d.DB.PingContext(ctx) == nil
}

// GetType 返回数据库类型
func (d *SQLiteDriver) GetType() database.DatabaseType {
	return database.DatabaseTypeSQLite
}

// GetID 返回连接标识符
func (d *SQLiteDriver) GetID() string {
	return d.ID
}

// ExecuteQuery 执行只读查询
func (d *SQLiteDriver) ExecuteQuery(ctx context.Context, query string, limit int, timeout time.Duration) (*database.QueryResult, error) {
	start := time.Now()

	// 验证查询是否为只读
	if err := d.ValidateQuery(query); err != nil {
		return nil, err
	}

	// 创建超时上下文
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// 执行查询
	rows, err := d.DB.QueryContext(ctx, query)
	if err != nil {
		return database.NewErrorResult(d.ID, "QUERY_ERROR", fmt.Sprintf("查询执行失败: %s", err)), err
	}
	defer rows.Close()

	// 获取列信息
	columns, err := rows.Columns()
	if err != nil {
		return database.NewErrorResult(d.ID, "QUERY_ERROR", fmt.Sprintf("获取列信息失败: %s", err)), err
	}

	// 读取数据
	data := []map[string]interface{}{}
	actualCount := 0

	for rows.Next() {
		// 创建值数组
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		// 读取行数据
		if err := rows.Scan(valuePtrs...); err != nil {
			utils.GlobalLogger.Warn("行数据读取失败: %s", err)
			continue
		}

		// 转换为map
		rowMap := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			// 处理NULL值和字节切片
			if b, ok := val.([]byte); ok {
				rowMap[col] = string(b)
			} else if val == nil {
				rowMap[col] = nil
			} else {
				rowMap[col] = val
			}
		}

		// 添加到结果（不超过限制）
		if len(data) < limit {
			data = append(data, rowMap)
		}
		actualCount++

		// 达到限制后停止读取
		if actualCount >= limit {
			break
		}
	}

	// 构建结果
	executionTime := database.MeasureExecutionTime(start)
	resultType := database.QueryTypeData

	if actualCount > limit {
		return database.NewTruncatedResult(d.ID, data, resultType, executionTime, actualCount), nil
	}

	return database.NewQueryResult(d.ID, data, resultType, executionTime), nil
}

// ValidateQuery 验证查询是否为只读
func (d *SQLiteDriver) ValidateQuery(query string) error {
	return ValidateSQLiteQuery(query)
}

// buildConnectionString 构建SQLite连接字符串（强制只读）
func (d *SQLiteDriver) buildConnectionString() string {
	// SQLite连接字符串格式: file:path?mode=ro
	// 强制只读模式确保安全
	path := d.Config.Path
	if path == "" {
		path = d.Config.Database // 兼容旧配置
	}

	// 添加只读模式和其他安全参数
	return fmt.Sprintf("file:%s?mode=ro&_busy_timeout=5000&_txlock=immediate", path)
}

// GetMaskedConnectionString 获取遮蔽路径的连接字符串（日志使用）
func (d *SQLiteDriver) GetMaskedConnectionString() string {
	path := d.Config.Path
	if path == "" {
		path = d.Config.Database
	}
	return fmt.Sprintf("file:%s?mode=ro", path)
}

// GetSchema 获取表结构元数据
func (d *SQLiteDriver) GetSchema(ctx context.Context, tableName string) (*database.SchemaMetadata, error) {
	return d.DescribeTable(ctx, tableName)
}

// GetIndexes 获取索引元数据
func (d *SQLiteDriver) GetIndexes(ctx context.Context, tableName string) (*database.IndexListMetadata, error) {
	return d.ShowIndexes(ctx, tableName)
}

// ListTables 列出所有表
func (d *SQLiteDriver) ListTables(ctx context.Context) ([]string, error) {
	return d.ShowTables(ctx)
}

// ExecuteSelectQuery 执行SELECT查询并返回结果
func (d *SQLiteDriver) ExecuteSelectQuery(ctx context.Context, query string, limit int) (*database.QueryResult, error) {
	return d.ExecuteQuery(ctx, query, limit, time.Duration(d.Config.Timeout)*time.Second)
}

// ExecuteFind MongoDB find查询（SQLite不支持，返回错误）
func (d *SQLiteDriver) ExecuteFind(ctx context.Context, collection string, filter map[string]interface{}, limit int) (*database.QueryResult, error) {
	return nil, fmt.Errorf("SQLite 不支持 MongoDB find 查询，请使用 ExecuteSelectQuery 执行 SQL 查询")
}

// ExecuteAggregate MongoDB聚合查询（SQLite不支持，返回错误）
func (d *SQLiteDriver) ExecuteAggregate(ctx context.Context, collection string, pipeline []map[string]interface{}, limit int) (*database.QueryResult, error) {
	return nil, fmt.Errorf("SQLite 不支持 MongoDB aggregate 查询，请使用 ExecuteSelectQuery 执行 SQL 查询")
}

// ExecuteCount MongoDB计数查询（SQLite不支持，返回错误）
func (d *SQLiteDriver) ExecuteCount(ctx context.Context, collection string, filter map[string]interface{}) (int64, error) {
	return 0, fmt.Errorf("SQLite 不支持 MongoDB count 查询，请使用 SELECT COUNT(*) 执行 SQL 计数")
}

// ExecuteDistinct MongoDB distinct查询（SQLite不支持，返回错误）
func (d *SQLiteDriver) ExecuteDistinct(ctx context.Context, collection string, fieldName string, filter map[string]interface{}) ([]interface{}, error) {
	return nil, fmt.Errorf("SQLite 不支持 MongoDB distinct 查询，请使用 SELECT DISTINCT 执行 SQL 查询")
}