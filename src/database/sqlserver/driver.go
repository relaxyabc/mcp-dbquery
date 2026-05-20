package sqlserver

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/microsoft/go-mssqldb" // SQL Server驱动

	"github.com/relaxyabc/mcp-dbquery/src/database"
	"github.com/relaxyabc/mcp-dbquery/src/utils"
)

// init() 自动注册 SQL Server 騱动到全局注册表
func init() {
	database.RegisterDriver(database.DatabaseTypeSQLServer, func(config database.DatabaseConfig) database.Database {
		return NewSQLServerDriver(config)
	})
}

// SQLServerDriver SQL Server数据库驱动实现
type SQLServerDriver struct {
	ID     string
	Config database.DatabaseConfig
	DB     *sql.DB
	State  database.ConnectionState
}

// NewSQLServerDriver 创建SQL Server驱动实例
func NewSQLServerDriver(config database.DatabaseConfig) *SQLServerDriver {
	return &SQLServerDriver{
		ID:     config.ID,
		Config: config,
		State:  database.StateDisconnected,
	}
}

// Connect 建立SQL Server连接
func (d *SQLServerDriver) Connect(ctx context.Context) error {
	d.State = database.StateConnecting
	utils.GlobalLogger.LogConnection(d.ID, d.GetMaskedConnectionString(), "connecting")

	// 构建连接字符串
	connStr := d.buildConnectionString()

	// 打开数据库连接
	db, err := sql.Open("mssql", connStr)
	if err != nil {
		d.State = database.StateError
		utils.GlobalLogger.LogError("CONNECTION_ERROR", "SQL Server连接打开失败", err.Error())
		return fmt.Errorf("SQL Server连接打开失败: %s", err)
	}

	// 配置连接池
	db.SetMaxOpenConns(d.Config.PoolSize)
	db.SetMaxIdleConns(d.Config.PoolSize / 2)
	db.SetConnMaxLifetime(time.Duration(d.Config.Timeout) * time.Second)

	// 测试连接
	if err := db.PingContext(ctx); err != nil {
		d.State = database.StateError
		utils.GlobalLogger.LogError("CONNECTION_ERROR", "SQL Server连接测试失败", err.Error())
		return fmt.Errorf("SQL Server连接测试失败: %s", err)
	}

	d.DB = db
	d.State = database.StateConnected
	utils.GlobalLogger.LogConnection(d.ID, d.GetMaskedConnectionString(), "connected")

	return nil
}

// Close 关闭SQL Server连接
func (d *SQLServerDriver) Close(ctx context.Context) error {
	if d.DB == nil {
		return nil
	}

	d.State = database.StateClosed
	utils.GlobalLogger.LogConnection(d.ID, d.GetMaskedConnectionString(), "closing")

	if err := d.DB.Close(); err != nil {
		utils.GlobalLogger.LogError("CLOSE_ERROR", "SQL Server连接关闭失败", err.Error())
		return err
	}

	d.DB = nil
	utils.GlobalLogger.LogConnection(d.ID, d.GetMaskedConnectionString(), "closed")
	return nil
}

