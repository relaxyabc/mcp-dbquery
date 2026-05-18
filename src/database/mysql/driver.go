package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql" // MySQL驱动

	"github.com/relaxyabc/mcp-dbquery/src/database"
	"github.com/relaxyabc/mcp-dbquery/src/utils"
)

// MySQLDriver MySQL数据库驱动实现
type MySQLDriver struct {
	ID     string
	Config database.DatabaseConfig
	DB     *sql.DB
	State  database.ConnectionState
}

// NewMySQLDriver 创建MySQL驱动实例
func NewMySQLDriver(config database.DatabaseConfig) *MySQLDriver {
	return &MySQLDriver{
		ID:     config.ID,
		Config: config,
		State:  database.StateDisconnected,
	}
}

// Connect 建立MySQL连接
func (d *MySQLDriver) Connect(ctx context.Context) error {
	d.State = database.StateConnecting
	utils.GlobalLogger.LogConnection(d.ID, d.GetMaskedConnectionString(), "connecting")

	// 构建连接字符串
	connStr := d.buildConnectionString()

	// 打开连接
	db, err := sql.Open("mysql", connStr)
	if err != nil {
		d.State = database.StateError
		utils.GlobalLogger.LogError("CONNECTION_ERROR", "MySQL连接失败", err.Error())
		return fmt.Errorf("MySQL连接失败: %s", err)
	}

	// 配置连接池
	db.SetMaxOpenConns(d.Config.PoolSize)
	db.SetMaxIdleConns(d.Config.PoolSize)
	db.SetConnMaxLifetime(time.Duration(d.Config.Timeout) * time.Second)

	// 测试连接
	if err := db.PingContext(ctx); err != nil {
		d.State = database.StateError
		utils.GlobalLogger.LogError("CONNECTION_ERROR", "MySQL连接测试失败", err.Error())
		return fmt.Errorf("MySQL连接测试失败: %s", err)
	}

	d.DB = db
	d.State = database.StateConnected
	utils.GlobalLogger.LogConnection(d.ID, d.GetMaskedConnectionString(), "connected")

	return nil
}

// Close 关闭MySQL连接
func (d *MySQLDriver) Close(ctx context.Context) error {
	if d.DB == nil {
		return nil
	}

	d.State = database.StateClosed
	utils.GlobalLogger.LogConnection(d.ID, d.GetMaskedConnectionString(), "closing")

	if err := d.DB.Close(); err != nil {
		utils.GlobalLogger.LogError("CONNECTION_ERROR", "MySQL连接关闭失败", err.Error())
		return err
	}

	d.DB = nil
	utils.GlobalLogger.LogConnection(d.ID, d.GetMaskedConnectionString(), "closed")
	return nil
}

// IsConnected 检查连接状态
func (d *MySQLDriver) IsConnected() bool {
	if d.DB == nil {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	return d.DB.PingContext(ctx) == nil
}

// GetType 返回数据库类型
func (d *MySQLDriver) GetType() database.DatabaseType {
	return database.DatabaseTypeMySQL
}

// GetID 返回连接标识符
func (d *MySQLDriver) GetID() string {
	return d.ID
}

// ExecuteQuery 执行只读查询
func (d *MySQLDriver) ExecuteQuery(ctx context.Context, query string, limit int, timeout time.Duration) (*database.QueryResult, error) {
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
		return database.NewErrorResult(d.ID, "QUERY_ERROR", "无法获取列信息"), err
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
			// 处理NULL值和字节切片
			val := values[i]
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

// ValidateQuery 验证查询是否为只读（宪章要求：严格只读）
func (d *MySQLDriver) ValidateQuery(query string) error {
	return ValidateMySQLQuery(query)
}

// buildConnectionString 构建MySQL连接字符串（内部使用）
func (d *MySQLDriver) buildConnectionString() string {
	tlsParam := ""
	if d.Config.TLSEnabled {
		tlsParam = "?tls=true"
	}

	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s%s",
		d.Config.Username, d.Config.Password,
		d.Config.Host, d.Config.Port,
		d.Config.Database, tlsParam)
}

// GetMaskedConnectionString 获取遮蔽密码的连接字符串（日志使用）
func (d *MySQLDriver) GetMaskedConnectionString() string {
	return fmt.Sprintf("%s:[REDACTED]@%s:%d/%s",
		d.Config.Username, d.Config.Host, d.Config.Port, d.Config.Database)
}

// GetSchema 获取表结构元数据
func (d *MySQLDriver) GetSchema(ctx context.Context, tableName string) (*database.SchemaMetadata, error) {
	return d.DescribeTable(ctx, tableName)
}

// GetIndexes 获取索引元数据
func (d *MySQLDriver) GetIndexes(ctx context.Context, tableName string) (*database.IndexListMetadata, error) {
	return d.ShowIndexes(ctx, tableName)
}

// ListTables 列出所有表
func (d *MySQLDriver) ListTables(ctx context.Context) ([]string, error) {
	return d.ShowTables(ctx)
}
