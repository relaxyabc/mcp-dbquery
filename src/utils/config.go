package utils

import (
	"fmt"
	"os"
	"strconv"
)

// ConfigHelper 配置辅助工具
// 提供环境变量解析和配置验证功能

// GetEnv 获取环境变量，如果不存在则返回默认值
func GetEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// GetEnvRequired 获取必需的环境变量，如果不存在则返回错误
func GetEnvRequired(key string) (string, error) {
	value := os.Getenv(key)
	if value == "" {
		return "", fmt.Errorf("环境变量 %s 必须设置", key)
	}
	return value, nil
}

// GetEnvInt 获取整数类型的环境变量
func GetEnvInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	intValue, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return intValue
}

// GetEnvBool 获取布尔类型的环境变量
func GetEnvBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value == "true" || value == "1" || value == "yes"
}

// ExpandEnvVars 扩展字符串中的环境变量引用 ${VAR_NAME}
func ExpandEnvVars(s string) string {
	return os.Expand(s, func(key string) string {
		return GetEnv(key, "")
	})
}

// ValidateAPIKey 验证API密钥长度（最小32字符）
func ValidateAPIKey(apiKey string) error {
	if apiKey == "" {
		return fmt.Errorf("API密钥不能为空")
	}
	if len(apiKey) < 32 {
		return fmt.Errorf("API密钥长度必须至少32字符（当前：%d）", len(apiKey))
	}
	return nil
}

// ValidatePort 验证端口范围
func ValidatePort(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("端口必须在1-65535范围内（当前：%d）", port)
	}
	if port < 1024 {
		// 警告：使用系统端口
		GlobalLogger.Warn("使用系统端口（<1024）可能需要管理员权限：%d", port)
	}
	return nil
}

// ValidatePoolSize 验证连接池大小
func ValidatePoolSize(size int) error {
	if size < 1 || size > 100 {
		return fmt.Errorf("连接池大小必须在1-100范围内（当前：%d）", size)
	}
	// 推荐5-20
	if size > 20 {
		GlobalLogger.Warn("连接池大小超过推荐值（>20）：%d", size)
	}
	return nil
}

// ValidateTimeout 验证超时范围
func ValidateTimeout(timeout int) error {
	if timeout < 1 || timeout > 600 {
		return fmt.Errorf("超时必须在1-600秒范围内（当前：%d）", timeout)
	}
	return nil
}

// GenerateAPIKey 生成随机API密钥（用于测试或示例）
func GenerateAPIKey() string {
	// 使用时间戳和随机数生成（实际应用应使用crypto/rand）
	// 注意：这只是示例，生产环境应使用更安全的方法
	return fmt.Sprintf("db-query-api-key-%d-example-32chars-minimum", os.Getpid())
}
