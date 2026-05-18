package database

// IndexMetadata 表示表/集合的索引信息
type IndexMetadata struct {
	IndexName string       // 索引名称
	TableName string       // 表/集合名称
	Database  string       // 数据库名称
	Fields    []IndexField // 索引字段列表
	Type      string       // 索引类型
	Unique    bool         // 是否唯一索引
	Sparse    bool         // 是否稀疏索引（仅MongoDB）
}

// IndexField 表示索引字段定义
type IndexField struct {
	Name  string // 索引字段名称
	Order string // 排序顺序："ASC"/"DESC"（MySQL）或"1"/"-1"（MongoDB）
}

// NewIndexMetadata 创建新的索引元数据
func NewIndexMetadata(indexName, tableName, database string) *IndexMetadata {
	return &IndexMetadata{
		IndexName: indexName,
		TableName: tableName,
		Database:  database,
		Fields:    []IndexField{},
		Unique:    false,
		Sparse:    false,
	}
}

// AddField 添加索引字段
func (i *IndexMetadata) AddField(name, order string) {
	i.Fields = append(i.Fields, IndexField{
		Name:  name,
		Order: order,
	})
}

// SetUnique 设置唯一索引属性
func (i *IndexMetadata) SetUnique(unique bool) {
	i.Unique = unique
}

// SetSparse 设置稀疏索引属性（仅MongoDB）
func (i *IndexMetadata) SetSparse(sparse bool) {
	i.Sparse = sparse
}

// ToJSON 转换为JSON格式（用于MCP响应）
func (i *IndexMetadata) ToJSON() map[string]interface{} {
	fields := make([]map[string]interface{}, len(i.Fields))
	for idx, f := range i.Fields {
		fields[idx] = map[string]interface{}{
			"name":  f.Name,
			"order": f.Order,
		}
	}

	result := map[string]interface{}{
		"indexName":  i.IndexName,
		"tableName":  i.TableName,
		"database":   i.Database,
		"fields":     fields,
		"type":       i.Type,
		"unique":     i.Unique,
		"fieldCount": len(i.Fields),
	}

	// MongoDB特有属性
	if i.Sparse {
		result["sparse"] = true
	}

	return result
}

// IndexListMetadata 表示多个索引的列表
type IndexListMetadata struct {
	TableName string          // 表/集合名称
	Indexes   []IndexMetadata // 索引列表
}

// NewIndexListMetadata 创建索引列表元数据
func NewIndexListMetadata(tableName string) *IndexListMetadata {
	return &IndexListMetadata{
		TableName: tableName,
		Indexes:   []IndexMetadata{},
	}
}

// AddIndex 添加索引
func (l *IndexListMetadata) AddIndex(index IndexMetadata) {
	l.Indexes = append(l.Indexes, index)
}

// ToJSON 转换为JSON格式
func (l *IndexListMetadata) ToJSON() map[string]interface{} {
	indexes := make([]map[string]interface{}, len(l.Indexes))
	for i, idx := range l.Indexes {
		indexes[i] = idx.ToJSON()
	}

	return map[string]interface{}{
		"tableName":  l.TableName,
		"indexes":    indexes,
		"indexCount": len(l.Indexes),
	}
}
