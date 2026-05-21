package crypto

import (
	"testing"
)

// format_test.go - 密文格式解析和构建单元测试

func TestParseEncryptedFormat(t *testing.T) {
	// 先加密一个密码
	key := "testKey12345678"
 plaintext := "testPassword"
 ciphertext, _ := Encrypt(key, []byte(plaintext))

	// 构建格式字符串
	formatted := BuildEncryptedFormat(ciphertext)

	// 解析格式字符串
	parsed, err := ParseEncryptedFormat(formatted)
	if err != nil {
		t.Fatalf("解析失败: %s", err)
	}

	// 验证解析结果
	if parsed.Prefix != FormatPrefix {
		t.Fatalf("前缀错误: got %s, want %s", parsed.Prefix, FormatPrefix)
	}
	if parsed.Version != FormatVersion {
		t.Fatalf("版本错误: got %s, want %s", parsed.Version, FormatVersion)
	}
	if len(parsed.Salt) != SaltSize {
		t.Fatalf("盐值长度错误: got %d, want %d", len(parsed.Salt), SaltSize)
	}
	if len(parsed.Ciphertext) == 0 {
		t.Fatal("密文为空")
	}
}

func TestParseInvalidFormat(t *testing.T) {
	tests := []string{
		"",                     // 空
		"plaintext",            // 无前缀
		"enc:v2:something",     // 错误版本
		"enc:v1:",              // 空 Base64
		"enc:v1:invalid!base64", // 无效 Base64
	}

	for _, test := range tests {
		_, err := ParseEncryptedFormat(test)
		if err == nil {
			t.Fatalf("解析无效格式成功: %s", test)
		}
	}
}

func TestBuildEncryptedFormat(t *testing.T) {
	key := "testKey12345678"
 plaintext := "testPassword"
 ciphertext, _ := Encrypt(key, []byte(plaintext))

	// 构建格式字符串
	formatted := BuildEncryptedFormat(ciphertext)

	// 验证格式字符串
	if !IsEncryptedPassword(formatted) {
		t.Fatalf("格式字符串未被识别为加密密码: %s", formatted)
	}
}

func TestIsEncryptedPassword(t *testing.T) {
	tests := []struct {
		password string
		expected bool
	}{
		{"enc:v1:something", true},
		{"plaintext", false},
		{"", false},
		{"ENC:V1:something", false}, // 大写不匹配
		{"enc:v2:something", false}, // 版本不匹配
	}

	for _, test := range tests {
		result := IsEncryptedPassword(test.password)
		if result != test.expected {
			t.Fatalf("IsEncryptedPassword(%s) = %v, want %v", test.password, result, test.expected)
		}
	}
}

func TestGetPasswordType(t *testing.T) {
	tests := []struct {
		password string
		expected string
	}{
		{"enc:v1:something", "encrypted"},
		{"plaintext", "plaintext"},
		{"", "plaintext"},
	}

	for _, test := range tests {
		result := GetPasswordType(test.password)
		if result != test.expected {
			t.Fatalf("GetPasswordType(%s) = %s, want %s", test.password, result, test.expected)
		}
	}
}

func TestFormatRoundTrip(t *testing.T) {
	key := "testKey12345678"
 plaintext := "roundTripTest"

	// 加密
 ciphertext, _ := Encrypt(key, []byte(plaintext))

	// 构建格式
	formatted := BuildEncryptedFormat(ciphertext)

	// 解析格式
	parsed, _ := ParseEncryptedFormat(formatted)

	// 组合数据
	data := append(parsed.Salt, parsed.Ciphertext...)

	// 解密
	decrypted, err := Decrypt(key, data)
	if err != nil {
		t.Fatalf("解密失败: %s", err)
	}

	// 验证
	if string(decrypted) != plaintext {
		t.Fatalf("往返测试失败: got %s, want %s", decrypted, plaintext)
	}
}