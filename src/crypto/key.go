package crypto

import (
	"fmt"
	"os"
	"strings"
)

// key.go - 加密密钥获取

// GetEncryptionKey 从环境变量获取加密密钥
func GetEncryptionKey() (string, error) {
	key := os.Getenv(EncryptionKeyEnv)
	if key == "" {
		return "", ErrEmptyKey
	}
	return key, nil
}

// GetEncryptionKeyFromFile 从文件获取加密密钥
func GetEncryptionKeyFromFile(path string) (string, error) {
	if path == "" {
		return "", ErrEmptyKey
	}

	// 检查文件是否存在
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return "", NewKeyFileError(path)
	}

	// 读取文件内容
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("读取密钥文件失败: %s", err)
	}

	// 去除可能的换行符和空格
	key := string(data)
	key = trimKey(key)

	if key == "" {
		return "", ErrEmptyKey
	}

	return key, nil
}

// GetEncryptionKeyWithFallback 先尝试环境变量，再尝试文件
func GetEncryptionKeyWithFallback(keyFile string) (string, string, error) {
	// 优先使用环境变量
	key, err := GetEncryptionKey()
	if err == nil {
		return key, "env", nil
	}

	// 环境变量不存在，尝试文件
	if keyFile != "" {
		key, err = GetEncryptionKeyFromFile(keyFile)
		if err == nil {
			return key, "file", nil
		}
	}

	// 都失败，返回错误
	return "", "", ErrEmptyKey
}

// trimKey 清理密钥字符串
func trimKey(key string) string {
	// 去除前后空白和换行
	key = strings.TrimSpace(key)
	return key
}