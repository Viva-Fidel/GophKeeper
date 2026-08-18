// Package crypto реализует клиентское шифрование AES-256-GCM
// с ключом, производным от мастер-пароля через PBKDF2.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/pbkdf2"
)

const (
	// KeySize — длина ключа AES-256 в байтах.
	KeySize = 32
	// SaltSize — длина соли PBKDF2 в байтах.
	SaltSize = 16
	// NonceSize — длина nonce для GCM.
	NonceSize = 12
	// DefaultIterations — число итераций PBKDF2.
	DefaultIterations = 100_000
)

// DeriveKey выводит ключ шифрования из пароля и соли.
func DeriveKey(password string, salt []byte, iterations int) []byte {
	if iterations <= 0 {
		iterations = DefaultIterations
	}
	return pbkdf2.Key([]byte(password), salt, iterations, KeySize, sha256.New)
}

// NewSalt генерирует криптографически стойкую соль.
func NewSalt() ([]byte, error) {
	salt := make([]byte, SaltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("generate salt: %w", err)
	}
	return salt, nil
}

// Encrypt шифрует plaintext ключом AES-GCM.
// Результат: nonce || ciphertext.
func Encrypt(key, plaintext []byte) ([]byte, error) {
	if len(key) != KeySize {
		return nil, errors.New("invalid key size")
	}
	block, err := aes.NewCipher(key)
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

// Decrypt расшифровывает данные, полученные от Encrypt.
func Decrypt(key, data []byte) ([]byte, error) {
	if len(key) != KeySize {
		return nil, errors.New("invalid key size")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

// EncodeBase64 кодирует байты в base64.
func EncodeBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// DecodeBase64 декодирует base64-строку.
func DecodeBase64(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}
