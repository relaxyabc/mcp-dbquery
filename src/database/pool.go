package database

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// PoolManager 管理所有数据库连接池
type PoolManager struct {
	mysqlDrivers map[string]interface{} // MySQL驱动实例（实际为*mysql.MySQLDriver）
	mongoDrivers map[string]interface{} // MongoDB驱动实例（实际为*mongodb.MongoDBDriver）
	configs      map[string]DatabaseConfig // 连接配置缓存
	mu           sync.RWMutex              // 读写锁保护并发访问
}

// NewPoolManager 创建新的连接池管理器
func NewPoolManager() *PoolManager {
	return &PoolManager{
		mysqlDrivers: make(map[string]interface{}),
		mongoDrivers: make(map[string]interface{}),
		configs:      make(map[string]DatabaseConfig),
	}
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

// SetMySQLDriver 设置MySQL驱动实例（由外部调用）
func (pm *PoolManager) SetMySQLDriver(id string, driver interface{}) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.mysqlDrivers[id] = driver
}

// SetMongoDriver 设置MongoDB驱动实例（由外部调用）
func (pm *PoolManager) SetMongoDriver(id string, driver interface{}) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.mongoDrivers[id] = driver
}

// GetMySQLDriver 获取MySQL驱动实例
func (pm *PoolManager) GetMySQLDriver(id string) (interface{}, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	driver, exists := pm.mysqlDrivers[id]
	return driver, exists
}

// GetMongoDriver 获取MongoDB驱动实例
func (pm *PoolManager) GetMongoDriver(id string) (interface{}, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	driver, exists := pm.mongoDrivers[id]
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

	// 关闭MySQL连接
	for id, driver := range pm.mysqlDrivers {
		if d, ok := driver.(Database); ok {
			if err := d.Close(ctx); err != nil {
				errors = append(errors, fmt.Sprintf("MySQL %s: %s", id, err))
			}
		}
	}

	// 关闭MongoDB连接
	for id, driver := range pm.mongoDrivers {
		if d, ok := driver.(Database); ok {
			if err := d.Close(ctx); err != nil {
				errors = append(errors, fmt.Sprintf("MongoDB %s: %s", id, err))
			}
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

	mysqlStatus := make(map[string]string)
	for id, driver := range pm.mysqlDrivers {
		if d, ok := driver.(Database); ok {
			if d.IsConnected() {
				mysqlStatus[id] = "connected"
			} else {
				mysqlStatus[id] = "disconnected"
			}
		} else {
			mysqlStatus[id] = "not_initialized"
		}
	}

	mongoStatus := make(map[string]string)
	for id, driver := range pm.mongoDrivers {
		if d, ok := driver.(Database); ok {
			if d.IsConnected() {
				mongoStatus[id] = "connected"
			} else {
				mongoStatus[id] = "disconnected"
			}
		} else {
			mongoStatus[id] = "not_initialized"
		}
	}

	status["mysql"] = mysqlStatus
	status["mongodb"] = mongoStatus
	status["total_connections"] = len(pm.mysqlDrivers) + len(pm.mongoDrivers)

	return status
}

// EnforceTimeout 强制执行查询超时（通过context）
func (pm *PoolManager) EnforceTimeout(ctx context.Context, timeoutSeconds int) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
}