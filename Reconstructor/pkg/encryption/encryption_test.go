package encryption

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDecryptErrors(t *testing.T) {
	// Generate test key
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	keyString := hex.EncodeToString(key)

	// Create some sample encrypted data (this would normally come from the Divider)
	sampleData := []byte{
		// AES-GCM encrypted "test data" with the above key
		// For now, we'll just use some dummy bytes
		0x01, 0x02, 0x03, 0x04,
	}

	testCases := []struct {
		name        string
		data        []byte
		key         string
		expectError bool
	}{
		{
			name:        "invalid key format",
			data:        sampleData,
			key:         "not-a-hex-key",
			expectError: true,
		},
		{
			name:        "key too short",
			data:        sampleData,
			key:         "0123456789abcdef", // Only 8 bytes
			expectError: true,
		},
		{
			name:        "invalid data",
			data:        []byte("not encrypted data"),
			key:         keyString,
			expectError: true,
		},
		{
			name:        "truncated data",
			data:        sampleData[:len(sampleData)-1],
			key:         keyString,
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Decrypt(tc.data, tc.key)
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRealWorldVectors(t *testing.T) {
	// Test vectors based on actual Divider outputs
	// These test vectors should be created by copying real encrypted data from the Divider
	testVectors := []struct {
		name     string
		key      string
		input    []byte // Encrypted data from Divider
		expected []byte // Expected decrypted output
	}{
		// TODO: Add real test vectors extracted from Divider outputs
		// For testing compilation, we'll just use a simple case
		{
			name:     "example_vector",
			key:      "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			input:    []byte{0x01, 0x02, 0x03, 0x04}, // Placeholder
			expected: []byte("test"),                 // Expected decrypted output
		},
	}

	for _, tv := range testVectors {
		t.Run(tv.name, func(t *testing.T) {
			// Skip actual decryption until we have real test vectors
			t.Skip("Needs real test vectors from Divider")

			// This is how we would test it with real vectors:
			// decrypted, err := Decrypt(tv.input, tv.key)
			// require.NoError(t, err)
			// assert.Equal(t, tv.expected, decrypted)
		})
	}
}
