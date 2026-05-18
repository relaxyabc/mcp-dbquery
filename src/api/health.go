package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/relaxyabc/mcp-dbquery/src/utils"
)

// HealthHandler 健康检查处理器
// 宪章要求：健康检查端点无需认证（FR-022）
type HealthHandler struct {
	version     string      // 服务器版本
	poolManager interface{} // 连接池管理器引用（实际类型为*database.PoolManager）
	startTime   time.Time   // 服务器启动时间
}

// NewHealthHandler 创建健康检查处理器
func NewHealthHandler(version string, poolManager interface{}) *HealthHandler {
	return &HealthHandler{
		version:     version,
		poolManager: poolManager,
		startTime:   time.Now(),
	}
}

// ServeHTTP 处理健康检查请求
func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 只允许GET请求
	if r.Method != "GET" {
		RespondBadRequest(w, "仅支持GET请求")
		return
	}

	// 构建响应
	response := HealthResponse{
		Status:      h.determineStatus(),
		Timestamp:   time.Now().Format(time.RFC3339),
		Version:     h.version,
		Connections: h.getConnectionStatus(),
	}

	// 添加运行时长信息
	metadata := map[string]interface{}{
		"uptime_seconds": int(time.Since(h.startTime).Seconds()),
		"start_time":     h.startTime.Format(time.RFC3339),
	}

	RespondSuccessWithMetadata(w, response, metadata)

	utils.GlobalLogger.Info("健康检查完成 [状态=%s]", response.Status)
}

// determineStatus 确定服务器健康状态
func (h *HealthHandler) determineStatus() string {
	// 获取连接池状态
	connStatus := h.getConnectionStatus()

	// 检查是否有连接
	if len(connStatus) == 0 {
		return "healthy" // 无配置连接也视为健康（可能未配置数据库）
	}

	// 检查连接状态
	hasError := false
	hasConnected := false

	for _, status := range connStatus {
		statusStr, ok := status.(string)
		if ok {
			switch statusStr {
			case "connected", "idle", "active":
				hasConnected = true
			case "error":
				hasError = true
			}
		}
	}

	// 确定状态
	if hasError {
		return "degraded" // 有错误连接，降级状态
	}

	if hasConnected {
		return "healthy" // 有活跃连接
	}

	return "healthy" // 默认健康
}

// getConnectionStatus 获取连接池状态
func (h *HealthHandler) getConnectionStatus() map[string]interface{} {
	// 实际实现需要访问PoolManager
	// 这里返回模拟数据，实际在集成时实现
	status := make(map[string]interface{})

	// TODO: 从实际PoolManager获取状态
	// if pm, ok := h.poolManager.(*database.PoolManager); ok {
	//     status = pm.GetPoolStatus()
	// }

	return status
}

// HealthzHandler 简化的健康检查处理器（Kubernetes风格）
func HealthzHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}

// ReadyHandler 就绪检查处理器（检查所有依赖是否就绪）
type ReadyHandler struct {
	healthHandler *HealthHandler
}

// NewReadyHandler 创建就绪检查处理器
func NewReadyHandler(healthHandler *HealthHandler) *ReadyHandler {
	return &ReadyHandler{
		healthHandler: healthHandler,
	}
}

// ServeHTTP 处理就绪检查请求
func (rh *ReadyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		RespondBadRequest(w, "仅支持GET请求")
		return
	}

	// 检查健康状态
	status := rh.healthHandler.determineStatus()

	// 只有healthy状态才返回200
	if status == "healthy" {
		RespondSuccessWithData(w, map[string]interface{}{
			"ready":   true,
			"status":  status,
			"message": "服务器就绪，可接受请求",
		})
	} else {
		RespondSuccessWithData(w, map[string]interface{}{
			"ready":   false,
			"status":  status,
			"message": "服务器未完全就绪",
		})
	}
}
