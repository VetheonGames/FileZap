package chunking

import (
    "bytes"
    "crypto/rand"
    "encoding/hex"
    "fmt"
    "os"
    "path/filepath"
    "sort"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

// createTestChunks creates a set of random chunks for testing
func createTestChunks(t *testing.T, tempDir string, numChunks int, chunkSize int64) ([]ChunkInfo, []byte) {
    var chunks []ChunkInfo
    var fullData []byte

    for i := 0; i < numChunks; i++ {
        // Generate random chunk data
        chunkData := make([]byte, chunkSize)
        _, err := rand.Read(chunkData)
        require.NoError(t, err)

        // Add to full data
        fullData = append(fullData, chunkData...)

        // Generate hash for chunk
        hash := make([]byte, 16)
        _, err = rand.Read(hash)
        require.NoError(t, err)
        hashStr := hex.EncodeToString(hash)

        // Write chunk to temp file
        chunkPath := filepath.Join(tempDir, hashStr)
        err = os.WriteFile(chunkPath, chunkData, 0644)
        require.NoError(t, err)

        chunks = append(chunks, ChunkInfo{
            Index:    i,
            Hash:     hashStr,
            Size:     chunkSize,
            Filename: chunkPath,
        })
    }

    return chunks, fullData
}

func TestReassembleFile(t *testing.T) {
    // Create temporary directories
    tempDir, err := os.MkdirTemp("", "chunks_*")
    require.NoError(t, err)
    defer os.RemoveAll(tempDir)

    outputDir, err := os.MkdirTemp("", "output_*")
    require.NoError(t, err)
    defer os.RemoveAll(outputDir)

    testCases := []struct {
        name      string
        numChunks int
        chunkSize int64
    }{
        {"single chunk", 1, 1024},
        {"small chunks", 5, 1024},
        {"large chunks", 3, 1024 * 1024},
        {"many small chunks", 20, 512},
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            // Create test chunks and record original data
            chunks, originalData := createTestChunks(t, tempDir, tc.numChunks, tc.chunkSize)

            // Shuffle chunks to ensure order doesn't matter
            sort.Slice(chunks, func(i, j int) bool {
                return chunks[i].Index > chunks[j].Index // Reverse order
            })

            // Reassemble file
            outputPath := filepath.Join(outputDir, "reassembled.dat")
            err := ReassembleFile(chunks, outputPath)
            require.NoError(t, err)

            // Read reassembled file
            reassembledData, err := os.ReadFile(outputPath)
            require.NoError(t, err)

            // Compare contents
            assert.Equal(t, originalData, reassembledData)

            // Cleanup test file
            os.Remove(outputPath)
        })
    }
}

func TestCleanupTempFiles(t *testing.T) {
    // Create temporary directory
    tempDir, err := os.MkdirTemp("", "cleanup_test_*")
    require.NoError(t, err)
    defer os.RemoveAll(tempDir)

    // Create test chunks
    chunks, _ := createTestChunks(t, tempDir, 3, 1024)

    // Verify files exist
    for _, chunk := range chunks {
        _, err := os.Stat(chunk.Filename)
        assert.NoError(t, err)
    }

    // Clean up files
    CleanupTempFiles(chunks)

    // Verify files are removed
    for _, chunk := range chunks {
        _, err := os.Stat(chunk.Filename)
        assert.True(t, os.IsNotExist(err))
    }
}

func TestReassembleFileErrors(t *testing.T) {
    tempDir, err := os.MkdirTemp("", "error_test_*")
    require.NoError(t, err)
    defer os.RemoveAll(tempDir)

    outputDir, err := os.MkdirTemp("", "output_*")
    require.NoError(t, err)
    defer os.RemoveAll(outputDir)

    t.Run("empty chunk list", func(t *testing.T) {
        err := ReassembleFile([]ChunkInfo{}, filepath.Join(outputDir, "test.dat"))
        assert.Error(t, err)
    })

    t.Run("missing chunk files", func(t *testing.T) {
        chunks := []ChunkInfo{
            {
                Index:    0,
                Hash:     "nonexistent",
                Size:     1024,
                Filename: filepath.Join(tempDir, "nonexistent"),
            },
        }
        err := ReassembleFile(chunks, filepath.Join(outputDir, "test.dat"))
        assert.Error(t, err)
    })

    t.Run("invalid output path", func(t *testing.T) {
        chunks, _ := createTestChunks(t, tempDir, 1, 1024)
        err := ReassembleFile(chunks, "/nonexistent/directory/test.dat")
        assert.Error(t, err)
    })

    t.Run("duplicate chunk indices", func(t *testing.T) {
        // Create two chunks with same index
        chunks, _ := createTestChunks(t, tempDir, 2, 1024)
        chunks[1].Index = chunks[0].Index

        err := ReassembleFile(chunks, filepath.Join(outputDir, "test.dat"))
        assert.Error(t, err)
    })

    t.Run("missing chunk indices", func(t *testing.T) {
        chunks, _ := createTestChunks(t, tempDir, 2, 1024)
        chunks[1].Index = chunks[0].Index + 2 // Skip an index

        err := ReassembleFile(chunks, filepath.Join(outputDir, "test.dat"))
        assert.Error(t, err)
    })
}

func TestChunkOrderIndependence(t *testing.T) {
    // Create temporary directories
    tempDir, err := os.MkdirTemp("", "chunks_*")
    require.NoError(t, err)
    defer os.RemoveAll(tempDir)

    outputDir, err := os.MkdirTemp("", "output_*")
    require.NoError(t, err)
    defer os.RemoveAll(outputDir)

    // Create test chunks
    chunks, originalData := createTestChunks(t, tempDir, 5, 1024)

    // Try different chunk orderings
    orderings := [][]int{
        {0, 1, 2, 3, 4},
        {4, 3, 2, 1, 0},
        {2, 0, 4, 1, 3},
        {1, 3, 0, 4, 2},
    }

    for i, ordering := range orderings {
        t.Run(fmt.Sprintf("ordering_%d", i), func(t *testing.T) {
            // Reorder chunks
            reorderedChunks := make([]ChunkInfo, len(chunks))
            for j, idx := range ordering {
                reorderedChunks[j] = chunks[idx]
            }

            // Reassemble file
            outputPath := filepath.Join(outputDir, fmt.Sprintf("test_%d.dat", i))
            err := ReassembleFile(reorderedChunks, outputPath)
            require.NoError(t, err)

            // Read reassembled file
            reassembledData, err := os.ReadFile(outputPath)
            require.NoError(t, err)

            // Compare with original
            assert.True(t, bytes.Equal(originalData, reassembledData))
        })
    }
}
