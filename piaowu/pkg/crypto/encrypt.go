// Package crypto 提供加密解密公共能力
// 使用AES-256-GCM加密算法保护敏感消息内容
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"os"
	"sync"
)

// MessageEncryptor 消息加密器
// 使用AES-256-GCM算法进行加密解密
type MessageEncryptor struct {
	key []byte
	gcm cipher.AEAD
	mu  sync.RWMutex
}

var (
	defaultEncryptor *MessageEncryptor
	once             sync.Once
)

// GetEncryptor 获取单例加密器
// 从环境变量 MSG_ENCRYPT_KEY 读取密钥
func GetEncryptor() *MessageEncryptor {
	once.Do(func() {
		key := os.Getenv("MSG_ENCRYPT_KEY")
		if key == "" {
			key = "default-piaowu-encrypt-key-2026" // 生产环境必须替换
		}
		defaultEncryptor, _ = NewMessageEncryptor(key)
	})
	return defaultEncryptor
}

// NewMessageEncryptor 创建消息加密器
// 参数:
//   - secretKey: 密钥字符串，将被SHA256哈希为32字节密钥
func NewMessageEncryptor(secretKey string) (*MessageEncryptor, error) {
	hash := sha256.Sum256([]byte(secretKey))
	key := hash[:]

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return &MessageEncryptor{
		key: key,
		gcm: gcm,
	}, nil
}

// Encrypt 加密消息
// 返回Base64编码的密文
func (e *MessageEncryptor) Encrypt(plaintext string) (string, error) {
	if e == nil || e.gcm == nil {
		return "", errors.New("encryptor not initialized")
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	nonce := make([]byte, e.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := e.gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt 解密消息
// 参数为Base64编码的密文
func (e *MessageEncryptor) Decrypt(ciphertext string) (string, error) {
	if e == nil || e.gcm == nil {
		return "", errors.New("encryptor not initialized")
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	nonceSize := e.gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := e.gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// IsEncrypted 检查消息是否已加密
// 通过检查是否为有效的Base64且不含中文来判断
func IsEncrypted(text string) bool {
	if len(text) < 20 {
		return false
	}
	_, err := base64.StdEncoding.DecodeString(text)
	return err == nil && !containsChinese(text)
}

// containsChinese 检查字符串是否包含中文字符
func containsChinese(s string) bool {
	for _, r := range s {
		if r >= 0x4E00 && r <= 0x9FFF {
			return true
		}
	}
	return false
}

// EncryptIfNeeded 如果未加密则加密
func (e *MessageEncryptor) EncryptIfNeeded(text string) (string, error) {
	if IsEncrypted(text) {
		return text, nil
	}
	return e.Encrypt(text)
}

// DecryptIfNeeded 如果已加密则解密
func (e *MessageEncryptor) DecryptIfNeeded(text string) (string, error) {
	if !IsEncrypted(text) {
		return text, nil
	}
	return e.Decrypt(text)
}
