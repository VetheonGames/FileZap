package filemanager

import (
	"log"
	"sync"
)

// Manager handles .zap files and chunk operations
type Manager struct {
	availableZaps map[string][]string // map[zapFileName][]chunkIDs
	mu            sync.RWMutex
}

// NewManager creates a new file manager
func NewManager() *Manager {
	return &Manager{
		availableZaps: make(map[string][]string),
	}
}

// RegisterZapFile registers a new .zap file and its chunks
func (m *Manager) RegisterZapFile(fileName string, chunkIDs []string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.availableZaps[fileName] = chunkIDs
	log.Printf("Registered .zap file %s with %d chunks", fileName, len(chunkIDs))
}

// RemoveZapFile removes a .zap file from available files
func (m *Manager) RemoveZapFile(fileName string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.availableZaps, fileName)
	log.Printf("Removed .zap file %s", fileName)
}

// GetAvailableZaps returns a list of available .zap files
func (m *Manager) GetAvailableZaps() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	zaps := make([]string, 0, len(m.availableZaps))
	for zapName := range m.availableZaps {
		zaps = append(zaps, zapName)
	}
	return zaps
}

// GetChunks returns the chunk IDs for a specific .zap file
func (m *Manager) GetChunks(fileName string) ([]string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	chunks, exists := m.availableZaps[fileName]
	return chunks, exists
}

// HasChunk checks if a specific chunk is available
func (m *Manager) HasChunk(fileName, chunkID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	chunks, exists := m.availableZaps[fileName]
	if !exists {
		return false
	}

	for _, id := range chunks {
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
	result := make(map[string][]string, len(m.availableZaps))
	for fileName, chunks := range m.availableZaps {
		chunksCopy := make([]string, len(chunks))
		copy(chunksCopy, chunks)
		result[fileName] = chunksCopy
	}

	return result
}
