package database_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/relaxyabc/mcp-dbquery/src/database"
)

// fakeDriver 测试用驱动桩：可控连接成败、可统计 Connect 调用次数
type fakeDriver struct {
	id           string
	dbType       database.DatabaseType
	connectErr   error         // 非nil时 Connect 返回该错误
	failNext     atomic.Int32  // >0 时消耗一次并返回 connectErr（模拟"先失败后恢复"）
	connectDelay time.Duration // 模拟建连耗时
	connectCalls atomic.Int64  // Connect 调用次数
	connected    atomic.Bool   // 当前连接状态
}

// Connect 模拟建连：统计调用次数，按配置返回成功或失败
func (f *fakeDriver) Connect(ctx context.Context) error {
	f.connectCalls.Add(1)
	if f.failNext.Load() > 0 {
		f.failNext.Add(-1)
		f.connected.Store(false)
		return f.connectErr
	}
	if f.connectErr != nil {
		f.connected.Store(false)
		return f.connectErr
	}
	if f.connectDelay > 0 {
		select {
		case <-time.After(f.connectDelay):
		case <-ctx.Done():
			f.connected.Store(false)
			return ctx.Err()
		}
	}
	f.connected.Store(true)
	return nil
}

// Close 模拟关闭连接
func (f *fakeDriver) Close(ctx context.Context) error {
	f.connected.Store(false)
	return nil
}

// IsConnected 返回当前连接状态
func (f *fakeDriver) IsConnected() bool { return f.connected.Load() }

// GetType 返回驱动类型
func (f *fakeDriver) GetType() database.DatabaseType { return f.dbType }

// GetID 返回连接标识
func (f *fakeDriver) GetID() string { return f.id }

func (f *fakeDriver) ExecuteQuery(ctx context.Context, query string, limit int, timeout time.Duration) (*database.QueryResult, error) {
	return nil, errors.New("测试桩不支持")
}

func (f *fakeDriver) ExecuteSelectQuery(ctx context.Context, query string, limit int) (*database.QueryResult, error) {
	return nil, errors.New("测试桩不支持")
}

func (f *fakeDriver) ExecuteFind(ctx context.Context, collection string, filter map[string]interface{}, limit int) (*database.QueryResult, error) {
	return nil, errors.New("测试桩不支持")
}

func (f *fakeDriver) ExecuteAggregate(ctx context.Context, collection string, pipeline []map[string]interface{}, limit int) (*database.QueryResult, error) {
	return nil, errors.New("测试桩不支持")
}

func (f *fakeDriver) ExecuteCount(ctx context.Context, collection string, filter map[string]interface{}) (int64, error) {
	return 0, errors.New("测试桩不支持")
}

func (f *fakeDriver) ExecuteDistinct(ctx context.Context, collection string, fieldName string, filter map[string]interface{}) ([]interface{}, error) {
	return nil, errors.New("测试桩不支持")
}

func (f *fakeDriver) GetSchema(ctx context.Context, tableName string) (*database.SchemaMetadata, error) {
	return nil, errors.New("测试桩不支持")
}

func (f *fakeDriver) GetIndexes(ctx context.Context, tableName string) (*database.IndexListMetadata, error) {
	return nil, errors.New("测试桩不支持")
}

func (f *fakeDriver) ListTables(ctx context.Context) ([]string, error) {
	return nil, errors.New("测试桩不支持")
}

func (f *fakeDriver) ValidateQuery(query string) error { return nil }

