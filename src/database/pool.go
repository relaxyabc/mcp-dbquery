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

	"github.com/relaxyabc/mcp-dbquery/src/utils"
)

// DriverConstructor 驱动构造函数签名
type DriverConstructor func(config DatabaseConfig) Database

// DefaultConnectTimeout 默认拨号连接超时
// 网络未就绪时快速失败，避免长时间阻塞 MCP 调用方
const DefaultConnectTimeout = 5 * time.Second

// defaultRecoveryInterval 后台恢复监测的默认检测间隔
const defaultRecoveryInterval = 10 * time.Second

// failFastWindow 连接失败后的快速失败窗口
// 窗口内不再重复拨号，直接返回缓存的失败信息，避免每次工具调用都等待完整超时
const failFastWindow = 10 * time.Second

// lastFail 最近一次连接失败记录
type lastFail struct {
	at     time.Time // 失败时间
	reason string    // 失败原因（用于返回给调用方）
}

// PoolManager 管理所有数据库连接池
type PoolManager struct {
	drivers  map[string]Database                // 所有驱动实例（按ID存储）
	configs  map[string]DatabaseConfig          // 连接配置缓存
	registry map[DatabaseType]DriverConstructor // 驱动注册表
	mu       sync.RWMutex                       // 读写锁保护并发访问

	connectLocks map[string]*sync.Mutex // per-ID 建连串行锁（连接过程不持全局锁）
	lastFails    map[string]lastFail    // 最近连接失败记录（fail-fast 窗口）
}

// NewPoolManager 创建新的连接池管理器
func NewPoolManager() *PoolManager {
	pm := &PoolManager{
		drivers:      make(map[string]Database),
		configs:      make(map[string]DatabaseConfig),
		registry:     make(map[DatabaseType]DriverConstructor),
		connectLocks: make(map[string]*sync.Mutex),
		lastFails:    make(map[string]lastFail),
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
	if config.ConnectTimeout <= 0 {
		config.ConnectTimeout = int(DefaultConnectTimeout / time.Second) // 默认5秒拨号超时
	}

	pm.configs[config.ID] = config
	return nil
}

// GetOrCreatePool 获取或创建数据库驱动实例（统一接口）
// 连接过程不持全局锁：单库建连失败不会阻塞其他库的工具调用
// 失败后在 failFastWindow 内快速返回缓存错误，避免网络未就绪时每次调用都重新拨号
func (pm *PoolManager) GetOrCreatePool(ctx context.Context, id string) (Database, error) {
	// 快速路径：已有驱动且最近无失败记录时检查连接健康状态
	pm.mu.RLock()
	driver, hasDriver := pm.drivers[id]
	pm.mu.RUnlock()

	if hasDriver && !pm.inFailWindow(id) {
		if driver.IsConnected() {
			pm.clearFail(id)
			return driver, nil
		}
	}

	// fail-fast：失败窗口内直接返回缓存错误，不重复拨号
	if reason, ok := pm.failReason(id); ok {
		return nil, fmt.Errorf("数据库 %s 暂不可达（上次失败: %s），请稍后重试", id, reason)
	}

	// per-ID 串行建连：避免并发重复拨号
	lock := pm.getConnectLock(id)
	lock.Lock()
	defer lock.Unlock()

	// double-check：等待期间可能已被其他 goroutine 连上
	pm.mu.RLock()
	driver, hasDriver = pm.drivers[id]
	pm.mu.RUnlock()
	if hasDriver && driver.IsConnected() {
		pm.clearFail(id)
		return driver, nil
	}

	// double-check：等待期间可能已有失败记录（前一持锁者刚失败）
	if reason, ok := pm.failReason(id); ok {
		return nil, fmt.Errorf("数据库 %s 暂不可达（上次失败: %s），请稍后重试", id, reason)
	}

	// 获取配置
	config, exists := pm.GetConfig(id)
	if !exists {
		return nil, fmt.Errorf("连接配置 %s 不存在", id)
	}

	// 建连上下文强制限时（兜底，防调用方传入无 deadline 的 ctx）
	connectTimeout := time.Duration(config.ConnectTimeout) * time.Second
	if connectTimeout <= 0 {
		connectTimeout = DefaultConnectTimeout
	}
	connectCtx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()

	if hasDriver {
		// 已有驱动：连接断开，尝试重连
		if err := driver.Connect(connectCtx); err != nil {
			pm.recordFail(id, err)
			return nil, fmt.Errorf("重新连接失败: %s", err)
		}
		pm.clearFail(id)
		return driver, nil
	}

	// 确定实际驱动类型（处理协议兼容）
	actualType := GetActualDriverType(config)

	// 从全局注册表获取驱动构造函数
	constructor, exists := GetRegisteredDriver(actualType)
	if !exists {
		return nil, fmt.Errorf("数据库类型 %s 未注册驱动，请检查是否已导入驱动包", actualType)
	}

	// 创建驱动实例并建立连接
	newDriver := constructor(config)
	if err := newDriver.Connect(connectCtx); err != nil {
		pm.recordFail(id, err)
		return nil, fmt.Errorf("连接失败: %s", err)
	}

	pm.mu.Lock()
	pm.drivers[id] = newDriver
	pm.mu.Unlock()
	pm.clearFail(id)
	return newDriver, nil
}

// inFailWindow 判断指定ID是否处于失败快速返回窗口内
func (pm *PoolManager) inFailWindow(id string) bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	rec, exists := pm.lastFails[id]
	return exists && time.Since(rec.at) < failFastWindow
}

