package peer

import (
"sync"
"time"

"github.com/libp2p/go-libp2p/core/peer"
"github.com/multiformats/go-multiaddr"
)

// PeerState represents the current state of a peer
type PeerState int

const (
// PeerUnknown indicates the peer's state is not known
PeerUnknown PeerState = iota
// PeerConnected indicates the peer is currently connected
PeerConnected
// PeerDisconnected indicates the peer was previously connected but is now disconnected
PeerDisconnected
// PeerBlocked indicates the peer has been blocked
PeerBlocked
)

// PeerInfo represents information about a peer in the network
type PeerInfo struct {
ID          peer.ID
Addrs       []multiaddr.Multiaddr
State       PeerState
LastSeen    time.Time
ChunkCount  int
TotalChunks int64 // total size of all chunks in bytes
mu          sync.RWMutex
}

// PeerManager handles peer tracking and management
type PeerManager struct {
peers sync.Map // peer.ID -> *PeerInfo
limits struct {
maxPeers     int
maxChunks    int
maxChunkSize int64
}
}

// NewPeerManager creates a new peer manager with default limits
func NewPeerManager() *PeerManager {
pm := &PeerManager{}
pm.SetLimits(100, 1000, 100*1024*1024) // 100 peers, 1000 chunks per peer, 100MB per chunk
return pm
}

// SetLimits sets the operational limits for the peer manager
func (pm *PeerManager) SetLimits(maxPeers, maxChunks int, maxChunkSize int64) {
pm.limits.maxPeers = maxPeers
pm.limits.maxChunks = maxChunks
pm.limits.maxChunkSize = maxChunkSize
}

// AddPeer adds or updates a peer
func (pm *PeerManager) AddPeer(id peer.ID, addrs []multiaddr.Multiaddr) (*PeerInfo, error) {
peerInfo, loaded := pm.peers.LoadOrStore(id, &PeerInfo{
ID:       id,
Addrs:    addrs,
State:    PeerConnected,
LastSeen: time.Now(),
})

info := peerInfo.(*PeerInfo)
if loaded {
info.mu.Lock()
info.Addrs = addrs
info.State = PeerConnected
info.LastSeen = time.Now()
info.mu.Unlock()
}

return info, nil
}

// GetPeer retrieves information about a peer
func (pm *PeerManager) GetPeer(id peer.ID) (*PeerInfo, bool) {
if value, ok := pm.peers.Load(id); ok {
return value.(*PeerInfo), true
}
return nil, false
}

// RemovePeer removes a peer from tracking
func (pm *PeerManager) RemovePeer(id peer.ID) {
pm.peers.Delete(id)
}

// UpdatePeerState updates a peer's connection state
func (pm *PeerManager) UpdatePeerState(id peer.ID, state PeerState) bool {
if value, ok := pm.peers.Load(id); ok {
info := value.(*PeerInfo)
info.mu.Lock()
info.State = state
info.LastSeen = time.Now()
info.mu.Unlock()
return true
}
return false
}

// ListPeers returns a list of all tracked peers
func (pm *PeerManager) ListPeers() []*PeerInfo {
var peers []*PeerInfo
pm.peers.Range(func(key, value interface{}) bool {
peers = append(peers, value.(*PeerInfo))
return true
})
return peers
}

// CountPeers returns the number of tracked peers
func (pm *PeerManager) CountPeers() int {
count := 0
pm.peers.Range(func(key, value interface{}) bool {
count++
return true
})
return count
}

// GetConnectedPeers returns a list of all currently connected peers
func (pm *PeerManager) GetConnectedPeers() []*PeerInfo {
var peers []*PeerInfo
pm.peers.Range(func(key, value interface{}) bool {
info := value.(*PeerInfo)
info.mu.RLock()
if info.State == PeerConnected {
peers = append(peers, info)
}
info.mu.RUnlock()
return true
})
return peers
}

// PeerInfo methods

// AddChunk updates peer info when a chunk is added
func (p *PeerInfo) AddChunk(size int64) bool {
p.mu.Lock()
defer p.mu.Unlock()

p.ChunkCount++
p.TotalChunks += size
return true
}

// RemoveChunk updates peer info when a chunk is removed
func (p *PeerInfo) RemoveChunk(size int64) bool {
p.mu.Lock()
defer p.mu.Unlock()

if p.ChunkCount > 0 {
p.ChunkCount--
p.TotalChunks -= size
return true
}
return false
}

// GetState returns the current peer state
func (p *PeerInfo) GetState() PeerState {
p.mu.RLock()
defer p.mu.RUnlock()
return p.State
}

// GetLastSeen returns when the peer was last seen
func (p *PeerInfo) GetLastSeen() time.Time {
p.mu.RLock()
defer p.mu.RUnlock()
return p.LastSeen
}

// GetChunkStats returns the current chunk count and total size
func (p *PeerInfo) GetChunkStats() (int, int64) {
p.mu.RLock()
defer p.mu.RUnlock()
return p.ChunkCount, p.TotalChunks
}
