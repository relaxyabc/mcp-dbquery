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

// MySQLQueryHandler MySQL查询处理器
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

		// 获取MySQL驱动实例
		driver, err := getOrConnectMySQL(ctx, poolManager, databaseID)
		if err != nil {
			utils.GlobalLogger.Error("获取MySQL驱动失败: %s", err)
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("数据库连接失败: %s", err)}},
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

// MongoDBQueryHandler MongoDB查询处理器
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

		// 获取MongoDB驱动实例
		driver, err := getOrConnectMongo(ctx, poolManager, databaseID)
		if err != nil {
			utils.GlobalLogger.Error("获取MongoDB驱动失败: %s", err)
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("数据库连接失败: %s", err)}},
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

// getOrConnectMySQL 获取或连接MySQL驱动
func getOrConnectMySQL(ctx context.Context, poolManager *database.PoolManager, databaseID string) (*mysql.MySQLDriver, error) {
	// 尝试从pool manager获取已连接的驱动
	driverInterface, exists := poolManager.GetMySQLDriver(databaseID)
	if exists {
		driver, ok := driverInterface.(*mysql.MySQLDriver)
		if ok && driver.IsConnected() {
			return driver, nil
		}
	}

	// 驱动不存在或未连接，创建新驱动
	config, exists := poolManager.GetConfig(databaseID)
	if !exists {
		return nil, fmt.Errorf("未找到数据库配置: %s", databaseID)
	}

	driver := mysql.NewMySQLDriver(config)
	if err := driver.Connect(ctx); err != nil {
		return nil, err
	}

	// 存储到pool manager
	poolManager.SetMySQLDriver(databaseID, driver)
	return driver, nil
}

// getOrConnectMongo 获取或连接MongoDB驱动
func getOrConnectMongo(ctx context.Context, poolManager *database.PoolManager, databaseID string) (*mongodb.MongoDBDriver, error) {
	utils.GlobalLogger.Info("getOrConnectMongo: 开始获取MongoDB驱动 [ID=%s]", databaseID)

	// 尝试从pool manager获取已连接的驱动
	driverInterface, exists := poolManager.GetMongoDriver(databaseID)
	utils.GlobalLogger.Info("getOrConnectMongo: PoolManager查询结果 [存在=%v]", exists)

	if exists {
		driver, ok := driverInterface.(*mongodb.MongoDBDriver)
		utils.GlobalLogger.Info("getOrConnectMongo: 类型转换结果 [成功=%v]", ok)
		if ok {
			isConnected := driver.IsConnected()
			utils.GlobalLogger.Info("getOrConnectMongo: 连接状态检查 [IsConnected=%v]", isConnected)
			if isConnected {
				utils.GlobalLogger.Info("getOrConnectMongo: 使用已存在的连接")
				return driver, nil
			}
			utils.GlobalLogger.Info("getOrConnectMongo: 驱动存在但未连接，需要重连")
		}
	}

	// 驱动不存在或未连接，创建新驱动
	config, exists := poolManager.GetConfig(databaseID)
	if !exists {
		utils.GlobalLogger.Error("getOrConnectMongo: 未找到数据库配置 [ID=%s]", databaseID)
		return nil, fmt.Errorf("未找到数据库配置: %s", databaseID)
	}

	utils.GlobalLogger.Info("getOrConnectMongo: 创建新驱动 [ID=%s] [类型=%s]", databaseID, config.Type)
	driver := mongodb.NewMongoDBDriver(config)

	utils.GlobalLogger.Info("getOrConnectMongo: 开始连接...")
	if err := driver.Connect(ctx); err != nil {
		utils.GlobalLogger.Error("getOrConnectMongo: 连接失败: %s", err)
		return nil, err
	}

	// 存储到pool manager
	utils.GlobalLogger.Info("getOrConnectMongo: 连接成功，存储到PoolManager")
	poolManager.SetMongoDriver(databaseID, driver)
	return driver, nil
}