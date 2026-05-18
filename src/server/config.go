package server

import (
	"fmt"
	"os"
	"sync"

	"gopkg.in/yaml.v3"

	"github.com/relaxyabc/mcp-dbquery/src/database"
	"github.com/relaxyabc/mcp-dbquery/src/utils"
)

// Config 服务器配置结构
type Config struct {
	Server    ServerConfig                     `yaml:"server"`    // 服务器配置
	Logging   LoggingConfig                    `yaml:"logging"`   // 日志配置
	Databases map[string]DatabaseConfigWrapper `yaml:"databases"` // 数据库连接配置
	Limits    LimitsConfig                     `yaml:"limits"`    // 限制配置
}

// TransportMode 传输模式类型
type TransportMode string

const (
	TransportModeStdio TransportMode = "stdio" // STDIO传输模式（stdin/stdout）
	TransportModeHTTP  TransportMode = "http"  // HTTP传输模式（HTTP服务器）
)

// ServerConfig HTTP服务器配置
type ServerConfig struct {
	Host      string        `yaml:"host"`      // 监听地址（默认0.0.0.0）
	Port      int           `yaml:"port"`      // 监听端口（默认8080）
	APIKey    string        `yaml:"api_key"`   // API密钥（环境变量）
	Transport TransportMode `yaml:"transport"` // 传输模式（默认stdio）
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Level         string `yaml:"level"`          // 日志级别（debug/info/warn/error）
	MaskPasswords bool   `yaml:"mask_passwords"` // 是否遮蔽密码（必须true）
}

// DatabaseConfigWrapper 数据库配置包装器（用于YAML解析）
type DatabaseConfigWrapper struct {
	Type          string `yaml:"type"`           // 数据库类型（mysql/mongodb）
	Host          string `yaml:"host"`           // 主机地址
	Port          int    `yaml:"port"`           // 端口
	Username      string `yaml:"username"`       // 用户名
	Password      string `yaml:"password"`       // 密码（环境变量）
	Database      string `yaml:"database"`       // 数据库名称
	TLSEnabled    bool   `yaml:"tls_enabled"`    // 是否启用TLS
	PoolSize      int    `yaml:"pool_size"`      // 连接池大小
	Timeout       int    `yaml:"timeout"`        // 超时时间（秒）
	AuthSource    string `yaml:"auth_source"`    // MongoDB认证源数据库
	AuthMechanism string `yaml:"auth_mechanism"` // MongoDB认证机制
	ReplicaSet    string `yaml:"replica_set"`    // MongoDB副本集名称
}

// LimitsConfig 限制配置
type LimitsConfig struct {
	MaxRows      int `yaml:"max_rows"`      // 最大返回行数（默认1000）
	QueryTimeout int `yaml:"query_timeout"` // 查询超时（秒，默认30）
}

// ConfigLoader 配置加载器
type ConfigLoader struct {
	configPath string       // 配置文件路径
	config     *Config      // 已加载的配置
	mu         sync.RWMutex // 读写锁
}

// NewConfigLoader 创建配置加载器
func NewConfigLoader(configPath string) *ConfigLoader {
	return &ConfigLoader{
		configPath: configPath,
	}
}

// Load 加载配置文件
func (cl *ConfigLoader) Load() (*Config, error) {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	// 检查文件是否存在
	if _, err := os.Stat(cl.configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("配置文件不存在: %s", cl.configPath)
	}

	// 读取文件
	data, err := os.ReadFile(cl.configPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %s", err)
	}

	// 解析YAML
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %s", err)
	}

	// 扩展环境变量
	cl.expandEnvVars(&config)

	// 验证配置
	if err := cl.validate(&config); err != nil {
		return nil, fmt.Errorf("配置验证失败: %s", err)
	}

	// 设置默认值
	cl.setDefaults(&config)

	cl.config = &config
	return &config, nil
}

// expandEnvVars 扩展配置中的环境变量
func (cl *ConfigLoader) expandEnvVars(config *Config) {
	// 扩展API密钥
	config.Server.APIKey = utils.ExpandEnvVars(config.Server.APIKey)

	// 扩展数据库配置
	for id, dbConfig := range config.Databases {
		dbConfig.Host = utils.ExpandEnvVars(dbConfig.Host)
		dbConfig.Password = utils.ExpandEnvVars(dbConfig.Password)
		dbConfig.Database = utils.ExpandEnvVars(dbConfig.Database)
		config.Databases[id] = dbConfig
	}
}

