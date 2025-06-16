package peer

import (
"fmt"
"testing"
"time"

"github.com/libp2p/go-libp2p/core/peer"
"github.com/multiformats/go-multiaddr"
"github.com/stretchr/testify/assert"
)

func createTestID(i int) peer.ID {
return peer.ID([]byte{byte(i)})
}

func createTestAddrs(port int) []multiaddr.Multiaddr {
addr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", port))
return []multiaddr.Multiaddr{addr}
}

func TestPeerManagerCreation(t *testing.T) {
pm := NewPeerManager()
assert.NotNil(t, pm)

// Verify default limits
assert.Equal(t, 100, pm.limits.maxPeers)
assert.Equal(t, 1000, pm.limits.maxChunks)
assert.Equal(t, int64(100*1024*1024), pm.limits.maxChunkSize)

// Test limit changes
pm.SetLimits(50, 500, 50*1024*1024)
assert.Equal(t, 50, pm.limits.maxPeers)
assert.Equal(t, 500, pm.limits.maxChunks)
assert.Equal(t, int64(50*1024*1024), pm.limits.maxChunkSize)
}

func TestPeerAdditionAndRetrieval(t *testing.T) {
pm := NewPeerManager()
id := createTestID(1)
addrs := createTestAddrs(8080)

// Add peer
info, err := pm.AddPeer(id, addrs)
assert.NoError(t, err)
assert.NotNil(t, info)
assert.Equal(t, id, info.ID)
assert.Equal(t, addrs, info.Addrs)
assert.Equal(t, PeerConnected, info.State)

// Retrieve peer
retrieved, exists := pm.GetPeer(id)
assert.True(t, exists)
assert.Equal(t, info, retrieved)

// Update existing peer
newAddrs := createTestAddrs(8081)
info, err = pm.AddPeer(id, newAddrs)
assert.NoError(t, err)
assert.Equal(t, newAddrs, info.Addrs)

// Non-existent peer
_, exists = pm.GetPeer(createTestID(99))
assert.False(t, exists)
}

func TestPeerRemoval(t *testing.T) {
pm := NewPeerManager()
id := createTestID(1)
addrs := createTestAddrs(8080)

// Add and then remove peer
_, err := pm.AddPeer(id, addrs)
assert.NoError(t, err)

pm.RemovePeer(id)
_, exists := pm.GetPeer(id)
assert.False(t, exists)
}

func TestPeerStateManagement(t *testing.T) {
pm := NewPeerManager()
id := createTestID(1)
addrs := createTestAddrs(8080)

info, err := pm.AddPeer(id, addrs)
assert.NoError(t, err)

// Test state updates
states := []PeerState{PeerDisconnected, PeerBlocked, PeerConnected}
for _, state := range states {
success := pm.UpdatePeerState(id, state)
assert.True(t, success)
assert.Equal(t, state, info.GetState())
}

// Test updating non-existent peer
success := pm.UpdatePeerState(createTestID(99), PeerConnected)
assert.False(t, success)
}

func TestPeerListing(t *testing.T) {
pm := NewPeerManager()
numPeers := 5

// Add multiple peers
for i := 0; i < numPeers; i++ {
id := createTestID(i)
addrs := createTestAddrs(8080 + i)
_, err := pm.AddPeer(id, addrs)
assert.NoError(t, err)
}

// Test listing all peers
peers := pm.ListPeers()
assert.Equal(t, numPeers, len(peers))

// Test peer count
count := pm.CountPeers()
assert.Equal(t, numPeers, count)

// Test connected peers filtering
pm.UpdatePeerState(createTestID(0), PeerDisconnected)
pm.UpdatePeerState(createTestID(1), PeerBlocked)
connected := pm.GetConnectedPeers()
assert.Equal(t, numPeers-2, len(connected))
}

func TestChunkTracking(t *testing.T) {
pm := NewPeerManager()
id := createTestID(1)
addrs := createTestAddrs(8080)

info, err := pm.AddPeer(id, addrs)
assert.NoError(t, err)

// Add chunks
chunkSize := int64(1024)
success := info.AddChunk(chunkSize)
assert.True(t, success)

count, total := info.GetChunkStats()
assert.Equal(t, 1, count)
assert.Equal(t, chunkSize, total)

// Add more chunks
info.AddChunk(chunkSize)
count, total = info.GetChunkStats()
assert.Equal(t, 2, count)
assert.Equal(t, 2*chunkSize, total)

// Remove chunks
success = info.RemoveChunk(chunkSize)
assert.True(t, success)
count, total = info.GetChunkStats()
assert.Equal(t, 1, count)
assert.Equal(t, chunkSize, total)

// Try to remove from empty peer
for i := 0; i < 2; i++ {
success = info.RemoveChunk(chunkSize)
}
assert.False(t, success)
count, total = info.GetChunkStats()
assert.Equal(t, 0, count)
assert.Equal(t, int64(0), total)
}

