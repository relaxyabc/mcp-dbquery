package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/relaxyabc/mcp-dbquery/src/database"
	"github.com/relaxyabc/mcp-dbquery/src/utils"
)

// IndexesHandler Index查询处理器（重构：统一驱动获取）
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

		// 使用统一接口获取驱动
		driver, err := GetDriver(ctx, poolManager, databaseID)
		if err != nil {
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
			}, nil
		}

		utils.GlobalLogger.Info("驱动获取成功 [ID=%s] [类型=%s]", databaseID, driver.GetType())

		// 查询指定表的索引
		if tableName != "" {
			indexes, err := driver.GetIndexes(ctx, tableName)
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

		// 获取所有表/集合的索引
		tables, err := driver.ListTables(ctx)
		if err != nil {
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("获取表列表失败: %s", err)}},
			}, nil
		}

		allIndexes := make(map[string]interface{})
		for _, table := range tables {
			indexes, err := driver.GetIndexes(ctx, table)
			if err != nil {
				continue
			}
			allIndexes[table] = indexes.ToJSON()
		}

		resultJSON, _ := json.Marshal(map[string]interface{}{
			"database_id": databaseID,
			"type":        driver.GetType(),
			"indexes":     allIndexes,
		})
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(resultJSON)}},
		}, nil
	}
}