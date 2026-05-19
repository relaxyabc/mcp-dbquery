package oracle

import (
	"context"
	"fmt"
	"strings"

	"github.com/relaxyabc/mcp-dbquery/src/database"
	"github.com/relaxyabc/mcp-dbquery/src/utils"
)

// ShowIndexes 获取表索引信息（使用ALL_INDEXES和ALL_IND_COLUMNS）
func (d *OracleDriver) ShowIndexes(ctx context.Context, tableName string) (*database.IndexListMetadata, error) {
	// 查询索引信息
	query := `
		SELECT
			idx.index_name,
			idx.index_type,
			idx.uniqueness,
			CASE WHEN cons.constraint_type = 'P' THEN 1 ELSE 0 END AS is_primary,
			STUFF((
				SELECT ',' || col.column_name
				FROM all_ind_columns col
				WHERE col.index_name = idx.index_name
				AND col.table_name = idx.table_name
				AND col.table_owner = idx.table_owner
				ORDER BY col.column_position
				FOR XML PATH('')
			), 1, 1, '') AS columns
		FROM all_indexes idx
		LEFT JOIN all_constraints cons ON idx.index_name = cons.constraint_name
			AND idx.table_name = cons.table_name
			AND idx.owner = cons.owner
			AND cons.constraint_type = 'P'
		WHERE idx.table_name = UPPER(:1)
		AND idx.table_owner = USER
		ORDER BY idx.index_name
	`

	// 注意：Oracle不支持FOR XML PATH，使用替代查询获取列列表
	query = `
		SELECT
			idx.index_name,
			idx.index_type,
			idx.uniqueness,
			CASE WHEN cons.constraint_type = 'P' THEN 1 ELSE 0 END AS is_primary
		FROM all_indexes idx
		LEFT JOIN all_constraints cons ON idx.index_name = cons.constraint_name
			AND idx.table_name = cons.table_name
			AND idx.owner = cons.owner
			AND cons.constraint_type = 'P'
		WHERE idx.table_name = UPPER(:1)
		AND idx.table_owner = USER
		ORDER BY idx.index_name
	`

	rows, err := d.DB.QueryContext(ctx, query, tableName)
	if err != nil {
		utils.GlobalLogger.LogError("INDEX_ERROR", "获取Oracle索引信息失败", err.Error())
		return nil, fmt.Errorf("获取索引信息失败: %s", err)
	}
	defer rows.Close()

	indexList := database.NewIndexListMetadata(tableName)

	for rows.Next() {
		var indexName, indexType, uniqueness string
		var isPrimary int

		if err := rows.Scan(&indexName, &indexType, &uniqueness, &isPrimary); err != nil {
			utils.GlobalLogger.Warn("读取索引信息失败: %s", err)
			continue
		}

		// 创建索引元数据
		index := database.NewIndexMetadata(indexName, tableName, d.Config.Username)
		index.Unique = uniqueness == "UNIQUE" || uniqueness == "UNIQUE BITMAP"
		index.Type = strings.ToLower(indexType) // normal/bitmap/function-based

		// 查询索引列
		colQuery := `
			SELECT column_name
			FROM all_ind_columns
			WHERE index_name = :1
			AND table_name = UPPER(:2)
			AND table_owner = USER
			ORDER BY column_position
		`

		colRows, err := d.DB.QueryContext(ctx, colQuery, indexName, tableName)
		if err == nil {
			defer colRows.Close()
			for colRows.Next() {
				var colName string
				if err := colRows.Scan(&colName); err == nil {
					// Oracle索引默认ASC
					index.AddField(colName, "ASC")
				}
			}
		}

		indexList.AddIndex(*index)
	}

	return indexList, nil
}