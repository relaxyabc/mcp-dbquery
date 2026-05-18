package mcp

import (
	"encoding/json"
	"fmt"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ToolDefinition MCP工具定义结构
type ToolDefinition struct {
	Name        string             // 工具名称
	Description string             // 工具描述
	InputSchema *jsonschema.Schema // 输入参数Schema（使用官方SDK类型）
}

// PredefinedTools 预定义的工具列表
// 基于contracts/mcp-tools.json定义
var PredefinedTools = []ToolDefinition{
	{
		Name:        "query_mysql_data",
		Description: "在MySQL数据库上执行只读SELECT查询。返回JSON数组格式的查询结果。仅允许SELECT、SHOW、DESCRIBE、EXPLAIN操作。",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"database_id": {Type: "string", Description: "服务器中配置的数据库连接标识符"},
				"query":       {Type: "string", Description: "要执行的SQL SELECT查询。必须以SELECT、SHOW、DESCRIBE或EXPLAIN开头。"},
				"limit":       {Type: "integer", Description: "最大返回行数（默认1000）"},
				"timeout":     {Type: "integer", Description: "查询超时时间（秒，默认30）"},
			},
			Required: []string{"database_id", "query"},
		},
	},
	{
		Name:        "query_mongodb_data",
		Description: "在MongoDB集合上执行只读find查询。返回JSON数组格式的文档结果。仅允许find和aggregate操作。",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"database_id": {Type: "string", Description: "服务器中配置的数据库连接标识符"},
				"collection":  {Type: "string", Description: "要查询的集合名称"},
				"filter":      {Type: "object", Description: "MongoDB过滤查询（bson.M格式）"},
				"limit":       {Type: "integer", Description: "最大返回文档数（默认1000）"},
				"timeout":     {Type: "integer", Description: "查询超时时间（秒，默认30）"},
			},
			Required: []string{"database_id", "collection"},
		},
	},
	{
		Name:        "get_schema",
		Description: "获取表/集合结构元数据，包括字段名称、类型、约束和关系。",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"database_id": {Type: "string", Description: "数据库连接标识符"},
				"table_name":  {Type: "string", Description: "要查看的表或集合名称。如果省略，返回所有表/集合。"},
			},
			Required: []string{"database_id"},
		},
	},
	{
		Name:        "get_indexes",
		Description: "获取表/集合的索引元数据，包括索引名称、索引字段、类型和唯一性。",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"database_id": {Type: "string", Description: "数据库连接标识符"},
				"table_name":  {Type: "string", Description: "要查看的表或集合名称。如果省略，返回所有表/集合的索引。"},
			},
			Required: []string{"database_id"},
		},
	},
	{
		Name:        "list_tables",
		Description: "列出配置的数据库凭据可访问的所有表（MySQL）或集合（MongoDB）。",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"database_id": {Type: "string", Description: "数据库连接标识符"},
			},
			Required: []string{"database_id"},
		},
	},
}

// GetToolDefinitions 获取所有工具定义
func GetToolDefinitions() []ToolDefinition {
	return PredefinedTools
}

// GetToolByName 根据名称获取工具定义
func GetToolByName(name string) *ToolDefinition {
	for _, tool := range PredefinedTools {
		if tool.Name == name {
			return &tool
		}
	}
	return nil
}

// ToolNames 返回所有工具名称列表
func ToolNames() []string {
	names := []string{}
	for _, tool := range PredefinedTools {
		names = append(names, tool.Name)
	}
	return names
}

// ValidateToolArguments 验证工具参数
func ValidateToolArguments(toolName string, args map[string]interface{}) error {
	tool := GetToolByName(toolName)
	if tool == nil {
		return fmt.Errorf("工具不存在: %s", toolName)
	}

	// 检查必需参数
	for _, required := range tool.InputSchema.Required {
		if _, exists := args[required]; !exists {
			return fmt.Errorf("缺少必需参数: %s", required)
		}
	}

	return nil
}

// ToMCPTool 转换ToolDefinition为官方SDK的Tool结构
func (td *ToolDefinition) ToMCPTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        td.Name,
		Description: td.Description,
		InputSchema: td.InputSchema,
	}
}

// GetRequiredParams 获取工具的必需参数列表
func (td *ToolDefinition) GetRequiredParams() []string {
	return td.InputSchema.Required
}

// GetToolSchemaJSON 获取工具Schema的JSON表示
func (td *ToolDefinition) GetToolSchemaJSON() (string, error) {
	data, err := json.MarshalIndent(td.InputSchema, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
