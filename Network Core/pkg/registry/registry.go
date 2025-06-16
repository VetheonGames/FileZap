package registry

import (
"sync"

"github.com/VetheonGames/FileZap/NetworkCore/pkg/types"
)

// FileRegistry handles file and chunk registration
type FileRegistry struct {
files     map[string]*types.FileInfo // filename -> FileInfo
chunks    map[string][]string        // chunkID -> []peerID
peerInfo  map[string]*types.PeerChunkInfo
mu       sync.RWMutex
}

// NewFileRegistry creates a new file registry
func NewFileRegistry() *FileRegistry {
return &FileRegistry{
files:    make(map[string]*types.FileInfo),
chunks:   make(map[string][]string),
peerInfo: make(map[string]*types.PeerChunkInfo),
}
}

// RegisterFile registers a file and its chunks
func (fr *FileRegistry) RegisterFile(info *types.FileInfo) error {
fr.mu.Lock()
defer fr.mu.Unlock()

fr.files[info.Name] = info

// Update chunk mappings
for _, chunkID := range info.ChunkIDs {
for _, peer := range info.Peers {
if contains(peer.ChunkIDs, chunkID) {
fr.chunks[chunkID] = appendUnique(fr.chunks[chunkID], peer.PeerID)
}
}
}

return nil
}

// UnregisterFile removes a file and its chunk mappings
func (fr *FileRegistry) UnregisterFile(filename string) {
fr.mu.Lock()
defer fr.mu.Unlock()

if info, exists := fr.files[filename]; exists {
// Remove chunk mappings
for _, chunkID := range info.ChunkIDs {
delete(fr.chunks, chunkID)
}
delete(fr.files, filename)
}
}

// GetFile retrieves file information
func (fr *FileRegistry) GetFile(filename string) (*types.FileInfo, bool) {
fr.mu.RLock()
defer fr.mu.RUnlock()

info, exists := fr.files[filename]
return info, exists
}

// ListFiles returns all registered files
func (fr *FileRegistry) ListFiles() []*types.FileInfo {
fr.mu.RLock()
defer fr.mu.RUnlock()

files := make([]*types.FileInfo, 0, len(fr.files))
for _, info := range fr.files {
files = append(files, info)
}
return files
}

// RegisterPeer registers a peer and its chunks
func (fr *FileRegistry) RegisterPeer(info *types.PeerChunkInfo) {
fr.mu.Lock()
defer fr.mu.Unlock()

fr.peerInfo[info.PeerID] = info

// Update chunk mappings
for _, chunkID := range info.ChunkIDs {
fr.chunks[chunkID] = appendUnique(fr.chunks[chunkID], info.PeerID)
}
}

// UnregisterPeer removes a peer and its chunk mappings
func (fr *FileRegistry) UnregisterPeer(peerID string) {
fr.mu.Lock()
defer fr.mu.Unlock()

if info, exists := fr.peerInfo[peerID]; exists {
// Remove chunk mappings
for _, chunkID := range info.ChunkIDs {
fr.chunks[chunkID] = remove(fr.chunks[chunkID], peerID)
if len(fr.chunks[chunkID]) == 0 {
delete(fr.chunks, chunkID)
}
}
delete(fr.peerInfo, peerID)
}
}

// GetChunkPeers returns peers that have a specific chunk
func (fr *FileRegistry) GetChunkPeers(chunkID string) []string {
fr.mu.RLock()
defer fr.mu.RUnlock()

peers := fr.chunks[chunkID]
result := make([]string, len(peers))
copy(result, peers)
return result
}

// UpdatePeerAvailability updates a peer's availability status
func (fr *FileRegistry) UpdatePeerAvailability(peerID string, available bool) bool {
fr.mu.Lock()
defer fr.mu.Unlock()

if info, exists := fr.peerInfo[peerID]; exists {
info.Available = available
return true
}
return false
}

// GetAvailablePeers returns all available peers
func (fr *FileRegistry) GetAvailablePeers() []*types.PeerChunkInfo {
fr.mu.RLock()
defer fr.mu.RUnlock()

var peers []*types.PeerChunkInfo
for _, info := range fr.peerInfo {
if info.Available {
peers = append(peers, info)
}
}
return peers
}

// Helper functions

func contains(slice []string, item string) bool {
for _, s := range slice {
if s == item {
return true
}
}
return false
}

func appendUnique(slice []string, item string) []string {
if !contains(slice, item) {
return append(slice, item)
}
return slice
}

func remove(slice []string, item string) []string {
for i, s := range slice {
if s == item {
return append(slice[:i], slice[i+1:]...)
}
}
return slice
}
