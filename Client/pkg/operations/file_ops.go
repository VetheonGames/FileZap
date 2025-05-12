package operations

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/VetheonGames/FileZap/Divider/pkg/chunking"
	"github.com/VetheonGames/FileZap/Divider/pkg/encryption"
	"github.com/VetheonGames/FileZap/Divider/pkg/zap"
	"github.com/VetheonGames/FileZap/NetworkCore/pkg/types"
	"github.com/VetheonGames/FileZap/NetworkCore/pkg/validator"
)

// FileOperations handles file splitting and joining operations
type FileOperations struct {
	client *validator.Client
	config *types.ChunkStorageConfig
}

// NewFileOperations creates a new FileOperations instance
func NewFileOperations(client *validator.Client, storageDir string) *FileOperations {
	if storageDir == "" {
		storageDir = filepath.Join(os.TempDir(), "filezap", "chunks")
	}

	return &FileOperations{
		client: client,
		config: &types.ChunkStorageConfig{
			StorageDir:     storageDir,
			MaxStorageSize: 10 * 1024 * 1024 * 1024, // 10GB default
			AutoCleanup:    true,
		},
	}
}

// SetStorageConfig updates the chunk storage configuration
func (f *FileOperations) SetStorageConfig(config *types.ChunkStorageConfig) {
	f.config = config
}

// GetStorageConfig returns the current chunk storage configuration
func (f *FileOperations) GetStorageConfig() *types.ChunkStorageConfig {
	return f.config
}

// SplitFile splits a file into chunks and creates a zap file
func (f *FileOperations) SplitFile(inputPath, outputDir string, chunkSizeStr string) error {
	// Parse chunk size
	chunkSize, err := strconv.ParseInt(chunkSizeStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid chunk size: %v", err)
	}

	// Create chunks directory using configured storage
	chunksDir := f.config.StorageDir
	if err := os.MkdirAll(chunksDir, 0755); err != nil {
		return fmt.Errorf("failed to create chunks directory: %v", err)
	}

	// Split file into chunks
	chunks, err := chunking.SplitFile(inputPath, chunkSize, chunksDir)
	if err != nil {
		return fmt.Errorf("failed to split file: %v", err)
	}

	// Check available space
	if err := f.ensureStorageSpace(chunkSize * int64(len(chunks))); err != nil {
		return fmt.Errorf("insufficient storage space: %v", err)
	}

	// Generate unique ID
	id, err := zap.GenerateID()
	if err != nil {
		return fmt.Errorf("failed to generate ID: %v", err)
	}

	// Generate encryption key and get RSA key pair
	key, err := encryption.GenerateKey()
	if err != nil {
		return fmt.Errorf("failed to generate encryption key: %v", err)
	}

// Create client keypair for secure key retrieval
privKey, err := rsa.GenerateKey(rand.Reader, 2048)
if err != nil {
return fmt.Errorf("failed to generate client keypair: %v", err)
}

// Register file with validator network
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(&privKey.PublicKey)
	if err != nil {
		return fmt.Errorf("failed to marshal public key: %v", err)
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

	// Create zap metadata without raw key
	metadata := &zap.FileMetadata{
		ID:           id,
		OriginalName: filepath.Base(inputPath),
		ChunkCount:   len(chunks),
		TotalSize:    chunkSize * int64(len(chunks)),
		Chunks:       zapChunks,
	}

	// Request validators to store key shares
	if err := f.client.RegisterEncryptionKey(id, key, pubKeyBytes); err != nil {
		return fmt.Errorf("failed to register encryption key: %v", err)
	}

	// Write zap file
	if err := zap.CreateZapFile(metadata, outputDir); err != nil {
		return fmt.Errorf("failed to create zap file: %v", err)
	}

	// If we have a validator client, register the file
	if f.client != nil && f.client.IsConnected() {
		fileInfo := types.FileInfo{
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

// ensureStorageSpace checks if there's enough space for the new chunks
func (f *FileOperations) ensureStorageSpace(requiredSpace int64) error {
	if f.config.MaxStorageSize == 0 {
		return nil // No size limit
	}

	// Get current storage usage
	var totalSize int64
	err := filepath.Walk(f.config.StorageDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to calculate storage usage: %v", err)
	}

	// Check if we have enough space
	if totalSize+requiredSpace > f.config.MaxStorageSize {
		if !f.config.AutoCleanup {
			return fmt.Errorf("storage limit exceeded")
		}

		// Find and remove old chunks until we have enough space
		// This is a simple implementation - could be improved with LRU cache
		files, err := os.ReadDir(f.config.StorageDir)
		if err != nil {
			return fmt.Errorf("failed to read storage directory: %v", err)
		}

		// Sort files by modification time
		type fileInfo struct {
			path    string
			modTime time.Time
		}
		var filesInfo []fileInfo
		for _, file := range files {
			info, err := file.Info()
			if err != nil {
				continue
			}
			filesInfo = append(filesInfo, fileInfo{
				path:    filepath.Join(f.config.StorageDir, file.Name()),
				modTime: info.ModTime(),
			})
		}
		sort.Slice(filesInfo, func(i, j int) bool {
			return filesInfo[i].modTime.Before(filesInfo[j].modTime)
		})

		// Remove oldest files until we have enough space
		for _, file := range filesInfo {
			info, err := os.Stat(file.path)
			if err != nil {
				continue
			}
			totalSize -= info.Size()
			if err := os.Remove(file.path); err != nil {
				continue
			}
			if totalSize+requiredSpace <= f.config.MaxStorageSize {
				break
			}
		}

		// Check if we now have enough space
		if totalSize+requiredSpace > f.config.MaxStorageSize {
			return fmt.Errorf("could not free enough storage space")
		}
	}

	return nil
}

// JoinFile reassembles a file from chunks using a zap file
func (f *FileOperations) JoinFile(zapPath, outputDir string) error {
	// Read zap metadata
	metadata, err := zap.ReadZapFile(zapPath)
	if err != nil {
		return fmt.Errorf("failed to read zap file: %v", err)
	}

	// Create client keypair for secure key retrieval
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate client keypair: %v", err)
	}

	// Request decryption key from validator network
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(&privKey.PublicKey)
	if err != nil {
		return fmt.Errorf("failed to marshal public key: %v", err)
	}

	decryptionKey, err := f.client.RequestDecryptionKey(metadata.ID, pubKeyBytes)
	if err != nil {
		return fmt.Errorf("failed to get decryption key: %v", err)
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

		// Decrypt chunk using retrieved key
		decrypted, err := encryption.Decrypt(encryptedData, decryptionKey)
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
