package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/relaxyabc/mcp-dbquery/src/database"
	"github.com/relaxyabc/mcp-dbquery/src/utils"
)

// SchemaHandler Schema查询处理器
func SchemaHandler(poolManager *database.PoolManager) func(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
		databaseID, _ := args["database_id"].(string)
		tableName, _ := args["table_name"].(string)

		if databaseID == "" {
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: "缺少必需参数: database_id"}},
			}, nil
		}

		utils.GlobalLogger.Info("Schema查询请求 [连接=%s] [表=%s]", databaseID, tableName)

		// 尝试获取MySQL驱动
		mysqlDriver, mysqlErr := getOrConnectMySQL(ctx, poolManager, databaseID)
		if mysqlErr == nil {
			// MySQL处理
			if tableName != "" {
				// 获取单个表结构
				schema, err := mysqlDriver.GetSchema(ctx, tableName)
				if err != nil {
					return &mcp.CallToolResult{
						IsError: true,
						Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("获取表结构失败: %s", err)}},
					}, nil
				}
				resultJSON, _ := json.Marshal(schema.ToJSON())
				return &mcp.CallToolResult{
					Content: []mcp.Content{&mcp.TextContent{Text: string(resultJSON)}},
				}, nil
			}

			// 获取所有表列表
			tables, err := mysqlDriver.ListTables(ctx)
			if err != nil {
				return &mcp.CallToolResult{
					IsError: true,
					Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("获取表列表失败: %s", err)}},
				}, nil
			}

			result := map[string]interface{}{
				"database_id": databaseID,
				"tables":      tables,
				"table_count": len(tables),
			}
			resultJSON, _ := json.Marshal(result)
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: string(resultJSON)}},
			}, nil
		}

		// 尝试MongoDB
		mongoDriver, mongoErr := getOrConnectMongo(ctx, poolManager, databaseID)
		if mongoErr == nil {
			// MongoDB处理
			if tableName != "" {
				// 获取单个集合结构（推断）
				schema, err := mongoDriver.GetSchema(ctx, tableName)
				if err != nil {
					return &mcp.CallToolResult{
						IsError: true,
						Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("获取集合结构失败: %s", err)}},
					}, nil
				}
				resultJSON, _ := json.Marshal(schema.ToJSON())
				return &mcp.CallToolResult{
					Content: []mcp.Content{&mcp.TextContent{Text: string(resultJSON)}},
				}, nil
			}

			// 获取所有集合列表
			collections, err := mongoDriver.ListTables(ctx)
			if err != nil {
				return &mcp.CallToolResult{
					IsError: true,
					Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("获取集合列表失败: %s", err)}},
				}, nil
			}

			result := map[string]interface{}{
				"database_id":      databaseID,
				"collections":      collections,
				"collection_count": len(collections),
			}
			resultJSON, _ := json.Marshal(result)
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: string(resultJSON)}},
			}, nil
		}

		// 两种数据库都找不到
		utils.GlobalLogger.Error("找不到数据库连接: %s [MySQL错误=%s] [MongoDB错误=%s]",
			databaseID, mysqlErr, mongoErr)
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("找不到数据库连接: %s", databaseID)}},
		}, nil
	}
}

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

		// 尝试MySQL
		mysqlDriver, mysqlErr := getOrConnectMySQL(ctx, poolManager, databaseID)
		if mysqlErr == nil {
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
				"indexes":     allIndexes,
			})
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: string(resultJSON)}},
			}, nil
		}

		// 尝试MongoDB
		mongoDriver, mongoErr := getOrConnectMongo(ctx, poolManager, databaseID)
		if mongoErr == nil {
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
				"indexes":     allIndexes,
			})
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: string(resultJSON)}},
			}, nil
		}

		utils.GlobalLogger.Error("找不到数据库连接: %s [MySQL错误=%s] [MongoDB错误=%s]",
			databaseID, mysqlErr, mongoErr)
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("找不到数据库连接: %s", databaseID)}},
		}, nil
	}
}

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

		// 尝试MySQL
		mysqlDriver, mysqlErr := getOrConnectMySQL(ctx, poolManager, databaseID)
		if mysqlErr == nil {
			tables, err := mysqlDriver.ListTables(ctx)
			if err != nil {
				return &mcp.CallToolResult{
					IsError: true,
					Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("获取表列表失败: %s", err)}},
				}, nil
			}

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

		// 尝试MongoDB
		mongoDriver, mongoErr := getOrConnectMongo(ctx, poolManager, databaseID)
		if mongoErr == nil {
			collections, err := mongoDriver.ListTables(ctx)
			if err != nil {
				return &mcp.CallToolResult{
					IsError: true,
					Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("获取集合列表失败: %s", err)}},
				}, nil
			}

			resultJSON, _ := json.Marshal(map[string]interface{}{
				"database_id":      databaseID,
				"type":             "mongodb",
				"collections":      collections,
				"collection_count": len(collections),
			})
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: string(resultJSON)}},
			}, nil
		}

		utils.GlobalLogger.Error("找不到数据库连接: %s [MySQL错误=%s] [MongoDB错误=%s]",
			databaseID, mysqlErr, mongoErr)
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("找不到数据库连接: %s", databaseID)}},
		}, nil
	}
}