package crypto

// 密码加密模块常量定义
// 使用 PBEWithMD5AndDES (PKCS#5 v1.5) 算法

// EncryptionKeyEnv 环境变量名称
const EncryptionKeyEnv = "DBQUERY_ENCRYPTION_KEY"

// 密文格式常量
const (
	// FormatPrefix 密文前缀标识
	FormatPrefix = "enc"
	// FormatVersion 当前算法版本
	FormatVersion = "v1"
	// FormatSeparator 格式分隔符
	FormatSeparator = ":"
	// SaltSize 盐值长度（8字节）
	SaltSize = 8
)

// 密文格式: enc:v1:<base64(salt+ciphertext)>