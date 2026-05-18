package database

import (
	"fmt"
	"strings"
)

// Connection 表示已配置的数据库连接
type Connection struct {
	ID       string          // 连接标识符
	Type     DatabaseType    // 数据库类型
	Config   DatabaseConfig  // 连接配置
	State    ConnectionState // 连接状态
	PoolSize int             // 连接池大小
}

// NewConnection 创建新的数据库连接配置
func NewConnection(config DatabaseConfig) *Connection {
	return &Connection{
		ID:       config.ID,
		Type:     config.Type,
		Config:   config,
		State:    StateDisconnected,
		PoolSize: config.PoolSize,
	}
}

// GetMaskedConnectionString 返回遮蔽密码的连接字符串
// 宪章原则 II 要求：所有日志和错误消息中必须遮蔽密码
func (c *Connection) GetMaskedConnectionString() string {
	switch c.Type {
	case DatabaseTypeMySQL:
		return fmt.Sprintf("%s:[REDACTED]@%s:%d/%s",
			c.Config.Username, c.Config.Host, c.Config.Port, c.Config.Database)
	case DatabaseTypeMongoDB:
		return fmt.Sprintf("mongodb://%s:[REDACTED]@%s:%d/%s",
			c.Config.Username, c.Config.Host, c.Config.Port, c.Config.Database)
	default:
		return "unknown:[REDACTED]@unknown:0/unknown"
	}
}

// GetConnectionString 返回实际连接字符串（仅内部使用）
// 警告：此字符串不得出现在日志或错误消息中
func (c *Connection) GetConnectionString() string {
	switch c.Type {
	case DatabaseTypeMySQL:
		tlsParam := ""
		if c.Config.TLSEnabled {
			tlsParam = "?tls=true"
		}
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s%s",
			c.Config.Username, c.Config.Password, c.Config.Host, c.Config.Port,
			c.Config.Database, tlsParam)
	case DatabaseTypeMongoDB:
		authStr := ""
		if c.Config.Username != "" && c.Config.Password != "" {
			authStr = fmt.Sprintf("%s:%s@", c.Config.Username, c.Config.Password)
		}
		return fmt.Sprintf("mongodb://%s%s:%d/%s",
			authStr, c.Config.Host, c.Config.Port, c.Config.Database)
	default:
		return ""
	}
}

// SetState 更新连接状态
func (c *Connection) SetState(state ConnectionState) {
	c.State = state
}

// IsHealthy 检查连接是否处于可用状态
func (c *Connection) IsHealthy() bool {
	return c.State == StateConnected || c.State == StateIdle || c.State == StateActive
}

// Validate 检查连接配置是否有效
func (c *Connection) Validate() error {
	if c.ID == "" {
		return fmt.Errorf("连接ID不能为空")
	}
	if c.Config.Host == "" {
		return fmt.Errorf("连接 %s 的主机地址不能为空", c.ID)
	}
	if c.Config.Port <= 0 {
		return fmt.Errorf("连接 %s 需要有效的端口号", c.ID)
	}
	if c.Config.Database == "" {
		return fmt.Errorf("连接 %s 需要指定数据库名称", c.ID)
	}
	if c.Type != DatabaseTypeMySQL && c.Type != DatabaseTypeMongoDB {
		return fmt.Errorf("连接 %s 的数据库类型无效: %s", c.ID, c.Type)
	}
	return nil
}

// MaskPasswordInString 替换字符串中的密码（工具函数）
func MaskPasswordInString(s, password string) string {
	if password == "" {
		return s
	}
	return strings.ReplaceAll(s, password, "[REDACTED]")
}
