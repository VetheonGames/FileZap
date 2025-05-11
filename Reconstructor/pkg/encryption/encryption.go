package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"fmt"
)

// Decrypt decrypts data using AES-GCM with additional validation
func Decrypt(encrypted []byte, keyString string) ([]byte, error) {
	// Validate key format
	key, err := hex.DecodeString(keyString)
	if err != nil {
		return nil, fmt.Errorf("invalid encryption key format: %v", err)
	}

	// Validate key size (must be 32 bytes for AES-256)
	if len(key) != 32 {
		return nil, fmt.Errorf("invalid key size: expected 32 bytes, got %d", len(key))
	}

	// Create cipher block
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %v", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %v", err)
	}

	// Get nonce size
	nonceSize := gcm.NonceSize()

	// Validate encrypted data length
	if len(encrypted) < nonceSize {
		return nil, fmt.Errorf("encrypted data too short: must be at least %d bytes", nonceSize)
	}

	// Extract nonce and ciphertext
	nonce, ciphertext := encrypted[:nonceSize], encrypted[nonceSize:]

	// Decrypt and authenticate the data
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: possible tampering detected")
	}

	return plaintext, nil
}
