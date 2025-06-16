package filemanager

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
)

const maxQuotaSize = 100 * 1024 * 1024 * 1024 // 100GB default quota

// ChunkManager handles storage and retrieval of file chunks
type ChunkManager struct {
baseDir    string
quotaSize  int64
mu         sync.RWMutex
}

// SetQuota sets the storage quota size in bytes
func (cm *ChunkManager) SetQuota(size int64) {
cm.mu.Lock()
defer cm.mu.Unlock()
cm.quotaSize = size
}

// NewChunkManager creates a new ChunkManager instance
func NewChunkManager(baseDir string) *ChunkManager {
return &ChunkManager{
baseDir:    baseDir,
quotaSize:  maxQuotaSize,
}
}

// StoreChunk stores a chunk with the given ID
func (cm *ChunkManager) StoreChunk(chunkID string, data []byte) error {
if chunkID == "" {
return errors.New("chunk ID cannot be empty")
}

cm.mu.Lock()
defer cm.mu.Unlock()

// Check write permissions
info, err := os.Stat(cm.baseDir)
if err != nil {
return fmt.Errorf("failed to check directory permissions: %v", err)
}
mode := info.Mode()
if mode&0200 == 0 { // Check if directory is writable
return fmt.Errorf("directory not writable")
}

// Check quota
usage, err := cm.getDiskUsageNoLock()
if err != nil {
return fmt.Errorf("failed to check disk usage: %v", err)
}

if usage+int64(len(data)) > cm.quotaSize {
return fmt.Errorf("quota exceeded: would exceed %d bytes", cm.quotaSize)
}

chunkPath := filepath.Join(cm.baseDir, chunkID)
return os.WriteFile(chunkPath, data, 0644)
}

// getDiskUsageNoLock returns the total size of all stored chunks without locking
// Caller must hold at least a read lock
func (cm *ChunkManager) getDiskUsageNoLock() (int64, error) {
var total int64
err := filepath.WalkDir(cm.baseDir, func(path string, d fs.DirEntry, err error) error {
if err != nil {
return err
}
if !d.IsDir() {
info, err := d.Info()
if err != nil {
return err
}
total += info.Size()
}
return nil
})

if err != nil {
return 0, fmt.Errorf("failed to calculate disk usage: %v", err)
}
return total, nil
}

// GetChunk retrieves a chunk by its ID
func (cm *ChunkManager) GetChunk(chunkID string) ([]byte, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	chunkPath := filepath.Join(cm.baseDir, chunkID)
	data, err := os.ReadFile(chunkPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("chunk %s not found", chunkID)
		}
		return nil, fmt.Errorf("failed to read chunk %s: %v", chunkID, err)
	}
	return data, nil
}

// DeleteChunk removes a chunk by its ID
func (cm *ChunkManager) DeleteChunk(chunkID string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	chunkPath := filepath.Join(cm.baseDir, chunkID)
	err := os.Remove(chunkPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("chunk %s not found", chunkID)
		}
		return fmt.Errorf("failed to delete chunk %s: %v", chunkID, err)
	}
	return nil
}

// ListChunks returns a list of all stored chunk IDs
func (cm *ChunkManager) ListChunks() ([]string, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	entries, err := os.ReadDir(cm.baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %v", err)
	}

	chunks := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			chunks = append(chunks, entry.Name())
		}
	}
	return chunks, nil
}

// GetDiskUsage returns the total size of all stored chunks
func (cm *ChunkManager) GetDiskUsage() (int64, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var total int64
	err := filepath.WalkDir(cm.baseDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			info, err := d.Info()
			if err != nil {
				return err
			}
			total += info.Size()
		}
		return nil
	})
	
	if err != nil {
		return 0, fmt.Errorf("failed to calculate disk usage: %v", err)
	}
	return total, nil
}
