package operations

import (
"encoding/json"
"os"
"path/filepath"
"strings"
"testing"
"time"

"github.com/VetheonGames/FileZap/Client/pkg/server"
"github.com/stretchr/testify/assert"
"github.com/stretchr/testify/require"
)

// mockServer implements ServerInterface for testing
type mockServer struct {
files     map[string]*server.FileInfo
failFetch bool
}

func newMockServer() ServerInterface {
return &mockServer{
files:     make(map[string]*server.FileInfo),
failFetch: false,
}
}

func (m *mockServer) GetPeersWithFile(fileID string) []string {
if _, exists := m.files[fileID]; exists {
return []string{"mock-peer-1"}
}
return nil
}

func (m *mockServer) RegisterFile(info *server.FileInfo) error {
m.files[info.ID] = info
return nil
}

func (m *mockServer) FetchChunks(fileInfo *server.FileInfo, peerID string) error {
if m.failFetch {
return assert.AnError
}
return nil
}

func TestFileOperations_SplitFile(t *testing.T) {
// Create temporary directories for testing
testDir, err := os.MkdirTemp("", "filezap-test-*")
require.NoError(t, err)
defer os.RemoveAll(testDir)

chunkDir := filepath.Join(testDir, "chunks")
require.NoError(t, os.MkdirAll(chunkDir, 0755))

// Create a test file
testFile := filepath.Join(testDir, "test.txt")
testData := "This is test data for FileZap testing. It will be split into chunks."
err = os.WriteFile(testFile, []byte(testData), 0644)
require.NoError(t, err)

// Initialize FileOperations with mock server
fileOps := NewFileOperations(newMockServer())

// Test file splitting
err = fileOps.SplitFile(testFile, chunkDir, "16")
require.NoError(t, err)

// Verify manifest file was created
manifestPath := filepath.Join(chunkDir, "test.txt.zap")
assert.FileExists(t, manifestPath)

// Load and verify manifest
manifestData, err := os.ReadFile(manifestPath)
require.NoError(t, err)

var fileInfo server.FileInfo
err = json.Unmarshal(manifestData, &fileInfo)
require.NoError(t, err)

// Verify file info
assert.Equal(t, "test.txt", fileInfo.Name)
assert.Equal(t, int64(len(testData)), fileInfo.TotalSize)
assert.NotEmpty(t, fileInfo.Chunks)

// Verify chunks were created
for _, chunk := range fileInfo.Chunks {
chunkPath := filepath.Join(chunkDir, chunk.ID)
assert.FileExists(t, chunkPath)
}
}

func TestFileOperations_JoinFile(t *testing.T) {
// Create temporary directories for testing
testDir, err := os.MkdirTemp("", "filezap-test-*")
require.NoError(t, err)
defer os.RemoveAll(testDir)

chunkDir := filepath.Join(testDir, "chunks")
outputDir := filepath.Join(testDir, "output")
require.NoError(t, os.MkdirAll(chunkDir, 0755))
require.NoError(t, os.MkdirAll(outputDir, 0755))

// Create test data and split it
testFile := filepath.Join(testDir, "test.txt")
testData := "This is test data for FileZap testing. It will be split and then joined back together."
err = os.WriteFile(testFile, []byte(testData), 0644)
require.NoError(t, err)

// Initialize FileOperations with mock server
mockSrv := &mockServer{
files:     make(map[string]*server.FileInfo),
failFetch: false,
}
fileOps := NewFileOperations(mockSrv)

// Split the file
err = fileOps.SplitFile(testFile, chunkDir, "16")
require.NoError(t, err)

// Get manifest path
manifestPath := filepath.Join(chunkDir, "test.txt.zap")

// Test joining the file
err = fileOps.JoinFile(manifestPath, outputDir)
require.NoError(t, err)

// Verify joined file
joinedFile := filepath.Join(outputDir, "test.txt")
assert.FileExists(t, joinedFile)

// Compare content
joinedData, err := os.ReadFile(joinedFile)
require.NoError(t, err)
assert.Equal(t, testData, string(joinedData))

// Test network failure case
mockSrv.failFetch = true
err = fileOps.JoinFile(manifestPath, outputDir)
assert.Error(t, err)
assert.Contains(t, err.Error(), "failed to fetch chunks")
}

