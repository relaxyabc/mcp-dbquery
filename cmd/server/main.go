package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/urfave/cli/v3"

	"github.com/relaxyabc/mcp-dbquery/src/api"
	"github.com/relaxyabc/mcp-dbquery/src/database"
	// 驱动包导入（触发 init() 自注册）
	_ "github.com/relaxyabc/mcp-dbquery/src/database/clickhouse"
	_ "github.com/relaxyabc/mcp-dbquery/src/database/doris"
	_ "github.com/relaxyabc/mcp-dbquery/src/database/mongodb"
	_ "github.com/relaxyabc/mcp-dbquery/src/database/mysql"
	_ "github.com/relaxyabc/mcp-dbquery/src/database/oracle"
	_ "github.com/relaxyabc/mcp-dbquery/src/database/postgres"
	_ "github.com/relaxyabc/mcp-dbquery/src/database/sqlite"
	_ "github.com/relaxyabc/mcp-dbquery/src/database/sqlserver"
	"github.com/relaxyabc/mcp-dbquery/src/mcp"
	"github.com/relaxyabc/mcp-dbquery/src/server"
	"github.com/relaxyabc/mcp-dbquery/src/utils"
)

// 版本信息
const (
	Version   = "1.0.0"
	BuildDate = "2025-05-20"
)

// main 服务器入口函数
func main() {
	cmd := &cli.Command{
		Name:    "db-tools",
		Usage:   "MCP数据库查询工具 - 安全只读数据库查询服务",
		Version: Version,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Value:   "configs/config.yaml",
				Usage:   "配置文件路径",
				Sources: cli.EnvVars("CONFIG_PATH"),
			},
			&cli.StringFlag{
				Name:    "transport",
				Aliases: []string{"t"},
				Value:   "stdio",
				Usage:   "传输模式: stdio/s 或 http/h",
			},
		},
		Action: runApp,
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "错误: %s\n", err)
		os.Exit(1)
	}
}

// runApp 主应用逻辑
func runApp(ctx context.Context, cmd *cli.Command) error {
	// 获取参数
	configPath := cmd.String("config")
	transportParam := cmd.String("transport")

	// 规范化传输模式
	transportMode := normalizeTransport(transportParam)

	// 加载配置
	configLoader := server.NewConfigLoader(configPath)
	config, err := configLoader.Load()

	// STDIO模式：禁止任何stdout输出（只允许MCP JSON-RPC）
	// HTTP模式：可以输出启动信息
	if transportMode == server.TransportModeHTTP {
		fmt.Printf("MCP数据库查询工具 v%s\n", Version)
		fmt.Printf("构建日期: %s\n", BuildDate)
		fmt.Println("=====================================")
	}

	// 配置加载失败处理
	if err != nil {
		// STDIO模式：错误信息写到stderr（不破坏stdout的JSON-RPC）
		fmt.Fprintf(os.Stderr, "配置加载失败: %s\n", err)
		os.Exit(1)
	}

	// 设置日志级别
	utils.SetLogLevel(utils.LogLevel(config.Logging.Level))

	// STDIO模式：日志写到stderr
	if transportMode == server.TransportModeStdio {
		utils.SetLogOutput(os.Stderr)
	}

	utils.GlobalLogger.Info("配置加载成功 [文件=%s]", configPath)

	// 创建连接池管理器
	poolManager := database.NewPoolManager()
	utils.GlobalLogger.Info("连接池管理器已初始化")

	// 注册数据库配置
	dbConfigs := configLoader.GetDatabaseConfigs()
	for id, dbConfig := range dbConfigs {
		if err := poolManager.RegisterConfig(dbConfig); err != nil {
			utils.GlobalLogger.Error("注册数据库配置失败 [ID=%s]: %s", id, err)
			continue
		}
		utils.GlobalLogger.Info("注册数据库配置 [ID=%s] [类型=%s] [主机=%s:%d]",
			id, dbConfig.Type, dbConfig.Host, dbConfig.Port)
	}

	// 连接所有数据库（启动时验证连接）
	utils.GlobalLogger.Info("开始连接所有数据库...")
	connectErrors := connectAllDatabases(ctx, poolManager, dbConfigs)
	for id, err := range connectErrors {
		utils.GlobalLogger.Error("数据库连接失败 [ID=%s]: %s", id, err)
	}
	utils.GlobalLogger.Info("数据库连接完成 [成功=%d] [失败=%d]",
		len(dbConfigs)-len(connectErrors), len(connectErrors))

	// HTTP模式需要认证管理器
	var authManager *server.AuthManager
	if transportMode == server.TransportModeHTTP {
		authManager = server.NewAuthManager()
		authManager.AddKeyFromString(config.Server.APIKey, "primary")
		utils.GlobalLogger.Info("认证管理器已初始化 [API密钥数量=%d]", authManager.Count())
	}

	// 创建MCP服务器
	mcpServer := mcp.NewMCPServer(poolManager)
	mcpServer.RegisterAllTools()
	utils.GlobalLogger.Info("MCP服务器已初始化 [工具数量=%d]", len(mcpServer.ListTools()))

	// 输出传输模式信息
	utils.GlobalLogger.Info("传输模式: %s", transportMode)

	// 根据传输模式启动服务器
	if transportMode == server.TransportModeStdio {
		// STDIO模式：直接运行，不需要HTTP服务器
		utils.GlobalLogger.Info("以STDIO模式运行 (Claude CLI集成)")
		runSTDIOMode(ctx, poolManager, mcpServer)
	} else {
		// HTTP模式：创建HTTP服务器
		utils.GlobalLogger.Info("以HTTP模式运行 (VS Code MCP集成)")
		httpServer := createHTTPServer(config, poolManager, authManager, mcpServer)
		startServer(httpServer, config.Server.Host, config.Server.Port)
		waitForShutdown(httpServer, poolManager)
	}

	utils.GlobalLogger.Info("服务器已停止")
	return nil
}

