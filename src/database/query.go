package database

import (
	"fmt"
	"time"
)

// QueryType 表示查询操作类型
type QueryType string

const (
	QueryTypeData    QueryType = "data"    // 数据查询
	QueryTypeSchema  QueryType = "schema"  // 结构查询
	QueryTypeIndexes QueryType = "indexes" // 索引查询
	QueryTypeList    QueryType = "list"    // 列表查询
)

// QueryRequest 表示MCP客户端的数据库查询请求
type QueryRequest struct {
	RequestID  string                 // JSON-RPC请求标识符
	DatabaseID string                 // 目标数据库连接ID
	Query      string                 // SQL查询或MongoDB查询表达式
	QueryType  QueryType              // 查询操作类型
	Filters    map[string]interface{} // 可选过滤条件
	Limit      int                    // 最大返回行数/文档数（默认1000，最大1000）
	Timeout    int                    // 查询超时时间（秒）
	APIKey     string                 // 认证API密钥
}

// Validate 检查查询请求的必填字段和约束条件
func (q *QueryRequest) Validate() error {
	if q.RequestID == "" {
		return fmt.Errorf("请求ID不能为空")
	}
	if q.DatabaseID == "" {
		return fmt.Errorf("数据库ID不能为空")
	}
	if q.Query == "" {
		return fmt.Errorf("查询语句不能为空")
	}
	if q.APIKey == "" {
		return fmt.Errorf("API密钥认证必须提供")
	}

	// 强制限制约束（FR-010）
	if q.Limit <= 0 {
		q.Limit = 1000 // 默认值
	}
	if q.Limit > 1000 {
		q.Limit = 1000 // 最大值强制
	}

	// 强制超时约束
	if q.Timeout <= 0 {
		q.Timeout = 30 // 默认值
	}
	if q.Timeout > 300 {
		q.Timeout = 300 // 最大值
	}

	return nil
}

// GetTimeoutDuration 返回超时时间作为time.Duration类型
func (q *QueryRequest) GetTimeoutDuration() time.Duration {
	return time.Duration(q.Timeout) * time.Second
}

// NewQueryRequest 创建带有默认值的查询请求
func NewQueryRequest(requestID, databaseID, query string, queryType QueryType) *QueryRequest {
	return &QueryRequest{
		RequestID:  requestID,
		DatabaseID: databaseID,
		Query:      query,
		QueryType:  queryType,
		Limit:      1000,
		Timeout:    30,
	}
}
