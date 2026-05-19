package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/relaxyabc/mcp-dbquery/src/database"
	"github.com/relaxyabc/mcp-dbquery/src/utils"
)

// DescribeTable 获取表结构信息
func (d *PostgresDriver) DescribeTable(ctx context.Context, tableName string) (*database.SchemaMetadata, error) {
	// 查询表列信息
	query := `
		SELECT
			column_name,
			data_type,
			is_nullable,
			column_default
		FROM information_schema.columns
		WHERE table_name = $1
		AND table_schema = 'public'
		ORDER BY ordinal_position
	`

	rows, err := d.Pool.Query(ctx, query, tableName)
	if err != nil {
		utils.GlobalLogger.LogError("SCHEMA_ERROR", "获取PostgreSQL表结构失败", err.Error())
		return nil, fmt.Errorf("获取表结构失败: %s", err)
	}
	defer rows.Close()

	fields := []database.FieldMetadata{}
	for rows.Next() {
		var colName, colType, isNullable string
		var colDefault sql.NullString

		if err := rows.Scan(&colName, &colType, &isNullable, &colDefault); err != nil {
			utils.GlobalLogger.Warn("读取列信息失败: %s", err)
			continue
		}

		defaultVal := ""
		if colDefault.Valid {
			defaultVal = colDefault.String
		}

		fields = append(fields, database.FieldMetadata{
			Name:         colName,
			Type:         colType,
			Nullable:     strings.ToUpper(isNullable) == "YES",
			DefaultValue: defaultVal,
			PrimaryKey:   false,
			ForeignKey:   nil,
		})
	}

	// 查询主键信息并更新字段
	pkQuery := `
		SELECT a.attname
		FROM pg_index i
		JOIN pg_attribute a ON a.attrelid = i.indrelid AND a.attnum = ANY(i.indkey)
		WHERE i.indrelid = (SELECT oid FROM pg_class WHERE relname = $1)
		AND i.indisprimary
	`

	pkRows, err := d.Pool.Query(ctx, pkQuery, tableName)
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

	// 获取表类型
	tableType := "table"
	typeQuery := `
		SELECT CASE
		 WHEN relkind = 'r' THEN 'table'
		 WHEN relkind = 'v' THEN 'view'
		 WHEN relkind = 'm' THEN 'materialized_view'
		 ELSE 'unknown'
		END as table_type
		FROM pg_class
		WHERE relname = $1
		AND relnamespace = (SELECT oid FROM pg_namespace WHERE nspname = 'public')
	`
	typeRows, err := d.Pool.Query(ctx, typeQuery, tableName)
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
func (d *PostgresDriver) ShowTables(ctx context.Context) ([]string, error) {
	query := `
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = 'public'
		AND table_type = 'BASE TABLE'
		ORDER BY table_name
	`

	rows, err := d.Pool.Query(ctx, query)
	if err != nil {
		utils.GlobalLogger.LogError("LIST_ERROR", "列出PostgreSQL表失败", err.Error())
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