// normalizeTransport 规范化传输模式简写
func normalizeTransport(value string) server.TransportMode {
	mode := strings.ToLower(value)
	switch mode {
	case "s", "stdio":
		return server.TransportModeStdio
	case "h", "http":
		return server.TransportModeHTTP
	default:
		return server.TransportModeStdio
	}
}

// runSTDIOMode 以STDIO模式运行服务器
func runSTDIOMode(ctx context.Context, poolManager *database.PoolManager, mcpServer *mcp.MCPServer) {
	// STDIO模式不需要认证中间件
	// 子进程由MCP客户端启动，隐式信任

	// 运行STDIO服务器（阻塞直到stdin关闭）
	if err := mcpServer.RunSTDIO(ctx); err != nil {
		utils.GlobalLogger.Error("STDIO服务器运行失败: %s", err)
		os.Exit(1)
	}

	// STDIO关闭后，清理连接池
	utils.GlobalLogger.Info("关闭所有数据库连接池...")
	if err := poolManager.CloseAll(ctx); err != nil {
		utils.GlobalLogger.Error("连接池关闭失败: %s", err)
	}
}

// connectAllDatabases 连接所有数据库（启动时验证）
// 驱动已通过 init() 自注册到全局注册表，无需手动注册
func connectAllDatabases(ctx context.Context, poolManager *database.PoolManager, dbConfigs map[string]database.DatabaseConfig) map[string]error {
	errors := make(map[string]error)

	for id, config := range dbConfigs {
		// 使用全局注册表获取驱动
		_, err := poolManager.GetOrCreatePool(ctx, id)
		if err != nil {
			errors[id] = err
		} else {
			utils.GlobalLogger.Info("数据库连接成功 [ID=%s] [类型=%s]", id, config.Type)
		}
	}

	return errors
}

// createHTTPServer 创建HTTP服务器
func createHTTPServer(config *server.Config, poolManager *database.PoolManager, authManager *server.AuthManager, mcpServer *mcp.MCPServer) *http.Server {
	// 创建路由器
	mux := http.NewServeMux()

	// 注册路由
	// 健康检查端点（无需认证）
	healthHandler := api.NewHealthHandler(Version, poolManager)
	mux.HandleFunc("/health", healthHandler.ServeHTTP)
	mux.HandleFunc("/healthz", api.HealthzHandler)
	mux.HandleFunc("/ready", api.NewReadyHandler(healthHandler).ServeHTTP)

	// OAuth discovery 端点（MCP SDK 自动发现）
	// RFC 9728: oauth-protected-resource 返回资源元数据（空 authorization_servers 表示不使用 OAuth）
	// RFC 8414: oauth-authorization-server 和 openid-configuration 返回 404（OAuth 服务器不存在）
	mux.HandleFunc("/.well-known/oauth-protected-resource", handleOAuthProtectedResource)
	mux.HandleFunc("/.well-known/oauth-protected-resource/", handleOAuthProtectedResource)
	mux.HandleFunc("/.well-known/oauth-authorization-server", handleOAuthAuthorizationServer404)
	mux.HandleFunc("/.well-known/oauth-authorization-server/", handleOAuthAuthorizationServer404)
	mux.HandleFunc("/.well-known/openid-configuration", handleOAuthAuthorizationServer404)
	mux.HandleFunc("/.well-known/openid-configuration/", handleOAuthAuthorizationServer404)
	mux.HandleFunc("/mcp/.well-known/", handleOAuthProtectedResource)

	// OAuth register 端点（MCP SDK 动态客户端注册）
	mux.HandleFunc("/register", handleOAuthRegister)

	// MCP端点（使用StreamableHTTP处理器）
	mcpHandler := mcpServer.NewHTTPHandler()
	mux.Handle("/mcp", mcpHandler)
	mux.Handle("/mcp/", mcpHandler)

	// 工具列表端点（需要认证）
	mux.HandleFunc("/api/tools", func(w http.ResponseWriter, r *http.Request) {
		tools := mcpServer.ListTools()
		api.RespondSuccessWithData(w, map[string]interface{}{
			"tools":      tools,
			"tool_count": len(tools),
		})
	})

	// 创建中间件链
	authMiddleware := server.NewAuthMiddleware(authManager)
	loggingMiddleware := server.NewLoggingMiddleware()

	// 应用中间件
	handler := server.ChainMiddleware(mux,
		loggingMiddleware.Middleware,
		authMiddleware.Middleware,
	)

	// 创建HTTP服务器
	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", config.Server.Host, config.Server.Port),
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	return srv
}

