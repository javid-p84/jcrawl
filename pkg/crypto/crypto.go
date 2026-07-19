package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"os"
)

// Manager handles encryption and decryption of sensitive data
type Manager struct {
	key []byte
}

func NewCryptoManager() (*Manager, error) {
	// Get encryption key from environment variable
	// Key must be 32 bytes for AES-256
	keyStr := os.Getenv("ENCRYPTION_KEY")
	if keyStr == "" {
		log.Println("WARNING: ENCRYPTION_KEY not set, using default (NOT SECURE FOR PRODUCTION)")
		// Generate a default key for development
		keyStr = "default-insecure-key-32-bytes-1234"
	}

	key := []byte(keyStr)
	if len(key) != 32 {
		return nil, fmt.Errorf("ENCRYPTION_KEY must be exactly 32 bytes (got %d)", len(key))
	}

	return &Manager{
		key: key,
	}, nil
}

// Encrypt encrypts plaintext using AES-256-GCM
func (m *Manager) Encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(m.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Create a nonce (IV)
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to create nonce: %w", err)
	}

	// Encrypt the plaintext
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	// Encode to base64 for storage
	encoded := base64.StdEncoding.EncodeToString(ciphertext)
	return encoded, nil
}

// Decrypt decrypts ciphertext using AES-256-GCM
func (m *Manager) Decrypt(ciphertext string) (string, error) {
	// Decode from base64
	decoded, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	block, err := aes.NewCipher(m.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Extract nonce and ciphertext
	nonceSize := gcm.NonceSize()
	if len(decoded) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ct := decoded[:nonceSize], decoded[nonceSize:]

	// Decrypt
	plaintext, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", fmt.Errorf("decryption failed: %w", err)
	}

	return string(plaintext), nil
}
