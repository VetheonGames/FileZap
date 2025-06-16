package filemanager

import (
    "errors"
    "fmt"
    "os"
    "path/filepath"
    "runtime"
    "sync"
)

const maxQuotaSize = 100 * 1024 * 1024 * 1024 // 100GB default quota

// Custom errors
var (
    ErrInvalidAccess = errors.New("directory access denied")
)

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

// verifyAccess tests if directory can be accessed for the required operation
func (cm *ChunkManager) verifyAccess(writeRequired bool) error {
    // Check if directory exists and is a directory
    info, err := os.Stat(cm.baseDir)
    if err != nil {
        if os.IsNotExist(err) {
            return fmt.Errorf("directory does not exist")
        }
        return ErrInvalidAccess
    }

    if !info.IsDir() {
        return fmt.Errorf("path is not a directory")
    }

    // Check read access
    if _, err := os.ReadDir(cm.baseDir); err != nil {
        return ErrInvalidAccess
    }

    if writeRequired {
        // Check write access by attempting to create a file
        testFile := filepath.Join(cm.baseDir, ".test")
        if err := os.WriteFile(testFile, []byte{}, 0644); err != nil {
            return ErrInvalidAccess
        }
        os.Remove(testFile)
    }

    return nil
}

// StoreChunk stores a chunk with the given ID
func (cm *ChunkManager) StoreChunk(chunkID string, data []byte) error {
    if chunkID == "" {
        return errors.New("chunk ID cannot be empty")
    }

    cm.mu.Lock()
    defer cm.mu.Unlock()

    // Verify write access
    if err := cm.verifyAccess(true); err != nil {
        if runtime.GOOS == "windows" {
            return ErrInvalidAccess
        }
        return err
    }

    // Check quota
    usage, err := cm.getDiskUsageNoLock()
    if err != nil {
        return fmt.Errorf("failed to check disk usage: %v", err)
    }

    if usage+int64(len(data)) > cm.quotaSize {
        return fmt.Errorf("quota exceeded: would exceed %d bytes", cm.quotaSize)
    }

    // Store chunk
    chunkPath := filepath.Join(cm.baseDir, chunkID)
    if err := os.WriteFile(chunkPath, data, 0644); err != nil {
        return ErrInvalidAccess
    }

    return nil
}

// getDiskUsageNoLock returns the total size of all stored chunks without locking
func (cm *ChunkManager) getDiskUsageNoLock() (int64, error) {
    entries, err := os.ReadDir(cm.baseDir)
    if err != nil {
        if runtime.GOOS == "windows" {
            return 0, ErrInvalidAccess
        }
        return 0, err
    }

    var total int64
    for _, entry := range entries {
        if !entry.IsDir() {
            info, err := entry.Info()
            if err != nil {
                if runtime.GOOS == "windows" {
                    return 0, ErrInvalidAccess
                }
                return 0, err
            }
            total += info.Size()
        }
    }
    return total, nil
}

// GetChunk retrieves a chunk by its ID
func (cm *ChunkManager) GetChunk(chunkID string) ([]byte, error) {
    cm.mu.RLock()
    defer cm.mu.RUnlock()

    // Verify read access
    if err := cm.verifyAccess(false); err != nil {
        if runtime.GOOS == "windows" {
            return nil, ErrInvalidAccess
        }
        return nil, err
    }

    chunkPath := filepath.Join(cm.baseDir, chunkID)
    data, err := os.ReadFile(chunkPath)
    if err != nil {
        if os.IsNotExist(err) {
            return nil, fmt.Errorf("chunk %s not found", chunkID)
        }
        return nil, ErrInvalidAccess
    }

    return data, nil
}

// DeleteChunk removes a chunk by its ID
func (cm *ChunkManager) DeleteChunk(chunkID string) error {
    cm.mu.Lock()
    defer cm.mu.Unlock()

    // Verify write access
    if err := cm.verifyAccess(true); err != nil {
        if runtime.GOOS == "windows" {
            return ErrInvalidAccess
        }
        return err
    }

    chunkPath := filepath.Join(cm.baseDir, chunkID)
    if err := os.Remove(chunkPath); err != nil {
        if os.IsNotExist(err) {
            return fmt.Errorf("chunk %s not found", chunkID)
        }
        return ErrInvalidAccess
    }

    return nil
}

// ListChunks returns a list of all stored chunk IDs
func (cm *ChunkManager) ListChunks() ([]string, error) {
    cm.mu.RLock()
    defer cm.mu.RUnlock()

    // Verify read access
    if err := cm.verifyAccess(false); err != nil {
        if runtime.GOOS == "windows" {
            return nil, ErrInvalidAccess
        }
        return nil, err
    }

    entries, err := os.ReadDir(cm.baseDir)
    if err != nil {
        return nil, ErrInvalidAccess
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

    // Verify read access
    if err := cm.verifyAccess(false); err != nil {
        return 0, ErrInvalidAccess
    }

    return cm.getDiskUsageNoLock()
}
