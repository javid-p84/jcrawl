package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
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
	// The AES-256 key is derived from ENCRYPTION_KEY via SHA-256, so the
	// passphrase can be any length. Changing the passphrase makes previously
	// encrypted values unreadable.
	keyStr := os.Getenv("ENCRYPTION_KEY")
	if keyStr == "" {
		log.Println("WARNING: ENCRYPTION_KEY not set, using default (NOT SECURE FOR PRODUCTION)")
		keyStr = "jcrawl-dev-only-default-key"
	}

	key := sha256.Sum256([]byte(keyStr))

	return &Manager{
		key: key[:],
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
