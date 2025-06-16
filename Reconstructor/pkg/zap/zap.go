package zap

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// FileMetadata represents the metadata stored in a .zap file
type FileMetadata struct {
	ID            string          `json:"id"`
	OriginalName  string          `json:"original_name"`
	ChunkCount    int             `json:"chunk_count"`
	TotalSize     int64           `json:"total_size"`
	EncryptionKey string          `json:"encryption_key"`
	Chunks        []ChunkMetadata `json:"chunks"`
}

// ChunkMetadata represents metadata for a single encrypted chunk
type ChunkMetadata struct {
	Index         int    `json:"index"`
	Hash          string `json:"hash"`
	Size          int64  `json:"size"`
	EncryptedHash string `json:"encrypted_hash"`
}

// ReadZapFile reads and parses a .zap file with enhanced validation
func ReadZapFile(zapPath string) (*FileMetadata, error) {
	data, err := os.ReadFile(zapPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read zap file: %v", err)
	}

	var metadata FileMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse zap file: %v", err)
	}

	// Basic validation
	if metadata.ID == "" || metadata.OriginalName == "" || metadata.ChunkCount <= 0 ||
		metadata.TotalSize <= 0 || metadata.EncryptionKey == "" || len(metadata.Chunks) == 0 {
		return nil, fmt.Errorf("invalid zap file: missing required fields")
	}

	// Validate chunk count matches actual chunks
	if len(metadata.Chunks) != metadata.ChunkCount {
		return nil, fmt.Errorf("chunk count mismatch: expected %d, got %d",
			metadata.ChunkCount, len(metadata.Chunks))
	}

	// Validate chunk indexes are sequential and unique
	seen := make(map[int]bool)
	for _, chunk := range metadata.Chunks {
		if chunk.Index < 0 || chunk.Index >= metadata.ChunkCount {
			return nil, fmt.Errorf("invalid chunk index: %d", chunk.Index)
		}
		if seen[chunk.Index] {
			return nil, fmt.Errorf("duplicate chunk index: %d", chunk.Index)
		}
		seen[chunk.Index] = true
	}

	return &metadata, nil
}

// ValidateChunk performs comprehensive validation of a single chunk
func ValidateChunk(chunk ChunkMetadata, chunkPath string, decryptedData []byte) error {
	// Check if chunk exists
	if _, err := os.Stat(chunkPath); err != nil {
		return fmt.Errorf("chunk file missing or inaccessible: %s", chunk.EncryptedHash)
	}

	// Validate decrypted data size
	if int64(len(decryptedData)) != chunk.Size {
		return fmt.Errorf("chunk size mismatch: expected %d, got %d",
			chunk.Size, len(decryptedData))
	}

	// Verify chunk hash
	hash := sha256.Sum256(decryptedData)
	if hex.EncodeToString(hash[:]) != chunk.Hash {
		return fmt.Errorf("chunk hash mismatch: possible tampering detected")
	}

	return nil
}

// ValidateChunks verifies all chunks exist and have correct sizes
func ValidateChunks(metadata *FileMetadata, chunksDir string) error {
	var totalSize int64
	for _, chunk := range metadata.Chunks {
		chunkPath := filepath.Join(chunksDir, chunk.EncryptedHash)
		info, err := os.Stat(chunkPath)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("chunk file missing: %s", chunk.EncryptedHash)
			}
			return fmt.Errorf("failed to access chunk: %v", err)
		}

// Verify encrypted chunk size
if info.Size() != chunk.Size {
    return fmt.Errorf("chunk size mismatch for %s: expected %d, got %d",
        chunk.EncryptedHash, chunk.Size, info.Size())
}

		// Track total size for final validation
		totalSize += chunk.Size
	}

	// Validate total size
	if totalSize != metadata.TotalSize {
		return fmt.Errorf("total size mismatch: expected %d, got %d",
			metadata.TotalSize, totalSize)
	}

	return nil
}
