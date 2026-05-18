package server

import (
	"crypto/sha256"
	"encoding/hex"
	"time"
)

// APIKey 表示认证API密钥
type APIKey struct {
	Key       string    // API密钥字符串（32+字符）
	Name      string    // 密钥标识符（用于日志/审计）
	CreatedAt time.Time // 创建时间
	ExpiresAt time.Time // 过期时间（可选）
	Active    bool      // 是否激活
}

// NewAPIKey 创建新的API密钥
func NewAPIKey(key, name string) *APIKey {
	return &APIKey{
		Key:       key,
		Name:      name,
		CreatedAt: time.Now(),
		Active:    true,
	}
}

// WithExpiration 设置过期时间
func (k *APIKey) WithExpiration(duration time.Duration) *APIKey {
	k.ExpiresAt = time.Now().Add(duration)
	return k
}

// Validate 验证API密钥
func (k *APIKey) Validate() bool {
	// 检查密钥是否激活
	if !k.Active {
		return false
	}

	// 检查是否过期
	if !k.ExpiresAt.IsZero() && time.Now().After(k.ExpiresAt) {
		return false
	}

	// 检查长度
	if len(k.Key) < 32 {
		return false
	}

	return true
}

// Matches 检查提供的密钥是否匹配
func (k *APIKey) Matches(providedKey string) bool {
	// 直接比较
	if k.Key == providedKey {
		return true
	}

	// 使用SHA256哈希比较（如果密钥以hash:开头）
	if len(k.Key) > 5 && k.Key[:5] == "hash:" {
		hash := sha256.Sum256([]byte(providedKey))
		hashedKey := "hash:" + hex.EncodeToString(hash[:])
		if k.Key == hashedKey {
			return true
		}
	}

	return false
}

// Deactivate 停用API密钥
func (k *APIKey) Deactivate() {
	k.Active = false
}

// Activate 激活API密钥
func (k *APIKey) Activate() {
	k.Active = true
}

// AuthManager API密钥认证管理器
type AuthManager struct {
	keys     map[string]*APIKey // 密钥存储（按密钥字符串索引）
	keyNames map[string]string  // 密钥名称映射
}

// NewAuthManager 创建认证管理器
func NewAuthManager() *AuthManager {
	return &AuthManager{
		keys:     make(map[string]*APIKey),
		keyNames: make(map[string]string),
	}
}

// AddKey 添加API密钥
func (am *AuthManager) AddKey(apiKey *APIKey) {
	am.keys[apiKey.Key] = apiKey
	am.keyNames[apiKey.Name] = apiKey.Key
}

// AddKeyFromString 从字符串添加API密钥
func (am *AuthManager) AddKeyFromString(key, name string) {
	apiKey := NewAPIKey(key, name)
	am.AddKey(apiKey)
}

// Validate 验证提供的API密钥
func (am *AuthManager) Validate(providedKey string) bool {
	apiKey, exists := am.keys[providedKey]
	if !exists {
		return false
	}
	return apiKey.Validate()
}

// ValidateByHash 通过哈希验证API密钥
func (am *AuthManager) ValidateByHash(providedKey string) bool {
	// 首先尝试直接匹配
	if am.Validate(providedKey) {
		return true
	}

	// 计算哈希并匹配
	hash := sha256.Sum256([]byte(providedKey))
	hashedKey := "hash:" + hex.EncodeToString(hash[:])

	return am.Validate(hashedKey)
}

// GetKeyByName 通过名称获取API密钥
func (am *AuthManager) GetKeyByName(name string) *APIKey {
	keyString, exists := am.keyNames[name]
	if !exists {
		return nil
	}
	return am.keys[keyString]
}

// RemoveKey 删除API密钥
func (am *AuthManager) RemoveKey(key string) {
	if apiKey, exists := am.keys[key]; exists {
		delete(am.keyNames, apiKey.Name)
		delete(am.keys, key)
	}
}

// Count 返回API密钥数量
func (am *AuthManager) Count() int {
	return len(am.keys)
}

// ListActive 返回所有活跃的API密钥名称
func (am *AuthManager) ListActive() []string {
	active := []string{}
	for _, apiKey := range am.keys {
		if apiKey.Validate() {
			active = append(active, apiKey.Name)
		}
	}
	return active
}
