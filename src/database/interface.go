package database

import (
	"context"
	"time"
)

// DatabaseType 表示支持的数据库类型
type DatabaseType string

const (
	DatabaseTypeMySQL   DatabaseType = "mysql"   // MySQL数据库
	DatabaseTypeMongoDB DatabaseType = "mongodb" // MongoDB数据库
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

	// ExecuteQuery 执行只读查询并返回结果
	ExecuteQuery(ctx context.Context, query string, limit int, timeout time.Duration) (*QueryResult, error)

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
	ID            string       `yaml:"id"`             // 连接标识符（用于池管理）
	Type          DatabaseType `yaml:"type"`           // 数据库类型
	Host          string       `yaml:"host"`           // 数据库服务器地址
	Port          int          `yaml:"port"`           // 数据库服务器端口
	Username      string       `yaml:"username"`       // 认证用户名
	Password      string       `yaml:"password"`       // 认证密码（日志中必须遮蔽）
	Database      string       `yaml:"database"`       // 数据库名称
	TLSEnabled    bool         `yaml:"tls_enabled"`    // 是否启用TLS连接
	PoolSize      int          `yaml:"pool_size"`      // 连接池大小
	Timeout       int          `yaml:"timeout"`        // 连接超时时间（秒）
	AuthSource    string       `yaml:"auth_source"`    // MongoDB认证源数据库（默认admin）
	AuthMechanism string       `yaml:"auth_mechanism"` // MongoDB认证机制（SCRAM-SHA-1/SCRAM-SHA-256）
	ReplicaSet    string       `yaml:"replica_set"`    // MongoDB副本集名称
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
