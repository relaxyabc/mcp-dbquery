package crypto_test

import (
	"testing"

	"github.com/relaxyabc/mcp-dbquery/src/crypto"
)

// pbe_test.go - PBE 加密解密单元测试

func TestEncryptDecrypt(t *testing.T) {
	key := "testKey12345678"
 plaintext := "mySecretPassword"

	// 加密
 ciphertext, err := crypto.Encrypt(key, []byte(plaintext))
	if err != nil {
		t.Fatalf("加密失败: %s", err)
	}

	// 验证密文不为空且不等于明文
	if len(ciphertext) == 0 {
		t.Fatal("密文为空")
	}
	if string(ciphertext) == plaintext {
		t.Fatal("密文与明文相同")
	}

	// 解密
 decrypted, err := crypto.Decrypt(key, ciphertext)
	if err != nil {
		t.Fatalf("解密失败: %s", err)
	}

	// 验证解密结果与明文一致
	if string(decrypted) != plaintext {
		t.Fatalf("解密结果不一致: got %s, want %s", decrypted, plaintext)
	}
}

func TestEncryptEmptyPlaintext(t *testing.T) {
	key := "testKey12345678"

	_, err := crypto.Encrypt(key, []byte(""))
	if err != crypto.ErrEmptyPlaintext {
		t.Fatalf("预期 ErrEmptyPlaintext, got: %s", err)
	}
}

func TestEncryptEmptyKey(t *testing.T) {
 plaintext := "mySecretPassword"

	_, err := crypto.Encrypt("", []byte(plaintext))
	if err != crypto.ErrEmptyKey {
		t.Fatalf("预期 ErrEmptyKey, got: %s", err)
	}
}

func TestDecryptEmptyKey(t *testing.T) {
	// 先加密一个密码
	key := "testKey12345678"
 plaintext := "mySecretPassword"
 ciphertext, _ := crypto.Encrypt(key, []byte(plaintext))

	// 使用空密钥解密
	_, err := crypto.Decrypt("", ciphertext)
	if err != crypto.ErrEmptyKey {
		t.Fatalf("预期 ErrEmptyKey, got: %s", err)
	}
}

func TestDecryptInvalidCiphertext(t *testing.T) {
	key := "testKey12345678"

	// 密文太短
	_, err := crypto.Decrypt(key, []byte("short"))
	if err != crypto.ErrInvalidCiphertext {
		t.Fatalf("预期 ErrInvalidCiphertext, got: %s", err)
	}

	// 密文长度不是8的倍数
	_, err = crypto.Decrypt(key, []byte("12345678abcdefghij")) // 18字节，不是8的倍数
	if err != crypto.ErrInvalidCiphertext {
		t.Fatalf("预期 ErrInvalidCiphertext, got: %s", err)
	}
}

func TestDecryptWrongKey(t *testing.T) {
	key1 := "testKey12345678"
	key2 := "wrongKey1234567"
 plaintext := "mySecretPassword"

	// 用 key1 加密
 ciphertext, _ := crypto.Encrypt(key1, []byte(plaintext))

	// 用 key2 解密（应该失败或结果不一致）
 decrypted, err := crypto.Decrypt(key2, ciphertext)
	if err == nil {
		// 解密成功但结果不一致
		if string(decrypted) == plaintext {
			t.Fatal("使用错误密钥解密成功且结果一致（不应该发生）")
		}
	}
	// 解密失败或结果不一致都是预期行为
}

func TestDifferentPasswordsDifferentCiphertext(t *testing.T) {
	key := "testKey12345678"
 passwords := []string{"password1", "password2", "password3"}

 ciphertexts := make([][]byte, len(passwords))
	for i, pwd := range passwords {
		ct, err := crypto.Encrypt(key, []byte(pwd))
		if err != nil {
			t.Fatalf("加密失败: %s", err)
		}
	 ciphertexts[i] = ct
	}

	// 验证不同密码产生不同密文
	for i := 0; i < len(ciphertexts)-1; i++ {
		for j := i + 1; j < len(ciphertexts); j++ {
			if string(ciphertexts[i]) == string(ciphertexts[j]) {
				t.Fatalf("密码 %d 和 %d 产生相同密文", i, j)
			}
		}
	}
}

func TestSamePasswordDifferentCiphertext(t *testing.T) {
	key := "testKey12345678"
 plaintext := "samePassword"

	// 加密两次
 ct1, _ := crypto.Encrypt(key, []byte(plaintext))
	ct2, _ := crypto.Encrypt(key, []byte(plaintext))

	// 由于随机盐值，两次加密结果应该不同
	if string(ct1) == string(ct2) {
		t.Fatal("相同密码两次加密产生相同密文（盐值可能未随机化）")
	}

	// 但两次解密结果都应该一致
	dec1, _ := crypto.Decrypt(key, ct1)
	dec2, _ := crypto.Decrypt(key, ct2)

	if string(dec1) != plaintext || string(dec2) != plaintext {
		t.Fatal("解密结果不一致")
	}
}