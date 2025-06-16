package filemanager

import (
    "fmt"
    "os"
    "path/filepath"
    "runtime"
    "sync"
    "testing"
    "time"
)

type fileHandles struct {
    handles []*os.File
}

func createRestrictedDir(t *testing.T, baseDir, name string) string {
    dir := filepath.Join(baseDir, name)
    if err := os.Mkdir(dir, 0755); err != nil {
        t.Fatalf("Failed to create temp directory: %v", err)
    }

    if runtime.GOOS == "windows" {
        // On Windows, create a file handle to simulate restrictions
        f, err := os.OpenFile(filepath.Join(dir, ".lock"), os.O_CREATE|os.O_EXCL|os.O_RDWR, 0666)
        if err != nil {
            t.Fatalf("Failed to create lock file: %v", err)
        }

        // Store the handle to keep the file locked
        h := &fileHandles{handles: []*os.File{f}}

        // Keep file handle open until garbage collection
        runtime.SetFinalizer(h, func(h *fileHandles) {
            for _, f := range h.handles {
                f.Close()
            }
        })

        // Give time for OS to register the lock
        time.Sleep(100 * time.Millisecond)
        t.Log("Note: Permission restrictions may not be reliable on Windows")
    } else {
        // On Unix systems, use chmod
        if err := os.Chmod(dir, 0); err != nil {
            t.Fatalf("Failed to set directory permissions: %v", err)
        }
    }

    return dir
}

func TestChunkStorage(t *testing.T) {
    tempDir, err := os.MkdirTemp("", "filezap-test-*")
    if err != nil {
        t.Fatalf("Failed to create temp directory: %v", err)
    }
    defer os.RemoveAll(tempDir)

    tests := []struct {
        name        string
        chunkID     string
        data        []byte
        shouldError bool
    }{
        {
            name:        "Store and retrieve valid chunk",
            chunkID:     "test-chunk-1",
            data:        []byte("test data content"),
            shouldError: false,
        },
        {
            name:        "Store empty chunk",
            chunkID:     "test-chunk-2",
            data:        []byte{},
            shouldError: false,
        },
        {
            name:        "Store large chunk",
            chunkID:     "test-chunk-3",
            data:        make([]byte, 1024*1024), // 1MB chunk
            shouldError: false,
        },
    }

    fm := NewChunkManager(tempDir)
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := fm.StoreChunk(tt.chunkID, tt.data)
            if (err != nil) != tt.shouldError {
                t.Errorf("StoreChunk() error = %v, wantErr %v", err, tt.shouldError)
                return
            }

            if tt.shouldError {
                return
            }

            retrieved, err := fm.GetChunk(tt.chunkID)
            if err != nil {
                t.Errorf("GetChunk() error = %v", err)
                return
            }

            if string(retrieved) != string(tt.data) {
                t.Errorf("GetChunk() = %v, want %v", string(retrieved), string(tt.data))
            }
        })
    }
}

func TestErrorCases(t *testing.T) {
    if runtime.GOOS == "windows" {
        t.Skip("Skipping permission tests on Windows")
    }

    tempDir, err := os.MkdirTemp("", "filezap-test-*")
    if err != nil {
        t.Fatalf("Failed to create temp directory: %v", err)
    }
    defer os.RemoveAll(tempDir)

    fm := NewChunkManager(tempDir)

    t.Run("Invalid chunk ID", func(t *testing.T) {
        err := fm.StoreChunk("", []byte("test data"))
        if err == nil {
            t.Error("StoreChunk() should error with empty chunk ID")
        }
    })

    t.Run("Get nonexistent chunk", func(t *testing.T) {
        _, err := fm.GetChunk("nonexistent-chunk")
        if err == nil {
            t.Error("GetChunk() should error for nonexistent chunk")
        }
    })

    t.Run("Delete nonexistent chunk", func(t *testing.T) {
        err := fm.DeleteChunk("nonexistent-chunk")
        if err == nil {
            t.Error("DeleteChunk() should error for nonexistent chunk")
        }
    })

    t.Run("Storage directory permissions", func(t *testing.T) {
        restrictedDir := createRestrictedDir(t, tempDir, "restricted")
        roFm := NewChunkManager(restrictedDir)

        err := roFm.StoreChunk("test-chunk", []byte("test data"))
        if err == nil {
            t.Error("StoreChunk() should error with restricted directory")
        }
    })
}

