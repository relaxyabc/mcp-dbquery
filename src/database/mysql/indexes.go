package mysql

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/relaxyabc/mcp-dbquery/src/database"
	"github.com/relaxyabc/mcp-dbquery/src/utils"
)

// GetIndexes 已在schema.go中实现，这里添加额外辅助方法

// GetPrimaryKey 获取表的主键信息
func (d *MySQLDriver) GetPrimaryKey(ctx context.Context, tableName string) (*database.IndexMetadata, error) {
	utils.GlobalLogger.Info("获取MySQL主键 [连接=%s] [表=%s]", d.ID, tableName)

	query := fmt.Sprintf("SHOW INDEX FROM %s WHERE Key_name = 'PRIMARY'", tableName)
	result, err := d.ExecuteQuery(ctx, query, 100, 5*time.Second)
	if err != nil {
		return nil, err
	}

	if len(result.Data) == 0 {
		return nil, fmt.Errorf("表 %s 没有主键", tableName)
	}

	// 构建主键索引信息
	primaryKey := database.NewIndexMetadata("PRIMARY", tableName, d.Config.Database)
	primaryKey.Type = "BTREE"
	primaryKey.SetUnique(true)

	for _, row := range result.Data {
		columnName := getStringValue(row, "Column_name")
		primaryKey.AddField(columnName, "ASC")
	}

	utils.GlobalLogger.Info("主键获取成功 [表=%s] [字段数=%d]", tableName, len(primaryKey.Fields))
	return primaryKey, nil
}

// GetUniqueIndexes 获取表的唯一索引
func (d *MySQLDriver) GetUniqueIndexes(ctx context.Context, tableName string) ([]database.IndexMetadata, error) {
	utils.GlobalLogger.Info("获取MySQL唯一索引 [连接=%s] [表=%s]", d.ID, tableName)

	query := fmt.Sprintf("SHOW INDEX FROM %s WHERE Non_unique = 0", tableName)
	result, err := d.ExecuteQuery(ctx, query, 100, 5*time.Second)
	if err != nil {
		return nil, err
	}

	// 聚合索引字段
	indexMap := make(map[string]*database.IndexMetadata)

	for _, row := range result.Data {
		indexName := getStringValue(row, "Key_name")
		if indexName == "PRIMARY" {
			continue // 跳过主键
		}

		columnName := getStringValue(row, "Column_name")
		indexType := getStringValue(row, "Index_type")

		if _, exists := indexMap[indexName]; !exists {
			indexMap[indexName] = database.NewIndexMetadata(indexName, tableName, d.Config.Database)
			indexMap[indexName].Type = indexType
			indexMap[indexName].SetUnique(true)
		}

		indexMap[indexName].AddField(columnName, "ASC")
	}

	// 转换为列表
	indexes := []database.IndexMetadata{}
	for _, index := range indexMap {
		indexes = append(indexes, *index)
	}

	utils.GlobalLogger.Info("唯一索引获取成功 [表=%s] [数量=%d]", tableName, len(indexes))
	return indexes, nil
}

// GetNonUniqueIndexes 获取表的非唯一索引
func (d *MySQLDriver) GetNonUniqueIndexes(ctx context.Context, tableName string) ([]database.IndexMetadata, error) {
	utils.GlobalLogger.Info("获取MySQL非唯一索引 [连接=%s] [表=%s]", d.ID, tableName)

	query := fmt.Sprintf("SHOW INDEX FROM %s WHERE Non_unique = 1", tableName)
	result, err := d.ExecuteQuery(ctx, query, 100, 5*time.Second)
	if err != nil {
		return nil, err
	}

	// 聚合索引字段
	indexMap := make(map[string]*database.IndexMetadata)

	for _, row := range result.Data {
		indexName := getStringValue(row, "Key_name")
		columnName := getStringValue(row, "Column_name")
		indexType := getStringValue(row, "Index_type")

		if _, exists := indexMap[indexName]; !exists {
			indexMap[indexName] = database.NewIndexMetadata(indexName, tableName, d.Config.Database)
			indexMap[indexName].Type = indexType
			indexMap[indexName].SetUnique(false)
		}

		indexMap[indexName].AddField(columnName, "ASC")
	}

	// 转换为列表
	indexes := []database.IndexMetadata{}
	for _, index := range indexMap {
		indexes = append(indexes, *index)
	}

	utils.GlobalLogger.Info("非唯一索引获取成功 [表=%s] [数量=%d]", tableName, len(indexes))
	return indexes, nil
}

// CheckIndexExists 检查索引是否存在
func (d *MySQLDriver) CheckIndexExists(ctx context.Context, tableName, indexName string) (bool, error) {
	query := fmt.Sprintf("SHOW INDEX FROM %s WHERE Key_name = '%s'", tableName, indexName)
	result, err := d.ExecuteQuery(ctx, query, 1, 2*time.Second)
	if err != nil {
		return false, err
	}

	return len(result.Data) > 0, nil
}

// GetIndexType 根索引名称获取索引类型
func GetIndexType(indexType string) string {
	switch strings.ToUpper(indexType) {
	case "BTREE":
		return "BTREE"
	case "HASH":
		return "HASH"
	case "FULLTEXT":
		return "FULLTEXT"
	case "RTREE":
		return "RTREE"
	default:
		return indexType
	}
}

// IsIndexTypeValid 检查索引类型是否有效
func IsIndexTypeValid(indexType string) bool {
	validTypes := []string{"BTREE", "HASH", "FULLTEXT", "RTREE"}
	for _, valid := range validTypes {
		if strings.ToUpper(indexType) == valid {
			return true
		}
	}
	return false
}
