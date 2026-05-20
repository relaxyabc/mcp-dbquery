package database

import (
	"sync"
)

// 全局驱动注册表
// 驱动包通过 init() 函数调用 RegisterDriver 注册自身
var globalRegistry = make(map[DatabaseType]DriverConstructor)
var registryMu sync.RWMutex

// RegisterDriver 注册驱动构造函数（供驱动包 init() 调用）
func RegisterDriver(dbType DatabaseType, constructor DriverConstructor) {
	registryMu.Lock()
	defer registryMu.Unlock()
	globalRegistry[dbType] = constructor
}

// GetRegisteredDriver 获取已注册的驱动构造函数
func GetRegisteredDriver(dbType DatabaseType) (DriverConstructor, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	constructor, exists := globalRegistry[dbType]
	return constructor, exists
}

// GetAllRegisteredDrivers 获取所有已注册驱动
func GetAllRegisteredDrivers() map[DatabaseType]DriverConstructor {
	registryMu.RLock()
	defer registryMu.RUnlock()
	result := make(map[DatabaseType]DriverConstructor)
	for k, v := range globalRegistry {
		result[k] = v
	}
	return result
}

// RegisterAllDrivers 注册所有已知数据库驱动
// 此函数在main.go启动时调用（已废弃，改用 init() 自注册）
func RegisterAllDrivers(pm *PoolManager) {
	// PostgreSQL驱动由postgres包提供
	// MySQL驱动由mysql包提供
	// MongoDB驱动由mongodb包提供
	// 新驱动在各自包实现后在此注册
}

// defaultDriverRegistry 返回默认驱动注册表
// 每个驱动包应在其init()中调用此函数注册（已废弃）
func defaultDriverRegistry() map[DatabaseType]DriverConstructor {
	return make(map[DatabaseType]DriverConstructor)
}