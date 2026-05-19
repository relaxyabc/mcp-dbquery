package sqlserver

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/relaxyabc/mcp-dbquery/src/database"
	"github.com/relaxyabc/mcp-dbquery/src/utils"
)

// DescribeTable 获取表结构信息（使用sys.columns系统视图）
func (d *SQLServerDriver) DescribeTable(ctx context.Context, tableName string) (*database.SchemaMetadata, error) {
	// 查询表列信息（使用sys.columns和sys.types）
	query := `
		SELECT
			c.name AS column_name,
			t.name AS data_type,
			c.is_nullable,
			c.column_id,
			OBJECT_DEFINITION(c.default_object_id) AS default_value,
			CASE WHEN pk.column_id IS NOT NULL THEN 1 ELSE 0 END AS is_primary_key
		FROM sys.columns c
		JOIN sys.types t ON c.user_type_id = t.user_type_id
		JOIN sys.tables tab ON c.object_id = tab.object_id
		LEFT JOIN (
			SELECT ic.object_id, ic.column_id
			FROM sys.index_columns ic
			JOIN sys.indexes i ON ic.object_id = i.object_id AND ic.index_id = i.index_id
			WHERE i.is_primary_key = 1
		) pk ON c.object_id = pk.object_id AND c.column_id = pk.column_id
		WHERE tab.name = @p1
		AND SCHEMA_NAME(tab.schema_id) = 'dbo'
		ORDER BY c.column_id
	`

	rows, err := d.DB.QueryContext(ctx, query, tableName)
	if err != nil {
		utils.GlobalLogger.LogError("SCHEMA_ERROR", "获取SQL Server表结构失败", err.Error())
		return nil, fmt.Errorf("获取表结构失败: %s", err)
	}
	defer rows.Close()

	fields := []database.FieldMetadata{}
	for rows.Next() {
		var colName, colType string
		var isNullable, colID, isPK int
		var defaultVal sql.NullString

		if err := rows.Scan(&colName, &colType, &isNullable, &colID, &defaultVal, &isPK); err != nil {
			utils.GlobalLogger.Warn("读取列信息失败: %s", err)
			continue
		}

		defaultStr := ""
		if defaultVal.Valid {
			defaultStr = defaultVal.String
		}

		fields = append(fields, database.FieldMetadata{
			Name:         colName,
			Type:         colType,
			Nullable:     isNullable == 1,
			DefaultValue: defaultStr,
			PrimaryKey:   isPK == 1,
			ForeignKey:   nil,
		})
	}

	// 查询外键信息
	fkQuery := `
		SELECT
			fk.name AS fk_name,
			parent_col.name AS column_name,
			ref_table.name AS referenced_table,
			ref_col.name AS referenced_column
		FROM sys.foreign_keys fk
		JOIN sys.foreign_key_columns fkc ON fk.object_id = fkc.constraint_object_id
		JOIN sys.tables parent_table ON fkc.parent_object_id = parent_table.object_id
		JOIN sys.columns parent_col ON fkc.parent_object_id = parent_col.object_id AND fkc.parent_column_id = parent_col.column_id
		JOIN sys.tables ref_table ON fkc.referenced_object_id = ref_table.object_id
		JOIN sys.columns ref_col ON fkc.referenced_object_id = ref_col.object_id AND fkc.referenced_column_id = ref_col.column_id
		WHERE parent_table.name = @p1
		AND SCHEMA_NAME(parent_table.schema_id) = 'dbo'
	`

	fkRows, err := d.DB.QueryContext(ctx, fkQuery, tableName)
	if err == nil {
		defer fkRows.Close()
		for fkRows.Next() {
			var fkName, colName, refTable, refColumn string

			if err := fkRows.Scan(&fkName, &colName, &refTable, &refColumn); err != nil {
				utils.GlobalLogger.Warn("读取外键信息失败: %s", err)
				continue
			}

			// 更新字段的外键信息
			for i := range fields {
				if fields[i].Name == colName {
					fields[i].ForeignKey = &database.ForeignKeyRef{
						ReferencedTable: refTable,
						ReferencedField: refColumn,
						OnDelete:        "", // SQL Server外键行为需要额外查询
						OnUpdate:        "",
					}
				}
			}
		}
	}

	// 获取表类型
	tableType := "table"
	typeQuery := `
		SELECT CASE
			WHEN tab.type = 'U' THEN 'table'
			WHEN tab.type = 'V' THEN 'view'
			ELSE 'unknown'
		END AS table_type
		FROM sys.tables tab
		WHERE tab.name = @p1
		AND SCHEMA_NAME(tab.schema_id) = 'dbo'
	`
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
		Database:  d.Config.Database,
		Type:      tableType,
		Fields:    fields,
	}, nil
}

// ShowTables 列出所有表
func (d *SQLServerDriver) ShowTables(ctx context.Context) ([]string, error) {
	query := `
		SELECT name
		FROM sys.tables
		WHERE SCHEMA_NAME(schema_id) = 'dbo'
		ORDER BY name
	`

	rows, err := d.DB.QueryContext(ctx, query)
	if err != nil {
		utils.GlobalLogger.LogError("LIST_ERROR", "列出SQL Server表失败", err.Error())
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