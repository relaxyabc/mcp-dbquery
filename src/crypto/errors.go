package crypto

import "fmt"

// errors.go - 加密模块错误定义

// 加密相关错误
var (
	// ErrEmptyPlaintext 明文为空
	ErrEmptyPlaintext = &CryptoError{Code: "E001", Message: "密码不能为空"}

	// ErrEmptyKey 加密密钥为空
	ErrEmptyKey = &CryptoError{Code: "E002", Message: "加密密钥未配置：请设置 DBQUERY_ENCRYPTION_KEY 环境变量"}

	// ErrInvalidPadding 无效的填充数据
	ErrInvalidPadding = &CryptoError{Code: "E003", Message: "解密失败：密钥不匹配或密文已损坏"}

	// ErrInvalidCiphertext 无效的密文数据
	ErrInvalidCiphertext = &CryptoError{Code: "E004", Message: "解密失败：密钥不匹配或密文已损坏"}

	// ErrInvalidFormat 密文格式错误
	ErrInvalidFormat = &CryptoError{Code: "E005", Message: "密文格式错误：应以 enc:v1: 开头"}

	// ErrKeyFileNotFound 密钥文件不存在
	ErrKeyFileNotFound = &CryptoError{Code: "E006", Message: "密钥文件不存在"}

	// ErrDecryptionFailed 解密失败
	ErrDecryptionFailed = &CryptoError{Code: "E007", Message: "解密失败：密钥不匹配或密文已损坏"}
)

// CryptoError 加密错误类型
type CryptoError struct {
	Code    string
	Message string
}

func (e *CryptoError) Error() string {
	return e.Message
}

// NewKeyFileError 创建密钥文件错误
func NewKeyFileError(path string) error {
	return fmt.Errorf("密钥文件不存在: %s", path)
}