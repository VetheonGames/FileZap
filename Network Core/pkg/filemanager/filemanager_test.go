package filemanager

import (
"fmt"
"io/fs"
"os"
"path/filepath"
"testing"
)

func TestChunkStorage(t *testing.T) {
	// Create temp directory for testing
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
			// Store chunk
			err := fm.StoreChunk(tt.chunkID, tt.data)
			if (err != nil) != tt.shouldError {
				t.Errorf("StoreChunk() error = %v, wantErr %v", err, tt.shouldError)
				return
			}

			if tt.shouldError {
				return
			}

			// Retrieve and verify chunk
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

func TestChunkDeletion(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "filezap-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

fm := NewChunkManager(tempDir)
	chunkID := "test-chunk-delete"
	data := []byte("test data to delete")

	// Store chunk
	if err := fm.StoreChunk(chunkID, data); err != nil {
		t.Fatalf("Failed to store chunk: %v", err)
	}

	// Verify chunk exists
	if _, err := fm.GetChunk(chunkID); err != nil {
		t.Fatalf("Chunk should exist before deletion: %v", err)
	}

	// Delete chunk
	if err := fm.DeleteChunk(chunkID); err != nil {
		t.Errorf("DeleteChunk() error = %v", err)
	}

	// Verify chunk doesn't exist
	if _, err := fm.GetChunk(chunkID); err == nil {
		t.Error("GetChunk() should return error for deleted chunk")
	}
}

func TestListChunks(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "filezap-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

fm := NewChunkManager(tempDir)
	
	expectedChunks := []string{
		"chunk1",
		"chunk2",
		"chunk3",
	}

	// Store test chunks
	for _, chunkID := range expectedChunks {
		err := fm.StoreChunk(chunkID, []byte("test data"))
		if err != nil {
			t.Fatalf("Failed to store chunk %s: %v", chunkID, err)
		}
	}

	// List chunks
	chunks, err := fm.ListChunks()
	if err != nil {
		t.Fatalf("ListChunks() error = %v", err)
	}

	// Verify all expected chunks are present
	if len(chunks) != len(expectedChunks) {
		t.Errorf("ListChunks() returned %d chunks, want %d", len(chunks), len(expectedChunks))
	}

	chunkMap := make(map[string]bool)
	for _, chunk := range chunks {
		chunkMap[chunk] = true
	}

	for _, expected := range expectedChunks {
		if !chunkMap[expected] {
			t.Errorf("ListChunks() missing expected chunk: %s", expected)
		}
	}
}

func TestErrorCases(t *testing.T) {
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
// Create a directory
roDir := filepath.Join(tempDir, "readonly")
if err := os.Mkdir(roDir, 0755); err != nil {
t.Fatalf("Failed to create directory: %v", err)
}

// Make it read-only
if err := os.Chmod(roDir, 0444); err != nil {
t.Fatalf("Failed to make directory read-only: %v", err)
}
defer os.Chmod(roDir, 0755) // Restore permissions for cleanup

roFm := NewChunkManager(roDir)
err := roFm.StoreChunk("test-chunk", []byte("test data"))
if err == nil {
t.Error("StoreChunk() should error with read-only directory")
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
	done := make(chan bool)
	const numGoroutines = 10

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			chunkID := fmt.Sprintf("concurrent-chunk-%d", id)
			data := []byte(fmt.Sprintf("concurrent data %d", id))

			// Store chunk
			err := fm.StoreChunk(chunkID, data)
			if err != nil {
				t.Errorf("Concurrent StoreChunk() error = %v", err)
			}

			// Retrieve chunk
			retrieved, err := fm.GetChunk(chunkID)
			if err != nil {
				t.Errorf("Concurrent GetChunk() error = %v", err)
			}

			if string(retrieved) != string(data) {
				t.Errorf("Concurrent GetChunk() = %v, want %v", string(retrieved), string(data))
			}

			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}

func TestMaxQuotaDefault(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "filezap-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	fm := NewChunkManager(tempDir)
	
	// Test storing data within default quota
	largeData := make([]byte, 1024*1024*1024) // 1GB
	err = fm.StoreChunk("large-chunk", largeData)
	if err != nil {
		t.Errorf("StoreChunk() should succeed within default quota: %v", err)
	}
}

func TestDirectoryPermissions(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "filezap-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create subdirectories with different permissions
	tests := []struct {
		name        string
		permissions fs.FileMode
		shouldError bool
	}{
		{
			name:        "No execute permission",
			permissions: 0644,
			shouldError: true,
		},
		{
			name:        "No write permission",
			permissions: 0555,
			shouldError: true,
		},
		{
			name:        "No read permission",
			permissions: 0333,
			shouldError: true,
		},
		{
			name:        "Full permissions",
			permissions: 0755,
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subDir := filepath.Join(tempDir, tt.name)
			if err := os.Mkdir(subDir, tt.permissions); err != nil {
				t.Fatalf("Failed to create subdirectory: %v", err)
			}
			defer os.Chmod(subDir, 0755) // Restore permissions for cleanup

			fm := NewChunkManager(subDir)
			err := fm.StoreChunk("test-chunk", []byte("test data"))
			if (err != nil) != tt.shouldError {
				t.Errorf("StoreChunk() error = %v, wantErr %v", err, tt.shouldError)
			}
		})
	}
}

func TestDiskUsageErrors(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "filezap-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	fm := NewChunkManager(tempDir)

	// Test with nonexistent directory
	nonExistentDir := filepath.Join(tempDir, "nonexistent")
	fm = NewChunkManager(nonExistentDir)
	if _, err := fm.GetDiskUsage(); err == nil {
		t.Error("GetDiskUsage() should error with nonexistent directory")
	}

	// Test with unreadable directory
	unreadableDir := filepath.Join(tempDir, "unreadable")
	if err := os.Mkdir(unreadableDir, 0); err != nil {
		t.Fatalf("Failed to create unreadable directory: %v", err)
	}
	defer os.Chmod(unreadableDir, 0755) // Restore permissions for cleanup

	fm = NewChunkManager(unreadableDir)
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
	done := make(chan bool)
	const numGoroutines = 10

	// Concurrently update quota and store chunks
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			// Set different quota sizes
			quotaSize := int64((id + 1) * 1024 * 1024) // 1MB to 10MB
			fm.SetQuota(quotaSize)

			// Try to store chunk
			data := make([]byte, 512*1024) // 512KB chunks
			err := fm.StoreChunk(fmt.Sprintf("quota-concurrent-%d", id), data)
			if err != nil {
				// Error is expected sometimes due to quota changes
				t.Logf("Concurrent store with quota change: %v", err)
			}

			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}

func TestStorageQuota(t *testing.T) {
tempDir, err := os.MkdirTemp("", "filezap-test-*")
if err != nil {
t.Fatalf("Failed to create temp directory: %v", err)
}
defer os.RemoveAll(tempDir)

fm := NewChunkManager(tempDir)
// Set a small quota for testing (5MB)
quotaSize := int64(5 * 1024 * 1024)
fm.SetQuota(quotaSize)

// Test sequential chunk storage
chunks := []struct {
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

for i, chunk := range chunks {
t.Run(chunk.name, func(t *testing.T) {
data := make([]byte, chunk.size)
for j := range data {
data[j] = byte(i) // Fill with different data to ensure proper size calculation
}

err := fm.StoreChunk(fmt.Sprintf("quota-test-%d", i), data)
if (err != nil) != chunk.shouldError {
t.Errorf("StoreChunk() error = %v, wantErr %v", err, chunk.shouldError)
return
}

// Verify disk usage after each operation
usage, err := fm.GetDiskUsage()
if err != nil {
t.Errorf("GetDiskUsage() error = %v", err)
return
}

if !chunk.shouldError && usage > quotaSize {
t.Errorf("Disk usage %d exceeds quota %d", usage, quotaSize)
}
})
}
}
