package sqlserver

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/relaxyabc/mcp-dbquery/src/database"
	"github.com/relaxyabc/mcp-dbquery/src/utils"
)

// ShowIndexes 获取表索引信息（使用sys.indexes和sys.index_columns）
func (d *SQLServerDriver) ShowIndexes(ctx context.Context, tableName string) (*database.IndexListMetadata, error) {
	// 查询索引信息
	query := `
		SELECT
			i.name AS index_name,
			i.type_desc AS index_type,
			i.is_unique,
			i.is_primary_key,
			STUFF((
				SELECT ',' + c.name
				FROM sys.index_columns ic
				JOIN sys.columns c ON ic.object_id = c.object_id AND ic.column_id = c.column_id
				WHERE ic.object_id = i.object_id AND ic.index_id = i.index_id
				ORDER BY ic.key_ordinal
				FOR XML PATH('')
			), 1, 1, '') AS columns
		FROM sys.indexes i
		JOIN sys.tables tab ON i.object_id = tab.object_id
		WHERE tab.name = @p1
		AND SCHEMA_NAME(tab.schema_id) = 'dbo'
		AND i.type > 0  -- 排除堆(HEAP)
		ORDER BY i.name
	`

	rows, err := d.DB.QueryContext(ctx, query, tableName)
	if err != nil {
		utils.GlobalLogger.LogError("INDEX_ERROR", "获取SQL Server索引信息失败", err.Error())
		return nil, fmt.Errorf("获取索引信息失败: %s", err)
	}
	defer rows.Close()

	indexList := database.NewIndexListMetadata(tableName)

	for rows.Next() {
		var indexName, indexType string
		var isUnique, isPK bool
		var columns sql.NullString // 使用 sql.NullString 处理可能为 NULL 的情况

		if err := rows.Scan(&indexName, &indexType, &isUnique, &isPK, &columns); err != nil {
			utils.GlobalLogger.Warn("读取索引信息失败: %s", err)
			continue
		}

		// 创建索引元数据
		index := database.NewIndexMetadata(indexName, tableName, d.Config.Database)
		index.Unique = isUnique
		index.Type = strings.ToLower(indexType) // clustered/nonclustered

		// 解析列名列表
		if columns.Valid && columns.String != "" {
			colList := strings.Split(columns.String, ",")
			for _, col := range colList {
				col = strings.TrimSpace(col)
				if col != "" {
					// SQL Server不存储ASC/DESC，默认ASC
					index.AddField(col, "ASC")
				}
			}
		}

		indexList.AddIndex(*index)
	}

	return indexList, nil
}