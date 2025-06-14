package zap

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func createTestZapFile(t *testing.T, dir string) (*FileMetadata, string) {
	// Create test metadata
	metadata := &FileMetadata{
		ID:            "test123",
		OriginalName:  "test.txt",
		ChunkCount:    2,
		TotalSize:     2048,
		EncryptionKey: "testkey",
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

	// Write metadata to file
	zapPath := filepath.Join(dir, metadata.ID+".zap")
	data, err := json.MarshalIndent(metadata, "", "  ")
	assert.NoError(t, err)

	err = os.WriteFile(zapPath, data, 0644)
	assert.NoError(t, err)

	return metadata, zapPath
}

func TestReadZapFile(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "zap_test_*")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create test zap file
	expectedMetadata, zapPath := createTestZapFile(t, tempDir)

	// Read zap file
	metadata, err := ReadZapFile(zapPath)
	assert.NoError(t, err)

	// Verify metadata matches
	assert.Equal(t, expectedMetadata.ID, metadata.ID)
	assert.Equal(t, expectedMetadata.OriginalName, metadata.OriginalName)
	assert.Equal(t, expectedMetadata.ChunkCount, metadata.ChunkCount)
	assert.Equal(t, expectedMetadata.TotalSize, metadata.TotalSize)
	assert.Equal(t, expectedMetadata.EncryptionKey, metadata.EncryptionKey)
	assert.Equal(t, len(expectedMetadata.Chunks), len(metadata.Chunks))

	// Verify chunk metadata
	for i, expectedChunk := range expectedMetadata.Chunks {
		actualChunk := metadata.Chunks[i]
		assert.Equal(t, expectedChunk.Index, actualChunk.Index)
		assert.Equal(t, expectedChunk.Hash, actualChunk.Hash)
		assert.Equal(t, expectedChunk.Size, actualChunk.Size)
		assert.Equal(t, expectedChunk.EncryptedHash, actualChunk.EncryptedHash)
	}
}

func TestChunkValidation(t *testing.T) {
	// Create temporary directories
	tempDir, err := os.MkdirTemp("", "zap_test_*")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	chunksDir := filepath.Join(tempDir, "chunks")
	err = os.MkdirAll(chunksDir, 0755)
	assert.NoError(t, err)

	// Create test metadata
	metadata, _ := createTestZapFile(t, tempDir)

	// Test cases
	t.Run("missing chunks directory", func(t *testing.T) {
		err := ValidateChunks(metadata, "/nonexistent/directory")
		assert.Error(t, err)
	})

	t.Run("missing chunks", func(t *testing.T) {
		err := ValidateChunks(metadata, chunksDir)
		assert.Error(t, err)
	})

	t.Run("successful validation", func(t *testing.T) {
		// Create chunk files
		for _, chunk := range metadata.Chunks {
			chunkPath := filepath.Join(chunksDir, chunk.EncryptedHash)
			data := make([]byte, chunk.Size)
			err := os.WriteFile(chunkPath, data, 0644)
			assert.NoError(t, err)
		}

		err := ValidateChunks(metadata, chunksDir)
		assert.NoError(t, err)
	})

	t.Run("wrong size chunks", func(t *testing.T) {
		// Create chunk with wrong size
		wrongSizeChunk := metadata.Chunks[0]
		chunkPath := filepath.Join(chunksDir, wrongSizeChunk.EncryptedHash)
		data := make([]byte, wrongSizeChunk.Size-1) // One byte too small
		err := os.WriteFile(chunkPath, data, 0644)
		assert.NoError(t, err)

		err = ValidateChunks(metadata, chunksDir)
		assert.Error(t, err)
	})
}

func TestZapFileErrors(t *testing.T) {
	t.Run("nonexistent file", func(t *testing.T) {
		_, err := ReadZapFile("/nonexistent/file.zap")
		assert.Error(t, err)
	})

	t.Run("invalid json", func(t *testing.T) {
		// Create temporary directory
		tempDir, err := os.MkdirTemp("", "zap_test_*")
		assert.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Create invalid zap file
		zapPath := filepath.Join(tempDir, "invalid.zap")
		err = os.WriteFile(zapPath, []byte("invalid json"), 0644)
		assert.NoError(t, err)

		_, err = ReadZapFile(zapPath)
		assert.Error(t, err)
	})

	t.Run("missing required fields", func(t *testing.T) {
		// Create temporary directory
		tempDir, err := os.MkdirTemp("", "zap_test_*")
		assert.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Create incomplete metadata
		incompleteData := map[string]interface{}{
			"id": "test123",
			// Missing other required fields
		}

		// Write to file
		zapPath := filepath.Join(tempDir, "incomplete.zap")
		data, err := json.Marshal(incompleteData)
		assert.NoError(t, err)
		err = os.WriteFile(zapPath, data, 0644)
		assert.NoError(t, err)

		_, err = ReadZapFile(zapPath)
		assert.Error(t, err)
	})
}
