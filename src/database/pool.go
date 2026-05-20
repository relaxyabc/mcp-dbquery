package database

import (
	"context"
	"fmt"
	"sync"
	"time"

	// PostgreSQL驱动（静态注册）
	_ "github.com/jackc/pgx/v5/pgxpool"
	// SQLite驱动（静态注册）
	_ "github.com/mattn/go-sqlite3"
)

// DriverConstructor 驱动构造函数签名
type DriverConstructor func(config DatabaseConfig) Database

// PoolManager 管理所有数据库连接池
type PoolManager struct {
	drivers     map[string]Database       // 所有驱动实例（按ID存储）
	configs     map[string]DatabaseConfig // 连接配置缓存
	registry    map[DatabaseType]DriverConstructor // 驱动注册表
	mu          sync.RWMutex              // 读写锁保护并发访问
}

// NewPoolManager 创建新的连接池管理器
func NewPoolManager() *PoolManager {
	pm := &PoolManager{
		drivers:  make(map[string]Database),
		configs:  make(map[string]DatabaseConfig),
		registry: make(map[DatabaseType]DriverConstructor),
	}

	// 驱动注册由外部调用RegisterDriver完成
	// 参见 main.go 或各驱动包的 init() 函数

	return pm
}

// RegisterDriver 注册数据库驱动构造函数
func (pm *PoolManager) RegisterDriver(dbType DatabaseType, constructor DriverConstructor) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.registry[dbType] = constructor
}

// RegisterConfig 注册数据库连接配置
func (pm *PoolManager) RegisterConfig(config DatabaseConfig) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if config.ID == "" {
		return fmt.Errorf("连接配置ID不能为空")
	}

	if _, exists := pm.configs[config.ID]; exists {
		return fmt.Errorf("连接配置 %s 已存在", config.ID)
	}

	// 设置默认值
	if config.PoolSize <= 0 {
		config.PoolSize = 5 // 默认5个连接
	}
	if config.Timeout <= 0 {
		config.Timeout = 30 // 默认30秒超时
	}

	pm.configs[config.ID] = config
	return nil
}

// GetOrCreatePool 获取或创建数据库驱动实例（统一接口）
// 使用全局注册表获取驱动构造函数
func (pm *PoolManager) GetOrCreatePool(ctx context.Context, id string) (Database, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// 检查已存在的驱动
	if driver, exists := pm.drivers[id]; exists {
		if driver.IsConnected() {
			return driver, nil
		}
		// 连接断开，尝试重连
		if err := driver.Connect(ctx); err != nil {
			return nil, fmt.Errorf("重新连接失败: %s", err)
		}
		return driver, nil
	}

	// 获取配置
	config, exists := pm.configs[id]
	if !exists {
		return nil, fmt.Errorf("连接配置 %s 不存在", id)
	}

	// 确定实际驱动类型（处理协议兼容）
	actualType := GetActualDriverType(config)

	// 从全局注册表获取驱动构造函数
	constructor, exists := GetRegisteredDriver(actualType)
	if !exists {
		return nil, fmt.Errorf("数据库类型 %s 未注册驱动，请检查是否已导入驱动包", actualType)
	}

	// 创建驱动实例
	driver := constructor(config)

	// 建立连接
	if err := driver.Connect(ctx); err != nil {
		return nil, fmt.Errorf("连接失败: %s", err)
	}

	pm.drivers[id] = driver
	return driver, nil
}

// GetDriver 获取已创建的驱动实例
func (pm *PoolManager) GetDriver(id string) (Database, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	driver, exists := pm.drivers[id]
	return driver, exists
}

// GetConfig 获取数据库配置
func (pm *PoolManager) GetConfig(id string) (DatabaseConfig, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	config, exists := pm.configs[id]
	return config, exists
}

// GetAllConfigs 获取所有数据库配置
func (pm *PoolManager) GetAllConfigs() map[string]DatabaseConfig {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	// 返回副本避免并发问题
	result := make(map[string]DatabaseConfig)
	for k, v := range pm.configs {
		result[k] = v
	}
	return result
}

// CloseAll 关闭所有数据库连接（优雅关闭）
func (pm *PoolManager) CloseAll(ctx context.Context) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	var errors []string

	for id, driver := range pm.drivers {
		if err := driver.Close(ctx); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %s", id, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("关闭连接失败: %s", errors)
	}

	return nil
}

// GetPoolStatus 获取所有连接状态（用于健康检查）
func (pm *PoolManager) GetPoolStatus() map[string]interface{} {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	status := make(map[string]interface{})
	driverStatus := make(map[string]string)

	for id, driver := range pm.drivers {
		if driver.IsConnected() {
			driverStatus[id] = "connected"
		} else {
			driverStatus[id] = "disconnected"
		}
	}

	// 添加未初始化的配置
	for id := range pm.configs {
		if _, exists := pm.drivers[id]; !exists {
			driverStatus[id] = "not_initialized"
		}
	}

	status["drivers"] = driverStatus
	status["total_connections"] = len(pm.drivers)
	status["registered_types"] = len(pm.registry)

	return status
}

// EnforceTimeout 强制执行查询超时（通过context）
func (pm *PoolManager) EnforceTimeout(ctx context.Context, timeoutSeconds int) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
}

// IsMySQLProtocolCompatible 检查数据库类型是否为MySQL协议兼容
func IsMySQLProtocolCompatible(dbType DatabaseType) bool {
	switch dbType {
	case DatabaseTypeClickHouse, DatabaseTypeDoris, DatabaseTypeMariaDB, DatabaseTypeTiDB:
		return true
	default:
		return false
	}
}

// GetActualDriverType 获取实际使用的驱动类型（处理协议兼容）
func GetActualDriverType(config DatabaseConfig) DatabaseType {
	if config.ProtocolCompat != "" {
		return DatabaseTypeMySQL
	}
	// MySQL协议兼容但无protocol_compatible标记的，也使用MySQL驱动
	if IsMySQLProtocolCompatible(config.Type) {
		return DatabaseTypeMySQL
	}
	return config.Type
}

// SetMySQLDriver 设置MySQL驱动实例（向后兼容，由外部调用）
func (pm *PoolManager) SetMySQLDriver(id string, driver interface{}) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if d, ok := driver.(Database); ok {
		pm.drivers[id] = d
	}
}

// SetMongoDriver 设置MongoDB驱动实例（向后兼容，由外部调用）
func (pm *PoolManager) SetMongoDriver(id string, driver interface{}) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if d, ok := driver.(Database); ok {
		pm.drivers[id] = d
	}
}

// GetMySQLDriver 获取MySQL驱动实例（向后兼容）
func (pm *PoolManager) GetMySQLDriver(id string) (interface{}, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	driver, exists := pm.drivers[id]
	return driver, exists
}

// GetMongoDriver 获取MongoDB驱动实例（向后兼容）
func (pm *PoolManager) GetMongoDriver(id string) (interface{}, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	driver, exists := pm.drivers[id]
	return driver, exists
}