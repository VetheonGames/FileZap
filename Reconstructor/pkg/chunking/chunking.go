package chunking

import (
"fmt"
"os"
"path/filepath"
"sort"
"strings"
)

// ChunkInfo represents metadata about a chunk
type ChunkInfo struct {
	Index    int    `json:"index"`
	Hash     string `json:"hash"`
	Size     int64  `json:"size"`
	Filename string `json:"filename"`
}

// ReassembleFile reassembles chunks back into the original file with enhanced validation
func ReassembleFile(chunks []ChunkInfo, outputPath string) error {
	// Validate chunks are present
	if len(chunks) == 0 {
		return fmt.Errorf("no chunks provided for reassembly")
	}

	// Sort chunks by index to ensure correct order
	sort.Slice(chunks, func(i, j int) bool {
		return chunks[i].Index < chunks[j].Index
	})

	// Validate chunk indexes are sequential
	for i, chunk := range chunks {
		if chunk.Index != i {
			return fmt.Errorf("non-sequential chunk index detected: expected %d, got %d", i, chunk.Index)
		}
	}

// Check if path is valid
if strings.HasPrefix(outputPath, "/") || // Unix absolute path
   (len(outputPath) > 2 && outputPath[1] == ':') { // Windows absolute path
    if !isWithinDirectory(outputPath, os.Getenv("USERPROFILE")) {
        return fmt.Errorf("invalid output path: must be within user directory")
    }
}

// Ensure output directory exists
outputDir := filepath.Dir(outputPath)
if err := os.MkdirAll(outputDir, 0755); err != nil {
    return fmt.Errorf("failed to create output directory: %v", err)
}

// Create output file
outFile, err := os.Create(outputPath)
if err != nil {
    return fmt.Errorf("failed to create output file: %v", err)
}
defer outFile.Close()

	// Process each chunk
	var processedSize int64
	for _, chunk := range chunks {
		// Validate chunk file exists
		if _, err := os.Stat(chunk.Filename); err != nil {
			return fmt.Errorf("chunk file not found or inaccessible: %s", chunk.Filename)
		}

		// Read chunk data
		chunkData, err := os.ReadFile(chunk.Filename)
		if err != nil {
			return fmt.Errorf("failed to read chunk %d: %v", chunk.Index, err)
		}

		// Validate chunk size
		if int64(len(chunkData)) != chunk.Size {
			return fmt.Errorf("chunk %d size mismatch: expected %d, got %d",
				chunk.Index, chunk.Size, len(chunkData))
		}

		// Write chunk to output file
		if _, err := outFile.Write(chunkData); err != nil {
			return fmt.Errorf("failed to write chunk %d: %v", chunk.Index, err)
		}

		processedSize += chunk.Size
	}

	// Verify final file size matches expected total
	finalInfo, err := outFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to get output file info: %v", err)
	}

	if finalInfo.Size() != processedSize {
		return fmt.Errorf("final file size mismatch: expected %d, got %d",
			processedSize, finalInfo.Size())
	}

	return nil
}

// CleanupTempFiles removes temporary decrypted chunk files
func CleanupTempFiles(chunks []ChunkInfo) {
	for _, chunk := range chunks {
		// Ignore errors since these are temporary files
		_ = os.Remove(chunk.Filename)
	}

// Try to remove the parent temp directory if it's empty
if len(chunks) > 0 {
_ = os.Remove(filepath.Dir(chunks[0].Filename))
}
}

// isWithinDirectory checks if a path is within a given directory
func isWithinDirectory(path, dir string) bool {
    // Clean and make absolute paths for comparison
    absPath, err := filepath.Abs(path)
    if err != nil {
        return false
    }
    absDir, err := filepath.Abs(dir)
    if err != nil {
        return false
    }

    // Use filepath.Rel to check if the path is within the directory
    rel, err := filepath.Rel(absDir, absPath)
    if err != nil {
        return false
    }

    // Check if the relative path tries to escape with ".."
    return !filepath.IsAbs(rel) && !strings.HasPrefix(rel, "..")
}
