package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
)

// SecretBox 用 AES-256-GCM 加密需要落库的敏感配置（例如 OIDC clientSecret）。
type SecretBox struct {
	aead cipher.AEAD
}

func New(key []byte) (*SecretBox, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("密钥长度必须为 32 字节，当前 %d", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &SecretBox{aead: aead}, nil
}

func (b *SecretBox) Seal(plain []byte) ([]byte, error) {
	nonce := make([]byte, b.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return b.aead.Seal(nonce, nonce, plain, nil), nil
}

func (b *SecretBox) Open(cipherText []byte) ([]byte, error) {
	ns := b.aead.NonceSize()
	if len(cipherText) < ns {
		return nil, fmt.Errorf("密文长度不足")
	}
	return b.aead.Open(nil, cipherText[:ns], cipherText[ns:], nil)
}
