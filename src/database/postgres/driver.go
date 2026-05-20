package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/relaxyabc/mcp-dbquery/src/database"
	"github.com/relaxyabc/mcp-dbquery/src/utils"
)

// init() 自动注册 PostgreSQL 驱动到全局注册表
func init() {
	database.RegisterDriver(database.DatabaseTypePostgreSQL, func(config database.DatabaseConfig) database.Database {
		return NewPostgresDriver(config)
	})
}

// PostgresDriver PostgreSQL数据库驱动实现
type PostgresDriver struct {
	ID     string
	Config database.DatabaseConfig
	Pool   *pgxpool.Pool
	State  database.ConnectionState
}

// NewPostgresDriver 创建PostgreSQL驱动实例
func NewPostgresDriver(config database.DatabaseConfig) *PostgresDriver {
	return &PostgresDriver{
		ID:     config.ID,
		Config: config,
		State:  database.StateDisconnected,
	}
}

// Connect 建立PostgreSQL连接
func (d *PostgresDriver) Connect(ctx context.Context) error {
	d.State = database.StateConnecting
	utils.GlobalLogger.LogConnection(d.ID, d.GetMaskedConnectionString(), "connecting")

	// 构建连接字符串
	connStr := d.buildConnectionString()

	// 创建连接池
	poolConfig, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		d.State = database.StateError
		utils.GlobalLogger.LogError("CONNECTION_ERROR", "PostgreSQL连接配置解析失败", err.Error())
		return fmt.Errorf("PostgreSQL连接配置解析失败: %s", err)
	}

	// 配置连接池
	poolConfig.MaxConns = int32(d.Config.PoolSize)
	poolConfig.MinConns = int32(d.Config.PoolSize / 2)
	poolConfig.MaxConnLifetime = time.Duration(d.Config.Timeout) * time.Second
	poolConfig.HealthCheckPeriod = 10 * time.Second

	// 创建连接池
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		d.State = database.StateError
		utils.GlobalLogger.LogError("CONNECTION_ERROR", "PostgreSQL连接池创建失败", err.Error())
		return fmt.Errorf("PostgreSQL连接池创建失败: %s", err)
	}

	// 测试连接
	if err := pool.Ping(ctx); err != nil {
		d.State = database.StateError
		utils.GlobalLogger.LogError("CONNECTION_ERROR", "PostgreSQL连接测试失败", err.Error())
		return fmt.Errorf("PostgreSQL连接测试失败: %s", err)
	}

	d.Pool = pool
	d.State = database.StateConnected
	utils.GlobalLogger.LogConnection(d.ID, d.GetMaskedConnectionString(), "connected")

	return nil
}

// Close 关闭PostgreSQL连接
func (d *PostgresDriver) Close(ctx context.Context) error {
	if d.Pool == nil {
		return nil
	}

	d.State = database.StateClosed
	utils.GlobalLogger.LogConnection(d.ID, d.GetMaskedConnectionString(), "closing")

	d.Pool.Close()

	d.Pool = nil
	utils.GlobalLogger.LogConnection(d.ID, d.GetMaskedConnectionString(), "closed")
	return nil
}

// IsConnected 检查连接状态
func (d *PostgresDriver) IsConnected() bool {
	if d.Pool == nil {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	return d.Pool.Ping(ctx) == nil
}

// GetType 返回数据库类型
func (d *PostgresDriver) GetType() database.DatabaseType {
	return database.DatabaseTypePostgreSQL
}

// GetID 返回连接标识符
func (d *PostgresDriver) GetID() string {
	return d.ID
}

// ExecuteQuery 执行只读查询
func (d *PostgresDriver) ExecuteQuery(ctx context.Context, query string, limit int, timeout time.Duration) (*database.QueryResult, error) {
	start := time.Now()

	// 验证查询是否为只读
	if err := d.ValidateQuery(query); err != nil {
		return nil, err
	}

	// 创建超时上下文
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// 执行查询
	rows, err := d.Pool.Query(ctx, query)
	if err != nil {
		return database.NewErrorResult(d.ID, "QUERY_ERROR", fmt.Sprintf("查询执行失败: %s", err)), err
	}
	defer rows.Close()

	// 获取列信息
	columns := make([]string, 0)
	for _, fd := range rows.FieldDescriptions() {
		columns = append(columns, fd.Name)
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
func (d *PostgresDriver) ValidateQuery(query string) error {
	return ValidatePostgresQuery(query)
}

// buildConnectionString 构建PostgreSQL连接字符串
func (d *PostgresDriver) buildConnectionString() string {
	// PostgreSQL连接字符串格式: postgres://user:password@host:port/database
	sslMode := "disable"
	if d.Config.TLSEnabled {
		sslMode = "require"
	}

	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s&pool_max_conns=%d",
		d.Config.Username, d.Config.Password,
		d.Config.Host, d.Config.Port,
		d.Config.Database, sslMode, d.Config.PoolSize)
}

// GetMaskedConnectionString 获取遮蔽密码的连接字符串（日志使用）
func (d *PostgresDriver) GetMaskedConnectionString() string {
	return fmt.Sprintf("postgres://%s:[REDACTED]@%s:%d/%s",
		d.Config.Username, d.Config.Host, d.Config.Port, d.Config.Database)
}

// GetSchema 获取表结构元数据
func (d *PostgresDriver) GetSchema(ctx context.Context, tableName string) (*database.SchemaMetadata, error) {
	return d.DescribeTable(ctx, tableName)
}

// GetIndexes 获取索引元数据
func (d *PostgresDriver) GetIndexes(ctx context.Context, tableName string) (*database.IndexListMetadata, error) {
	return d.ShowIndexes(ctx, tableName)
}

// ListTables 列出所有表
func (d *PostgresDriver) ListTables(ctx context.Context) ([]string, error) {
	return d.ShowTables(ctx)
}