// TestGetOrCreatePool_ConnectFailureFailFast 连接失败后，失败窗口内的后续调用应快速返回且不再拨号
func TestGetOrCreatePool_ConnectFailureFailFast(t *testing.T) {
	pm := database.NewPoolManager()
	if err := pm.RegisterConfig(database.DatabaseConfig{
		ID:             "bad",
		Type:           database.DatabaseTypeMySQL,
		ConnectTimeout: 1,
	}); err != nil {
		t.Fatalf("注册配置失败: %v", err)
	}

	// 预置一个连接总是失败的驱动实例
	bad := &fakeDriver{id: "bad", dbType: database.DatabaseTypeMySQL, connectErr: errors.New("dial tcp: connection refused")}
	pm.SetMySQLDriver("bad", bad)

	ctx := context.Background()

	// 第一次调用：执行拨号并失败
	_, err := pm.GetOrCreatePool(ctx, "bad")
	if err == nil || !strings.Contains(err.Error(), "重新连接失败") {
		t.Fatalf("第一次调用应返回重新连接失败, got: %v", err)
	}
	if got := bad.connectCalls.Load(); got != 1 {
		t.Fatalf("第一次调用应拨号1次, got %d", got)
	}

	// 第二次调用（失败窗口内）：应快速失败且不再拨号
	start := time.Now()
	_, err = pm.GetOrCreatePool(ctx, "bad")
	elapsed := time.Since(start)
	if err == nil || !strings.Contains(err.Error(), "暂不可达") {
		t.Fatalf("失败窗口内应快速返回暂不可达, got: %v", err)
	}
	if got := bad.connectCalls.Load(); got != 1 {
		t.Fatalf("失败窗口内不应重新拨号, 拨号次数=%d", got)
	}
	if elapsed > 200*time.Millisecond {
		t.Fatalf("快速失败耗时过长: %v", elapsed)
	}
}

// TestGetOrCreatePool_ConcurrencySingleConnect 并发调用同一ID时只应拨号一次（per-ID 串行 + double-check）
func TestGetOrCreatePool_ConcurrencySingleConnect(t *testing.T) {
	pm := database.NewPoolManager()
	if err := pm.RegisterConfig(database.DatabaseConfig{
		ID:             "slow",
		Type:           database.DatabaseTypeMySQL,
		ConnectTimeout: 5,
	}); err != nil {
		t.Fatalf("注册配置失败: %v", err)
	}

	// 预置一个建连耗时100ms的驱动实例
	slow := &fakeDriver{id: "slow", dbType: database.DatabaseTypeMySQL, connectDelay: 100 * time.Millisecond}
	pm.SetMySQLDriver("slow", slow)

	ctx := context.Background()
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := pm.GetOrCreatePool(ctx, "slow"); err != nil {
				t.Errorf("并发调用失败: %v", err)
			}
		}()
	}
	wg.Wait()

	if got := slow.connectCalls.Load(); got != 1 {
		t.Fatalf("并发下应只拨号1次, got %d", got)
	}
}

// TestGetOrCreatePool_ReuseDriver 连接成功后应复用同一驱动实例，不再重复拨号
func TestGetOrCreatePool_ReuseDriver(t *testing.T) {
	pm := database.NewPoolManager()
	// GetOrCreatePool 使用全局驱动注册表
	database.RegisterDriver(database.DatabaseType("fake-succ"), func(config database.DatabaseConfig) database.Database {
		return &fakeDriver{id: config.ID, dbType: config.Type}
	})
	if err := pm.RegisterConfig(database.DatabaseConfig{
		ID:   "ok",
		Type: database.DatabaseType("fake-succ"),
	}); err != nil {
		t.Fatalf("注册配置失败: %v", err)
	}

	// RegisterConfig 应补全默认拨号超时
	cfg, ok := pm.GetConfig("ok")
	if !ok {
		t.Fatal("配置应已注册")
	}
	if cfg.ConnectTimeout != 5 {
		t.Fatalf("默认 connect_timeout 应为5秒, got %d", cfg.ConnectTimeout)
	}

	ctx := context.Background()
	d1, err := pm.GetOrCreatePool(ctx, "ok")
	if err != nil {
		t.Fatalf("首次建连应成功: %v", err)
	}
	d2, err := pm.GetOrCreatePool(ctx, "ok")
	if err != nil {
		t.Fatalf("复用应成功: %v", err)
	}
	if d1 != d2 {
		t.Fatal("应返回同一驱动实例")
	}
}