// validate 验证配置有效性
func (cl *ConfigLoader) validate(config *Config) error {
	// 验证传输模式（空值允许，后续会设置默认值）
	if config.Server.Transport != "" &&
		config.Server.Transport != TransportModeStdio &&
		config.Server.Transport != TransportModeHTTP {
		return fmt.Errorf("传输模式无效: %s (必须是 stdio 或 http)", config.Server.Transport)
	}

	// HTTP模式需要验证API密钥（如果transport已明确指定为http）
	if config.Server.Transport == TransportModeHTTP {
		if config.Server.APIKey == "" {
			return fmt.Errorf("HTTP模式需要设置API密钥 (api_key)")
		}
		if err := utils.ValidateAPIKey(config.Server.APIKey); err != nil {
			return err
		}
	}

	// 验证端口（HTTP模式使用）
	if err := utils.ValidatePort(config.Server.Port); err != nil {
		return err
	}

	// 验证数据库配置
	for id, dbConfig := range config.Databases {
		if dbConfig.Type != "mysql" && dbConfig.Type != "mongodb" {
			return fmt.Errorf("数据库 %s 类型无效: %s", id, dbConfig.Type)
		}
		if dbConfig.Host == "" {
			return fmt.Errorf("数据库 %s 主机地址不能为空", id)
		}
		if dbConfig.PoolSize > 0 {
			if err := utils.ValidatePoolSize(dbConfig.PoolSize); err != nil {
				return fmt.Errorf("数据库 %s: %s", id, err)
			}
		}
	}

	// 强制密码遮蔽（宪章要求）
	if !config.Logging.MaskPasswords {
		config.Logging.MaskPasswords = true // 强制设置为true
	}

	return nil
}

// setDefaults 设置默认值
func (cl *ConfigLoader) setDefaults(config *Config) {
	// 传输模式默认值（默认为stdio）
	if config.Server.Transport == "" {
		config.Server.Transport = TransportModeStdio
	}

	// 服务器默认值
	if config.Server.Host == "" {
		config.Server.Host = "0.0.0.0"
	}
	if config.Server.Port == 0 {
		config.Server.Port = 8080
	}

	// 日志默认值
	if config.Logging.Level == "" {
		config.Logging.Level = "info"
	}

	// 限制默认值
	if config.Limits.MaxRows == 0 {
		config.Limits.MaxRows = 1000
	}
	if config.Limits.QueryTimeout == 0 {
		config.Limits.QueryTimeout = 30
	}

	// 数据库连接池默认值
	for id, dbConfig := range config.Databases {
		if dbConfig.PoolSize == 0 {
			dbConfig.PoolSize = 5
		}
		if dbConfig.Timeout == 0 {
			dbConfig.Timeout = 30
		}
		config.Databases[id] = dbConfig
	}
}

// GetDatabaseConfigs 获取数据库配置列表
func (cl *ConfigLoader) GetDatabaseConfigs() map[string]database.DatabaseConfig {
	cl.mu.RLock()
	defer cl.mu.RUnlock()

	configs := make(map[string]database.DatabaseConfig)
	for id, wrapper := range cl.config.Databases {
		dbType := database.DatabaseTypeMySQL
		if wrapper.Type == "mongodb" {
			dbType = database.DatabaseTypeMongoDB
		}

		configs[id] = database.DatabaseConfig{
			ID:            id,
			Type:          dbType,
			Host:          wrapper.Host,
			Port:          wrapper.Port,
			Username:      wrapper.Username,
			Password:      wrapper.Password,
			Database:      wrapper.Database,
			TLSEnabled:    wrapper.TLSEnabled,
			PoolSize:      wrapper.PoolSize,
			Timeout:       wrapper.Timeout,
			AuthSource:    wrapper.AuthSource,
			AuthMechanism: wrapper.AuthMechanism,
			ReplicaSet:    wrapper.ReplicaSet,
		}
	}

	return configs
}

// GetConfig 获取已加载的配置
func (cl *ConfigLoader) GetConfig() *Config {
	cl.mu.RLock()
	defer cl.mu.RUnlock()
	return cl.config
}

// Reload 重新加载配置
func (cl *ConfigLoader) Reload() (*Config, error) {
	return cl.Load()
}
