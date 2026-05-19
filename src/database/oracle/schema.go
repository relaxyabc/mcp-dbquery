package oracle

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/relaxyabc/mcp-dbquery/src/database"
	"github.com/relaxyabc/mcp-dbquery/src/utils"
)

// DescribeTable 获取表结构信息（使用ALL_TAB_COLUMNS系统视图）
func (d *OracleDriver) DescribeTable(ctx context.Context, tableName string) (*database.SchemaMetadata, error) {
	// 查询表列信息（使用ALL_TAB_COLUMNS）
	// 注意：Oracle列名区分大小写，默认存储为大写
	query := `
		SELECT
			column_name,
			data_type ||
				CASE
					WHEN data_type IN ('VARCHAR2', 'NVARCHAR2', 'CHAR', 'NCHAR') THEN '(' || data_length || ')'
					WHEN data_type = 'NUMBER' AND data_precision IS NOT NULL THEN '(' || data_precision || ',' || data_scale || ')'
					ELSE ''
				END AS data_type,
			nullable,
			data_default
		FROM all_tab_columns
		WHERE table_name = UPPER(:1)
		AND owner = USER
		ORDER BY column_id
	`

	rows, err := d.DB.QueryContext(ctx, query, tableName)
	if err != nil {
		utils.GlobalLogger.LogError("SCHEMA_ERROR", "获取Oracle表结构失败", err.Error())
		return nil, fmt.Errorf("获取表结构失败: %s", err)
	}
	defer rows.Close()

	fields := []database.FieldMetadata{}
	for rows.Next() {
		var colName, colType, isNullable string
		var defaultVal sql.NullString

		if err := rows.Scan(&colName, &colType, &isNullable, &defaultVal); err != nil {
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
			Nullable:     isNullable == "Y",
			DefaultValue: defaultStr,
			PrimaryKey:   false, // 主键信息需要额外查询
			ForeignKey:   nil,
		})
	}

	// 查询主键信息
	pkQuery := `
		SELECT cols.column_name
		FROM all_constraints cons
		JOIN all_cons_columns cols ON cons.constraint_name = cols.constraint_name
		WHERE cons.constraint_type = 'P'
		AND cons.table_name = UPPER(:1)
		AND cons.owner = USER
		ORDER BY cols.position
	`

	pkRows, err := d.DB.QueryContext(ctx, pkQuery, tableName)
	if err == nil {
		defer pkRows.Close()
		pkCols := []string{}
		for pkRows.Next() {
			var pkCol string
			if err := pkRows.Scan(&pkCol); err == nil {
				pkCols = append(pkCols, pkCol)
			}
		}
		// 更新主键标记
		for i := range fields {
			for _, pkCol := range pkCols {
				if fields[i].Name == pkCol {
					fields[i].PrimaryKey = true
				}
			}
		}
	}

	// 查询外键信息
	fkQuery := `
		SELECT
			cols.column_name,
			ref_cons.table_name AS referenced_table,
			ref_cols.column_name AS referenced_column
		FROM all_constraints cons
		JOIN all_cons_columns cols ON cons.constraint_name = cols.constraint_name
		JOIN all_constraints ref_cons ON cons.r_constraint_name = ref_cons.constraint_name
		JOIN all_cons_columns ref_cols ON ref_cons.constraint_name = ref_cols.constraint_name
		WHERE cons.constraint_type = 'R'
		AND cons.table_name = UPPER(:1)
		AND cons.owner = USER
		AND cols.position = ref_cols.position
	`

	fkRows, err := d.DB.QueryContext(ctx, fkQuery, tableName)
	if err == nil {
		defer fkRows.Close()
		for fkRows.Next() {
			var colName, refTable, refColumn string

			if err := fkRows.Scan(&colName, &refTable, &refColumn); err != nil {
				utils.GlobalLogger.Warn("读取外键信息失败: %s", err)
				continue
			}

			// 更新字段的外键信息
			for i := range fields {
				if fields[i].Name == colName {
					fields[i].ForeignKey = &database.ForeignKeyRef{
						ReferencedTable: refTable,
						ReferencedField: refColumn,
						OnDelete:        "", // Oracle外键行为需要额外查询
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
			WHEN object_type = 'TABLE' THEN 'table'
			WHEN object_type = 'VIEW' THEN 'view'
			ELSE 'unknown'
		END AS table_type
		FROM all_objects
		WHERE object_name = UPPER(:1)
		AND owner = USER
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
		Database:  d.Config.Username, // Oracle使用用户名作为Schema
		Type:      tableType,
		Fields:    fields,
	}, nil
}

// ShowTables 列出所有表
func (d *OracleDriver) ShowTables(ctx context.Context) ([]string, error) {
	query := `
		SELECT object_name
		FROM all_objects
		WHERE object_type = 'TABLE'
		AND owner = USER
		ORDER BY object_name
	`

	rows, err := d.DB.QueryContext(ctx, query)
	if err != nil {
		utils.GlobalLogger.LogError("LIST_ERROR", "列出Oracle表失败", err.Error())
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