package api

import (
	"encoding/json"
	"net/http"
)

// 错误代码定义
// 用于API响应中的标准化错误识别
const (
	// 认证错误
	ErrCodeAuthFailed     = "AUTH_FAILED"      // 认证失败
	ErrCodeInvalidAPIKey  = "INVALID_API_KEY"  // API密钥无效
	ErrCodeAPIKeyExpired  = "API_KEY_EXPIRED"  // API密钥过期
	ErrCodeAPIKeyDisabled = "API_KEY_DISABLED" // API密钥停用

	// 请求错误
	ErrCodeInvalidRequest = "INVALID_REQUEST" // 无效请求
	ErrCodeMissingParam   = "MISSING_PARAM"   // 缺少参数
	ErrCodeInvalidParam   = "INVALID_PARAM"   // 参数无效

	// 查询错误
	ErrCodeQueryRejected = "QUERY_REJECTED" // 查询被拒绝（只读违规）
	ErrCodeQueryTimeout  = "QUERY_TIMEOUT"  // 查询超时
	ErrCodeQueryError    = "QUERY_ERROR"    // 查询执行错误

	// 连接错误
	ErrCodeConnectionError = "CONNECTION_ERROR" // 连接错误
	ErrCodePoolExhausted   = "POOL_EXHAUSTED"   // 连接池耗尽
	ErrCodeDBNotConnected  = "DB_NOT_CONNECTED" // 数据库未连接

	// 资源错误
	ErrCodeNotFound      = "NOT_FOUND"       // 资源未找到
	ErrCodeDBNotFound    = "DB_NOT_FOUND"    // 数据库配置未找到
	ErrCodeTableNotFound = "TABLE_NOT_FOUND" // 表未找到
	ErrCodeToolNotFound  = "TOOL_NOT_FOUND"  // 工具未找到

	// 内部错误
	ErrCodeInternalError = "INTERNAL_ERROR" // 内部错误
	ErrCodeJSONError     = "JSON_ERROR"     // JSON解析错误
	ErrCodeSchemaError   = "SCHEMA_ERROR"   // Schema推断错误
)

// Response HTTP响应结构
type Response struct {
	Success  bool                   `json:"success"`            // 是否成功
	Data     interface{}            `json:"data,omitempty"`     // 响应数据
	Error    *ErrorResponse         `json:"error,omitempty"`    // 错误信息
	Metadata map[string]interface{} `json:"metadata,omitempty"` // 元数据（如执行时间）
}

// ErrorResponse 错误响应结构
type ErrorResponse struct {
	Code    string `json:"code"`              // 错误代码（使用上述常量）
	Message string `json:"message"`           // 用户友好的错误消息
	Details string `json:"details,omitempty"` // 技术详细信息（可选）
}

// HealthResponse 健康检查响应结构
type HealthResponse struct {
	Status      string                 `json:"status"`         // 状态（healthy/degraded/unhealthy）
	Timestamp   string                 `json:"timestamp"`      // 时间戳
	Version     string                 `json:"version"`        // 版本号
	Uptime      int64                  `json:"uptime_seconds"` // 运行时间（秒）
	Connections map[string]interface{} `json:"connections"`    // 连接池状态
}

// QueryResultResponse 查询结果响应结构
type QueryResultResponse struct {
	DatabaseID  string                   `json:"database_id"`  // 数据库标识
	QueryType   string                   `json:"query_type"`   // 查询类型
	RowCount    int                      `json:"row_count"`    // 返回行数
	Data        []map[string]interface{} `json:"data"`         // 查询数据
	Truncated   bool                     `json:"truncated"`    // 是否被截断
	ExecutionMs int                      `json:"execution_ms"` // 执行时间（毫秒）
}

// NewSuccessResponse 创建成功响应
func NewSuccessResponse(data interface{}) *Response {
	return &Response{
		Success: true,
		Data:    data,
	}
}

// NewErrorResponse 创建错误响应
func NewErrorResponse(code, message string) *Response {
	return &Response{
		Success: false,
		Error: &ErrorResponse{
			Code:    code,
			Message: message,
		},
	}
}

// NewErrorResponseWithDetails 创建带详情的错误响应
func NewErrorResponseWithDetails(code, message, details string) *Response {
	return &Response{
		Success: false,
		Error: &ErrorResponse{
			Code:    code,
			Message: message,
			Details: details,
		},
	}
}

// WithMetadata 添加元数据
func (r *Response) WithMetadata(metadata map[string]interface{}) *Response {
	r.Metadata = metadata
	return r
}

// WriteJSON 将响应写入HTTP响应流
func (r *Response) WriteJSON(w http.ResponseWriter, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(r)
}

// WriteSuccess 写入成功响应（状态码200）
func (r *Response) WriteSuccess(w http.ResponseWriter) {
	r.WriteJSON(w, http.StatusOK)
}

// WriteError 写入错误响应
func (r *Response) WriteError(w http.ResponseWriter, statusCode int) {
	r.WriteJSON(w, statusCode)
}

// 预定义的错误响应函数

// RespondUnauthorized 返回401未授权响应
func RespondUnauthorized(w http.ResponseWriter, message string) {
	NewErrorResponse(ErrCodeAuthFailed, message).WriteError(w, http.StatusUnauthorized)
}

// RespondInvalidAPIKey 返回401无效API密钥响应
func RespondInvalidAPIKey(w http.ResponseWriter, message string) {
	NewErrorResponse(ErrCodeInvalidAPIKey, message).WriteError(w, http.StatusUnauthorized)
}

