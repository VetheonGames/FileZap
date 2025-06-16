package registry

import (
"fmt"
"testing"

"github.com/VetheonGames/FileZap/NetworkCore/pkg/types"
"github.com/stretchr/testify/assert"
)

func TestFileRegistration(t *testing.T) {
fr := NewFileRegistry()

// Create test data
fileInfo := &types.FileInfo{
Name:     "test.txt",
ChunkIDs: []string{"chunk1", "chunk2"},
Peers: []types.PeerChunkInfo{
{
PeerID:    "peer1",
ChunkIDs:  []string{"chunk1"},
Available: true,
},
{
PeerID:    "peer2",
ChunkIDs:  []string{"chunk2"},
Available: true,
},
},
Available: true,
}

// Test registration
err := fr.RegisterFile(fileInfo)
assert.NoError(t, err)

// Test retrieval
retrieved, exists := fr.GetFile("test.txt")
assert.True(t, exists)
assert.Equal(t, fileInfo, retrieved)

// Test chunk mapping
peers := fr.GetChunkPeers("chunk1")
assert.Contains(t, peers, "peer1")
peers = fr.GetChunkPeers("chunk2")
assert.Contains(t, peers, "peer2")

// Test unregistration
fr.UnregisterFile("test.txt")
_, exists = fr.GetFile("test.txt")
assert.False(t, exists)
assert.Empty(t, fr.GetChunkPeers("chunk1"))
assert.Empty(t, fr.GetChunkPeers("chunk2"))
}

func TestPeerRegistration(t *testing.T) {
fr := NewFileRegistry()

// Create test data
peerInfo := &types.PeerChunkInfo{
PeerID:    "peer1",
ChunkIDs:  []string{"chunk1", "chunk2"},
Available: true,
}

// Test registration
fr.RegisterPeer(peerInfo)

// Test chunk mapping
peers := fr.GetChunkPeers("chunk1")
assert.Contains(t, peers, "peer1")
peers = fr.GetChunkPeers("chunk2")
assert.Contains(t, peers, "peer1")

// Test availability updates
success := fr.UpdatePeerAvailability("peer1", false)
assert.True(t, success)
availablePeers := fr.GetAvailablePeers()
assert.Empty(t, availablePeers)

success = fr.UpdatePeerAvailability("peer1", true)
assert.True(t, success)
availablePeers = fr.GetAvailablePeers()
assert.Len(t, availablePeers, 1)

// Test unregistration
fr.UnregisterPeer("peer1")
assert.Empty(t, fr.GetChunkPeers("chunk1"))
assert.Empty(t, fr.GetChunkPeers("chunk2"))
}

func TestFileList(t *testing.T) {
fr := NewFileRegistry()

files := []types.FileInfo{
{
Name:     "file1.txt",
ChunkIDs: []string{"chunk1"},
},
{
Name:     "file2.txt",
ChunkIDs: []string{"chunk2"},
},
}

// Register files
for _, file := range files {
fileCopy := file // Create a copy to avoid slice element address reuse
err := fr.RegisterFile(&fileCopy)
assert.NoError(t, err)
}

// Test listing
listed := fr.ListFiles()
assert.Len(t, listed, len(files))

fileMap := make(map[string]bool)
for _, file := range listed {
fileMap[file.Name] = true
}

for _, file := range files {
assert.True(t, fileMap[file.Name])
}
}

func TestConcurrentOperations(t *testing.T) {
fr := NewFileRegistry()
done := make(chan bool)
numRoutines := 10

// Test concurrent file operations
for i := 0; i < numRoutines; i++ {
go func(i int) {
fileInfo := &types.FileInfo{
Name:     fmt.Sprintf("file%d.txt", i),
ChunkIDs: []string{fmt.Sprintf("chunk%d", i)},
}

// Register
err := fr.RegisterFile(fileInfo)
assert.NoError(t, err)

// Get
retrieved, exists := fr.GetFile(fileInfo.Name)
assert.True(t, exists)
assert.Equal(t, fileInfo, retrieved)

// List
files := fr.ListFiles()
assert.NotEmpty(t, files)

// Unregister
fr.UnregisterFile(fileInfo.Name)
_, exists = fr.GetFile(fileInfo.Name)
assert.False(t, exists)

done <- true
}(i)
}

// Wait for all routines to complete
for i := 0; i < numRoutines; i++ {
<-done
}

// Test concurrent peer operations
for i := 0; i < numRoutines; i++ {
go func(i int) {
peerInfo := &types.PeerChunkInfo{
PeerID:    fmt.Sprintf("peer%d", i),
ChunkIDs:  []string{fmt.Sprintf("chunk%d", i)},
Available: true,
}

// Register
fr.RegisterPeer(peerInfo)

// Update availability
fr.UpdatePeerAvailability(peerInfo.PeerID, false)
peers := fr.GetAvailablePeers()
for _, p := range peers {
assert.NotEqual(t, p.PeerID, peerInfo.PeerID)
}

// Unregister
fr.UnregisterPeer(peerInfo.PeerID)
assert.Empty(t, fr.GetChunkPeers(fmt.Sprintf("chunk%d", i)))

done <- true
}(i)
}

// Wait for all routines to complete
for i := 0; i < numRoutines; i++ {
<-done
}
}

