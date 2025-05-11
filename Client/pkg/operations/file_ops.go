package operations

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/VetheonGames/FileZap/Divider/pkg/chunking"
	"github.com/VetheonGames/FileZap/Divider/pkg/encryption"
	"github.com/VetheonGames/FileZap/Divider/pkg/zap"
	"github.com/VetheonGames/FileZap/NetworkCore/pkg/validator"
)

// FileOperations handles file splitting and joining operations
type FileOperations struct {
	client *validator.Client
}

// NewFileOperations creates a new FileOperations instance
func NewFileOperations(client *validator.Client) *FileOperations {
	return &FileOperations{
		client: client,
	}
}

// SplitFile splits a file into chunks and creates a zap file
func (f *FileOperations) SplitFile(inputPath, outputDir string, chunkSizeStr string) error {
	// Parse chunk size
	chunkSize, err := strconv.ParseInt(chunkSizeStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid chunk size: %v", err)
	}

	// Create chunks directory
	chunksDir := filepath.Join(outputDir, "chunks")
	if err := os.MkdirAll(chunksDir, 0755); err != nil {
		return fmt.Errorf("failed to create chunks directory: %v", err)
	}

	// Generate encryption key
	key, err := encryption.GenerateKey()
	if err != nil {
		return fmt.Errorf("failed to generate encryption key: %v", err)
	}

	// Split file into chunks
	chunks, err := chunking.SplitFile(inputPath, chunkSize, chunksDir)
	if err != nil {
		return fmt.Errorf("failed to split file: %v", err)
	}

	// Generate unique ID
	id, err := zap.GenerateID()
	if err != nil {
		return fmt.Errorf("failed to generate ID: %v", err)
	}

	// Process chunks
	var zapChunks []zap.ChunkMetadata
	for _, chunk := range chunks {
		// Read chunk
		data, err := os.ReadFile(chunk.Filename)
		if err != nil {
			return fmt.Errorf("failed to read chunk: %v", err)
		}

		// Encrypt chunk
		encrypted, err := encryption.Encrypt(data, key)
		if err != nil {
			return fmt.Errorf("failed to encrypt chunk: %v", err)
		}

		// Write encrypted chunk
		encryptedPath := filepath.Join(chunksDir, chunk.Hash)
		if err := os.WriteFile(encryptedPath, encrypted, 0644); err != nil {
			return fmt.Errorf("failed to write encrypted chunk: %v", err)
		}

		zapChunks = append(zapChunks, zap.ChunkMetadata{
			Index:         chunk.Index,
			Hash:          chunk.Hash,
			Size:          chunk.Size,
			EncryptedHash: chunk.Hash,
		})
	}

	// Create zap metadata
	metadata := &zap.FileMetadata{
		ID:            id,
		OriginalName:  filepath.Base(inputPath),
		ChunkCount:    len(chunks),
		TotalSize:     chunkSize * int64(len(chunks)),
		EncryptionKey: key,
		Chunks:        zapChunks,
	}

	// Write zap file
	if err := zap.CreateZapFile(metadata, outputDir); err != nil {
		return fmt.Errorf("failed to create zap file: %v", err)
	}

	// If we have a validator client, register the file
	if f.client != nil && f.client.IsConnected() {
		fileInfo := validator.FileInfo{
			Name:      metadata.ID + ".zap",
			ChunkIDs:  make([]string, len(zapChunks)),
			Available: true,
		}
		for i, chunk := range zapChunks {
			fileInfo.ChunkIDs[i] = chunk.Hash
		}
		if err := f.client.UploadZapFile(fileInfo); err != nil {
			return fmt.Errorf("failed to register file with validator: %v", err)
		}
	}

	return nil
}

// JoinFile reassembles a file from chunks using a zap file
func (f *FileOperations) JoinFile(zapPath, outputDir string) error {
	// Read zap file
	metadata, err := zap.ReadZapFile(zapPath)
	if err != nil {
		return fmt.Errorf("failed to read zap file: %v", err)
	}

	// Validate chunks
	chunksDir := filepath.Join(filepath.Dir(zapPath), "chunks")
	if err := zap.ValidateChunks(metadata, chunksDir); err != nil {
		return fmt.Errorf("chunk validation failed: %v", err)
	}

	// Create temporary directory for decrypted chunks
	tempDir := filepath.Join(outputDir, "temp")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	var chunkInfos []chunking.ChunkInfo
	// Process each chunk
	for _, chunk := range metadata.Chunks {
		// Read encrypted chunk
		encryptedData, err := os.ReadFile(filepath.Join(chunksDir, chunk.EncryptedHash))
		if err != nil {
			return fmt.Errorf("failed to read encrypted chunk: %v", err)
		}

		// Decrypt chunk
		decrypted, err := encryption.Decrypt(encryptedData, metadata.EncryptionKey)
		if err != nil {
			return fmt.Errorf("failed to decrypt chunk: %v", err)
		}

		// Write decrypted chunk
		tempPath := filepath.Join(tempDir, chunk.Hash)
		if err := os.WriteFile(tempPath, decrypted, 0644); err != nil {
			return fmt.Errorf("failed to write decrypted chunk: %v", err)
		}

		chunkInfos = append(chunkInfos, chunking.ChunkInfo{
			Index:    chunk.Index,
			Hash:     chunk.Hash,
			Size:     chunk.Size,
			Filename: tempPath,
		})
	}

	// Reassemble file
	outputPath := filepath.Join(outputDir, metadata.OriginalName)
	if err := chunking.ReassembleFile(chunkInfos, outputPath); err != nil {
		return fmt.Errorf("failed to reassemble file: %v", err)
	}

	return nil
}