// failReason 返回失败窗口内的缓存失败原因（ok=false 表示无有效失败记录）
func (pm *PoolManager) failReason(id string) (string, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	rec, exists := pm.lastFails[id]
	if !exists || time.Since(rec.at) >= failFastWindow {
		return "", false
	}
	return rec.reason, true
}

// recordFail 记录一次连接失败
func (pm *PoolManager) recordFail(id string, err error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.lastFails[id] = lastFail{at: time.Now(), reason: err.Error()}
}

// clearFail 清除失败记录（连接恢复时调用）
func (pm *PoolManager) clearFail(id string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	delete(pm.lastFails, id)
}

// getConnectLock 获取指定ID的建连串行锁（不存在则创建）
func (pm *PoolManager) getConnectLock(id string) *sync.Mutex {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if lock, exists := pm.connectLocks[id]; exists {
		return lock
	}
	lock := &sync.Mutex{}
	pm.connectLocks[id] = lock
	return lock
}

// StartRecoveryMonitor 启动后台恢复监测 goroutine
// 首轮立即执行（兼作启动预热，不阻塞服务启动）；此后每 interval 一轮，
// 仅当存在连接失败记录时才对未连接的数据库并行重连（fail-fast + per-ID 建连锁）。
// 全部连接恢复后转入休眠：不再做周期性网络探测，直到出现新的失败记录后的下一轮自动恢复检测。
func (pm *PoolManager) StartRecoveryMonitor(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = defaultRecoveryInterval
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		// 首轮立即执行：兼作启动预热
		if pm.recoverOnce(ctx) {
			utils.GlobalLogger.Info("数据库恢复监测：首轮检测全部连接正常，转入休眠")
		}

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// 健康门控：无失败记录时跳过本轮，不做任何网络探测
				if !pm.hasFailRecords() {
					continue
				}
			}
			if pm.recoverOnce(ctx) {
				utils.GlobalLogger.Info("数据库恢复监测：全部连接恢复，转入休眠")
			}
		}
	}()
}

// recoverOnce 对所有未连接的数据库并行尝试重连；返回 true 表示全部已连接
func (pm *PoolManager) recoverOnce(ctx context.Context) bool {
	configs := pm.GetAllConfigs()
	var wg sync.WaitGroup
	var mu sync.Mutex
	healthy := true
	for id := range configs {
		// 已连接的驱动跳过（IsConnected 含网络 IO，仅在不健康轮次执行）
		if driver, ok := pm.GetDriver(id); ok && driver.IsConnected() {
			continue
		}
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			// 恢复检测必须真实拨号：先清除失败缓存，绕过 fail-fast 窗口，
			// 否则监测会被 10s 失败缓存吞掉而无法探测网络恢复
			pm.clearFail(id)
			if _, err := pm.GetOrCreatePool(ctx, id); err != nil {
				mu.Lock()
				healthy = false
				mu.Unlock()
				utils.GlobalLogger.Warn("数据库恢复监测：连接失败 [ID=%s]: %s", id, err)
			} else {
				utils.GlobalLogger.Info("数据库恢复监测：连接成功 [ID=%s]", id)
			}
		}(id)
	}
	wg.Wait()
	return healthy
}

// hasFailRecords 是否存在未清除的连接失败记录（健康门控，O(1)、无网络 IO）
func (pm *PoolManager) hasFailRecords() bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return len(pm.lastFails) > 0
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
