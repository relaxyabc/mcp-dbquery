package database

import (
	"fmt"
	"time"
)

// QueryResult 表示查询的结构化响应
type QueryResult struct {
	RequestID     string                   // 对应请求ID
	Success       bool                     // 查询执行是否成功
	ResultType    QueryType                // 结果数据类型
	Data          []map[string]interface{} // 查询结果行/文档或元数据
	RowCount      int                      // 返回的行数/文档数
	Truncated     bool                     // 是否因限制而被截断
	Warning       string                   // 截断警告消息
	Error         *QueryError              // 失败时的错误详情
	ExecutionTime int                      // 查询执行时间（毫秒）
}

// QueryError 表示查询失败时的错误详情
type QueryError struct {
	Code    string // 错误代码（AUTH_FAILED、QUERY_REJECTED、CONNECTION_ERROR等）
	Message string // 人类可读的错误消息（不得包含密码）
	Details string // 附加上下文信息
}

// NewQueryResult 创建成功的查询结果
func NewQueryResult(requestID string, data []map[string]interface{}, resultType QueryType, executionTime int) *QueryResult {
	return &QueryResult{
		RequestID:     requestID,
		Success:       true,
		ResultType:    resultType,
		Data:          data,
		RowCount:      len(data),
		Truncated:     false,
		ExecutionTime: executionTime,
	}
}

// NewTruncatedResult 创建带截断警告的结果
func NewTruncatedResult(requestID string, data []map[string]interface{}, resultType QueryType, executionTime int, actualCount int) *QueryResult {
	result := NewQueryResult(requestID, data, resultType, executionTime)
	result.Truncated = true
	result.Warning = fmt.Sprintf("结果已截断：返回 %d 行/文档，共 %d 行/文档（限制：1000）",
		len(data), actualCount)
	return result
}

// NewErrorResult 创建失败的查询结果
func NewErrorResult(requestID string, code, message string) *QueryResult {
	return &QueryResult{
		RequestID: requestID,
		Success:   false,
		Error: &QueryError{
			Code:    code,
			Message: message,
		},
	}
}

// NewErrorResultWithDetails 创建带详情的失败查询结果
func NewErrorResultWithDetails(requestID string, code, message, details string) *QueryResult {
	result := NewErrorResult(requestID, code, message)
	result.Error.Details = details
	return result
}

// ToJSON 将结果转换为JSON友好格式（用于MCP响应）
func (r *QueryResult) ToJSON() map[string]interface{} {
	result := map[string]interface{}{
		"requestId":     r.RequestID,
		"success":       r.Success,
		"resultType":    r.ResultType,
		"executionTime": r.ExecutionTime,
	}

	if r.Success {
		result["data"] = r.Data
		result["rowCount"] = r.RowCount
		if r.Truncated {
			result["truncated"] = true
			result["warning"] = r.Warning
		}
	} else {
		result["error"] = map[string]interface{}{
			"code":    r.Error.Code,
			"message": r.Error.Message,
		}
		if r.Error.Details != "" {
			result["error"].(map[string]interface{})["details"] = r.Error.Details
		}
	}

	return result
}

// Validate 检查结果完整性
func (r *QueryResult) Validate() error {
	if r.RequestID == "" {
		return fmt.Errorf("结果必须包含请求ID")
	}
	if r.Success && r.Data == nil && r.RowCount > 0 {
		return fmt.Errorf("成功结果rowCount > 0时必须有数据")
	}
	if !r.Success && r.Error == nil {
		return fmt.Errorf("失败结果必须有错误详情")
	}
	return nil
}

// MeasureExecutionTime 测量执行时间的辅助函数
func MeasureExecutionTime(start time.Time) int {
	return int(time.Since(start).Milliseconds())
}
