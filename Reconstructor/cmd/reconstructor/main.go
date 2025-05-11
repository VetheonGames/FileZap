package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/VetheonGames/FileZap/Reconstructor/pkg/chunking"
	"github.com/VetheonGames/FileZap/Reconstructor/pkg/encryption"
	"github.com/VetheonGames/FileZap/Reconstructor/pkg/zap"
)

func main() {
	// Command line flags
	zapFile := flag.String("zap", "", "Path to .zap file containing chunk metadata")
	outputPath := flag.String("output", "", "Output path for reconstructed file")

	flag.Parse()

	// Validate flags
	if *zapFile == "" {
		fmt.Println("Error: .zap file path is required")
		flag.Usage()
		os.Exit(1)
	}

	if *outputPath == "" {
		fmt.Println("Error: Output path is required")
		flag.Usage()
		os.Exit(1)
	}

	// Create output directory if it doesn't exist
	outputDir := filepath.Dir(*outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Printf("Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	if err := reconstruct(*zapFile, *outputPath); err != nil {
		fmt.Printf("Error during reconstruction: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("File successfully reconstructed!")
}

func reconstruct(zapPath, outputPath string) error {
	// Read and validate zap file
	metadata, err := zap.ReadZapFile(zapPath)
	if err != nil {
		return fmt.Errorf("failed to read zap file: %v", err)
	}

	// Validate chunks directory exists
	chunksDir := filepath.Join(filepath.Dir(zapPath), "chunks")
	if err := zap.ValidateChunks(metadata, chunksDir); err != nil {
		return fmt.Errorf("chunk validation failed: %v", err)
	}

	// Create temporary directory for decrypted chunks
	tempDir := filepath.Join(filepath.Dir(outputPath), ".tmp")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	var chunkInfos []chunking.ChunkInfo
	// Process each chunk
	for _, chunk := range metadata.Chunks {
		encryptedPath := filepath.Join(chunksDir, chunk.EncryptedHash)
		encryptedData, err := os.ReadFile(encryptedPath)
		if err != nil {
			return fmt.Errorf("failed to read encrypted chunk: %v", err)
		}

		// Decrypt chunk
		decrypted, err := encryption.Decrypt(encryptedData, metadata.EncryptionKey)
		if err != nil {
			return fmt.Errorf("failed to decrypt chunk %d: %v", chunk.Index, err)
		}

		// Store decrypted chunk in temp directory
		tempPath := filepath.Join(tempDir, chunk.Hash)
		if err := os.WriteFile(tempPath, decrypted, 0644); err != nil {
			return fmt.Errorf("failed to write decrypted chunk: %v", err)
		}

		// Validate decrypted chunk
		if err := zap.ValidateChunk(chunk, tempPath, decrypted); err != nil {
			return fmt.Errorf("chunk validation failed: %v", err)
		}

		chunkInfos = append(chunkInfos, chunking.ChunkInfo{
			Index:    chunk.Index,
			Hash:     chunk.Hash,
			Size:     chunk.Size,
			Filename: tempPath,
		})
	}

	// Reassemble file
	if err := chunking.ReassembleFile(chunkInfos, outputPath); err != nil {
		return fmt.Errorf("failed to reassemble file: %v", err)
	}

	// Cleanup temporary files
	chunking.CleanupTempFiles(chunkInfos)

	return nil
}
