package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"io"
)

// 固定密钥：由常量字符串 SHA256 得到 32 字节，用于 AES-256-GCM
var aesKey []byte

func init() {
	h := sha256.Sum256([]byte("JTwZwHaMq3g55PfVgxkah1Edst59LoyF"))
	aesKey = h[:]
}

// Encrypt / Decrypt 供其它本地敏感数据（如知识库）复用与配置文件相同的加密方式
func Encrypt(plaintext []byte) ([]byte, error)  { return encrypt(plaintext) }
func Decrypt(ciphertext []byte) ([]byte, error) { return decrypt(ciphertext) }

func encrypt(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func decrypt(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("密文过短")
	}
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}
