package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/relaxyabc/mcp-dbquery/src/database"
	"github.com/relaxyabc/mcp-dbquery/src/utils"
)

// DescribeTable 获取表结构信息（使用PRAGMA table_info）
func (d *SQLiteDriver) DescribeTable(ctx context.Context, tableName string) (*database.SchemaMetadata, error) {
	// SQLite使用PRAGMA获取表结构
	query := fmt.Sprintf("PRAGMA table_info(%s)", tableName)

	rows, err := d.DB.QueryContext(ctx, query)
	if err != nil {
		utils.GlobalLogger.LogError("SCHEMA_ERROR", "获取SQLite表结构失败", err.Error())
		return nil, fmt.Errorf("获取表结构失败: %s", err)
	}
	defer rows.Close()

	fields := []database.FieldMetadata{}
	for rows.Next() {
		var cid int
		var name, colType string
		var notNull int
		var defaultVal sql.NullString
		var pk int

		if err := rows.Scan(&cid, &name, &colType, &notNull, &defaultVal, &pk); err != nil {
			utils.GlobalLogger.Warn("读取列信息失败: %s", err)
			continue
		}

		defaultStr := ""
		if defaultVal.Valid {
			defaultStr = defaultVal.String
		}

		fields = append(fields, database.FieldMetadata{
			Name:         name,
			Type:         colType,
			Nullable:     notNull == 0,
			DefaultValue: defaultStr,
			PrimaryKey:   pk > 0,
			ForeignKey:   nil,
		})
	}

	// 查询外键信息
	fkQuery := fmt.Sprintf("PRAGMA foreign_key_list(%s)", tableName)
	fkRows, err := d.DB.QueryContext(ctx, fkQuery)
	if err == nil {
		defer fkRows.Close()
		for fkRows.Next() {
			var id, seq int
			var table, from, to string
			var onUpdate, onDelete string
			var match sql.NullString

			if err := fkRows.Scan(&id, &seq, &table, &from, &to, &onUpdate, &onDelete, &match); err != nil {
				utils.GlobalLogger.Warn("读取外键信息失败: %s", err)
				continue
			}

			// 更新字段的外键信息
			for i := range fields {
				if fields[i].Name == from {
					fields[i].ForeignKey = &database.ForeignKeyRef{
						ReferencedTable:  table,
						ReferencedField: to,
						OnUpdate:     onUpdate,
						OnDelete:     onDelete,
					}
				}
			}
		}
	}

	// 获取表类型（SQLite没有视图类型区分，通过sqlite_master查询）
	tableType := "table"
	typeQuery := `SELECT type FROM sqlite_master WHERE name = ? AND type IN ('table', 'view')`
	typeRows, err := d.DB.QueryContext(ctx, typeQuery, tableName)
	if err == nil {
		defer typeRows.Close()
		if typeRows.Next() {
			if err := typeRows.Scan(&tableType); err != nil {
				tableType = "table"
			}
		}
	}

	return &database.SchemaMetadata{
		TableName: tableName,
		Database:  d.Config.Path,
		Type:      tableType,
		Fields:    fields,
	}, nil
}

// ShowTables 列出所有表
func (d *SQLiteDriver) ShowTables(ctx context.Context) ([]string, error) {
	query := `SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name`

	rows, err := d.DB.QueryContext(ctx, query)
	if err != nil {
		utils.GlobalLogger.LogError("LIST_ERROR", "列出SQLite表失败", err.Error())
		return nil, fmt.Errorf("列出表失败: %s", err)
	}
	defer rows.Close()

	tables := []string{}
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			utils.GlobalLogger.Warn("读取表名失败: %s", err)
			continue
		}
		tables = append(tables, tableName)
	}

	return tables, nil
}