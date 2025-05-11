package zap

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
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

// GenerateID creates a unique ID for a .zap file
func GenerateID() (string, error) {
	id := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, id); err != nil {
		return "", err
	}
	return hex.EncodeToString(id), nil
}

// CreateZapFile creates a .zap file with the provided metadata
func CreateZapFile(metadata *FileMetadata, outputDir string) error {
	metadataBytes, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}

	zapPath := filepath.Join(outputDir, fmt.Sprintf("%s.zap", metadata.ID))
	return os.WriteFile(zapPath, metadataBytes, 0644)
}

// ReadZapFile reads and parses a .zap file
func ReadZapFile(zapPath string) (*FileMetadata, error) {
	data, err := os.ReadFile(zapPath)
	if err != nil {
		return nil, err
	}

	var metadata FileMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, err
	}

	return &metadata, nil
}

// ValidateChunks verifies that all chunks exist and have correct hashes
func ValidateChunks(metadata *FileMetadata, chunksDir string) error {
	for _, chunk := range metadata.Chunks {
		chunkPath := filepath.Join(chunksDir, chunk.EncryptedHash)
		if _, err := os.Stat(chunkPath); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("chunk file missing: %s", chunk.EncryptedHash)
			}
			return err
		}
	}
	return nil
}
