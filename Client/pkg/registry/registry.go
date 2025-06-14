package registry

import (
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "sync"
    "time"

    "github.com/VetheonGames/FileZap/NetworkCore/pkg/types"
)

// FileInfo represents a registered .zap file in the network
type FileInfo struct {
    ID              string   `json:"id"`
    Name            string   `json:"name"`
    ChunkCount      int      `json:"chunk_count"`
    PeerIDs         []string `json:"peer_ids"`
    TotalSize       int64    `json:"total_size"`
    ZapMetadata     []byte   `json:"zap_metadata"`
    ReplicationGoal int      `json:"replication_goal"`
}

// ChunkPeerInfo stores information about peers hosting chunks
type ChunkPeerInfo struct {
    Info     types.PeerChunkInfo `json:"info"`
    LastSeen int64              `json:"last_seen"`
}

// Registry manages .zap file registrations and peer associations
type Registry struct {
    files      map[string]*FileInfo // map[fileID]FileInfo
    dataDir    string
    mu         sync.RWMutex
    peerChunks map[string]map[string]*ChunkPeerInfo // map[chunkID]map[peerID]ChunkPeerInfo
}

// NewRegistry creates a new .zap file registry
func NewRegistry(dataDir string) (*Registry, error) {
    if err := os.MkdirAll(dataDir, 0755); err != nil {
        return nil, fmt.Errorf("failed to create data directory: %v", err)
    }

    r := &Registry{
        files:      make(map[string]*FileInfo),
        dataDir:    dataDir,
        peerChunks: make(map[string]map[string]*ChunkPeerInfo),
    }

    // Load existing registry data
    if err := r.loadRegistry(); err != nil {
        return nil, fmt.Errorf("failed to load registry: %v", err)
    }

    return r, nil
}

// RegisterFile adds or updates a .zap file registration
func (r *Registry) RegisterFile(file *FileInfo) error {
    r.mu.Lock()
    defer r.mu.Unlock()

    r.files[file.ID] = file
    return r.saveRegistry()
}

// RegisterPeerChunks registers which chunks a peer has available
func (r *Registry) RegisterPeerChunks(peerID string, address string, chunkIDs []string) {
    r.mu.Lock()
    defer r.mu.Unlock()

    // Create ChunkPeerInfo with types.PeerChunkInfo
    info := &ChunkPeerInfo{
        Info: types.PeerChunkInfo{
            PeerID:    peerID,
            Address:   address,
            ChunkIDs:  chunkIDs,
            Available: true,
        },
        LastSeen: time.Now().Unix(),
    }

    // Update chunk availability mapping
    for _, chunkID := range chunkIDs {
        if r.peerChunks[chunkID] == nil {
            r.peerChunks[chunkID] = make(map[string]*ChunkPeerInfo)
        }
        r.peerChunks[chunkID][peerID] = info
    }

    // Save changes
    r.saveRegistry()
}

// GetPeersForChunk returns all peers that have a specific chunk
func (r *Registry) GetPeersForChunk(chunkID string) []types.PeerChunkInfo {
    r.mu.RLock()
    defer r.mu.RUnlock()

    var peers []types.PeerChunkInfo
    if peerMap, exists := r.peerChunks[chunkID]; exists {
        for _, info := range peerMap {
            peers = append(peers, info.Info)
        }
    }
    return peers
}

// CleanupStaleChunks removes chunk entries from peers that haven't been seen recently
func (r *Registry) CleanupStaleChunks(maxAge time.Duration) {
    r.mu.Lock()
    defer r.mu.Unlock()

    now := time.Now().Unix()
    for chunkID, peerMap := range r.peerChunks {
        for peerID, info := range peerMap {
            if now-info.LastSeen > int64(maxAge.Seconds()) {
                delete(peerMap, peerID)
            }
        }
        // Remove empty chunk entries
        if len(peerMap) == 0 {
            delete(r.peerChunks, chunkID)
        }
    }

    // Save changes
    r.saveRegistry()
}

// GetFile retrieves file information by ID
func (r *Registry) GetFile(id string) (*FileInfo, bool) {
    r.mu.RLock()
    defer r.mu.RUnlock()

    file, exists := r.files[id]
    return file, exists
}

// GetFileByName retrieves file information by name
func (r *Registry) GetFileByName(name string) (*FileInfo, bool) {
    r.mu.RLock()
    defer r.mu.RUnlock()

    for _, file := range r.files {
        if file.Name == name {
            return file, true
        }
    }
    return nil, false
}

// AddPeerToFile associates a peer with a .zap file
func (r *Registry) AddPeerToFile(fileID, peerID string) error {
    r.mu.Lock()
    defer r.mu.Unlock()

    file, exists := r.files[fileID]
    if !exists {
        return fmt.Errorf("file not found: %s", fileID)
    }

    // Check if peer is already associated
    for _, id := range file.PeerIDs {
        if id == peerID {
            return nil
        }
    }

    file.PeerIDs = append(file.PeerIDs, peerID)
    return r.saveRegistry()
}

// RemovePeerFromFile removes a peer association from a .zap file
func (r *Registry) RemovePeerFromFile(fileID, peerID string) error {
    r.mu.Lock()
    defer r.mu.Unlock()

    file, exists := r.files[fileID]
    if !exists {
        return fmt.Errorf("file not found: %s", fileID)
    }

    // Remove peer ID
    for i, id := range file.PeerIDs {
        if id == peerID {
            file.PeerIDs = append(file.PeerIDs[:i], file.PeerIDs[i+1:]...)
            break
        }
    }

    return r.saveRegistry()
}

// GetPeerFiles returns all files associated with a peer
func (r *Registry) GetPeerFiles(peerID string) []*FileInfo {
    r.mu.RLock()
    defer r.mu.RUnlock()

    var files []*FileInfo
    for _, file := range r.files {
        for _, id := range file.PeerIDs {
            if id == peerID {
                files = append(files, file)
                break
            }
        }
    }
    return files
}

// saveRegistry persists the registry to disk
func (r *Registry) saveRegistry() error {
    data, err := json.MarshalIndent(r.files, "", "  ")
    if err != nil {
        return fmt.Errorf("failed to marshal registry: %v", err)
    }

    path := filepath.Join(r.dataDir, "registry.json")
    if err := os.WriteFile(path, data, 0644); err != nil {
        return fmt.Errorf("failed to save registry: %v", err)
    }

    return nil
}

// loadRegistry loads the registry from disk
func (r *Registry) loadRegistry() error {
    path := filepath.Join(r.dataDir, "registry.json")

    data, err := os.ReadFile(path)
    if os.IsNotExist(err) {
        return nil // New registry
    }
    if err != nil {
        return fmt.Errorf("failed to read registry: %v", err)
    }

    if err := json.Unmarshal(data, &r.files); err != nil {
        return fmt.Errorf("failed to parse registry: %v", err)
    }

    return nil
}

// GetAllFiles returns all registered files
func (r *Registry) GetAllFiles() []*FileInfo {
    r.mu.RLock()
    defer r.mu.RUnlock()

    files := make([]*FileInfo, 0, len(r.files))
    for _, file := range r.files {
        files = append(files, file)
    }
    return files
}

// GetPeersForFile returns all peers that have a specific file
func (r *Registry) GetPeersForFile(fileID string) []string {
    r.mu.RLock()
    defer r.mu.RUnlock()

    if file, exists := r.files[fileID]; exists {
        return file.PeerIDs
    }
    return nil
}
