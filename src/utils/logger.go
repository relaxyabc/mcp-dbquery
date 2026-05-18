package utils

import (
	"fmt"
	"log"
	"os"
	"strings"
)

// LogLevel 表示日志级别
type LogLevel string

const (
	LogLevelDebug LogLevel = "debug" // 调试级别
	LogLevelInfo  LogLevel = "info"  // 信息级别
	LogLevelWarn  LogLevel = "warn"  // 警告级别
	LogLevelError LogLevel = "error" // 错误级别
)

// MaskedLogger 提供密码遮蔽功能的日志器
// 宪章原则 II 要求：所有日志输出必须遮蔽密码
type MaskedLogger struct {
	level          LogLevel
	maskPasswords  bool        // 是否遮蔽密码（必须为true）
	sensitiveWords []string    // 需要遮蔽的敏感词汇
	logger         *log.Logger // 标准日志器
}

// NewMaskedLogger 创建新的遮蔽日志器
func NewMaskedLogger(level LogLevel) *MaskedLogger {
	return &MaskedLogger{
		level:          level,
		maskPasswords:  true, // 强制开启，符合宪章要求
		sensitiveWords: []string{"password", "passwd", "pwd", "secret", "key", "token"},
		logger:         log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile),
	}
}

// maskString 遮蔽字符串中的敏感信息
func (l *MaskedLogger) maskString(s string) string {
	// 遮蔽常见密码格式
	result := s

	// 遮蔽模式：password=xxx, pwd=xxx
	for _, word := range l.sensitiveWords {
		// 遮蔽 "password=value" 格式
		patterns := []string{
			word + "=",
			word + ":",
		}
		for _, pattern := range patterns {
			if strings.Contains(result, pattern) {
				// 找到模式后遮蔽后面的值
				idx := strings.Index(result, pattern)
				start := idx + len(pattern)
				// 找到值的结束位置（空格、逗号或结尾）
				end := len(result)
				for i := start; i < len(result); i++ {
					if result[i] == ' ' || result[i] == ',' || result[i] == '&' {
						end = i
						break
					}
				}
				if start < end {
					masked := result[:start] + "[REDACTED]" + result[end:]
					result = masked
				}
			}
		}
	}

	// 遮蔽连接字符串中的密码：user:password@host
	if strings.Contains(result, ":") && strings.Contains(result, "@") {
		// MongoDB格式：mongodb://user:password@host
		if strings.Contains(result, "mongodb://") || strings.Contains(result, "@tcp(") {
			// 在 : 和 @ 之间的内容可能是密码
			atIdx := strings.Index(result, "@")
			colonIdx := strings.Index(result, ":")
			if colonIdx >= 0 && atIdx > colonIdx {
				// 遮蔽冒号到@之间的内容
				pre := result[:colonIdx+1]
				post := result[atIdx:]
				result = pre + "[REDACTED]" + post
			}
		}
	}

	return result
}

// Debug 输出调试级别日志
func (l *MaskedLogger) Debug(format string, args ...interface{}) {
	if l.level == LogLevelDebug {
		msg := fmt.Sprintf(format, args...)
		l.logger.Printf("[DEBUG] %s", l.maskString(msg))
	}
}

// Info 输出信息级别日志
func (l *MaskedLogger) Info(format string, args ...interface{}) {
	if l.level == LogLevelDebug || l.level == LogLevelInfo {
		msg := fmt.Sprintf(format, args...)
		l.logger.Printf("[INFO] %s", l.maskString(msg))
	}
}

// Warn 输出警告级别日志
func (l *MaskedLogger) Warn(format string, args ...interface{}) {
	if l.level != LogLevelError {
		msg := fmt.Sprintf(format, args...)
		l.logger.Printf("[WARN] %s", l.maskString(msg))
	}
}

// Error 输出错误级别日志
func (l *MaskedLogger) Error(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	l.logger.Printf("[ERROR] %s", l.maskString(msg))
}

// LogQuery 记录查询日志（带请求追踪）
func (l *MaskedLogger) LogQuery(requestID, databaseID, query string) {
	l.Info("查询请求 [请求ID=%s] [数据库=%s] [查询=%s]", requestID, databaseID, query)
}

// LogQueryResult 记录查询结果日志
func (l *MaskedLogger) LogQueryResult(requestID string, success bool, rowCount int, executionTime int) {
	if success {
		l.Info("查询成功 [请求ID=%s] [行数=%d] [耗时=%dms]", requestID, rowCount, executionTime)
	} else {
		l.Warn("查询失败 [请求ID=%s] [耗时=%dms]", requestID, executionTime)
	}
}

// LogConnection 记录连接日志（使用遮蔽的连接字符串）
func (l *MaskedLogger) LogConnection(databaseID, maskedConnStr string, state string) {
	l.Info("数据库连接 [ID=%s] [连接=%s] [状态=%s]", databaseID, maskedConnStr, state)
}

// LogError 记录错误日志（带错误代码）
func (l *MaskedLogger) LogError(code, message string, details string) {
	if details != "" {
		l.Error("[%s] %s - %s", code, message, details)
	} else {
		l.Error("[%s] %s", code, message)
	}
}

// GlobalLogger 全局日志器实例
var GlobalLogger = NewMaskedLogger(LogLevelInfo)

// SetLogLevel 设置全局日志级别
func SetLogLevel(level LogLevel) {
	GlobalLogger = NewMaskedLogger(level)
}

// SetLogOutput 设置日志输出目标（用于STDIO模式将日志写到stderr）
func SetLogOutput(w *os.File) {
	GlobalLogger.logger = log.New(w, "", log.LstdFlags|log.Lshortfile)
}
