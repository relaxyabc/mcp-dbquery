package crypto

import (
	"encoding/base64"
	"strings"
)

// format.go - 密文格式解析和构建
// 格式: enc:v1:<base64(salt+ciphertext)>

// EncryptedFormat 表示加密后的密码格式
type EncryptedFormat struct {
	Prefix     string // "enc"
	Version    string // "v1"
	Salt       []byte // 8字节盐值
	Ciphertext []byte // DES加密后的密文
	Raw        string // 原始字符串
}

// ParseEncryptedFormat 解析密文格式字符串
// 输入格式: enc:v1:<base64>
func ParseEncryptedFormat(s string) (*EncryptedFormat, error) {
	if s == "" {
		return nil, ErrEmptyPlaintext
	}

	// 检查格式前缀
	if !strings.HasPrefix(s, FormatPrefix+FormatSeparator+FormatVersion+FormatSeparator) {
		return nil, ErrInvalidFormat
	}

	// 分离各部分
	parts := strings.SplitN(s, FormatSeparator, 4)
	if len(parts) < 3 {
		return nil, ErrInvalidFormat
	}

	prefix := parts[0]
	version := parts[1]
	base64Data := parts[2]

	// 验证前缀和版本
	if prefix != FormatPrefix {
		return nil, ErrInvalidFormat
	}
	if version != FormatVersion {
		return nil, ErrInvalidFormat
	}

	// Base64 解码
	data, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return nil, ErrInvalidFormat
	}

	// 验证数据长度（至少 salt + 1个block）
	if len(data) < SaltSize {
		return nil, ErrInvalidCiphertext
	}

	// 分离盐值和密文
	salt := data[:SaltSize]
	ciphertext := data[SaltSize:]

	return &EncryptedFormat{
		Prefix:     prefix,
		Version:    version,
		Salt:       salt,
		Ciphertext: ciphertext,
		Raw:        s,
	}, nil
}

// BuildEncryptedFormat 构建密文格式字符串
// 输入: salt + ciphertext 组合数据
func BuildEncryptedFormat(data []byte) string {
	// Base64 编码
	base64Data := base64.StdEncoding.EncodeToString(data)

	// 构建格式字符串
	return FormatPrefix + FormatSeparator + FormatVersion + FormatSeparator + base64Data
}

// IsEncryptedPassword 检测密码是否为加密格式
func IsEncryptedPassword(password string) bool {
	if password == "" {
		return false
	}

	// 检查格式前缀
	expectedPrefix := FormatPrefix + FormatSeparator + FormatVersion + FormatSeparator
	return strings.HasPrefix(password, expectedPrefix)
}

// GetPasswordType 获取密码类型
// 返回 "encrypted" 或 "plaintext"
func GetPasswordType(password string) string {
	if IsEncryptedPassword(password) {
		return "encrypted"
	}
	return "plaintext"
}