func TestEdgeCases(t *testing.T) {
fr := NewFileRegistry()

// Test registering file with empty name
fileInfo := &types.FileInfo{
Name:     "",
ChunkIDs: []string{"chunk1"},
}
err := fr.RegisterFile(fileInfo)
assert.NoError(t, err) // Should still work as name is just an identifier

// Test registering file with empty chunks
fileInfo = &types.FileInfo{
Name:     "empty_chunks.txt",
ChunkIDs: []string{},
}
err = fr.RegisterFile(fileInfo)
assert.NoError(t, err)

// Test registering file with nil peers
fileInfo = &types.FileInfo{
Name:     "no_peers.txt",
ChunkIDs: []string{"chunk1"},
Peers:    nil,
}
err = fr.RegisterFile(fileInfo)
assert.NoError(t, err)

// Test getting chunk peers for non-existent chunk
peers := fr.GetChunkPeers("nonexistent-chunk")
assert.Empty(t, peers)

// Test updating availability for non-existent peer
success := fr.UpdatePeerAvailability("nonexistent-peer", true)
assert.False(t, success)
}

func TestComplexPeerScenarios(t *testing.T) {
fr := NewFileRegistry()

// Register multiple peers with overlapping chunks
peers := []types.PeerChunkInfo{
{
PeerID:    "peer1",
ChunkIDs:  []string{"chunk1", "chunk2"},
Available: true,
},
{
PeerID:    "peer2",
ChunkIDs:  []string{"chunk2", "chunk3"},
Available: true,
},
{
PeerID:    "peer3",
ChunkIDs:  []string{"chunk1", "chunk3"},
Available: true,
},
}

for _, peer := range peers {
peerCopy := peer
fr.RegisterPeer(&peerCopy)
}

// Test chunk distribution
chunk1Peers := fr.GetChunkPeers("chunk1")
assert.Len(t, chunk1Peers, 2)
assert.Contains(t, chunk1Peers, "peer1")
assert.Contains(t, chunk1Peers, "peer3")

chunk2Peers := fr.GetChunkPeers("chunk2")
assert.Len(t, chunk2Peers, 2)
assert.Contains(t, chunk2Peers, "peer1")
assert.Contains(t, chunk2Peers, "peer2")

// Test availability changes
fr.UpdatePeerAvailability("peer1", false)
fr.UpdatePeerAvailability("peer2", false)
availablePeers := fr.GetAvailablePeers()
assert.Len(t, availablePeers, 1)
assert.Equal(t, "peer3", availablePeers[0].PeerID)

// Test unregistering middle peer
fr.UnregisterPeer("peer2")
chunk2Peers = fr.GetChunkPeers("chunk2")
assert.Len(t, chunk2Peers, 1)
assert.Contains(t, chunk2Peers, "peer1")
}

func TestFileDuplication(t *testing.T) {
fr := NewFileRegistry()

// Register initial file
file1 := &types.FileInfo{
Name:     "test.txt",
ChunkIDs: []string{"chunk1", "chunk2"},
Peers: []types.PeerChunkInfo{
{
PeerID:    "peer1",
ChunkIDs:  []string{"chunk1"},
Available: true,
},
},
}
err := fr.RegisterFile(file1)
assert.NoError(t, err)

// Register same file with different chunks/peers
file2 := &types.FileInfo{
Name:     "test.txt",
ChunkIDs: []string{"chunk2", "chunk3"},
Peers: []types.PeerChunkInfo{
{
PeerID:    "peer2",
ChunkIDs:  []string{"chunk2", "chunk3"},
Available: true,
},
},
}
err = fr.RegisterFile(file2)
assert.NoError(t, err)

// Verify the second registration overwrote the first
retrieved, exists := fr.GetFile("test.txt")
assert.True(t, exists)
assert.Equal(t, file2.ChunkIDs, retrieved.ChunkIDs)
assert.Len(t, retrieved.Peers, 1)
assert.Equal(t, "peer2", retrieved.Peers[0].PeerID)

// Verify chunk mappings were updated
chunk1Peers := fr.GetChunkPeers("chunk1")
assert.Empty(t, chunk1Peers)

chunk2Peers := fr.GetChunkPeers("chunk2")
assert.Contains(t, chunk2Peers, "peer2")

chunk3Peers := fr.GetChunkPeers("chunk3")
assert.Contains(t, chunk3Peers, "peer2")
}

func TestHelperFunctions(t *testing.T) {
// Test contains
slice := []string{"a", "b", "c"}
assert.True(t, contains(slice, "b"))
assert.False(t, contains(slice, "d"))

// Test appendUnique
slice = appendUnique(slice, "d")
assert.Contains(t, slice, "d")
assert.Len(t, slice, 4)

// Test duplicate appendUnique
slice = appendUnique(slice, "d")
assert.Len(t, slice, 4)

// Test remove
slice = remove(slice, "b")
assert.NotContains(t, slice, "b")
assert.Len(t, slice, 3)

// Test remove non-existent
slice = remove(slice, "x")
assert.Len(t, slice, 3)
}
