package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/relaxyabc/mcp-dbquery/src/database"
	"github.com/relaxyabc/mcp-dbquery/src/utils"
)

// IndexesHandler Index查询处理器
func IndexesHandler(poolManager *database.PoolManager) func(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
		databaseID, _ := args["database_id"].(string)
		tableName, _ := args["table_name"].(string)

		if databaseID == "" {
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: "缺少必需参数: database_id"}},
			}, nil
		}

		utils.GlobalLogger.Info("Index查询请求 [连接=%s] [表=%s]", databaseID, tableName)

		// 获取数据库配置，根据类型直接选择驱动
		config, exists := poolManager.GetConfig(databaseID)
		if !exists {
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("未找到数据库配置: %s", databaseID)}},
			}, nil
		}

		utils.GlobalLogger.Info("数据库配置 [ID=%s] [类型=%s]", databaseID, config.Type)

		// 根据数据库类型选择驱动
		switch config.Type {
		case database.DatabaseTypeMongoDB:
			utils.GlobalLogger.Info("尝试获取MongoDB驱动 [ID=%s]", databaseID)
			mongoDriver, err := getOrConnectMongo(ctx, poolManager, databaseID)
			if err != nil {
				return &mcp.CallToolResult{
					IsError: true,
					Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("获取MongoDB驱动失败: %s", err)}},
				}, nil
			}

			if tableName != "" {
				indexes, err := mongoDriver.GetIndexes(ctx, tableName)
				if err != nil {
					return &mcp.CallToolResult{
						IsError: true,
						Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("获取索引失败: %s", err)}},
					}, nil
				}
				resultJSON, _ := json.Marshal(indexes.ToJSON())
				return &mcp.CallToolResult{
					Content: []mcp.Content{&mcp.TextContent{Text: string(resultJSON)}},
				}, nil
			}

			// 获取所有集合的索引
			collections, err := mongoDriver.ListTables(ctx)
			if err != nil {
				return &mcp.CallToolResult{
					IsError: true,
					Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("获取集合列表失败: %s", err)}},
				}, nil
			}

			allIndexes := make(map[string]interface{})
			for _, coll := range collections {
				indexes, err := mongoDriver.GetIndexes(ctx, coll)
				if err != nil {
					continue
				}
				allIndexes[coll] = indexes.ToJSON()
			}

			resultJSON, _ := json.Marshal(map[string]interface{}{
				"database_id": databaseID,
				"type":        "mongodb",
				"indexes":     allIndexes,
			})
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: string(resultJSON)}},
			}, nil

		default:
			// MySQL 及其他 SQL 数据库
			utils.GlobalLogger.Info("尝试获取MySQL驱动 [ID=%s]", databaseID)
			mysqlDriver, err := getOrConnectMySQL(ctx, poolManager, databaseID)
			if err != nil {
				return &mcp.CallToolResult{
					IsError: true,
					Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("获取MySQL驱动失败: %s", err)}},
				}, nil
			}

			if tableName != "" {
				indexes, err := mysqlDriver.GetIndexes(ctx, tableName)
				if err != nil {
					return &mcp.CallToolResult{
						IsError: true,
						Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("获取索引失败: %s", err)}},
					}, nil
				}
				resultJSON, _ := json.Marshal(indexes.ToJSON())
				return &mcp.CallToolResult{
					Content: []mcp.Content{&mcp.TextContent{Text: string(resultJSON)}},
				}, nil
			}

			// 获取所有表的索引
			tables, err := mysqlDriver.ListTables(ctx)
			if err != nil {
				return &mcp.CallToolResult{
					IsError: true,
					Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("获取表列表失败: %s", err)}},
				}, nil
			}

			allIndexes := make(map[string]interface{})
			for _, table := range tables {
				indexes, err := mysqlDriver.GetIndexes(ctx, table)
				if err != nil {
					continue
				}
				allIndexes[table] = indexes.ToJSON()
			}

			resultJSON, _ := json.Marshal(map[string]interface{}{
				"database_id": databaseID,
				"type":        "mysql",
				"indexes":     allIndexes,
			})
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: string(resultJSON)}},
			}, nil
		}
	}
}