// startServer 启动HTTP服务器
func startServer(srv *http.Server, host string, port int) {
	go func() {
		utils.GlobalLogger.Info("HTTP服务器启动 [地址=%s:%d]", host, port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			utils.GlobalLogger.Error("HTTP服务器启动失败: %s", err)
			os.Exit(1)
		}
	}()
}

// handleOAuthProtectedResource 处理 OAuth Protected Resource Metadata 请求 (RFC 9728)
// 返回资源元数据，空 authorization_servers 表示不使用 OAuth 认证
// MCP SDK 读取此响应后会知道不需要 OAuth，回退到其他认证方式（如 X-API-Key）
func handleOAuthProtectedResource(w http.ResponseWriter, r *http.Request) {
	utils.GlobalLogger.Info("OAuth protected resource 请求 [路径=%s]", r.URL.Path)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// 动态构建 resource URL（使用请求的 host）
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	resourceURL := fmt.Sprintf("%s://%s/mcp", scheme, r.Host)

	// RFC 9728 格式：空的 authorization_servers 表示资源不需要 OAuth
	w.Write([]byte(fmt.Sprintf(`{
	"resource": "%s",
	"authorization_servers": [],
	"scopes_supported": [],
	"bearer_methods_supported": ["header"],
	"resource_documentation": "%s://%s/health"
}`, resourceURL, scheme, r.Host)))
}

// handleOAuthAuthorizationServer404 处理 OAuth Authorization Server Metadata 请求 (RFC 8414)
// 返回 404 表示 OAuth Authorization Server 不存在
// SDK 在看到 oauth-protected-resource 的空 authorization_servers 后不应再请求此端点
func handleOAuthAuthorizationServer404(w http.ResponseWriter, r *http.Request) {
	utils.GlobalLogger.Info("OAuth authorization server 请求 [路径=%s] - 返回404（不支持OAuth）", r.URL.Path)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte(`{"error":"not_found","error_description":"OAuth authorization server not available. This server uses X-API-Key authentication."}`))
}

// handleOAuthRegister 处理 OAuth 动态客户端注册请求
// MCP SDK OAuth 流程的一部分，返回错误表示不支持动态注册
func handleOAuthRegister(w http.ResponseWriter, r *http.Request) {
	utils.GlobalLogger.Info("OAuth register 请求 [路径=%s]", r.URL.Path)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte(`{"error":"invalid_request","error_description":"Dynamic client registration not supported, use X-API-Key header for authentication"}`))
}

// waitForShutdown 等待关闭信号并优雅关闭
// 宪章要求FR-021：优雅关闭，最多30秒等待正在处理的请求
func waitForShutdown(srv *http.Server, poolManager *database.PoolManager) {
	// 创建信号通道
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// 等待信号
	sig := <-quit
	utils.GlobalLogger.Info("收到关闭信号: %s", sig)

	// 创建关闭上下文（30秒超时）
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 优雅关闭HTTP服务器
	utils.GlobalLogger.Info("开始优雅关闭HTTP服务器...")
	if err := srv.Shutdown(ctx); err != nil {
		utils.GlobalLogger.Error("HTTP服务器关闭失败: %s", err)
	}

	// 关闭所有数据库连接池
	utils.GlobalLogger.Info("关闭所有数据库连接池...")
	if err := poolManager.CloseAll(ctx); err != nil {
		utils.GlobalLogger.Error("连接池关闭失败: %s", err)
	}

	utils.GlobalLogger.Info("优雅关闭完成")
}
