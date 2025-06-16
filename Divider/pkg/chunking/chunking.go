package chunking

import (
"crypto/sha256"
"encoding/hex"
"fmt"
"io"
"os"
"path/filepath"
)

const DefaultChunkSize = 1024 * 1024 // 1MB default chunk size

// ChunkInfo represents metadata about a chunk
type ChunkInfo struct {
	Index    int    `json:"index"`
	Hash     string `json:"hash"`
	Size     int64  `json:"size"`
	Filename string `json:"filename"`
}

// SplitFile splits a file into chunks of specified size
func SplitFile(inputPath string, chunkSize int64, outputDir string) ([]ChunkInfo, error) {
    // Validate chunk size
    if chunkSize <= 0 {
        return nil, fmt.Errorf("invalid chunk size: must be greater than 0")
    }

    // Check if the path is valid for the current OS
    if filepath.VolumeName(outputDir) == "" && (len(outputDir) > 0 && (outputDir[0] == '/' || outputDir[0] == '\\')) {
        return nil, fmt.Errorf("invalid output directory path: must be a valid OS-specific path")
    }

    // Get absolute path and validate the output directory
    absOutputDir, err := filepath.Abs(outputDir)
    if err != nil {
        return nil, fmt.Errorf("invalid output directory path: %v", err)
    }
    
    // Check if output directory exists and is accessible
    dirInfo, err := os.Stat(absOutputDir)
    if err != nil {
        return nil, fmt.Errorf("invalid output directory: %v", err)
    }
    
    // Ensure it's actually a directory
    if !dirInfo.IsDir() {
        return nil, fmt.Errorf("output path is not a directory")
    }

    // Use the absolute path for all operations
    outputDir = absOutputDir

    // Validate output directory is writable by trying to create a test file
    testFile := filepath.Join(outputDir, ".test")
    if err := os.WriteFile(testFile, []byte{}, 0644); err != nil {
        return nil, fmt.Errorf("output directory not writable: %v", err)
    }
    os.Remove(testFile) // Clean up test file

    file, err := os.Open(inputPath)
    if err != nil {
        return nil, err
    }
    defer file.Close()

	var chunks []ChunkInfo
	var index int
	for {
		buffer := make([]byte, chunkSize)
		bytesRead, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			return nil, err
		}
		if bytesRead == 0 {
			break
		}

		// Trim buffer to actual bytes read
		buffer = buffer[:bytesRead]

		// Calculate hash
		hash := sha256.Sum256(buffer)
		hashString := hex.EncodeToString(hash[:])

		// Create chunk filename
		chunkFilename := filepath.Join(outputDir, hashString)

		// Write chunk to file
		if err := os.WriteFile(chunkFilename, buffer, 0644); err != nil {
			return nil, err
		}

		chunks = append(chunks, ChunkInfo{
			Index:    index,
			Hash:     hashString,
			Size:     int64(bytesRead),
			Filename: chunkFilename,
		})

		index++
		if err == io.EOF {
			break
		}
	}

	return chunks, nil
}

// ReassembleFile reassembles chunks back into the original file
func ReassembleFile(chunks []ChunkInfo, outputPath string) error {
    // Validate chunk list
    if len(chunks) == 0 {
        return fmt.Errorf("empty chunk list")
    }

    // Ensure output directory exists
    outputDir := filepath.Dir(outputPath)
    if err := os.MkdirAll(outputDir, 0755); err != nil {
        return fmt.Errorf("failed to create output directory: %w", err)
    }

    outFile, err := os.Create(outputPath)
    if err != nil {
        return err
    }
    defer outFile.Close()

	for _, chunk := range chunks {
		chunkData, err := os.ReadFile(chunk.Filename)
		if err != nil {
			return err
		}

		// Verify hash
		hash := sha256.Sum256(chunkData)
		if hex.EncodeToString(hash[:]) != chunk.Hash {
			return err
		}

		if _, err := outFile.Write(chunkData); err != nil {
			return err
		}
	}

	return nil
}
