package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/relaxyabc/mcp-dbquery/src/database"
	"github.com/relaxyabc/mcp-dbquery/src/utils"
)

// SchemaHandler Schema查询处理器（重构：统一驱动获取）
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

		// 使用统一接口获取驱动
		driver, err := GetDriver(ctx, poolManager, databaseID)
		if err != nil {
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
			}, nil
		}

		utils.GlobalLogger.Info("驱动获取成功 [ID=%s] [类型=%s]", databaseID, driver.GetType())

		// 统一调用接口方法
		if tableName != "" {
			schema, err := driver.GetSchema(ctx, tableName)
			if err != nil {
				return &mcp.CallToolResult{
					IsError: true,
					Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("获取结构失败: %s", err)}},
				}, nil
			}
			resultJSON, _ := json.Marshal(schema.ToJSON())
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: string(resultJSON)}},
			}, nil
		}

		// 列出所有表/集合
		tables, err := driver.ListTables(ctx)
		if err != nil {
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("获取列表失败: %s", err)}},
			}, nil
		}

		result := map[string]interface{}{
			"database_id": databaseID,
			"type":        driver.GetType(),
			"tables":      tables,
			"table_count": len(tables),
		}
		resultJSON, _ := json.Marshal(result)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(resultJSON)}},
		}, nil
	}
}