package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/relaxyabc/mcp-dbquery/src/database"
	"github.com/relaxyabc/mcp-dbquery/src/utils"
)

// ListTablesHandler 表列表处理器（重构：统一驱动获取）
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

		// 使用统一接口获取驱动
		driver, err := GetDriver(ctx, poolManager, databaseID)
		if err != nil {
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
			}, nil
		}

		utils.GlobalLogger.Info("驱动获取成功 [ID=%s] [类型=%s]", databaseID, driver.GetType())

		// 统一调用 ListTables 接口方法
		tables, err := driver.ListTables(ctx)
		if err != nil {
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("获取表列表失败: %s", err)}},
			}, nil
		}

		utils.GlobalLogger.Info("表列表查询完成 [连接=%s] [表数=%d]", databaseID, len(tables))

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