// IsConnected 检查连接状态
func (d *SQLServerDriver) IsConnected() bool {
	if d.DB == nil {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	return d.DB.PingContext(ctx) == nil
}

// GetType 返回数据库类型
func (d *SQLServerDriver) GetType() database.DatabaseType {
	return database.DatabaseTypeSQLServer
}

// GetID 返回连接标识符
func (d *SQLServerDriver) GetID() string {
	return d.ID
}

// ExecuteQuery 执行只读查询
func (d *SQLServerDriver) ExecuteQuery(ctx context.Context, query string, limit int, timeout time.Duration) (*database.QueryResult, error) {
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
func (d *SQLServerDriver) ValidateQuery(query string) error {
	return ValidateSQLServerQuery(query)
}

// buildConnectionString 构建SQL Server连接字符串
func (d *SQLServerDriver) buildConnectionString() string {
	// SQL Server连接字符串格式: server=host,port;user id=username;password=password;database=dbname
	// 支持Windows认证: server=host;database=dbname;trusted_connection=yes

	var connStr string

	if d.Config.Username == "" && d.Config.Password == "" {
		// Windows认证模式
		connStr = fmt.Sprintf("server=%s,%d;database=%s;trusted_connection=yes",
			d.Config.Host, d.Config.Port, d.Config.Database)
	} else {
		// SQL认证模式
		connStr = fmt.Sprintf("server=%s,%d;user id=%s;password=%s;database=%s",
			d.Config.Host, d.Config.Port,
			d.Config.Username, d.Config.Password,
			d.Config.Database)
	}

	// TLS设置
	if d.Config.TLSEnabled {
		connStr += ";encrypt=true"
	} else {
		connStr += ";encrypt=false"
	}

	// 连接超时
	connStr += fmt.Sprintf(";connection timeout=%d", d.Config.Timeout)

	return connStr
}

// GetMaskedConnectionString 获取遮蔽密码的连接字符串（日志使用）
func (d *SQLServerDriver) GetMaskedConnectionString() string {
	if d.Config.Username == "" && d.Config.Password == "" {
		return fmt.Sprintf("server=%s,%d;database=%s;trusted_connection=yes",
			d.Config.Host, d.Config.Port, d.Config.Database)
	}
	return fmt.Sprintf("server=%s,%d;user id=%s;password=[REDACTED];database=%s",
		d.Config.Host, d.Config.Port, d.Config.Username, d.Config.Database)
}

// GetSchema 获取表结构元数据
func (d *SQLServerDriver) GetSchema(ctx context.Context, tableName string) (*database.SchemaMetadata, error) {
	return d.DescribeTable(ctx, tableName)
}

// GetIndexes 获取索引元数据
func (d *SQLServerDriver) GetIndexes(ctx context.Context, tableName string) (*database.IndexListMetadata, error) {
	return d.ShowIndexes(ctx, tableName)
}

// ListTables 列出所有表
func (d *SQLServerDriver) ListTables(ctx context.Context) ([]string, error) {
	return d.ShowTables(ctx)
}

// ExecuteSelectQuery 执行SELECT查询并返回结果
func (d *SQLServerDriver) ExecuteSelectQuery(ctx context.Context, query string, limit int) (*database.QueryResult, error) {
	return d.ExecuteQuery(ctx, query, limit, time.Duration(d.Config.Timeout)*time.Second)
}

// ExecuteFind MongoDB find查询（SQL Server不支持，返回错误）
func (d *SQLServerDriver) ExecuteFind(ctx context.Context, collection string, filter map[string]interface{}, limit int) (*database.QueryResult, error) {
	return nil, fmt.Errorf("SQL Server 不支持 MongoDB find 查询，请使用 ExecuteSelectQuery 执行 SQL 查询")
}

// ExecuteAggregate MongoDB聚合查询（SQL Server不支持，返回错误）
func (d *SQLServerDriver) ExecuteAggregate(ctx context.Context, collection string, pipeline []map[string]interface{}, limit int) (*database.QueryResult, error) {
	return nil, fmt.Errorf("SQL Server 不支持 MongoDB aggregate 查询，请使用 ExecuteSelectQuery 执行 SQL 查询")
}

// ExecuteCount MongoDB计数查询（SQL Server不支持，返回错误）
func (d *SQLServerDriver) ExecuteCount(ctx context.Context, collection string, filter map[string]interface{}) (int64, error) {
	return 0, fmt.Errorf("SQL Server 不支持 MongoDB count 查询，请使用 SELECT COUNT(*) 执行 SQL 计数")
}

// ExecuteDistinct MongoDB distinct查询（SQL Server不支持，返回错误）
func (d *SQLServerDriver) ExecuteDistinct(ctx context.Context, collection string, fieldName string, filter map[string]interface{}) ([]interface{}, error) {
	return nil, fmt.Errorf("SQL Server 不支持 MongoDB distinct 查询，请使用 SELECT DISTINCT 执行 SQL 查询")
}