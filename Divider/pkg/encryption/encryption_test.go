package encryption

import (
    "bytes"
    "encoding/hex"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestGenerateKey(t *testing.T) {
    // Generate multiple keys to ensure uniqueness
    key1, err := GenerateKey()
    require.NoError(t, err)
    key2, err := GenerateKey()
    require.NoError(t, err)

    // Verify keys are valid hex strings
    _, err = hex.DecodeString(key1)
    assert.NoError(t, err)
    _, err = hex.DecodeString(key2)
    assert.NoError(t, err)

    // Verify keys are different
    assert.NotEqual(t, key1, key2)

    // Verify key length (32 bytes = 64 hex characters)
    assert.Equal(t, 64, len(key1))
    assert.Equal(t, 64, len(key2))
}

func TestEncryptDecrypt(t *testing.T) {
    // Generate a key
    key, err := GenerateKey()
    require.NoError(t, err)

    testCases := []struct {
        name        string
        data        []byte
        shouldError bool
    }{
        {
            name:        "empty data",
            data:        []byte{},
            shouldError: false,
        },
        {
            name:        "small data",
            data:        []byte("test data"),
            shouldError: false,
        },
        {
            name:        "large data",
            data:        bytes.Repeat([]byte("large data test "), 1000),
            shouldError: false,
        },
        {
            name:        "binary data",
            data:        []byte{0x00, 0xFF, 0x10, 0xAB, 0xCD},
            shouldError: false,
        },
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            // Encrypt data
            encrypted, err := Encrypt(tc.data, key)
            if tc.shouldError {
                assert.Error(t, err)
                return
            }
            require.NoError(t, err)

            // Verify encrypted data is different from original
            assert.NotEqual(t, tc.data, encrypted)

            // Decrypt data
            decrypted, err := Decrypt(encrypted, key)
            require.NoError(t, err)

            // Verify decrypted data matches original
            assert.Equal(t, tc.data, decrypted)
        })
    }
}

func TestEncryptionErrors(t *testing.T) {
    // Generate valid key and data
    validKey, err := GenerateKey()
    require.NoError(t, err)
    validData := []byte("test data")

    t.Run("invalid key format", func(t *testing.T) {
        _, err := Encrypt(validData, "not-a-hex-key")
        assert.Error(t, err)
    })

    t.Run("key too short", func(t *testing.T) {
        shortKey := "0123456789abcdef" // Only 8 bytes
        _, err := Encrypt(validData, shortKey)
        assert.Error(t, err)
    })

    t.Run("different key decryption", func(t *testing.T) {
        // Encrypt with first key
        encrypted, err := Encrypt(validData, validKey)
        require.NoError(t, err)

        // Try to decrypt with different key
        differentKey, err := GenerateKey()
        require.NoError(t, err)
        _, err = Decrypt(encrypted, differentKey)
        assert.Error(t, err)
    })
}

func TestDecryptionErrors(t *testing.T) {
    key, err := GenerateKey()
    require.NoError(t, err)

    t.Run("invalid encrypted data", func(t *testing.T) {
        invalidData := []byte("not encrypted data")
        _, err := Decrypt(invalidData, key)
        assert.Error(t, err)
    })

    t.Run("corrupted encrypted data", func(t *testing.T) {
        // Encrypt some data
        encrypted, err := Encrypt([]byte("test data"), key)
        require.NoError(t, err)

        // Corrupt the encrypted data
        encrypted[len(encrypted)-1] ^= 0xFF
        _, err = Decrypt(encrypted, key)
        assert.Error(t, err)
    })

    t.Run("truncated encrypted data", func(t *testing.T) {
        // Encrypt some data
        encrypted, err := Encrypt([]byte("test data"), key)
        require.NoError(t, err)

        // Truncate the encrypted data
        truncated := encrypted[:len(encrypted)-1]
        _, err = Decrypt(truncated, key)
        assert.Error(t, err)
    })
}