// RespondAPIKeyExpired 返回401 API密钥过期响应
func RespondAPIKeyExpired(w http.ResponseWriter, message string) {
	NewErrorResponse(ErrCodeAPIKeyExpired, message).WriteError(w, http.StatusUnauthorized)
}

// RespondBadRequest 返回400错误请求响应
func RespondBadRequest(w http.ResponseWriter, message string) {
	NewErrorResponse(ErrCodeInvalidRequest, message).WriteError(w, http.StatusBadRequest)
}

// RespondMissingParam 返回400缺少参数响应
func RespondMissingParam(w http.ResponseWriter, paramName string) {
	NewErrorResponseWithDetails(ErrCodeMissingParam,
		"缺少必需参数",
		"参数名: "+paramName).WriteError(w, http.StatusBadRequest)
}

// RespondInvalidParam 返回400参数无效响应
func RespondInvalidParam(w http.ResponseWriter, paramName, reason string) {
	NewErrorResponseWithDetails(ErrCodeInvalidParam,
		"参数无效",
		"参数名: "+paramName+", 原因: "+reason).WriteError(w, http.StatusBadRequest)
}

// RespondForbidden 返回403禁止访问响应（只读操作违规）
func RespondForbidden(w http.ResponseWriter, message string) {
	NewErrorResponse(ErrCodeQueryRejected, message).WriteError(w, http.StatusForbidden)
}

// RespondReadOnlyViolation 返回403只读违规响应
func RespondReadOnlyViolation(w http.ResponseWriter, operation string) {
	NewErrorResponseWithDetails(ErrCodeQueryRejected,
		"操作被拒绝：只允许只读操作",
		"违规操作: "+operation).WriteError(w, http.StatusForbidden)
}

// RespondNotFound 返回404未找到响应
func RespondNotFound(w http.ResponseWriter, message string) {
	NewErrorResponse(ErrCodeNotFound, message).WriteError(w, http.StatusNotFound)
}

// RespondDBNotFound 返回404数据库未找到响应
func RespondDBNotFound(w http.ResponseWriter, dbID string) {
	NewErrorResponseWithDetails(ErrCodeDBNotFound,
		"数据库配置未找到",
		"数据库ID: "+dbID).WriteError(w, http.StatusNotFound)
}

// RespondToolNotFound 返回404工具未找到响应
func RespondToolNotFound(w http.ResponseWriter, toolName string) {
	NewErrorResponseWithDetails(ErrCodeToolNotFound,
		"MCP工具未找到",
		"工具名: "+toolName).WriteError(w, http.StatusNotFound)
}

// RespondInternalError 返回500内部错误响应
func RespondInternalError(w http.ResponseWriter, message string) {
	NewErrorResponse(ErrCodeInternalError, message).WriteError(w, http.StatusInternalServerError)
}

// RespondTimeout 返回504超时响应
func RespondTimeout(w http.ResponseWriter, message string) {
	NewErrorResponse(ErrCodeQueryTimeout, message).WriteError(w, http.StatusGatewayTimeout)
}

// RespondQueryTimeout 返回504查询超时响应
func RespondQueryTimeout(w http.ResponseWriter, timeout int) {
	NewErrorResponseWithDetails(ErrCodeQueryTimeout,
		"查询执行超时",
		"超时设置: "+string(timeout)+"秒").WriteError(w, http.StatusGatewayTimeout)
}

// RespondConnectionError 返回连接错误响应
func RespondConnectionError(w http.ResponseWriter, message string) {
	NewErrorResponse(ErrCodeConnectionError, message).WriteError(w, http.StatusServiceUnavailable)
}

// RespondDBConnectionError 返回数据库连接错误响应
func RespondDBConnectionError(w http.ResponseWriter, dbID, errDetail string) {
	NewErrorResponseWithDetails(ErrCodeConnectionError,
		"数据库连接失败",
		"数据库ID: "+dbID+", 错误: "+errDetail).WriteError(w, http.StatusServiceUnavailable)
}

// RespondPoolExhausted 返回连接池耗尽响应
func RespondPoolExhausted(w http.ResponseWriter, dbID string) {
	NewErrorResponseWithDetails(ErrCodePoolExhausted,
		"连接池已耗尽，无法获取连接",
		"数据库ID: "+dbID).WriteError(w, http.StatusServiceUnavailable)
}

// RespondQueryError 返回查询执行错误响应
func RespondQueryError(w http.ResponseWriter, errDetail string) {
	NewErrorResponseWithDetails(ErrCodeQueryError,
		"查询执行失败",
		errDetail).WriteError(w, http.StatusInternalServerError)
}

// RespondJSONError 返回JSON解析错误响应
func RespondJSONError(w http.ResponseWriter, errDetail string) {
	NewErrorResponseWithDetails(ErrCodeJSONError,
		"JSON解析失败",
		errDetail).WriteError(w, http.StatusBadRequest)
}

// RespondSuccessWithData 返回成功响应（带数据）
func RespondSuccessWithData(w http.ResponseWriter, data interface{}) {
	NewSuccessResponse(data).WriteSuccess(w)
}

// RespondSuccessWithMetadata 返回成功响应（带元数据）
func RespondSuccessWithMetadata(w http.ResponseWriter, data interface{}, metadata map[string]interface{}) {
	NewSuccessResponse(data).WithMetadata(metadata).WriteSuccess(w)
}

// RespondQueryResult 返回查询结果响应
func RespondQueryResult(w http.ResponseWriter, result *QueryResultResponse) {
	RespondSuccessWithMetadata(w, result, map[string]interface{}{
		"execution_ms": result.ExecutionMs,
		"truncated":    result.Truncated,
	})
}
