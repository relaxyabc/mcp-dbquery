package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/relaxyabc/mcp-dbquery/src/database"
	"github.com/relaxyabc/mcp-dbquery/src/utils"
)

// ShowIndexes 获取表索引信息
func (d *PostgresDriver) ShowIndexes(ctx context.Context, tableName string) (*database.IndexListMetadata, error) {
	query := `
		SELECT
			i.relname as index_name,
			array_to_string(array_agg(a.attname ORDER BY a.attnum), ',') as columns,
			ix.indisunique as is_unique,
			ix.indisprimary as is_primary,
			CASE
				WHEN am.amname = 'btree' THEN 'btree'
				WHEN am.amname = 'hash' THEN 'hash'
				WHEN am.amname = 'gist' THEN 'gist'
				WHEN am.amname = 'gin' THEN 'gin'
				WHEN am.amname = 'spgist' THEN 'spgist'
				WHEN am.amname = 'brin' THEN 'brin'
				ELSE am.amname
			END as index_type
		FROM pg_class t
		JOIN pg_index ix ON t.oid = ix.indrelid
		JOIN pg_class i ON i.oid = ix.indexrelid
		JOIN pg_am am ON i.relam = am.oid
		JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = ANY(ix.indkey)
		WHERE t.relname = $1
		AND t.relnamespace = (SELECT oid FROM pg_namespace WHERE nspname = 'public')
		GROUP BY i.relname, ix.indisunique, ix.indisprimary, am.amname
		ORDER BY i.relname
	`

	rows, err := d.Pool.Query(ctx, query, tableName)
	if err != nil {
		utils.GlobalLogger.LogError("INDEX_ERROR", "获取PostgreSQL索引信息失败", err.Error())
		return nil, fmt.Errorf("获取索引信息失败: %s", err)
	}
	defer rows.Close()

	indexList := database.NewIndexListMetadata(tableName)

	for rows.Next() {
		var indexName, columns, indexType string
		var isUnique, isPrimary bool

		if err := rows.Scan(&indexName, &columns, &isUnique, &isPrimary, &indexType); err != nil {
			utils.GlobalLogger.Warn("读取索引信息失败: %s", err)
			continue
		}

		// 解析列名列表
		colList := strings.Split(columns, ",")

		index := database.NewIndexMetadata(indexName, tableName, d.Config.Database)
		index.Type = indexType
		index.Unique = isUnique

		// 添加索引字段
		for _, col := range colList {
			col = strings.TrimSpace(col)
			if col != "" {
				index.AddField(col, "ASC")
			}
		}

		indexList.AddIndex(*index)
	}

	return indexList, nil
}