package database

import (
	"context"
	"time"
)

// DatabaseType 表示支持的数据库类型
type DatabaseType string

const (
	DatabaseTypeMySQL       DatabaseType = "mysql"       // MySQL数据库
	DatabaseTypeMongoDB     DatabaseType = "mongodb"     // MongoDB数据库
	DatabaseTypePostgreSQL  DatabaseType = "postgres"    // PostgreSQL数据库
	DatabaseTypeSQLite      DatabaseType = "sqlite"      // SQLite数据库
	DatabaseTypeSQLServer   DatabaseType = "sqlserver"   // SQL Server数据库
	DatabaseTypeOracle      DatabaseType = "oracle"      // Oracle数据库
	DatabaseTypeClickHouse  DatabaseType = "clickhouse"  // ClickHouse数据库 (MySQL协议兼容)
	DatabaseTypeDoris       DatabaseType = "doris"       // Doris数据库 (MySQL协议兼容)
	DatabaseTypeMariaDB     DatabaseType = "mariadb"     // MariaDB数据库 (MySQL兼容)
	DatabaseTypeTiDB        DatabaseType = "tidb"        // TiDB数据库 (MySQL兼容)
)

// Database 接口定义所有数据库类型的通用操作
type Database interface {
	// Connect 建立数据库连接
	Connect(ctx context.Context) error

	// Close 关闭数据库连接
	Close(ctx context.Context) error

	// IsConnected 检查连接是否活跃
	IsConnected() bool

	// GetType 返回数据库类型
	GetType() DatabaseType

	// GetID 返回数据库标识符
	GetID() string

	// ExecuteQuery 执行只读查询并返回结果（通用接口）
	ExecuteQuery(ctx context.Context, query string, limit int, timeout time.Duration) (*QueryResult, error)

	// ExecuteSelectQuery 执行SQL SELECT查询（SQL数据库专用，MongoDB返回不支持）
	ExecuteSelectQuery(ctx context.Context, query string, limit int) (*QueryResult, error)

	// ExecuteFind 执行MongoDB find查询（MongoDB专用，SQL数据库返回不支持）
	ExecuteFind(ctx context.Context, collection string, filter map[string]interface{}, limit int) (*QueryResult, error)

	// ExecuteAggregate 执行MongoDB聚合查询（MongoDB专用，SQL数据库返回不支持）
	ExecuteAggregate(ctx context.Context, collection string, pipeline []map[string]interface{}, limit int) (*QueryResult, error)

	// ExecuteCount 执行计数查询（MongoDB专用计数，SQL数据库返回不支持）
	ExecuteCount(ctx context.Context, collection string, filter map[string]interface{}) (int64, error)

	// ExecuteDistinct 执行distinct查询（MongoDB专用，SQL数据库返回不支持）
	ExecuteDistinct(ctx context.Context, collection string, fieldName string, filter map[string]interface{}) ([]interface{}, error)

	// GetSchema 返回表/集合结构元数据
	GetSchema(ctx context.Context, tableName string) (*SchemaMetadata, error)

	// GetIndexes 返回表/集合的索引元数据列表
	GetIndexes(ctx context.Context, tableName string) (*IndexListMetadata, error)

	// ListTables 返回所有可访问的表/集合列表
	ListTables(ctx context.Context) ([]string, error)

	// ValidateQuery 验证查询是否为只读操作（禁止修改操作）
	ValidateQuery(query string) error
}

// DatabaseConfig 数据库连接配置
type DatabaseConfig struct {
	ID               string       `yaml:"id"`               // 连接标识符（用于池管理）
	Type             DatabaseType `yaml:"type"`             // 数据库类型
	Host             string       `yaml:"host"`             // 数据库服务器地址
	Port             int          `yaml:"port"`             // 数据库服务器端口
	Username         string       `yaml:"username"`         // 认证用户名
	Password         string       `yaml:"password"`         // 认证密码（日志中必须遮蔽）
	Database         string       `yaml:"database"`         // 数据库名称
	Path             string       `yaml:"path"`             // SQLite文件路径（仅SQLite使用）
	TLSEnabled       bool         `yaml:"tls_enabled"`      // 是否启用TLS连接
	PoolSize         int          `yaml:"pool_size"`        // 连接池大小
	Timeout          int          `yaml:"timeout"`          // 连接超时时间（秒）
	AuthSource       string       `yaml:"auth_source"`      // MongoDB认证源数据库（默认admin）
	AuthMechanism    string       `yaml:"auth_mechanism"`   // MongoDB认证机制（SCRAM-SHA-1/SCRAM-SHA-256）
	ReplicaSet       string       `yaml:"replica_set"`      // MongoDB副本集名称
	ProtocolCompat   string       `yaml:"protocol_compatible"` // MySQL协议兼容标记（clickhouse, doris等复用MySQL驱动）
}

// ConnectionState 表示数据库连接状态
type ConnectionState string

const (
	StateDisconnected ConnectionState = "disconnected" // 已断开
	StateConnecting   ConnectionState = "connecting"   // 正在连接
	StateConnected    ConnectionState = "connected"    // 已连接
	StateIdle         ConnectionState = "idle"         // 空闲状态
	StateActive       ConnectionState = "active"       // 活跃状态
	StateError        ConnectionState = "error"        // 错误状态
	StateClosed       ConnectionState = "closed"       // 已关闭
)
