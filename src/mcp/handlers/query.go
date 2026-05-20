package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.mongodb.org/mongo-driver/bson"

	"github.com/relaxyabc/mcp-dbquery/src/database"
	"github.com/relaxyabc/mcp-dbquery/src/database/mongodb"
	"github.com/relaxyabc/mcp-dbquery/src/database/mysql"
	"github.com/relaxyabc/mcp-dbquery/src/utils"
)

// MySQLQueryHandler MySQL查询处理器（重构：统一驱动获取）
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
		timeout, _ := args["timeout"].(int)
		if timeout <= 0 {
			timeout = 30
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

		utils.GlobalLogger.Info("MySQL查询请求 [连接=%s] [查询=%s]", databaseID, query)

		// 使用统一接口获取驱动
		driverInterface, err := GetDriver(ctx, poolManager, databaseID)
		if err != nil {
			utils.GlobalLogger.Error("获取驱动失败: %s", err)
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
			}, nil
		}

		// 类型断言为 MySQL 驱动（用于调用 ExecuteSelectQuery）
		driver, ok := driverInterface.(*mysql.MySQLDriver)
		if !ok {
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: "该连接不是 MySQL 类型驱动"}},
			}, nil
		}

		// 执行查询
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

		utils.GlobalLogger.Info("MySQL查询完成 [连接=%s] [返回行数=%d] [耗时=%dms]",
			databaseID, result.RowCount, executionTime)

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

// MongoDBQueryHandler MongoDB查询处理器（重构：统一驱动获取）
func MongoDBQueryHandler(poolManager *database.PoolManager) func(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
		// 提取参数
		databaseID, _ := args["database_id"].(string)
		collection, _ := args["collection"].(string)
		filterMap, _ := args["filter"].(map[string]interface{})
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
		if collection == "" {
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: "缺少必需参数: collection"}},
			}, nil
		}

		// 转换filter为bson.M
		filter := bson.M{}
		for k, v := range filterMap {
			filter[k] = v
		}

		utils.GlobalLogger.Info("MongoDB查询请求 [连接=%s] [集合=%s]", databaseID, collection)

		// 使用统一接口获取驱动
		driverInterface, err := GetDriver(ctx, poolManager, databaseID)
		if err != nil {
			utils.GlobalLogger.Error("获取驱动失败: %s", err)
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
			}, nil
		}

		// 类型断言为 MongoDB 驱动（用于调用 ExecuteFind）
		driver, ok := driverInterface.(*mongodb.MongoDBDriver)
		if !ok {
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: "该连接不是 MongoDB 类型驱动"}},
			}, nil
		}

		// 执行查询
		result, err := driver.ExecuteFind(ctx, collection, filter, limit)
		if err != nil {
			utils.GlobalLogger.Error("查询执行失败: %s", err)
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("查询执行失败: %s", err)}},
			}, nil
		}

		// 返回结果
		utils.GlobalLogger.Info("MongoDB查询完成 [连接=%s] [集合=%s] [返回文档数=%d] [耗时=%dms]",
			databaseID, collection, result.RowCount, result.ExecutionTime)

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