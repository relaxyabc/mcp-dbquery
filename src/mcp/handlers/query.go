package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/relaxyabc/mcp-dbquery/src/database"
	"github.com/relaxyabc/mcp-dbquery/src/utils"
)

// MySQLQueryHandler SQL查询处理器（重构：统一驱动接口）
// 支持所有SQL数据库类型（MySQL、PostgreSQL、SQLite、SQL Server、Oracle等）
func MySQLQueryHandler(poolManager *database.PoolManager) func(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
		start := time.Now()

		// 提取参数
		databaseID, _ := args["database_id"].(string)
		query, _ := args["query"].(string)
		limit, _ := args["limit"].(int)
		if limit <= 0 || limit > 1000 {
			limit = 1000
		}

		// 参数验证
		if databaseID == "" {
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: "缺少必需参数: database_id"}},
			}, nil
		}
		if query == "" {
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: "缺少必需参数: query"}},
			}, nil
		}

		utils.GlobalLogger.Info("SQL查询请求 [连接=%s] [查询=%s]", databaseID, query)

		// 使用统一接口获取驱动
		driver, err := GetDriver(ctx, poolManager, databaseID)
		if err != nil {
			utils.GlobalLogger.Error("获取驱动失败: %s", err)
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
			}, nil
		}

		// 检查驱动类型是否支持SQL查询
		driverType := driver.GetType()
		if driverType == database.DatabaseTypeMongoDB {
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: "该连接是 MongoDB 类型，不支持 SQL SELECT 查询，请使用 query_mongodb_data 工具"}},
			}, nil
		}

		// 验证查询是否为只读操作
		if err := driver.ValidateQuery(query); err != nil {
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("查询验证失败: %s", err)}},
			}, nil
		}

		// 统一调用 ExecuteSelectQuery 接口方法
		result, err := driver.ExecuteSelectQuery(ctx, query, limit)
		if err != nil {
			utils.GlobalLogger.Error("查询执行失败: %s", err)
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("查询执行失败: %s", err)}},
			}, nil
		}

		// 返回结果
		executionTime := int(time.Since(start).Milliseconds())
		result.ExecutionTime = executionTime

		utils.GlobalLogger.Info("SQL查询完成 [连接=%s] [类型=%s] [返回行数=%d] [耗时=%dms]",
			databaseID, driverType, result.RowCount, executionTime)

		// 将结果转换为JSON字符串
		resultJSON, err := json.Marshal(result.ToJSON())
		if err != nil {
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("JSON转换失败: %s", err)}},
			}, nil
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(resultJSON)}},
		}, nil
	}
}