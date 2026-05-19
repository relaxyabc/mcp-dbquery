package oracle

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	go_ora "github.com/sijms/go-ora/v2" // Oracle驱动连接字符串构建

	"github.com/relaxyabc/mcp-dbquery/src/database"
	"github.com/relaxyabc/mcp-dbquery/src/utils"
)

// OracleDriver Oracle数据库驱动实现
type OracleDriver struct {
	ID     string
	Config database.DatabaseConfig
	DB     *sql.DB
	State  database.ConnectionState
}

// NewOracleDriver 创建Oracle驱动实例
func NewOracleDriver(config database.DatabaseConfig) *OracleDriver {
	return &OracleDriver{
		ID:     config.ID,
		Config: config,
		State:  database.StateDisconnected,
	}
}

// Connect 建立Oracle连接
func (d *OracleDriver) Connect(ctx context.Context) error {
	d.State = database.StateConnecting
	utils.GlobalLogger.LogConnection(d.ID, d.GetMaskedConnectionString(), "connecting")

	// 构建连接字符串
	connStr := d.buildConnectionString()

	// 打开数据库连接
	db, err := sql.Open("oracle", connStr)
	if err != nil {
		d.State = database.StateError
		utils.GlobalLogger.LogError("CONNECTION_ERROR", "Oracle连接打开失败", err.Error())
		return fmt.Errorf("Oracle连接打开失败: %s", err)
	}

	// 配置连接池
	db.SetMaxOpenConns(d.Config.PoolSize)
	db.SetMaxIdleConns(d.Config.PoolSize / 2)
	db.SetConnMaxLifetime(time.Duration(d.Config.Timeout) * time.Second)

	// 测试连接
	if err := db.PingContext(ctx); err != nil {
		d.State = database.StateError
		utils.GlobalLogger.LogError("CONNECTION_ERROR", "Oracle连接测试失败", err.Error())
		return fmt.Errorf("Oracle连接测试失败: %s", err)
	}

	d.DB = db
	d.State = database.StateConnected
	utils.GlobalLogger.LogConnection(d.ID, d.GetMaskedConnectionString(), "connected")

	return nil
}

// Close 关闭Oracle连接
func (d *OracleDriver) Close(ctx context.Context) error {
	if d.DB == nil {
		return nil
	}

	d.State = database.StateClosed
	utils.GlobalLogger.LogConnection(d.ID, d.GetMaskedConnectionString(), "closing")

	if err := d.DB.Close(); err != nil {
		utils.GlobalLogger.LogError("CLOSE_ERROR", "Oracle连接关闭失败", err.Error())
		return err
	}

	d.DB = nil
	utils.GlobalLogger.LogConnection(d.ID, d.GetMaskedConnectionString(), "closed")
	return nil
}

// IsConnected 检查连接状态
func (d *OracleDriver) IsConnected() bool {
	if d.DB == nil {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	return d.DB.PingContext(ctx) == nil
}

// GetType 返回数据库类型
func (d *OracleDriver) GetType() database.DatabaseType {
	return database.DatabaseTypeOracle
}

// GetID 返回连接标识符
func (d *OracleDriver) GetID() string {
	return d.ID
}

// ExecuteQuery 执行只读查询
func (d *OracleDriver) ExecuteQuery(ctx context.Context, query string, limit int, timeout time.Duration) (*database.QueryResult, error) {
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
func (d *OracleDriver) ValidateQuery(query string) error {
	return ValidateOracleQuery(query)
}

// buildConnectionString 构建Oracle连接字符串（使用go-ora）
func (d *OracleDriver) buildConnectionString() string {
	// 使用go-ora v2构建连接字符串
	// 格式：oracle://user:password@host:port/service_name

	options := map[string]string{}

	// 如果配置中有SID而不是Service Name
	if d.Config.AuthSource != "" {
		options["SID"] = d.Config.AuthSource
	}

	// SSL设置
	if d.Config.TLSEnabled {
		options["ssl"] = "true"
	}

	url := go_ora.BuildUrl(d.Config.Host, d.Config.Port, d.Config.Database,
		d.Config.Username, d.Config.Password, options)
	return url
}

// GetMaskedConnectionString 获取遮蔽密码的连接字符串（日志使用）
func (d *OracleDriver) GetMaskedConnectionString() string {
	return fmt.Sprintf("oracle://%s:[REDACTED]@%s:%d/%s",
		d.Config.Username, d.Config.Host, d.Config.Port, d.Config.Database)
}

// GetSchema 获取表结构元数据
func (d *OracleDriver) GetSchema(ctx context.Context, tableName string) (*database.SchemaMetadata, error) {
	return d.DescribeTable(ctx, tableName)
}

// GetIndexes 获取索引元数据
func (d *OracleDriver) GetIndexes(ctx context.Context, tableName string) (*database.IndexListMetadata, error) {
	return d.ShowIndexes(ctx, tableName)
}

// ListTables 列出所有表
func (d *OracleDriver) ListTables(ctx context.Context) ([]string, error) {
	return d.ShowTables(ctx)
}