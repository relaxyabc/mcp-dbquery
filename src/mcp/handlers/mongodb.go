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

// MongoDBQueryHandler MongoDB find查询处理器
func MongoDBQueryHandler(poolManager *database.PoolManager) func(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
		start := time.Now()

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

		utils.GlobalLogger.Info("MongoDB查询请求 [连接=%s] [集合=%s]", databaseID, collection)

		// 使用统一接口获取驱动
		driver, err := GetDriver(ctx, poolManager, databaseID)
		if err != nil {
			utils.GlobalLogger.Error("获取驱动失败: %s", err)
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
			}, nil
		}

		// 检查驱动类型
		driverType := driver.GetType()
		if driverType != database.DatabaseTypeMongoDB {
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("该连接是 %s 类型，不支持 MongoDB find 查询，请使用 query_mysql_data 工具执行 SQL 查询", driverType)}},
			}, nil
		}

		// 执行find查询
		result, err := driver.ExecuteFind(ctx, collection, filterMap, limit)
		if err != nil {
			utils.GlobalLogger.Error("查询执行失败: %s", err)
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("查询执行失败: %s", err)}},
			}, nil
		}

		executionTime := int(time.Since(start).Milliseconds())
		result.ExecutionTime = executionTime

		utils.GlobalLogger.Info("MongoDB查询完成 [连接=%s] [集合=%s] [返回文档数=%d] [耗时=%dms]",
			databaseID, collection, result.RowCount, executionTime)

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

// MongoDBAggregateHandler MongoDB聚合查询处理器
func MongoDBAggregateHandler(poolManager *database.PoolManager) func(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
		start := time.Now()

		// 提取参数
		databaseID, _ := args["database_id"].(string)
		collection, _ := args["collection"].(string)
		pipelineRaw, _ := args["pipeline"].([]interface{})
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

		// 转换 pipeline
		pipeline := []map[string]interface{}{}
		for _, stage := range pipelineRaw {
			if stageMap, ok := stage.(map[string]interface{}); ok {
				pipeline = append(pipeline, stageMap)
			}
		}

		if len(pipeline) == 0 {
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: "聚合管道不能为空"}},
			}, nil
		}

		utils.GlobalLogger.Info("MongoDB聚合查询请求 [连接=%s] [集合=%s] [管道阶段数=%d]", databaseID, collection, len(pipeline))

		// 使用统一接口获取驱动
		driver, err := GetDriver(ctx, poolManager, databaseID)
		if err != nil {
			utils.GlobalLogger.Error("获取驱动失败: %s", err)
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
			}, nil
		}

		// 检查驱动类型
		driverType := driver.GetType()
		if driverType != database.DatabaseTypeMongoDB {
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("该连接是 %s 类型，不支持 MongoDB aggregate 查询", driverType)}},
			}, nil
		}

		// 执行聚合查询
		result, err := driver.ExecuteAggregate(ctx, collection, pipeline, limit)
		if err != nil {
			utils.GlobalLogger.Error("聚合查询执行失败: %s", err)
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("聚合查询执行失败: %s", err)}},
			}, nil
		}

		executionTime := int(time.Since(start).Milliseconds())
		result.ExecutionTime = executionTime

		utils.GlobalLogger.Info("MongoDB聚合查询完成 [连接=%s] [集合=%s] [返回文档数=%d] [耗时=%dms]",
			databaseID, collection, result.RowCount, executionTime)

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

// MongoDBCountHandler MongoDB计数查询处理器
func MongoDBCountHandler(poolManager *database.PoolManager) func(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
		start := time.Now()

		// 提取参数
		databaseID, _ := args["database_id"].(string)
		collection, _ := args["collection"].(string)
		filterMap, _ := args["filter"].(map[string]interface{})

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

		utils.GlobalLogger.Info("MongoDB计数查询请求 [连接=%s] [集合=%s]", databaseID, collection)

		// 使用统一接口获取驱动
		driver, err := GetDriver(ctx, poolManager, databaseID)
		if err != nil {
			utils.GlobalLogger.Error("获取驱动失败: %s", err)
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
			}, nil
		}

		// 检查驱动类型
		driverType := driver.GetType()
		if driverType != database.DatabaseTypeMongoDB {
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("该连接是 %s 类型，不支持 MongoDB count 查询", driverType)}},
			}, nil
		}

		// 执行计数查询
		count, err := driver.ExecuteCount(ctx, collection, filterMap)
		if err != nil {
			utils.GlobalLogger.Error("计数查询执行失败: %s", err)
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("计数查询执行失败: %s", err)}},
			}, nil
		}

		executionTime := int(time.Since(start).Milliseconds())

		utils.GlobalLogger.Info("MongoDB计数查询完成 [连接=%s] [集合=%s] [文档数=%d] [耗时=%dms]",
			databaseID, collection, count, executionTime)

		result := map[string]interface{}{
			"database_id":    databaseID,
			"collection":     collection,
			"count":          count,
			"execution_time": executionTime,
		}
		resultJSON, _ := json.Marshal(result)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(resultJSON)}},
		}, nil
	}
}

// MongoDBDistinctHandler MongoDB distinct查询处理器
func MongoDBDistinctHandler(poolManager *database.PoolManager) func(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
		start := time.Now()

		// 提取参数
		databaseID, _ := args["database_id"].(string)
		collection, _ := args["collection"].(string)
		field, _ := args["field"].(string)
		filterMap, _ := args["filter"].(map[string]interface{})

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
		if field == "" {
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: "缺少必需参数: field"}},
			}, nil
		}

		utils.GlobalLogger.Info("MongoDB distinct查询请求 [连接=%s] [集合=%s] [字段=%s]", databaseID, collection, field)

		// 使用统一接口获取驱动
		driver, err := GetDriver(ctx, poolManager, databaseID)
		if err != nil {
			utils.GlobalLogger.Error("获取驱动失败: %s", err)
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
			}, nil
		}

		// 检查驱动类型
		driverType := driver.GetType()
		if driverType != database.DatabaseTypeMongoDB {
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("该连接是 %s 类型，不支持 MongoDB distinct 查询", driverType)}},
			}, nil
		}

		// 执行distinct查询
		values, err := driver.ExecuteDistinct(ctx, collection, field, filterMap)
		if err != nil {
			utils.GlobalLogger.Error("Distinct查询执行失败: %s", err)
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Distinct查询执行失败: %s", err)}},
			}, nil
		}

		executionTime := int(time.Since(start).Milliseconds())

		utils.GlobalLogger.Info("MongoDB distinct查询完成 [连接=%s] [集合=%s] [字段=%s] [唯一值数=%d] [耗时=%dms]",
			databaseID, collection, field, len(values), executionTime)

		result := map[string]interface{}{
			"database_id":    databaseID,
			"collection":     collection,
			"field":          field,
			"values":         values,
			"count":          len(values),
			"execution_time": executionTime,
		}
		resultJSON, _ := json.Marshal(result)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(resultJSON)}},
		}, nil
	}
}