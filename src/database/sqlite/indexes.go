package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/relaxyabc/mcp-dbquery/src/database"
	"github.com/relaxyabc/mcp-dbquery/src/utils"
)

// ShowIndexes 获取表索引信息（使用PRAGMA index_list和index_info）
func (d *SQLiteDriver) ShowIndexes(ctx context.Context, tableName string) (*database.IndexListMetadata, error) {
	// 获取索引列表
	listQuery := fmt.Sprintf("PRAGMA index_list(%s)", tableName)

	listRows, err := d.DB.QueryContext(ctx, listQuery)
	if err != nil {
		utils.GlobalLogger.LogError("INDEX_ERROR", "获取SQLite索引列表失败", err.Error())
		return nil, fmt.Errorf("获取索引信息失败: %s", err)
	}
	defer listRows.Close()

	indexList := database.NewIndexListMetadata(tableName)

	for listRows.Next() {
		var seq int
		var indexName, origin string
		var unique int
		var partial sql.NullInt64

		if err := listRows.Scan(&seq, &indexName, &unique, &origin, &partial); err != nil {
			utils.GlobalLogger.Warn("读取索引列表失败: %s", err)
			continue
		}

		// 创建索引元数据
		index := database.NewIndexMetadata(indexName, tableName, d.Config.Path)
		index.Unique = unique == 1

		// 确定索引类型
		indexType := "btree" // SQLite默认使用B-tree
		if origin == "c" {
			indexType = "btree" // CREATE INDEX创建
		} else if origin == "pk" {
			indexType = "btree" // 主键索引
		} else if origin == "u" {
			indexType = "btree" // UNIQUE约束
		}
		index.Type = indexType

		// 获取索引字段信息
		infoQuery := fmt.Sprintf("PRAGMA index_info(%s)", indexName)
		infoRows, err := d.DB.QueryContext(ctx, infoQuery)
		if err == nil {
			defer infoRows.Close()
			for infoRows.Next() {
				var seqNo, cid int
				var colName sql.NullString

				if err := infoRows.Scan(&seqNo, &cid, &colName); err != nil {
					utils.GlobalLogger.Warn("读取索引字段失败: %s", err)
					continue
				}

				if colName.Valid && colName.String != "" {
					// SQLite不支持ASC/DESC标记，默认ASC
					index.AddField(colName.String, "ASC")
				}
			}
		}

		indexList.AddIndex(*index)
	}

	// 检查主键是否作为索引列出（SQLite主键可能不在index_list中显式出现）
	// 通过sqlite_master查询PRIMARY KEY约束
	pkQuery := `SELECT sql FROM sqlite_master WHERE type='table' AND name = ?`
	pkRows, err := d.DB.QueryContext(ctx, pkQuery, tableName)
	if err == nil {
		defer pkRows.Close()
		if pkRows.Next() {
			var createSQL sql.NullString
			if err := pkRows.Scan(&createSQL); err == nil && createSQL.Valid {
				// 检查是否定义了PRIMARY KEY
				if strings.Contains(strings.ToUpper(createSQL.String), "PRIMARY KEY") {
					// 如果PRIMARY KEY不在索引列表中，手动添加
					pkIndexName := fmt.Sprintf("sqlite_autoindex_%s_1", tableName)
					exists := false
					for _, idx := range indexList.Indexes {
						if idx.IndexName == pkIndexName {
							exists = true
							break
						}
					}
					if !exists {
						// 从schema获取主键字段
						schema, err := d.DescribeTable(ctx, tableName)
						if err == nil {
							pkFields := []string{}
							for _, f := range schema.Fields {
								if f.PrimaryKey {
									pkFields = append(pkFields, f.Name)
								}
							}
							if len(pkFields) > 0 {
								pkIndex := database.NewIndexMetadata(pkIndexName, tableName, d.Config.Path)
								pkIndex.Unique = true
								pkIndex.Type = "btree"
								for _, f := range pkFields {
									pkIndex.AddField(f, "ASC")
								}
								indexList.AddIndex(*pkIndex)
							}
						}
					}
				}
			}
		}
	}

	return indexList, nil
}