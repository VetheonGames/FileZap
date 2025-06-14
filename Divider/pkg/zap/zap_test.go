package zap

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateID(t *testing.T) {
	// Generate multiple IDs to test uniqueness
	id1, err := GenerateID()
	require.NoError(t, err)
	id2, err := GenerateID()
	require.NoError(t, err)

	// Verify IDs are valid hex strings
	_, err = hex.DecodeString(id1)
	assert.NoError(t, err)
	_, err = hex.DecodeString(id2)
	assert.NoError(t, err)

	// Verify IDs are different
	assert.NotEqual(t, id1, id2)

	// Verify ID length (16 bytes = 32 hex characters)
	assert.Equal(t, 32, len(id1))
	assert.Equal(t, 32, len(id2))
}

func TestChunkMetadata(t *testing.T) {
	chunk := &ChunkMetadata{
		Index: 1,
		Hash:  "original_hash",
		Size:  1024,
	}

	// Test updating encrypted hash
	err := chunk.UpdateEncryptedHash([]byte("test data"))
	require.NoError(t, err)

	// Verify encrypted hash is a valid hex string
	_, err = hex.DecodeString(chunk.EncryptedHash)
	assert.NoError(t, err)

	// Verify encrypted hash length (32 bytes = 64 hex characters)
	assert.Equal(t, 64, len(chunk.EncryptedHash))

	// Update again to verify it changes
	prevHash := chunk.EncryptedHash
	err = chunk.UpdateEncryptedHash([]byte("different data"))
	require.NoError(t, err)
	assert.NotEqual(t, prevHash, chunk.EncryptedHash)
}

func TestZapFileOperations(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "zap_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create test metadata
	testMeta := &FileMetadata{
		ID:            "test123",
		OriginalName:  "test.txt",
		ChunkCount:    2,
		TotalSize:     2048,
		EncryptionKey: "testkey",
		Chunks: []ChunkMetadata{
			{
				Index: 0,
				Hash:  "hash1",
				Size:  1024,
			},
			{
				Index: 1,
				Hash:  "hash2",
				Size:  1024,
			},
		},
	}

	// Update encrypted hashes
	for i := range testMeta.Chunks {
		err := testMeta.Chunks[i].UpdateEncryptedHash([]byte("test"))
		require.NoError(t, err)
	}

	// Create zap file
	err = CreateZapFile(testMeta, tempDir)
	require.NoError(t, err)

	// Verify file exists
	zapPath := filepath.Join(tempDir, testMeta.ID+".zap")
	_, err = os.Stat(zapPath)
	assert.NoError(t, err)

	// Read zap file
	readMeta, err := ReadZapFile(zapPath)
	require.NoError(t, err)

	// Verify metadata matches
	assert.Equal(t, testMeta.ID, readMeta.ID)
	assert.Equal(t, testMeta.OriginalName, readMeta.OriginalName)
	assert.Equal(t, testMeta.ChunkCount, readMeta.ChunkCount)
	assert.Equal(t, testMeta.TotalSize, readMeta.TotalSize)
	assert.Equal(t, testMeta.EncryptionKey, readMeta.EncryptionKey)
	assert.Equal(t, len(testMeta.Chunks), len(readMeta.Chunks))

	// Verify JSON formatting
	data, err := os.ReadFile(zapPath)
	require.NoError(t, err)
	var jsonMap map[string]interface{}
	err = json.Unmarshal(data, &jsonMap)
	assert.NoError(t, err)
}

func TestChunkValidation(t *testing.T) {
	// Create temporary directories
	tempDir, err := os.MkdirTemp("", "zap_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	chunksDir := filepath.Join(tempDir, "chunks")
	require.NoError(t, os.MkdirAll(chunksDir, 0755))

	// Create test metadata
	testMeta := &FileMetadata{
		ID:           "test123",
		OriginalName: "test.txt",
		ChunkCount:   2,
		TotalSize:    2048,
		Chunks: []ChunkMetadata{
			{
				Index:         0,
				Hash:          "hash1",
				Size:          1024,
				EncryptedHash: "enc_hash1",
			},
			{
				Index:         1,
				Hash:          "hash2",
				Size:          1024,
				EncryptedHash: "enc_hash2",
			},
		},
	}

	// Create fake chunk files
	for _, chunk := range testMeta.Chunks {
		chunkPath := filepath.Join(chunksDir, chunk.EncryptedHash)
		data := make([]byte, chunk.Size)
		require.NoError(t, os.WriteFile(chunkPath, data, 0644))
	}

	// Test validation
	err = ValidateChunks(testMeta, chunksDir)
	assert.NoError(t, err)

	// Test cleanup
	err = CleanupChunks(testMeta, chunksDir)
	assert.NoError(t, err)

	// Verify chunks are removed
	for _, chunk := range testMeta.Chunks {
		_, err := os.Stat(filepath.Join(chunksDir, chunk.EncryptedHash))
		assert.True(t, os.IsNotExist(err))
	}
}

func TestValidationErrors(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "zap_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	chunksDir := filepath.Join(tempDir, "chunks")
	require.NoError(t, os.MkdirAll(chunksDir, 0755))

	testMeta := &FileMetadata{
		ID:           "test123",
		OriginalName: "test.txt",
		ChunkCount:   1,
		TotalSize:    1024,
		Chunks: []ChunkMetadata{
			{
				Index:         0,
				Hash:          "hash1",
				Size:          1024,
				EncryptedHash: "enc_hash1",
			},
		},
	}

	t.Run("missing chunk", func(t *testing.T) {
		err := ValidateChunks(testMeta, chunksDir)
		assert.Error(t, err)
	})

	t.Run("wrong size chunk", func(t *testing.T) {
		// Create chunk with wrong size
		chunkPath := filepath.Join(chunksDir, testMeta.Chunks[0].EncryptedHash)
		data := make([]byte, 512) // Half the expected size
		require.NoError(t, os.WriteFile(chunkPath, data, 0644))

		err := ValidateChunks(testMeta, chunksDir)
		assert.Error(t, err)

		os.Remove(chunkPath)
	})

	t.Run("invalid chunks directory", func(t *testing.T) {
		err := ValidateChunks(testMeta, "/nonexistent/directory")
		assert.Error(t, err)
	})
}

func TestZapFileErrors(t *testing.T) {
	t.Run("nonexistent zap file", func(t *testing.T) {
		_, err := ReadZapFile("/nonexistent/file.zap")
		assert.Error(t, err)
	})

	t.Run("invalid json in zap file", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "zap_test_*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		zapPath := filepath.Join(tempDir, "invalid.zap")
		err = os.WriteFile(zapPath, []byte("invalid json"), 0644)
		require.NoError(t, err)

		_, err = ReadZapFile(zapPath)
		assert.Error(t, err)
	})

	t.Run("create in nonexistent directory", func(t *testing.T) {
		metadata := &FileMetadata{ID: "test"}
		err := CreateZapFile(metadata, "/nonexistent/directory")
		assert.Error(t, err)
	})
}
