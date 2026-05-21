package crypto

import (
	"testing"
)

// password_test.go - 密码加密解密单元测试

func TestEncryptPassword(t *testing.T) {
	key := "testKey12345678"
	password := "mySecretPassword"

	ciphertext, err := EncryptPassword(key, password)
	if err != nil {
		t.Fatalf("加密失败: %s", err)
	}

	// 验证密文格式
	if !IsEncryptedPassword(ciphertext) {
		t.Fatalf("密文格式不正确: %s", ciphertext)
	}
}

func TestEncryptPasswordEmpty(t *testing.T) {
	key := "testKey12345678"

	_, err := EncryptPassword(key, "")
	if err != ErrEmptyPlaintext {
		t.Fatalf("预期 ErrEmptyPlaintext, got: %s", err)
	}
}

func TestEncryptPasswordEmptyKey(t *testing.T) {
	password := "mySecretPassword"

	_, err := EncryptPassword("", password)
	if err != ErrEmptyKey {
		t.Fatalf("预期 ErrEmptyKey, got: %s", err)
	}
}

func TestDecryptPassword(t *testing.T) {
	key := "testKey12345678"
	password := "mySecretPassword"

	// 先加密
	ciphertext, _ := EncryptPassword(key, password)

	// 再解密
	plaintext, err := DecryptPassword(key, ciphertext)
	if err != nil {
		t.Fatalf("解密失败: %s", err)
	}

	// 验证解密结果
	if plaintext != password {
		t.Fatalf("解密结果不一致: got %s, want %s", plaintext, password)
	}
}

func TestDecryptPasswordInvalidFormat(t *testing.T) {
	key := "testKey12345678"

	_, err := DecryptPassword(key, "invalidFormat")
	if err != ErrInvalidFormat {
		t.Fatalf("预期 ErrInvalidFormat, got: %s", err)
	}
}

func TestProcessPasswordEncrypted(t *testing.T) {
	key := "testKey12345678"
	password := "mySecretPassword"

	// 加密密码
	ciphertext, _ := EncryptPassword(key, password)

	// 处理加密密码
	plaintext, pwType, err := ProcessPassword(key, ciphertext)
	if err != nil {
		t.Fatalf("处理失败: %s", err)
	}

	if pwType != "encrypted" {
		t.Fatalf("密码类型错误: got %s, want encrypted", pwType)
	}

	if plaintext != password {
		t.Fatalf("解密结果不一致: got %s, want %s", plaintext, password)
	}
}

func TestProcessPasswordPlaintext(t *testing.T) {
	key := "testKey12345678"
	password := "plaintextPassword"

	// 处理明文密码
	plaintext, pwType, err := ProcessPassword(key, password)
	if err != nil {
		t.Fatalf("处理失败: %s", err)
	}

	if pwType != "plaintext" {
		t.Fatalf("密码类型错误: got %s, want plaintext", pwType)
	}

	if plaintext != password {
		t.Fatalf("处理结果不一致: got %s, want %s", plaintext, password)
	}
}

func TestProcessPasswordEmptyKey(t *testing.T) {
	password := "plaintextPassword"

	// 先加密
	ciphertext, _ := EncryptPassword("someKey", password)

	// 使用空密钥处理加密密码
	_, pwType, err := ProcessPassword("", ciphertext)
	if err != ErrEmptyKey {
		t.Fatalf("预期 ErrEmptyKey, got: %s", err)
	}

	if pwType != "encrypted" {
		t.Fatalf("密码类型错误: got %s, want encrypted", pwType)
	}
}

func TestMaskPassword(t *testing.T) {
	tests := []struct {
		password string
		expected string
	}{
		{"enc:v1:something", "[ENCRYPTED]"},
		{"plaintext", "[REDACTED]"},
		{"", "[EMPTY]"},
	}

	for _, test := range tests {
		result := MaskPassword(test.password)
		if result != test.expected {
			t.Fatalf("MaskPassword(%s) = %s, want %s", test.password, result, test.expected)
		}
	}
}