// TestGetOrCreatePool_UnknownDriverType 未注册驱动类型应返回明确错误
func TestGetOrCreatePool_UnknownDriverType(t *testing.T) {
	pm := database.NewPoolManager()
	if err := pm.RegisterConfig(database.DatabaseConfig{
		ID:   "no-driver",
		Type: database.DatabaseType("no-such-type"),
	}); err != nil {
		t.Fatalf("注册配置失败: %v", err)
	}

	_, err := pm.GetOrCreatePool(context.Background(), "no-driver")
	if err == nil || !strings.Contains(err.Error(), "未注册驱动") {
		t.Fatalf("应返回未注册驱动错误, got: %v", err)
	}
}

// TestStartRecoveryMonitor_AllHealthySkipsDials 全部健康时监测应休眠，不发起任何拨号
func TestStartRecoveryMonitor_AllHealthySkipsDials(t *testing.T) {
	pm := database.NewPoolManager()
	if err := pm.RegisterConfig(database.DatabaseConfig{
		ID:             "healthy",
		Type:           database.DatabaseTypeMySQL,
		ConnectTimeout: 1,
	}); err != nil {
		t.Fatalf("注册配置失败: %v", err)
	}
	good := &fakeDriver{id: "healthy", dbType: database.DatabaseTypeMySQL}
	good.connected.Store(true) // 预置为已连接：模拟真实路径（驱动仅在连接成功后入池）
	pm.SetMySQLDriver("healthy", good)

	pm.StartRecoveryMonitor(context.Background(), 20*time.Millisecond)

	// 首轮与多个检测周期内：已连接的驱动不应被重新拨号（无失败记录 → 健康门控跳过）
	time.Sleep(150 * time.Millisecond)
	if got := good.connectCalls.Load(); got != 0 {
		t.Fatalf("健康状态下不应拨号, 拨号次数=%d", got)
	}
}

// TestStartRecoveryMonitor_RecoversFailedDatabase 存在失败记录时按周期重连，恢复后清除失败状态
func TestStartRecoveryMonitor_RecoversFailedDatabase(t *testing.T) {
	pm := database.NewPoolManager()
	if err := pm.RegisterConfig(database.DatabaseConfig{
		ID:             "flaky",
		Type:           database.DatabaseTypeMySQL,
		ConnectTimeout: 1,
	}); err != nil {
		t.Fatalf("注册配置失败: %v", err)
	}
	flaky := &fakeDriver{id: "flaky", dbType: database.DatabaseTypeMySQL, connectErr: errors.New("dial tcp: i/o timeout")}
	flaky.failNext.Store(1)
	pm.SetMySQLDriver("flaky", flaky)

	ctx := context.Background()
	// 预置一次失败：进入失败缓存
	if _, err := pm.GetOrCreatePool(ctx, "flaky"); err == nil {
		t.Fatal("预置失败应返回错误")
	}
	if got := flaky.connectCalls.Load(); got != 1 {
		t.Fatalf("预置失败应拨号1次, got %d", got)
	}

	// 模拟网络恢复：清除永久错误（在启动监测前修改，此后仅监测 goroutine 读取，无竞态）
	flaky.connectErr = nil

	// 启动监测：下一轮应重连成功
	pm.StartRecoveryMonitor(ctx, 20*time.Millisecond)

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if driver, err := pm.GetOrCreatePool(ctx, "flaky"); err == nil && driver != nil {
			if got := flaky.connectCalls.Load(); got < 2 {
				t.Fatalf("恢复应由监测完成拨号, 拨号次数=%d", got)
			}
			return // 恢复成功：失败记录已清除，后续调用直接复用驱动
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("监测未在3秒内恢复连接")
}