func TestFileOperations_ErrorHandling(t *testing.T) {
// Create temporary test file
testDir, err := os.MkdirTemp("", "filezap-test-*")
require.NoError(t, err)
defer os.RemoveAll(testDir)

testFile := filepath.Join(testDir, "test.txt")
testData := "This is test data for FileZap testing."
err = os.WriteFile(testFile, []byte(testData), 0644)
require.NoError(t, err)

fileOps := NewFileOperations(newMockServer())

tests := []struct {
name        string
inputPath   string
outputPath  string
chunkSize   string
expectError bool
errorMsg    string
}{
{
name:        "NonexistentInputFile",
inputPath:   "nonexistent.txt",
outputPath:  "output",
chunkSize:   "16",
expectError: true,
errorMsg:    "failed to open input file",
},
{
name:        "InvalidChunkSize",
inputPath:   testFile,
outputPath:  "output",
chunkSize:   "invalid",
expectError: true,
errorMsg:    "invalid chunk size",
},
{
name:        "NegativeChunkSize",
inputPath:   testFile,
outputPath:  "output",
chunkSize:   "-16",
expectError: true,
errorMsg:    "invalid chunk size",
},
{
name:        "ZeroChunkSize",
inputPath:   testFile,
outputPath:  "output",
chunkSize:   "0",
expectError: true,
errorMsg:    "invalid chunk size",
},
{
name:        "InvalidOutputPath",
inputPath:   "test.txt",
outputPath:  string([]byte{0}),
chunkSize:   "16",
expectError: true,
errorMsg:    "failed to create output directory",
},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
err := fileOps.SplitFile(tt.inputPath, tt.outputPath, tt.chunkSize)
if tt.expectError {
assert.Error(t, err)
assert.Contains(t, err.Error(), tt.errorMsg)
} else {
assert.NoError(t, err)
}
})
}
}

func TestManifestOperations(t *testing.T) {
testDir, err := os.MkdirTemp("", "filezap-test-*")
require.NoError(t, err)
defer os.RemoveAll(testDir)

// Create a test manifest
testInfo := &server.FileInfo{
Name:      "test.txt",
ID:        "test-id",
ChunkDir:  testDir,
TotalSize: 100,
Chunks: []server.ChunkInfo{
{
ID:    "chunk1",
Size:  50,
Hash:  "hash1",
Index: 0,
},
{
ID:    "chunk2",
Size:  50,
Hash:  "hash2",
Index: 1,
},
},
}

manifestPath := filepath.Join(testDir, "test.txt.zap")

// Test saving manifest
err = saveManifest(manifestPath, testInfo)
require.NoError(t, err)
assert.FileExists(t, manifestPath)

// Test loading manifest
loadedInfo, err := loadManifest(manifestPath)
require.NoError(t, err)
assert.Equal(t, testInfo.Name, loadedInfo.Name)
assert.Equal(t, testInfo.ID, loadedInfo.ID)
assert.Equal(t, testInfo.TotalSize, loadedInfo.TotalSize)
assert.Equal(t, len(testInfo.Chunks), len(loadedInfo.Chunks))
assert.Equal(t, testInfo.Chunks[0].ID, loadedInfo.Chunks[0].ID)
assert.Equal(t, testInfo.Chunks[1].ID, loadedInfo.Chunks[1].ID)
}

func TestHelperFunctions(t *testing.T) {
t.Run("generateFileID", func(t *testing.T) {
// Test that same path at different times produces different IDs
path := "test/file.txt"
id1 := generateFileID(path)
time.Sleep(time.Nanosecond)
id2 := generateFileID(path)
assert.NotEqual(t, id1, id2, "FileIDs should be unique even for same path")

// Test that IDs are valid hex strings
assert.True(t, isValidHexString(id1), "FileID should be a valid hex string")
})

t.Run("generateChunkID", func(t *testing.T) {
// Test that same data produces same chunk ID
data := []byte("test chunk data")
id1 := generateChunkID(data)
id2 := generateChunkID(data)
assert.Equal(t, id1, id2, "Same data should produce same chunk ID")

// Test that different data produces different chunk ID
data2 := []byte("different chunk data")
id3 := generateChunkID(data2)
assert.NotEqual(t, id1, id3, "Different data should produce different chunk IDs")

// Test that IDs are valid hex strings
assert.True(t, isValidHexString(id1), "ChunkID should be a valid hex string")
})
}

// Helper function to validate hex strings
func isValidHexString(s string) bool {
return len(s) == 64 && strings.Trim(s, "0123456789abcdef") == ""
}
