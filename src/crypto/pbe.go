package crypto

import (
	"crypto/cipher"
	"crypto/des"
	"crypto/md5"
	"crypto/rand"
	"io"
)

// pbe.go - PBEWithMD5AndDES (PKCS#5 v1.5) 实现
// 兼容 Java PBEWithMD5AndDES 算法

// deriveKey 从密码和盐值派生 DES 密钥和 IV
// 使用 MD5 进行密钥派生，符合 PKCS#5 v1.5 规范
func deriveKey(password string, salt []byte) (key []byte, iv []byte) {
	// 密钥派生步骤:
	// 1. 将密码和盐值拼接
	// 2. 计算 MD5 哈希
	// 3. 前8字节作为 DES 密钥
	// 4. 后8字节作为 IV

	// 拼接密码和盐值
	data := append([]byte(password), salt...)

	// 计算 MD5 哈希（16字节）
	hash := md5.Sum(data)

	// DES 密钥：前8字节
	key = hash[:8]

	// IV：后8字节
	iv = hash[8:16]

	return key, iv
}

// pkcs5Pad PKCS#5/PKCS#7 填充
func pkcs5Pad(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	padded := make([]byte, len(data)+padding)
	copy(padded, data)
	for i := len(data); i < len(padded); i++ {
		padded[i] = byte(padding)
	}
	return padded
}

// pkcs5Unpad PKCS#5/PKCS#7 去填充
func pkcs5Unpad(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, ErrInvalidPadding
	}

	padding := int(data[len(data)-1])
	if padding > des.BlockSize || padding > len(data) {
		return nil, ErrInvalidPadding
	}

	// 验证填充字节
	for i := len(data) - padding; i < len(data); i++ {
		if data[i] != byte(padding) {
			return nil, ErrInvalidPadding
		}
	}

	return data[:len(data)-padding], nil
}

// generateSalt 生成随机盐值
func generateSalt(size int) ([]byte, error) {
	salt := make([]byte, size)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, err
	}
	return salt, nil
}

// Encrypt 使用 PBEWithMD5AndDES 加密数据
// 返回 salt + ciphertext 的组合
func Encrypt(password string, plaintext []byte) ([]byte, error) {
	if len(plaintext) == 0 {
		return nil, ErrEmptyPlaintext
	}

	if password == "" {
		return nil, ErrEmptyKey
	}

	// 生成随机盐值（8字节）
	salt, err := generateSalt(SaltSize)
	if err != nil {
		return nil, err
	}

	// 派生密钥和 IV
	key, iv := deriveKey(password, salt)

	// 创建 DES cipher
	block, err := des.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// PKCS#5 填充
	padded := pkcs5Pad(plaintext, des.BlockSize)

	// CBC 模式加密
	ciphertext := make([]byte, len(padded))
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext, padded)

	// 返回 salt + ciphertext
	result := append(salt, ciphertext...)
	return result, nil
}

// Decrypt 使用 PBEWithMD5AndDES 解密数据
// 输入应为 salt + ciphertext 的组合
func Decrypt(password string, data []byte) ([]byte, error) {
	if len(data) < SaltSize+des.BlockSize {
		return nil, ErrInvalidCiphertext
	}

	if password == "" {
		return nil, ErrEmptyKey
	}

	// 分离盐值和密文
	salt := data[:SaltSize]
	ciphertext := data[SaltSize:]

	// 验证密文长度（必须是8字节的倍数）
	if len(ciphertext)%des.BlockSize != 0 {
		return nil, ErrInvalidCiphertext
	}

	// 派生密钥和 IV
	key, iv := deriveKey(password, salt)

	// 创建 DES cipher
	block, err := des.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// CBC 模式解密
	padded := make([]byte, len(ciphertext))
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(padded, ciphertext)

	// PKCS#5 去填充
	plaintext, err := pkcs5Unpad(padded)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}