func TestConcurrentAccess(t *testing.T) {
    tempDir, err := os.MkdirTemp("", "filezap-test-*")
    if err != nil {
        t.Fatalf("Failed to create temp directory: %v", err)
    }
    defer os.RemoveAll(tempDir)

    fm := NewChunkManager(tempDir)
    var wg sync.WaitGroup
    const numGoroutines = 10

    for i := 0; i < numGoroutines; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            chunkID := fmt.Sprintf("concurrent-chunk-%d", id)
            data := []byte(fmt.Sprintf("concurrent data %d", id))

            // Store chunk
            err := fm.StoreChunk(chunkID, data)
            if err != nil {
                t.Errorf("Concurrent StoreChunk() error = %v", err)
                return
            }

            // Retrieve chunk
            retrieved, err := fm.GetChunk(chunkID)
            if err != nil {
                t.Errorf("Concurrent GetChunk() error = %v", err)
                return
            }

            if string(retrieved) != string(data) {
                t.Errorf("Concurrent GetChunk() = %v, want %v", string(retrieved), string(data))
            }
        }(i)
    }

    wg.Wait()
}

func TestDiskUsageErrors(t *testing.T) {
    if runtime.GOOS == "windows" {
        t.Skip("Skipping permission tests on Windows")
    }

    tempDir, err := os.MkdirTemp("", "filezap-test-*")
    if err != nil {
        t.Fatalf("Failed to create temp directory: %v", err)
    }
    defer os.RemoveAll(tempDir)

    testDir := createRestrictedDir(t, tempDir, "unreadable")
    fm := NewChunkManager(testDir)

    if _, err := fm.GetDiskUsage(); err == nil {
        t.Error("GetDiskUsage() should error with unreadable directory")
    }
}

func TestConcurrentQuotaUpdates(t *testing.T) {
    tempDir, err := os.MkdirTemp("", "filezap-test-*")
    if err != nil {
        t.Fatalf("Failed to create temp directory: %v", err)
    }
    defer os.RemoveAll(tempDir)

    fm := NewChunkManager(tempDir)
    var wg sync.WaitGroup
    const numGoroutines = 10

    for i := 0; i < numGoroutines; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            quotaSize := int64((id + 1) * 1024 * 1024) // 1MB to 10MB
            fm.SetQuota(quotaSize)

            data := make([]byte, 512*1024) // 512KB chunks
            err := fm.StoreChunk(fmt.Sprintf("quota-concurrent-%d", id), data)
            if err != nil {
                t.Logf("Concurrent store with quota change: %v", err)
            }
        }(i)
    }

    wg.Wait()
}

func TestStorageQuota(t *testing.T) {
    tempDir, err := os.MkdirTemp("", "filezap-test-*")
    if err != nil {
        t.Fatalf("Failed to create temp directory: %v", err)
    }
    defer os.RemoveAll(tempDir)

    fm := NewChunkManager(tempDir)
    quotaSize := int64(5 * 1024 * 1024) // 5MB
    fm.SetQuota(quotaSize)

    tests := []struct {
        name        string
        size        int
        shouldError bool
    }{
        {
            name:        "First chunk (2MB)",
            size:        2 * 1024 * 1024,
            shouldError: false,
        },
        {
            name:        "Second chunk (2MB)",
            size:        2 * 1024 * 1024,
            shouldError: false,
        },
        {
            name:        "Third chunk exceeding quota (2MB)",
            size:        2 * 1024 * 1024,
            shouldError: true,
        },
    }

    for i, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            data := make([]byte, tt.size)
            for j := range data {
                data[j] = byte(i)
            }

            err := fm.StoreChunk(fmt.Sprintf("quota-test-%d", i), data)
            if (err != nil) != tt.shouldError {
                t.Errorf("StoreChunk() error = %v, wantErr %v", err, tt.shouldError)
                return
            }

            usage, err := fm.GetDiskUsage()
            if err != nil {
                t.Errorf("GetDiskUsage() error = %v", err)
                return
            }

            if !tt.shouldError && usage > quotaSize {
                t.Errorf("Disk usage %d exceeds quota %d", usage, quotaSize)
            }
        })
    }
}
