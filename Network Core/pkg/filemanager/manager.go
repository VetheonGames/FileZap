package filemanager

import (
    "log"
    "sync"

    "github.com/VetheonGames/FileZap/NetworkCore/pkg/types"
)

// Manager handles .zap files and chunk operations
type Manager struct {
    files        map[string]*types.FileInfo // map[zapFileName]*FileInfo
    peerChunks   map[string][]types.PeerChunkInfo // map[chunkID][]PeerChunkInfo
    mu           sync.RWMutex
}

// NewManager creates a new file manager
func NewManager() *Manager {
    return &Manager{
        files:      make(map[string]*types.FileInfo),
        peerChunks: make(map[string][]types.PeerChunkInfo),
    }
}

// RegisterPeerChunks registers which chunks a peer has available
func (m *Manager) RegisterPeerChunks(peerID string, address string, chunkIDs []string) {
    m.mu.Lock()
    defer m.mu.Unlock()

    // Create PeerChunkInfo for this peer
    peerInfo := types.PeerChunkInfo{
        PeerID:    peerID,
        ChunkIDs:  chunkIDs,
        Address:   address,
        Available: true,
    }

    // Update chunk availability mapping
    for _, chunkID := range chunkIDs {
        peers := m.peerChunks[chunkID]
        // Check if peer already exists
        exists := false
        for i, p := range peers {
            if p.PeerID == peerID {
                // Update existing peer info
                peers[i] = peerInfo
                exists = true
                break
            }
        }
        if !exists {
            // Add new peer for this chunk
            m.peerChunks[chunkID] = append(m.peerChunks[chunkID], peerInfo)
        }
    }
}

// GetPeersForChunk returns all peers that have a specific chunk
func (m *Manager) GetPeersForChunk(chunkID string) []types.PeerChunkInfo {
    m.mu.RLock()
    defer m.mu.RUnlock()

    peers := m.peerChunks[chunkID]
    result := make([]types.PeerChunkInfo, 0, len(peers))
    for _, peer := range peers {
        if peer.Available {
            result = append(result, peer)
        }
    }
    return result
}

// UpdatePeerStatus updates a peer's availability status
func (m *Manager) UpdatePeerStatus(peerID string, available bool) {
    m.mu.Lock()
    defer m.mu.Unlock()

    // Update peer status in all chunk mappings
    for _, peers := range m.peerChunks {
        for i := range peers {
            if peers[i].PeerID == peerID {
                peers[i].Available = available
            }
        }
    }
}

// RegisterZapFile registers a new .zap file and its chunks
func (m *Manager) RegisterZapFile(fileName string, chunkIDs []string) {
    m.mu.Lock()
    defer m.mu.Unlock()

    m.files[fileName] = &types.FileInfo{
        Name:      fileName,
        ChunkIDs:  chunkIDs,
        Available: true,
    }
    log.Printf("Registered .zap file %s with %d chunks", fileName, len(chunkIDs))
}

// RemoveZapFile removes a .zap file from available files
func (m *Manager) RemoveZapFile(fileName string) {
    m.mu.Lock()
    defer m.mu.Unlock()

    delete(m.files, fileName)
    log.Printf("Removed .zap file %s", fileName)
}

// GetAvailableZaps returns a list of available .zap files with their info
func (m *Manager) GetAvailableZaps() []types.FileInfo {
    m.mu.RLock()
    defer m.mu.RUnlock()

    zaps := make([]types.FileInfo, 0, len(m.files))
    for _, info := range m.files {
        if info.Available {
            zaps = append(zaps, *info)
        }
    }
    return zaps
}

// GetChunks returns the chunk IDs for a specific .zap file
func (m *Manager) GetChunks(fileName string) ([]string, bool) {
    m.mu.RLock()
    defer m.mu.RUnlock()

    if info, exists := m.files[fileName]; exists && info.Available {
        return info.ChunkIDs, true
    }
    return nil, false
}

// HasChunk checks if a specific chunk is available
func (m *Manager) HasChunk(fileName, chunkID string) bool {
    m.mu.RLock()
    defer m.mu.RUnlock()

    info, exists := m.files[fileName]
    if !exists || !info.Available {
        return false
    }

    for _, id := range info.ChunkIDs {
        if id == chunkID {
            return true
        }
    }
    return false
}

// GetAllAvailableChunks returns all available chunks for all .zap files
func (m *Manager) GetAllAvailableChunks() map[string][]string {
    m.mu.RLock()
    defer m.mu.RUnlock()

    // Create a deep copy to prevent concurrent access issues
    result := make(map[string][]string)
    for fileName, info := range m.files {
        if info.Available {
            chunksCopy := make([]string, len(info.ChunkIDs))
            copy(chunksCopy, info.ChunkIDs)
            result[fileName] = chunksCopy
        }
    }

    return result
}
