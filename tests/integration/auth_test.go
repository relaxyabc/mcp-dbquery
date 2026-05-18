package integration

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"github.com/relaxyabc/mcp-dbquery/src/server"
	"github.com/relaxyabc/mcp-dbquery/src/utils"
)

// TestAPIKeyValidation 测试API密钥验证
// 宪章原则 II 要求：API密钥认证强制执行
func TestAPIKeyValidation(t *testing.T) {
	utils.GlobalLogger.Info("开始API密钥验证测试")

	authManager := server.NewAuthManager()

	// 测试有效密钥
	validKey := "test-api-key-32-characters-minimum-length"
	authManager.AddKeyFromString(validKey, "test-key")

	if !authManager.Validate(validKey) {
		t.Error("有效API密钥验证失败")
	}

	// 测试无效密钥
	invalidKey := "invalid-key"
	if authManager.Validate(invalidKey) {
		t.Error("无效API密钥未被拒绝")
	}

	// 测试空密钥
	if authManager.Validate("") {
		t.Error("空API密钥未被拒绝")
	}

	// 测试短密钥（少于32字符）
	shortKey := "short-key-less-than-32"
	apiKey := server.NewAPIKey(shortKey, "short-key")
	if apiKey.Validate() {
		t.Error("短密钥（<32字符）不应通过验证")
	}

	utils.GlobalLogger.Info("API密钥验证测试通过")
}

// TestAPIKeyExpiration 测试API密钥过期机制
func TestAPIKeyExpiration(t *testing.T) {
	authManager := server.NewAuthManager()

	// 创建带过期时间的密钥
	expiredKey := server.NewAPIKey("expired-key-32-characters-minimum", "expired")
	expiredKey.WithExpiration(-1 * time.Hour) // 已过期

	authManager.AddKey(expiredKey)

	// 过期密钥不应通过验证
	if authManager.Validate(expiredKey.Key) {
		t.Error("过期API密钥未被拒绝")
	}

	// 创建未过期密钥
	validKey := server.NewAPIKey("valid-key-32-characters-minimum", "valid")
	validKey.WithExpiration(1 * time.Hour) // 1小时后过期

	authManager.AddKey(validKey)

	// 未过期密钥应通过验证
	if !authManager.Validate(validKey.Key) {
		t.Error("未过期API密钥验证失败")
	}

	utils.GlobalLogger.Info("API密钥过期机制测试通过")
}

// TestAPIKeyActivation 测试API密钥激活/停用
func TestAPIKeyActivation(t *testing.T) {
	authManager := server.NewAuthManager()

	key := server.NewAPIKey("test-key-32-characters-minimum-length", "test")
	authManager.AddKey(key)

	// 初始状态应通过验证
	if !authManager.Validate(key.Key) {
		t.Error("初始密钥验证失败")
	}

	// 停用密钥
	key.Deactivate()

	// 停用后不应通过验证
	if authManager.Validate(key.Key) {
		t.Error("停用密钥未被拒绝")
	}

	// 重新激活
	key.Activate()

	// 激活后应通过验证
	if !authManager.Validate(key.Key) {
		t.Error("重新激活密钥验证失败")
	}

	utils.GlobalLogger.Info("API密钥激活/停用测试通过")
}

// TestAPIKeyHashValidation 测试API密钥哈希验证
func TestAPIKeyHashValidation(t *testing.T) {
	authManager := server.NewAuthManager()

	plainKey := "plain-key-32-characters-minimum-length"

	// 计算哈希
	hash := sha256.Sum256([]byte(plainKey))
	hashedKey := "hash:" + hex.EncodeToString(hash[:])

	// 存储哈希密钥
	apiKey := server.NewAPIKey(hashedKey, "hashed-key")
	authManager.AddKey(apiKey)

	// 通过哈希验证原始密钥
	if !authManager.ValidateByHash(plainKey) {
		t.Error("哈希密钥验证失败")
	}

	// 直接验证哈希密钥不应成功
	if authManager.Validate(plainKey) {
		t.Error("哈希密钥直接验证不应成功")
	}

	utils.GlobalLogger.Info("API密钥哈希验证测试通过")
}

// TestAPIKeyManagerOperations 测试认证管理器操作
func TestAPIKeyManagerOperations(t *testing.T) {
	authManager := server.NewAuthManager()

	// 添加多个密钥
	for i := 0; i < 5; i++ {
		keyName := fmt.Sprintf("key-%d", i)
		keyValue := fmt.Sprintf("key-value-%d-32-characters-minimum", i)
		authManager.AddKeyFromString(keyValue, keyName)
	}

	// 检查密钥数量
	if authManager.Count() != 5 {
		t.Errorf("密钥数量错误: 期望5, 实际%d", authManager.Count())
	}

	// 列出活跃密钥
	activeKeys := authManager.ListActive()
	if len(activeKeys) != 5 {
		t.Errorf("活跃密钥数量错误: 期望5, 实际%d", len(activeKeys))
	}

	// 删除密钥
	keyToDelete := "key-value-0-32-characters-minimum"
	authManager.RemoveKey(keyToDelete)

	if authManager.Count() != 4 {
		t.Errorf("删除后密钥数量错误: 期望4, 实际%d", authManager.Count())
	}

	// 通过名称获取密钥
	retrievedKey := authManager.GetKeyByName("key-1")
	if retrievedKey == nil {
		t.Error("通过名称获取密钥失败")
	}

	utils.GlobalLogger.Info("认证管理器操作测试通过")
}

// TestConstitutionSecurityRequirements 测试宪章安全要求
// 宪章原则 II (NON-NEGOTIABLE): 安全优先
func TestConstitutionSecurityRequirements(t *testing.T) {
	utils.GlobalLogger.Info("开始宪章安全要求测试")

	// 测试1：API密钥必须至少32字符
	shortKey := "short-16-char-key"
	apiKey := server.NewAPIKey(shortKey, "short")
	if apiKey.Validate() {
		t.Error("宪章违规：短密钥（<32字符）不应通过验证")
	}

	// 测试2：有效密钥长度验证
	validKey := "valid-32-character-minimum-api-key"
	apiKey = server.NewAPIKey(validKey, "valid")
	if !apiKey.Validate() {
		t.Error("有效长度密钥验证失败")
	}

	// 测试3：空密钥不应通过验证
	apiKey = server.NewAPIKey("", "empty")
	if apiKey.Validate() {
		t.Error("宪章违规：空密钥不应通过验证")
	}

	utils.GlobalLogger.Info("宪章安全要求测试通过")
}
