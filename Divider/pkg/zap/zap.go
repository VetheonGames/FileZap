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
	EncryptionKey string          `json:"encryption_key,omitempty"`
	Chunks        []ChunkMetadata `json:"chunks"`
}

// ChunkMetadata represents metadata for a single encrypted chunk
type ChunkMetadata struct {
	Index         int    `json:"index"`          // Index of the chunk in the original file
	Hash          string `json:"hash"`           // Hash of the original chunk data
	Size          int64  `json:"size"`           // Size of the original chunk
	EncryptedHash string `json:"encrypted_hash"` // Hash of the encrypted chunk data
}

// UpdateEncryptedHash updates the encrypted hash for a chunk
func (c *ChunkMetadata) UpdateEncryptedHash(encryptedData []byte) error {
	hash := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, hash); err != nil {
		return fmt.Errorf("failed to generate encrypted hash: %v", err)
	}
	c.EncryptedHash = hex.EncodeToString(hash)
	return nil
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
    // Check if the path is valid for the current OS
    if filepath.VolumeName(outputDir) == "" && (len(outputDir) > 0 && (outputDir[0] == '/' || outputDir[0] == '\\')) {
        return fmt.Errorf("invalid output directory path: must be a valid OS-specific path")
    }

    // Clean and get the absolute path
    outputDir = filepath.Clean(outputDir)
    absOutputDir, err := filepath.Abs(outputDir)
    if err != nil {
        return fmt.Errorf("invalid output directory path: %v", err)
    }

    // Check if output directory exists and is accessible
    dirInfo, err := os.Stat(absOutputDir)
    if err != nil {
        return fmt.Errorf("invalid output directory: %v", err)
    }
    
    // Ensure it's actually a directory
    if !dirInfo.IsDir() {
        return fmt.Errorf("output path is not a directory")
    }

    // Validate output directory is writable by trying to create a test file
    testFile := filepath.Join(outputDir, ".test")
    if err := os.WriteFile(testFile, []byte{}, 0644); err != nil {
        return fmt.Errorf("output directory not writable: %v", err)
    }
    os.Remove(testFile) // Clean up test file

    metadataBytes, err := json.MarshalIndent(metadata, "", "  ")
    if err != nil {
        return fmt.Errorf("failed to marshal metadata: %v", err)
    }

    zapPath := filepath.Join(outputDir, fmt.Sprintf("%s.zap", metadata.ID))
    if err := os.WriteFile(zapPath, metadataBytes, 0644); err != nil {
        return fmt.Errorf("failed to write zap file: %v", err)
    }

    return nil
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
	// Create chunks directory if it doesn't exist
	if err := os.MkdirAll(chunksDir, 0755); err != nil {
		return fmt.Errorf("failed to create chunks directory: %v", err)
	}

	for _, chunk := range metadata.Chunks {
		chunkPath := filepath.Join(chunksDir, chunk.EncryptedHash)

		// Check if chunk exists
		if _, err := os.Stat(chunkPath); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("chunk file missing: %s", chunk.EncryptedHash)
			}
			return fmt.Errorf("failed to check chunk file: %v", err)
		}

		// Read chunk data
		data, err := os.ReadFile(chunkPath)
		if err != nil {
			return fmt.Errorf("failed to read chunk %s: %v", chunk.EncryptedHash, err)
		}

		// Verify chunk size
		if int64(len(data)) != chunk.Size {
			return fmt.Errorf("chunk %s size mismatch: expected %d, got %d",
				chunk.EncryptedHash, chunk.Size, len(data))
		}
	}
	return nil
}

// CleanupChunks removes all chunk files
func CleanupChunks(metadata *FileMetadata, chunksDir string) error {
	for _, chunk := range metadata.Chunks {
		chunkPath := filepath.Join(chunksDir, chunk.EncryptedHash)
		if err := os.Remove(chunkPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove chunk %s: %v", chunk.EncryptedHash, err)
		}
	}
	return nil
}
