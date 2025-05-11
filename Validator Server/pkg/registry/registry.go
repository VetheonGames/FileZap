package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// FileInfo represents a registered .zap file in the network
type FileInfo struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	ChunkCount  int      `json:"chunk_count"`
	PeerIDs     []string `json:"peer_ids"`
	TotalSize   int64    `json:"total_size"`
	ZapMetadata []byte   `json:"zap_metadata"`
}

// Registry manages .zap file registrations and peer associations
type Registry struct {
	files   map[string]*FileInfo // map[fileID]FileInfo
	dataDir string
	mu      sync.RWMutex
}

// NewRegistry creates a new .zap file registry
func NewRegistry(dataDir string) (*Registry, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %v", err)
	}

	r := &Registry{
		files:   make(map[string]*FileInfo),
		dataDir: dataDir,
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