func TestConcurrentPeerOperations(t *testing.T) {
pm := NewPeerManager()
numRoutines := 10
done := make(chan bool)

for i := 0; i < numRoutines; i++ {
go func(i int) {
id := createTestID(i)
addrs := createTestAddrs(8080 + i)

// Add peer
info, err := pm.AddPeer(id, addrs)
assert.NoError(t, err)

// Update state
pm.UpdatePeerState(id, PeerConnected)

// Add and remove chunks
info.AddChunk(1024)
info.RemoveChunk(1024)

// Get peer info
retrieved, exists := pm.GetPeer(id)
assert.True(t, exists)
assert.Equal(t, info, retrieved)

done <- true
}(i)
}

// Wait for all routines to complete
for i := 0; i < numRoutines; i++ {
<-done
}

assert.Equal(t, numRoutines, pm.CountPeers())
}

func TestPeerLimits(t *testing.T) {
pm := NewPeerManager()
pm.SetLimits(2, 2, 1024) // 2 peers max, 2 chunks max, 1KB chunk size

// Test peer limit
for i := 0; i < 3; i++ {
id := createTestID(i)
addrs := createTestAddrs(8080 + i)
info, err := pm.AddPeer(id, addrs)

if i < 2 {
assert.NoError(t, err)
assert.NotNil(t, info)

// Test chunk limit
for j := 0; j < 3; j++ {
success := info.AddChunk(512) // 512 bytes per chunk
if j < 2 {
assert.True(t, success)
} else {
assert.False(t, success, "Should not allow more than maxChunks chunks")
}
}

// Test chunk size limit
success := info.AddChunk(2048) // 2KB chunk
assert.False(t, success, "Should not allow chunks larger than maxChunkSize")
}
}
}

func TestStateTransitions(t *testing.T) {
pm := NewPeerManager()
id := createTestID(1)
addrs := createTestAddrs(8080)

info, _ := pm.AddPeer(id, addrs)

// Test valid state transitions
validTransitions := []struct {
from PeerState
to   PeerState
}{
{PeerConnected, PeerDisconnected},
{PeerDisconnected, PeerConnected},
{PeerConnected, PeerBlocked},
{PeerDisconnected, PeerBlocked},
}

for _, tt := range validTransitions {
info.State = tt.from
success := pm.UpdatePeerState(id, tt.to)
assert.True(t, success)
assert.Equal(t, tt.to, info.GetState())
}

// Test state transitions when blocked
info.State = PeerBlocked
success := pm.UpdatePeerState(id, PeerConnected)
assert.True(t, success)
assert.Equal(t, PeerConnected, info.GetState())
}

func TestPeerRaceConditions(t *testing.T) {
info := &PeerInfo{
ID:    createTestID(1),
State: PeerConnected,
}

const numGoroutines = 100
done := make(chan bool)

// Test concurrent state access
go func() {
for i := 0; i < numGoroutines; i++ {
go func(i int) {
if i%2 == 0 {
info.GetState()
} else {
info.mu.Lock()
info.State = PeerDisconnected
info.mu.Unlock()
}
done <- true
}(i)
}
}()

// Test concurrent chunk operations
go func() {
for i := 0; i < numGoroutines; i++ {
go func(i int) {
if i%2 == 0 {
info.AddChunk(1024)
} else {
info.RemoveChunk(1024)
}
done <- true
}(i)
}
}()

// Test concurrent last seen updates
go func() {
for i := 0; i < numGoroutines; i++ {
go func() {
info.mu.Lock()
info.LastSeen = time.Now()
info.mu.Unlock()
done <- true
}()
}
}()

// Wait for all operations to complete
for i := 0; i < numGoroutines*3; i++ {
<-done
}
}

func TestInvalidInputHandling(t *testing.T) {
pm := NewPeerManager()

// Test nil address list
id := createTestID(1)
info, err := pm.AddPeer(id, nil)
assert.NoError(t, err)
assert.NotNil(t, info)
assert.Empty(t, info.Addrs)

// Test empty peer ID
var emptyID peer.ID
_, err = pm.AddPeer(emptyID, createTestAddrs(8080))
assert.NoError(t, err)

// Test negative limits
pm.SetLimits(-1, -1, -1)
assert.Equal(t, -1, pm.limits.maxPeers)
assert.Equal(t, -1, pm.limits.maxChunks)
assert.Equal(t, int64(-1), pm.limits.maxChunkSize)

// Test removing non-existent peer
pm.RemovePeer(createTestID(99))

// Test getting state of nil peer
success := pm.UpdatePeerState(createTestID(99), PeerConnected)
assert.False(t, success)
}

func TestPeerLastSeen(t *testing.T) {
pm := NewPeerManager()
id := createTestID(1)
addrs := createTestAddrs(8080)

before := time.Now()
info, err := pm.AddPeer(id, addrs)
after := time.Now()
assert.NoError(t, err)

lastSeen := info.GetLastSeen()
assert.True(t, lastSeen.After(before) || lastSeen.Equal(before))
assert.True(t, lastSeen.Before(after) || lastSeen.Equal(after))

// Update state should update LastSeen
time.Sleep(time.Millisecond)
pm.UpdatePeerState(id, PeerDisconnected)
newLastSeen := info.GetLastSeen()
assert.True(t, newLastSeen.After(lastSeen))
}
