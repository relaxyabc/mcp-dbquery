package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/relaxyabc/mcp-dbquery/src/database"
	"github.com/relaxyabc/mcp-dbquery/src/mcp/handlers"
	"github.com/relaxyabc/mcp-dbquery/src/utils"
)

// MCPServer MCP服务器包装器
// 使用官方Go MCP SDK实现协议支持
type MCPServer struct {
	server      *mcp.Server            // MCP服务器实例
	poolManager *database.PoolManager  // 连接池管理器
	authManager interface{}            // 认证管理器引用
	tools       map[string]ToolHandler // 工具处理器映射
}

// ToolHandler 工具处理器函数类型
type ToolHandler func(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error)

// NewMCPServer 创建MCP服务器
func NewMCPServer(poolManager *database.PoolManager) *MCPServer {
	// 使用官方SDK创建MCP服务器
	s := mcp.NewServer(
		&mcp.Implementation{
			Name:    "db-query-tool", // 服务器名称
			Version: "1.0.0",         // 版本
		},
		nil, // 使用默认选项
	)

	return &MCPServer{
		server:      s,
		poolManager: poolManager,
		tools:       make(map[string]ToolHandler),
	}
}

// RegisterTool 注册MCP工具
func (ms *MCPServer) RegisterTool(toolDef ToolDefinition, handler ToolHandler) {
	// 使用SDK的Tool结构定义工具
	tool := &mcp.Tool{
		Name:        toolDef.Name,
		Description: toolDef.Description,
		InputSchema: toolDef.InputSchema,
	}

	// 注册到MCP服务器
	ms.server.AddTool(tool, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		utils.GlobalLogger.Info("执行工具 [工具名=%s]", toolDef.Name)

		// 解析参数
		var args map[string]interface{}
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			utils.GlobalLogger.Error("[参数解析失败] %s - %s", toolDef.Name, err)
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("参数解析失败: %s", err)}},
			}, nil
		}

		result, err := handler(ctx, args)
		if err != nil {
			utils.GlobalLogger.Error("[工具执行失败] %s - %s", toolDef.Name, err)
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("工具执行失败: %s", err)}},
			}, nil
		}

		utils.GlobalLogger.Info("工具执行成功 [工具名=%s]", toolDef.Name)
		return result, nil
	})

	// 存储处理器引用
	ms.tools[toolDef.Name] = handler

	utils.GlobalLogger.Info("注册MCP工具: %s", toolDef.Name)
}

// RegisterAllTools 注册所有预定义工具
func (ms *MCPServer) RegisterAllTools() {
	// 注册MySQL查询工具
	mysqlQueryTool := GetToolByName("query_mysql_data")
	if mysqlQueryTool != nil {
		ms.RegisterTool(*mysqlQueryTool, handlers.MySQLQueryHandler(ms.poolManager))
	}

	// 注册MongoDB查询工具
	mongoQueryTool := GetToolByName("query_mongodb_data")
	if mongoQueryTool != nil {
		ms.RegisterTool(*mongoQueryTool, handlers.MongoDBQueryHandler(ms.poolManager))
	}

	// 注册Schema工具
	schemaTool := GetToolByName("get_schema")
	if schemaTool != nil {
		ms.RegisterTool(*schemaTool, handlers.SchemaHandler(ms.poolManager))
	}

	// 注册Index工具
	indexTool := GetToolByName("get_indexes")
	if indexTool != nil {
		ms.RegisterTool(*indexTool, handlers.IndexesHandler(ms.poolManager))
	}

	// 注册ListTables工具
	listTablesTool := GetToolByName("list_tables")
	if listTablesTool != nil {
		ms.RegisterTool(*listTablesTool, handlers.ListTablesHandler(ms.poolManager))
	}

	utils.GlobalLogger.Info("所有MCP工具已注册 [数量=%d]", len(ms.tools))
}

// GetServer 获取底层MCP服务器实例
func (ms *MCPServer) GetServer() *mcp.Server {
	return ms.server
}

// NewHTTPHandler 创建HTTP处理器
// 使用官方SDK的StreamableHTTP传输
// 使用 Stateless 模式以避免 session 管理复杂性
func (ms *MCPServer) NewHTTPHandler() http.Handler {
	// 创建StreamableHTTP处理器
	handler := mcp.NewStreamableHTTPHandler(
		func(r *http.Request) *mcp.Server {
			// 返回MCP服务器实例
			// 注意：认证验证在外层middleware中完成
			return ms.server
		},
		&mcp.StreamableHTTPOptions{
			// Stateless 模式：不验证 session ID，每个请求独立处理
			// 这简化了 HTTP 模式的 session 管理，避免 "session not found" 错误
			Stateless:    true,
			JSONResponse: false,
		},
	)

	utils.GlobalLogger.Info("创建MCP HTTP处理器 [传输=StreamableHTTP] [模式=Stateless]")
	return handler
}

// ListTools 返回已注册的工具列表
func (ms *MCPServer) ListTools() []string {
	tools := []string{}
	for name := range ms.tools {
		tools = append(tools, name)
	}
	return tools
}

// RunSTDIO 以STDIO传输模式运行MCP服务器
// 使用官方SDK的StdioTransport实现stdin/stdout通信
func (ms *MCPServer) RunSTDIO(ctx context.Context) error {
	utils.GlobalLogger.Info("启动MCP STDIO服务器 [传输=StdioTransport]")

	// 使用官方SDK的StdioTransport运行服务器
	// server.Run会阻塞直到stdin关闭或收到shutdown通知
	if err := ms.server.Run(ctx, &mcp.StdioTransport{}); err != nil {
		utils.GlobalLogger.Error("STDIO服务器运行失败: %s", err)
		return err
	}

	utils.GlobalLogger.Info("STDIO服务器已停止")
	return nil
}

// RequireAuth 验证认证状态
// 注意：实际实现在server/middleware.go中
func RequireAuth(ctx context.Context) error {
	auth, ok := ctx.Value("authenticated").(bool)
	if !ok || !auth {
		return fmt.Errorf("请求未认证")
	}
	return nil
}
