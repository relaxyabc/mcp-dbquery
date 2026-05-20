package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/relaxyabc/mcp-dbquery/src/utils"
)

// AuthMiddleware API密钥认证中间件
// 宪章原则 II 要求：所有HTTP请求必须验证API密钥（/health除外）
type AuthMiddleware struct {
	authManager *AuthManager
}

// NewAuthMiddleware 创建认证中间件
func NewAuthMiddleware(authManager *AuthManager) *AuthMiddleware {
	return &AuthMiddleware{
		authManager: authManager,
	}
}

// Middleware 返回中间件函数
func (am *AuthMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 健康检查端点无需认证
		if r.URL.Path == "/health" || r.URL.Path == "/healthz" || r.URL.Path == "/ready" {
			next.ServeHTTP(w, r)
			return
		}

		// OAuth discovery 端点无需认证（MCP SDK 自动发现流程）
		// 包括 /.well-known/* 和 /mcp/.well-known/* 路径变体
		if strings.HasPrefix(r.URL.Path, "/.well-known/") ||
			strings.HasPrefix(r.URL.Path, "/mcp/.well-known/") ||
			r.URL.Path == "/register" {
			next.ServeHTTP(w, r)
			return
		}

		// MCP端点：支持多种认证方式
		// 1. X-API-Key header（推荐）
		// 2. Authorization Bearer header
		// 3. URL query parameter (api_key) - 用于某些MCP客户端
		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" {
			// 尝试从Authorization头获取
			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				apiKey = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}
		if apiKey == "" {
			// 尝试从URL查询参数获取（兼容某些客户端）
			apiKey = r.URL.Query().Get("api_key")
		}

		// 验证API密钥
		if apiKey == "" {
			am.respondUnauthorized(w, "缺少API密钥，请在X-API-Key头中提供")
			return
		}

		if !am.authManager.Validate(apiKey) && !am.authManager.ValidateByHash(apiKey) {
			am.respondUnauthorized(w, "无效的API密钥")
			return
		}

		// 将认证信息添加到请求上下文
		ctx := context.WithValue(r.Context(), "authenticated", true)
		ctx = context.WithValue(ctx, "api_key_provided", apiKey != "")

		// 继续处理请求
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// respondUnauthorized 返回401未授权响应
func (am *AuthMiddleware) respondUnauthorized(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)

	response := map[string]interface{}{
		"success": false,
		"error": map[string]interface{}{
			"code":    "AUTH_FAILED",
			"message": message,
		},
	}

	json.NewEncoder(w).Encode(response)

	utils.GlobalLogger.Error("[AUTH_FAILED] %s", message)
}

// LoggingMiddleware 请求日志中间件
type LoggingMiddleware struct{}

// NewLoggingMiddleware 创建日志中间件
func NewLoggingMiddleware() *LoggingMiddleware {
	return &LoggingMiddleware{}
}

// Middleware 返回中间件函数
func (lm *LoggingMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 记录请求开始
		utils.GlobalLogger.Info("请求开始 [方法=%s] [路径=%s] [来源=%s]",
			r.Method, r.URL.Path, r.RemoteAddr)

		// 创建响应包装器以记录状态码
		wrapped := &responseWriterWrapper{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// 处理请求
		next.ServeHTTP(wrapped, r)

		// 记录请求完成
		utils.GlobalLogger.Info("请求完成 [方法=%s] [路径=%s] [状态=%d]",
			r.Method, r.URL.Path, wrapped.statusCode)
	})
}

// responseWriterWrapper 响应包装器（用于捕获状态码）
type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader 捕获状态码
func (w *responseWriterWrapper) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

// CORSMiddleware 跨域请求中间件（可选）
type CORSMiddleware struct {
	allowedOrigins []string
}

// NewCORSMiddleware 创建CORS中间件
func NewCORSMiddleware(allowedOrigins []string) *CORSMiddleware {
	return &CORSMiddleware{
		allowedOrigins: allowedOrigins,
	}
}

// Middleware 返回中间件函数
func (cm *CORSMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// 检查是否允许该来源
		allowed := false
		for _, allowedOrigin := range cm.allowedOrigins {
			if allowedOrigin == "*" || allowedOrigin == origin {
				allowed = true
				break
			}
		}

		if allowed {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-API-Key, Authorization")
		}

		// 处理OPTIONS预检请求
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// ChainMiddleware 链式组合多个中间件
func ChainMiddleware(handler http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}

// IsAuthenticated 检查请求是否已认证
func IsAuthenticated(ctx context.Context) bool {
	auth, ok := ctx.Value("authenticated").(bool)
	return ok && auth
}

// RequireAuth 验证认证状态（辅助函数）
func RequireAuth(ctx context.Context) error {
	if !IsAuthenticated(ctx) {
		return fmt.Errorf("请求未认证")
	}
	return nil
}
