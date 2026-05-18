package database

// SchemaMetadata 表示表/集合结构信息
type SchemaMetadata struct {
	TableName string          // 表/集合名称
	Database  string          // 数据库名称
	Type      string          // "table"（MySQL）或"collection"（MongoDB）
	Fields    []FieldMetadata // 字段/列定义列表
}

// FieldMetadata 表示字段/列元数据
type FieldMetadata struct {
	Name         string         // 字段/列名称
	Type         string         // 数据类型（如INT、VARCHAR、STRING、OBJECT等）
	Nullable     bool           // 是否允许为空（MySQL）
	DefaultValue string         // 默认值（如有）
	PrimaryKey   bool           // 是否为主键（MySQL）
	ForeignKey   *ForeignKeyRef // 外键引用（如有）
	Validation   interface{}    // MongoDB验证规则（如有）
}

// ForeignKeyRef 表示外键引用信息
type ForeignKeyRef struct {
	ReferencedTable string // 引用的表名
	ReferencedField string // 引用的字段名
	OnDelete        string // 删除时行为
	OnUpdate        string // 更新时行为
}

// NewSchemaMetadata 创建新的结构元数据
func NewSchemaMetadata(tableName, database, tableType string) *SchemaMetadata {
	return &SchemaMetadata{
		TableName: tableName,
		Database:  database,
		Type:      tableType,
		Fields:    []FieldMetadata{},
	}
}

// AddField 添加字段元数据
func (s *SchemaMetadata) AddField(field FieldMetadata) {
	s.Fields = append(s.Fields, field)
}

// ToJSON 转换为JSON格式（用于MCP响应）
func (s *SchemaMetadata) ToJSON() map[string]interface{} {
	fields := make([]map[string]interface{}, len(s.Fields))
	for i, f := range s.Fields {
		fieldMap := map[string]interface{}{
			"name":     f.Name,
			"type":     f.Type,
			"nullable": f.Nullable,
		}
		if f.DefaultValue != "" {
			fieldMap["defaultValue"] = f.DefaultValue
		}
		if f.PrimaryKey {
			fieldMap["primaryKey"] = true
		}
		if f.ForeignKey != nil {
			fieldMap["foreignKey"] = map[string]interface{}{
				"table": f.ForeignKey.ReferencedTable,
				"field": f.ForeignKey.ReferencedField,
			}
		}
		if f.Validation != nil {
			fieldMap["validation"] = f.Validation
		}
		fields[i] = fieldMap
	}

	return map[string]interface{}{
		"tableName":  s.TableName,
		"database":   s.Database,
		"type":       s.Type,
		"fields":     fields,
		"fieldCount": len(s.Fields),
	}
}
