package chunking

import (
	"crypto/rand"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestFile creates a temporary file with random data for testing
func createTestFile(t *testing.T, size int64) string {
	tempFile, err := os.CreateTemp("", "test_file_*.dat")
	require.NoError(t, err)
	defer tempFile.Close()

	// Generate random data
	data := make([]byte, size)
	_, err = rand.Read(data)
	require.NoError(t, err)

	// Write data to file
	_, err = tempFile.Write(data)
	require.NoError(t, err)

	return tempFile.Name()
}

func TestSplitFile(t *testing.T) {
	// Create temp directory for chunks
	tempDir, err := os.MkdirTemp("", "chunks_*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create test file (5MB)
	fileSize := int64(5 * 1024 * 1024)
	testFile := createTestFile(t, fileSize)
	defer os.Remove(testFile)

	// Test cases with different chunk sizes
	testCases := []struct {
		name        string
		chunkSize   int64
		expectedNum int
	}{
		{"1MB chunks", 1024 * 1024, 5},
		{"2MB chunks", 2 * 1024 * 1024, 3},
		{"5MB chunks", 5 * 1024 * 1024, 1},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Split file into chunks
			chunks, err := SplitFile(testFile, tc.chunkSize, tempDir)
			require.NoError(t, err)

			// Verify number of chunks
			assert.Equal(t, tc.expectedNum, len(chunks))

			// Verify each chunk
			for _, chunk := range chunks {
				// Check file exists
				_, err := os.Stat(chunk.Filename)
				assert.NoError(t, err)

				// Check chunk size
				info, err := os.Stat(chunk.Filename)
				require.NoError(t, err)

				// Last chunk might be smaller
				if chunk.Index < len(chunks)-1 {
					assert.Equal(t, tc.chunkSize, info.Size())
				} else {
					assert.LessOrEqual(t, info.Size(), tc.chunkSize)
				}

				// Verify hash is not empty
				assert.NotEmpty(t, chunk.Hash)
			}
		})
	}
}

func TestReassembleFile(t *testing.T) {
	// Create temp directories
	tempDir, err := os.MkdirTemp("", "chunks_*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	outputDir, err := os.MkdirTemp("", "output_*")
	require.NoError(t, err)
	defer os.RemoveAll(outputDir)

	// Create test file (3MB)
	fileSize := int64(3 * 1024 * 1024)
	testFile := createTestFile(t, fileSize)
	defer os.Remove(testFile)

	// Read original file content
	originalData, err := os.ReadFile(testFile)
	require.NoError(t, err)

	// Split into chunks
	chunkSize := int64(1024 * 1024) // 1MB chunks
	chunks, err := SplitFile(testFile, chunkSize, tempDir)
	require.NoError(t, err)

	// Reassemble file
	outputPath := filepath.Join(outputDir, "reassembled.dat")
	err = ReassembleFile(chunks, outputPath)
	require.NoError(t, err)

	// Read reassembled file
	reassembledData, err := os.ReadFile(outputPath)
	require.NoError(t, err)

	// Compare contents
	assert.Equal(t, originalData, reassembledData)
}

func TestSplitFileErrors(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "chunks_*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	t.Run("nonexistent file", func(t *testing.T) {
		_, err := SplitFile("nonexistent.file", DefaultChunkSize, tempDir)
		assert.Error(t, err)
	})

	t.Run("invalid chunk size", func(t *testing.T) {
		testFile := createTestFile(t, 1024)
		defer os.Remove(testFile)

		_, err := SplitFile(testFile, 0, tempDir)
		assert.Error(t, err)
	})

	t.Run("invalid output directory", func(t *testing.T) {
		testFile := createTestFile(t, 1024)
		defer os.Remove(testFile)

		_, err := SplitFile(testFile, DefaultChunkSize, "/nonexistent/directory")
		assert.Error(t, err)
	})
}

func TestReassembleFileErrors(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "output_*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	t.Run("missing chunks", func(t *testing.T) {
		chunks := []ChunkInfo{
			{Index: 0, Hash: "hash1", Filename: "nonexistent.chunk"},
		}
		err := ReassembleFile(chunks, filepath.Join(tempDir, "output.dat"))
		assert.Error(t, err)
	})

	t.Run("invalid output path", func(t *testing.T) {
		chunks := []ChunkInfo{}
		err := ReassembleFile(chunks, "/nonexistent/directory/output.dat")
		assert.Error(t, err)
	})

	t.Run("empty chunk list", func(t *testing.T) {
		err := ReassembleFile([]ChunkInfo{}, filepath.Join(tempDir, "output.dat"))
		assert.Error(t, err)
	})
}
