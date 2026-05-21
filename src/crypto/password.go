package crypto

// password.go - 密码加密/解密操作

// EncryptPassword 加密密码并返回密文格式字符串
func EncryptPassword(key string, password string) (string, error) {
	if password == "" {
		return "", ErrEmptyPlaintext
	}

	if key == "" {
		return "", ErrEmptyKey
	}

	// 加密
	data, err := Encrypt(key, []byte(password))
	if err != nil {
		return "", err
	}

	// 构建密文格式
	return BuildEncryptedFormat(data), nil
}

// DecryptPassword 解密密文格式字符串并返回明文密码
func DecryptPassword(key string, ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", ErrEmptyPlaintext
	}

	if key == "" {
		return "", ErrEmptyKey
	}

	// 解析密文格式
	format, err := ParseEncryptedFormat(ciphertext)
	if err != nil {
		return "", err
	}

	// 组合 salt + ciphertext 进行解密
	data := append(format.Salt, format.Ciphertext...)

	// 解密
 plaintext, err := Decrypt(key, data)
	if err != nil {
		return "", ErrDecryptionFailed
	}

	return string(plaintext), nil
}

// ProcessPassword 处理密码（检测类型并自动解密）
// 如果是加密格式，自动解密；如果是明文，直接返回
func ProcessPassword(key string, password string) (string, string, error) {
	if password == "" {
		return "", "empty", nil
	}

	// 检测密码类型
	passwordType := GetPasswordType(password)

	if passwordType == "plaintext" {
		// 明文密码，直接返回
		return password, "plaintext", nil
	}

	// 加密密码，需要解密
	if key == "" {
		return "", "encrypted", ErrEmptyKey
	}

	plaintext, err := DecryptPassword(key, password)
	if err != nil {
		return "", "encrypted", err
	}

	return plaintext, "encrypted", nil
}

// MaskPassword 返回密码的掩码形式
func MaskPassword(password string) string {
	if IsEncryptedPassword(password) {
		return "[ENCRYPTED]"
	}
	if password == "" {
		return "[EMPTY]"
	}
	return "[REDACTED]"
}