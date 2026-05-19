package database

import (
	// 驱动注册导入 - 添加新驱动时在此导入
)

// RegisterAllDrivers 注册所有已知数据库驱动
// 此函数在main.go启动时调用
func RegisterAllDrivers(pm *PoolManager) {
	// PostgreSQL驱动由postgres包提供
	// MySQL驱动由mysql包提供
	// MongoDB驱动由mongodb包提供
	// 新驱动在各自包实现后在此注册
}

// DefaultDriverRegistry 返回默认驱动注册表
// 每个驱动包应在其init()中调用此函数注册
func defaultDriverRegistry() map[DatabaseType]DriverConstructor {
	return make(map[DatabaseType]DriverConstructor)
}