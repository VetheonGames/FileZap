package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/VetheonGames/FileZap/Divider/pkg/chunking"
	"github.com/VetheonGames/FileZap/Divider/pkg/encryption"
	"github.com/VetheonGames/FileZap/Divider/pkg/zap"
)

func main() {
	// Command line flags
	inputFile := flag.String("input", "", "Input file to process")
	outputDir := flag.String("output", "", "Output directory for chunks and zap file")
	chunkSize := flag.Int64("chunksize", chunking.DefaultChunkSize, "Size of each chunk in bytes")
	mode := flag.String("mode", "split", "Mode: 'split' to divide file or 'join' to reassemble")
	zapFile := flag.String("zap", "", "Path to .zap file (required for join mode)")

	flag.Parse()

	// Validate flags
	if *inputFile == "" {
		fmt.Println("Error: Input file is required")
		flag.Usage()
		os.Exit(1)
	}

	if *outputDir == "" {
		fmt.Println("Error: Output directory is required")
		flag.Usage()
		os.Exit(1)
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		fmt.Printf("Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	switch *mode {
	case "split":
		if err := splitMode(*inputFile, *outputDir, *chunkSize); err != nil {
			fmt.Printf("Error in split mode: %v\n", err)
			os.Exit(1)
		}
	case "join":
		if *zapFile == "" {
			fmt.Println("Error: ZAP file is required for join mode")
			flag.Usage()
			os.Exit(1)
		}
		if err := joinMode(*zapFile, *outputDir); err != nil {
			fmt.Printf("Error in join mode: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Printf("Error: Invalid mode '%s'. Use 'split' or 'join'\n", *mode)
		flag.Usage()
		os.Exit(1)
	}
}

func splitMode(inputFile, outputDir string, chunkSize int64) error {
	// Generate encryption key
	key, err := encryption.GenerateKey()
	if err != nil {
		return fmt.Errorf("failed to generate encryption key: %v", err)
	}

	// Create chunks directory
	chunksDir := filepath.Join(outputDir, "chunks")
	if err := os.MkdirAll(chunksDir, 0755); err != nil {
		return fmt.Errorf("failed to create chunks directory: %v", err)
	}

	// Split file into chunks
	chunks, err := chunking.SplitFile(inputFile, chunkSize, chunksDir)
	if err != nil {
		return fmt.Errorf("failed to split file: %v", err)
	}

	// Generate unique ID
	id, err := zap.GenerateID()
	if err != nil {
		return fmt.Errorf("failed to generate ID: %v", err)
	}

	// Encrypt chunks and collect metadata
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

		// Create chunk metadata
		chunkMeta := zap.ChunkMetadata{
			Index: chunk.Index,
			Hash:  chunk.Hash,
			Size:  chunk.Size,
		}

		// Generate unique encrypted hash
		if err := chunkMeta.UpdateEncryptedHash(encrypted); err != nil {
			return fmt.Errorf("failed to generate encrypted hash: %v", err)
		}

		// Write encrypted chunk
		encryptedPath := filepath.Join(chunksDir, chunkMeta.EncryptedHash)
		if err := os.WriteFile(encryptedPath, encrypted, 0644); err != nil {
			return fmt.Errorf("failed to write encrypted chunk: %v", err)
		}

		zapChunks = append(zapChunks, chunkMeta)
	}

	// Create zap metadata
	metadata := &zap.FileMetadata{
		ID:            id,
		OriginalName:  filepath.Base(inputFile),
		ChunkCount:    len(chunks),
		TotalSize:     chunkSize * int64(len(chunks)),
		EncryptionKey: key,
		Chunks:        zapChunks,
	}

	// Write zap file
	if err := zap.CreateZapFile(metadata, outputDir); err != nil {
		return fmt.Errorf("failed to create zap file: %v", err)
	}

	fmt.Printf("Successfully split file into %d chunks\n", len(chunks))
	fmt.Printf("ZAP file created: %s.zap\n", id)
	return nil
}

func joinMode(zapFile, outputDir string) error {
	// Read zap file
	metadata, err := zap.ReadZapFile(zapFile)
	if err != nil {
		return fmt.Errorf("failed to read zap file: %v", err)
	}

	// Validate chunks
	chunksDir := filepath.Join(filepath.Dir(zapFile), "chunks")
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

	fmt.Printf("Successfully reassembled file: %s\n", outputPath)
	return nil
}
