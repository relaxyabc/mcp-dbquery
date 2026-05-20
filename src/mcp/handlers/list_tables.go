package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/relaxyabc/mcp-dbquery/src/database"
	"github.com/relaxyabc/mcp-dbquery/src/utils"
)

// ListTablesHandler 表列表处理器
func ListTablesHandler(poolManager *database.PoolManager) func(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
		databaseID, _ := args["database_id"].(string)

		if databaseID == "" {
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: "缺少必需参数: database_id"}},
			}, nil
		}

		utils.GlobalLogger.Info("表列表查询请求 [连接=%s]", databaseID)

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
				utils.GlobalLogger.Error("获取MongoDB驱动失败: %s", err)
				return &mcp.CallToolResult{
					IsError: true,
					Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("获取MongoDB驱动失败: %s", err)}},
				}, nil
			}
			utils.GlobalLogger.Info("MongoDB驱动获取成功，开始列出集合...")
			collections, err := mongoDriver.ListTables(ctx)
			if err != nil {
				utils.GlobalLogger.Error("获取集合列表失败: %s", err)
				return &mcp.CallToolResult{
					IsError: true,
					Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("获取集合列表失败: %s", err)}},
				}, nil
			}

			utils.GlobalLogger.Info("MongoDB集合列表完成 [集合数=%d]", len(collections))
			resultJSON, _ := json.Marshal(map[string]interface{}{
				"database_id":      databaseID,
				"type":             "mongodb",
				"collections":      collections,
				"collection_count": len(collections),
			})
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: string(resultJSON)}},
			}, nil

		default:
			// MySQL 及其他 SQL 数据库
			utils.GlobalLogger.Info("尝试获取MySQL驱动 [ID=%s]", databaseID)
			mysqlDriver, err := getOrConnectMySQL(ctx, poolManager, databaseID)
			if err != nil {
				utils.GlobalLogger.Error("获取MySQL驱动失败: %s", err)
				return &mcp.CallToolResult{
					IsError: true,
					Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("获取MySQL驱动失败: %s", err)}},
				}, nil
			}
			utils.GlobalLogger.Info("MySQL驱动获取成功，开始列出表...")
			tables, err := mysqlDriver.ListTables(ctx)
			if err != nil {
				utils.GlobalLogger.Error("获取表列表失败: %s", err)
				return &mcp.CallToolResult{
					IsError: true,
					Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("获取表列表失败: %s", err)}},
				}, nil
			}

			utils.GlobalLogger.Info("MySQL表列表完成 [表数=%d]", len(tables))
			resultJSON, _ := json.Marshal(map[string]interface{}{
				"database_id": databaseID,
				"type":        "mysql",
				"tables":      tables,
				"table_count": len(tables),
			})
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: string(resultJSON)}},
			}, nil
		}
	}
}