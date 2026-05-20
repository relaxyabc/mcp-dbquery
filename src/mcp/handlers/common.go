package handlers

import (
	"context"
	"fmt"

	"github.com/relaxyabc/mcp-dbquery/src/database"
)

// GetDriver 统一的驱动获取函数
// 从 PoolManager 获取或创建数据库驱动实例
func GetDriver(ctx context.Context, pm *database.PoolManager, databaseID string) (database.Database, error) {
	if databaseID == "" {
		return nil, fmt.Errorf("缺少必需参数: database_id")
	}

	driver, err := pm.GetOrCreatePool(ctx, databaseID)
	if err != nil {
		return nil, fmt.Errorf("获取数据库驱动失败: %s", err)
	}

	return driver, nil
}