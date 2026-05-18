package mysql

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/relaxyabc/mcp-dbquery/src/database"
)

// ExecuteSelectQuery 执行SELECT查询并返回结果
func (d *MySQLDriver) ExecuteSelectQuery(ctx context.Context, query string, limit int) (*database.QueryResult, error) {
	// 添加LIMIT限制（如果查询中没有）
	if !strings.Contains(strings.ToUpper(query), "LIMIT") && limit > 0 {
		query = fmt.Sprintf("%s LIMIT %d", strings.TrimRight(query, ";"), limit)
	}

	return d.ExecuteQuery(ctx, query, limit, time.Duration(d.Config.Timeout)*time.Second)
}

// ExecuteShowQuery 执行SHOW命令查询
func (d *MySQLDriver) ExecuteShowQuery(ctx context.Context, query string) (*database.QueryResult, error) {
	return d.ExecuteQuery(ctx, query, 1000, time.Duration(d.Config.Timeout)*time.Second)
}

// DescribeTable 描述表结构（DESCRIBE命令）
func (d *MySQLDriver) DescribeTable(ctx context.Context, tableName string) (*database.SchemaMetadata, error) {
	query := fmt.Sprintf("DESCRIBE %s", tableName)
	result, err := d.ExecuteQuery(ctx, query, 1000, time.Duration(d.Config.Timeout)*time.Second)
	if err != nil {
		return nil, err
	}

	// 将结果转换为SchemaMetadata
	schema := database.NewSchemaMetadata(tableName, d.Config.Database, "table")

	for _, row := range result.Data {
		field := database.FieldMetadata{
			Name:         getStringValue(row, "Field"),
			Type:         getStringValue(row, "Type"),
			Nullable:     getStringValue(row, "Null") == "YES",
			DefaultValue: getStringValue(row, "Default"),
			PrimaryKey:   getStringValue(row, "Key") == "PRI",
		}
		schema.AddField(field)
	}

	return schema, nil
}

// ShowTables 显示所有表列表
func (d *MySQLDriver) ShowTables(ctx context.Context) ([]string, error) {
	query := "SHOW TABLES"
	result, err := d.ExecuteQuery(ctx, query, 1000, time.Duration(d.Config.Timeout)*time.Second)
	if err != nil {
		return nil, err
	}

	tables := []string{}
	for _, row := range result.Data {
		// SHOW TABLES返回单列，列名可能是Tables_in_xxx
		for _, value := range row {
			if tableName, ok := value.(string); ok {
				tables = append(tables, tableName)
				break
			}
		}
	}

	return tables, nil
}

// ShowIndexes 显示表的索引信息
func (d *MySQLDriver) ShowIndexes(ctx context.Context, tableName string) (*database.IndexListMetadata, error) {
	query := fmt.Sprintf("SHOW INDEX FROM %s", tableName)
	result, err := d.ExecuteQuery(ctx, query, 1000, time.Duration(d.Config.Timeout)*time.Second)
	if err != nil {
		return nil, err
	}

	indexList := database.NewIndexListMetadata(tableName)

	// SHOW INDEX返回多行，每个索引字段一行，需要聚合
	indexMap := make(map[string]*database.IndexMetadata)

	for _, row := range result.Data {
		indexName := getStringValue(row, "Key_name")
		columnName := getStringValue(row, "Column_name")
		nonUnique := getIntValue(row, "Non_unique")
		indexType := getStringValue(row, "Index_type")

		// 创建或获取索引元数据
		if _, exists := indexMap[indexName]; !exists {
			indexMap[indexName] = database.NewIndexMetadata(indexName, tableName, d.Config.Database)
			indexMap[indexName].Type = indexType
			indexMap[indexName].SetUnique(nonUnique == 0)
		}

		// 添加索引字段
		_ = getIntValue(row, "Seq_in_index") // 用于确定字段顺序
		order := "ASC"                       // MySQL默认ASC
		indexMap[indexName].AddField(columnName, order)
	}

	// 转换为列表
	for _, index := range indexMap {
		indexList.AddIndex(*index)
	}

	return indexList, nil
}

// ExecuteWithTimeout 执行带超时的查询
func (d *MySQLDriver) ExecuteWithTimeout(ctx context.Context, query string, timeoutSeconds int) (*database.QueryResult, error) {
	return d.ExecuteQuery(ctx, query, 1000, time.Duration(timeoutSeconds)*time.Second)
}

// 辅助函数：获取字符串值
func getStringValue(row map[string]interface{}, key string) string {
	if val, exists := row[key]; exists {
		if str, ok := val.(string); ok {
			return str
		}
		if b, ok := val.([]byte); ok {
			return string(b)
		}
		return fmt.Sprintf("%v", val)
	}
	return ""
}

// 辅助函数：获取整数值
func getIntValue(row map[string]interface{}, key string) int {
	if val, exists := row[key]; exists {
		switch v := val.(type) {
		case int:
			return v
		case int64:
			return int(v)
		case float64:
			return int(v)
		case string:
			// 尝试解析字符串
			var intValue int
			fmt.Sscanf(v, "%d", &intValue)
			return intValue
		default:
			return 0
		}
	}
	return 0
}

// Ping 测试连接是否正常
func (d *MySQLDriver) Ping(ctx context.Context) error {
	if d.DB == nil {
		return fmt.Errorf("数据库连接未建立")
	}

	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	return d.DB.PingContext(ctx)
}

// GetConnectionStatus 获取连接状态信息
func (d *MySQLDriver) GetConnectionStatus() map[string]interface{} {
	stats := d.DB.Stats()

	return map[string]interface{}{
		"id":             d.ID,
		"type":           "mysql",
		"state":          string(d.State),
		"host":           d.Config.Host,
		"port":           d.Config.Port,
		"database":       d.Config.Database,
		"max_open_conns": stats.MaxOpenConnections,
		"open_conns":     stats.OpenConnections,
		"in_use":         stats.InUse,
		"idle":           stats.Idle,
		"wait_count":     stats.WaitCount,
		"wait_duration":  stats.WaitDuration.Milliseconds(),
	}
}
