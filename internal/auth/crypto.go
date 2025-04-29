package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
)

var (
	encryptionKey []byte
)

// InitCrypto initializes the encryption by setting up the encryption key
func InitCrypto() error {
	// Get and validate encryption key
	keyEnv := "REFRESH_TOKEN_ENCRYPTION_KEY"
	key := os.Getenv(keyEnv)
	if key == "" {
		return fmt.Errorf("required environment variable %s is not set", keyEnv)
	}

	// Store the key for later use
	encryptionKey = []byte(key)

	// Validate key length for AES-256 (32 bytes)
	if len(encryptionKey) != 32 {
		return fmt.Errorf("%s must be exactly 32 bytes long for AES-256 encryption", keyEnv)
	}

	log.Println("Token encryption initialized successfully")
	return nil
}

// EncryptRefreshToken encrypts a refresh token using AES-256
func EncryptRefreshToken(token string) (string, error) {
	if token == "" {
		return "", nil
	}

	// Validate encryption is initialized
	if len(encryptionKey) == 0 {
		return "", errors.New("encryption key not initialized, call InitCrypto first")
	}

	// Create a new cipher block from the key
	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create a new GCM cipher with the default nonce size
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Create a nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt the data
	ciphertext := gcm.Seal(nonce, nonce, []byte(token), nil)

	// Return base64 encoded data
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptRefreshToken decrypts an encrypted refresh token
func DecryptRefreshToken(encryptedToken string) (string, error) {
	if encryptedToken == "" {
		return "", nil
	}

	// Validate encryption is initialized
	if len(encryptionKey) == 0 {
		return "", errors.New("encryption key not initialized, call InitCrypto first")
	}

	// Decode the base64 encoded data
	ciphertext, err := base64.StdEncoding.DecodeString(encryptedToken)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	// Create a new cipher block from the key
	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create a new GCM cipher with the default nonce size
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Ensure the ciphertext is at least the size of the nonce
	if len(ciphertext) < gcm.NonceSize() {
		return "", errors.New("ciphertext too short")
	}

	// Extract the nonce and decrypt the data
	nonce, ciphertext := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}
