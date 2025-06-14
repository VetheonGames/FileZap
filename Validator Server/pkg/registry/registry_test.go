package registry

import (
    "fmt"
    "os"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func setupTestRegistry(t *testing.T) (*Registry, string, func()) {
    // Create temporary directory
    tempDir, err := os.MkdirTemp("", "registry_test_*")
    require.NoError(t, err)

    // Create registry
    reg, err := NewRegistry(tempDir)
    require.NoError(t, err)

    // Return cleanup function
    cleanup := func() {
        os.RemoveAll(tempDir)
    }

    return reg, tempDir, cleanup
}

func TestFileRegistration(t *testing.T) {
    reg, _, cleanup := setupTestRegistry(t)
    defer cleanup()

    // Create test file info
    fileInfo := &FileInfo{
        ID:          "testhash",
        Name:        "test.txt",
        ChunkCount:  2,
        PeerIDs:     []string{"peer1"},
        TotalSize:   1024,
        ZapMetadata: []byte("test metadata"),
    }

    // Test registration
    t.Run("register file", func(t *testing.T) {
        err := reg.RegisterFile(fileInfo)
        require.NoError(t, err)

        // Verify file exists
        stored, exists := reg.GetFileByName("test.txt")
        assert.True(t, exists)
        assert.Equal(t, fileInfo.Name, stored.Name)
        assert.Equal(t, fileInfo.TotalSize, stored.TotalSize)
        assert.Equal(t, fileInfo.ID, stored.ID)
    })

    // Test duplicate registration
    t.Run("duplicate registration", func(t *testing.T) {
        err := reg.RegisterFile(fileInfo)
        assert.NoError(t, err) // Should update existing file
    })

    // Test invalid file info
    t.Run("invalid file info", func(t *testing.T) {
        err := reg.RegisterFile(&FileInfo{}) // Empty file info
        assert.Error(t, err)
    })
}

func TestPeerOperations(t *testing.T) {
    reg, _, cleanup := setupTestRegistry(t)
    defer cleanup()

    // Create test file info
    fileInfo := &FileInfo{
        ID:          "testhash",
        Name:        "test.txt",
        ChunkCount:  2,
        PeerIDs:     []string{},
        TotalSize:   1024,
        ZapMetadata: []byte("test metadata"),
    }

    err := reg.RegisterFile(fileInfo)
    require.NoError(t, err)

    // Test adding peer to file
    t.Run("add peer to file", func(t *testing.T) {
        err := reg.AddPeerToFile("test.txt", "peer1")
        require.NoError(t, err)

        stored, exists := reg.GetFileByName("test.txt")
        assert.True(t, exists)
        assert.Contains(t, stored.PeerIDs, "peer1")
    })

    // Test adding same peer again
    t.Run("add duplicate peer", func(t *testing.T) {
        err := reg.AddPeerToFile("test.txt", "peer1")
        assert.NoError(t, err)

        stored, exists := reg.GetFileByName("test.txt")
        assert.True(t, exists)
        assert.Equal(t, 1, len(stored.PeerIDs)) // Should not duplicate
    })

    // Test adding peer to nonexistent file
    t.Run("add peer to nonexistent file", func(t *testing.T) {
        err := reg.AddPeerToFile("nonexistent.txt", "peer1")
        assert.Error(t, err)
    })
}

func TestChunkOperations(t *testing.T) {
    reg, _, cleanup := setupTestRegistry(t)
    defer cleanup()

    // Test chunk registration
    t.Run("register chunks", func(t *testing.T) {
        reg.RegisterPeerChunks("peer1", "localhost:8081", []string{"chunk1", "chunk2"})

        // Verify chunks are registered
        peers := reg.GetPeersForChunk("chunk1")
        assert.Contains(t, peers, "peer1")
        peers = reg.GetPeersForChunk("chunk2")
        assert.Contains(t, peers, "peer1")
    })

    // Test getting peers for nonexistent chunk
    t.Run("get peers for nonexistent chunk", func(t *testing.T) {
        peers := reg.GetPeersForChunk("nonexistent")
        assert.Empty(t, peers)
    })

    // Test updating peer chunks
    t.Run("update peer chunks", func(t *testing.T) {
        // Register new chunks for same peer
        reg.RegisterPeerChunks("peer1", "localhost:8081", []string{"chunk3"})

        // Verify all chunks
        peers := reg.GetPeersForChunk("chunk1")
        assert.Contains(t, peers, "peer1")
        peers = reg.GetPeersForChunk("chunk3")
        assert.Contains(t, peers, "peer1")
    })
}

func TestPersistence(t *testing.T) {
    reg, dataDir, cleanup := setupTestRegistry(t)
    defer cleanup()

    // Create and register a file
    fileInfo := &FileInfo{
        ID:          "testhash",
        Name:        "test.txt",
        ChunkCount:  2,
        PeerIDs:     []string{"peer1"},
        TotalSize:   1024,
        ZapMetadata: []byte("test metadata"),
    }

    err := reg.RegisterFile(fileInfo)
    require.NoError(t, err)

    // Register some chunks
    reg.RegisterPeerChunks("peer1", "localhost:8081", []string{"chunk1", "chunk2"})

    // Create new registry instance with same data directory
    reg2, err := NewRegistry(dataDir)
    require.NoError(t, err)

    // Verify file data persisted
    t.Run("file persistence", func(t *testing.T) {
        stored, exists := reg2.GetFileByName("test.txt")
        assert.True(t, exists)
        assert.Equal(t, fileInfo.Name, stored.Name)
        assert.Equal(t, fileInfo.TotalSize, stored.TotalSize)
        assert.Equal(t, fileInfo.ID, stored.ID)
        assert.Contains(t, stored.PeerIDs, "peer1")
    })

    // Verify chunk data persisted
    t.Run("chunk persistence", func(t *testing.T) {
        peers := reg2.GetPeersForChunk("chunk1")
        assert.Contains(t, peers, "peer1")
        peers = reg2.GetPeersForChunk("chunk2")
        assert.Contains(t, peers, "peer1")
    })
}

func TestConcurrentOperations(t *testing.T) {
    reg, _, cleanup := setupTestRegistry(t)
    defer cleanup()

    // Create test file
    fileInfo := &FileInfo{
        ID:          "testhash",
        Name:        "test.txt",
        ChunkCount:  2,
        PeerIDs:     []string{},
        TotalSize:   1024,
        ZapMetadata: []byte("test metadata"),
    }

    err := reg.RegisterFile(fileInfo)
    require.NoError(t, err)

    // Test concurrent peer additions
    t.Run("concurrent peer additions", func(t *testing.T) {
        done := make(chan bool)
        for i := 0; i < 10; i++ {
            go func(id int) {
                peerID := fmt.Sprintf("peer%d", id)
                err := reg.AddPeerToFile("test.txt", peerID)
                assert.NoError(t, err)
                done <- true
            }(i)
        }

        // Wait for all goroutines
        for i := 0; i < 10; i++ {
            <-done
        }

        // Verify all peers were added
        stored, exists := reg.GetFileByName("test.txt")
        assert.True(t, exists)
        assert.Equal(t, 10, len(stored.PeerIDs))
    })

    // Test concurrent chunk registrations
    t.Run("concurrent chunk registrations", func(t *testing.T) {
        done := make(chan bool)
        for i := 0; i < 10; i++ {
            go func(id int) {
                peerID := fmt.Sprintf("peer%d", id)
                chunks := []string{
                    fmt.Sprintf("chunk%d_1", id),
                    fmt.Sprintf("chunk%d_2", id),
                }
                reg.RegisterPeerChunks(peerID, fmt.Sprintf("localhost:%d", 8081+id), chunks)
                done <- true
            }(i)
        }

        // Wait for all goroutines
        for i := 0; i < 10; i++ {
            <-done
        }

        // Verify chunks were registered
        for i := 0; i < 10; i++ {
            peers := reg.GetPeersForChunk(fmt.Sprintf("chunk%d_1", i))
            assert.Contains(t, peers, fmt.Sprintf("peer%d", i))
        }
    })
}
